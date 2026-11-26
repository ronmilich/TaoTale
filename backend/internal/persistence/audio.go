package persistence

import (
	"context"
	"os"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"taotale/backend/internal/media"
	"taotale/backend/internal/persistence/queries"
)

func (s Library) audioPresent(id pgtype.UUID, key string, size int64) bool {
	if !id.Valid {
		return false
	}
	f, err := media.OpenAudio(s.MediaRoot, key, size)
	if err != nil {
		return false
	}
	f.Close()
	return true
}

func (s Library) OpenItemAudio(ctx context.Context, itemID, assetID pgtype.UUID) (*os.File, int64, error) {
	var storagePath string
	var byteSize int64
	err := pgx.BeginTxFunc(ctx, s.Pool, snapshot, func(tx pgx.Tx) error {
		q := queries.New(tx)
		item, err := q.GetItem(ctx, itemID)
		if err != nil {
			return missing(err, ErrItemNotFound)
		}
		if item.PublishedAt.Valid {
			rows, err := q.PublishedAudioAssets(ctx, []pgtype.UUID{itemID})
			if err != nil {
				return err
			}
			if len(rows) == 0 || rows[0].AssetID != assetID {
				return ErrAudioNotFound
			}
			storagePath = rows[0].StorageKey
			byteSize = rows[0].ByteSize
			return nil
		}
		rows, err := q.LatestDraftCandidates(ctx, []pgtype.UUID{itemID})
		if err != nil {
			return err
		}
		if len(rows) == 0 || rows[0].Status != "READY" || rows[0].AssetID != assetID {
			return ErrAudioNotFound
		}
		row := rows[0]
		if !s.matchesDraft(item, row.SourceFingerprint) {
			return ErrAudioNotFound
		}
		storagePath = row.StorageKey.String
		byteSize = row.ByteSize.Int64
		return nil
	})
	if err != nil {
		return nil, 0, err
	}
	file, err := media.OpenAudio(s.MediaRoot, storagePath, byteSize)
	return file, byteSize, err
}
