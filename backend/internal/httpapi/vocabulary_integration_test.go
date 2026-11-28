package httpapi

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/routers"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"taotale/backend/internal/persistence"
	"taotale/backend/internal/persistence/queries"
	"taotale/backend/internal/tts"
)

func TestVocabularyPostgres(t *testing.T) {
	pool, _ := libraryTestDatabase(t)
	spec, err := openapi3.NewLoader().LoadFromFile("../../api/openapi.yaml")
	if err != nil {
		t.Fatal(err)
	}
	api := Router(nil, slog.New(slog.DiscardHandler), "", queries.New(pool), nil, tts.Mock{}, &persistence.Library{Pool: pool})
	send := func(method, path string, payload any) (*http.Request, *httptest.ResponseRecorder) {
		var body []byte
		if payload != nil {
			body, _ = json.Marshal(payload)
		}
		req := httptest.NewRequest(method, "http://localhost/api/v1"+path, strings.NewReader(string(body)))
		if payload != nil {
			req.Header.Set("Content-Type", "application/json")
		}
		res := httptest.NewRecorder()
		api.ServeHTTP(res, req)
		return req, res
	}
	call := func(method, route, path string, payload any, status int) map[string]any {
		t.Helper()
		req, res := send(method, path, payload)
		item := spec.Paths.Find(route)
		requireContractResponse(t, req, &routers.Route{Spec: spec, Path: route, PathItem: item, Operation: item.GetOperation(method)}, res, status)
		var result map[string]any
		if status != 204 && json.Unmarshal(res.Body.Bytes(), &result) != nil {
			t.Fatal(res.Body.String())
		}
		return result
	}
	const vocabulary = "/vocabulary"
	const entryRoute = vocabulary + "/{vocabularyId}"
	const refRoute = entryRoute + "/references"
	const itemRoute = "/items/{itemId}"
	const zh = "?languageCode=zh-CN"
	const ko = "?languageCode=ko-KR"
	create := func(payload map[string]any) map[string]any {
		return call("POST", vocabulary, vocabulary, payload, 201)
	}
	newItem := func(language string) string {
		return call("POST", "/items", "/items", map[string]any{"languageCode": language, "title": "Vocabulary reference"}, 201)["id"].(string)
	}
	entry := func(id string) map[string]any {
		return call("GET", entryRoute, vocabulary+"/"+id, nil, 200)
	}
	errorCode := func(result map[string]any, code string) {
		t.Helper()
		if result["error"].(map[string]any)["code"] != code {
			t.Fatalf("expected %s: %v", code, result)
		}
	}
	stats := func(query string) map[string]any {
		return call("GET", vocabulary+"/stats", vocabulary+"/stats"+query, nil, 200)
	}
	before := stats(zh)
	chinese, korean, secondItem := newItem("zh-CN"), newItem("ko-KR"), newItem("zh-CN")
	var suffix string
	if err := pool.QueryRow(t.Context(), "SELECT gen_random_uuid()::text").Scan(&suffix); err != nil {
		t.Fatal(err)
	}
	topic := call("POST", "/topics", "/topics", map[string]any{"name": "Vocabulary fixture " + suffix[:8]}, 201)["id"].(string)
	original := "Cafe\u0301\u00a0\t" + suffix
	word := "Café " + suffix
	first := create(map[string]any{"languageCode": "zh-CN", "word": "\u3000" + original + " ",
		"reading": "33%_literal", "readingSystem": "pinyin", "meaning": "Example", "topicIds": []string{topic}, "itemId": chinese})
	zhID := first["id"].(string)
	koID := create(map[string]any{"languageCode": "ko-KR", "word": word, "itemId": korean})["id"].(string)
	other := create(map[string]any{"languageCode": "zh-CN", "word": strings.ToLower(word), "itemId": chinese})["id"].(string)
	if first["word"] != original || first["status"] != "LEARNING" || len(first["references"].([]any)) != 1 {
		t.Fatal(first)
	}
	lookup := vocabulary + "/lookup"
	for query, expected := range map[string]string{zh: zhID, ko: koID} {
		found := call("GET", lookup, lookup+query+"&word="+url.QueryEscape(word), nil, 200)["entry"].(map[string]any)
		if found["id"] != expected {
			t.Fatal(found)
		}
	}
	const absent = "absent"
	if result := call("GET", lookup, lookup+ko+"&word="+absent, nil, 200); result["entry"] != nil {
		t.Fatal(result)
	}
	duplicate := call("POST", vocabulary, vocabulary, map[string]any{"languageCode": "zh-CN", "word": word}, 409)
	errorCode(duplicate, "DUPLICATE_VOCABULARY")
	if details := duplicate["error"].(map[string]any)["details"].(map[string]any); details["existingId"] != zhID {
		t.Fatal(details)
	}
	race := make(chan int, 2)
	for range 2 {
		go func() {
			_, response := send("POST", vocabulary, map[string]any{"languageCode": "ko-KR", "word": "race " + suffix})
			race <- response.Code
		}()
	}
	a, b := <-race, <-race
	if min(a, b) != 201 || max(a, b) != 409 {
		t.Fatalf("concurrent duplicate creation returned %d and %d", a, b)
	}
	for _, payload := range []map[string]any{
		{"languageCode": "ko-KR", "word": strings.Repeat("한", 121)},
		{"languageCode": "ko-KR", "word": word, "reading": strings.Repeat("가", 161)},
		{"languageCode": "ko-KR", "word": word, "readingSystem": strings.Repeat("가", 41)},
		{"languageCode": "ko-KR", "word": word, "meaning": strings.Repeat("가", 301)},
		{"languageCode": "ko-KR", "word": word, "note": strings.Repeat("가", 5001)},
	} {
		errorCode(call("POST", vocabulary, vocabulary, payload, 400), "VALIDATION_ERROR")
	}
	phrase := strings.Repeat("한국어 ", 29) + suffix[:4]
	if created := create(map[string]any{"languageCode": "ko-KR", "word": phrase}); created["word"] != phrase {
		t.Fatal(created)
	}
	ref := vocabulary + "/" + zhID + "/references"
	errorCode(call("POST", refRoute, ref, map[string]any{"itemId": korean}, 400), "VOCABULARY_LANGUAGE_MISMATCH")
	errorCode(call("POST", vocabulary, vocabulary, map[string]any{"languageCode": "zh-CN", "word": "invalid " + suffix, "itemId": korean}, 400), "VOCABULARY_LANGUAGE_MISMATCH")
	if leaked := call("GET", lookup, lookup+zh+"&word="+url.QueryEscape("invalid "+suffix), nil, 200); leaked["entry"] != nil {
		t.Fatal(leaked)
	}
	const insertReference = `INSERT INTO public.vocabulary_references
		(user_id, language_code, vocabulary_entry_id, library_item_id, manual_linked, auto_linked)
		SELECT id, 'zh-CN', $1, $2, $3, $4 FROM public.users WHERE singleton`
	for _, test := range []struct {
		item string
		auto bool
	}{{korean, true}, {secondItem, false}} {
		if _, err := pool.Exec(t.Context(), insertReference, zhID, test.item, false, test.auto); err == nil {
			t.Fatalf("database allowed invalid reference: %+v", test)
		}
	}
	if _, err := pool.Exec(t.Context(), insertReference, zhID, secondItem, false, true); err != nil {
		t.Fatal(err)
	}
	for range 2 {
		linked := call("POST", refRoute, ref, map[string]any{"itemId": secondItem}, 200)
		if linked["manualLinked"] != true || linked["autoLinked"] != true {
			t.Fatal(linked)
		}
	}
	reader := call("GET", itemRoute+"/vocabulary", "/items/"+secondItem+"/vocabulary", nil, 200)["items"].([]any)
	if len(reader) != 1 || reader[0].(map[string]any)["id"] != zhID {
		t.Fatal(reader)
	}
	after := stats(zh)
	if after["total"].(float64)-before["total"].(float64) != 2 ||
		after["referencedTexts"].(float64)-before["referencedTexts"].(float64) != 2 {
		t.Fatalf("unexpected scoped statistics: %v, initially %v", after, before)
	}
	filtered := func(query string) map[string]any { return call("GET", vocabulary, vocabulary+query, nil, 200) }
	for _, q := range []string{suffix[:8], "33%_", "eXaMpLe"} {
		page := filtered(zh + "&q=" + url.QueryEscape(q) + "&topics=" + topic + "&status=LEARNING")
		if page["total"] != float64(1) || page["items"].([]any)[0].(map[string]any)["id"] != zhID {
			t.Fatal(page)
		}
	}
	if page := filtered(ko + "&topics=" + topic); page["total"] != float64(0) {
		t.Fatal(page)
	}
	bulk := func(action string, ids []string, status int) map[string]any {
		input := map[string]any{"languageCode": "zh-CN", "action": action, "ids": ids}
		if action == "setStatus" {
			input["status"] = "ACQUIRED"
		}
		return call("POST", vocabulary+"/bulk", vocabulary+"/bulk", input, status)
	}
	errorCode(bulk("setStatus", []string{zhID, koID}, 404), "VOCABULARY_SELECTION_NOT_FOUND")
	errorCode(bulk("delete", []string{zhID, contractID}, 404), "VOCABULARY_SELECTION_NOT_FOUND")
	errorCode(bulk("delete", []string{zhID, zhID}, 400), "VALIDATION_ERROR")
	if current := entry(zhID); current["status"] != "LEARNING" || len(current["references"].([]any)) != 2 {
		t.Fatal(current)
	}
	if result := bulk("setStatus", []string{zhID, other}, 200); result["affected"] != float64(2) {
		t.Fatal(result)
	}
	if current := entry(zhID); current["status"] != "ACQUIRED" || current["meaning"] != "Example" {
		t.Fatal(current)
	}
	edited := call("PATCH", entryRoute, vocabulary+"/"+zhID, map[string]any{"reading": nil, "note": "Saved", "status": "LEARNING"}, 200)
	if edited["reading"] != nil || edited["note"] != "Saved" || edited["meaning"] != "Example" || edited["status"] != "LEARNING" {
		t.Fatal(edited)
	}
	errorCode(call("PATCH", entryRoute, vocabulary+"/"+zhID, map[string]any{"word": strings.ToLower(word), "meaning": "changed"}, 409), "DUPLICATE_VOCABULARY")
	if current := entry(zhID); current["meaning"] != "Example" {
		t.Fatal(current)
	}
	topicView := call("PATCH", "/topics/{topicId}", "/topics/"+topic, map[string]any{"name": "Vocabulary topics " + suffix[:8]}, 200)
	if topicView["usageCount"] != float64(0) || topicView["vocabularyUsageCount"] != float64(1) {
		t.Fatal(topicView)
	}
	call("DELETE", refRoute+"/{itemId}", ref+"/"+secondItem, nil, 204)
	if rows := call("GET", itemRoute+"/vocabulary", "/items/"+secondItem+"/vocabulary", nil, 200)["items"].([]any); len(rows) != 0 {
		t.Fatal(rows)
	}
	if _, err := pool.Exec(t.Context(), "UPDATE public.vocabulary_entries SET source_item_id=$1 WHERE id=$2", chinese, zhID); err != nil {
		t.Fatal(err)
	}
	call("DELETE", itemRoute, "/items/"+chinese, nil, 204)
	var legacySourceCleared bool
	if err := pool.QueryRow(t.Context(), "SELECT source_item_id IS NULL FROM public.vocabulary_entries WHERE id=$1", zhID).Scan(&legacySourceCleared); err != nil || !legacySourceCleared {
		t.Fatalf("item deletion did not preserve the word and clear its legacy pointer: %v", err)
	}
	if current := entry(zhID); len(current["references"].([]any)) != 0 {
		t.Fatal(current)
	}
	call("DELETE", "/topics/{topicId}", "/topics/"+topic, nil, 204)
	if current := entry(zhID); len(current["topicIds"].([]any)) != 0 {
		t.Fatal(current)
	}
	if result := bulk("delete", []string{zhID, other}, 200); result["affected"] != float64(2) {
		t.Fatal(result)
	}
	errorCode(call("GET", entryRoute, vocabulary+"/"+zhID, nil, 404), "VOCABULARY_NOT_FOUND")
	call("DELETE", entryRoute, vocabulary+"/"+koID, nil, 204)
	call("GET", itemRoute, "/items/"+korean, nil, 200)
}

type vocabularyQueryTrace struct {
	queries []pgx.TraceQueryStartData
}

func (trace *vocabularyQueryTrace) TraceQueryStart(ctx context.Context, _ *pgx.Conn, data pgx.TraceQueryStartData) context.Context {
	if strings.HasPrefix(data.SQL, "-- name: ") {
		trace.queries = append(trace.queries, data)
	}
	return ctx
}

func (*vocabularyQueryTrace) TraceQueryEnd(context.Context, *pgx.Conn, pgx.TraceQueryEndData) {}

func TestVocabularyScalePostgres(t *testing.T) {
	pool, _ := libraryTestDatabase(t)
	ctx := t.Context()
	var fixture string
	if err := pool.QueryRow(ctx, "SELECT gen_random_uuid()::text").Scan(&fixture); err != nil {
		t.Fatal(err)
	}
	for _, sql := range []string{
		`INSERT INTO public.library_items (user_id, language_code, title, published_at)
		SELECT u.id, CASE WHEN n % 2 = 0 THEN 'zh-CN' ELSE 'ko-KR' END, $1,
			CASE WHEN n <= 1000 THEN now() END
		FROM public.users u CROSS JOIN generate_series(1, 1100) n WHERE u.singleton`,
		`INSERT INTO public.topics (id, user_id, name, name_key, light_background, light_text, dark_background, dark_text)
		SELECT gen_random_uuid(), u.id, $1 || n, $1 || n, '#ECE7FA', '#554095', '#3E345C', '#E4D8FF'
		FROM public.users u CROSS JOIN generate_series(1, 100) n WHERE u.singleton`,
		`INSERT INTO public.vocabulary_entries (user_id, language_code, word, normalized_word_key, status)
		SELECT u.id, CASE WHEN n % 2 = 0 THEN 'zh-CN' ELSE 'ko-KR' END,
			$1 || ' ' || lpad(n::text, 5, '0'), $1 || ' ' || lpad(n::text, 5, '0'),
			CASE WHEN n % 4 < 2 THEN 'LEARNING' ELSE 'ACQUIRED' END
		FROM public.users u CROSS JOIN generate_series(1, 10000) n WHERE u.singleton`,
		`INSERT INTO public.vocabulary_topics (user_id, vocabulary_entry_id, topic_id)
		SELECT v.user_id, v.id, t.id
		FROM public.vocabulary_entries v JOIN public.topics t
			ON t.name_key = $1 || (right(v.word, 5)::integer % 100 + 1)
		WHERE starts_with(v.word, $1)`,
		`WITH items AS (
			SELECT id, language_code, row_number() OVER (PARTITION BY language_code ORDER BY id) AS n
			FROM public.library_items WHERE title = $1
		)
		INSERT INTO public.vocabulary_references (user_id, language_code, vocabulary_entry_id, library_item_id, manual_linked)
		SELECT v.user_id, v.language_code, v.id, i.id, true
		FROM public.vocabulary_entries v JOIN items i ON i.language_code = v.language_code
			AND i.n = right(v.word, 5)::integer % 550 + 1
		WHERE starts_with(v.word, $1)`,
	} {
		if _, err := pool.Exec(ctx, sql, fixture); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := pool.Exec(ctx, "ANALYZE public.vocabulary_entries, public.vocabulary_references, public.vocabulary_topics"); err != nil {
		t.Fatal(err)
	}
	var topic string
	if err := pool.QueryRow(ctx, "SELECT id::text FROM public.topics WHERE name_key = $1", fixture+"1").Scan(&topic); err != nil {
		t.Fatal(err)
	}
	trace := &vocabularyQueryTrace{}
	config := pool.Config()
	config.ConnConfig.Tracer = trace
	traced, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		t.Fatal(err)
	}
	defer traced.Close()
	api := Router(nil, slog.New(slog.DiscardHandler), "", queries.New(traced), nil, tts.Mock{}, &persistence.Library{Pool: traced})
	get := func(path string, target any) {
		t.Helper()
		res := httptest.NewRecorder()
		api.ServeHTTP(res, httptest.NewRequest("GET", "http://localhost/api/v1"+path, nil))
		if res.Code != 200 || json.Unmarshal(res.Body.Bytes(), target) != nil {
			t.Fatalf("%s: %d %s", path, res.Code, res.Body.String())
		}
	}
	filter := "/vocabulary?languageCode=zh-CN&status=LEARNING&sort=word_asc&topics=" + topic + "&q=" + fixture
	for page, firstWord := range map[int]string{1: "00100", 2: "01100", 11: ""} {
		trace.queries = nil
		var result struct {
			Items []struct {
				Word     string
				TopicIDs []string
			}
			Total int
		}
		get(fmt.Sprintf("%s&page=%d", filter, page), &result)
		if result.Total != 100 {
			t.Fatalf("filtered total %d on page %d", result.Total, page)
		}
		if firstWord == "" {
			if len(result.Items) != 0 || len(trace.queries) != 2 {
				t.Fatal("empty page should only run count and page queries")
			}
			continue
		}
		if len(result.Items) != 10 || len(trace.queries) != 3 {
			t.Fatalf("expected ten rows in three queries, got %d rows in %d queries", len(result.Items), len(trace.queries))
		}
		if result.Items[0].Word != fixture+" "+firstWord || len(result.Items[0].TopicIDs) != 1 || result.Items[0].TopicIDs[0] != topic {
			t.Fatalf("unexpected first row: %+v", result.Items[0])
		}
		if page == 1 {
			logVocabularyPlans(t, pool, trace.queries)
		}
	}
	trace.queries = nil
	var stats struct {
		Total, Learning, Acquired, ReferencedTexts int
	}
	get("/vocabulary/stats?languageCode=zh-CN", &stats)
	if stats.Total < 5000 || stats.Learning < 2500 || stats.Acquired < 2500 ||
		stats.ReferencedTexts < 275 || len(trace.queries) != 1 {
		t.Fatalf("unexpected scale stats: %+v in %d queries", stats, len(trace.queries))
	}
	logVocabularyPlans(t, pool, trace.queries)
}

func logVocabularyPlans(t *testing.T, pool *pgxpool.Pool, queries []pgx.TraceQueryStartData) {
	t.Helper()
	for _, query := range queries {
		rows, err := pool.Query(t.Context(), "EXPLAIN (ANALYZE, BUFFERS) "+query.SQL, query.Args...)
		if err != nil {
			t.Fatal(err)
		}
		plan, err := pgx.CollectRows(rows, pgx.RowTo[string])
		if err != nil {
			t.Fatal(err)
		}
		name, _, _ := strings.Cut(query.SQL, "\n")
		t.Logf("%s\n%s", name, strings.Join(plan, "\n"))
	}
}
