package httpapi

import (
	"net/http"
	"net/url"

	"github.com/gin-gonic/gin"
	"taotale/backend/internal/tts"
)

func listVoices(provider tts.TTSProvider) gin.HandlerFunc {
	return func(c *gin.Context) {
		query, err := url.ParseQuery(c.Request.URL.RawQuery)
		languages := query["languageCode"]
		if err != nil || len(languages) != 1 || !tts.SupportsLanguage(languages[0]) {
			fail(c, http.StatusBadRequest, string(tts.Invalid), "Choose one supported learning language.")
			return
		}
		voices, err := provider.ListVoices(c.Request.Context(), languages[0])
		if err != nil {
			fail(c, http.StatusServiceUnavailable, string(tts.Unavailable), "Speech synthesis is unavailable.")
			return
		}
		c.JSON(http.StatusOK, gin.H{"items": voices})
	}
}
