package httpapi

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgtype"
	"taotale/backend/internal/persistence"
)

func registerPublication(router *gin.Engine, store *persistence.Library, providerID string) {
	router.POST("/api/v1/items/:itemId/publish", func(c *gin.Context) {
		id, ok := resourceID(c, "itemId", "ITEM_NOT_FOUND")
		if !ok {
			return
		}
		var input struct {
			ExpectedVersion int32         `json:"expectedVersion"`
			AssetID         field[string] `json:"audioAssetId"`
		}
		if !DecodeJSON(c, &input) {
			return
		}
		var pin pgtype.UUID
		if input.AssetID.Set {
			pin, ok = parseID(defaultValue(input.AssetID.Value))
		}
		if !ok || input.ExpectedVersion < 1 {
			fail(c, 400, "VALIDATION_ERROR", "Send a positive expected version and a valid recording ID when pinning audio.")
			return
		}
		if !hasLibrary(c, store) {
			return
		}
		if err := store.PublishDraft(c.Request.Context(), id, input.ExpectedVersion, pin); err != nil {
			failLibrary(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"itemId": id})
	})
	router.POST("/api/v1/items/publish", func(c *gin.Context) {
		key, ok := generationKey(c)
		if !ok {
			return
		}
		input, ok := decodeNewItem(c, providerID)
		if !ok {
			return
		}
		if !input.PreviewID.Valid {
			fail(c, 400, "VALIDATION_ERROR", "Send the matching preview generation ID.")
			return
		}
		if !hasLibrary(c, store) {
			return
		}
		id, err := store.PublishPreview(c.Request.Context(), key, input)
		if err != nil {
			failLibrary(c, err)
			return
		}
		c.JSON(http.StatusCreated, gin.H{"itemId": id})
	})
}
