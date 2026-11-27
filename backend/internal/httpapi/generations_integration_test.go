package httpapi

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/routers"
	"taotale/backend/internal/fingerprint"
	"taotale/backend/internal/persistence"
	"taotale/backend/internal/tts"
)

type admissionVoices struct {
	tts.Mock
	lookups atomic.Int32
	offline atomic.Bool
}

func (v *admissionVoices) ListVoices(ctx context.Context, language string) ([]tts.Voice, error) {
	v.lookups.Add(1)
	if v.offline.Load() {
		return nil, &tts.Error{Code: tts.Unavailable}
	}
	return v.Mock.ListVoices(ctx, language)
}

func (*admissionVoices) Synthesize(context.Context, tts.Request) (tts.Result, error) {
	panic("admission attempted synthesis")
}

const draftRoute = "/items/{itemId}"
const admitRoute = draftRoute + "/audio/generate"
const statusRoute = draftRoute + "/audio/generations/{generationId}"

type admissionResponse struct {
	req      *http.Request
	res      *httptest.ResponseRecorder
	template string
	target   string
}

func TestGenerationPostgres(t *testing.T) {
	pool, _ := libraryTestDatabase(t)
	spec, err := openapi3.NewLoader().LoadFromFile("../../api/openapi.yaml")
	if err != nil {
		t.Fatal(err)
	}
	voices := &admissionVoices{}
	mediaRoot := t.TempDir()
	api := Router(nil, slog.New(slog.DiscardHandler), "", nil, nil, voices, &persistence.Library{Pool: pool, MediaRoot: mediaRoot})
	request := func(method, template, item, job, key, body string) admissionResponse {
		path := strings.NewReplacer("{itemId}", item, "{generationId}", job).Replace(template)
		req := httptest.NewRequest(method, "http://localhost/api/v1"+path, strings.NewReader(body))
		if key != "" {
			req.Header.Set("Idempotency-Key", key)
		}
		if body != "" {
			req.Header.Set("Content-Type", "application/json")
		}
		res := httptest.NewRecorder()
		api.ServeHTTP(res, req)
		return admissionResponse{req, res, strings.Split(template, "?")[0], item}
	}
	expect := func(r admissionResponse, code int, errorCode string) map[string]any {
		t.Helper()
		route := spec.Paths.Find(r.template)
		requireContractResponse(t, r.req, &routers.Route{Spec: spec, Path: r.template, PathItem: route, Operation: route.GetOperation(r.req.Method)}, r.res, code)
		if errorCode != "" {
			assertAPIError(t, r.res, code, errorCode)
		}
		var dto map[string]any
		if code != 204 {
			if err := json.Unmarshal(r.res.Body.Bytes(), &dto); err != nil {
				t.Fatal(err)
			}
		}
		return dto
	}
	exec := func(sql string, args ...any) {
		t.Helper()
		if _, err := pool.Exec(t.Context(), sql, args...); err != nil {
			t.Fatal(err)
		}
	}
	key := func(i int) string { return fmt.Sprintf("aadd0000-0000-4000-b000-%012x", i) }
	create := func(body string) string {
		t.Helper()
		return expect(request("POST", "/items", "", "", "", body), 201, "")["id"].(string)
	}
	admit := func(item, k string) admissionResponse {
		return request("POST", admitRoute, item, "", k, "")
	}
	status := func(item, job, want string) map[string]any {
		t.Helper()
		dto := expect(request("GET", statusRoute, item, job, "", ""), 200, "")
		if dto["status"] != want || (want != "READY" && dto["audioAssetId"] != nil) {
			t.Fatal(dto)
		}
		return dto
	}
	source := " 한글 𠀀.\r\n두 번째! "
	body, _ := json.Marshal(map[string]any{"sourceText": source, "languageCode": "ko-KR", "speakingRate": 0.75, "voiceId": "ko-KR-mock"})
	a := create(string(body))
	b := create(`{"languageCode":"zh-CN","sourceText":"甲。"}`)
	c := create(`{"languageCode":"ko-KR","sourceText":"다음."}`)
	empty := create(`{"languageCode":"ko-KR"}`)
	expect(admit(a, ""), 400, "VALIDATION_ERROR")
	expect(admit(a, "bad-key"), 400, "VALIDATION_ERROR")
	expect(request("POST", admitRoute, a, "", key(1), `{"sourceText":"override"}`), 400, "VALIDATION_ERROR")
	expect(request("POST", admitRoute, a, "", key(1), `{"expectedVersion":null}`), 400, "VALIDATION_ERROR")
	expect(request("POST", admitRoute, a, "", key(1), `{"expectedVersion":2}`), 409, "VERSION_CONFLICT")
	expect(admit(empty, key(1)), 409, "INVALID_ITEM_STATE")
	voices.offline.Store(true)
	expect(admit(a, key(1)), 503, "TTS_UNAVAILABLE")
	voices.offline.Store(false)
	expect(admit(key(999), key(1)), 404, "ITEM_NOT_FOUND")

	// Race independent HTTP requests and connections against one durable key.
	replies := make(chan admissionResponse, 16)
	start := make(chan struct{})
	for range cap(replies) {
		go func() { <-start; replies <- admit(a, key(1)) }()
	}
	close(start)
	var id string
	for range cap(replies) {
		next := expect(<-replies, 202, "")["generationId"].(string)
		if id != "" && id != next {
			t.Fatal("duplicate admission", id, next)
		}
		id = next
	}
	var jobs, receipts int
	if err := pool.QueryRow(t.Context(), "SELECT (SELECT count(*) FROM audio_generations), (SELECT count(*) FROM idempotency_records)").Scan(&jobs, &receipts); err != nil || jobs != 1 || receipts != 1 {
		t.Fatalf("concurrent admissions left %d jobs and %d receipts, error %v", jobs, receipts, err)
	}
	expected := fingerprint.Identity{LanguageCode: "ko-KR", ProviderID: "mock", VoiceID: "ko-KR-mock", SpeakingRate: 0.75,
		StrategyID: "ko-kr-phrase", StrategyVersion: "1.0.0", ExactSourceText: source}
	sum, err := expected.Sum()
	if err != nil || status(a, id, "PENDING")["sourceFingerprint"] != sum {
		t.Fatal("snapshot fingerprint mismatch")
	}
	var snapshot string
	if err := pool.QueryRow(t.Context(), "SELECT source_text FROM audio_generations WHERE id=$1", id).Scan(&snapshot); err != nil || source != snapshot {
		t.Fatal("source snapshot", snapshot, err)
	}
	if _, err := pool.Exec(t.Context(), "UPDATE audio_generations SET source_text='changed' WHERE id=$1", id); err == nil {
		t.Fatal("mutable generation snapshot")
	}
	expect(admit(b, key(1)), 409, "IDEMPOTENCY_KEY_REUSED")
	expect(admit(a, key(2)), 409, "GENERATION_IN_PROGRESS")
	expect(request("GET", statusRoute, b, id, "", ""), 404, "GENERATION_NOT_FOUND")
	if got := expect(request("PATCH", draftRoute, a, "", "", `{"expectedVersion":1,"title":"test"}`), 200, "")["audioStatus"]; got != "PENDING" {
		t.Fatal(got)
	}
	if got := expect(request("PATCH", draftRoute, a, "", "", `{"expectedVersion":2,"speakingRate":1}`), 200, "")["audioStatus"]; got != "OUTDATED" {
		t.Fatal(got)
	}
	if got := status(a, id, "OUTDATED")["jobStatus"]; got != "PENDING" {
		t.Fatal("edit hid the active job", got)
	}
	page := expect(request("GET", "/items?languageCode=ko-KR&pageSize=50", "", "", "", ""), 200, "")
	found := false
	for _, raw := range page["items"].([]any) {
		row := raw.(map[string]any)
		if row["id"] == a {
			found = row["audioStatus"] == "OUTDATED"
		}
	}
	if !found {
		t.Fatal("outdated candidate missing from library page")
	}

	// With one slot left, only one of two different targets may be admitted.
	before := voices.lookups.Load()
	go func() { replies <- admit(b, key(2)) }()
	go func() { replies <- admit(c, key(3)) }()
	winners := 0
	var winningRequest admissionResponse
	var winningID string
	for range 2 {
		r := <-replies
		if r.res.Code == 202 {
			winningRequest = r
			winningID = expect(r, 202, "")["generationId"].(string)
			winners++
		} else {
			expect(r, 429, "TTS_RATE_LIMITED")
		}
	}
	if winners != 1 || voices.lookups.Load() != before+1 {
		t.Fatal("quota did not precede provider lookup")
	}
	voices.offline.Store(true)
	if got := expect(request("POST", admitRoute, a, "", key(1), "{}"), 202, "")["generationId"]; got != id {
		t.Fatal(got)
	}
	voices.offline.Store(false)
	exec("UPDATE idempotency_records SET expires_at = now() - interval '1 second' WHERE key=$1", key(1))
	expect(admit(a, key(1)), 409, "IDEMPOTENCY_KEY_EXPIRED")
	expect(admit(b, key(1)), 409, "IDEMPOTENCY_KEY_REUSED")
	exec("UPDATE audio_generations SET status='FAILED',error_code='GENERATION_INTERRUPTED'")
	expect(request("PATCH", draftRoute, a, "", "", `{"expectedVersion":3,"speakingRate":0.75}`), 200, "")
	if got := status(a, id, "FAILED")["errorCode"]; got != "GENERATION_INTERRUPTED" {
		t.Fatal(got)
	}
	expect(admit(a, key(1)), 409, "IDEMPOTENCY_KEY_EXPIRED")
	expect(request("PATCH", draftRoute, a, "", "", `{"expectedVersion":4,"sourceText":"가."}`), 200, "")
	newID := expect(request("POST", admitRoute, a, "", key(4), `{"expectedVersion":5}`), 202, "")["generationId"].(string)
	if newID == id {
		t.Fatal("explicit new attempt reused a job")
	}
	expect(request("PATCH", draftRoute, a, "", "", `{"expectedVersion":5,"title":"Newer title"}`), 200, "")
	if got := expect(request("POST", admitRoute, a, "", key(4), `{"expectedVersion":5}`), 202, "")["generationId"]; got != newID {
		t.Fatal("versioned replay changed the original receipt", got)
	}
	expect(request("POST", admitRoute, a, "", key(4), `{"expectedVersion":6}`), 409, "IDEMPOTENCY_KEY_REUSED")
	if got := status(a, id, "OUTDATED")["jobStatus"]; got != "FAILED" {
		t.Fatal("superseded job lost its outcome", got)
	}
	exec("UPDATE audio_generations SET status='PROCESSING' WHERE id=$1", newID)
	status(a, newID, "PROCESSING")
	exec("UPDATE audio_generations SET status='READY' WHERE id=$1", newID)
	if got := status(a, newID, "FAILED")["errorCode"]; got != "AUDIO_UNAVAILABLE" {
		t.Fatal(got)
	}
	// A persisted, validated asset must still have its file of the recorded size.
	audio, err := (tts.Mock{}).Synthesize(t.Context(), tts.Request{Language: "ko-KR", VoiceID: "ko-KR-mock", Rate: 0.75, Source: "가."})
	if err != nil {
		t.Fatal(err)
	}
	exec(`WITH asset AS (
		INSERT INTO audio_assets(user_id,language_code,provider_id,voice_id,speaking_rate,strategy_id,strategy_version,
			fingerprint_format_version,source_fingerprint,storage_key,byte_size,duration_ms,segments)
		SELECT user_id,language_code,provider_id,voice_id,speaking_rate,strategy_id,strategy_version,
			fingerprint_format_version,source_fingerprint,'candidate.mp3',$2,1000,
			'[{"index":0,"text":"가.","textStart":0,"textEnd":2,"startMs":0,"endMs":1000}]'::jsonb
		FROM audio_generations WHERE id=$1 RETURNING id)
		UPDATE audio_generations SET audio_asset_id=(SELECT id FROM asset) WHERE id=$1`, newID, len(audio.MP3))
	status(a, newID, "FAILED")
	if err := os.WriteFile(filepath.Join(mediaRoot, "candidate.mp3"), audio.MP3, 0600); err != nil {
		t.Fatal(err)
	}
	if got := status(a, newID, "READY")["audioAssetId"]; got == nil {
		t.Fatal("ready without asset ID")
	}
	exec("UPDATE library_items SET published_at=now() WHERE id=$1", a)
	expect(admit(a, key(5)), 409, "INVALID_ITEM_STATE")
	// All three prior admissions count, including the deleted target.
	expect(request("DELETE", draftRoute, b, "", "", ""), 204, "")
	expect(request("DELETE", draftRoute, c, "", "", ""), 204, "")
	if got := expect(admit(winningRequest.target, winningRequest.req.Header.Get("Idempotency-Key")), 202, "")["generationId"]; got != winningID {
		t.Fatal("lost the receipt for a deleted draft", got)
	}
	expect(request("PATCH", draftRoute, empty, "", "", `{"expectedVersion":1,"sourceText":"다시."}`), 200, "")
	for i := 4; i <= 60; i++ {
		job := expect(admit(empty, key(i+20)), 202, "")["generationId"]
		exec("UPDATE audio_generations SET status='FAILED',error_code='TTS_PROVIDER_ERROR' WHERE id=$1", job)
	}
	before = voices.lookups.Load()
	expect(admit(empty, key(999)), 429, "TTS_RATE_LIMITED")
	if voices.lookups.Load() != before {
		t.Fatal("hourly rejection consulted provider")
	}
	exec("UPDATE idempotency_records SET created_at=now()-interval '1 hour 1 second' WHERE key=$1", key(1))
	expect(admit(empty, key(999)), 202, "")
}
