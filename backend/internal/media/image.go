package media

import (
	"context"
	"crypto/rand"
	"errors"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"os"

	_ "golang.org/x/image/webp"
)

const MaxCoverBytes = 1_000_000
const maxCoverDimension = 4096

var (
	ErrCoverTooLarge = errors.New("cover exceeds encoded size limit")
	ErrInvalidCover  = errors.New("invalid cover format or dimensions")
	coverDecoders    = make(chan struct{}, 2)
)

type Cover struct {
	Key, ContentType string
	Size             int64
	Width, Height    int32
}

// SaveCover validates the whole image before atomically making its immutable file available.
func SaveCover(ctx context.Context, mediaRoot string, source io.Reader, contentType string) (Cover, error) {
	root, err := os.OpenRoot(mediaRoot)
	if err != nil {
		return Cover{}, err
	}
	defer root.Close()
	stage := "staging/" + rand.Text()
	file, err := root.OpenFile(stage, os.O_CREATE|os.O_EXCL|os.O_RDWR, 0600)
	if err != nil {
		return Cover{}, err
	}
	defer root.Remove(stage)
	defer file.Close()

	size, err := io.Copy(file, io.LimitReader(source, MaxCoverBytes+1))
	if err != nil {
		return Cover{}, err
	}
	if size > MaxCoverBytes {
		return Cover{}, ErrCoverTooLarge
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return Cover{}, err
	}
	cfg, format, err := image.DecodeConfig(file)
	if err != nil || cfg.Width < 1 || cfg.Height < 1 || cfg.Width > maxCoverDimension || cfg.Height > maxCoverDimension ||
		contentType != "image/"+format {
		return Cover{}, ErrInvalidCover
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return Cover{}, err
	}
	select {
	case coverDecoders <- struct{}{}:
		defer func() { <-coverDecoders }()
	case <-ctx.Done():
		return Cover{}, ctx.Err()
	}
	decoded, actual, err := image.Decode(file)
	if err != nil || actual != format || decoded.Bounds().Dx() != cfg.Width || decoded.Bounds().Dy() != cfg.Height {
		return Cover{}, ErrInvalidCover
	}
	if err := ctx.Err(); err != nil {
		return Cover{}, err
	}
	if err := file.Sync(); err != nil {
		return Cover{}, err
	}
	if err := file.Close(); err != nil {
		return Cover{}, err
	}
	key := "images/" + rand.Text() + "." + format
	if err := root.Rename(stage, key); err != nil {
		return Cover{}, err
	}
	return Cover{key, contentType, size, int32(cfg.Width), int32(cfg.Height)}, nil
}
