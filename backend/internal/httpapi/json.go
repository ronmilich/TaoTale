package httpapi

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net"
	"net/http"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
)

const MaxJSONBytes = 100 * 1024

func isJSON(contentType string) bool {
	kind, _, err := mime.ParseMediaType(contentType)
	return err == nil && (kind == "application/json" ||
		(strings.HasPrefix(kind, "application/") && strings.HasSuffix(kind, "+json")))
}

func limitJSON(c *gin.Context) {
	if isJSON(c.GetHeader("Content-Type")) {
		if c.Request.ContentLength > MaxJSONBytes {
			fail(c, http.StatusRequestEntityTooLarge, "PAYLOAD_TOO_LARGE", "JSON exceeds the 100 KiB limit.")
			return
		}
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, MaxJSONBytes)
	}
	c.Next()
}

// DecodeJSON reads one UTF-8 JSON object into a DTO. It writes a sanitized
// error response when the body fails validation.
func DecodeJSON(c *gin.Context, dst any) bool {
	if !isJSON(c.GetHeader("Content-Type")) {
		fail(c, http.StatusUnsupportedMediaType, "UNSUPPORTED_MEDIA_TYPE", "Send an application/json request body.")
		return false
	}
	body, err := io.ReadAll(c.Request.Body)
	var tooLarge *http.MaxBytesError
	if errors.As(err, &tooLarge) {
		fail(c, http.StatusRequestEntityTooLarge, "PAYLOAD_TOO_LARGE", "JSON exceeds the 100 KiB limit.")
		return false
	}
	var networkError net.Error
	if errors.As(err, &networkError) && networkError.Timeout() {
		fail(c, http.StatusGatewayTimeout, "REQUEST_TIMEOUT", "The request timed out.")
		return false
	}
	if err == nil && utf8.Valid(body) && json.Valid(body) && bytes.TrimSpace(body)[0] == '{' && validUnicodeEscapes(body) {
		decoder := json.NewDecoder(bytes.NewReader(body))
		decoder.DisallowUnknownFields()
		if decoder.Decode(dst) == nil {
			return true
		}
	}
	fail(c, http.StatusBadRequest, "VALIDATION_ERROR", "Send one valid JSON object using the documented fields.")
	return false
}

// Call after json.Valid. encoding/json alone replaces isolated UTF-16
// surrogates with U+FFFD, which would alter authored text.
func validUnicodeEscapes(body []byte) bool {
	for i := 0; i < len(body); i++ {
		if body[i] != '\\' {
			continue
		}
		i++
		if body[i] != 'u' {
			continue
		}
		unit, _ := strconv.ParseUint(string(body[i+1:i+5]), 16, 16)
		i += 4
		switch {
		case unit >= 0xdc00 && unit <= 0xdfff:
			return false
		case unit >= 0xd800 && unit <= 0xdbff:
			if i+6 >= len(body) || body[i+1] != '\\' || body[i+2] != 'u' {
				return false
			}
			low, _ := strconv.ParseUint(string(body[i+3:i+7]), 16, 16)
			if low < 0xdc00 || low > 0xdfff {
				return false
			}
			i += 6
		}
	}
	return true
}
