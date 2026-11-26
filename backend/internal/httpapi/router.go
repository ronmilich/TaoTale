package httpapi

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"taotale/backend/internal/persistence"
	"taotale/backend/internal/tts"
)

const RequestTimeout = 10 * time.Second

func Router(ready func(context.Context) error, logger *slog.Logger, devOrigin string, catalog Catalog, spa *SPA, provider tts.VoiceCatalog, library *persistence.Library) *gin.Engine {
	router := gin.New()
	router.HandleMethodNotAllowed = true
	router.RedirectTrailingSlash = false
	router.Use(requestMetadata(logger), recoverRequest, localRequests(devOrigin), requestDeadline(RequestTimeout), limitJSON)
	router.NoRoute(func(c *gin.Context) {
		if spa != nil && spa.serve(c) {
			return
		}
		fail(c, http.StatusNotFound, "NOT_FOUND", "The route does not exist.")
	})
	router.NoMethod(func(c *gin.Context) {
		fail(c, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "This method is not supported.")
	})
	for _, prefix := range []string{"", "/api/v1"} {
		group := router.Group(prefix + "/health")
		group.GET("/live", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"status": "ok"})
		})
		group.GET("/ready", func(c *gin.Context) {
			ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
			defer cancel()
			if err := ready(ctx); err != nil {
				fail(c, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "Backend storage unavailable.")
				return
			}
			c.JSON(http.StatusOK, gin.H{"status": "ok"})
		})
	}
	router.GET("/api/v1/languages", listLanguages(catalog))
	router.GET("/api/v1/topics", listTopics(catalog))
	router.GET("/api/v1/tts/voices", listVoices(provider))
	if library != nil {
		store := *library
		store.Voices = provider
		library = &store
	}
	registerLibrary(router, library, provider.ProviderID())
	registerGenerations(router, library)
	registerPreviews(router, library)
	router.GET("/api/v1/items/:itemId/audio", itemAudio(library, false))
	router.GET("/api/v1/items/:itemId/audio/download", itemAudio(library, true))
	registerTopicWrites(router, library)
	return router
}

type apiError struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	RequestID string `json:"requestId"`
	Details   gin.H  `json:"details,omitempty"`
}

func fail(c *gin.Context, status int, code, message string) {
	failDetails(c, status, code, message, nil)
}

func failDetails(c *gin.Context, status int, code, message string, details gin.H) {
	c.AbortWithStatusJSON(status, gin.H{"error": apiError{code, message, c.Writer.Header().Get("X-Request-Id"), details}})
}
