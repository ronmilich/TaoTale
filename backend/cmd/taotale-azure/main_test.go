//go:build azure_sdk && linux && amd64 && cgo

package main

import (
	"testing"

	"github.com/Microsoft/cognitive-services-speech-sdk-go/common"
	"taotale/backend/internal/tts"
)

func TestNativeMapping(t *testing.T) {
	for code, want := range map[common.CancellationErrorCode]tts.Code{
		common.AuthenticationFailure: tts.Unavailable,
		common.ConnectionFailure:     tts.Unavailable,
		common.TooManyRequests:       tts.RateLimited,
		common.ServiceTimeout:        tts.Timeout,
		common.ServiceError:          tts.Failed,
	} {
		if got := cancellationCode(code); got != want {
			t.Errorf("cancellation %s: got %s, want %s", code, got, want)
		}
	}
	for kind, want := range map[common.SpeechSynthesisBoundaryType]string{
		common.WordBoundary:        "Word",
		common.PunctuationBoundary: "Punctuation",
		common.SentenceBoundary:    "Sentence",
	} {
		if got := boundaryKind(kind); got != want {
			t.Errorf("boundary %d: got %s, want %s", kind, got, want)
		}
	}
	if reply := synthesize(t.Context(), tts.WorkerRequest{}); reply.Code != tts.Unavailable {
		t.Errorf("missing configuration: %s", reply.Code)
	}
}
