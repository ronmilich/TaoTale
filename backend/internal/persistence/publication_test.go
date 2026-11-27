package persistence

import (
	"testing"

	"github.com/jackc/pgx/v5/pgtype"
	"taotale/backend/internal/persistence/queries"
)

func TestPublicationIntentWithCover(t *testing.T) {
	input := NewItem{Item: queries.AddItemParams{LanguageCode: "zh-CN"}}
	legacy, err := input.intentHash()
	if err != nil {
		t.Fatal(err)
	}
	const oldReceipt = "8c8db6e5b0dd67c23dad2c4128a24215cb90fe81d01566ba05d81c4109ddab94"
	if legacy != oldReceipt {
		t.Fatalf("stored publication receipt would no longer match: %s", legacy)
	}
	input.Item.CoverImageID = pgtype.UUID{Bytes: [16]byte{1}, Valid: true}
	withCover, err := input.intentHash()
	if err != nil || withCover == legacy {
		t.Fatalf("cover must change the publication intent: %s, %v", withCover, err)
	}
}
