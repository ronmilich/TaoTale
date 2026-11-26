package httpapi

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestDecodeJSON(t *testing.T) {
	router := Router(nil, slog.New(slog.DiscardHandler), "", nil)
	router.POST("/write", func(c *gin.Context) {
		var input struct {
			Text string `json:"text"`
		}
		if DecodeJSON(c, &input) {
			c.String(http.StatusOK, "%d:%s", len(input.Text), input.Text[:min(len(input.Text), 7)])
		}
	})
	maximum := `{"text":"` + strings.Repeat("x", MaxJSONBytes-len(`{"text":""}`)) + `"}`
	for _, tc := range []struct {
		name, body, contentType string
		status                  int
		codeOrBody              string
		unknownLength           bool
	}{
		{"UTF-8 and surrogate pair", `{"text":"中\uD83D\uDE00"}`, "application/json", 200, "7:中😀", false},
		{"invalid UTF-8", "{\"text\":\"\xff\"}", "application/json", 400, "VALIDATION_ERROR", false},
		{"isolated high surrogate", `{"text":"\uD800"}`, "application/json", 400, "VALIDATION_ERROR", false},
		{"isolated low surrogate", `{"text":"\uDC00"}`, "application/json", 400, "VALIDATION_ERROR", false},
		{"escaped backslash", `{"text":"\\uD800"}`, "application/json", 200, `6:\uD800`, false},
		{"unknown field", `{"private-unknown":"secret"}`, "application/json", 400, "VALIDATION_ERROR", false},
		{"second value", `{}{}`, "application/json", 400, "VALIDATION_ERROR", false},
		{"null", "null", "application/json", 400, "VALIDATION_ERROR", false},
		{"empty", "", "application/json", 400, "VALIDATION_ERROR", false},
		{"incorrect content type", "{}", "text/plain", 415, "UNSUPPORTED_MEDIA_TYPE", false},
		{"exact limit", maximum, "application/json", 200, "102389:xxxxxxx", false},
		{"known oversized length", maximum + " ", "application/json", 413, "PAYLOAD_TOO_LARGE", false},
		{"streamed oversized length", maximum + " ", "application/json", 413, "PAYLOAD_TOO_LARGE", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var body io.Reader = strings.NewReader(tc.body)
			if tc.unknownLength {
				body = io.NopCloser(body)
			}
			req := httptest.NewRequest("POST", "http://127.0.0.1/write", body)
			req.Header.Set("Content-Type", tc.contentType)
			res := httptest.NewRecorder()
			router.ServeHTTP(res, req)
			if tc.status == 200 {
				if res.Code != 200 || res.Body.String() != tc.codeOrBody {
					t.Fatalf("unexpected decoded response: %d %s", res.Code, res.Body)
				}
			} else {
				assertAPIError(t, res, tc.status, tc.codeOrBody)
				if strings.Contains(res.Body.String(), "private-unknown") {
					t.Fatal("validation leaked a field name")
				}
			}
		})
	}
}
