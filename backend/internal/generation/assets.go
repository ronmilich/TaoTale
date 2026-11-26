package generation

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"taotale/backend/internal/alignment"
	"taotale/backend/internal/fingerprint"
	"taotale/backend/internal/persistence/queries"
	"taotale/backend/internal/tts"
)

func (w *Worker) store(ctx context.Context, job queries.AudioGeneration, id fingerprint.Identity, audio tts.Result) string {
	if len(audio.MP3) == 0 || len(audio.MP3) > alignment.MaxMP3Bytes {
		return "ALIGNMENT_UNRELIABLE"
	}
	stage := "staging/tts-" + job.ID.String() + ".part"
	f, err := w.root.OpenFile(stage, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return storageError
	}
	defer w.root.Remove(stage)
	_, writeErr := f.Write(audio.MP3)
	if err := errors.Join(writeErr, f.Chmod(0400), f.Sync(), f.Close()); err != nil {
		return storageError
	}
	// Validate the persisted bytes, including decoded duration, before making
	// them available under an immutable asset key.
	audio.MP3, err = w.root.ReadFile(stage)
	if err != nil {
		return storageError
	}
	verified, err := alignment.Validate(ctx, id, job.SourceFingerprint, audio)
	if err != nil {
		return "ALIGNMENT_UNRELIABLE"
	}
	segments, err := json.Marshal(verified.Segments)
	if err != nil {
		return storageError
	}
	err = pgx.BeginFunc(ctx, w.pool, func(tx pgx.Tx) error {
		q := queries.New(tx)
		assetID, err := q.LockGenerationForAsset(ctx, job.ID)
		if err != nil {
			return err
		}
		key := "audio/" + assetID.String() + ".mp3"
		// A hard link atomically promotes without replacing an existing asset.
		if err := w.promote(stage, key); err != nil {
			return err
		}
		if err := q.InsertVerifiedAsset(ctx, queries.InsertVerifiedAssetParams{
			AssetID: assetID, GenerationID: job.ID, StorageKey: key, ByteSize: int64(len(audio.MP3)),
			DurationMs: verified.DurationMS, Segments: segments,
		}); err != nil {
			return err
		}
		return q.FinishGeneration(ctx, queries.FinishGenerationParams{ID: job.ID, AssetID: assetID})
	})
	// A lost commit response can mean the asset is already visible. Recovery
	// consults the committed references before removing an orphan.
	if err != nil {
		return storageError
	}
	return ""
}

func (w *Worker) promote(stage, key string) error {
	if err := w.root.Link(stage, key); err != nil {
		return err
	}
	dir, err := w.root.Open("audio")
	if err != nil {
		return err
	}
	return errors.Join(dir.Sync(), dir.Close())
}

func (w *Worker) cleanOrphans(ctx context.Context) error {
	for _, dir := range []string{"staging", "audio"} {
		if err := w.cleanDirectory(ctx, dir); err != nil {
			return err
		}
	}
	return nil
}

func (w *Worker) cleanDirectory(ctx context.Context, dir string) error {
	f, err := w.root.Open(dir)
	if err != nil {
		return err
	}
	defer f.Close()
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		entries, err := f.ReadDir(128)
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return err
		}
		var keys []string
		for _, entry := range entries {
			if entry.Type().IsRegular() && workerFilename(dir, entry.Name()) {
				keys = append(keys, dir+"/"+entry.Name())
			}
		}
		if len(keys) == 0 {
			continue
		}
		referenced := map[string]bool{}
		if dir == "audio" {
			saved, err := queries.New(w.leader).ReferencedAudioFiles(ctx, keys)
			if err != nil {
				return err
			}
			for _, key := range saved {
				referenced[key] = true
			}
		}
		for _, key := range keys {
			if !referenced[key] {
				if err := w.root.Remove(key); err != nil && !errors.Is(err, os.ErrNotExist) {
					return err
				}
			}
		}
	}
}

func workerFilename(dir, name string) bool {
	var ok bool
	if dir == "staging" {
		name, ok = strings.CutPrefix(name, "tts-")
		if !ok {
			return false
		}
		name, ok = strings.CutSuffix(name, ".part")
	} else {
		name, ok = strings.CutSuffix(name, ".mp3")
	}
	var id pgtype.UUID
	return ok && len(name) == 36 && id.Scan(name) == nil && id.String() == name
}
