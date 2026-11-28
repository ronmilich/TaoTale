package persistence

import (
	"context"
	"errors"
	"log/slog"
	"os"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"taotale/backend/internal/media"
	"taotale/backend/internal/persistence/queries"
	"taotale/backend/internal/tts"
)

var (
	ErrRevisionNotFound = errors.New("working revision not found")
	ErrUnpublished      = errors.New("a working revision needs a published recording")
	ErrRevisionBase     = errors.New("published recording differs from revision base")
)

type Revision struct {
	queries.WorkingRevision
	LanguageCode string
	ItemVersion  int32
	Audio        Candidate
}

// Evaluate just the candidate's speech inputs using the same synthesis identity
// rules as drafts. The language and owner always come from the parent item.
func revisionInputs(item queries.LibraryItem, rev queries.WorkingRevision) queries.LibraryItem {
	item.SourceText, item.VoiceID, item.SpeakingRate = rev.SourceText, rev.VoiceID, rev.SpeakingRate
	item.Version = rev.Version
	return item
}

func (s Library) revisionView(ctx context.Context, q *queries.Queries, item queries.LibraryItem, rev queries.WorkingRevision) (Revision, error) {
	view := Revision{WorkingRevision: rev, LanguageCode: item.LanguageCode, ItemVersion: item.Version, Audio: Candidate{Status: "NOT_GENERATED"}}
	row, err := q.LatestRevisionCandidate(ctx, rev.ID)
	if errors.Is(err, pgx.ErrNoRows) {
		return view, nil
	}
	if err == nil {
		view.Audio = s.projectCandidate(revisionInputs(item, rev), queries.LatestDraftCandidatesRow(row))
	}
	return view, err
}

func (s Library) StartRevision(ctx context.Context, itemID pgtype.UUID) (Revision, bool, error) {
	var view Revision
	var created bool
	err := pgx.BeginFunc(ctx, s.Pool, func(tx pgx.Tx) error {
		q := queries.New(tx)
		item, err := q.LockItem(ctx, itemID)
		if err != nil {
			return missing(err, ErrItemNotFound)
		}
		if !item.PublishedAt.Valid || !item.ActiveAudioAssetID.Valid {
			return ErrUnpublished
		}
		rev, err := q.GetItemRevision(ctx, queries.GetItemRevisionParams{ItemID: itemID, UserID: item.UserID})
		if errors.Is(err, pgx.ErrNoRows) {
			created = true
			rev, err = q.CreateRevision(ctx, itemID)
		}
		if err != nil {
			return err
		}
		view, err = s.revisionView(ctx, q, item, rev)
		return err
	})
	return view, created, err
}

func readRevision(ctx context.Context, q *queries.Queries, itemID, revisionID pgtype.UUID) (queries.LibraryItem, queries.WorkingRevision, error) {
	item, err := q.GetItem(ctx, itemID)
	if err != nil {
		return item, queries.WorkingRevision{}, missing(err, ErrItemNotFound)
	}
	rev, err := q.GetItemRevision(ctx, queries.GetItemRevisionParams{ItemID: itemID, UserID: item.UserID})
	if err == nil && rev.ID != revisionID {
		err = ErrRevisionNotFound
	}
	return item, rev, missing(err, ErrRevisionNotFound)
}

func (s Library) GetRevision(ctx context.Context, itemID, revisionID pgtype.UUID) (Revision, error) {
	var view Revision
	err := pgx.BeginTxFunc(ctx, s.Pool, snapshot, func(tx pgx.Tx) error {
		q := queries.New(tx)
		item, rev, err := readRevision(ctx, q, itemID, revisionID)
		if err != nil {
			return err
		}
		view, err = s.revisionView(ctx, q, item, rev)
		return err
	})
	return view, err
}

func lockRevision(ctx context.Context, q *queries.Queries, item queries.LibraryItem, revisionID pgtype.UUID) (queries.WorkingRevision, error) {
	rev, err := q.LockItemRevision(ctx, queries.LockItemRevisionParams{ItemID: item.ID, UserID: item.UserID})
	if err == nil && rev.ID != revisionID {
		err = ErrRevisionNotFound
	}
	return rev, missing(err, ErrRevisionNotFound)
}

// All mutations lock the parent before the candidate, including admission.
func (s Library) changeRevision(ctx context.Context, itemID, revisionID pgtype.UUID, version int32, change func(*queries.Queries, queries.LibraryItem, queries.WorkingRevision) error) error {
	return pgx.BeginFunc(ctx, s.Pool, func(tx pgx.Tx) error {
		q := queries.New(tx)
		item, err := q.LockItem(ctx, itemID)
		if err != nil {
			return missing(err, ErrItemNotFound)
		}
		rev, err := lockRevision(ctx, q, item, revisionID)
		if err != nil {
			return err
		}
		if rev.Version != version {
			return &VersionConflict{rev.Version}
		}
		return change(q, item, rev)
	})
}

func (s Library) PatchRevision(ctx context.Context, itemID, revisionID pgtype.UUID, patch ItemPatch) (Revision, error) {
	var view Revision
	err := s.changeRevision(ctx, itemID, revisionID, patch.ExpectedVersion, func(q *queries.Queries, item queries.LibraryItem, rev queries.WorkingRevision) error {
		if patch.Voice != nil && patch.Voice.Valid && !tts.AllowedVoice(item.LanguageCode, s.Voices.ProviderID(), patch.Voice.String) {
			return ErrInvalidVoice
		}
		saved, err := q.UpdateRevision(ctx, queries.UpdateRevisionParams{ID: rev.ID, Version: rev.Version,
			SourceText: valueOr(patch.SourceText, rev.SourceText), VoiceID: valueOr(patch.Voice, rev.VoiceID), SpeakingRate: valueOr(patch.Rate, rev.SpeakingRate)})
		if err != nil {
			return err
		}
		view, err = s.revisionView(ctx, q, item, saved)
		return err
	})
	return view, err
}

func (s Library) OpenRevisionAudio(ctx context.Context, itemID, revisionID, pin pgtype.UUID) (*os.File, int64, error) {
	var row queries.LatestRevisionCandidateRow
	err := pgx.BeginTxFunc(ctx, s.Pool, snapshot, func(tx pgx.Tx) error {
		q := queries.New(tx)
		item, rev, err := readRevision(ctx, q, itemID, revisionID)
		if err != nil {
			return err
		}
		row, err = q.LatestRevisionCandidate(ctx, rev.ID)
		if err != nil {
			return missing(err, ErrAudioNotFound)
		}
		if row.Status != "READY" || row.AssetID != pin || !s.matchesDraft(revisionInputs(item, rev), row.SourceFingerprint) {
			return ErrAudioNotFound
		}
		return nil
	})
	if err != nil {
		return nil, 0, err
	}
	f, err := media.OpenAudio(s.MediaRoot, row.StorageKey.String, row.ByteSize.Int64)
	return f, row.ByteSize.Int64, err
}

func (s Library) DiscardRevision(ctx context.Context, itemID, revisionID pgtype.UUID, version int32) error {
	return s.finishRevision(ctx, itemID, revisionID, version, nil, pgtype.UUID{})
}

func (s Library) PublishRevision(ctx context.Context, itemID, revisionID pgtype.UUID, version, itemVersion int32, pin pgtype.UUID) error {
	return s.finishRevision(ctx, itemID, revisionID, version, &itemVersion, pin)
}

func (s Library) finishRevision(ctx context.Context, itemID, revisionID pgtype.UUID, version int32, publishAt *int32, pin pgtype.UUID) error {
	var keys []string
	err := s.changeRevision(ctx, itemID, revisionID, version, func(q *queries.Queries, item queries.LibraryItem, rev queries.WorkingRevision) error {
		if publishAt != nil {
			if item.Version != *publishAt {
				return &VersionConflict{item.Version}
			}
			if item.ActiveAudioAssetID != rev.BaseAudioAssetID {
				return ErrRevisionBase
			}
			audio, err := s.publicationAudio(ctx, q, revisionInputs(item, rev), rev.ID, pin)
			if err != nil {
				return err
			}
			count, err := q.PublishRevision(ctx, queries.PublishRevisionParams{ID: item.ID, ExpectedVersion: item.Version,
				BaseAssetID: rev.BaseAudioAssetID, AssetID: audio.ID, SourceText: rev.SourceText,
				VoiceID: pgtype.Text{String: audio.VoiceID, Valid: true}, SpeakingRate: pgtype.Float8{Float64: audio.SpeakingRate, Valid: true}})
			if err != nil {
				return err
			}
			if count != 1 {
				return ErrRevisionBase
			}
			if _, err := reconcileVocabulary(ctx, q, revisionInputs(item, rev)); err != nil {
				return err
			}
		}
		ids, err := q.DeleteRevisionGenerations(ctx, rev.ID)
		if err != nil {
			return err
		}
		if err := q.DeleteRevision(ctx, rev.ID); err != nil {
			return err
		}
		keys, err = deleteUnusedAudio(ctx, q, item.UserID, append(ids, rev.BaseAudioAssetID))
		return err
	})
	if err != nil {
		return err
	}
	if err := s.releaseItemMedia(keys, ""); err != nil {
		slog.WarnContext(ctx, "Could not release revision media", "revision_id", revisionID.String())
	}
	return nil
}
