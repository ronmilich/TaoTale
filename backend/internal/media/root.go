package media

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// PrepareRoot initializes the private directory layout on the durable media volume.
func PrepareRoot(path string) error {
	if !filepath.IsAbs(path) {
		return errors.New("MEDIA_ROOT must be an absolute path")
	}
	path = filepath.Clean(path)
	if err := os.MkdirAll(path, 0700); err != nil {
		return errors.New("cannot create MEDIA_ROOT")
	}
	info, err := os.Lstat(path)
	if err != nil || !info.IsDir() {
		return errors.New("MEDIA_ROOT must be a directory")
	}
	root, err := os.OpenRoot(path)
	if err != nil {
		return errors.New("cannot open MEDIA_ROOT")
	}
	defer root.Close()

	// Check parents before their children. Existing volumes can have a more
	// permissive root mode from the old image, so apply the mode on each boot.
	directories := []string{".", "staging", "temporary", "temporary/audio", "audio", "temporary/images", "images"}
	for _, dir := range directories {
		if dir != "." {
			if err := root.Mkdir(dir, 0700); err != nil && !errors.Is(err, os.ErrExist) {
				return fmt.Errorf("cannot create media directory %q", dir)
			}
		}
		info, err := root.Lstat(dir)
		if err != nil || !info.IsDir() {
			return fmt.Errorf("media path %q must be a directory", dir)
		}
		if info.Mode().Perm() != 0700 {
			if err := root.Chmod(dir, 0700); err != nil {
				return fmt.Errorf("cannot restrict permissions on media directory %q", dir)
			}
		}
	}
	return nil
}
