//go:build azure_sdk && linux && amd64 && cgo

// The worker owns the native SDK handles for one request. The backend can
// forcibly terminate it if the SDK does not finish cancellation.
package main

import (
	"context"
	"encoding/json"
	"io"
	"os"
	"os/signal"
	"sync"

	"github.com/Microsoft/cognitive-services-speech-sdk-go/common"
	"github.com/Microsoft/cognitive-services-speech-sdk-go/speech"
	"taotale/backend/internal/tts"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	var request tts.WorkerRequest
	if err := json.NewDecoder(io.LimitReader(os.Stdin, 64*1024)).Decode(&request); err != nil {
		os.Exit(1)
	}
	if err := json.NewEncoder(os.Stdout).Encode(synthesize(ctx, request)); err != nil {
		os.Exit(1)
	}
}

func synthesize(ctx context.Context, request tts.WorkerRequest) tts.WorkerReply {
	unavailable := tts.WorkerReply{Code: tts.Unavailable}
	failed := tts.WorkerReply{Code: tts.Failed}
	if request.Key == "" || request.Region == "" || request.SSML == "" {
		return unavailable
	}
	config, err := speech.NewSpeechConfigFromSubscription(request.Key, request.Region)
	if err != nil {
		return unavailable
	}
	defer config.Close()
	if config.SetSpeechSynthesisOutputFormat(common.Audio24Khz48KBitRateMonoMp3) != nil ||
		config.SetProperty(common.SpeechServiceResponseRequestWordBoundary, "true") != nil {
		return unavailable
	}
	synth, err := speech.NewSpeechSynthesizerFromConfig(config, nil)
	if err != nil {
		return unavailable
	}
	defer synth.Close()
	var mu sync.Mutex
	var events []tts.Boundary
	synth.WordBoundary(func(event speech.SpeechSynthesisWordBoundaryEventArgs) {
		defer event.Close()
		mu.Lock()
		defer mu.Unlock()
		events = append(events, tts.Boundary{
			Text: event.Text, Kind: boundaryKind(event.BoundaryType), TextOffset: event.TextOffset,
			TextLength: event.WordLength, AudioTicks: event.AudioOffset, DurationTicks: event.Duration.Nanoseconds() / 100,
		})
	})
	if ctx.Err() != nil {
		return tts.WorkerReply{Code: tts.Canceled}
	}
	pending := synth.SpeakSsmlAsync(request.SSML)
	select {
	case <-ctx.Done():
		<-synth.StopSpeakingAsync()
		outcome := <-pending
		outcome.Close()
		return tts.WorkerReply{Code: tts.Canceled}
	case outcome := <-pending:
		defer outcome.Close()
		if outcome.Error != nil || outcome.Result == nil {
			return failed
		}
		switch outcome.Result.Reason {
		case common.SynthesizingAudioCompleted:
			mu.Lock()
			defer mu.Unlock()
			return tts.WorkerReply{MP3: outcome.Result.AudioData, Boundaries: append([]tts.Boundary(nil), events...)}
		case common.Canceled:
			details, err := speech.NewCancellationDetailsFromSpeechSynthesisResult(outcome.Result)
			if err != nil {
				return failed
			}
			if details.Reason == common.CancelledByUser {
				return tts.WorkerReply{Code: tts.Canceled}
			}
			return tts.WorkerReply{Code: cancellationCode(details.ErrorCode)}
		default:
			return failed
		}
	}
}

func boundaryKind(kind common.SpeechSynthesisBoundaryType) string {
	switch kind {
	case common.WordBoundary:
		return "Word"
	case common.PunctuationBoundary:
		return "Punctuation"
	case common.SentenceBoundary:
		return "Sentence"
	default:
		return "Unknown"
	}
}

func cancellationCode(code common.CancellationErrorCode) tts.Code {
	switch code {
	case common.AuthenticationFailure, common.Forbidden, common.ConnectionFailure, common.ServiceUnavailable:
		return tts.Unavailable
	case common.TooManyRequests:
		return tts.RateLimited
	case common.ServiceTimeout:
		return tts.Timeout
	default:
		return tts.Failed
	}
}
