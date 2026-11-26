package persistence

import (
	"context"
	"errors"
	"os"
	"slices"
	"sync"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"taotale/backend/internal/persistence/queries"
)

func TestPostgresMigrateAndSeed(t *testing.T) {
	dsn := os.Getenv("TAOTALE_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("run scripts/test-db.sh for a disposable PostgreSQL")
	}
	config, err := pgxpool.ParseConfig(dsn)
	if err != nil || config.ConnConfig.Database != "taotale_test" {
		t.Fatal("integration tests require the disposable taotale_test database")
	}
	ctx := t.Context()
	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	var tables int
	if err := pool.QueryRow(ctx, "SELECT count(*) FROM information_schema.tables WHERE table_schema = 'public'").Scan(&tables); err != nil || tables != 0 {
		t.Fatalf("integration database must start empty, found %d tables, error %v", tables, err)
	}
	schema, err := LoadSchema()
	if err != nil {
		t.Fatal(err)
	}
	if schema.Check(ctx, pool) == nil {
		t.Fatal("empty database reported ready")
	}
	broken := Schema{migrations: append(slices.Clone(schema.migrations), migration{
		version: schema.migrations[len(schema.migrations)-1].version + 1,
		sql:     "SELECT * FROM public.absent_tao07_probe",
	})}
	if broken.Prepare(ctx, pool) == nil {
		t.Fatal("broken migration succeeded")
	}
	if err := pool.QueryRow(ctx, "SELECT count(*) FROM information_schema.tables WHERE table_schema = 'public'").Scan(&tables); err != nil || tables != 0 {
		t.Fatalf("failed migration did not roll back, found %d tables, error %v", tables, err)
	}

	results := make(chan error, 2)
	var calls sync.WaitGroup
	for range 2 {
		calls.Go(func() { results <- schema.Prepare(ctx, pool) })
	}
	calls.Wait()
	close(results)
	for err := range results {
		if err != nil {
			t.Fatal(err)
		}
	}
	if err := schema.Check(ctx, pool); err != nil {
		t.Fatal(err)
	}
	owner, err := queries.New(pool).GetLocalOwnerID(ctx)
	if err != nil || owner.String() != "06dc9132-13d3-42da-a09c-000000000001" {
		t.Fatalf("unexpected owner %v, error %v", owner, err)
	}
	var languages, defaults []string
	if err := pool.QueryRow(ctx, "SELECT ARRAY(SELECT code FROM public.learning_languages WHERE enabled ORDER BY code)").Scan(&languages); err != nil ||
		!slices.Equal(languages, []string{"ko-KR", "zh-CN"}) {
		t.Fatalf("unexpected enabled languages %v, error %v", languages, err)
	}
	if err := pool.QueryRow(ctx, "SELECT ARRAY(SELECT system_key FROM public.topics WHERE user_id = $1 ORDER BY system_key)", owner).Scan(&defaults); err != nil ||
		!slices.Equal(defaults, []string{"business", "culture", "daily_life", "health", "history", "science", "technology", "travel"}) {
		t.Fatalf("unexpected default topics %v, error %v", defaults, err)
	}

	expectDBError(t, ctx, pool, "23505", "INSERT INTO public.users (id) VALUES ('06dc9132-13d3-42da-a09c-000000000002')")
	expectDBError(t, ctx, pool, "23514", "DELETE FROM public.topics WHERE system_key = 'history'")
	expectDBError(t, ctx, pool, "23514", "UPDATE public.topics SET system_key = NULL WHERE system_key = 'history'")
	expectDBError(t, ctx, pool, "23514", "UPDATE public.topics SET dark_text = 'red' WHERE system_key = 'history'")
	expectDBError(t, ctx, pool, "23505", "UPDATE public.topics SET name_key = 'science' WHERE system_key = 'history'")

	if _, err := pool.Exec(ctx, "UPDATE public.users SET display_name = 'Edited owner' WHERE id = $1", owner); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `UPDATE public.topics SET name = 'Edited history', name_key = 'edited history',
		light_background = '#ECE7FA', light_text = '#554095', dark_background = '#3E345C',
		dark_text = '#E4D8FF', updated_at = now() WHERE system_key = 'history'`); err != nil {
		t.Fatal(err)
	}
	before := catalogSnapshot(t, ctx, pool)
	for range 2 {
		if err := schema.Prepare(ctx, pool); err != nil {
			t.Fatal(err)
		}
	}
	if after := catalogSnapshot(t, ctx, pool); after != before {
		t.Fatal("repeat migration or seed modified catalog rows")
	}

	last := schema.migrations[len(schema.migrations)-1]
	if _, err := pool.Exec(ctx, "UPDATE public.schema_migrations SET checksum = repeat('0', 64) WHERE version = $1", last.version); err != nil {
		t.Fatal(err)
	}
	if schema.Check(ctx, pool) == nil || schema.Prepare(ctx, pool) == nil {
		t.Fatal("modified migration checksum was accepted")
	}
	if _, err := pool.Exec(ctx, "UPDATE public.schema_migrations SET checksum = $2 WHERE version = $1", last.version, last.checksum); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, "INSERT INTO public.schema_migrations (version, checksum) VALUES ($1, $2)", last.version+1, last.checksum); err != nil {
		t.Fatal(err)
	}
	if schema.Check(ctx, pool) == nil || schema.Prepare(ctx, pool) == nil {
		t.Fatal("newer migration was accepted")
	}
	if _, err := pool.Exec(ctx, "DELETE FROM public.schema_migrations WHERE version = $1", last.version+1); err != nil {
		t.Fatal(err)
	}
	if err := schema.Check(ctx, pool); err != nil {
		t.Fatal(err)
	}
}

func expectDBError(t *testing.T, ctx context.Context, pool *pgxpool.Pool, code, sql string) {
	t.Helper()
	_, err := pool.Exec(ctx, sql)
	var failure *pgconn.PgError
	if !errors.As(err, &failure) || failure.Code != code {
		t.Fatalf("expected SQLSTATE %s, got %v", code, err)
	}
}

func catalogSnapshot(t *testing.T, ctx context.Context, pool *pgxpool.Pool) string {
	t.Helper()
	var snapshot string
	err := pool.QueryRow(ctx, `SELECT jsonb_build_array(
		(SELECT jsonb_agg(u ORDER BY id) FROM public.users AS u),
		(SELECT jsonb_agg(l ORDER BY code) FROM public.learning_languages AS l),
		(SELECT jsonb_agg(t ORDER BY id) FROM public.topics AS t),
		(SELECT jsonb_agg(m ORDER BY version) FROM public.schema_migrations AS m)
	)::text`).Scan(&snapshot)
	if err != nil {
		t.Fatal(err)
	}
	return snapshot
}
