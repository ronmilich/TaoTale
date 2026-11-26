// Package fingerprint identifies an exact synthesis intent independently of item metadata.
package fingerprint

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"math"
	"strconv"
	"strings"
	"unicode/utf8"
)

const SchemaVersion = 2

type Identity struct {
	LanguageCode, ProviderID, VoiceID string
	SpeakingRate                      float64
	StrategyID, StrategyVersion       string
	ExactSourceText                   string
}

// CanonicalJSON returns the exact UTF-8 bytes hashed by Sum.
func (id Identity) CanonicalJSON() ([]byte, error) {
	for _, field := range []string{id.LanguageCode, id.ProviderID, id.VoiceID, id.StrategyID, id.StrategyVersion} {
		if field == "" || !utf8.ValidString(field) {
			return nil, errors.New("missing or invalid UTF-8 fingerprint field")
		}
	}
	if !utf8.ValidString(id.ExactSourceText) {
		return nil, errors.New("invalid UTF-8 source")
	}
	rate, err := rateJSON(id.SpeakingRate)
	if err != nil {
		return nil, err
	}
	s := "{\"fingerprintSchemaVersion\":" + strconv.Itoa(SchemaVersion) +
		",\"languageCode\":" + quote(id.LanguageCode) +
		",\"providerId\":" + quote(id.ProviderID) +
		",\"voiceId\":" + quote(id.VoiceID) +
		",\"speakingRate\":" + rate +
		",\"strategyId\":" + quote(id.StrategyID) +
		",\"strategyVersion\":" + quote(id.StrategyVersion) +
		",\"exactSourceText\":" + quote(id.ExactSourceText) + "}"
	return []byte(s), nil
}

func (id Identity) Sum() (string, error) {
	b, err := id.CanonicalJSON()
	if err != nil {
		return "", err
	}
	return digest(b), nil
}

// LegacyIdentity contains the four fields used by Mandarin Reader.
type LegacyIdentity struct {
	SegmentationVersion int32
	SpeakingRate        float64
	Text, VoiceID       string
}

func (id LegacyIdentity) canonicalJSON() ([]byte, error) {
	if id.SegmentationVersion < 1 || id.VoiceID == "" || !utf8.ValidString(id.VoiceID) || !utf8.ValidString(id.Text) {
		return nil, errors.New("invalid legacy fingerprint identity")
	}
	rate, err := rateJSON(id.SpeakingRate)
	if err != nil {
		return nil, err
	}
	s := "{\"segmentationVersion\":" + strconv.FormatInt(int64(id.SegmentationVersion), 10) +
		",\"speakingRate\":" + rate + ",\"text\":" + quote(id.Text) +
		",\"voiceId\":" + quote(id.VoiceID) + "}"
	return []byte(s), nil
}

// Verify compares a stored lowercase SHA-256 digest with the original four-field identity.
func (id LegacyIdentity) Verify(stored string) (bool, error) {
	b, err := id.canonicalJSON()
	if err != nil {
		return false, err
	}
	return digest(b) == stored, nil
}

func digest(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

func rateJSON(rate float64) (string, error) {
	if rate <= 0 || math.IsNaN(rate) || math.IsInf(rate, 0) {
		return "", errors.New("speaking rate must be finite and positive")
	}
	b, err := json.Marshal(rate)
	return string(b), err
}

// quote implements JSON.stringify string escaping for validated UTF-8. In particular,
// HTML characters and U+2028/U+2029 remain literal even when preceded by a backslash.
func quote(s string) string {
	const digits = "0123456789abcdef"
	var out strings.Builder
	out.WriteByte('"')
	for _, r := range s {
		switch r {
		case '"', '\\':
			out.WriteByte('\\')
			out.WriteRune(r)
		case '\b':
			out.WriteString("\\b")
		case '\f':
			out.WriteString("\\f")
		case '\n':
			out.WriteString("\\n")
		case '\r':
			out.WriteString("\\r")
		case '\t':
			out.WriteString("\\t")
		default:
			if r < 0x20 {
				out.WriteString("\\u00")
				out.WriteByte(digits[r>>4])
				out.WriteByte(digits[r&15])
			} else {
				out.WriteRune(r)
			}
		}
	}
	out.WriteByte('"')
	return out.String()
}
