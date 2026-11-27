package httpapi

import (
	"encoding/json"

	"taotale/backend/internal/persistence/queries"
)

type transcriptSegment struct {
	Index     *int64  `json:"index"`
	TextStart *int64  `json:"textStart"`
	TextEnd   *int64  `json:"textEnd"`
	Text      *string `json:"text"`
	StartMS   *int64  `json:"startMs"`
	EndMS     *int64  `json:"endMs"`
}

type transcript struct {
	DurationMS int64               `json:"durationMs"`
	Segments   []transcriptSegment `json:"segments"`
}

func readTranscript(row *queries.GetTranscriptRow) *transcript {
	if row == nil || row.DurationMs <= 0 {
		return nil
	}
	var segments []transcriptSegment
	if err := json.Unmarshal(row.Segments, &segments); err != nil || len(segments) == 0 {
		return nil
	}
	for _, segment := range segments {
		if segment.Index == nil || *segment.Index < 0 ||
			segment.TextStart == nil || *segment.TextStart < 0 ||
			segment.TextEnd == nil || *segment.TextEnd < 1 ||
			segment.Text == nil || *segment.Text == "" ||
			segment.StartMS == nil || *segment.StartMS < 0 ||
			segment.EndMS == nil || *segment.EndMS < 1 {
			return nil
		}
	}
	return &transcript{DurationMS: row.DurationMs, Segments: segments}
}
