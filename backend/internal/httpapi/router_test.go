package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func TestHealthAndSanitizedErrors(t *testing.T) {
	router := Router(func(context.Context) error { return errors.New("private diagnostic") }, slog.New(slog.DiscardHandler), "", nil)
	for _, prefix := range []string{"", "/api/v1"} {
		live := httptest.NewRecorder()
		router.ServeHTTP(live, httptest.NewRequest("GET", "http://localhost"+prefix+"/health/live", nil))
		if live.Code != 200 || live.Body.String() != `{"status":"ok"}` || live.Header().Get("Cache-Control") != "no-store" {
			t.Fatalf("unexpected liveness response: %d %s", live.Code, live.Body)
		}
		if live.Header().Get("X-Request-Id") == "" {
			t.Fatal("liveness response has no request ID")
		}
		ready := httptest.NewRecorder()
		router.ServeHTTP(ready, httptest.NewRequest("GET", "http://127.0.0.1"+prefix+"/health/ready", nil))
		assertAPIError(t, ready, 503, "SERVICE_UNAVAILABLE")
		if strings.Contains(ready.Body.String(), "private diagnostic") {
			t.Fatal("readiness leaked an internal diagnostic")
		}
	}
	wrongMethod := httptest.NewRecorder()
	router.ServeHTTP(wrongMethod, httptest.NewRequest("POST", "http://localhost/health/live", nil))
	assertAPIError(t, wrongMethod, 405, "METHOD_NOT_ALLOWED")
}

func TestLocalWritePolicy(t *testing.T) {
	router := Router(nil, slog.New(slog.DiscardHandler), "localhost:3000", nil)
	router.POST("/write", func(c *gin.Context) { c.Status(204) })
	for _, tc := range []struct {
		name, host, origin, fetchSite string
		status                        int
	}{
		{"local CLI", "127.0.0.1:8080", "", "", 204},
		{"same origin", "127.0.0.1:8080", "http://127.0.0.1:8080", "", 204},
		{"IPv6", "[::1]:8080", "http://[::1]:8080", "", 204},
		{"configured development origin", "127.0.0.1:8080", "http://localhost:3000", "", 204},
		{"rebinding hostname", "other.example:8080", "http://other.example:8080", "", 403},
		{"unexpected local port", "localhost:8080", "http://localhost:3001", "", 403},
		{"cross-site form", "localhost:8080", "http://other.example", "", 403},
		{"null origin", "localhost:8080", "null", "", 403},
		{"cross-site without origin", "localhost:8080", "", "cross-site", 403},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "http://"+tc.host+"/write", nil)
			if tc.origin != "" {
				req.Header.Set("Origin", tc.origin)
			}
			req.Header.Set("Sec-Fetch-Site", tc.fetchSite)
			res := httptest.NewRecorder()
			router.ServeHTTP(res, req)
			if res.Code != tc.status {
				t.Fatalf("status = %d, want %d, body = %s", res.Code, tc.status, res.Body)
			}
			if tc.status == 403 {
				assertAPIError(t, res, 403, "FORBIDDEN")
			}
		})
	}
}

func TestRequestLogsExcludePrivateData(t *testing.T) {
	var logs bytes.Buffer
	router := Router(nil, slog.New(slog.NewJSONHandler(&logs, nil)), "", nil)
	router.POST("/items/:id", func(c *gin.Context) {
		_, _ = io.ReadAll(c.Request.Body)
		panic("private-panic")
	})
	req := httptest.NewRequest("POST", "http://localhost/items/private-path?q=private-query", strings.NewReader("private-body"))
	req.Header.Set("Authorization", "private-token")
	req.Header.Set("X-Request-Id", "private-client-id")
	res := httptest.NewRecorder()
	router.ServeHTTP(res, req)
	assertAPIError(t, res, 500, "INTERNAL_ERROR")
	if strings.Contains(logs.String()+res.Body.String(), "private-") {
		t.Fatalf("private request data leaked")
	}
	var record struct {
		Route     string `json:"route"`
		RequestID string `json:"request_id"`
		Status    int    `json:"status"`
	}
	if err := json.Unmarshal(logs.Bytes(), &record); err != nil {
		t.Fatal(err)
	}
	if record.Route != "/items/:id" || record.Status != 500 || record.RequestID != res.Header().Get("X-Request-Id") {
		t.Fatalf("unexpected request metadata: %+v", record)
	}
	missing := httptest.NewRecorder()
	router.ServeHTTP(missing, httptest.NewRequest("GET", "http://localhost/missing/private-path", nil))
	assertAPIError(t, missing, 404, "NOT_FOUND")
	if strings.Contains(logs.String(), "private-path") {
		t.Fatal("404 logged the unmatched request path")
	}
}

func TestRequestDeadline(t *testing.T) {
	router := gin.New()
	router.Use(requestMetadata(slog.New(slog.DiscardHandler)), requestDeadline(time.Millisecond))
	router.GET("/wait", func(c *gin.Context) { <-c.Request.Context().Done() })
	res := httptest.NewRecorder()
	router.ServeHTTP(res, httptest.NewRequest("GET", "http://localhost/wait", nil))
	assertAPIError(t, res, 504, "REQUEST_TIMEOUT")
}

func assertAPIError(t *testing.T, res *httptest.ResponseRecorder, status int, code string) {
	t.Helper()
	var body struct {
		Error apiError `json:"error"`
	}
	if err := json.Unmarshal(res.Body.Bytes(), &body); err != nil {
		t.Fatalf("invalid error response: %v", err)
	}
	if res.Code != status || body.Error.Code != code || body.Error.Message == "" ||
		body.Error.RequestID == "" || body.Error.RequestID != res.Header().Get("X-Request-Id") {
		t.Fatalf("unexpected error response: %d %s", res.Code, res.Body)
	}
}
