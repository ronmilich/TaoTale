package httpapi

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"hash/crc32"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"taotale/backend/internal/media"
	"taotale/backend/internal/persistence"
	"taotale/backend/internal/persistence/queries"
	"taotale/backend/internal/tts"
)

func TestCoverPostgres(t *testing.T) {
	pool, _ := libraryTestDatabase(t)
	root := t.TempDir()
	if err := media.PrepareRoot(root); err != nil {
		t.Fatal(err)
	}
	api := Router(HealthChecks{}, slog.New(slog.DiscardHandler), "", queries.New(pool), nil, tts.Mock{}, &persistence.Library{Pool: pool, MediaRoot: root})
	call := func(method, target, contentType string, body io.Reader, etag string, status int) *httptest.ResponseRecorder {
		t.Helper()
		req := httptest.NewRequest(method, "http://localhost/api/v1"+target, body)
		req.Header.Set("Content-Type", contentType)
		req.Header.Set("If-None-Match", etag)
		res := httptest.NewRecorder()
		api.ServeHTTP(res, req)
		if res.Code != status {
			if res.Body.Len() > 512 {
				t.Fatalf("%s %s: got %d want %d (%d response bytes)", method, target, res.Code, status, res.Body.Len())
			}
			t.Fatalf("%s %s: got %d want %d: %s", method, target, res.Code, status, res.Body)
		}
		return res
	}
	object := func(res *httptest.ResponseRecorder) map[string]any {
		t.Helper()
		var data map[string]any
		if err := json.Unmarshal(res.Body.Bytes(), &data); err != nil {
			t.Fatal(err)
		}
		return data
	}
	upload := func(contentType string, data []byte, status int) map[string]any {
		return object(call("POST", "/storage/images", contentType, bytes.NewReader(data), "", status))
	}
	exact := paddedPNG(t, media.MaxCoverBytes)
	uploaded := upload("image/png", exact, http.StatusCreated)
	if uploaded["sizeBytes"] != float64(media.MaxCoverBytes) || uploaded["width"] != float64(1) {
		t.Fatal(uploaded)
	}
	first := uploaded["id"].(string)
	target := "/storage/images/" + first
	sent := call("GET", target, "", nil, "", 200)
	if sent.Header().Get("Content-Type") != "image/png" || !bytes.Equal(sent.Body.Bytes(), exact) {
		t.Fatal("image changed between upload and read")
	}
	etag := sent.Header().Get("ETag")
	if etag == "" {
		t.Fatal("image response is missing ETag")
	}
	unchanged := call("GET", target, "", nil, "W/"+etag, 304)
	if unchanged.Body.Len() != 0 || unchanged.Header().Get("ETag") != etag {
		t.Fatal("invalid conditional response")
	}
	oversized := paddedPNG(t, media.MaxCoverBytes+1)
	upload("image/png", oversized, 413)
	// A reader with no known content length exercises the streamed byte limit.
	call("POST", "/storage/images", "image/png", io.LimitReader(bytes.NewReader(oversized), int64(len(oversized))), "", 413)
	upload("image/jpeg", exact, 400)
	upload("image/png", exact[:40], 400)
	var wide bytes.Buffer
	if err := png.Encode(&wide, image.NewRGBA(image.Rect(0, 0, 4097, 1))); err != nil {
		t.Fatal(err)
	}
	upload("image/png", wide.Bytes(), 400)
	var jpg bytes.Buffer
	if err := jpeg.Encode(&jpg, image.NewRGBA(image.Rect(0, 0, 8, 8)), nil); err != nil {
		t.Fatal(err)
	}
	second := upload("image/jpeg", jpg.Bytes(), 201)["id"].(string)
	webp, err := base64.StdEncoding.DecodeString("UklGRjgAAABXRUJQVlA4ICwAAACQAQCdASoIAAgAAgA0JaACdLoAA5gA/u8pR8TxdvIb/6MP/xpH/GkfDmAAAA==")
	if err != nil {
		t.Fatal(err)
	}
	upload("image/webp", webp, 201)
	for _, path := range []string{"/storage/images/%2e%2e", "/storage/images/%2e%2e%2f" + first, "/storage/images/not-a-uuid"} {
		call("GET", path, "", nil, "", 404)
	}
	draft := object(call("POST", "/items", "application/json", bytes.NewBufferString(`{"languageCode":"ko-KR","coverImageId":"`+first+`"}`), "", 201))
	itemPath := "/items/" + draft["id"].(string)
	if draft["coverImageId"] != first {
		t.Fatal(draft)
	}
	// A failed transaction must leave both the previous attachment and the staged lease intact.
	patch := func(raw string, status int) map[string]any {
		return object(call("PATCH", itemPath, "application/json", bytes.NewBufferString(raw), "", status))
	}
	patch(`{"expectedVersion":1,"coverImageId":"`+second+`","topicIds":["`+first+`"]}`, 404)
	persisted := object(call("GET", itemPath, "", nil, "", 200))
	if persisted["coverImageId"] != first || persisted["version"] != float64(1) {
		t.Fatal(persisted)
	}
	var pending bool
	if err := pool.QueryRow(t.Context(), "SELECT expires_at IS NOT NULL FROM public.media_images WHERE id=$1", second).Scan(&pending); err != nil || !pending {
		t.Fatalf("replacement should remain staged: %v", err)
	}
	if saved := patch(`{"expectedVersion":1,"coverImageId":"`+second+`"}`, 200); saved["coverImageId"] != second {
		t.Fatal(saved)
	}
	// Simulate a legacy oversized immutable image in storage. Reads do not enforce the upload gate.
	legacyKey := "images/legacy.png"
	if err := os.WriteFile(filepath.Join(root, legacyKey), oversized, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(t.Context(), "UPDATE public.media_images SET storage_key=$2, size_bytes=$3 WHERE id=$1", first, legacyKey, len(oversized)); err != nil {
		t.Fatal(err)
	}
	legacy := call("GET", target, "", nil, "", 200)
	if legacy.Body.Len() != media.MaxCoverBytes+1 || legacy.Header().Get("Content-Length") != strconv.Itoa(len(oversized)) {
		t.Fatal("legacy image must keep all its original bytes")
	}
}

func paddedPNG(t *testing.T, size int) []byte {
	t.Helper()
	var pngBytes bytes.Buffer
	if err := png.Encode(&pngBytes, image.NewRGBA(image.Rect(0, 0, 1, 1))); err != nil {
		t.Fatal(err)
	}
	raw := pngBytes.Bytes()
	data := make([]byte, size-len(raw))
	binary.BigEndian.PutUint32(data, uint32(len(data)-12))
	copy(data[4:], "npAd")
	binary.BigEndian.PutUint32(data[len(data)-4:], crc32.ChecksumIEEE(data[4:len(data)-4]))
	return bytes.Join([][]byte{raw[:len(raw)-12], data, raw[len(raw)-12:]}, nil)
}
