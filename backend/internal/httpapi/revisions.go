package httpapi

import (
	"math"
	"net/http"
	"net/url"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgtype"
	"taotale/backend/internal/persistence"
)

func registerRevisions(router *gin.Engine, store *persistence.Library, providerID string) {
	revisions := router.Group("/api/v1/items/:itemId/revisions")
	revisions.POST("", func(c *gin.Context) {
		item, ok := resourceID(c, "itemId", "ITEM_NOT_FOUND")
		if !ok || (c.Request.ContentLength != 0 && !DecodeJSON(c, &struct{}{})) || !hasLibrary(c, store) {
			return
		}
		rev, created, err := store.StartRevision(c.Request.Context(), item)
		status := http.StatusOK
		if created {
			status = http.StatusCreated
		}
		writeRevision(c, rev, err, status)
	})
	scoped := func(handle func(*gin.Context, pgtype.UUID, pgtype.UUID)) gin.HandlerFunc {
		return func(c *gin.Context) {
			item, ok := resourceID(c, "itemId", "ITEM_NOT_FOUND")
			if !ok {
				return
			}
			rev, ok := resourceID(c, "revisionId", "REVISION_NOT_FOUND")
			if ok && hasLibrary(c, store) {
				handle(c, item, rev)
			}
		}
	}
	revision := revisions.Group("/:revisionId")
	revision.GET("", scoped(func(c *gin.Context, item, rev pgtype.UUID) {
		view, err := store.GetRevision(c.Request.Context(), item, rev)
		writeRevision(c, view, err, http.StatusOK)
	}))
	revision.PATCH("", scoped(func(c *gin.Context, item, rev pgtype.UUID) {
		var input struct {
			ExpectedVersion int32          `json:"expectedVersion"`
			SourceText      field[string]  `json:"sourceText"`
			VoiceID         field[string]  `json:"voiceId"`
			SpeakingRate    field[float64] `json:"speakingRate"`
		}
		if !DecodeJSON(c, &input) || !positiveVersion(c, input.ExpectedVersion) {
			return
		}
		patch, message := (itemInput{SourceText: input.SourceText, VoiceID: input.VoiceID, SpeakingRate: input.SpeakingRate}).validate(providerID)
		if message != "" {
			fail(c, 400, "VALIDATION_ERROR", message)
			return
		}
		patch.ExpectedVersion = input.ExpectedVersion
		view, err := store.PatchRevision(c.Request.Context(), item, rev, patch)
		writeRevision(c, view, err, http.StatusOK)
	}))
	revision.DELETE("", scoped(func(c *gin.Context, item, rev pgtype.UUID) {
		query, err := url.ParseQuery(c.Request.URL.RawQuery)
		version, ok := pageValue(query["expectedVersion"], 0, math.MaxInt32)
		if err != nil || !ok || version < 1 {
			fail(c, 400, "VALIDATION_ERROR", "Send one positive expectedVersion query parameter.")
			return
		}
		writeDeletion(c, store.DiscardRevision(c.Request.Context(), item, rev, version))
	}))
	revision.POST("/audio/generate", scoped(func(c *gin.Context, item, rev pgtype.UUID) {
		key, ok := generationKey(c)
		var input struct {
			ExpectedVersion int32 `json:"expectedVersion"`
		}
		if !ok || !DecodeJSON(c, &input) || !positiveVersion(c, input.ExpectedVersion) {
			return
		}
		id, err := store.AdmitRevisionGeneration(c.Request.Context(), item, rev, key, input.ExpectedVersion)
		if err != nil {
			failGeneration(c, err)
			return
		}
		c.JSON(http.StatusAccepted, gin.H{"generationId": id})
	}))
	audio := func(download bool) gin.HandlerFunc {
		return scoped(func(c *gin.Context, item, rev pgtype.UUID) {
			pin, ok := audioAssetPin(c)
			if !ok {
				return
			}
			f, size, err := store.OpenRevisionAudio(c.Request.Context(), item, rev, pin)
			if err != nil {
				failLibrary(c, err)
				return
			}
			defer f.Close()
			serveAudio(c, f, size, "private, no-cache", "taotale-revision-"+rev.String()+".mp3", download)
		})
	}
	revision.GET("/audio", audio(false))
	revision.GET("/audio/download", audio(true))
	revision.POST("/publish", scoped(func(c *gin.Context, item, rev pgtype.UUID) {
		var input struct {
			ExpectedVersion     int32         `json:"expectedVersion"`
			ExpectedItemVersion int32         `json:"expectedItemVersion"`
			AssetID             field[string] `json:"audioAssetId"`
		}
		if !DecodeJSON(c, &input) || !positiveVersion(c, input.ExpectedVersion) {
			return
		}
		if input.ExpectedItemVersion < 1 {
			fail(c, 400, "VALIDATION_ERROR", "Send a positive expectedItemVersion.")
			return
		}
		var pin pgtype.UUID
		if input.AssetID.Set {
			var ok bool
			pin, ok = parseID(defaultValue(input.AssetID.Value))
			if !ok {
				fail(c, 400, "VALIDATION_ERROR", "Send a valid recording ID when pinning audio.")
				return
			}
		}
		if err := store.PublishRevision(c.Request.Context(), item, rev, input.ExpectedVersion, input.ExpectedItemVersion, pin); err != nil {
			failLibrary(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"itemId": item})
	}))
}

func positiveVersion(c *gin.Context, version int32) bool {
	if version < 1 {
		fail(c, 400, "VALIDATION_ERROR", "Send a positive expectedVersion.")
		return false
	}
	return true
}

func writeRevision(c *gin.Context, rev persistence.Revision, err error, status int) {
	if err != nil {
		failLibrary(c, err)
		return
	}
	c.JSON(status, gin.H{
		"id": rev.ID, "itemId": rev.ItemID, "itemVersion": rev.ItemVersion, "version": rev.Version,
		"baseAudioAssetId": rev.BaseAudioAssetID, "languageCode": rev.LanguageCode, "sourceText": rev.SourceText,
		"voiceId": rev.VoiceID, "speakingRate": rev.SpeakingRate, "createdAt": rev.CreatedAt.Time.UTC(), "updatedAt": rev.UpdatedAt.Time.UTC(),
		"audioStatus": rev.Audio.Status, "audioGenerationId": rev.Audio.ID, "audioAssetId": rev.Audio.AssetID, "errorCode": rev.Audio.ErrorCode,
	})
}
