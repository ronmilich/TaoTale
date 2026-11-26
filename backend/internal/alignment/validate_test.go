package alignment

import (
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"testing"

	"taotale/backend/internal/fingerprint"
	"taotale/backend/internal/segmentation"
	"taotale/backend/internal/tts"
)

func requireDecoder(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("FFmpeg is required for MP3 validation")
	}
}

func mockAudio(t *testing.T, source string) (fingerprint.Identity, string, tts.Result) {
	t.Helper()
	id := fingerprint.Identity{
		LanguageCode: "zh-CN", ProviderID: "mock", VoiceID: "zh-CN-mock", SpeakingRate: 1,
		StrategyID: "zh-cn-phrase", StrategyVersion: segmentation.Version1, ExactSourceText: source,
	}
	digest, err := id.Sum()
	if err != nil {
		t.Fatal(err)
	}
	audio, err := (tts.Mock{}).Synthesize(t.Context(), tts.Request{Language: id.LanguageCode, VoiceID: id.VoiceID, Source: source, Rate: 1})
	if err != nil {
		t.Fatal(err)
	}
	return id, digest, audio
}

func TestReadyGate(t *testing.T) {
	requireDecoder(t)
	id, digest, audio := mockAudio(t, "😀甲𠀀，乙。")
	verified, err := Validate(t.Context(), id, digest, audio)
	if err != nil || verified.DurationMS < 1000 || len(verified.Segments) != 2 {
		t.Fatalf("valid audio: %+v, %v", verified, err)
	}
	if first := verified.Segments[0]; first.TextEnd != 6 || first.StartMS != 0 || first.EndMS != 500 {
		t.Fatalf("first timed range: %+v", first)
	}
	for _, tc := range []struct {
		name   string
		change func(*tts.Result)
	}{
		{"missing MP3", func(r *tts.Result) { r.MP3 = nil }},
		{"garbage MP3", func(r *tts.Result) { r.MP3 = []byte("bad audio") }},
		{"truncated frame", func(r *tts.Result) { r.MP3 = r.MP3[:len(r.MP3)-13] }},
		{"missing boundary", func(r *tts.Result) { r.Boundaries = r.Boundaries[:1] }},
		{"wrong document", func(r *tts.Result) { r.BoundaryText += " " }},
		{"wrong boundary text", func(r *tts.Result) { r.Boundaries[0].Text = "wrong" }},
		{"word crosses gap", func(r *tts.Result) {
			r.Boundaries[0].Text = ""
			r.Boundaries[0].TextLength += 2
		}},
		{"raw overlap", func(r *tts.Result) { r.Boundaries[1].AudioTicks-- }},
		{"rounded overlap", func(r *tts.Result) {
			r.Boundaries[0].DurationTicks--
			r.Boundaries[1].AudioTicks--
		}},
		{"raw audio overrun", func(r *tts.Result) { r.Boundaries[1].DurationTicks += 10_000_000 }},
		{"punctuation has no timing", func(r *tts.Result) { r.Boundaries[1].Kind = "Punctuation" }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			changed := audio
			changed.Boundaries = slices.Clone(audio.Boundaries)
			tc.change(&changed)
			if _, err := Validate(t.Context(), id, digest, changed); !errors.Is(err, ErrUnreliable) {
				t.Fatalf("invalid candidate returned %v", err)
			}
		})
	}
	if _, err := Validate(t.Context(), id, "wrong fingerprint", audio); !errors.Is(err, ErrUnreliable) {
		t.Fatalf("mismatched intent: %v", err)
	}
}

func TestLegacyOverrun(t *testing.T) {
	requireDecoder(t)
	const source = "你好，世界。"
	_, _, audio := mockAudio(t, source)
	id := fingerprint.LegacyIdentity{SegmentationVersion: 1, SpeakingRate: 1, Text: source, VoiceID: "zh-CN-YunxiNeural"}
	const hash = "dc5a5bdcd4221414d35fcd2885a0a35872805e95f08701a49e7e47b26561cd39"
	duration, err := measureMP3(t.Context(), audio.MP3)
	if err != nil {
		t.Fatal(err)
	}
	ms := int64(duration / ticksPerMS)
	result, err := segmentation.SegmentSource(segmentation.Key{LanguageCode: "zh-CN", StrategyID: "zh-cn-legacy-v1", Version: segmentation.Version1}, source)
	if err != nil {
		t.Fatal(err)
	}
	stored := []TimedSegment{{result.Segments[0], 0, 500}, {result.Segments[1], 500, ms + 50}}
	verified, err := ValidateLegacy(t.Context(), id, hash, audio.MP3, stored)
	if err != nil || verified.Segments[1].EndMS != ms {
		t.Fatalf("legacy clamp: %+v, %v", verified, err)
	}
	stored[1].EndMS++
	if _, err := ValidateLegacy(t.Context(), id, hash, audio.MP3, stored); !errors.Is(err, ErrUnreliable) {
		t.Fatalf("legacy overrun above 50 ms: %v", err)
	}
}

func TestEscapedOffsetsAndLeadingClosingQuote(t *testing.T) {
	const source = "甲。 ” 乙&😀"
	request := tts.Request{Language: "zh-CN", VoiceID: "zh-CN-XiaoxiaoNeural", Source: source, Rate: 1}
	doc, offsets := tts.BoundaryDocument("azure", request)
	indexed := []rune(doc)
	var entity uint
	for i := range indexed {
		if indexed[i] == '&' {
			entity = uint(i)
			break
		}
	}
	if left, ok := offsets[entity]; !ok || left != 6 || offsets[entity+5] != 7 || offsets[entity+6] != 9 {
		t.Fatalf("incorrect source offsets around escaped ampersand and emoji: %v", offsets)
	}
	for i := uint(1); i < 5; i++ {
		if _, ok := offsets[entity+i]; ok {
			t.Fatal("entity interior was mapped to source")
		}
	}
	key, _ := segmentation.Default("zh-CN")
	const adjacent = "甲。” 乙"
	spans, err := segmentation.SegmentSource(key, adjacent)
	if err != nil {
		t.Fatal(err)
	}
	events := []tts.Boundary{
		{Text: "甲", Kind: "Word", TextLength: 1, DurationTicks: 10000},
		{Text: "” 乙", Kind: "Word", TextOffset: 2, TextLength: 3, AudioTicks: 20000, DurationTicks: 10000},
	}
	_, direct := tts.BoundaryDocument("mock", tts.Request{Source: adjacent})
	if _, err := alignEvents(events, []rune(adjacent), direct, spans.Segments, adjacent, 40000); err != nil {
		t.Fatal(err)
	}
	events[1].Text = "。” 乙"
	events[1].TextOffset--
	events[1].TextLength++
	if _, err := alignEvents(events, []rune(adjacent), direct, spans.Segments, adjacent, 40000); !errors.Is(err, ErrUnreliable) {
		t.Fatalf("unproven crossing accepted: %v", err)
	}
}

func TestTAO03Replay(t *testing.T) {
	root := os.Getenv("TAOTALE_AZURE_REPLAY_DIR")
	if root == "" {
		t.Skip("set TAOTALE_AZURE_REPLAY_DIR to the existing TAO-03 captures")
	}
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Fatal("FFmpeg is required for replay")
	}
	for _, dir := range []string{"zh-CN-0.75", "zh-CN-1.25", "ko-KR-0.75", "ko-KR-1.25", "pair/zh-CN", "pair/ko-KR"} {
		t.Run(dir, func(t *testing.T) {
			var capture struct {
				Locale, Voice, Rate, Source, SSML string
				DecodedSamples24kHz               int64
				Events                            []struct {
					tts.Boundary
					Type       int
					WordLength uint
				}
			}
			path := filepath.Join(root, dir)
			raw, err := os.ReadFile(filepath.Join(path, "events.json"))
			if err != nil {
				t.Fatal(err)
			}
			if err := json.Unmarshal(raw, &capture); err != nil {
				t.Fatal(err)
			}
			mp3, err := os.ReadFile(filepath.Join(path, "audio.mp3"))
			if err != nil {
				t.Fatal(err)
			}
			rate, err := strconv.ParseFloat(capture.Rate, 64)
			if err != nil {
				t.Fatal(err)
			}
			key, err := segmentation.Default(capture.Locale)
			if err != nil {
				t.Fatal(err)
			}
			id := fingerprint.Identity{
				LanguageCode: capture.Locale, ProviderID: "azure", VoiceID: capture.Voice, SpeakingRate: rate,
				StrategyID: key.StrategyID, StrategyVersion: key.Version, ExactSourceText: capture.Source,
			}
			hash, err := id.Sum()
			if err != nil {
				t.Fatal(err)
			}
			audio := tts.Result{MP3: mp3, BoundaryText: capture.SSML}
			for _, raw := range capture.Events {
				if raw.Type < 0 || raw.Type > 2 {
					t.Fatal("unexpected SDK boundary type")
				}
				b := raw.Boundary
				b.Kind = []string{"Word", "Punctuation", "Sentence"}[raw.Type]
				b.TextLength = raw.WordLength
				audio.Boundaries = append(audio.Boundaries, b)
			}
			verified, err := Validate(t.Context(), id, hash, audio)
			if err != nil || verified.DurationMS != capture.DecodedSamples24kHz/24 {
				t.Fatalf("replay duration %d ms, expected %d ms, error %v", verified.DurationMS, capture.DecodedSamples24kHz/24, err)
			}
			t.Logf("%d decoded ms, %d timed segments", verified.DurationMS, len(verified.Segments))
		})
	}
}
