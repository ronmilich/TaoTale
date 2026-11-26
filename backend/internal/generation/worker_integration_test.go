package generation

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"taotale/backend/internal/media"
	"taotale/backend/internal/persistence"
	"taotale/backend/internal/persistence/queries"
	"taotale/backend/internal/tts"
)

type testProvider struct {
	tts.Mock
	calls    atomic.Int32
	blockDir string
}

func (p *testProvider) Synthesize(ctx context.Context, r tts.Request) (tts.Result, error) {
	p.calls.Add(1)
	if p.blockDir != "" {
		if err := os.WriteFile(filepath.Join(p.blockDir, "provider-called"), []byte("1"), 0600); err != nil {
			return tts.Result{}, err
		}
		<-ctx.Done()
		return tts.Result{}, ctx.Err()
	}
	switch r.Source {
	case "provider error":
		return tts.Result{}, &tts.Error{Code: tts.Timeout}
	case "invalid audio":
		return tts.Result{MP3: []byte("invalid")}, nil
	}
	return p.Mock.Synthesize(ctx, r)
}

func testPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	dsn := os.Getenv("TAOTALE_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("requires disposable PostgreSQL")
	}
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil || cfg.ConnConfig.Database != "taotale_test" {
		t.Fatal("expected disposable taotale_test database")
	}
	pool, err := pgxpool.NewWithConfig(t.Context(), cfg)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	return pool
}

// The parent kills this process while it is inside Synthesize. Neither Go
// defers nor a graceful database failure update can run on that path.
func TestCrashWorkerChild(t *testing.T) {
	dir := os.Getenv("TAOTALE_TEST_CRASH_WORKER")
	if dir == "" {
		t.Skip("subprocess helper")
	}
	w, err := Open(t.Context(), testPool(t), dir, &testProvider{blockDir: dir})
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()
	w.heartbeat = 50 * time.Millisecond
	if err := os.WriteFile(filepath.Join(dir, "started"), nil, 0600); err != nil {
		t.Fatal(err)
	}
	if err := w.Run(t.Context(), 1); err != nil {
		t.Fatal(err)
	}
}

func until(t *testing.T, check func() bool) {
	t.Helper()
	deadline := time.Now().Add(15 * time.Second)
	for !check() {
		if time.Now().After(deadline) {
			t.Fatal("timed out waiting for worker")
		}
		time.Sleep(20 * time.Millisecond)
	}
}

func TestWorkerPostgres(t *testing.T) {
	pool := testPool(t)
	ctx := t.Context()
	dir := t.TempDir()
	if err := media.PrepareRoot(dir); err != nil {
		t.Fatal(err)
	}
	provider := &testProvider{}
	lib := persistence.Library{Pool: pool, Voices: provider, MediaRoot: dir, Limits: persistence.GenerationLimits{Active: 4, Hourly: 1000}}
	draft := func(source string) pgtype.UUID {
		t.Helper()
		item, err := lib.CreateItem(ctx, queries.AddItemParams{LanguageCode: "zh-CN", SourceText: source}, nil)
		if err != nil {
			t.Fatal(err)
		}
		return item.ID
	}
	admit := func(item, key pgtype.UUID) pgtype.UUID {
		t.Helper()
		id, err := lib.AdmitGeneration(ctx, item, key)
		if err != nil {
			t.Fatal(err)
		}
		return id
	}
	query := func(sql string, args ...any) {
		t.Helper()
		if _, err := pool.Exec(ctx, sql, args...); err != nil {
			t.Fatal(err)
		}
	}
	key := func(n byte) pgtype.UUID { return pgtype.UUID{Bytes: [16]byte{0x19, 15: n}, Valid: true} }
	await := func(item, id pgtype.UUID, status, code string) persistence.Candidate {
		t.Helper()
		var result persistence.Candidate
		until(t, func() bool {
			var err error
			result, err = lib.GetGeneration(ctx, item, id)
			if err != nil {
				t.Fatal(err)
			}
			if result.JobStatus == "FAILED" && result.ErrorCode.String != code {
				t.Fatalf("unexpected failure %+v", result)
			}
			return result.Status == status && result.ErrorCode.String == code
		})
		return result
	}
	command := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestCrashWorkerChild$")
	command.Env = append(os.Environ(), "TAOTALE_TEST_CRASH_WORKER="+dir)
	command.Stdout, command.Stderr = os.Stdout, os.Stderr
	if err := command.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if command.ProcessState == nil {
			_ = command.Process.Kill()
			_ = command.Wait()
		}
	}()
	until(t, func() bool { _, err := os.Stat(filepath.Join(dir, "started")); return err == nil })
	a, b := draft("你好，世界。"), draft("再次。")
	crashed := admit(a, key(1))
	until(t, func() bool {
		_, err := os.Stat(filepath.Join(dir, "provider-called"))
		var beat bool
		return err == nil && pool.QueryRow(ctx, "SELECT heartbeat_at > started_at FROM audio_generations WHERE id=$1", crashed).Scan(&beat) == nil && beat
	})
	if other, err := Open(ctx, pool, dir, provider); err == nil {
		other.Close()
		t.Fatal("second process reconciled a live worker")
	}
	pending := admit(b, key(2))
	if err := command.Process.Kill(); err != nil {
		t.Fatal(err)
	}
	if err := command.Wait(); err == nil {
		t.Fatal("expected abrupt exit")
	}
	stage := filepath.Join(dir, "staging", "tts-"+crashed.String()+".part")
	if err := os.WriteFile(stage, []byte("partial"), 0600); err != nil {
		t.Fatal(err)
	}
	w, err := Open(ctx, pool, dir, provider)
	if err != nil {
		t.Fatal(err)
	}
	stopCtx, cancel := context.WithCancel(ctx)
	done := make(chan error, 1)
	go func() { done <- w.Run(stopCtx, 2) }()
	stopped := false
	stop := func() {
		if stopped {
			return
		}
		stopped = true
		cancel()
		err := <-done
		w.Close()
		if err != nil {
			t.Fatal(err)
		}
	}
	defer stop()
	if _, err := os.Stat(stage); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("crash staging file survived", err)
	}
	await(a, crashed, "FAILED", interrupted)
	await(b, pending, "FAILED", interrupted)
	if admit(a, key(1)) != crashed || admit(b, key(2)) != pending {
		t.Fatal("replay after restart created jobs")
	}
	first := admit(a, key(3))
	for range 8 {
		if admit(a, key(3)) != first {
			t.Fatal("receipt changed")
		}
	}
	ready := await(a, first, "READY", "")
	if !ready.AssetID.Valid || provider.calls.Load() != 1 {
		t.Fatalf("asset %v, calls after restart %d", ready.AssetID, provider.calls.Load())
	}
	var storage string
	var duration int64
	var raw []byte
	if err := pool.QueryRow(ctx, `SELECT a.storage_key,a.duration_ms,a.segments FROM audio_assets a JOIN audio_generations g ON g.audio_asset_id=a.id
		WHERE g.id=$1 AND g.started_at IS NOT NULL AND g.heartbeat_at IS NOT NULL AND g.completed_at IS NOT NULL`, first).Scan(&storage, &duration, &raw); err != nil {
		t.Fatal(err)
	}
	var segments []map[string]any
	if err := json.Unmarshal(raw, &segments); err != nil || duration < 1000 || len(segments) != 2 || segments[0]["text"] != "你好，" || segments[0]["textStart"] != float64(0) || segments[0]["endMs"] != float64(500) {
		t.Fatalf("invalid verified asset %d %s", duration, raw)
	}
	path := filepath.Join(dir, storage)
	original, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	query("UPDATE library_items SET published_at=now() WHERE id=$1", a)
	// Deliberately target the published path and verify no bytes are replaced.
	if err := os.WriteFile(stage, []byte("different"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := w.promote("staging/"+filepath.Base(stage), storage); !errors.Is(err, os.ErrExist) {
		t.Fatal("expected promotion collision", err)
	}
	for i, tc := range []struct{ source, code string }{{"provider error", "TTS_TIMEOUT"}, {"invalid audio", "ALIGNMENT_UNRELIABLE"}} {
		item := draft(tc.source)
		await(item, admit(item, key(byte(i+4))), "FAILED", tc.code)
	}
	// Fail the READY update after file promotion and asset insertion.
	query("ALTER TABLE audio_generations ADD CONSTRAINT worker_test_rollback CHECK (item_id <> '" + b.String() + "' OR status <> 'READY') NOT VALID")
	defer query("ALTER TABLE audio_generations DROP CONSTRAINT worker_test_rollback")
	await(b, admit(b, key(6)), "FAILED", storageError)
	stop()
	w2, err := Open(ctx, pool, dir, provider)
	if err != nil {
		t.Fatal(err)
	}
	defer w2.Close()
	files, err := os.ReadDir(filepath.Join(dir, "audio"))
	if err != nil || len(files) != 1 || files[0].Name() != filepath.Base(storage) {
		t.Fatal("orphan asset after recovery", files, err)
	}
	staged, err := os.ReadDir(filepath.Join(dir, "staging"))
	if err != nil || len(staged) != 0 {
		t.Fatal("unreleased staging files", staged, err)
	}
	after, err := os.ReadFile(path)
	if err != nil || !bytes.Equal(original, after) {
		t.Fatal("failed promotion changed published media", err)
	}
	await(a, first, "READY", "")
}
