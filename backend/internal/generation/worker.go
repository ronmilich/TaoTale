// Package generation runs the durable speech synthesis queue.
package generation

import (
	"context"
	"errors"
	"os"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"taotale/backend/internal/fingerprint"
	"taotale/backend/internal/persistence/queries"
	"taotale/backend/internal/tts"
)

const interrupted = "GENERATION_INTERRUPTED"
const storageError = "STORAGE_ERROR"

type Worker struct {
	pool      *pgxpool.Pool
	leader    *pgx.Conn
	root      *os.Root
	provider  tts.TTSProvider
	heartbeat time.Duration
}

// Open holds a dedicated session lock until Close. Only that session may
// claim jobs or reconcile the previous process's uncertain outcomes.
func Open(ctx context.Context, pool *pgxpool.Pool, mediaRoot string, provider tts.TTSProvider) (*Worker, error) {
	conn, err := pgx.ConnectConfig(ctx, pool.Config().ConnConfig)
	if err != nil {
		return nil, errors.New("cannot connect generation worker")
	}
	q := queries.New(conn)
	owned, err := q.AcquireWorkerLeadership(ctx)
	if err != nil || !owned {
		_ = conn.Close(ctx)
		return nil, errors.New("cannot acquire generation worker lock")
	}
	root, err := os.OpenRoot(mediaRoot)
	if err != nil {
		_ = conn.Close(ctx)
		return nil, errors.New("cannot open audio storage")
	}
	w := &Worker{pool: pool, leader: conn, root: root, provider: provider, heartbeat: 5 * time.Second}
	if err := q.InterruptUnfinishedJobs(ctx); err != nil {
		w.Close()
		return nil, errors.New("cannot reconcile interrupted generations")
	}
	if err := w.cleanOrphans(ctx); err != nil {
		w.Close()
		return nil, errors.New("cannot reconcile generation files")
	}
	return w, nil
}

func (w *Worker) Close() {
	_ = w.root.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	_ = w.leader.Close(ctx)
}

func (w *Worker) Run(ctx context.Context, limit int64) error {
	if limit < 1 {
		return errors.New("invalid worker concurrency")
	}
	ctx, cancel := context.WithCancel(ctx)
	var jobs sync.WaitGroup
	defer func() { cancel(); jobs.Wait() }()
	done := make(chan error)
	tick := time.NewTicker(500 * time.Millisecond)
	defer tick.Stop()
	var active int64
	for ctx.Err() == nil {
		if active < limit {
			job, err := queries.New(w.leader).ClaimGeneration(ctx)
			switch {
			case err == nil:
				active++
				jobs.Go(func() {
					err := w.process(ctx, job)
					select {
					case done <- err:
					case <-ctx.Done():
					}
				})
				continue
			case ctx.Err() != nil:
				return nil
			case !errors.Is(err, pgx.ErrNoRows):
				return errors.New("generation queue connection failed")
			}
		}
		select {
		case <-ctx.Done():
			return nil
		case <-tick.C:
		case err := <-done:
			active--
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (w *Worker) process(parent context.Context, job queries.AudioGeneration) error {
	ctx, cancel := context.WithCancel(parent)
	watched := make(chan struct{})
	go func() {
		defer close(watched)
		tick := time.NewTicker(w.heartbeat)
		defer tick.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-tick.C:
				n, err := queries.New(w.pool).HeartbeatGeneration(ctx, job.ID)
				if err != nil || n == 0 {
					cancel()
					return
				}
			}
		}
	}()
	code := w.generate(ctx, job)
	if ctx.Err() != nil {
		code = interrupted
	}
	cancel()
	<-watched
	if code == "" {
		return nil
	}
	// Finish shutdown and ambiguous commits conditionally. A job already
	// committed as READY or removed with its draft cannot be overwritten.
	finish, stop := context.WithTimeout(context.WithoutCancel(parent), 3*time.Second)
	defer stop()
	if err := queries.New(w.pool).FailGeneration(finish, queries.FailGenerationParams{ID: job.ID, Code: code}); err != nil {
		return errors.New("cannot persist generation outcome")
	}
	return nil
}

func (w *Worker) generate(ctx context.Context, job queries.AudioGeneration) string {
	if job.ExpiresAt.Valid && !job.StartedAt.Time.Before(job.ExpiresAt.Time) {
		return "PREVIEW_EXPIRED"
	}
	id := fingerprint.Identity{LanguageCode: job.LanguageCode, ProviderID: job.ProviderID, VoiceID: job.VoiceID,
		SpeakingRate: job.SpeakingRate, StrategyID: job.StrategyID, StrategyVersion: job.StrategyVersion, ExactSourceText: job.SourceText}
	sum, err := id.Sum()
	if err != nil || job.FingerprintFormatVersion != fingerprint.SchemaVersion || sum != job.SourceFingerprint {
		return "ALIGNMENT_UNRELIABLE"
	}
	if job.ProviderID != w.provider.ProviderID() {
		return string(tts.Unavailable)
	}
	request := tts.Request{Language: job.LanguageCode, VoiceID: job.VoiceID, Source: job.SourceText, Rate: job.SpeakingRate}
	audio, err := w.provider.Synthesize(ctx, request)
	if err != nil {
		return string(tts.CodeOf(err))
	}
	return w.store(ctx, job, id, audio)
}
