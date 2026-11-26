package httpapi

import (
	"log/slog"
	"net/http/httptest"
	"strings"
	"taotale/backend/internal/tts"
	"testing"
	"testing/fstest"
)

func TestSPAFallback(t *testing.T) {
	spa, err := NewSPA(fstest.MapFS{
		"index.html":       {Data: []byte("<!doctype html><title>TaoTale</title>")},
		"assets/app.css":   {Data: []byte("body{}")},
		"assets/.private":  {Data: []byte("private")},
		".vite/index.json": {Data: []byte("private")},
	})
	if err != nil {
		t.Fatal(err)
	}
	router := Router(nil, slog.New(slog.DiscardHandler), "", nil, spa, tts.Mock{}, nil)
	for _, test := range []struct {
		method, url, contentType string
		status                   int
	}{
		{"GET", "/items/example/edit", "text/html", 200},
		{"HEAD", "/library", "text/html", 200},
		{"GET", "/assets/app.css", "text/css", 200},
		{"GET", "/assets/missing.js", "application/json", 404},
		{"GET", "/api", "application/json", 404},
		{"GET", "/api/v1/missing", "application/json", 404},
		{"GET", "/assets/.private", "application/json", 404},
		{"GET", "/.vite/index.json", "application/json", 404},
		{"POST", "/items/example/edit", "application/json", 404},
	} {
		t.Run(test.method+" "+test.url, func(t *testing.T) {
			req := httptest.NewRequest(test.method, "http://127.0.0.1"+test.url, nil)
			response := httptest.NewRecorder()
			router.ServeHTTP(response, req)
			if response.Code != test.status || !strings.HasPrefix(response.Header().Get("Content-Type"), test.contentType) {
				t.Fatalf("unexpected response: %d %s", response.Code, response.Header())
			}
			if test.status == 404 {
				assertAPIError(t, response, 404, "NOT_FOUND")
			}
		})
	}
}
