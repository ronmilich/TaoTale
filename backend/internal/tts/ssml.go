package tts

import (
	"strconv"
	"strings"
	"unicode/utf16"
	"unicode/utf8"
)

var xmlText = strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", "\"", "&quot;", "'", "&apos;", "\r", "&#13;")

func ssml(request Request) string {
	doc, _ := BoundaryDocument("azure", request)
	return doc
}

// BoundaryDocument builds the exact indexed document and maps provider scalar
// boundaries to source UTF-16 offsets. Wrapper and entity interiors are absent.
// Callers must first validate the provider, voice and source.
func BoundaryDocument(provider string, request Request) (string, map[uint]int) {
	var doc strings.Builder
	if provider == "azure" {
		rate := strconv.FormatFloat(request.Rate, 'f', -1, 64)
		doc.WriteString(`<speak version="1.0" xmlns="http://www.w3.org/2001/10/synthesis" xml:lang="` +
			request.Language + `"><voice name="` + request.VoiceID + `"><prosody rate="` + rate + `">`)
	}
	position := uint(doc.Len()) // Azure's validated wrapper consists of ASCII.
	sourcePosition := 0
	boundaries := map[uint]int{position: 0}
	for _, r := range request.Source {
		encoded := string(r)
		if provider == "azure" {
			encoded = xmlText.Replace(encoded)
		}
		doc.WriteString(encoded)
		position += uint(utf8.RuneCountInString(encoded))
		sourcePosition += utf16.RuneLen(r)
		boundaries[position] = sourcePosition
	}
	if provider == "azure" {
		doc.WriteString("</prosody></voice></speak>")
	}
	return doc.String(), boundaries
}
