package media

import (
	"errors"
	"os"
	"path/filepath"
)

var ErrAudioUnavailable = errors.New("verified audio file unavailable")

// OpenAudio opens the recorded file inside the media root and checks the open
// handle before any bytes are returned. The caller owns the file.
func OpenAudio(mediaRoot, key string, size int64) (*os.File, error) {
	file, err := OpenAsset(mediaRoot, key, size)
	if err != nil {
		return nil, ErrAudioUnavailable
	}
	return file, nil
}

var ErrAssetUnavailable = errors.New("media file unavailable")

func OpenAsset(mediaRoot, key string, size int64) (*os.File, error) {
	if size <= 0 || !filepath.IsLocal(key) || filepath.ToSlash(filepath.Clean(key)) != key {
		return nil, ErrAssetUnavailable
	}
	root, err := os.OpenRoot(mediaRoot)
	if err != nil {
		return nil, ErrAssetUnavailable
	}
	defer root.Close()
	entry, err := root.Lstat(key)
	if err != nil || !entry.Mode().IsRegular() {
		return nil, ErrAssetUnavailable
	}
	file, err := root.Open(key)
	if err != nil {
		return nil, ErrAssetUnavailable
	}
	info, err := file.Stat()
	if err != nil || !info.Mode().IsRegular() || !os.SameFile(entry, info) || info.Size() != size {
		file.Close()
		return nil, ErrAssetUnavailable
	}
	return file, nil
}
