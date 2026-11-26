package tts

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestAzureWorkerTransport(t *testing.T) {
	request := Request{"zh-CN", "zh-CN-XiaoxiaoNeural", "甲 & 乙", 1}
	for _, tc := range []struct {
		name, body string
		want       Code
	}{
		{"success", `cat >/dev/null
printf '%s' '{"MP3":"SUQz","Boundaries":[{"Text":"甲","TextLength":1,"AudioTicks":100}]}'`, ""},
		{"sanitized failure", `printf '%s' '{"Code":"private-provider-diagnostics"}'`, Failed},
		{"quota", `printf '%s' '{"Code":"TTS_RATE_LIMITED"}'`, RateLimited},
		{"deadline", "exec /bin/sleep 10", Timeout},
	} {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "worker")
			if err := os.WriteFile(path, []byte("#!/bin/sh\n"+tc.body+"\n"), 0700); err != nil {
				t.Fatal(err)
			}
			timeout := 5 * time.Second
			if tc.want == Timeout {
				timeout = 50 * time.Millisecond
			}
			provider := NewAzure("offline test configuration", "eastus", path, timeout)
			voices, err := provider.ListVoices(t.Context(), request.Language)
			if err != nil || len(voices) != 1 || voices[0].ID != request.VoiceID {
				t.Fatalf("catalog: %v, %v", voices, err)
			}
			got, err := provider.Synthesize(t.Context(), request)
			if tc.want == "" {
				if err != nil || string(got.MP3) != "ID3" || len(got.Boundaries) != 1 || got.BoundaryText != ssml(request) {
					t.Fatalf("unexpected worker result %+v, %v", got, err)
				}
			} else if CodeOf(err) != tc.want {
				t.Fatalf("got %+v, %v, want %s", got, err, tc.want)
			}
		})
	}
}
