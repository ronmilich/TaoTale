package fingerprint

import (
	"encoding/json"
	"math"
	"os"
	"testing"
)

type vector[T any] struct {
	Name                  string
	Identity              T
	CanonicalJSON, SHA256 string
}

func readFixture(t *testing.T, path string, dest any) {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(b, dest); err != nil {
		t.Fatal(err)
	}
}

func TestCanonicalVectors(t *testing.T) {
	var vectors []vector[Identity]
	readFixture(t, "testdata/v2.json", &vectors)
	if len(vectors) < 4 {
		t.Fatal("missing v2 vectors")
	}
	for _, v := range vectors {
		t.Run(v.Name, func(t *testing.T) {
			b, err := v.Identity.CanonicalJSON()
			if err != nil || string(b) != v.CanonicalJSON {
				t.Fatalf("canonical bytes %q, error %v", b, err)
			}
			sum, err := v.Identity.Sum()
			if err != nil || sum != v.SHA256 {
				t.Fatalf("digest %q, error %v", sum, err)
			}
		})
	}
}

func TestFrozenLegacyVectors(t *testing.T) {
	var fixture struct{ Fingerprints []vector[LegacyIdentity] }
	readFixture(t, "../../../testdata/legacy/mandarin-v1.json", &fixture)
	if len(fixture.Fingerprints) != 5 {
		t.Fatal("missing frozen legacy vectors")
	}
	for _, v := range fixture.Fingerprints {
		t.Run(v.Name, func(t *testing.T) {
			b, err := v.Identity.canonicalJSON()
			if err != nil || string(b) != v.CanonicalJSON {
				t.Fatalf("canonical bytes %q, error %v", b, err)
			}
			if ok, err := v.Identity.Verify(v.SHA256); err != nil || !ok {
				t.Fatalf("verify: %v %v", ok, err)
			}
			v.Identity.Text += " "
			if ok, err := v.Identity.Verify(v.SHA256); err != nil || ok {
				t.Fatalf("changed text: %v %v", ok, err)
			}
		})
	}
}

func TestInvalidIdentity(t *testing.T) {
	id := Identity{LanguageCode: "zh-CN", ProviderID: "azure", VoiceID: "voice", StrategyID: "zh-cn", StrategyVersion: "1.0.0", SpeakingRate: 1}
	for _, rate := range []float64{0, -1, math.NaN(), math.Inf(1)} {
		bad := id
		bad.SpeakingRate = rate
		if _, err := bad.Sum(); err == nil {
			t.Fatalf("accepted rate %v", rate)
		}
	}
	id.ExactSourceText = "\xff"
	if _, err := id.Sum(); err == nil {
		t.Fatal("accepted malformed text")
	}
	if ok, err := (LegacyIdentity{1, 1, "\xff", "voice"}).Verify(""); err == nil || ok {
		t.Fatal("accepted malformed legacy text")
	}
}
