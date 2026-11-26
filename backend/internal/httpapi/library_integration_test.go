package httpapi

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/routers"
	"github.com/jackc/pgx/v5/pgxpool"
	"taotale/backend/internal/persistence"
	"taotale/backend/internal/persistence/queries"
	"taotale/backend/internal/tts"
)

func TestLibraryPostgres(t *testing.T) {
	dsn := os.Getenv("TAOTALE_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("run scripts/test-db.sh with disposable PostgreSQL")
	}
	config, err := pgxpool.ParseConfig(dsn)
	if err != nil || config.ConnConfig.Database != "taotale_test" {
		t.Fatal("integration tests require the disposable taotale_test database")
	}
	pool, err := pgxpool.NewWithConfig(t.Context(), config)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	schema, err := persistence.LoadSchema()
	if err != nil {
		t.Fatal(err)
	}
	if err := schema.Prepare(t.Context(), pool); err != nil {
		t.Fatal(err)
	}
	spec, err := openapi3.NewLoader().LoadFromFile("../../api/openapi.yaml")
	if err != nil {
		t.Fatal(err)
	}
	api := Router(nil, slog.New(slog.DiscardHandler), "", queries.New(pool), nil, tts.Mock{}, &persistence.Library{Pool: pool})
	call := func(method, route, endpoint, body string, status int) map[string]any {
		t.Helper()
		req := httptest.NewRequest(method, "http://localhost/api/v1"+endpoint, strings.NewReader(body))
		if body != "" {
			req.Header.Set("Content-Type", "application/json")
		}
		response := httptest.NewRecorder()
		api.ServeHTTP(response, req)
		item := spec.Paths.Find(route)
		operation := item.GetOperation(method)
		requireContractResponse(t, req, &routers.Route{Spec: spec, Path: route, PathItem: item, Operation: operation}, response, status)
		var result map[string]any
		if status != http.StatusNoContent {
			if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
				t.Fatal(err)
			}
		}
		return result
	}
	checkError := func(response map[string]any, expected string) map[string]any {
		t.Helper()
		got, _ := response["error"].(map[string]any)
		if got["code"] != expected {
			t.Fatalf("expected %s, got %v", expected, got)
		}
		return got
	}

	topic := call("POST", "/topics", "/topics", `{"name":"  Straße  ","colorPreset":"sage"}`, 201)
	id := topic["id"].(string)
	if topic["name"] != "Straße" || topic["usageCount"] != float64(0) {
		t.Fatal(topic)
	}
	duplicate := call("POST", "/topics", "/topics", `{"name":"STRASSE"}`, 409)
	if details := checkError(duplicate, "DUPLICATE_TOPIC")["details"].(map[string]any); details["existingId"] != id {
		t.Fatal(details)
	}
	for _, body := range []string{
		`{"name":"bad color","colorPreset":"pink"}`, `{"name":"   "}`, `{"name":null}`,
		`{"name":"custom colors","colors":{"light":"#ffffff"}}`,
	} {
		checkError(call("POST", "/topics", "/topics", body, 400), "VALIDATION_ERROR")
	}

	source := " \t甲𠀀😀。\r\n乙 "
	draftBody, _ := json.Marshal(map[string]any{
		"languageCode": "zh-CN", "sourceText": source, "title": strings.Repeat("😀", 120), "description": "preserve me",
		"notes": "clear me", "difficulty": "intermediate", "isFavorite": true, "topicIds": []string{id},
		"voiceId": "zh-CN-XiaoxiaoNeural", "speakingRate": 1,
	})
	item := call("POST", "/items", "/items", string(draftBody), 201)
	itemID := item["id"].(string)
	itemPath := "/items/" + itemID
	if item["sourceText"] != source || item["version"] != float64(1) || item["isFavorite"] != true ||
		item["status"] != "draft" || item["publishedAt"] != nil || item["audioStatus"] != "NOT_GENERATED" {
		t.Fatal(item)
	}
	korean := call("POST", "/items", "/items", `{"languageCode":"ko-KR","topicIds":["`+id+`"]}`, 201)
	koPath := "/items/" + korean["id"].(string)
	catalog := call("GET", "/topics", "/topics", "", 200)
	var usage any
	for _, value := range catalog["items"].([]any) {
		row := value.(map[string]any)
		if row["id"] == id {
			usage = row["usageCount"]
		}
	}
	if usage != float64(2) {
		t.Fatalf("shared topic usage = %v", usage)
	}

	for _, body := range []string{
		`{"languageCode":"zh-CN","difficulty":"native"}`,
		`{"languageCode":"ja-JP"}`,
		`{"languageCode":"ko-KR","voiceId":"zh-CN-XiaoxiaoNeural"}`,
		`{"languageCode":"zh-CN","sourceText":null}`,
		`{"languageCode":"zh-CN","sourceText":"` + strings.Repeat("😀", 751) + `"}`,
		`{"languageCode":"zh-CN","topicIds":["` + id + `","` + id + `"]}`,
	} {
		checkError(call("POST", "/items", "/items", body, 400), "VALIDATION_ERROR")
	}
	for _, query := range []string{"", "?languageCode=zh-CN&pageSize=51", "?languageCode=ko-KR&page=0"} {
		checkError(call("GET", "/items", "/items"+query, "", 400), "VALIDATION_ERROR")
	}
	patched := call("PATCH", "/items/{itemId}", itemPath, `{"expectedVersion":1,"isFavorite":false,"sourceText":"","title":"","notes":null,"difficulty":null}`, 200)
	if patched["version"] != float64(2) || patched["isFavorite"] != false || patched["sourceText"] != "" ||
		patched["description"] != "preserve me" || patched["title"] != "" || patched["notes"] != nil ||
		patched["difficulty"] != nil || len(patched["topicIds"].([]any)) != 1 {
		t.Fatal(patched)
	}
	conflict := call("PATCH", "/items/{itemId}", itemPath, `{"expectedVersion":1,"sourceText":"stale"}`, 409)
	if details := checkError(conflict, "VERSION_CONFLICT")["details"].(map[string]any); details["currentVersion"] != float64(2) {
		t.Fatal(details)
	}
	for _, body := range []string{
		`{"expectedVersion":2,"languageCode":"ko-KR"}`, `{"expectedVersion":2,"isFavorite":null}`,
		`{"expectedVersion":2,"topicIds":null}`, `{"expectedVersion":0}`,
	} {
		checkError(call("PATCH", "/items/{itemId}", itemPath, body, 400), "VALIDATION_ERROR")
	}
	missingID := "b757aac1-5624-4db8-a1c8-0b973900dc04"
	checkError(call("PATCH", "/items/{itemId}", itemPath, `{"expectedVersion":2,"sourceText":"rollback","topicIds":["`+missingID+`"]}`, 404), "TOPIC_NOT_FOUND")
	after := call("GET", "/items/{itemId}", itemPath, "", 200)
	if after["version"] != float64(2) || after["sourceText"] != "" || len(after["topicIds"].([]any)) != 1 {
		t.Fatal("partial item update persisted", after)
	}

	for range 8 {
		call("POST", "/items", "/items", `{"languageCode":"zh-CN"}`, 201)
	}
	if _, err := pool.Exec(t.Context(), "UPDATE public.library_items SET created_at = '2025-01-01T00:00Z'"); err != nil {
		t.Fatal(err)
	}
	first := call("GET", "/items", "/items?languageCode=zh-CN", "", 200)
	last := call("GET", "/items", "/items?languageCode=zh-CN&page=2", "", 200)
	if len(first["items"].([]any)) != 8 || len(last["items"].([]any)) != 1 || first["total"] != float64(9) {
		t.Fatal("unstable pagination", first, last)
	}
	prev := "z"
	for _, raw := range append(first["items"].([]any), last["items"].([]any)...) {
		row := raw.(map[string]any)
		if current := row["id"].(string); current >= prev || row["languageCode"] != "zh-CN" {
			t.Fatal("bad UUID tie ordering or language", row)
		} else {
			prev = current
		}
	}

	customPath := "/topics/" + id
	recolored := call("PATCH", "/topics/{topicId}", customPath, `{"name":"Reading","colorPreset":"orchid"}`, 200)
	if recolored["usageCount"] != float64(2) || recolored["name"] != "Reading" {
		t.Fatal(recolored)
	}
	defaultPath := "/topics/" + contractID
	defaults := call("PATCH", "/topics/{topicId}", defaultPath, `{"name":"Renamed history","colorPreset":"slate"}`, 200)
	if err := schema.Prepare(t.Context(), pool); err != nil {
		t.Fatal(err)
	}
	postSeed := call("PATCH", "/topics/{topicId}", defaultPath, `{"name":"Renamed history"}`, 200)
	if postSeed["colors"].(map[string]any)["dark"].(map[string]any)["text"] != "#DEDAED" || defaults["systemKey"] != "history" {
		t.Fatal("seed changed a default", defaults, postSeed)
	}
	checkError(call("DELETE", "/topics/{topicId}", defaultPath, "", 409), "DEFAULT_TOPIC_PROTECTED")
	call("DELETE", "/topics/{topicId}", customPath, "", 204)
	checkError(call("DELETE", "/topics/{topicId}", customPath, "", 404), "TOPIC_NOT_FOUND")
	if row := call("GET", "/items/{itemId}", koPath, "", 200); len(row["topicIds"].([]any)) != 0 {
		t.Fatal("deleting topic did not remove its join", row)
	}
	// Leave a live association for the database backup and restore comparison.
	call("PATCH", "/items/{itemId}", koPath, `{"expectedVersion":1,"topicIds":["`+contractID+`"]}`, 200)
	call("DELETE", "/items/{itemId}", itemPath, "", 204)
	checkError(call("GET", "/items/{itemId}", itemPath, "", 404), "ITEM_NOT_FOUND")
	checkError(call("GET", "/items/{itemId}", "/items/"+missingID, "", 404), "ITEM_NOT_FOUND")
}
