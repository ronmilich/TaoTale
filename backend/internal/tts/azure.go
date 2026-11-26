package tts

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

const DefaultTimeout = 30 * time.Second

var (
	azureRegion = regexp.MustCompile("^[a-z][a-z0-9]{0,31}$")
)

type Azure struct {
	key, region, worker string
	timeout             time.Duration
}

func NewAzure(key, region, worker string, timeout time.Duration) *Azure {
	if timeout <= 0 {
		timeout = DefaultTimeout
	}
	return &Azure{strings.TrimSpace(key), strings.TrimSpace(region), worker, timeout}
}

func (*Azure) ProviderID() string { return "azure" }

func (a *Azure) ListVoices(ctx context.Context, language string) ([]Voice, error) {
	if err := contextError(ctx); err != nil {
		return nil, err
	}
	catalog, err := voices(language, "azure")
	if err != nil {
		return nil, err
	}
	if a == nil || a.key == "" || !azureRegion.MatchString(a.region) || !filepath.IsAbs(a.worker) {
		return nil, &Error{Unavailable}
	}
	info, err := os.Stat(a.worker)
	if err != nil || !info.Mode().IsRegular() || info.Mode().Perm()&0111 == 0 {
		return nil, &Error{Unavailable}
	}
	return catalog, nil
}

// WorkerRequest and WorkerReply travel on private pipes between the backend and
// the native SDK process. No credentials are passed on the command line.
type WorkerRequest struct {
	Key, Region, SSML string
}

type WorkerReply struct {
	Code       Code
	MP3        []byte
	Boundaries []Boundary
}

func (a *Azure) Synthesize(ctx context.Context, request Request) (Result, error) {
	catalog, err := a.ListVoices(ctx, request.Language)
	if err != nil {
		return Result{}, err
	}
	if err := validate(request, catalog); err != nil {
		return Result{}, err
	}
	doc := ssml(request)
	input, err := json.Marshal(WorkerRequest{a.key, a.region, doc})
	if err != nil {
		return Result{}, &Error{Failed}
	}
	ctx, cancel := context.WithTimeout(ctx, a.timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, a.worker)
	cmd.Stdin = bytes.NewReader(input)
	cmd.Cancel = func() error { return cmd.Process.Signal(os.Interrupt) }
	cmd.WaitDelay = 2 * time.Second
	// The worker needs the native library path. Inherit no database or app credentials.
	cmd.Env = []string{"LD_LIBRARY_PATH=" + os.Getenv("LD_LIBRARY_PATH")}
	output, err := cmd.Output()
	if err := contextError(ctx); err != nil {
		return Result{}, err
	}
	if errors.Is(err, fs.ErrNotExist) || errors.Is(err, fs.ErrPermission) {
		return Result{}, &Error{Unavailable}
	}
	var reply WorkerReply
	if err != nil || json.Unmarshal(output, &reply) != nil {
		return Result{}, &Error{Failed}
	}
	switch reply.Code {
	case "":
		if len(reply.MP3) > 0 {
			return Result{MP3: reply.MP3, Boundaries: reply.Boundaries, BoundaryText: doc}, nil
		}
	case Unavailable, RateLimited, Timeout, Canceled:
		return Result{}, &Error{reply.Code}
	}
	return Result{}, &Error{Failed}
}
