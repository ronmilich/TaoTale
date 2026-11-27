package httpapi

import (
	"errors"
	"io"
	"mime"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"taotale/backend/internal/media"
	"taotale/backend/internal/persistence"
)

func registerImages(router *gin.Engine, store *persistence.Library) {
	router.POST("/api/v1/storage/images", func(c *gin.Context) {
		contentType, _, _ := mime.ParseMediaType(c.GetHeader("Content-Type"))
		switch contentType {
		case "image/jpeg", "image/png", "image/webp":
		default:
			fail(c, http.StatusUnsupportedMediaType, "UNSUPPORTED_MEDIA_TYPE", "Send JPEG, PNG or WebP image bytes.")
			return
		}
		if c.Request.ContentLength > media.MaxCoverBytes {
			coverTooLarge(c)
			return
		}
		if !hasLibrary(c, store) {
			return
		}
		record, err := store.CreateCover(c.Request.Context(), c.Request.Body, contentType)
		switch {
		case errors.Is(err, media.ErrCoverTooLarge):
			coverTooLarge(c)
		case errors.Is(err, media.ErrInvalidCover):
			fail(c, http.StatusBadRequest, "INVALID_IMAGE", "Use a decodable JPEG, PNG or WebP image up to 4096 × 4096 pixels.")
		case err != nil:
			failLibrary(c, err)
		default:
			c.JSON(http.StatusCreated, gin.H{
				"id": record.ID, "contentType": record.ContentType, "sizeBytes": record.SizeBytes,
				"width": record.Width, "height": record.Height, "expiresAt": record.ExpiresAt,
			})
		}
	})
	router.GET("/api/v1/storage/images/:imageId", func(c *gin.Context) {
		id, ok := resourceID(c, "imageId", "IMAGE_NOT_FOUND")
		if !ok || !hasLibrary(c, store) {
			return
		}
		record, file, err := store.Cover(c.Request.Context(), id)
		if err != nil {
			failLibrary(c, err)
			return
		}
		defer file.Close()
		etag := strconv.Quote(record.ID.String())
		c.Header("ETag", etag)
		c.Header("Cache-Control", "private, no-cache")
		for _, candidate := range strings.Split(c.GetHeader("If-None-Match"), ",") {
			tag := strings.TrimSpace(candidate)
			if tag == "*" || strings.TrimPrefix(tag, "W/") == etag {
				c.Status(http.StatusNotModified)
				return
			}
		}
		c.Header("Content-Type", record.ContentType)
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("Content-Length", strconv.FormatInt(record.SizeBytes, 10))
		c.Status(http.StatusOK)
		_, _ = io.Copy(c.Writer, file)
	})
}

func coverTooLarge(c *gin.Context) {
	fail(c, http.StatusRequestEntityTooLarge, "IMAGE_TOO_LARGE", "Cover images must be 1,000,000 bytes or smaller.")
}
