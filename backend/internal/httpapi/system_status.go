package httpapi

import (
	"context"
	"net/http"
	"os/exec"
	"time"

	"github.com/gin-gonic/gin"
	"taotale/backend/internal/segmentation"
	"taotale/backend/internal/tts"
)

const appVersion = "0.1.0-dev"

type HealthChecks struct {
	Database, Schema, Media func(context.Context) error
	ExpectedSchemaRevision  int64
}

func passes(ctx context.Context, probe func(context.Context) error) bool {
	return probe != nil && probe(ctx) == nil
}

func systemStatus(health HealthChecks, provider tts.VoiceCatalog) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		database, schema, media := "unavailable", "unavailable", "unavailable"
		dbContext, cancel := context.WithTimeout(ctx, 2*time.Second)
		if passes(dbContext, health.Database) {
			database = "ready"
			if passes(dbContext, health.Schema) {
				schema = "current"
			}
		}
		cancel()
		if passes(ctx, health.Media) {
			media = "writable"
		}

		providerID := "azure"
		if provider.ProviderID() == "mock" {
			providerID = "mock"
		}
		var configured bool
		if azure, ok := provider.(*tts.Azure); ok {
			configured = azure.Configured()
		}
		_, decoderErr := exec.LookPath("ffmpeg")
		var strategies []gin.H
		available := decoderErr == nil
		for _, language := range []string{"zh-CN", "ko-KR"} {
			strategy, err := segmentation.Default(language)
			if err == nil {
				strategies = append(strategies, gin.H{
					"languageCode": strategy.LanguageCode, "strategyId": strategy.StrategyID, "version": strategy.Version,
				})
			}
			if _, err := provider.ListVoices(ctx, language); err != nil {
				available = false
			}
		}
		var revision *int64
		if health.ExpectedSchemaRevision > 0 {
			revision = &health.ExpectedSchemaRevision
		}
		c.Header("Cache-Control", "no-store")
		c.JSON(http.StatusOK, gin.H{
			"appVersion": appVersion, "database": database, "schema": schema,
			"expectedSchemaRevision": revision, "media": media, "segmentation": strategies,
			"speech": gin.H{"provider": providerID, "azureConfigured": configured, "available": available},
		})
	}
}
