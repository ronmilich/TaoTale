package media

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestPrepareRootPreservesExistingMedia(t *testing.T) {
	path := t.TempDir()
	audio := filepath.Join(path, "audio")
	if err := os.Mkdir(audio, 0755); err != nil {
		t.Fatal(err)
	}
	asset := filepath.Join(audio, "existing.mp3")
	if err := os.WriteFile(asset, []byte("existing bytes"), 0600); err != nil {
		t.Fatal(err)
	}
	for range 2 {
		if err := PrepareRoot(path); err != nil {
			t.Fatal(err)
		}
	}
	for _, dir := range []string{".", "audio", "images", "staging", "temporary", "temporary/audio", "temporary/images"} {
		info, err := os.Stat(filepath.Join(path, dir))
		if err != nil {
			t.Fatal(err)
		}
		if !info.IsDir() || (runtime.GOOS != "windows" && info.Mode().Perm() != 0700) {
			t.Errorf("%s has mode %v", dir, info.Mode())
		}
	}
	if data, err := os.ReadFile(asset); err != nil || string(data) != "existing bytes" {
		t.Fatalf("existing media was not preserved, read error %v", err)
	}
}

func TestPrepareRootRejectsTemporarySymlink(t *testing.T) {
	path := t.TempDir()
	target := t.TempDir()
	if err := os.Symlink(target, filepath.Join(path, "temporary")); err != nil {
		if runtime.GOOS == "windows" {
			t.Skipf("symlink unavailable: %v", err)
		}
		t.Fatal(err)
	}
	if err := PrepareRoot(path); err == nil {
		t.Fatal("expected a symlink to be rejected")
	}
	entries, err := os.ReadDir(target)
	if err != nil || len(entries) != 0 {
		t.Fatalf("unexpected changes outside the media root, read error %v", err)
	}
}
