package httpapi

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"path"
	"strings"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/getkin/kin-openapi/routers"
	"github.com/jackc/pgx/v5/pgtype"
	"taotale/backend/internal/persistence/queries"
	"taotale/backend/internal/tts"
)

func TestOpenAPIConformance(t *testing.T) {
	spec, err := openapi3.NewLoader().LoadFromFile("../../api/openapi.yaml")
	if err != nil {
		t.Fatal(err)
	}
	if err := spec.Validate(t.Context()); err != nil {
		t.Fatal(err)
	}
	logger := slog.New(slog.DiscardHandler)
	catalog := contractCatalog(t)
	healthy := Router(func(context.Context) error { return nil }, logger, "", catalog, nil, tts.Mock{}, nil)
	storageError := errors.New("private catalog error")
	unavailable := Router(func(context.Context) error { return storageError }, logger, "", fixtureCatalog{err: storageError}, nil, tts.NewAzure("", "", "", 0), nil)

	operations := make(map[string]*routers.Route)
	for template, item := range spec.Paths.Map() {
		for method, operation := range item.Operations() {
			servers := spec.Servers
			if len(item.Servers) > 0 {
				servers = item.Servers
			}
			if operation.Servers != nil {
				servers = *operation.Servers
			}
			for _, server := range servers {
				key := method + " " + strings.NewReplacer("{itemId}", ":itemId", "{revisionId}", ":revisionId", "{topicId}", ":topicId", "{generationId}", ":generationId", "{imageId}", ":imageId", "{vocabularyId}", ":vocabularyId").Replace(path.Join(server.URL, template))
				if operations[key] != nil {
					t.Fatalf("duplicate contract route %s", key)
				}
				operations[key] = &routers.Route{Spec: spec, Server: server, Path: template, PathItem: item, Method: method, Operation: operation}
			}
		}
	}
	for _, route := range healthy.Routes() {
		key := route.Method + " " + route.Path
		contractRoute := operations[key]
		if contractRoute == nil {
			t.Errorf("Gin route %s is missing from OpenAPI", key)
			continue
		}
		delete(operations, key)
		t.Run(key, func(t *testing.T) {
			url := "http://127.0.0.1" + strings.NewReplacer(":itemId", contractID, ":revisionId", contractID, ":topicId", contractID, ":generationId", contractID, ":imageId", contractID, ":vocabularyId", contractID).Replace(route.Path)
			operation := contractRoute.Operation.OperationID
			var body string
			status := http.StatusOK
			switch operation {
			case "listTTSVoices":
				url += "?languageCode=ko-KR"
			case "listItems", "listVocabulary", "getVocabularyStats":
				url += "?languageCode=ko-KR"
				status = http.StatusServiceUnavailable
			case "lookupVocabulary":
				url += "?languageCode=ko-KR&word=example"
				status = http.StatusServiceUnavailable
			case "createVocabulary":
				body, status = `{"languageCode":"ko-KR","word":"example"}`, http.StatusServiceUnavailable
			case "patchVocabulary":
				body, status = `{"meaning":"Example"}`, http.StatusServiceUnavailable
			case "linkVocabularyReference":
				body, status = `{"itemId":"`+contractID+`"}`, http.StatusServiceUnavailable
			case "bulkVocabulary":
				body, status = `{"languageCode":"ko-KR","action":"delete","ids":["`+contractID+`"]}`, http.StatusServiceUnavailable
			case "getItem", "deleteItem", "deleteTopic", "generateDraftAudio", "getDraftGeneration",
				"getTTSPreview", "streamTTSPreview", "downloadTTSPreview", "getCoverImage", "openRevision", "getRevision",
				"getVocabulary", "deleteVocabulary", "getItemVocabulary", "unlinkVocabularyReference":
				status = http.StatusServiceUnavailable
			case "uploadCoverImage":
				body, status = "image bytes", http.StatusServiceUnavailable
			case "streamItemAudio", "downloadItemAudio", "streamRevisionAudio", "downloadRevisionAudio":
				url += "?audioAssetId=" + contractID
				status = http.StatusServiceUnavailable
			case "createTTSPreview":
				body, status = `{"languageCode":"zh-CN","sourceText":"你好。","voiceId":"zh-CN-mock","speakingRate":1}`, http.StatusServiceUnavailable
			case "createItem":
				body, status = `{"languageCode":"zh-CN"}`, http.StatusServiceUnavailable
			case "publishPreview":
				body, status = `{"languageCode":"zh-CN","title":"Title","sourceText":"你好。","previewGenerationId":"`+contractID+`"}`, http.StatusServiceUnavailable
			case "discardRevision":
				url += "?expectedVersion=1"
				status = http.StatusServiceUnavailable
			case "publishRevision":
				body, status = `{"expectedVersion":1,"expectedItemVersion":1}`, http.StatusServiceUnavailable
			case "publishDraft", "patchRevision", "generateRevisionAudio":
				body, status = `{"expectedVersion":1}`, http.StatusServiceUnavailable
			case "patchItem":
				body, status = `{"expectedVersion":1,"notes":"Example"}`, http.StatusServiceUnavailable
			case "setItemFavorite":
				body, status = `{"isFavorite":false}`, http.StatusServiceUnavailable
			case "createTopic", "patchTopic":
				body, status = `{"name":"Example"}`, http.StatusServiceUnavailable
			}
			req := httptest.NewRequest(route.Method, url, strings.NewReader(body))
			if operation == "generateDraftAudio" || operation == "generateRevisionAudio" || operation == "createTTSPreview" || operation == "publishPreview" {
				req.Header.Set("Idempotency-Key", contractID)
			}
			if body != "" {
				req.Header.Set("Content-Type", "application/json")
			}
			if operation == "uploadCoverImage" {
				req.Header.Set("Content-Type", "image/png")
			}
			result := httptest.NewRecorder()
			healthy.ServeHTTP(result, req)
			requireContractResponse(t, req, contractRoute, result, status)

			switch operation {
			case "listTTSVoices":
				if !strings.Contains(result.Body.String(), "SunHi (mock)") || strings.Contains(result.Body.String(), "zh-CN") {
					t.Fatal("expected only a labeled Korean voice")
				}
				result = httptest.NewRecorder()
				unavailable.ServeHTTP(result, req)
				requireContractResponse(t, req, contractRoute, result, http.StatusServiceUnavailable)
				assertAPIError(t, result, 503, string(tts.Unavailable))
				for _, query := range []string{"", "?languageCode=ja-JP", "?languageCode=zh-CN&languageCode=ko-KR"} {
					bad := httptest.NewRequest(route.Method, "http://127.0.0.1"+route.Path+query, nil)
					result = httptest.NewRecorder()
					healthy.ServeHTTP(result, bad)
					requireContractResponse(t, bad, contractRoute, result, http.StatusBadRequest)
					assertAPIError(t, result, 400, string(tts.Invalid))
				}
			case "healthReady", "listLanguages", "listTopics":
				result = httptest.NewRecorder()
				unavailable.ServeHTTP(result, req)
				requireContractResponse(t, req, contractRoute, result, http.StatusServiceUnavailable)
				assertAPIError(t, result, 503, "SERVICE_UNAVAILABLE")
			}

			forbidden := httptest.NewRequest(route.Method, strings.Replace(url, "127.0.0.1", "invalid.example", 1), nil)
			result = httptest.NewRecorder()
			healthy.ServeHTTP(result, forbidden)
			requireContractResponse(t, forbidden, contractRoute, result, http.StatusForbidden)
			assertAPIError(t, result, 403, "FORBIDDEN")

			oversized := httptest.NewRequest(route.Method, req.URL.String(), strings.NewReader(strings.Repeat("x", MaxJSONBytes+1)))
			oversized.Header.Set("Content-Type", "application/json")
			result = httptest.NewRecorder()
			healthy.ServeHTTP(result, oversized)
			requireContractResponse(t, oversized, contractRoute, result, http.StatusRequestEntityTooLarge)
			assertAPIError(t, result, 413, "PAYLOAD_TOO_LARGE")
		})
	}
	for key := range operations {
		t.Errorf("OpenAPI route %s is missing from Gin", key)
	}
}

func requireContractResponse(t *testing.T, req *http.Request, route *routers.Route, result *httptest.ResponseRecorder, status int) {
	t.Helper()
	if result.Code != status {
		t.Fatalf("HTTP %d, expected %d, body %s", result.Code, status, result.Body)
	}
	input := &openapi3filter.ResponseValidationInput{
		RequestValidationInput: &openapi3filter.RequestValidationInput{Request: req, Route: route},
		Status:                 result.Code,
		Header:                 result.Header(),
		Options:                &openapi3filter.Options{IncludeResponseStatus: true},
	}
	if err := openapi3filter.ValidateResponse(t.Context(), input.SetBodyBytes(result.Body.Bytes())); err != nil {
		t.Fatal(err)
	}
	if result.Header().Get("X-Request-Id") == "" || (status != http.StatusNoContent && !strings.HasPrefix(result.Header().Get("Content-Type"), "application/json")) {
		t.Fatal("response lacks request ID or JSON content type")
	}
}

const contractID = "06dc9132-13d3-42da-a09c-000000000101"

type fixtureCatalog struct {
	languages []queries.ListEnabledLanguagesRow
	topics    []queries.ListTopicsRow
	err       error
}

func (f fixtureCatalog) ListEnabledLanguages(context.Context) ([]queries.ListEnabledLanguagesRow, error) {
	return f.languages, f.err
}

func (f fixtureCatalog) ListTopics(context.Context) ([]queries.ListTopicsRow, error) {
	return f.topics, f.err
}

func contractCatalog(t *testing.T) fixtureCatalog {
	t.Helper()
	var id1, id2 pgtype.UUID
	if err := id1.Scan("06dc9132-13d3-42da-a09c-000000000101"); err != nil {
		t.Fatal(err)
	}
	if err := id2.Scan("06dc9132-13d3-42da-a09c-000000000102"); err != nil {
		t.Fatal(err)
	}
	return fixtureCatalog{
		languages: []queries.ListEnabledLanguagesRow{
			{Code: "ko-KR", DisplayName: "Korean"}, {Code: "zh-CN", DisplayName: "Mandarin Chinese"},
		},
		topics: []queries.ListTopicsRow{
			{ID: id1, Name: "Custom Topic", LightBackground: "#ECE7FA", LightText: "#554095", DarkBackground: "#3E345C", DarkText: "#E4D8FF"},
			{ID: id2, Name: "History", SystemKey: pgtype.Text{String: "history", Valid: true}, LightBackground: "#FFF0DA", LightText: "#835921", DarkBackground: "#54422B", DarkText: "#F8DEB1"},
		},
	}
}
