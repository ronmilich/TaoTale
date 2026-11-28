package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"taotale/backend/internal/generation"
	"taotale/backend/internal/media"
	"taotale/backend/internal/persistence"
	"taotale/backend/internal/tts"
)

type previewProvider struct {
	tts.Mock
	calls atomic.Int32
}

func (p *previewProvider) Synthesize(ctx context.Context, input tts.Request) (tts.Result, error) {
	p.calls.Add(1)
	return p.Mock.Synthesize(ctx, input)
}

type audioTestCall func(method, endpoint, key, body string, ranges ...string) *httptest.ResponseRecorder

func startAudioTest(t *testing.T, provider tts.TTSProvider) (*pgxpool.Pool, string, *persistence.Library, audioTestCall) {
	t.Helper()
	pool, _ := libraryTestDatabase(t)
	root := t.TempDir()
	if err := media.PrepareRoot(root); err != nil {
		t.Fatal(err)
	}
	worker, err := generation.Open(t.Context(), pool, root, provider)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan error, 1)
	go func() { done <- worker.Run(ctx, 1) }()
	t.Cleanup(func() {
		cancel()
		if err := <-done; err != nil {
			t.Error(err)
		}
		worker.Close()
	})
	store := &persistence.Library{Pool: pool, MediaRoot: root, Limits: persistence.GenerationLimits{Active: 2, Hourly: 1000}}
	api := Router(nil, slog.New(slog.DiscardHandler), "", nil, nil, provider, store)

	call := func(method, endpoint, key, body string, ranges ...string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(method, "http://localhost/api/v1"+endpoint, strings.NewReader(body))
		if body != "" {
			req.Header.Set("Content-Type", "application/json")
		}
		if key != "" {
			req.Header.Set("Idempotency-Key", key)
		}
		for _, value := range ranges {
			req.Header.Add("Range", value)
		}
		res := httptest.NewRecorder()
		api.ServeHTTP(res, req)
		return res
	}
	return pool, root, store, call
}

func audioResult(t *testing.T, res *httptest.ResponseRecorder, status int) map[string]any {
	t.Helper()
	var result map[string]any
	if res.Code != status || json.Unmarshal(res.Body.Bytes(), &result) != nil {
		t.Fatalf("wanted %d, got %d: %s", status, res.Code, res.Body)
	}
	return result
}

func TestPreviewAudioPostgres(t *testing.T) {
	provider := &previewProvider{}
	pool, root, store, call := startAudioTest(t, provider)
	object := func(res *httptest.ResponseRecorder, status int) map[string]any {
		t.Helper()
		return audioResult(t, res, status)
	}
	itemCount := func() int {
		t.Helper()
		var count int
		if err := pool.QueryRow(t.Context(), "SELECT count(*) FROM library_items").Scan(&count); err != nil {
			t.Fatal(err)
		}
		return count
	}
	sqlExec := func(sql string, args ...any) {
		t.Helper()
		if _, err := pool.Exec(t.Context(), sql, args...); err != nil {
			t.Fatal(err)
		}
	}
	const previews = "/tts/previews"
	const firstKey = "aaee0000-0000-4000-b000-000000000001"
	const secondKey = "aaee0000-0000-4000-b000-000000000002"
	const expiryKey = "aaee0000-0000-4000-b000-000000000003"
	const firstBody = `{"languageCode":"zh-CN","sourceText":"你好。","voiceId":"zh-CN-mock","speakingRate":1}`
	const secondBody = `{"languageCode":"zh-CN","sourceText":"再见。","voiceId":"zh-CN-mock","speakingRate":1}`
	admit := func(key, body string) string {
		t.Helper()
		return object(call(http.MethodPost, previews, key, body), 202)["generationId"].(string)
	}
	ready := func(id string) string {
		t.Helper()
		deadline := time.Now().Add(12 * time.Second)
		for {
			result := object(call(http.MethodGet, previews+"/"+id, "", ""), 200)
			switch result["status"] {
			case "READY":
				return result["audioAssetId"].(string)
			case "FAILED":
				t.Fatalf("synthesis failed: %v", result)
			}
			if time.Now().After(deadline) {
				t.Fatalf("preview did not become ready: %v", result)
			}
			time.Sleep(25 * time.Millisecond)
		}
	}
	assertGone := func(path string) {
		t.Helper()
		for _, ending := range []string{"", "/audio", "/audio/download"} {
			assertAPIError(t, call(http.MethodGet, path+ending, "", "", "bytes=0-1"), 404, "PREVIEW_NOT_FOUND")
		}
	}

	startCount := itemCount()
	first := admit(firstKey, firstBody)
	if replay := admit(firstKey, firstBody); replay != first {
		t.Fatalf("replay returned %s for %s", replay, first)
	}
	assertAPIError(t, call(http.MethodPost, previews, firstKey, secondBody), 409, "IDEMPOTENCY_KEY_REUSED")
	firstAsset := ready(first)
	if provider.calls.Load() != 1 || itemCount() != startCount {
		t.Fatal("previewing should synthesize once and leave the library untouched")
	}
	preview := previews + "/" + first
	full := call(http.MethodGet, preview+"/audio", "", "")
	audio := full.Body.Bytes()
	if full.Code != 200 || len(audio) < 10 || full.Header().Get("Content-Type") != "audio/mpeg" ||
		full.Header().Get("Cache-Control") != "private, no-store" || full.Header().Get("Accept-Ranges") != "bytes" ||
		full.Header().Get("Content-Length") != strconv.Itoa(len(audio)) {
		t.Fatalf("invalid preview stream: %d %v", full.Code, full.Header())
	}
	for _, r := range []struct {
		header   string
		from, to int
	}{
		{"bytes=1-4", 1, 5},
		{"bytes=4-", 4, len(audio)},
		{"bytes=-3", len(audio) - 3, len(audio)},
		{"bytes=0-9999999999999999999999", 0, len(audio)},
	} {
		part := call(http.MethodGet, preview+"/audio/download", "", "", r.header)
		wantRange := fmt.Sprintf("bytes %d-%d/%d", r.from, r.to-1, len(audio))
		wantName := fmt.Sprintf(`attachment; filename="taotale-preview-%s.mp3"`, first)
		if part.Code != 206 || !bytes.Equal(part.Body.Bytes(), audio[r.from:r.to]) ||
			part.Header().Get("Content-Range") != wantRange || part.Header().Get("Content-Length") != strconv.Itoa(r.to-r.from) ||
			part.Header().Get("Content-Type") != "audio/mpeg" || part.Header().Get("Content-Disposition") != wantName {
			t.Fatalf("invalid partial download for %s: %d %v", r.header, part.Code, part.Header())
		}
	}
	for _, headers := range [][]string{{"bytes=-0"}, {"bytes=4-2"}, {"bytes=999999999-"}, {"bytes=0-1,3-4"}, {"bytes=0-1", "bytes=3-4"}, {"items=1-2"}} {
		res := call(http.MethodGet, preview+"/audio", "", "", headers...)
		assertAPIError(t, res, 416, "INVALID_RANGE")
		if res.Header().Get("Content-Range") != fmt.Sprintf("bytes */%d", len(audio)) {
			t.Fatal(res.Header())
		}
	}

	save := func(generationID, source string) string {
		return fmt.Sprintf(`{"languageCode":"zh-CN","title":"Dangerous / File","sourceText":%q,"previewGenerationId":%q}`, source, generationID)
	}
	assertAPIError(t, call(http.MethodPost, "/items", "", save(first, " 你好。")), 409, "PREVIEW_MISMATCH")
	results := make(chan *httptest.ResponseRecorder, 2)
	for range 2 {
		go func() { results <- call(http.MethodPost, "/items", "", save(first, "你好。")) }()
	}
	winner, loser := <-results, <-results
	if winner.Code != 201 {
		winner, loser = loser, winner
	}
	item := object(winner, 201)
	assertAPIError(t, loser, 404, "PREVIEW_NOT_FOUND")
	if itemCount() != startCount+1 || item["audioStatus"] != "READY" || item["audioAssetId"] != firstAsset ||
		provider.calls.Load() != 1 {
		t.Fatalf("claim did not atomically reuse the preview: %v", item)
	}
	assertGone(preview)
	itemID := item["id"].(string)
	itemPath := "/items/" + itemID
	pinned := func(asset string) string { return itemPath + "/audio?audioAssetId=" + asset }
	assertAPIError(t, call(http.MethodGet, itemPath+"/audio", "", ""), 400, "VALIDATION_ERROR")
	dl := call(http.MethodGet, itemPath+"/audio/download?audioAssetId="+firstAsset, "", "", "bytes=0-2")
	if dl.Code != 206 || !bytes.Equal(dl.Body.Bytes(), audio[:3]) || dl.Header().Get("Cache-Control") != "private, no-cache" ||
		dl.Header().Get("Content-Disposition") != fmt.Sprintf(`attachment; filename="taotale-%s.mp3"`, itemID) {
		t.Fatalf("invalid saved download: %d %v", dl.Code, dl.Header())
	}

	second := admit(secondKey, secondBody)
	secondAsset := ready(second)
	mismatch := fmt.Sprintf(`{"expectedVersion":1,"title":"Changed","previewGenerationId":%q}`, second)
	assertAPIError(t, call(http.MethodPatch, itemPath, "", mismatch), 409, "PREVIEW_MISMATCH")
	patch := fmt.Sprintf(`{"expectedVersion":1,"sourceText":"再见。","previewGenerationId":%q}`, second)
	updated := object(call(http.MethodPatch, itemPath, "", patch), 200)
	if updated["title"] != item["title"] || updated["version"] != float64(2) || updated["audioAssetId"] != secondAsset {
		t.Fatalf("draft patch or previous rollback failed: %v", updated)
	}
	assertAPIError(t, call(http.MethodGet, pinned(firstAsset), "", "", "bytes=0-2"), 404, "AUDIO_NOT_FOUND")
	if res := call(http.MethodGet, pinned(secondAsset), "", "", "bytes=0-2"); res.Code != 206 {
		t.Fatalf("replacement pin unavailable: %d %s", res.Code, res.Body)
	}

	t.Run("saved publication rechecks the version, exact source, latest pin and media", func(t *testing.T) {
		publish := func(body string) *httptest.ResponseRecorder { return call("POST", itemPath+"/publish", "", body) }
		assertAPIError(t, publish(`{}`), 400, "VALIDATION_ERROR")
		assertAPIError(t, publish(`{"expectedVersion":1}`), 409, "VERSION_CONFLICT")
		assertAPIError(t, publish(fmt.Sprintf(`{"expectedVersion":2,"audioAssetId":%q}`, firstAsset)), 409, "AUDIO_NOT_READY")
		// An older READY asset matches this source, but the latest admission does not.
		object(call("PATCH", itemPath, "", `{"expectedVersion":2,"sourceText":"你好。"}`), 200)
		assertAPIError(t, publish(`{"expectedVersion":3}`), 409, "AUDIO_NOT_READY")
		object(call("PATCH", itemPath, "", `{"expectedVersion":3,"sourceText":"再见。","title":"　"}`), 200)
		assertAPIError(t, publish(`{"expectedVersion":4}`), 400, "VALIDATION_ERROR")
		object(call("PATCH", itemPath, "", `{"expectedVersion":4,"title":"Saved article"}`), 200)

		var storage string
		if err := pool.QueryRow(t.Context(), "SELECT storage_key FROM audio_assets WHERE id=$1", secondAsset).Scan(&storage); err != nil {
			t.Fatal(err)
		}
		file := filepath.Join(root, storage)
		if err := os.Rename(file, file+".offline"); err != nil {
			t.Fatal(err)
		}
		assertAPIError(t, publish(`{"expectedVersion":5}`), 503, "AUDIO_UNAVAILABLE")
		if err := os.Rename(file+".offline", file); err != nil {
			t.Fatal(err)
		}
		before := object(call("GET", itemPath, "", ""), 200)
		if before["status"] != "draft" || before["version"] != float64(5) || before["audioAssetId"] != secondAsset {
			t.Fatalf("failed publication changed draft: %v", before)
		}
		body := fmt.Sprintf(`{"expectedVersion":5,"audioAssetId":%q}`, secondAsset)
		for range 2 {
			go func() { results <- publish(body) }()
		}
		winner, loser := <-results, <-results
		if winner.Code != 200 {
			winner, loser = loser, winner
		}
		if object(winner, 200)["itemId"] != itemID {
			t.Fatal("wrong published item")
		}
		assertAPIError(t, loser, 409, "VERSION_CONFLICT")
		after := object(call("GET", itemPath, "", ""), 200)
		if after["version"] != float64(6) || after["publishedAt"] == nil || after["status"] != "published" ||
			after["audioAssetId"] != secondAsset || after["voiceId"] != "zh-CN-mock" || after["speakingRate"] != float64(1) {
			t.Fatalf("incorrect publication: %v", after)
		}
		if before["transcript"] != nil {
			t.Fatal("draft detail included a published transcript")
		}
		transcript, ok := after["transcript"].(map[string]any)
		if !ok || transcript["durationMs"].(float64) <= 0 ||
			transcript["segments"].([]any)[0].(map[string]any)["text"] != "再见。" {
			t.Fatalf("published detail did not include the stored transcript: %v", transcript)
		}
		library := object(call("GET", "/items?languageCode=zh-CN", "", ""), 200)
		for _, entry := range library["items"].([]any) {
			if entry.(map[string]any)["transcript"] != nil {
				t.Fatal("library page included a published transcript")
			}
		}
		assertAPIError(t, publish(`{"expectedVersion":6}`), 409, "INVALID_ITEM_STATE")
		assertAPIError(t, call("PATCH", itemPath, "", `{"expectedVersion":6,"title":null}`), 400, "VALIDATION_ERROR")
	})

	sqlExec("UPDATE library_items SET active_audio_asset_id=$2 WHERE id=$1", itemID, firstAsset)
	if res := call(http.MethodGet, pinned(firstAsset), "", ""); res.Code != 200 {
		t.Fatalf("published recording did not take precedence over draft history: %d %s", res.Code, res.Body)
	}
	sqlExec("UPDATE library_items SET active_audio_asset_id=$2 WHERE id=$1", itemID, secondAsset)
	assertAPIError(t, call(http.MethodGet, pinned(firstAsset), "", "", "bytes=0-2"), 404, "AUDIO_NOT_FOUND")
	if res := call(http.MethodGet, pinned(secondAsset), "", ""); res.Code != 200 {
		t.Fatalf("published active pin unavailable: %d %s", res.Code, res.Body)
	}

	expired := admit(expiryKey, firstBody)
	expiredAsset := ready(expired)
	var expiresIn24Hours bool
	err := pool.QueryRow(t.Context(), "SELECT expires_at - created_at = interval '24 hours' FROM audio_generations WHERE id=$1", expired).Scan(&expiresIn24Hours)
	if err != nil || !expiresIn24Hours {
		t.Fatalf("preview retention should be 24 hours: %v", err)
	}
	sqlExec("UPDATE audio_generations SET expires_at=statement_timestamp() WHERE id=$1", expired)
	sqlExec("UPDATE idempotency_records SET expires_at=statement_timestamp() WHERE generation_id=$1", expired)
	assertGone(previews + "/" + expired)
	assertAPIError(t, call(http.MethodPost, "/items", "", save(expired, "你好。")), 404, "PREVIEW_NOT_FOUND")
	assertAPIError(t, call(http.MethodPost, previews, expiryKey, firstBody), 409, "IDEMPOTENCY_KEY_EXPIRED")
	if itemCount() != startCount+1 || provider.calls.Load() != 3 {
		t.Fatal("an expired preview created a draft or repeated synthesis")
	}

	t.Run("direct publication atomically claims and replays the original result", func(t *testing.T) {
		const previewKey = "aaee0000-0000-4000-b000-000000000004"
		const publishKey = "aaee0000-0000-4000-b000-000000000005"
		const mismatchKey = "aaee0000-0000-4000-b000-000000000006"
		const otherKey = "aaee0000-0000-4000-b000-000000000007"
		direct := func(key, body string) *httptest.ResponseRecorder { return call("POST", "/items/publish", key, body) }
		id := admit(previewKey, firstBody)
		asset := ready(id)
		body, wrong := save(id, "你好。"), save(id, " 你好。")
		assertAPIError(t, direct("", body), 400, "VALIDATION_ERROR")
		assertAPIError(t, direct(previewKey, body), 409, "IDEMPOTENCY_KEY_REUSED")
		assertAPIError(t, direct(mismatchKey, wrong), 409, "PREVIEW_MISMATCH")
		assertAPIError(t, direct(mismatchKey, body), 409, "IDEMPOTENCY_KEY_REUSED")
		if itemCount() != startCount+1 {
			t.Fatal("rejected publication left a draft")
		}
		for range 2 {
			go func() { results <- direct(publishKey, body) }()
		}
		first, replay := object(<-results, 201), object(<-results, 201)
		if first["itemId"] != replay["itemId"] || itemCount() != startCount+2 || provider.calls.Load() != 4 {
			t.Fatal("duplicate direct publication or synthesis")
		}
		assertGone(previews + "/" + id)
		path := "/items/" + first["itemId"].(string)
		published := object(call("GET", path, "", ""), 200)
		if published["status"] != "published" || published["audioAssetId"] != asset {
			t.Fatal(published)
		}
		assertAPIError(t, direct(otherKey, body), 404, "PREVIEW_NOT_FOUND")
		assertAPIError(t, direct(mismatchKey, wrong), 409, "PREVIEW_MISMATCH")
		if res := call("DELETE", path, "", ""); res.Code != 204 {
			t.Fatal(res.Code)
		}
		if object(direct(publishKey, body), 201)["itemId"] != first["itemId"] || itemCount() != startCount+1 {
			t.Fatal("replay after deletion created an item")
		}
	})

	t.Run("item deletion releases only media with no remaining references", func(t *testing.T) {
		cover, err := store.CreateCover(t.Context(), bytes.NewReader(paddedPNG(t, 200)), "image/png")
		if err != nil {
			t.Fatal(err)
		}
		sibling := object(call("POST", "/items", "", fmt.Sprintf(`{"languageCode":"zh-CN","title":"Shared recording","coverImageId":%q}`, cover.ID.String())), 201)
		siblingID := sibling["id"].(string)
		sqlExec("UPDATE library_items SET cover_image_id=$2 WHERE id=$1", itemID, cover.ID)
		sqlExec(`UPDATE library_items SET active_audio_asset_id=$2, source_text='你好。',
			voice_id='zh-CN-mock', speaking_rate=1, published_at=now() WHERE id=$1`, siblingID, firstAsset)
		audioKey := func(id string) string {
			t.Helper()
			var key string
			if err := pool.QueryRow(t.Context(), "SELECT storage_key FROM audio_assets WHERE id=$1", id).Scan(&key); err != nil {
				t.Fatal(err)
			}
			return key
		}
		shared, exclusive := audioKey(firstAsset), audioKey(secondAsset)
		unrelated := audioKey(expiredAsset)
		assertFile := func(key string, present bool) {
			t.Helper()
			_, err := os.Stat(filepath.Join(root, key))
			if (present && err != nil) || (!present && !os.IsNotExist(err)) {
				t.Fatalf("file %s, expected present=%t: %v", key, present, err)
			}
		}
		if res := call("DELETE", itemPath, "", ""); res.Code != 204 {
			t.Fatalf("delete returned %d: %s", res.Code, res.Body)
		}
		assertFile(exclusive, false)
		assertFile(shared, true)
		assertFile(cover.StorageKey, true)
		if res := call("GET", "/items/"+siblingID+"/audio/download?audioAssetId="+firstAsset, "", ""); res.Code != 200 {
			t.Fatalf("shared download returned %d: %s", res.Code, res.Body)
		}
		assertAPIError(t, call("GET", itemPath, "", ""), 404, "ITEM_NOT_FOUND")
		if res := call("DELETE", "/items/"+siblingID, "", ""); res.Code != 204 {
			t.Fatalf("delete returned %d: %s", res.Code, res.Body)
		}
		assertFile(shared, false)
		assertFile(cover.StorageKey, false)
		assertFile(unrelated, true)
		assertAPIError(t, call("GET", "/storage/images/"+cover.ID.String(), "", ""), 404, "IMAGE_NOT_FOUND")
	})
}
