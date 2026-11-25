package httpapi

import (
	"context"
	"crypto/rand"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

func requestMetadata(logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := rand.Text()
		c.Header("X-Request-Id", id)
		c.Header("Cache-Control", "no-store")
		start := time.Now()
		defer func() {
			method := c.Request.Method
			switch method {
			case http.MethodGet, http.MethodHead, http.MethodOptions, http.MethodPost,
				http.MethodPut, http.MethodPatch, http.MethodDelete, http.MethodTrace, http.MethodConnect:
			default:
				method = "OTHER"
			}
			logger.Info("http request", "request_id", id, "method", method, "route", c.FullPath(),
				"status", c.Writer.Status(), "duration_ms", time.Since(start).Milliseconds())
		}()
		c.Next()
	}
}

func recoverRequest(c *gin.Context) {
	defer func() {
		if recover() != nil {
			if c.Writer.Written() {
				c.Abort()
				return
			}
			fail(c, http.StatusInternalServerError, "INTERNAL_ERROR", "An internal error occurred.")
		}
	}()
	c.Next()
}

func requestDeadline(timeout time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), timeout)
		defer cancel()
		c.Request = c.Request.WithContext(ctx)
		c.Next()
		if !c.Writer.Written() && errors.Is(ctx.Err(), context.DeadlineExceeded) {
			fail(c, http.StatusGatewayTimeout, "REQUEST_TIMEOUT", "The request timed out.")
		}
	}
}
