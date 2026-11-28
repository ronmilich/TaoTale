package httpapi

import (
	"crypto/rand"
	"encoding/json"
	"net/http"
	"testing"
	"time"
)

func TestVocabularyMatchingPostgres(t *testing.T) {
	pool, _, _, call := startAudioTest(t, &previewProvider{})
	key := func() string {
		t.Helper()
		var id string
		if err := pool.QueryRow(t.Context(), "SELECT gen_random_uuid()::text").Scan(&id); err != nil {
			t.Fatal(err)
		}
		return id
	}
	send := func(method, path, idempotency string, body map[string]any, status int) map[string]any {
		t.Helper()
		var payload []byte
		if body != nil {
			payload, _ = json.Marshal(body)
		}
		return audioResult(t, call(method, path, idempotency, string(payload)), status)
	}
	wait := func(path, field string) map[string]any {
		t.Helper()
		deadline := time.Now().Add(12 * time.Second)
		for {
			row := send(http.MethodGet, path, "", nil, 200)
			if row[field] == "READY" {
				return row
			}
			if row[field] == "FAILED" || time.Now().After(deadline) {
				t.Fatalf("waiting for READY on %s: %v", path, row)
			}
			time.Sleep(25 * time.Millisecond)
		}
	}
	const vocabulary = "/vocabulary"
	token := rand.Text()[:6]
	first := token + "𠀀学"
	long := first + "语言"
	replacement := token + "重读"
	word := func(language, text string) string {
		t.Helper()
		return send("POST", vocabulary, "", map[string]any{
			"languageCode": language, "word": text, "meaning": "Manual meaning", "status": "ACQUIRED",
		}, 201)["id"].(string)
	}
	shortID, longID, newID := word("zh-CN", first), word("zh-CN", long), word("zh-CN", replacement)
	koID := word("ko-KR", first)
	word("zh-CN", "𠀀 学"+token)
	newDraft := func(language, source string) string {
		t.Helper()
		return "/items/" + send("POST", "/items", "", map[string]any{
			"languageCode": language, "title": "Vocabulary sync fixture", "sourceText": source,
		}, 201)["id"].(string)
	}
	original := "前言。" + long + "。"
	path := newDraft("zh-CN", original)
	itemID := path[len("/items/"):]
	syncURL := path + "/vocabulary/sync"
	projection := func(item string) (map[string]map[string]any, map[string]any) {
		t.Helper()
		body := send("GET", item+"/vocabulary", "", nil, 200)
		entries := map[string]map[string]any{}
		for _, row := range body["items"].([]any) {
			entry := row.(map[string]any)
			entries[entry["id"].(string)] = entry
		}
		return entries, body
	}
	ref := func(rows map[string]map[string]any, id string, manual, automatic bool) map[string]any {
		t.Helper()
		reference, ok := rows[id]["reference"].(map[string]any)
		if !ok || reference["manualLinked"] != manual || reference["autoLinked"] != automatic {
			t.Fatalf("unexpected provenance for %s: %+v", id, rows[id])
		}
		return reference
	}
	assertAPIError(t, call("POST", syncURL, "", ""), 409, "INVALID_ITEM_STATE")
	send("POST", vocabulary+"/"+shortID+"/references", "", map[string]any{"itemId": itemID}, 200)
	send("POST", path+"/audio/generate", key(), map[string]any{"expectedVersion": 1}, 202)
	audio := wait(path, "audioStatus")["audioAssetId"].(string)

	// A reconciliation failure rolls back publication. The prepared recording can be reused.
	if _, err := pool.Exec(t.Context(), `
		CREATE FUNCTION public.reject_test_automatic_reference() RETURNS trigger LANGUAGE plpgsql AS $$
		BEGIN RAISE EXCEPTION 'test reconciliation failure'; END;
		$$;
		CREATE TRIGGER reject_test_automatic_reference BEFORE INSERT ON public.vocabulary_references
		FOR EACH ROW WHEN (NEW.auto_linked) EXECUTE FUNCTION public.reject_test_automatic_reference()
	`); err != nil {
		t.Fatal(err)
	}
	failed := call("POST", path+"/publish", "", `{"expectedVersion":1}`)
	if _, err := pool.Exec(t.Context(), "DROP TRIGGER reject_test_automatic_reference ON public.vocabulary_references; DROP FUNCTION public.reject_test_automatic_reference()"); err != nil {
		t.Fatal(err)
	}
	assertAPIError(t, failed, 503, "SERVICE_UNAVAILABLE")
	if draft := send("GET", path, "", nil, 200); draft["status"] != "draft" || draft["version"] != float64(1) {
		t.Fatal(draft)
	}
	entries, _ := projection(path)
	ref(entries, shortID, true, false)
	if len(entries) != 1 {
		t.Fatal(entries)
	}
	send("POST", path+"/publish", "", map[string]any{"expectedVersion": 1}, 200)
	published := send("GET", path, "", nil, 200)
	if published["automaticVocabularyMatching"] != true || published["audioAssetId"] != audio {
		t.Fatal(published)
	}
	entries, body := projection(path)
	dual := ref(entries, shortID, true, true)
	ref(entries, longID, false, true)
	if len(entries) != 2 || body["sourceText"] != original {
		t.Fatal(body)
	}
	ranges := body["annotations"].([]any)
	if len(ranges) != 1 {
		t.Fatal(ranges)
	}
	match := ranges[0].(map[string]any)
	if match["vocabularyId"] != longID || match["start"] != float64(3) || match["end"] != float64(14) {
		t.Fatal(match)
	}
	if response := send("POST", syncURL, "", nil, 200); response["matched"] != float64(2) {
		t.Fatal(response)
	}
	entries, _ = projection(path)
	if ref(entries, shortID, true, true)["addedAt"] != dual["addedAt"] ||
		entries[shortID]["status"] != "ACQUIRED" || entries[shortID]["meaning"] != "Manual meaning" {
		t.Fatal(entries)
	}
	if current := send("GET", path, "", nil, 200); current["version"] != published["version"] || current["audioAssetId"] != audio {
		t.Fatal(current)
	}
	lateID := word("zh-CN", token+"𠀀")
	if response := send("POST", syncURL, "", nil, 200); response["matched"] != float64(3) {
		t.Fatal(response)
	}
	entries, _ = projection(path)
	ref(entries, lateID, false, true)
	send("PATCH", vocabulary+"/"+lateID, "", map[string]any{"word": token + "未出现"}, 200)
	entries, _ = projection(path)
	if len(entries) != 2 || entries[lateID] != nil {
		t.Fatal("obsolete automatic spelling was shown in the Reader", entries)
	}
	rev := send("POST", path+"/revisions", "", nil, 201)
	revURL := path + "/revisions/" + rev["id"].(string)
	send("PATCH", revURL, "", map[string]any{"expectedVersion": 1, "sourceText": replacement + "。"}, 200)
	send("POST", revURL+"/audio/generate", key(), map[string]any{"expectedVersion": 2}, 202)
	candidate := wait(revURL, "audioStatus")
	send("POST", revURL+"/publish", "", map[string]any{
		"expectedVersion": 2, "expectedItemVersion": published["version"], "audioAssetId": candidate["audioAssetId"],
	}, 200)
	entries, body = projection(path)
	ref(entries, shortID, true, false)
	ref(entries, newID, false, true)
	if len(entries) != 2 || body["sourceText"] != replacement+"。" {
		t.Fatal(body)
	}
	if late := send("GET", vocabulary+"/"+lateID, "", nil, 200); len(late["references"].([]any)) != 0 {
		t.Fatal("revision publication retained an obsolete automatic reference", late)
	}
	send("PATCH", vocabulary+"/"+newID, "", map[string]any{"word": token + "已改名"}, 200)
	if response := send("POST", syncURL, "", nil, 200); response["matched"] != float64(0) {
		t.Fatal(response)
	}
	entries, _ = projection(path)
	ref(entries, shortID, true, false)
	if len(entries) != 1 {
		t.Fatal(entries)
	}

	// The direct-preview flow uses the same transactional reconciliation.
	preview := send("POST", "/tts/previews", key(), map[string]any{
		"languageCode": "zh-CN", "sourceText": original, "voiceId": "zh-CN-mock", "speakingRate": 1,
	}, 202)["generationId"].(string)
	wait("/tts/previews/"+preview, "status")
	direct := send("POST", "/items/publish", key(), map[string]any{
		"languageCode": "zh-CN", "sourceText": original, "title": "Direct vocabulary sync",
		"previewGenerationId": preview,
	}, 201)
	entries, _ = projection("/items/" + direct["itemId"].(string))
	ref(entries, longID, false, true)

	korean := newDraft("ko-KR", "안녕. "+long+".")
	send("POST", korean+"/audio/generate", key(), map[string]any{"expectedVersion": 1}, 202)
	wait(korean, "audioStatus")
	send("POST", korean+"/publish", "", map[string]any{"expectedVersion": 1}, 200)
	assertAPIError(t, call("POST", korean+"/vocabulary/sync", "", ""), 422, "VOCABULARY_MATCHING_UNSUPPORTED")
	entries, body = projection(korean)
	if len(entries) != 0 || body["annotations"] != nil {
		t.Fatal(body)
	}
	send("POST", vocabulary+"/"+koID+"/references", "", map[string]any{"itemId": korean[len("/items/"):]}, 200)
	entries, body = projection(korean)
	ref(entries, koID, true, false)
	if body["annotations"] != nil {
		t.Fatal(body)
	}
}
