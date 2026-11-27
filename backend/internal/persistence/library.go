package persistence

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"taotale/backend/internal/persistence/queries"
	"taotale/backend/internal/tts"
)

type Library struct {
	Pool      *pgxpool.Pool
	Voices    tts.VoiceCatalog
	MediaRoot string
	Limits    GenerationLimits
}

var (
	ErrItemNotFound        = errors.New("item not found")
	ErrTopicNotFound       = errors.New("topic not found")
	ErrUnsupportedLanguage = errors.New("learning language unavailable")
	ErrDefaultTopic        = errors.New("default topic is protected")
	ErrPublishedSource     = errors.New("published synthesis inputs require a revision")
	ErrInvalidVoice        = errors.New("voice is incompatible with the learning language")
)

type VersionConflict struct{ Current int32 }

func (e *VersionConflict) Error() string { return "item version conflict" }

type DuplicateTopic struct{ ID pgtype.UUID }

func (e *DuplicateTopic) Error() string { return "duplicate topic name" }

type Item struct {
	queries.LibraryItem
	TopicIDs   []pgtype.UUID
	Audio      Candidate
	Transcript *queries.GetTranscriptRow
}

// Write transactions serialize updates to an item and its topic associations.
func (s Library) CreateItem(ctx context.Context, input queries.AddItemParams, topics []pgtype.UUID, previewID pgtype.UUID) (Item, error) {
	var result Item
	err := pgx.BeginFunc(ctx, s.Pool, func(tx pgx.Tx) error {
		q := queries.New(tx)
		row, err := s.insertDraft(ctx, q, input, topics, previewID)
		if err != nil {
			return err
		}
		result, err = s.loadItem(ctx, q, row)
		return err
	})
	return result, err
}

func (s Library) insertDraft(ctx context.Context, q *queries.Queries, input queries.AddItemParams, topics []pgtype.UUID, previewID pgtype.UUID) (queries.LibraryItem, error) {
	if err := s.claimCover(ctx, q, input.CoverImageID); err != nil {
		return queries.LibraryItem{}, err
	}
	row, err := q.AddItem(ctx, input)
	if err != nil {
		return row, missing(err, ErrUnsupportedLanguage)
	}
	if err := insertItemTopics(ctx, q, row.ID, topics); err != nil {
		return row, err
	}
	if previewID.Valid {
		err = s.claimPreview(ctx, q, row, previewID)
	}
	return row, err
}

type ItemPatch struct {
	ExpectedVersion                              int32
	PreviewID                                    pgtype.UUID
	Title, Description, Notes, Difficulty, Voice *pgtype.Text
	SourceText                                   *string
	Rate                                         *pgtype.Float8
	TopicIDs                                     *[]pgtype.UUID
	CoverImageID                                 *pgtype.UUID
}

func (s Library) PatchItem(ctx context.Context, id pgtype.UUID, patch ItemPatch) (Item, error) {
	var result Item
	err := pgx.BeginFunc(ctx, s.Pool, func(tx pgx.Tx) error {
		q := queries.New(tx)
		row, err := q.LockItem(ctx, id)
		if err != nil {
			return missing(err, ErrItemNotFound)
		}
		if row.Version != patch.ExpectedVersion {
			return &VersionConflict{row.Version}
		}
		if row.PublishedAt.Valid && patch.Title != nil && strings.TrimSpace(patch.Title.String) == "" {
			return ErrPublicationIncomplete
		}
		if row.PublishedAt.Valid && ((patch.SourceText != nil && *patch.SourceText != row.SourceText) ||
			(patch.Voice != nil && *patch.Voice != row.VoiceID) || (patch.Rate != nil && *patch.Rate != row.SpeakingRate)) {
			return ErrPublishedSource
		}
		if patch.Voice != nil && patch.Voice.Valid && !tts.AllowedVoice(row.LanguageCode, s.Voices.ProviderID(), patch.Voice.String) {
			return ErrInvalidVoice
		}
		if patch.CoverImageID != nil && *patch.CoverImageID != row.CoverImageID {
			if err := s.claimCover(ctx, q, *patch.CoverImageID); err != nil {
				return err
			}
		}
		row, err = q.UpdateItem(ctx, queries.UpdateItemParams{
			ID: id, ExpectedVersion: patch.ExpectedVersion, Title: valueOr(patch.Title, row.Title),
			SourceText: valueOr(patch.SourceText, row.SourceText), Description: valueOr(patch.Description, row.Description),
			Notes: valueOr(patch.Notes, row.Notes), Difficulty: valueOr(patch.Difficulty, row.Difficulty),
			VoiceID:      valueOr(patch.Voice, row.VoiceID),
			SpeakingRate: valueOr(patch.Rate, row.SpeakingRate),
			CoverImageID: valueOr(patch.CoverImageID, row.CoverImageID),
		})
		if err != nil {
			return err
		}
		if patch.TopicIDs != nil {
			if len(*patch.TopicIDs) > 0 {
				locked, err := q.LockOwnedTopics(ctx, *patch.TopicIDs)
				if err != nil {
					return err
				}
				if len(locked) != len(*patch.TopicIDs) {
					return ErrTopicNotFound
				}
			}
			if err := q.ClearItemTopics(ctx, id); err != nil {
				return err
			}
			if err := insertItemTopics(ctx, q, id, *patch.TopicIDs); err != nil {
				return err
			}
		}
		if patch.PreviewID.Valid {
			if err := s.claimPreview(ctx, q, row, patch.PreviewID); err != nil {
				return err
			}
		}
		result, err = s.loadItem(ctx, q, row)
		return err
	})
	return result, err
}

func (s Library) SetItemFavorite(ctx context.Context, id pgtype.UUID, favorite bool) (queries.SetItemFavoriteRow, error) {
	row, err := queries.New(s.Pool).SetItemFavorite(ctx, queries.SetItemFavoriteParams{ID: id, IsFavorite: favorite})
	return row, missing(err, ErrItemNotFound)
}

func valueOr[T any](value *T, fallback T) T {
	if value != nil {
		return *value
	}
	return fallback
}

func missing(err, notFound error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return notFound
	}
	return err
}

func insertItemTopics(ctx context.Context, q *queries.Queries, id pgtype.UUID, topics []pgtype.UUID) error {
	if len(topics) == 0 {
		return nil
	}
	count, err := q.AddItemTopics(ctx, queries.AddItemTopicsParams{ItemID: id, TopicIds: topics})
	var foreignKey *pgconn.PgError
	if (err == nil && count != int64(len(topics))) || (errors.As(err, &foreignKey) && foreignKey.Code == "23503") {
		return ErrTopicNotFound
	}
	return err
}

func (s Library) GetItem(ctx context.Context, id pgtype.UUID) (Item, error) {
	var result Item
	err := pgx.BeginTxFunc(ctx, s.Pool, snapshot, func(tx pgx.Tx) error {
		q := queries.New(tx)
		row, err := q.GetItem(ctx, id)
		if err != nil {
			return missing(err, ErrItemNotFound)
		}
		result, err = s.loadItem(ctx, q, row)
		if err != nil || !row.PublishedAt.Valid || !result.Audio.AssetID.Valid {
			return err
		}
		transcript, err := q.GetTranscript(ctx, queries.GetTranscriptParams{
			UserID: row.UserID, AssetID: result.Audio.AssetID,
		})
		if err == nil {
			result.Transcript = &transcript
		}
		return err
	})
	return result, err
}

func (s Library) DeleteItem(ctx context.Context, id pgtype.UUID) error {
	count, err := queries.New(s.Pool).DeleteItem(ctx, id)
	if err == nil && count == 0 {
		return ErrItemNotFound
	}
	return err
}

var snapshot = pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly}

func (s Library) ListItems(ctx context.Context, params queries.PageItemsParams) ([]Item, int64, error) {
	var result []Item
	var total int64
	err := pgx.BeginTxFunc(ctx, s.Pool, snapshot, func(tx pgx.Tx) error {
		q := queries.New(tx)
		var err error
		total, err = q.CountItems(ctx, queries.CountItemsParams{
			LanguageCode: params.LanguageCode, TitlePattern: params.TitlePattern, Status: params.Status,
			Difficulty: params.Difficulty, FavoritesOnly: params.FavoritesOnly, TopicIds: params.TopicIds,
		})
		if err != nil {
			return err
		}
		rows, err := q.PageItems(ctx, params)
		if err != nil {
			return err
		}
		result, err = s.loadItems(ctx, q, rows)
		return err
	})
	return result, total, err
}

func (s Library) loadItem(ctx context.Context, q *queries.Queries, row queries.LibraryItem) (Item, error) {
	items, err := s.loadItems(ctx, q, []queries.LibraryItem{row})
	if err != nil {
		return Item{}, err
	}
	return items[0], nil
}

func (s Library) loadItems(ctx context.Context, q *queries.Queries, rows []queries.LibraryItem) ([]Item, error) {
	items := make([]Item, len(rows))
	ids := make([]pgtype.UUID, len(rows))
	var drafts, published []pgtype.UUID
	positions := make(map[pgtype.UUID]int, len(rows))
	for i, row := range rows {
		ids[i] = row.ID
		positions[row.ID] = i
		items[i] = Item{LibraryItem: row, TopicIDs: []pgtype.UUID{}, Audio: Candidate{Status: "NOT_GENERATED"}}
		if !row.PublishedAt.Valid {
			drafts = append(drafts, row.ID)
		} else if row.ActiveAudioAssetID.Valid {
			published = append(published, row.ID)
			items[i].Audio.Status = "FAILED"
		}
	}
	if len(rows) == 0 {
		return items, nil
	}
	joins, err := q.GetItemTopics(ctx, ids)
	if err != nil {
		return nil, err
	}
	for _, join := range joins {
		i := positions[join.ItemID]
		items[i].TopicIDs = append(items[i].TopicIDs, join.TopicID)
	}
	if len(drafts) != 0 {
		candidates, err := q.LatestDraftCandidates(ctx, drafts)
		if err != nil {
			return nil, err
		}
		for _, row := range candidates {
			item := &items[positions[row.ItemID]]
			item.Audio = s.projectCandidate(item.LibraryItem, row)
		}
	}
	if len(published) != 0 {
		assets, err := q.PublishedAudioAssets(ctx, published)
		if err != nil {
			return nil, err
		}
		for _, row := range assets {
			if s.audioPresent(row.AssetID, row.StorageKey, row.ByteSize) {
				items[positions[row.ItemID]].Audio = Candidate{Status: "READY", AssetID: row.AssetID}
			}
		}
	}
	return items, nil
}
