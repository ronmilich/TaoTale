package tts

import (
	"context"
	_ "embed"
	"slices"
	"unicode"
)

//go:embed silence.mp3
var silence []byte

type Mock struct{}

func (Mock) ProviderID() string { return "mock" }

func (Mock) ListVoices(ctx context.Context, language string) ([]Voice, error) {
	if err := contextError(ctx); err != nil {
		return nil, err
	}
	return voices(language, "mock")
}

func (m Mock) Synthesize(ctx context.Context, request Request) (Result, error) {
	catalog, err := m.ListVoices(ctx, request.Language)
	if err != nil {
		return Result{}, err
	}
	if err := validate(request, catalog); err != nil {
		return Result{}, err
	}
	result := Result{MP3: slices.Clone(silence), BoundaryText: request.Source}
	runes := []rune(request.Source)
	for i := 0; i < len(runes); {
		if !speechRune(runes[i]) {
			i++
			continue
		}
		start := i
		for i < len(runes) && speechRune(runes[i]) {
			i++
		}
		result.Boundaries = append(result.Boundaries, Boundary{
			Text: string(runes[start:i]), Kind: "Word", TextOffset: uint(start), TextLength: uint(i - start),
		})
	}
	for i := range result.Boundaries {
		duration := int64(1000/len(result.Boundaries)) * 10_000
		result.Boundaries[i].AudioTicks = uint64(int64(i) * duration)
		result.Boundaries[i].DurationTicks = duration
	}
	return result, nil
}

func speechRune(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsNumber(r) || unicode.IsMark(r)
}
