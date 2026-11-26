package alignment

import (
	"bytes"
	"context"
	"errors"
	"os/exec"
	"time"
)

const (
	decodeRate      = 24000
	MaxMP3Bytes     = 4 * 1024 * 1024
	maxDecodedBytes = 20 * 60 * decodeRate * 2
)

type pcmCount int64

func (count *pcmCount) Write(data []byte) (int, error) {
	if int64(*count)+int64(len(data)) > maxDecodedBytes {
		return 0, errors.New("decoded audio exceeds the limit")
	}
	*count += pcmCount(len(data))
	return len(data), nil
}

// measureMP3 returns the duration in 100 ns ticks, rounded down. Decode the
// actual bytes to mono PCM instead of trusting the provider or MP3 metadata.
func measureMP3(ctx context.Context, data []byte) (uint64, error) {
	if len(data) == 0 || len(data) > MaxMP3Bytes {
		return 0, unreliable("missing or oversized MP3")
	}
	if !completeFrames(data) {
		return 0, unreliable("invalid or truncated MP3 frames")
	}
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	var count pcmCount
	cmd := exec.CommandContext(ctx, "ffmpeg", "-nostdin", "-v", "error", "-xerror", "-err_detect", "explode",
		"-f", "mp3", "-i", "pipe:0", "-map", "0:a:0", "-ac", "1", "-ar", "24000", "-f", "s16le", "pipe:1")
	cmd.Stdin, cmd.Stdout = bytes.NewReader(data), &count
	if err := cmd.Run(); err != nil || count <= 0 || count%2 != 0 {
		return 0, unreliable("MP3 did not decode completely")
	}
	return uint64(count/2) * 10_000_000 / decodeRate, nil
}
