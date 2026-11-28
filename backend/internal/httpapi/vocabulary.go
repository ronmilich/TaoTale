package httpapi

import (
	"net/url"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgtype"
	"taotale/backend/internal/persistence"
	"taotale/backend/internal/persistence/queries"
	"taotale/backend/internal/tts"
)

const (
	vocabularyWordLimit  = 120
	vocabularyTopicLimit = 10
	vocabularyPageSize   = 10
	vocabularyBulkLimit  = 100
)

type vocabularyFields struct {
	Word          field[string]   `json:"word"`
	Reading       field[string]   `json:"reading"`
	ReadingSystem field[string]   `json:"readingSystem"`
	Meaning       field[string]   `json:"meaning"`
	Note          field[string]   `json:"note"`
	Status        field[string]   `json:"status"`
	TopicIDs      field[[]string] `json:"topicIds"`
}

func (input vocabularyFields) patch() (queries.UpdateVocabularyParams, *[]pgtype.UUID, string) {
	var result queries.UpdateVocabularyParams
	if input.Word.Set {
		word, key, ok := vocabularyWord(defaultValue(input.Word.Value))
		if !ok {
			return result, nil, "Enter a word or phrase of 1 to 120 Unicode code points."
		}
		result.Word = pgtype.Text{String: word, Valid: true}
		result.NormalizedWordKey = pgtype.Text{String: key, Valid: true}
	}
	for _, f := range []struct {
		label string
		input field[string]
		limit int
	}{
		{"reading", input.Reading, 160}, {"reading system", input.ReadingSystem, 40},
		{"meaning", input.Meaning, 300}, {"note", input.Note, 5000},
	} {
		if f.input.Value != nil && !validVocabularyText(*f.input.Value, f.limit) {
			return result, nil, "The " + f.label + " is too long or contains unsupported characters."
		}
	}
	if input.Status.Set {
		status := defaultValue(input.Status.Value)
		if !vocabularyStatus(status) {
			return result, nil, "Choose LEARNING or ACQUIRED."
		}
		result.Status = pgtype.Text{String: status, Valid: true}
	}
	result.SetReading, result.Reading = input.Reading.Set, defaultValue(textField(input.Reading))
	result.SetReadingSystem, result.ReadingSystem = input.ReadingSystem.Set, defaultValue(textField(input.ReadingSystem))
	result.SetMeaning, result.Meaning = input.Meaning.Set, defaultValue(textField(input.Meaning))
	result.SetNote, result.Note = input.Note.Set, defaultValue(textField(input.Note))
	var topics *[]pgtype.UUID
	if input.TopicIDs.Set {
		ids, ok := vocabularyIDs(defaultValue(input.TopicIDs.Value), 0, vocabularyTopicLimit)
		if input.TopicIDs.Value == nil || !ok {
			return result, nil, "Send up to ten distinct topic IDs."
		}
		topics = &ids
	}
	return result, topics, ""
}

func validVocabularyText(value string, limit int) bool {
	return utf8.ValidString(value) && !strings.ContainsRune(value, 0) && utf8.RuneCountInString(value) <= limit
}

func vocabularyWord(raw string) (word, key string, ok bool) {
	if !utf8.ValidString(raw) {
		return "", "", false
	}
	word, key = persistence.NormalizeVocabularyWord(raw)
	return word, key, key != "" && validVocabularyText(word, vocabularyWordLimit)
}

func vocabularyStatus(status string) bool { return status == "LEARNING" || status == "ACQUIRED" }

func vocabularyIDs(values []string, minimum, limit int) ([]pgtype.UUID, bool) {
	if len(values) < minimum || len(values) > limit {
		return nil, false
	}
	seen := make(map[pgtype.UUID]bool, len(values))
	ids := make([]pgtype.UUID, 0, len(values))
	for _, value := range values {
		id, ok := parseID(value)
		if !ok || seen[id] {
			return nil, false
		}
		seen[id] = true
		ids = append(ids, id)
	}
	return ids, true
}

type vocabularyResponse struct {
	ID            pgtype.UUID   `json:"id"`
	LanguageCode  string        `json:"languageCode"`
	Word          string        `json:"word"`
	Reading       pgtype.Text   `json:"reading"`
	ReadingSystem pgtype.Text   `json:"readingSystem"`
	Meaning       pgtype.Text   `json:"meaning"`
	Note          pgtype.Text   `json:"note"`
	Status        string        `json:"status"`
	TopicIDs      []pgtype.UUID `json:"topicIds"`
	CreatedAt     time.Time     `json:"createdAt"`
	UpdatedAt     time.Time     `json:"updatedAt"`
}

type vocabularyReference struct {
	ItemID       pgtype.UUID `json:"itemId"`
	AddedAt      time.Time   `json:"addedAt"`
	ManualLinked bool        `json:"manualLinked"`
	AutoLinked   bool        `json:"autoLinked"`
}

type vocabularyReferenceDetail struct {
	vocabularyReference
	Title pgtype.Text `json:"title"`
}

func vocabularyDTO(row persistence.Vocabulary) vocabularyResponse {
	return vocabularyResponse{
		ID: row.ID, LanguageCode: row.LanguageCode, Word: row.Word, Reading: row.Reading,
		ReadingSystem: row.ReadingSystem, Meaning: row.Meaning, Note: row.Note, Status: row.Status,
		TopicIDs: row.TopicIDs, CreatedAt: row.CreatedAt.Time.UTC(), UpdatedAt: row.UpdatedAt.Time.UTC(),
	}
}

func referenceDTO(row queries.VocabularyReference) vocabularyReference {
	return vocabularyReference{row.LibraryItemID, row.AddedAt.Time.UTC(), row.ManualLinked, row.AutoLinked}
}

func writeVocabulary(c *gin.Context, result persistence.VocabularyDetail, err error, status int) {
	if err != nil {
		failLibrary(c, err)
		return
	}
	references := make([]vocabularyReferenceDetail, 0, len(result.References))
	for _, row := range result.References {
		references = append(references, vocabularyReferenceDetail{
			vocabularyReference: vocabularyReference{row.LibraryItemID, row.AddedAt.Time.UTC(), row.ManualLinked, row.AutoLinked},
			Title:               row.Title,
		})
	}
	c.JSON(status, struct {
		vocabularyResponse
		References []vocabularyReferenceDetail `json:"references"`
	}{vocabularyDTO(result.Vocabulary), references})
}

func registerVocabulary(router *gin.Engine, store *persistence.Library) {
	group := router.Group("/api/v1/vocabulary")
	group.GET("", func(c *gin.Context) { listVocabulary(c, store) })
	group.POST("", func(c *gin.Context) { createVocabulary(c, store) })
	group.GET("/lookup", func(c *gin.Context) { lookupVocabulary(c, store) })
	group.GET("/stats", func(c *gin.Context) {
		query, ok := vocabularyLanguageQuery(c)
		if !ok || !hasLibrary(c, store) {
			return
		}
		language := query.Get("languageCode")
		stats, err := store.VocabularyStats(c.Request.Context(), language)
		if err != nil {
			failLibrary(c, err)
			return
		}
		c.JSON(200, gin.H{"languageCode": language, "total": stats.Total, "learning": stats.Learning,
			"acquired": stats.Acquired, "referencedTexts": stats.ReferencedTexts})
	})
	group.POST("/bulk", func(c *gin.Context) { bulkVocabulary(c, store) })
	entry := group.Group("/:vocabularyId")
	entry.GET("", func(c *gin.Context) {
		if id, ok := resourceID(c, "vocabularyId", "VOCABULARY_NOT_FOUND"); ok && hasLibrary(c, store) {
			result, err := store.GetVocabulary(c.Request.Context(), id)
			writeVocabulary(c, result, err, 200)
		}
	})
	entry.PATCH("", func(c *gin.Context) { patchVocabulary(c, store) })
	entry.DELETE("", func(c *gin.Context) {
		if id, ok := resourceID(c, "vocabularyId", "VOCABULARY_NOT_FOUND"); ok && hasLibrary(c, store) {
			writeDeletion(c, store.DeleteVocabulary(c.Request.Context(), id))
		}
	})
	entry.POST("/references", func(c *gin.Context) { changeVocabularyReference(c, store, true) })
	entry.DELETE("/references/:itemId", func(c *gin.Context) { changeVocabularyReference(c, store, false) })
	router.GET("/api/v1/items/:itemId/vocabulary", func(c *gin.Context) { itemVocabulary(c, store) })
}

func createVocabulary(c *gin.Context, store *persistence.Library) {
	var input struct {
		vocabularyFields
		LanguageCode string        `json:"languageCode"`
		ItemID       field[string] `json:"itemId"`
	}
	if !DecodeJSON(c, &input) {
		return
	}
	patch, topics, message := input.patch()
	if message == "" && (!tts.SupportsLanguage(input.LanguageCode) || !patch.Word.Valid) {
		message = "Choose a supported learning language and enter a word or phrase."
	}
	var item pgtype.UUID
	if message == "" && input.ItemID.Set {
		var ok bool
		item, ok = parseID(defaultValue(input.ItemID.Value))
		if !ok {
			message = "Send a valid item ID."
		}
	}
	if message != "" {
		fail(c, 400, "VALIDATION_ERROR", message)
		return
	}
	if !hasLibrary(c, store) {
		return
	}
	status := "LEARNING"
	if patch.Status.Valid {
		status = patch.Status.String
	}
	result, err := store.CreateVocabulary(c.Request.Context(), queries.InsertVocabularyParams{
		LanguageCode: input.LanguageCode, Word: patch.Word.String, NormalizedWordKey: patch.NormalizedWordKey.String,
		Reading: patch.Reading, ReadingSystem: patch.ReadingSystem, Meaning: patch.Meaning, Note: patch.Note, Status: status,
	}, defaultValue(topics), item)
	writeVocabulary(c, result, err, 201)
}

func patchVocabulary(c *gin.Context, store *persistence.Library) {
	id, ok := resourceID(c, "vocabularyId", "VOCABULARY_NOT_FOUND")
	var input vocabularyFields
	if !ok || !DecodeJSON(c, &input) {
		return
	}
	patch, topics, message := input.patch()
	if message == "" && input == (vocabularyFields{}) {
		message = "Send at least one vocabulary field to update."
	}
	if message != "" {
		fail(c, 400, "VALIDATION_ERROR", message)
		return
	}
	if !hasLibrary(c, store) {
		return
	}
	patch.ID = id
	result, err := store.PatchVocabulary(c.Request.Context(), patch, topics)
	writeVocabulary(c, result, err, 200)
}

func vocabularyLanguageQuery(c *gin.Context) (url.Values, bool) {
	query, err := url.ParseQuery(c.Request.URL.RawQuery)
	if err != nil || len(query["languageCode"]) != 1 || !tts.SupportsLanguage(query.Get("languageCode")) {
		fail(c, 400, "VALIDATION_ERROR", "Choose one supported learning language.")
		return nil, false
	}
	return query, true
}

func lookupVocabulary(c *gin.Context, store *persistence.Library) {
	query, ok := vocabularyLanguageQuery(c)
	if !ok {
		return
	}
	_, key, valid := vocabularyWord(query.Get("word"))
	if !valid || len(query["word"]) != 1 {
		fail(c, 400, "VALIDATION_ERROR", "Send one word or phrase to look up.")
		return
	}
	if !hasLibrary(c, store) {
		return
	}
	row, err := store.FindVocabulary(c.Request.Context(), query.Get("languageCode"), key)
	if err != nil {
		failLibrary(c, err)
		return
	}
	var entry *vocabularyResponse
	if row != nil {
		matched := vocabularyDTO(*row)
		entry = &matched
	}
	c.JSON(200, gin.H{"entry": entry})
}

func listVocabulary(c *gin.Context, store *persistence.Library) {
	query, ok := vocabularyLanguageQuery(c)
	if !ok {
		return
	}
	params, page, ok := vocabularyFilters(query)
	if !ok {
		fail(c, 400, "VALIDATION_ERROR", "Send valid vocabulary filters and page.")
		return
	}
	if !hasLibrary(c, store) {
		return
	}
	rows, total, err := store.ListVocabulary(c.Request.Context(), params)
	if err != nil {
		failLibrary(c, err)
		return
	}
	items := make([]vocabularyResponse, 0, len(rows))
	for _, row := range rows {
		items = append(items, vocabularyDTO(row))
	}
	c.JSON(200, gin.H{"items": items, "total": total, "page": page, "pageSize": vocabularyPageSize})
}

func vocabularyFilters(query url.Values) (queries.PageVocabularyParams, int32, bool) {
	for _, key := range []string{"q", "status", "topics", "sort", "pageSize"} {
		if len(query[key]) > 1 {
			return queries.PageVocabularyParams{}, 0, false
		}
	}
	page, pageOK := pageValue(query["page"], 1, 2147483647)
	size, sizeOK := pageValue(query["pageSize"], vocabularyPageSize, vocabularyPageSize)
	search, status, sort := query.Get("q"), query.Get("status"), query.Get("sort")
	if !pageOK || !sizeOK || size != vocabularyPageSize ||
		!validVocabularyText(search, vocabularyWordLimit) || (status != "" && !vocabularyStatus(status)) ||
		!slices.Contains([]string{"", "newest", "oldest", "word_asc", "word_desc"}, sort) {
		return queries.PageVocabularyParams{}, 0, false
	}
	var topics []pgtype.UUID
	if query.Has("topics") {
		var ok bool
		topics, ok = vocabularyIDs(strings.Split(query.Get("topics"), ","), 1, vocabularyTopicLimit)
		if !ok {
			return queries.PageVocabularyParams{}, 0, false
		}
	}
	pattern := strings.TrimSpace(search)
	if pattern != "" {
		pattern = "%" + literalLike.Replace(pattern) + "%"
	}
	return queries.PageVocabularyParams{LanguageCode: query.Get("languageCode"), Pattern: pattern, Status: status,
		SortOrder: sort, TopicIds: topics, PageOffset: (int64(page) - 1) * vocabularyPageSize}, page, true
}

func changeVocabularyReference(c *gin.Context, store *persistence.Library, link bool) {
	id, ok := resourceID(c, "vocabularyId", "VOCABULARY_NOT_FOUND")
	if !ok {
		return
	}
	var itemID pgtype.UUID
	if link {
		var input struct {
			ItemID string `json:"itemId"`
		}
		if !DecodeJSON(c, &input) {
			return
		}
		itemID, ok = parseID(input.ItemID)
		if !ok {
			fail(c, 400, "VALIDATION_ERROR", "Send a valid item ID.")
			return
		}
	} else if itemID, ok = resourceID(c, "itemId", "ITEM_NOT_FOUND"); !ok {
		return
	}
	if !hasLibrary(c, store) {
		return
	}
	reference, err := store.ChangeVocabularyReference(c.Request.Context(), id, itemID, link)
	if !link || err != nil {
		writeDeletion(c, err)
		return
	}
	c.JSON(200, referenceDTO(reference))
}

func itemVocabulary(c *gin.Context, store *persistence.Library) {
	id, ok := resourceID(c, "itemId", "ITEM_NOT_FOUND")
	if !ok || !hasLibrary(c, store) {
		return
	}
	rows, err := store.ItemVocabulary(c.Request.Context(), id)
	if err != nil {
		failLibrary(c, err)
		return
	}
	type itemWord struct {
		vocabularyResponse
		Reference vocabularyReference `json:"reference"`
	}
	items := make([]itemWord, 0, len(rows))
	for _, row := range rows {
		items = append(items, itemWord{vocabularyDTO(row.Vocabulary), referenceDTO(row.Reference)})
	}
	c.JSON(200, gin.H{"items": items})
}

func bulkVocabulary(c *gin.Context, store *persistence.Library) {
	var input struct {
		LanguageCode string        `json:"languageCode"`
		Action       string        `json:"action"`
		IDs          []string      `json:"ids"`
		Status       field[string] `json:"status"`
	}
	if !DecodeJSON(c, &input) {
		return
	}
	ids, ok := vocabularyIDs(input.IDs, 1, vocabularyBulkLimit)
	status := input.Status.Value
	validAction := (input.Action == "delete" && !input.Status.Set) ||
		(input.Action == "setStatus" && status != nil && vocabularyStatus(*status))
	if !ok || !validAction || !tts.SupportsLanguage(input.LanguageCode) {
		fail(c, 400, "VALIDATION_ERROR", "Choose a supported language and action for 1 to 100 distinct vocabulary IDs.")
		return
	}
	if !hasLibrary(c, store) {
		return
	}
	count, err := store.BulkVocabulary(c.Request.Context(), input.LanguageCode, ids, status)
	if err != nil {
		failLibrary(c, err)
		return
	}
	c.JSON(200, gin.H{"affected": count})
}
