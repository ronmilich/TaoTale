package persistence

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"path/filepath"
	"slices"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"taotale/backend/internal/persistence/queries"
)

func (s Library) DeleteItem(ctx context.Context, id pgtype.UUID) error {
	var audioKeys []string
	var coverKey string
	err := pgx.BeginFunc(ctx, s.Pool, func(tx pgx.Tx) error {
		q := queries.New(tx)
		item, err := q.LockItem(ctx, id)
		if err != nil {
			return missing(err, ErrItemNotFound)
		}
		assetIDs, err := q.DeleteItemGenerations(ctx, queries.DeleteItemGenerationsParams{UserID: item.UserID, ItemID: id})
		if err != nil {
			return err
		}
		if _, err := q.DeleteItem(ctx, id); err != nil {
			return err
		}
		assetIDs = slices.DeleteFunc(append(assetIDs, item.ActiveAudioAssetID), func(id pgtype.UUID) bool {
			return !id.Valid
		})
		if len(assetIDs) > 0 {
			params := queries.LockDeletionAudioParams{UserID: item.UserID, AssetIds: assetIDs}
			if err := q.LockDeletionAudio(ctx, params); err != nil {
				return err
			}
			audioKeys, err = q.DeleteUnreferencedAudio(ctx, queries.DeleteUnreferencedAudioParams(params))
			if err != nil {
				return err
			}
		}
		if item.CoverImageID.Valid {
			params := queries.LockDeletionCoverParams{OwnerID: item.UserID, ID: item.CoverImageID}
			if err := q.LockDeletionCover(ctx, params); err != nil {
				return err
			}
			coverKey, err = q.DeleteUnreferencedCover(ctx, queries.DeleteUnreferencedCoverParams(params))
			if err != nil && !errors.Is(err, pgx.ErrNoRows) {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	// After commit, the deleted media rows can no longer acquire references.
	// A filesystem failure leaves an orphan for later cleanup.
	if err := s.releaseItemMedia(audioKeys, coverKey); err != nil {
		slog.WarnContext(ctx, "Could not release deleted item media", "item_id", id.String())
	}
	return nil
}

func (s Library) releaseItemMedia(audioKeys []string, coverKey string) error {
	if len(audioKeys) == 0 && coverKey == "" {
		return nil
	}
	root, err := os.OpenRoot(s.MediaRoot)
	if err != nil {
		return err
	}
	defer root.Close()
	remove := func(key string, dirs ...string) error {
		if !filepath.IsLocal(key) || filepath.ToSlash(filepath.Clean(key)) != key ||
			!slices.Contains(dirs, filepath.ToSlash(filepath.Dir(key))) {
			return errors.New("invalid media key")
		}
		info, err := root.Lstat(key)
		if err == nil && info.Mode().IsRegular() {
			err = root.Remove(key)
		}
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	for _, key := range audioKeys {
		err = errors.Join(err, remove(key, "audio", "temporary/audio"))
	}
	if coverKey != "" {
		err = errors.Join(err, remove(coverKey, "images", "temporary/images"))
	}
	return err
}
