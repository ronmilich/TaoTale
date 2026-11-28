package persistence

import (
	"context"
	"errors"
	"slices"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"golang.org/x/text/unicode/norm"
	"taotale/backend/internal/persistence/queries"
	"taotale/backend/internal/vocabmatch"
)

var (
	ErrVocabularyNotFound  = errors.New("vocabulary entry not found")
	ErrVocabularyLanguage  = errors.New("vocabulary entry and item languages differ")
	ErrVocabularySelection = errors.New("vocabulary selection contains unavailable entries")
)

type DuplicateVocabulary struct{ ID pgtype.UUID }

func (e *DuplicateVocabulary) Error() string { return "duplicate vocabulary word" }

// NormalizeVocabularyWord preserves display spelling after outer trim. The identity
// key uses NFC and collapses each run of Unicode whitespace to one ASCII space.
func NormalizeVocabularyWord(raw string) (word, key string) {
	word = strings.TrimSpace(raw)
	return word, strings.Join(strings.Fields(norm.NFC.String(word)), " ")
}

type Vocabulary struct {
	queries.VocabularyEntry
	TopicIDs []pgtype.UUID
}

type VocabularyDetail struct {
	Vocabulary
	References []queries.GetVocabularyReferencesRow
}

type ItemVocabularyEntry struct {
	Vocabulary
	Reference queries.VocabularyReference
}

type ItemVocabularyView struct {
	Items       []ItemVocabularyEntry
	SourceText  *string
	Annotations []vocabmatch.Annotation
}

func (s Library) CreateVocabulary(ctx context.Context, input queries.InsertVocabularyParams, topics []pgtype.UUID, itemID pgtype.UUID) (VocabularyDetail, error) {
	var result VocabularyDetail
	err := pgx.BeginFunc(ctx, s.Pool, func(tx pgx.Tx) error {
		q := queries.New(tx)
		if itemID.Valid {
			if err := checkVocabularySource(ctx, q, itemID, input.LanguageCode); err != nil {
				return err
			}
		}
		if err := lockVocabularyTopics(ctx, q, topics); err != nil {
			return err
		}
		row, err := q.InsertVocabulary(ctx, input)
		if err != nil {
			return missing(err, ErrUnsupportedLanguage)
		}
		if err := setVocabularyTopics(ctx, q, row.ID, topics); err != nil {
			return err
		}
		if itemID.Valid {
			if _, err := q.LinkVocabulary(ctx, queries.LinkVocabularyParams{ID: row.ID, ItemID: itemID}); err != nil {
				return err
			}
		}
		result, err = loadVocabularyDetail(ctx, q, row)
		return err
	})
	return result, s.vocabularyWriteError(ctx, err, input.LanguageCode, input.NormalizedWordKey)
}

func (s Library) PatchVocabulary(ctx context.Context, patch queries.UpdateVocabularyParams, topics *[]pgtype.UUID) (VocabularyDetail, error) {
	var result VocabularyDetail
	var language string
	err := pgx.BeginFunc(ctx, s.Pool, func(tx pgx.Tx) error {
		q := queries.New(tx)
		row, err := q.GetVocabulary(ctx, patch.ID)
		if err != nil {
			return missing(err, ErrVocabularyNotFound)
		}
		language = row.LanguageCode
		if topics != nil {
			if err := lockVocabularyTopics(ctx, q, *topics); err != nil {
				return err
			}
		}
		row, err = q.UpdateVocabulary(ctx, patch)
		if err != nil {
			return missing(err, ErrVocabularyNotFound)
		}
		if topics != nil {
			if err := q.ClearVocabularyTopics(ctx, row.ID); err != nil {
				return err
			}
			if err := setVocabularyTopics(ctx, q, row.ID, *topics); err != nil {
				return err
			}
		}
		result, err = loadVocabularyDetail(ctx, q, row)
		return err
	})
	return result, s.vocabularyWriteError(ctx, err, language, patch.NormalizedWordKey.String)
}

func (s Library) vocabularyWriteError(ctx context.Context, err error, language, key string) error {
	var violation *pgconn.PgError
	if !errors.As(err, &violation) || violation.Code != "23505" || violation.ConstraintName != "vocabulary_unique_word" {
		return err
	}
	existing, lookupErr := queries.New(s.Pool).FindVocabulary(ctx, queries.FindVocabularyParams{
		LanguageCode: language, NormalizedWordKey: key,
	})
	if lookupErr != nil && !errors.Is(lookupErr, pgx.ErrNoRows) {
		return lookupErr
	}
	return &DuplicateVocabulary{ID: existing.ID}
}

func lockVocabularyTopics(ctx context.Context, q *queries.Queries, topics []pgtype.UUID) error {
	if len(topics) == 0 {
		return nil
	}
	locked, err := q.LockOwnedTopics(ctx, topics)
	if err == nil && len(locked) != len(topics) {
		return ErrTopicNotFound
	}
	return err
}

func setVocabularyTopics(ctx context.Context, q *queries.Queries, id pgtype.UUID, topics []pgtype.UUID) error {
	if len(topics) == 0 {
		return nil
	}
	return q.AddVocabularyTopics(ctx, queries.AddVocabularyTopicsParams{ID: id, TopicIds: topics})
}

func (s Library) GetVocabulary(ctx context.Context, id pgtype.UUID) (VocabularyDetail, error) {
	var result VocabularyDetail
	err := pgx.BeginTxFunc(ctx, s.Pool, snapshot, func(tx pgx.Tx) error {
		q := queries.New(tx)
		row, err := q.GetVocabulary(ctx, id)
		if err != nil {
			return missing(err, ErrVocabularyNotFound)
		}
		result, err = loadVocabularyDetail(ctx, q, row)
		return err
	})
	return result, err
}

func (s Library) FindVocabulary(ctx context.Context, language, key string) (*Vocabulary, error) {
	var result *Vocabulary
	err := pgx.BeginTxFunc(ctx, s.Pool, snapshot, func(tx pgx.Tx) error {
		q := queries.New(tx)
		row, err := q.FindVocabulary(ctx, queries.FindVocabularyParams{LanguageCode: language, NormalizedWordKey: key})
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		if err != nil {
			return err
		}
		rows, err := loadVocabularyTopics(ctx, q, []queries.VocabularyEntry{row})
		if err == nil {
			result = &rows[0]
		}
		return err
	})
	return result, err
}

func (s Library) DeleteVocabulary(ctx context.Context, id pgtype.UUID) error {
	count, err := queries.New(s.Pool).DeleteVocabulary(ctx, id)
	if err == nil && count == 0 {
		return ErrVocabularyNotFound
	}
	return err
}

func checkVocabularySource(ctx context.Context, q *queries.Queries, itemID pgtype.UUID, language string) error {
	sourceLanguage, err := q.LockVocabularySource(ctx, itemID)
	if err != nil {
		return missing(err, ErrItemNotFound)
	}
	if sourceLanguage != language {
		return ErrVocabularyLanguage
	}
	return nil
}

func (s Library) ChangeVocabularyReference(ctx context.Context, id, itemID pgtype.UUID, link bool) (queries.VocabularyReference, error) {
	var result queries.VocabularyReference
	err := pgx.BeginFunc(ctx, s.Pool, func(tx pgx.Tx) error {
		q := queries.New(tx)
		itemLanguage, err := q.LockVocabularySource(ctx, itemID)
		if err != nil {
			return missing(err, ErrItemNotFound)
		}
		language, err := q.LockVocabularyLanguage(ctx, id)
		if err != nil {
			return missing(err, ErrVocabularyNotFound)
		}
		if language != itemLanguage {
			return ErrVocabularyLanguage
		}
		if !link {
			return q.UnlinkVocabulary(ctx, queries.UnlinkVocabularyParams{ID: id, ItemID: itemID})
		}
		result, err = q.LinkVocabulary(ctx, queries.LinkVocabularyParams{ID: id, ItemID: itemID})
		return err
	})
	return result, err
}

func (s Library) ItemVocabulary(ctx context.Context, id pgtype.UUID) (ItemVocabularyView, error) {
	var result ItemVocabularyView
	err := pgx.BeginTxFunc(ctx, s.Pool, snapshot, func(tx pgx.Tx) error {
		q := queries.New(tx)
		item, err := q.GetItem(ctx, id)
		if err != nil {
			return missing(err, ErrItemNotFound)
		}
		rows, err := q.ItemVocabulary(ctx, id)
		if err != nil {
			return err
		}
		entries := make([]queries.VocabularyEntry, 0, len(rows))
		for _, row := range rows {
			entries = append(entries, row.VocabularyEntry)
		}
		words, err := loadVocabularyTopics(ctx, q, entries)
		if err != nil {
			return err
		}
		for i, row := range rows {
			result.Items = append(result.Items, ItemVocabularyEntry{
				Vocabulary: words[i],
				Reference: queries.VocabularyReference{
					LibraryItemID: id, AddedAt: row.AddedAt, ManualLinked: row.ManualLinked, AutoLinked: row.AutoLinked,
				},
			})
		}
		if item.PublishedAt.Valid && vocabmatch.Enabled(item.LanguageCode) {
			result.SourceText = &item.SourceText
		}
		return nil
	})
	if err != nil {
		return result, err
	}
	// An automatic-only link can outlive a word's previous spelling until
	// the next sync. Show it in the Reader only while the exact match holds.
	result.Items = slices.DeleteFunc(result.Items, func(entry ItemVocabularyEntry) bool {
		return !entry.Reference.ManualLinked &&
			(result.SourceText == nil || !strings.Contains(*result.SourceText, entry.Word))
	})
	if result.SourceText != nil {
		words := make([]vocabmatch.Word, 0, len(result.Items))
		for _, word := range result.Items {
			words = append(words, vocabmatch.Word{ID: word.ID.String(), Text: word.Word})
		}
		result.Annotations = vocabmatch.Mandarin(*result.SourceText, words)
	}
	return result, nil
}

func loadVocabularyDetail(ctx context.Context, q *queries.Queries, row queries.VocabularyEntry) (VocabularyDetail, error) {
	words, err := loadVocabularyTopics(ctx, q, []queries.VocabularyEntry{row})
	if err != nil {
		return VocabularyDetail{}, err
	}
	refs, err := q.GetVocabularyReferences(ctx, row.ID)
	return VocabularyDetail{Vocabulary: words[0], References: refs}, err
}

func loadVocabularyTopics(ctx context.Context, q *queries.Queries, rows []queries.VocabularyEntry) ([]Vocabulary, error) {
	result := make([]Vocabulary, 0, len(rows))
	if len(rows) == 0 {
		return result, nil
	}
	ids := make([]pgtype.UUID, 0, len(rows))
	for _, row := range rows {
		ids = append(ids, row.ID)
	}
	joins, err := q.GetVocabularyTopics(ctx, ids)
	if err != nil {
		return nil, err
	}
	topics := make(map[pgtype.UUID][]pgtype.UUID, len(rows))
	for _, join := range joins {
		topics[join.VocabularyEntryID] = append(topics[join.VocabularyEntryID], join.TopicID)
	}
	for _, row := range rows {
		result = append(result, Vocabulary{VocabularyEntry: row, TopicIDs: append([]pgtype.UUID{}, topics[row.ID]...)})
	}
	return result, nil
}

func (s Library) ListVocabulary(ctx context.Context, params queries.PageVocabularyParams) ([]Vocabulary, int64, error) {
	var items []Vocabulary
	var total int64
	err := pgx.BeginTxFunc(ctx, s.Pool, snapshot, func(tx pgx.Tx) error {
		q := queries.New(tx)
		var err error
		total, err = q.CountVocabulary(ctx, queries.CountVocabularyParams{
			LanguageCode: params.LanguageCode, Status: params.Status, Pattern: params.Pattern, TopicIds: params.TopicIds,
		})
		if err != nil {
			return err
		}
		rows, err := q.PageVocabulary(ctx, params)
		if err == nil {
			items, err = loadVocabularyTopics(ctx, q, rows)
		}
		return err
	})
	return items, total, err
}

func (s Library) VocabularyStats(ctx context.Context, language string) (queries.VocabularyStatsRow, error) {
	return queries.New(s.Pool).VocabularyStats(ctx, language)
}

func (s Library) BulkVocabulary(ctx context.Context, language string, ids []pgtype.UUID, status *string) (int64, error) {
	var count int64
	err := pgx.BeginFunc(ctx, s.Pool, func(tx pgx.Tx) error {
		q := queries.New(tx)
		locked, err := q.LockVocabularySelection(ctx, queries.LockVocabularySelectionParams{LanguageCode: language, Ids: ids})
		if err != nil {
			return err
		}
		if len(locked) != len(ids) {
			return ErrVocabularySelection
		}
		if status == nil {
			count, err = q.DeleteVocabularySelection(ctx, queries.DeleteVocabularySelectionParams{LanguageCode: language, Ids: ids})
		} else {
			count, err = q.SetVocabularySelectionStatus(ctx, queries.SetVocabularySelectionStatusParams{LanguageCode: language, Ids: ids, Status: *status})
		}
		if err == nil && count != int64(len(ids)) {
			return ErrVocabularySelection
		}
		return err
	})
	return count, err
}
