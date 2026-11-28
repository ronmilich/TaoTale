package persistence

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"taotale/backend/internal/persistence/queries"
	"taotale/backend/internal/vocabmatch"
)

var (
	ErrVocabularyMatchingUnsupported = errors.New("automatic vocabulary matching is unsupported for this language")
	ErrVocabularySyncUnpublished     = errors.New("automatic vocabulary matching needs a published item")
)

func (s Library) SyncItemVocabulary(ctx context.Context, id pgtype.UUID) (int, error) {
	var matched int
	err := pgx.BeginFunc(ctx, s.Pool, func(tx pgx.Tx) error {
		q := queries.New(tx)
		item, err := q.LockItem(ctx, id)
		if err != nil {
			return missing(err, ErrItemNotFound)
		}
		if !vocabmatch.Enabled(item.LanguageCode) {
			return ErrVocabularyMatchingUnsupported
		}
		if !item.PublishedAt.Valid {
			return ErrVocabularySyncUnpublished
		}
		matched, err = reconcileVocabulary(ctx, q, item)
		return err
	})
	return matched, err
}

// The caller holds the item lock (or just inserted it), including on publication.
// Lock matching entries in ID order so the chosen spellings survive until commit.
func reconcileVocabulary(ctx context.Context, q *queries.Queries, item queries.LibraryItem) (int, error) {
	if !vocabmatch.Enabled(item.LanguageCode) {
		return 0, nil
	}
	ids, err := q.LockMatchingVocabulary(ctx, queries.LockMatchingVocabularyParams{
		UserID: item.UserID, LanguageCode: item.LanguageCode, SourceText: item.SourceText,
	})
	if err != nil {
		return 0, err
	}
	if err := q.RemoveObsoleteAutomaticVocabulary(ctx, queries.RemoveObsoleteAutomaticVocabularyParams{
		UserID: item.UserID, ItemID: item.ID, MatchingIds: ids,
	}); err != nil {
		return 0, err
	}
	if len(ids) == 0 {
		return 0, nil
	}
	err = q.AddAutomaticVocabulary(ctx, queries.AddAutomaticVocabularyParams{
		UserID: item.UserID, LanguageCode: item.LanguageCode, ItemID: item.ID, MatchingIds: ids,
	})
	return len(ids), err
}
