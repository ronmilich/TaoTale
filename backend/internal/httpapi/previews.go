package httpapi

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"taotale/backend/internal/persistence"
	"taotale/backend/internal/tts"
)

func registerPreviews(router *gin.Engine, store *persistence.Library) {
	previews := router.Group("/api/v1/tts/previews")
	previews.POST("", func(c *gin.Context) {
		key, ok := generationKey(c)
		if !ok {
			return
		}
		var input struct {
			Language string  `json:"languageCode"`
			Source   string  `json:"sourceText"`
			Voice    string  `json:"voiceId"`
			Rate     float64 `json:"speakingRate"`
		}
		if !DecodeJSON(c, &input) {
			return
		}
		if !hasLibrary(c, store) {
			return
		}
		request := tts.Request{Language: input.Language, Source: input.Source, VoiceID: input.Voice, Rate: input.Rate}
		id, err := store.AdmitPreview(c.Request.Context(), key, request)
		if err != nil {
			failGeneration(c, err)
			return
		}
		c.JSON(http.StatusAccepted, gin.H{"generationId": id})
	})
	previews.GET("/:generationId", func(c *gin.Context) {
		id, ok := resourceID(c, "generationId", "PREVIEW_NOT_FOUND")
		if !ok || !hasLibrary(c, store) {
			return
		}
		preview, expires, err := store.GetPreview(c.Request.Context(), id)
		if err != nil {
			failLibrary(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"generationId": preview.ID, "status": preview.Status,
			"expiresAt": expires.Time.UTC(), "sourceFingerprint": preview.SourceFingerprint,
			"audioAssetId": preview.AssetID, "errorCode": preview.ErrorCode})
	})
	previews.GET("/:generationId/audio", previewAudio(store, false))
	previews.GET("/:generationId/audio/download", previewAudio(store, true))
}
