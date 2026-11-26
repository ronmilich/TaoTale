package persistence

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"taotale/backend/internal/media"
	"taotale/backend/internal/persistence/queries"
	"taotale/backend/internal/tts"
)

var (
	ErrPreviewNotFound = errors.New("preview not found")
	ErrPreviewMismatch = errors.New("preview synthesis inputs do not match the draft")
	ErrAudioNotReady   = errors.New("audio is not ready")
	ErrAudioNotFound   = errors.New("audio asset not found")
)

func (s Library) AdmitPreview(ctx context.Context, key pgtype.UUID, request tts.Request) (pgtype.UUID, error) {
	body, err := json.Marshal(request)
	if err != nil {
		return pgtype.UUID{}, err
	}
	digest := sha256.Sum256(body)
	intent := hex.EncodeToString(digest[:])

	var id pgtype.UUID
	err = pgx.BeginFunc(ctx, s.Pool, func(tx pgx.Tx) error {
		q := queries.New(tx)
		owner, err := q.LockOwner(ctx)
		if err != nil {
			return err
		}
		id, err = generationReceipt(ctx, q, owner, key, previewOperation, intent)
		if err != nil || id.Valid {
			return err
		}
		identity, err := s.identity(queries.LibraryItem{LanguageCode: request.Language, SourceText: request.Source,
			VoiceID: pgtype.Text{String: request.VoiceID, Valid: true}, SpeakingRate: pgtype.Float8{Float64: request.Rate, Valid: true}})
		if err != nil {
			return &tts.Error{Code: tts.Invalid}
		}
		if err := validateSynthesis(identity); err != nil {
			return err
		}
		itemID := pgtype.UUID{}
		if err := s.checkCapacity(ctx, q, owner, itemID); err != nil {
			return err
		}
		id, err = s.queueSynthesis(ctx, q, owner, itemID, key, previewOperation, intent, identity)
		return err
	})
	return id, err
}

func (s Library) GetPreview(ctx context.Context, id pgtype.UUID) (Candidate, pgtype.Timestamptz, error) {
	row, err := queries.New(s.Pool).FindPreview(ctx, id)
	if err != nil {
		return Candidate{}, pgtype.Timestamptz{}, missing(err, ErrPreviewNotFound)
	}
	candidate := Candidate{ID: id, Status: row.Status, SourceFingerprint: row.SourceFingerprint, ErrorCode: row.ErrorCode}
	if row.Status == "READY" {
		if s.audioPresent(row.AssetID, row.StorageKey.String, row.ByteSize.Int64) {
			candidate.AssetID = row.AssetID
		} else {
			candidate.Status = "FAILED"
			candidate.ErrorCode = pgtype.Text{String: "AUDIO_UNAVAILABLE", Valid: true}
		}
	}
	return candidate, row.ExpiresAt, nil
}

func (s Library) OpenPreviewAudio(ctx context.Context, id pgtype.UUID) (*os.File, int64, error) {
	row, err := queries.New(s.Pool).FindPreview(ctx, id)
	if err != nil {
		return nil, 0, missing(err, ErrPreviewNotFound)
	}
	if row.Status != "READY" {
		return nil, 0, ErrAudioNotReady
	}
	if !row.AssetID.Valid {
		return nil, 0, media.ErrAudioUnavailable
	}
	file, err := media.OpenAudio(s.MediaRoot, row.StorageKey.String, row.ByteSize.Int64)
	return file, row.ByteSize.Int64, err
}

// Attach a ready preview to the just-saved synthesis snapshot within the same
// transaction as the item and its topics. A competing claim or expiry is a miss.
func (s Library) claimPreview(ctx context.Context, q *queries.Queries, item queries.LibraryItem, id pgtype.UUID) error {
	if item.PublishedAt.Valid {
		return ErrInvalidDraft
	}
	row, err := q.FindPreview(ctx, id)
	if err != nil {
		return missing(err, ErrPreviewNotFound)
	}
	if row.Status != "READY" {
		return ErrAudioNotReady
	}
	if !s.matchesDraft(item, row.SourceFingerprint) {
		return ErrPreviewMismatch
	}
	if !s.audioPresent(row.AssetID, row.StorageKey.String, row.ByteSize.Int64) {
		return media.ErrAudioUnavailable
	}
	count, err := q.AttachPreview(ctx, queries.AttachPreviewParams{ItemID: item.ID, ID: id})
	if count == 0 && err == nil {
		return ErrPreviewNotFound
	}
	return err
}
