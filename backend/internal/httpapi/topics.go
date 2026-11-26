package httpapi

import (
	"net/http"
	"strings"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgtype"
	"taotale/backend/internal/persistence"
	"taotale/backend/internal/persistence/queries"
)

var topicPresets = map[string]persistence.TopicPalette{
	"iris":       {LightBackground: "#ECE7FA", LightText: "#554095", DarkBackground: "#3E345C", DarkText: "#E4D8FF"},
	"periwinkle": {LightBackground: "#E7EAFE", LightText: "#444D9E", DarkBackground: "#31385C", DarkText: "#DCE0FF"},
	"orchid":     {LightBackground: "#F6E6F3", LightText: "#813E7B", DarkBackground: "#513450", DarkText: "#F4C8EC"},
	"sage":       {LightBackground: "#E1F0E9", LightText: "#286449", DarkBackground: "#294C3F", DarkText: "#B9E9D2"},
	"amber":      {LightBackground: "#FFF0DA", LightText: "#835921", DarkBackground: "#54422B", DarkText: "#F8DEB1"},
	"slate":      {LightBackground: "#EDECF3", LightText: "#59566B", DarkBackground: "#3A3749", DarkText: "#DEDAED"},
}

func registerTopicWrites(router *gin.Engine, store *persistence.Library) {
	router.POST("/api/v1/topics", func(c *gin.Context) {
		saveTopic(c, store, pgtype.UUID{})
	})
	router.PATCH("/api/v1/topics/:topicId", func(c *gin.Context) {
		if id, ok := resourceID(c, "topicId", "TOPIC_NOT_FOUND"); ok {
			saveTopic(c, store, id)
		}
	})
	router.DELETE("/api/v1/topics/:topicId", func(c *gin.Context) {
		if id, ok := resourceID(c, "topicId", "TOPIC_NOT_FOUND"); ok && hasLibrary(c, store) {
			writeDeletion(c, store.DeleteTopic(c.Request.Context(), id))
		}
	})
}

func saveTopic(c *gin.Context, store *persistence.Library, id pgtype.UUID) {
	var input struct {
		Name   field[string] `json:"name"`
		Preset field[string] `json:"colorPreset"`
	}
	if !DecodeJSON(c, &input) {
		return
	}
	if (input.Name.Set && input.Name.Value == nil) || (!id.Valid && input.Name.Value == nil) ||
		(!input.Name.Set && !input.Preset.Set) {
		fail(c, 400, "VALIDATION_ERROR", "Send a topic name or color preset for the update.")
		return
	}
	if name := input.Name.Value; name != nil {
		*name = strings.TrimSpace(*name)
		if *name == "" || utf8.RuneCountInString(*name) > 40 || strings.ContainsRune(*name, 0) {
			fail(c, 400, "VALIDATION_ERROR", "A topic name must have 1 to 40 Unicode code points.")
			return
		}
	}
	var palette *persistence.TopicPalette
	if input.Preset.Set {
		selected, ok := topicPresets[defaultValue(input.Preset.Value)]
		if !ok {
			fail(c, 400, "VALIDATION_ERROR", "Choose a valid topic color preset.")
			return
		}
		palette = &selected
	} else if !id.Valid {
		selected := topicPresets["iris"]
		palette = &selected
	}
	if !hasLibrary(c, store) {
		return
	}
	result, err := store.SaveTopic(c.Request.Context(), id, input.Name.Value, palette)
	if err != nil {
		failLibrary(c, err)
		return
	}
	status := http.StatusOK
	if !id.Valid {
		status = http.StatusCreated
	}
	c.JSON(status, topicResponse(queries.ListTopicsRow{
		ID: result.ID, Name: result.Name, SystemKey: result.SystemKey, UsageCount: result.UsageCount,
		LightBackground: result.LightBackground, LightText: result.LightText,
		DarkBackground: result.DarkBackground, DarkText: result.DarkText,
	}))
}
