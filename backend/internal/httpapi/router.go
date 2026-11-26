package httpapi

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

const RequestTimeout = 10 * time.Second

func Router(ready func(context.Context) error, logger *slog.Logger, devOrigin string, catalog Catalog) *gin.Engine {
	router := gin.New()
	router.HandleMethodNotAllowed = true
	router.RedirectTrailingSlash = false
	router.Use(requestMetadata(logger), recoverRequest, localRequests(devOrigin), requestDeadline(RequestTimeout), limitJSON)
	router.NoRoute(func(c *gin.Context) {
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
	return router
}

type apiError struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	RequestID string `json:"requestId"`
}

func fail(c *gin.Context, status int, code, message string) {
	c.AbortWithStatusJSON(status, gin.H{"error": apiError{code, message, c.Writer.Header().Get("X-Request-Id")}})
}
