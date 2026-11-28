package persistence

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"slices"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"taotale/backend/internal/fingerprint"
	"taotale/backend/internal/media"
	"taotale/backend/internal/persistence/queries"
)

var ErrPublicationIncomplete = errors.New("publication requires a title and source text")

func (s Library) PublishDraft(ctx context.Context, id pgtype.UUID, version int32, pin pgtype.UUID) error {
	return pgx.BeginFunc(ctx, s.Pool, func(tx pgx.Tx) error {
		q := queries.New(tx)
		item, err := q.LockItem(ctx, id)
		if err != nil {
			return missing(err, ErrItemNotFound)
		}
		if item.Version != version {
			return &VersionConflict{item.Version}
		}
		return s.publish(ctx, q, item, pin)
	})
}

// The item is either newly inserted or locked for update by the caller.
func (s Library) publish(ctx context.Context, q *queries.Queries, item queries.LibraryItem, pin pgtype.UUID) error {
	if item.PublishedAt.Valid {
		return ErrInvalidDraft
	}
	if strings.TrimSpace(item.Title.String) == "" || strings.TrimSpace(item.SourceText) == "" {
		return ErrPublicationIncomplete
	}
	audio, err := s.publicationAudio(ctx, q, item, pgtype.UUID{}, pin)
	if err != nil {
		return err
	}
	count, err := q.PublishItem(ctx, queries.PublishItemParams{
		ItemID: item.ID, ExpectedVersion: item.Version, AssetID: audio.ID,
		VoiceID:      pgtype.Text{String: audio.VoiceID, Valid: true},
		SpeakingRate: pgtype.Float8{Float64: audio.SpeakingRate, Valid: true},
	})
	if err != nil {
		return err
	}
	if count != 1 {
		return ErrInvalidDraft
	}
	_, err = reconcileVocabulary(ctx, q, item)
	return err
}

func (s Library) publicationAudio(ctx context.Context, q *queries.Queries, item queries.LibraryItem, revisionID, pin pgtype.UUID) (queries.LockPublicationAudioRow, error) {
	var audio queries.LockPublicationAudioRow
	identity, err := s.identity(item)
	if err != nil {
		return audio, ErrAudioNotReady
	}
	sum, err := identity.Sum()
	if err != nil {
		return audio, err
	}
	audio, err = q.LockPublicationAudio(ctx, queries.LockPublicationAudioParams{
		RevisionID: revisionID,
		ItemID:     item.ID, UserID: item.UserID, SourceText: item.SourceText, SourceFingerprint: sum,
		LanguageCode: identity.LanguageCode, ProviderID: identity.ProviderID, VoiceID: identity.VoiceID,
		SpeakingRate: identity.SpeakingRate, StrategyID: identity.StrategyID, StrategyVersion: identity.StrategyVersion,
		FingerprintFormatVersion: fingerprint.SchemaVersion,
	})
	if err != nil {
		return audio, missing(err, ErrAudioNotReady)
	}
	if pin.Valid && pin != audio.ID {
		return audio, ErrAudioNotReady
	}
	if !s.audioPresent(audio.ID, audio.StorageKey, audio.ByteSize) {
		return audio, media.ErrAudioUnavailable
	}
	return audio, nil
}

type NewItem struct {
	Item      queries.AddItemParams
	TopicIDs  []pgtype.UUID
	PreviewID pgtype.UUID
}

func (input NewItem) intentHash() (string, error) {
	// Topic membership is a set. Treat omitted/empty and permuted IDs alike.
	input.TopicIDs = append([]pgtype.UUID{}, input.TopicIDs...)
	slices.SortFunc(input.TopicIDs, func(a, b pgtype.UUID) int { return strings.Compare(a.String(), b.String()) })
	// Older receipts hashed an Item object without a cover key. Keep its
	// field order and omit the new key when the request has no cover.
	type itemIntent struct {
		queries.AddItemParams
		CoverImageID *pgtype.UUID `json:",omitempty"`
	}
	item := itemIntent{AddItemParams: input.Item}
	if input.Item.CoverImageID.Valid {
		item.CoverImageID = &input.Item.CoverImageID
	}
	body, err := json.Marshal(struct {
		Item      itemIntent
		TopicIDs  []pgtype.UUID
		PreviewID pgtype.UUID
	}{item, input.TopicIDs, input.PreviewID})
	if err != nil {
		return "", err
	}
	digest := sha256.Sum256(body)
	return hex.EncodeToString(digest[:]), nil
}

// These outcomes are durable for an admitted intent. Storage and media failures
// roll back the entire transaction so the original request can be retried.
var publicationErrors = map[string]error{
	"PREVIEW_NOT_FOUND":    ErrPreviewNotFound,
	"PREVIEW_MISMATCH":     ErrPreviewMismatch,
	"AUDIO_NOT_READY":      ErrAudioNotReady,
	"TOPIC_NOT_FOUND":      ErrTopicNotFound,
	"INCOMPLETE":           ErrPublicationIncomplete,
	"UNSUPPORTED_LANGUAGE": ErrUnsupportedLanguage,
}

func (s Library) PublishPreview(ctx context.Context, key pgtype.UUID, input NewItem) (pgtype.UUID, error) {
	if !input.PreviewID.Valid {
		return pgtype.UUID{}, ErrPreviewNotFound
	}
	intent, err := input.intentHash()
	if err != nil {
		return pgtype.UUID{}, err
	}
	var id pgtype.UUID
	var outcome error
	err = pgx.BeginFunc(ctx, s.Pool, func(tx pgx.Tx) error {
		q := queries.New(tx)
		owner, err := q.LockOwner(ctx)
		if err != nil {
			return err
		}
		receipt, err := operationReceipt(ctx, q, owner, key, "publish_preview", intent)
		if err != nil {
			return err
		}
		if receipt.Operation != "" {
			id = receipt.PublishedItemID
			if receipt.PublicationError.Valid {
				outcome = publicationErrors[receipt.PublicationError.String]
				if outcome == nil {
					return errors.New("unknown publication receipt outcome")
				}
			}
			return nil
		}
		// A rejected claim must roll back the item and its topics, while
		// allowing the outer transaction to remember the rejection.
		outcome = pgx.BeginFunc(ctx, tx, func(savepoint pgx.Tx) error {
			if strings.TrimSpace(input.Item.Title.String) == "" || strings.TrimSpace(input.Item.SourceText) == "" {
				return ErrPublicationIncomplete
			}
			writes := queries.New(savepoint)
			item, err := s.insertDraft(ctx, writes, input.Item, input.TopicIDs, input.PreviewID)
			if err != nil {
				return err
			}
			if err := s.publish(ctx, writes, item, pgtype.UUID{}); err != nil {
				return err
			}
			id = item.ID
			return nil
		})
		record := queries.RecordPublicationParams{UserID: owner, Key: key, RequestHash: intent, PublishedItemID: id}
		if outcome != nil {
			for code, cause := range publicationErrors {
				if errors.Is(outcome, cause) {
					record.PublicationError = pgtype.Text{String: code, Valid: true}
					break
				}
			}
			if !record.PublicationError.Valid {
				return outcome
			}
		}
		return q.RecordPublication(ctx, record)
	})
	if err != nil {
		return pgtype.UUID{}, err
	}
	return id, outcome
}
