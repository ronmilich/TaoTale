package httpapi

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgtype"
	"taotale/backend/internal/persistence"
	"taotale/backend/internal/tts"
)

func registerGenerations(router *gin.Engine, store *persistence.Library) {
	audio := router.Group("/api/v1/items/:itemId/audio")
	audio.POST("/generate", func(c *gin.Context) {
		item, ok := resourceID(c, "itemId", "ITEM_NOT_FOUND")
		if !ok {
			return
		}
		key, ok := generationKey(c)
		if !ok {
			return
		}
		var input struct {
			ExpectedVersion field[int32] `json:"expectedVersion"`
		}
		if c.Request.ContentLength != 0 && !DecodeJSON(c, &input) {
			return
		}
		if input.ExpectedVersion.Set && (input.ExpectedVersion.Value == nil || *input.ExpectedVersion.Value < 1) {
			fail(c, 400, "VALIDATION_ERROR", "Send a positive expected version.")
			return
		}
		if !hasLibrary(c, store) {
			return
		}
		id, err := store.AdmitGeneration(c.Request.Context(), item, key, input.ExpectedVersion.Value)
		if err != nil {
			failGeneration(c, err)
			return
		}
		c.JSON(http.StatusAccepted, gin.H{"generationId": id})
	})
	audio.GET("/generations/:generationId", func(c *gin.Context) {
		item, ok := resourceID(c, "itemId", "GENERATION_NOT_FOUND")
		if !ok {
			return
		}
		id, ok := resourceID(c, "generationId", "GENERATION_NOT_FOUND")
		if !ok || !hasLibrary(c, store) {
			return
		}
		job, err := store.GetGeneration(c.Request.Context(), item, id)
		if err != nil {
			failGeneration(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"generationId": job.ID, "itemId": item, "status": job.Status, "jobStatus": job.JobStatus,
			"sourceFingerprint": job.SourceFingerprint, "audioAssetId": job.AssetID, "errorCode": job.ErrorCode})
	})
}

func generationKey(c *gin.Context) (pgtype.UUID, bool) {
	key, ok := parseID(c.GetHeader("Idempotency-Key"))
	if !ok || len(c.Request.Header.Values("Idempotency-Key")) != 1 {
		fail(c, 400, "VALIDATION_ERROR", "Send one UUID Idempotency-Key.")
		return pgtype.UUID{}, false
	}
	return key, true
}

func failGeneration(c *gin.Context, err error) {
	switch {
	case errors.Is(err, persistence.ErrKeyReused):
		fail(c, 409, "IDEMPOTENCY_KEY_REUSED", "This key was used for a different operation target.")
	case errors.Is(err, persistence.ErrKeyExpired):
		fail(c, 409, "IDEMPOTENCY_KEY_EXPIRED", "The original generation receipt has expired.")
	case errors.Is(err, persistence.ErrGenerationActive):
		fail(c, 409, "GENERATION_IN_PROGRESS", "This draft already has an active generation.")
	case errors.Is(err, persistence.ErrInvalidDraft):
		fail(c, 409, "INVALID_ITEM_STATE", "Save a draft with valid speech text, voice and rate before generating.")
	case errors.Is(err, persistence.ErrGenerationNotFound):
		fail(c, 404, "GENERATION_NOT_FOUND", "Resource not found.")
	case tts.CodeOf(err) == tts.RateLimited:
		fail(c, 429, string(tts.RateLimited), "The speech generation limit was reached.")
	case tts.CodeOf(err) == tts.Unavailable:
		fail(c, 503, string(tts.Unavailable), "Speech synthesis is unavailable.")
	case tts.CodeOf(err) == tts.Invalid:
		fail(c, 400, string(tts.Invalid), "Send valid speech text, language, voice and rate.")
	default:
		failLibrary(c, err)
	}
}
