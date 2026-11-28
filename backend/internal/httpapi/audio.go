package httpapi

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgtype"
	"taotale/backend/internal/persistence"
)

func audioAssetPin(c *gin.Context) (pgtype.UUID, bool) {
	query, err := url.ParseQuery(c.Request.URL.RawQuery)
	if pins := query["audioAssetId"]; err == nil && len(pins) == 1 {
		if pin, ok := parseID(pins[0]); ok {
			return pin, true
		}
	}
	fail(c, 400, "VALIDATION_ERROR", "Send one expected audioAssetId.")
	return pgtype.UUID{}, false
}

func itemAudio(store *persistence.Library, download bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, ok := resourceID(c, "itemId", "ITEM_NOT_FOUND")
		if !ok {
			return
		}
		pin, ok := audioAssetPin(c)
		if !ok || !hasLibrary(c, store) {
			return
		}
		file, size, err := store.OpenItemAudio(c.Request.Context(), id, pin)
		if err != nil {
			failLibrary(c, err)
			return
		}
		defer file.Close()
		serveAudio(c, file, size, "private, no-cache", "taotale-"+id.String()+".mp3", download)
	}
}

func previewAudio(store *persistence.Library, download bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, ok := resourceID(c, "generationId", "PREVIEW_NOT_FOUND")
		if !ok || !hasLibrary(c, store) {
			return
		}
		file, size, err := store.OpenPreviewAudio(c.Request.Context(), id)
		if err != nil {
			failLibrary(c, err)
			return
		}
		defer file.Close()
		serveAudio(c, file, size, "private, no-store", "taotale-preview-"+id.String()+".mp3", download)
	}
}

func serveAudio(c *gin.Context, file *os.File, size int64, cache, filename string, download bool) {
	start, end, status := int64(0), size-1, http.StatusOK
	c.Header("Accept-Ranges", "bytes")
	// The route has no validator for If-Range, so conditional ranges get a
	// complete response for the already checked asset pin.
	if ranges := c.Request.Header.Values("Range"); len(ranges) != 0 && c.GetHeader("If-Range") == "" {
		var ok bool
		if len(ranges) == 1 {
			start, end, ok = audioRange(ranges[0], size)
		}
		if !ok {
			c.Header("Content-Range", fmt.Sprintf("bytes */%d", size))
			fail(c, http.StatusRequestedRangeNotSatisfiable, "INVALID_RANGE", "Send one satisfiable byte range.")
			return
		}
		c.Header("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, end, size))
		status = http.StatusPartialContent
	}
	length := end - start + 1
	c.Header("Content-Type", "audio/mpeg")
	c.Header("Content-Length", strconv.FormatInt(length, 10))
	c.Header("Cache-Control", cache)
	c.Header("X-Content-Type-Options", "nosniff")
	if download {
		c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	}
	c.Status(status)
	c.Writer.WriteHeaderNow()
	_, _ = io.Copy(c.Writer, io.NewSectionReader(file, start, length))
}

func audioRange(header string, size int64) (int64, int64, bool) {
	unit, bounds, ok := strings.Cut(strings.TrimSpace(header), "=")
	if !ok || !strings.EqualFold(unit, "bytes") || strings.Contains(bounds, ",") {
		return 0, 0, false
	}
	first, last, ok := strings.Cut(bounds, "-")
	if !ok {
		return 0, 0, false
	}
	limit := uint64(size)
	if first == "" {
		suffix, ok := rangeNumber(last)
		return size - int64(min(suffix, limit)), size - 1, ok && suffix > 0
	}
	start, ok := rangeNumber(first)
	if !ok || start >= limit {
		return 0, 0, false
	}
	if last == "" {
		return int64(start), size - 1, true
	}
	end, ok := rangeNumber(last)
	return int64(start), int64(min(end, limit-1)), ok && end >= start
}

func rangeNumber(raw string) (uint64, bool) {
	if raw == "" {
		return 0, false
	}
	for _, digit := range raw {
		if digit < '0' || digit > '9' {
			return 0, false
		}
	}
	// ParseUint saturates on overflow. Such values are greater than any file
	// size, so they can be clipped for end and suffix positions.
	n, _ := strconv.ParseUint(raw, 10, 64)
	return n, true
}
