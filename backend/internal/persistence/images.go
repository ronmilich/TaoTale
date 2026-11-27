package persistence

import (
	"context"
	"crypto/rand"
	"errors"
	"io"
	"os"
	"path/filepath"

	"github.com/jackc/pgx/v5/pgtype"
	"taotale/backend/internal/media"
	"taotale/backend/internal/persistence/queries"
)

var ErrCoverNotFound = errors.New("cover unavailable")

func (s Library) CreateCover(ctx context.Context, source io.Reader, contentType string) (queries.MediaImage, error) {
	cover, err := media.SaveCover(ctx, s.MediaRoot, source, contentType)
	if err != nil {
		return queries.MediaImage{}, err
	}
	var id pgtype.UUID
	rand.Read(id.Bytes[:])
	id.Bytes[6] = (id.Bytes[6] & 0x0f) | 0x40
	id.Bytes[8] = (id.Bytes[8] & 0x3f) | 0x80
	id.Valid = true
	// If the insert outcome is ambiguous, leave the immutable file for orphan reconciliation.
	return queries.New(s.Pool).InsertCoverImage(ctx, queries.InsertCoverImageParams{
		ID: id, StorageKey: cover.Key, ContentType: cover.ContentType, SizeBytes: cover.Size,
		Width: cover.Width, Height: cover.Height,
	})
}

func (s Library) Cover(ctx context.Context, id pgtype.UUID) (queries.MediaImage, *os.File, error) {
	record, err := queries.New(s.Pool).GetCoverImage(ctx, id)
	if err != nil {
		return record, nil, missing(err, ErrCoverNotFound)
	}
	file, err := s.openCover(record)
	return record, file, err
}

func (s Library) openCover(record queries.MediaImage) (*os.File, error) {
	dir := filepath.ToSlash(filepath.Dir(record.StorageKey))
	if dir != "images" && dir != "temporary/images" {
		return nil, ErrCoverNotFound
	}
	file, err := media.OpenAsset(s.MediaRoot, record.StorageKey, record.SizeBytes)
	if errors.Is(err, media.ErrAssetUnavailable) {
		return nil, ErrCoverNotFound
	}
	return file, err
}

func (s Library) claimCover(ctx context.Context, q *queries.Queries, id pgtype.UUID) error {
	if !id.Valid {
		return nil
	}
	record, err := q.ClaimCoverImage(ctx, id)
	if err != nil {
		return missing(err, ErrCoverNotFound)
	}
	file, err := s.openCover(record)
	if err != nil {
		return err
	}
	return file.Close()
}
