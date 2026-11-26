// Package alignment verifies an audio candidate against its exact synthesis intent.
package alignment

import (
	"context"
	"errors"
	"strings"
	"unicode"
	"unicode/utf16"

	"taotale/backend/internal/fingerprint"
	"taotale/backend/internal/segmentation"
	"taotale/backend/internal/tts"
)

const ticksPerMS = 10000

var ErrUnreliable = errors.New("ALIGNMENT_UNRELIABLE")

type Error struct{ Reason string }

func (e *Error) Error() string { return ErrUnreliable.Error() + ": " + e.Reason }
func (e *Error) Unwrap() error { return ErrUnreliable }

func unreliable(reason string) error { return &Error{reason} }

type TimedSegment struct {
	segmentation.Segment
	StartMS int64 `json:"startMs"`
	EndMS   int64 `json:"endMs"`
}

type Verified struct {
	DurationMS int64
	Segments   []TimedSegment
}

type tickRange struct{ start, end uint64 }

func Validate(ctx context.Context, id fingerprint.Identity, expected string, audio tts.Result) (Verified, error) {
	actual, err := id.Sum()
	if err != nil || actual != expected || id.StrategyID == "zh-cn-legacy-v1" {
		return Verified{}, unreliable("synthesis fingerprint mismatch")
	}
	request := tts.Request{Language: id.LanguageCode, VoiceID: id.VoiceID, Source: id.ExactSourceText, Rate: id.SpeakingRate}
	if err := tts.ValidateRequest(id.ProviderID, request); err != nil {
		return Verified{}, unreliable("invalid synthesis request")
	}
	strategy := segmentation.Key{LanguageCode: id.LanguageCode, StrategyID: id.StrategyID, Version: id.StrategyVersion}
	spans, err := segmentation.SegmentSource(strategy, request.Source)
	if err != nil || len(spans.Segments) == 0 {
		return Verified{}, unreliable("invalid source segments")
	}
	doc, sourceOffsets := tts.BoundaryDocument(id.ProviderID, request)
	if doc != audio.BoundaryText {
		return Verified{}, unreliable("boundary document mismatch")
	}
	duration, err := measureMP3(ctx, audio.MP3)
	if err != nil {
		return Verified{}, err
	}
	timing, err := alignEvents(audio.Boundaries, []rune(doc), sourceOffsets, spans.Segments, request.Source, duration)
	if err != nil {
		return Verified{}, err
	}
	verified := Verified{DurationMS: int64(duration / ticksPerMS)}
	for i, span := range spans.Segments {
		rangeTicks := timing[i]
		start := int64(rangeTicks.start / ticksPerMS)
		end := int64((rangeTicks.end + ticksPerMS - 1) / ticksPerMS)
		if rangeTicks.end == 0 || start >= end || end > verified.DurationMS ||
			(i > 0 && start < verified.Segments[i-1].EndMS) {
			return Verified{}, unreliable("incomplete or overlapping millisecond ranges")
		}
		verified.Segments = append(verified.Segments, TimedSegment{span, start, end})
	}
	return verified, nil
}

func alignEvents(events []tts.Boundary, indexed []rune, offsets map[uint]int, segments []segmentation.Segment, source string, duration uint64) ([]tickRange, error) {
	ranges := make([]tickRange, len(segments))
	units := utf16.Encode([]rune(source))
	covered := make([]bool, len(units))
	var lastSource int
	var lastAudio, lastWordEnd uint64
	current := 0
	for _, event := range events {
		start := event.TextOffset
		if start > uint(len(indexed)) || event.TextLength > uint(len(indexed))-start || event.TextLength == 0 {
			return nil, unreliable("boundary outside the indexed document")
		}
		end := start + event.TextLength
		left, leftOK := offsets[start]
		right, rightOK := offsets[end]
		if !leftOK || !rightOK || left < lastSource || (event.Text != "" && event.Text != string(indexed[start:end])) {
			return nil, unreliable("boundary does not match an ordered source slice")
		}
		if event.AudioTicks < lastAudio || event.AudioTicks > duration || event.DurationTicks < 0 ||
			uint64(event.DurationTicks) > duration-event.AudioTicks {
			return nil, unreliable("boundary timing is outside decoded audio")
		}
		lastSource, lastAudio = right, event.AudioTicks
		if event.Kind == "Punctuation" || event.Kind == "Sentence" {
			continue
		}
		if event.Kind != "Word" || event.DurationTicks == 0 || event.AudioTicks < lastWordEnd {
			return nil, unreliable("invalid or overlapping speech timing")
		}
		// The SDK includes a closing quote from the preceding phrase in the
		// next word's text. Its measured time belongs to the word. Drop only
		// this untimed prefix for association and keep the exact raw checks above.
		wordStart := left
		for wordStart < right && strings.ContainsRune("”’」』\"' \t\u3000", rune(units[wordStart])) {
			wordStart++
		}
		for current < len(segments) && segments[current].TextEnd <= wordStart {
			current++
		}
		if wordStart == right || current >= len(segments) || wordStart < segments[current].TextStart || right > segments[current].TextEnd {
			return nil, unreliable("word crosses a segment boundary or gap")
		}
		lastWordEnd = event.AudioTicks + uint64(event.DurationTicks)
		if ranges[current].end == 0 {
			ranges[current].start = event.AudioTicks
		}
		ranges[current].end = lastWordEnd
		for i := wordStart; i < right; i++ {
			covered[i] = true
		}
	}
	unit := 0
	for _, r := range source {
		if (unicode.IsLetter(r) || unicode.IsNumber(r)) && !covered[unit] {
			return nil, unreliable("speech-bearing source is missing word timing")
		}
		unit += utf16.RuneLen(r)
	}
	return ranges, nil
}

// ValidateLegacy checks stored Mandarin ranges against the frozen fingerprint and
// strategy. Only the final stored end may be clamped, by at most 50 ms.
func ValidateLegacy(ctx context.Context, id fingerprint.LegacyIdentity, expected string, mp3 []byte, stored []TimedSegment) (Verified, error) {
	match, err := id.Verify(expected)
	if err != nil || !match || id.SegmentationVersion != 1 || len(stored) == 0 {
		return Verified{}, unreliable("legacy synthesis fingerprint mismatch")
	}
	key := segmentation.Key{LanguageCode: "zh-CN", StrategyID: "zh-cn-legacy-v1", Version: segmentation.Version1}
	spans, err := segmentation.SegmentSource(key, id.Text)
	if err != nil || len(spans.Segments) != len(stored) {
		return Verified{}, unreliable("legacy segment mismatch")
	}
	duration, err := measureMP3(ctx, mp3)
	if err != nil {
		return Verified{}, err
	}
	result := Verified{DurationMS: int64(duration / ticksPerMS)}
	var prevEnd int64
	for i, segment := range stored {
		if segment.EndMS > result.DurationMS && i == len(stored)-1 && segment.EndMS-result.DurationMS <= 50 {
			segment.EndMS = result.DurationMS
		}
		if segment.Segment != spans.Segments[i] || segment.StartMS < prevEnd ||
			segment.EndMS <= segment.StartMS || segment.EndMS > result.DurationMS {
			return Verified{}, unreliable("invalid legacy source or timing")
		}
		result.Segments = append(result.Segments, segment)
		prevEnd = segment.EndMS
	}
	return result, nil
}
