package persistence

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"path"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	dbschema "taotale/backend/db"
	"taotale/backend/internal/persistence/queries"
)

// Keep this key stable across releases so overlapping instances serialize startup.
const schemaLock int64 = 0x54616f54616c6501

var ErrSchemaHistoryMismatch = errors.New("database migration history does not match this build")

type Schema struct {
	migrations []migration
}

func (s Schema) Revision() int64 { return s.migrations[len(s.migrations)-1].version }

type migration struct {
	version  int64
	checksum string
	sql      string
}

func LoadSchema() (Schema, error) {
	files, err := fs.ReadDir(dbschema.Migrations, "migrations")
	if err != nil {
		return Schema{}, err
	}
	schema := Schema{}
	var previous int64
	for _, file := range files {
		number, _, ok := strings.Cut(file.Name(), "_")
		version, err := strconv.ParseInt(number, 10, 64)
		if !ok || err != nil || version <= previous {
			return Schema{}, fmt.Errorf("invalid migration order at %q", file.Name())
		}
		body, err := dbschema.Migrations.ReadFile(path.Join("migrations", file.Name()))
		if err != nil {
			return Schema{}, err
		}
		// Use the same checksum for source checkouts on any supported OS.
		sql := strings.ReplaceAll(string(body), "\r\n", "\n")
		checksum := sha256.Sum256([]byte(sql))
		schema.migrations = append(schema.migrations, migration{version, hex.EncodeToString(checksum[:]), sql})
		previous = version
	}
	if len(schema.migrations) == 0 {
		return Schema{}, errors.New("no embedded migrations")
	}
	return schema, nil
}

// Prepare applies pending forward migrations and seeds in one transaction.
// A failing migration or seed leaves the database at its previous version.
func (s Schema) Prepare(ctx context.Context, pool *pgxpool.Pool) error {
	return pgx.BeginTxFunc(ctx, pool, pgx.TxOptions{IsoLevel: pgx.ReadCommitted}, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, "SELECT pg_advisory_xact_lock($1)", schemaLock); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `
			CREATE TABLE IF NOT EXISTS public.schema_migrations (
			    version bigint PRIMARY KEY CHECK (version > 0),
			    checksum text NOT NULL CHECK (checksum ~ '^[0-9a-f]{64}$'),
			    applied_at timestamptz NOT NULL DEFAULT now()
			)`); err != nil {
			return err
		}
		applied, err := s.appliedCount(ctx, tx)
		if err != nil {
			return err
		}
		for _, next := range s.migrations[applied:] {
			if _, err := tx.Exec(ctx, next.sql, pgx.QueryExecModeSimpleProtocol); err != nil {
				return fmt.Errorf("apply migration %d: %w", next.version, err)
			}
			if _, err := tx.Exec(ctx, "INSERT INTO public.schema_migrations (version, checksum) VALUES ($1, $2)", next.version, next.checksum); err != nil {
				return err
			}
		}
		return seedCatalog(ctx, queries.New(tx))
	})
}

// Check uses a read-only history query. It rejects missing, altered and newer migrations.
func (s Schema) Check(ctx context.Context, pool *pgxpool.Pool) error {
	count, err := s.appliedCount(ctx, pool)
	if err != nil {
		return err
	}
	if count != len(s.migrations) {
		return errors.New("pending schema migrations")
	}
	return nil
}

func (s Schema) appliedCount(ctx context.Context, db queries.DBTX) (int, error) {
	rows, err := db.Query(ctx, "SELECT version, checksum FROM public.schema_migrations ORDER BY version")
	if err != nil {
		return 0, err
	}
	defer rows.Close()
	count := 0
	for rows.Next() {
		var version int64
		var checksum string
		if err := rows.Scan(&version, &checksum); err != nil {
			return 0, err
		}
		if count >= len(s.migrations) || version != s.migrations[count].version || checksum != s.migrations[count].checksum {
			return 0, fmt.Errorf("%w at version %d", ErrSchemaHistoryMismatch, version)
		}
		count++
	}
	return count, rows.Err()
}

func seedCatalog(ctx context.Context, q *queries.Queries) error {
	if err := q.SeedLocalOwner(ctx); err != nil {
		return err
	}
	ownerID, err := q.GetLocalOwnerID(ctx)
	if err != nil {
		return err
	}
	if err := q.SeedLearningLanguages(ctx); err != nil {
		return err
	}
	return q.SeedDefaultTopics(ctx, ownerID)
}
