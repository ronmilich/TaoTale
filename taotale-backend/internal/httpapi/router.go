package httpapi

import (
	"context"
	"crypto/rand"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

func Router(ready func(context.Context) error) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(func(c *gin.Context) {
		c.Header("Cache-Control", "no-store")
		c.Header("X-Request-Id", rand.Text())
		c.Next()
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
				c.JSON(http.StatusServiceUnavailable, gin.H{"error": gin.H{
					"code": "SERVICE_UNAVAILABLE", "message": "Backend storage unavailable.",
					"requestId": c.Writer.Header().Get("X-Request-Id"),
				}})
				return
			}
			c.JSON(http.StatusOK, gin.H{"status": "ok"})
		})
	}
	return router
}
