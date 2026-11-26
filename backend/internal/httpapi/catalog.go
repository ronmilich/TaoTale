package httpapi

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"taotale/backend/internal/persistence/queries"
)

type Catalog interface {
	ListEnabledLanguages(context.Context) ([]queries.ListEnabledLanguagesRow, error)
	ListTopics(context.Context) ([]queries.ListTopicsRow, error)
}

type learningLanguage struct {
	Code        string `json:"code"`
	DisplayName string `json:"displayName"`
}

type themeColors struct {
	Background string `json:"background"`
	Text       string `json:"text"`
}

type topicColors struct {
	Light themeColors `json:"light"`
	Dark  themeColors `json:"dark"`
}

type topic struct {
	ID        string      `json:"id"`
	Name      string      `json:"name"`
	IsDefault bool        `json:"isDefault"`
	SystemKey *string     `json:"systemKey"`
	Colors    topicColors `json:"colors"`
}

func listLanguages(catalog Catalog) gin.HandlerFunc {
	return func(c *gin.Context) {
		rows, err := catalog.ListEnabledLanguages(c.Request.Context())
		if err != nil {
			fail(c, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "Backend storage unavailable.")
			return
		}
		items := make([]learningLanguage, 0, len(rows))
		for _, row := range rows {
			items = append(items, learningLanguage{Code: row.Code, DisplayName: row.DisplayName})
		}
		c.JSON(http.StatusOK, gin.H{"items": items})
	}
}

func listTopics(catalog Catalog) gin.HandlerFunc {
	return func(c *gin.Context) {
		rows, err := catalog.ListTopics(c.Request.Context())
		if err != nil {
			fail(c, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "Backend storage unavailable.")
			return
		}
		items := make([]topic, 0, len(rows))
		for _, row := range rows {
			var key *string
			if row.SystemKey.Valid {
				key = &row.SystemKey.String
			}
			items = append(items, topic{
				ID: row.ID.String(), Name: row.Name, IsDefault: key != nil, SystemKey: key,
				Colors: topicColors{
					Light: themeColors{Background: row.LightBackground, Text: row.LightText},
					Dark:  themeColors{Background: row.DarkBackground, Text: row.DarkText},
				},
			})
		}
		c.JSON(http.StatusOK, gin.H{"items": items})
	}
}
