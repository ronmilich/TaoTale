package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgtype"
	"taotale/backend/internal/media"
	"taotale/backend/internal/persistence"
	"taotale/backend/internal/persistence/queries"
	"taotale/backend/internal/sourcetext"
	"taotale/backend/internal/tts"
)

// An explicit null clears a nullable field. Omitted fields stay untouched.
type field[T any] struct {
	Set   bool
	Value *T
}

func (f *field[T]) UnmarshalJSON(data []byte) error {
	f.Set = true
	return json.Unmarshal(data, &f.Value)
}

type itemInput struct {
	Title        field[string]   `json:"title"`
	SourceText   field[string]   `json:"sourceText"`
	Description  field[string]   `json:"description"`
	Notes        field[string]   `json:"notes"`
	Difficulty   field[string]   `json:"difficulty"`
	IsFavorite   field[bool]     `json:"isFavorite"`
	VoiceID      field[string]   `json:"voiceId"`
	SpeakingRate field[float64]  `json:"speakingRate"`
	TopicIDs     field[[]string] `json:"topicIds"`
	PreviewID    field[string]   `json:"previewGenerationId"`
}

func (i itemInput) validate(providerID string) (persistence.ItemPatch, string) {
	for _, entry := range []struct {
		value field[string]
		max   int
		name  string
	}{
		{i.Title, 120, "title"}, {i.Description, 1000, "description"}, {i.Notes, 5000, "notes"},
	} {
		if entry.value.Value != nil && (utf8.RuneCountInString(*entry.value.Value) > entry.max || strings.ContainsRune(*entry.value.Value, 0)) {
			return persistence.ItemPatch{}, "Invalid " + entry.name + "."
		}
	}
	if i.SourceText.Set && (i.SourceText.Value == nil || !validSource(*i.SourceText.Value)) {
		return persistence.ItemPatch{}, "Source must be a string of at most 1500 UTF-16 units."
	}
	if i.Difficulty.Value != nil {
		switch *i.Difficulty.Value {
		case "beginner", "intermediate", "advanced":
		default:
			return persistence.ItemPatch{}, "Choose a supported difficulty."
		}
	}
	if i.VoiceID.Value != nil && !tts.AllowedVoice("zh-CN", providerID, *i.VoiceID.Value) &&
		!tts.AllowedVoice("ko-KR", providerID, *i.VoiceID.Value) {
		return persistence.ItemPatch{}, "Choose a supported voice."
	}
	if i.IsFavorite.Set && i.IsFavorite.Value == nil {
		return persistence.ItemPatch{}, "Favorite must be a Boolean."
	}
	var rate *pgtype.Float8
	if i.SpeakingRate.Set {
		rate = &pgtype.Float8{}
		if value := i.SpeakingRate.Value; value != nil {
			if *value != .75 && *value != 1 && *value != 1.25 {
				return persistence.ItemPatch{}, "Choose a supported speaking rate."
			}
			*rate = pgtype.Float8{Float64: *value, Valid: true}
		}
	}
	var topics *[]pgtype.UUID
	if i.TopicIDs.Set {
		if i.TopicIDs.Value == nil || len(*i.TopicIDs.Value) > 10 {
			return persistence.ItemPatch{}, "Send up to ten distinct topic IDs."
		}
		ids := make([]pgtype.UUID, 0, len(*i.TopicIDs.Value))
		seen := make(map[pgtype.UUID]bool)
		for _, raw := range *i.TopicIDs.Value {
			id, ok := parseID(raw)
			if !ok || seen[id] {
				return persistence.ItemPatch{}, "Send up to ten distinct topic IDs."
			}
			seen[id] = true
			ids = append(ids, id)
		}
		topics = &ids
	}
	var previewID pgtype.UUID
	if i.PreviewID.Set {
		if id, ok := parseID(defaultValue(i.PreviewID.Value)); ok {
			previewID = id
		} else {
			return persistence.ItemPatch{}, "Send a valid preview generation ID."
		}
	}
	return persistence.ItemPatch{
		Title: textField(i.Title), Description: textField(i.Description), Notes: textField(i.Notes),
		Difficulty: textField(i.Difficulty), Voice: textField(i.VoiceID), SourceText: i.SourceText.Value,
		Favorite: i.IsFavorite.Value, Rate: rate, TopicIDs: topics, PreviewID: previewID,
	}, ""
}

func validSource(source string) bool {
	index, err := sourcetext.NewIndex(source)
	if err != nil || strings.ContainsRune(source, 0) {
		return false
	}
	end, _ := index.AtByte(len(source))
	return end.UTF16 <= 1500
}

func textField(f field[string]) *pgtype.Text {
	if !f.Set {
		return nil
	}
	if f.Value == nil {
		return &pgtype.Text{}
	}
	return &pgtype.Text{String: *f.Value, Valid: true}
}

func parseID(raw string) (pgtype.UUID, bool) {
	var id pgtype.UUID
	ok := len(raw) == 36 && id.Scan(raw) == nil && strings.EqualFold(raw, id.String())
	return id, ok
}

func itemResponse(item persistence.Item) gin.H {
	r := item.LibraryItem
	var rate *float64
	if r.SpeakingRate.Valid {
		rate = &r.SpeakingRate.Float64
	}
	status := "draft"
	if r.PublishedAt.Valid {
		status = "published"
	}
	return gin.H{
		"id": r.ID, "languageCode": r.LanguageCode, "title": r.Title, "sourceText": r.SourceText,
		"description": r.Description, "notes": r.Notes, "difficulty": r.Difficulty,
		"isFavorite": r.IsFavorite, "voiceId": r.VoiceID, "speakingRate": rate,
		"topicIds": item.TopicIDs, "version": r.Version, "status": status, "publishedAt": r.PublishedAt,
		"createdAt": r.CreatedAt.Time.UTC(), "updatedAt": r.UpdatedAt.Time.UTC(),
		"audioStatus": item.Audio.Status, "audioAssetId": item.Audio.AssetID, "audioGenerationId": item.Audio.ID,
	}
}

func registerLibrary(router *gin.Engine, store *persistence.Library, providerID string) {
	router.GET("/api/v1/items", func(c *gin.Context) {
		query, err := url.ParseQuery(c.Request.URL.RawQuery)
		page, pageOK := pageValue(query["page"], 1, 2147483647)
		size, sizeOK := pageValue(query["pageSize"], 8, 50)
		languages := query["languageCode"]
		if err != nil || len(languages) != 1 || !tts.SupportsLanguage(languages[0]) || !pageOK || !sizeOK {
			fail(c, http.StatusBadRequest, "VALIDATION_ERROR", "Choose one learning language and valid paging values.")
			return
		}
		if !hasLibrary(c, store) {
			return
		}
		items, total, err := store.ListItems(c.Request.Context(), queries.PageItemsParams{
			LanguageCode: languages[0], PageSize: size, PageOffset: (int64(page) - 1) * int64(size),
		})
		if err != nil {
			failLibrary(c, err)
			return
		}
		result := make([]gin.H, 0, len(items))
		for _, item := range items {
			result = append(result, itemResponse(item))
		}
		c.JSON(http.StatusOK, gin.H{"items": result, "total": total, "page": page, "pageSize": size})
	})
	router.POST("/api/v1/items", func(c *gin.Context) {
		input, ok := decodeNewItem(c, providerID)
		if !ok || !hasLibrary(c, store) {
			return
		}
		item, err := store.CreateItem(c.Request.Context(), input.Item, input.TopicIDs, input.PreviewID)
		writeItem(c, item, err, http.StatusCreated)
	})
	router.GET("/api/v1/items/:itemId", func(c *gin.Context) {
		id, ok := resourceID(c, "itemId", "ITEM_NOT_FOUND")
		if ok && hasLibrary(c, store) {
			item, err := store.GetItem(c.Request.Context(), id)
			writeItem(c, item, err, http.StatusOK)
		}
	})
	router.PATCH("/api/v1/items/:itemId", func(c *gin.Context) {
		id, ok := resourceID(c, "itemId", "ITEM_NOT_FOUND")
		if !ok {
			return
		}
		var input struct {
			itemInput
			ExpectedVersion int32 `json:"expectedVersion"`
		}
		if !DecodeJSON(c, &input) {
			return
		}
		patch, message := input.validate(providerID)
		if message == "" && input.ExpectedVersion < 1 {
			message = "Send the current expected version."
		}
		if message != "" {
			fail(c, http.StatusBadRequest, "VALIDATION_ERROR", message)
			return
		}
		if !hasLibrary(c, store) {
			return
		}
		patch.ExpectedVersion = input.ExpectedVersion
		item, err := store.PatchItem(c.Request.Context(), id, patch)
		writeItem(c, item, err, http.StatusOK)
	})
	router.DELETE("/api/v1/items/:itemId", func(c *gin.Context) {
		id, ok := resourceID(c, "itemId", "ITEM_NOT_FOUND")
		if ok && hasLibrary(c, store) {
			writeDeletion(c, store.DeleteItem(c.Request.Context(), id))
		}
	})
}

func decodeNewItem(c *gin.Context, providerID string) (persistence.NewItem, bool) {
	var input struct {
		itemInput
		Language string `json:"languageCode"`
	}
	if !DecodeJSON(c, &input) {
		return persistence.NewItem{}, false
	}
	patch, message := input.validate(providerID)
	if message == "" && !tts.SupportsLanguage(input.Language) {
		message = "Choose a supported learning language."
	}
	if message == "" && patch.Voice != nil && patch.Voice.Valid && !tts.AllowedVoice(input.Language, providerID, patch.Voice.String) {
		message = "Choose a voice compatible with the learning language."
	}
	if message != "" {
		fail(c, http.StatusBadRequest, "VALIDATION_ERROR", message)
		return persistence.NewItem{}, false
	}
	return persistence.NewItem{Item: queries.AddItemParams{
		LanguageCode: input.Language, Title: defaultValue(patch.Title), SourceText: defaultValue(patch.SourceText),
		Description: defaultValue(patch.Description), Notes: defaultValue(patch.Notes), Difficulty: defaultValue(patch.Difficulty),
		IsFavorite: defaultValue(patch.Favorite), VoiceID: defaultValue(patch.Voice), SpeakingRate: defaultValue(patch.Rate),
	}, TopicIDs: defaultValue(patch.TopicIDs), PreviewID: patch.PreviewID}, true
}

func defaultValue[T any](value *T) (zero T) {
	if value != nil {
		return *value
	}
	return zero
}

func pageValue(values []string, fallback, max int32) (int32, bool) {
	if len(values) == 0 {
		return fallback, true
	}
	if len(values) != 1 {
		return 0, false
	}
	value, err := strconv.ParseInt(values[0], 10, 32)
	return int32(value), err == nil && value >= 1 && value <= int64(max)
}

func resourceID(c *gin.Context, param, code string) (pgtype.UUID, bool) {
	id, ok := parseID(c.Param(param))
	if !ok {
		fail(c, http.StatusNotFound, code, "Resource not found.")
	}
	return id, ok
}

func hasLibrary(c *gin.Context, store *persistence.Library) bool {
	if store == nil {
		fail(c, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "Backend storage unavailable.")
	}
	return store != nil
}

func writeItem(c *gin.Context, item persistence.Item, err error, status int) {
	if err != nil {
		failLibrary(c, err)
		return
	}
	c.JSON(status, itemResponse(item))
}

func writeDeletion(c *gin.Context, err error) {
	if err != nil {
		failLibrary(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func failLibrary(c *gin.Context, err error) {
	var conflict *persistence.VersionConflict
	var duplicate *persistence.DuplicateTopic
	switch {
	case errors.As(err, &conflict):
		failDetails(c, 409, "VERSION_CONFLICT", "The item changed since it was loaded.", gin.H{"currentVersion": conflict.Current})
	case errors.As(err, &duplicate):
		failDetails(c, 409, "DUPLICATE_TOPIC", "A topic already has that name.", gin.H{"existingId": duplicate.ID})
	case errors.Is(err, persistence.ErrItemNotFound):
		fail(c, 404, "ITEM_NOT_FOUND", "Resource not found.")
	case errors.Is(err, persistence.ErrPreviewNotFound):
		fail(c, 404, "PREVIEW_NOT_FOUND", "Resource not found.")
	case errors.Is(err, persistence.ErrPreviewMismatch):
		fail(c, 409, "PREVIEW_MISMATCH", "The preview does not match the draft's speech inputs.")
	case errors.Is(err, persistence.ErrAudioNotReady):
		fail(c, 409, "AUDIO_NOT_READY", "The latest recording must be ready and match the current speech inputs.")
	case errors.Is(err, persistence.ErrAudioNotFound):
		fail(c, 404, "AUDIO_NOT_FOUND", "The requested recording is not current or does not exist.")
	case errors.Is(err, media.ErrAudioUnavailable):
		fail(c, 503, "AUDIO_UNAVAILABLE", "The verified recording is unavailable.")
	case errors.Is(err, persistence.ErrInvalidDraft):
		fail(c, 409, "INVALID_ITEM_STATE", "This action requires an unpublished draft.")
	case errors.Is(err, persistence.ErrPublicationIncomplete):
		fail(c, 400, "VALIDATION_ERROR", "Publication requires a title and source text.")
	case errors.Is(err, persistence.ErrKeyReused):
		fail(c, 409, "IDEMPOTENCY_KEY_REUSED", "This key was used for a different request.")
	case errors.Is(err, persistence.ErrKeyExpired):
		fail(c, 409, "IDEMPOTENCY_KEY_EXPIRED", "The original receipt has expired.")
	case errors.Is(err, persistence.ErrTopicNotFound):
		fail(c, 404, "TOPIC_NOT_FOUND", "Resource not found.")
	case errors.Is(err, persistence.ErrDefaultTopic):
		fail(c, 409, "DEFAULT_TOPIC_PROTECTED", "Default topics cannot be deleted.")
	case errors.Is(err, persistence.ErrPublishedSource):
		fail(c, 409, "INVALID_ITEM_STATE", "Published synthesis inputs require a working revision.")
	case errors.Is(err, persistence.ErrUnsupportedLanguage), errors.Is(err, persistence.ErrInvalidVoice):
		fail(c, 400, "VALIDATION_ERROR", "Choose an enabled language and compatible voice.")
	default:
		fail(c, 503, "SERVICE_UNAVAILABLE", "Backend storage unavailable.")
	}
}
