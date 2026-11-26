package persistence

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"slices"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"taotale/backend/internal/fingerprint"
	"taotale/backend/internal/persistence/queries"
	"taotale/backend/internal/segmentation"
	"taotale/backend/internal/tts"
)

var (
	ErrKeyReused          = errors.New("idempotency key used for another intent")
	ErrKeyExpired         = errors.New("idempotency receipt expired")
	ErrGenerationActive   = errors.New("draft has an active generation")
	ErrInvalidDraft       = errors.New("generation requires a saved draft with valid synthesis inputs")
	ErrGenerationNotFound = errors.New("generation not found")
)

const DefaultActiveLimit = 2
const DefaultHourlyLimit = 60

type GenerationLimits struct{ Active, Hourly int64 }

type Candidate struct {
	ID, AssetID                          pgtype.UUID
	SourceFingerprint, Status, JobStatus string
	ErrorCode                            pgtype.Text
}

func (s Library) identity(item queries.LibraryItem) (fingerprint.Identity, error) {
	strategy, err := segmentation.Default(item.LanguageCode)
	if err != nil {
		return fingerprint.Identity{}, err
	}
	provider := s.Voices.ProviderID()
	voice := item.VoiceID.String
	if !item.VoiceID.Valid {
		voice, err = tts.DefaultVoice(item.LanguageCode, provider)
		if err != nil {
			return fingerprint.Identity{}, err
		}
	}
	rate := 1.0
	if item.SpeakingRate.Valid {
		rate = item.SpeakingRate.Float64
	}
	return fingerprint.Identity{LanguageCode: item.LanguageCode, ProviderID: provider, VoiceID: voice,
		SpeakingRate: rate, StrategyID: strategy.StrategyID, StrategyVersion: strategy.Version, ExactSourceText: item.SourceText}, nil
}

func (s Library) AdmitGeneration(ctx context.Context, itemID, key pgtype.UUID) (pgtype.UUID, error) {
	var id pgtype.UUID
	err := pgx.BeginFunc(ctx, s.Pool, func(tx pgx.Tx) error {
		q := queries.New(tx)
		// All admission writers acquire the owner before the draft. READ COMMITTED
		// gives each check a fresh snapshot after waiting for the preceding writer.
		owner, err := q.LockOwner(ctx)
		if err != nil {
			return err
		}
		// No client synthesis overrides: the canonical request is the operation
		// and target UUID. The mutable draft fingerprint is a separate identity.
		digest := sha256.Sum256([]byte("generate_draft_audio/" + itemID.String()))
		intent := hex.EncodeToString(digest[:])
		receipt, err := q.GetGenerationReceipt(ctx, queries.GetGenerationReceiptParams{UserID: owner, Key: key})
		switch {
		case err == nil:
			if receipt.RequestHash != intent {
				return ErrKeyReused
			}
			if !receipt.Replayable {
				return ErrKeyExpired
			}
			id = receipt.GenerationID
			return nil
		case !errors.Is(err, pgx.ErrNoRows):
			return err
		}
		item, err := q.LockItem(ctx, itemID)
		if err != nil {
			return missing(err, ErrItemNotFound)
		}
		if item.PublishedAt.Valid {
			return ErrInvalidDraft
		}
		if err := s.checkCapacity(ctx, q, owner, itemID); err != nil {
			return err
		}
		identity, err := s.identity(item)
		if err != nil {
			return ErrInvalidDraft
		}
		if err := tts.ValidateRequest(identity.ProviderID, tts.Request{Language: identity.LanguageCode,
			VoiceID: identity.VoiceID, Source: identity.ExactSourceText, Rate: identity.SpeakingRate}); err != nil {
			return ErrInvalidDraft
		}
		// Voice discovery only inspects the local configuration and worker binary.
		voices, err := s.Voices.ListVoices(ctx, item.LanguageCode)
		if err != nil || !slices.ContainsFunc(voices, func(v tts.Voice) bool {
			return v.ID == identity.VoiceID && slices.Contains(v.Rates, identity.SpeakingRate)
		}) {
			return &tts.Error{Code: tts.Unavailable}
		}
		sum, err := identity.Sum()
		if err != nil {
			return err
		}
		id, err = q.ReserveGenerationKey(ctx, queries.ReserveGenerationKeyParams{UserID: owner, Key: key, RequestHash: intent})
		if err != nil {
			return err
		}
		return q.InsertGeneration(ctx, queries.InsertGenerationParams{
			ID: id, UserID: owner, ItemID: itemID, IdempotencyKey: key,
			LanguageCode: identity.LanguageCode, ProviderID: identity.ProviderID, VoiceID: identity.VoiceID,
			SpeakingRate: identity.SpeakingRate, StrategyID: identity.StrategyID, StrategyVersion: identity.StrategyVersion,
			FingerprintFormatVersion: fingerprint.SchemaVersion, SourceText: identity.ExactSourceText, SourceFingerprint: sum,
		})
	})
	return id, err
}

func (s Library) checkCapacity(ctx context.Context, q *queries.Queries, owner, itemID pgtype.UUID) error {
	usage, err := q.GenerationUsage(ctx, queries.GenerationUsageParams{UserID: owner, ItemID: itemID})
	if err != nil {
		return err
	}
	if usage.TargetActive {
		return ErrGenerationActive
	}
	active, hourly := s.Limits.Active, s.Limits.Hourly
	if active == 0 {
		active = DefaultActiveLimit
	}
	if hourly == 0 {
		hourly = DefaultHourlyLimit
	}
	if usage.ActiveCount >= active || usage.HourlyCount >= hourly {
		return &tts.Error{Code: tts.RateLimited}
	}
	return nil
}

// GetGeneration reads the requested job and latest candidate in one snapshot.
func (s Library) GetGeneration(ctx context.Context, itemID, id pgtype.UUID) (Candidate, error) {
	var result Candidate
	err := pgx.BeginTxFunc(ctx, s.Pool, snapshot, func(tx pgx.Tx) error {
		q := queries.New(tx)
		item, err := q.GetItem(ctx, itemID)
		if err != nil {
			return missing(err, ErrGenerationNotFound)
		}
		latest, err := q.LatestDraftCandidates(ctx, []pgtype.UUID{itemID})
		if err != nil {
			return err
		}
		if len(latest) != 0 && latest[0].ID == id {
			result = s.projectCandidate(item, latest[0])
			return nil
		}
		job, err := q.FindDraftGeneration(ctx, queries.FindDraftGenerationParams{ItemID: itemID, ID: id})
		if err != nil {
			return missing(err, ErrGenerationNotFound)
		}
		result = Candidate{ID: id, SourceFingerprint: job.SourceFingerprint, JobStatus: job.Status, ErrorCode: job.ErrorCode, Status: "OUTDATED"}
		return nil
	})
	return result, err
}

func (s Library) projectCandidate(item queries.LibraryItem, row queries.LatestDraftCandidatesRow) Candidate {
	result := Candidate{ID: row.ID, SourceFingerprint: row.SourceFingerprint, JobStatus: row.Status, ErrorCode: row.ErrorCode, Status: "OUTDATED"}
	id, err := s.identity(item)
	if err != nil {
		return result
	}
	sum, err := id.Sum()
	if err != nil || sum != row.SourceFingerprint {
		return result
	}
	result.Status = row.Status
	if row.Status == "READY" {
		if row.AssetID.Valid && s.assetPresent(row.StorageKey.String, row.ByteSize.Int64) {
			result.AssetID = row.AssetID
		} else {
			result.Status = "FAILED"
			result.ErrorCode = pgtype.Text{String: "AUDIO_UNAVAILABLE", Valid: true}
		}
	}
	return result
}

func (s Library) assetPresent(key string, size int64) bool {
	if !filepath.IsLocal(key) || size <= 0 {
		return false
	}
	root, err := os.OpenRoot(s.MediaRoot)
	if err != nil {
		return false
	}
	defer root.Close()
	info, err := root.Lstat(key)
	return err == nil && info.Mode().IsRegular() && info.Size() == size
}
