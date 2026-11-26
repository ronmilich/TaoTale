// Package tts defines the speech provider contract and verified voice catalog.
package tts

import (
	"context"
	"errors"
	"slices"
	"unicode"

	"taotale/backend/internal/sourcetext"
)

type Code string

const (
	Invalid     Code = "VALIDATION_ERROR"
	Unavailable Code = "TTS_UNAVAILABLE"
	Failed      Code = "TTS_PROVIDER_ERROR"
	RateLimited Code = "TTS_RATE_LIMITED"
	Timeout     Code = "TTS_TIMEOUT"
	Canceled    Code = "TTS_CANCELED"
)

type Error struct{ Code Code }

func (e *Error) Error() string { return string(e.Code) }

func CodeOf(err error) Code {
	if err == nil {
		return ""
	}
	var providerErr *Error
	if errors.As(err, &providerErr) {
		return providerErr.Code
	}
	return Failed
}

type Voice struct {
	ID          string    `json:"voiceId"`
	DisplayName string    `json:"displayName"`
	Language    string    `json:"languageCode"`
	Provider    string    `json:"providerId"`
	Rates       []float64 `json:"speakingRates"`
}

type Request struct {
	Language, VoiceID, Source string
	Rate                      float64
}

type Boundary struct {
	Text, Kind             string
	TextOffset, TextLength uint
	AudioTicks             uint64
	DurationTicks          int64
}

type Result struct {
	MP3        []byte
	Boundaries []Boundary
	// BoundaryText is the exact document indexed by the provider in Unicode
	// scalars. Azure indexes escaped SSML, while the mock indexes source text.
	BoundaryText string
}

type VoiceCatalog interface {
	ProviderID() string
	ListVoices(ctx context.Context, language string) ([]Voice, error)
}

type TTSProvider interface {
	VoiceCatalog
	Synthesize(ctx context.Context, request Request) (Result, error)
}

func DefaultVoice(language, provider string) (string, error) {
	catalog, err := voices(language, provider)
	if err != nil {
		return "", err
	}
	return catalog[0].ID, nil
}

func SupportsLanguage(language string) bool {
	return language == "zh-CN" || language == "ko-KR"
}

func AllowedVoice(language, provider, id string) bool {
	if provider != "azure" && provider != "mock" {
		return false
	}
	catalog, err := voices(language, provider)
	return err == nil && slices.ContainsFunc(catalog, func(voice Voice) bool { return voice.ID == id })
}

func ValidateRequest(provider string, request Request) error {
	if !AllowedVoice(request.Language, provider, request.VoiceID) {
		return &Error{Invalid}
	}
	catalog, _ := voices(request.Language, provider)
	return validate(request, catalog)
}

func voices(language, provider string) ([]Voice, error) {
	var id, name string
	switch language {
	case "zh-CN":
		id, name = "zh-CN-XiaoxiaoNeural", "Xiaoxiao"
	case "ko-KR":
		id, name = "ko-KR-SunHiNeural", "SunHi"
	default:
		return nil, &Error{Invalid}
	}
	if provider == "mock" {
		id, name = language+"-mock", name+" (mock)"
	}
	return []Voice{{id, name, language, provider, []float64{0.75, 1, 1.25}}}, nil
}

func validate(request Request, catalog []Voice) error {
	for _, voice := range catalog {
		if voice.ID == request.VoiceID && slices.Contains(voice.Rates, request.Rate) {
			index, err := sourcetext.NewIndex(request.Source)
			if err != nil {
				break
			}
			end, _ := index.AtByte(len(request.Source))
			if end.UTF16 <= 1500 && validSpeechSource(request.Source) {
				return nil
			}
			break
		}
	}
	return &Error{Invalid}
}

func validSpeechSource(source string) bool {
	hasSpeech := false
	for _, r := range source {
		if (r < 0x20 && r != '\t' && r != '\n' && r != '\r') || r == 0xfffe || r == 0xffff {
			return false
		}
		hasSpeech = hasSpeech || unicode.IsLetter(r) || unicode.IsNumber(r)
	}
	return hasSpeech
}

func contextError(ctx context.Context) error {
	switch ctx.Err() {
	case nil:
		return nil
	case context.DeadlineExceeded:
		return &Error{Timeout}
	default:
		return &Error{Canceled}
	}
}
