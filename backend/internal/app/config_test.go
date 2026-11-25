package app

import "testing"

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
