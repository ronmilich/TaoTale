package app

import "testing"

func TestGenerationLimits(t *testing.T) {
	for _, tc := range []struct {
		active, hourly         string
		wantActive, wantHourly int64
	}{
		{"", "", 2, 60}, {"1", "12", 1, 12}, {"0", "", 0, 0}, {"", "-1", 0, 0}, {"invalid", "", 0, 0},
	} {
		env := map[string]string{"MEDIA_ROOT": t.TempDir(), "TTS_MAX_CONCURRENT_GENERATIONS": tc.active, "TTS_MAX_GENERATIONS_PER_HOUR": tc.hourly}
		config, err := LoadConfig(t.Context(), DefaultListen, func(name string) string { return env[name] })
		if (err != nil) != (tc.wantActive == 0) || config.GenerationLimits.Active != tc.wantActive || config.GenerationLimits.Hourly != tc.wantHourly {
			t.Fatalf("limits (%q, %q) produced %+v, error %v", tc.active, tc.hourly, config.GenerationLimits, err)
		}
	}
}

func TestLoadConfigValidatesLocalSettings(t *testing.T) {
	path := t.TempDir()
	for _, tc := range []struct {
		name, listen, media, origin, database string
		valid                                 bool
	}{
		{"default", DefaultListen, path, "", "", true},
		{"localhost", "localhost:8080", path, "", "", true},
		{"IPv6 loopback", "[::1]:8080", path, "", "", true},
		{"IPv4 wildcard", "0.0.0.0:8080", path, "", "", false},
		{"IPv6 wildcard", "[::]:8080", path, "", "", false},
		{"implicit wildcard", ":8080", path, "", "", false},
		{"LAN address", "192.168.1.2:8080", path, "", "", false},
		{"invalid port", "127.0.0.1:65536", path, "", "", false},
		{"relative media", DefaultListen, "relative", "", "", false},
		{"remote development origin", DefaultListen, path, "http://example.com", "", false},
		{"development origin with a path", DefaultListen, path, "http://localhost:3000/", "", false},
		{"invalid database config", DefaultListen, path, "", "malformed", false},
		{"configured development origin", DefaultListen, path, "http://localhost:3000", "", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			config, err := LoadConfig(t.Context(), tc.listen, func(name string) string {
				return map[string]string{"MEDIA_ROOT": tc.media, "TAOTALE_DEV_ORIGIN": tc.origin, "DATABASE_URL": tc.database}[name]
			})
			if (err == nil) != tc.valid {
				t.Fatalf("valid = %t, error = %v", tc.valid, err)
			}
			if tc.valid && (config.MediaRoot != path || config.Database == nil) {
				t.Fatal("missing typed media or database settings")
			}
		})
	}
}
