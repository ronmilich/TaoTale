package httpapi

import (
	"bytes"
	"context"
	"fmt"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"taotale/backend/internal/tts"
)

type revisionTestProvider struct{ tts.Mock }

func (p revisionTestProvider) Synthesize(ctx context.Context, req tts.Request) (tts.Result, error) {
	if req.Source == "失败。" {
		return tts.Result{}, &tts.Error{Code: tts.Failed}
	}
	return p.Mock.Synthesize(ctx, req)
}

func TestWorkingRevisionPostgres(t *testing.T) {
	pool, root, _, call := startAudioTest(t, revisionTestProvider{})
	jsonCall := func(method, path, key, body string, status int) map[string]any {
		t.Helper()
		return audioResult(t, call(method, path, key, body), status)
	}
	const draftKey = "aa340000-0000-4000-b000-000000000001"
	const failedKey = "aa340000-0000-4000-b000-000000000002"
	const readyKey = "aa340000-0000-4000-b000-000000000003"
	draft := jsonCall("POST", "/items", "", `{"languageCode":"zh-CN","title":"Original","sourceText":"你好。"}`, 201)
	itemID := draft["id"].(string)
	item := "/items/" + itemID
	revisions := item + "/revisions"
	waitAudio := func(path, status string) map[string]any {
		t.Helper()
		deadline := time.Now().Add(12 * time.Second)
		for {
			view := jsonCall("GET", path, "", "", 200)
			if view["audioStatus"] == status {
				return view
			}
			if view["audioStatus"] == "FAILED" || time.Now().After(deadline) {
				t.Fatalf("waiting for %s: %v", status, view)
			}
			time.Sleep(25 * time.Millisecond)
		}
	}
	assertAPIError(t, call("POST", revisions, "", ""), 409, "INVALID_ITEM_STATE")
	jsonCall("POST", item+"/audio/generate", draftKey, `{"expectedVersion":1}`, 202)
	original := waitAudio(item, "READY")["audioAssetId"].(string)
	jsonCall("POST", item+"/publish", "", `{"expectedVersion":1}`, 200)
	originalURL := item + "/audio?audioAssetId=" + original
	reader := call("GET", originalURL, "", "")
	if reader.Code != 200 {
		t.Fatal(reader.Code)
	}
	stillOriginal := func() {
		t.Helper()
		view := jsonCall("GET", item, "", "", 200)
		res := call("GET", originalURL, "", "")
		if res.Code != 200 || !bytes.Equal(res.Body.Bytes(), reader.Body.Bytes()) || view["sourceText"] != "你好。" || view["audioAssetId"] != original {
			t.Fatalf("published pair changed: %v, stream %d", view, res.Code)
		}
	}
	results := make(chan *httptest.ResponseRecorder, 2)
	for range 2 {
		go func() { results <- call("POST", revisions, "", "") }()
	}
	first, second := <-results, <-results
	if first.Code == 200 {
		first, second = second, first
	}
	opened, resumed := audioResult(t, first, 201), audioResult(t, second, 200)
	if opened["id"] != resumed["id"] || opened["baseAudioAssetId"] != original || opened["version"] != float64(1) ||
		opened["itemVersion"] != float64(2) || opened["audioStatus"] != "NOT_GENERATED" || opened["audioAssetId"] != nil ||
		opened["audioGenerationId"] != nil || opened["sourceText"] != "你好。" {
		t.Fatalf("unexpected initial revision: %v / %v", opened, resumed)
	}
	rev := revisions + "/" + opened["id"].(string)
	for _, path := range []string{"/audio", "/audio/download"} {
		assertAPIError(t, call("GET", rev+path+"?audioAssetId="+original, "", ""), 404, "AUDIO_NOT_FOUND")
	}
	const v1 = `{"expectedVersion":1}`
	const v2 = `{"expectedVersion":2}`
	const publish2 = `{"expectedVersion":2,"expectedItemVersion":2}`
	jsonCall("PATCH", rev, "", `{"expectedVersion":1,"sourceText":"失败。"}`, 200)
	assertAPIError(t, call("PATCH", rev, "", v1), 409, "VERSION_CONFLICT")
	assertAPIError(t, call("DELETE", rev+"?expectedVersion=1", "", ""), 409, "VERSION_CONFLICT")
	assertAPIError(t, call("POST", rev+"/audio/generate", failedKey, v1), 409, "VERSION_CONFLICT")
	jsonCall("POST", rev+"/audio/generate", failedKey, v2, 202)
	failed := waitAudio(rev, "FAILED")
	if failed["errorCode"] != string(tts.Failed) || failed["audioAssetId"] != nil {
		t.Fatal(failed)
	}
	assertAPIError(t, call("POST", rev+"/publish", "", publish2), 409, "AUDIO_NOT_READY")
	stillOriginal()
	if res := call("DELETE", rev+"?expectedVersion=2", "", ""); res.Code != 204 {
		t.Fatal(res.Code, res.Body)
	}
	assertAPIError(t, call("GET", rev, "", ""), 404, "REVISION_NOT_FOUND")
	stillOriginal()

	fresh := jsonCall("POST", revisions, "", "", 201)
	if fresh["audioStatus"] != "NOT_GENERATED" || fresh["sourceText"] != "你好。" {
		t.Fatal(fresh)
	}
	rev = revisions + "/" + fresh["id"].(string)
	source := "  再见。"
	patch := fmt.Sprintf(`{"expectedVersion":1,"sourceText":%q,"speakingRate":0.75}`, source)
	jsonCall("PATCH", rev, "", patch, 200)
	generation := jsonCall("POST", rev+"/audio/generate", readyKey, v2, 202)["generationId"]
	if replay := jsonCall("POST", rev+"/audio/generate", readyKey, v2, 202); replay["generationId"] != generation {
		t.Fatal(replay)
	}
	candidate := waitAudio(rev, "READY")["audioAssetId"].(string)
	if candidate == original {
		t.Fatal("candidate reused published asset")
	}
	for _, path := range []string{"/audio", "/audio/download"} {
		assertAPIError(t, call("GET", rev+path+"?audioAssetId="+original, "", ""), 404, "AUDIO_NOT_FOUND")
		part := call("GET", rev+path+"?audioAssetId="+candidate, "", "", "bytes=0-3")
		if part.Code != 206 || part.Body.Len() != 4 || (path == "/audio/download" && part.Header().Get("Content-Disposition") == "") {
			t.Fatal(part.Code, part.Header())
		}
	}
	assertAPIError(t, call("GET", item+"/audio?audioAssetId="+candidate, "", ""), 404, "AUDIO_NOT_FOUND")
	outdated := jsonCall("PATCH", rev, "", `{"expectedVersion":2,"sourceText":"再见。"}`, 200)
	if outdated["audioStatus"] != "OUTDATED" || outdated["audioAssetId"] != nil {
		t.Fatal(outdated)
	}
	assertAPIError(t, call("GET", rev+"/audio?audioAssetId="+candidate, "", ""), 404, "AUDIO_NOT_FOUND")
	assertAPIError(t, call("POST", rev+"/publish", "", publish2), 409, "VERSION_CONFLICT")
	assertAPIError(t, call("POST", rev+"/publish", "", `{"expectedVersion":3,"expectedItemVersion":2}`), 409, "AUDIO_NOT_READY")
	jsonCall("PATCH", rev, "", fmt.Sprintf(`{"expectedVersion":3,"sourceText":%q}`, source), 200)
	jsonCall("PATCH", item, "", `{"expectedVersion":2,"title":"Retitled","description":"Description","notes":"Kept","difficulty":"advanced"}`, 200)
	jsonCall("PUT", item+"/favorite", "", `{"isFavorite":true}`, 200)
	body := func(itemVersion int, pin string) string {
		return fmt.Sprintf(`{"expectedVersion":4,"expectedItemVersion":%d,"audioAssetId":%q}`, itemVersion, pin)
	}
	assertAPIError(t, call("POST", rev+"/publish", "", body(2, candidate)), 409, "VERSION_CONFLICT")
	assertAPIError(t, call("POST", rev+"/publish", "", body(3, original)), 409, "AUDIO_NOT_READY")
	// Simulate a changed base even with the expected item version.
	for _, base := range []string{candidate, original} {
		if _, err := pool.Exec(t.Context(), "UPDATE library_items SET active_audio_asset_id=$2 WHERE id=$1", itemID, base); err != nil {
			t.Fatal(err)
		}
		if base == candidate {
			assertAPIError(t, call("POST", rev+"/publish", "", body(3, candidate)), 409, "REVISION_BASE_CHANGED")
		}
	}
	var key string
	if err := pool.QueryRow(t.Context(), "SELECT storage_key FROM audio_assets WHERE id=$1", candidate).Scan(&key); err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(root, key)
	if err := os.Rename(file, file+".offline"); err != nil {
		t.Fatal(err)
	}
	assertAPIError(t, call("POST", rev+"/publish", "", body(3, candidate)), 503, "AUDIO_UNAVAILABLE")
	stillOriginal()
	if err := os.Rename(file+".offline", file); err != nil {
		t.Fatal(err)
	}
	expected := call("GET", rev+"/audio?audioAssetId="+candidate, "", "")
	jsonCall("POST", rev+"/publish", "", body(3, candidate), 200)
	view := jsonCall("GET", item, "", "", 200)
	actual := call("GET", item+"/audio?audioAssetId="+candidate, "", "")
	if expected.Code != 200 || actual.Code != 200 || !bytes.Equal(actual.Body.Bytes(), expected.Body.Bytes()) ||
		view["audioAssetId"] != candidate || view["sourceText"] != source || view["speakingRate"] != 0.75 ||
		view["version"] != float64(4) || view["notes"] != "Kept" || view["isFavorite"] != true || view["title"] != "Retitled" ||
		view["description"] != "Description" || view["difficulty"] != "advanced" || view["transcript"] == nil {
		t.Fatalf("incorrect published revision: %v, stream %d", view, actual.Code)
	}
	assertAPIError(t, call("GET", rev, "", ""), 404, "REVISION_NOT_FOUND")
	assertAPIError(t, call("GET", originalURL, "", ""), 404, "AUDIO_NOT_FOUND")

	// Discarding a playable candidate releases its private file. The active
	// recording stays accessible, including when both have identical inputs.
	opened = jsonCall("POST", revisions, "", "", 201)
	rev = revisions + "/" + opened["id"].(string)
	const abandonKey = "aa340000-0000-4000-b000-000000000004"
	jsonCall("POST", rev+"/audio/generate", abandonKey, v1, 202)
	abandoned := waitAudio(rev, "READY")["audioAssetId"].(string)
	if err := pool.QueryRow(t.Context(), "SELECT storage_key FROM audio_assets WHERE id=$1", abandoned).Scan(&key); err != nil {
		t.Fatal(err)
	}
	if res := call("DELETE", rev+"?expectedVersion=1", "", ""); res.Code != 204 {
		t.Fatal(res.Code, res.Body)
	}
	if _, err := os.Stat(filepath.Join(root, key)); !os.IsNotExist(err) {
		t.Fatalf("discarded candidate file was not released: %v", err)
	}
	if res := call("GET", item+"/audio?audioAssetId="+candidate, "", ""); res.Code != 200 || !bytes.Equal(res.Body.Bytes(), expected.Body.Bytes()) {
		t.Fatal("discard changed the active recording")
	}
	// The dump/restore suite compares this saved, unpublished revision too.
	opened = jsonCall("POST", revisions, "", "", 201)
	rev = revisions + "/" + opened["id"].(string)
	jsonCall("PATCH", rev, "", `{"expectedVersion":1,"sourceText":"保存。"}`, 200)
	if saved := jsonCall("POST", revisions, "", "", 200); saved["id"] != opened["id"] || saved["sourceText"] != "保存。" || saved["version"] != float64(2) {
		t.Fatalf("resume lost the saved revision: %v", saved)
	}
}
