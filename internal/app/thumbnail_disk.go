package app

import (
	"fmt"
	"image"
	"image/jpeg"
	"os"
	"path/filepath"

	"github.com/nfnt/resize"
)

const thumbnailMaxEdgePx = 256

// ThumbnailCachePath returns the on-disk JPEG path for a cached thumbnail (architecture §3.8).
func ThumbnailCachePath(libraryRoot string, assetID int64, contentHash string) string {
	sub := "_"
	if len(contentHash) >= 2 {
		sub = contentHash[:2]
	}
	return filepath.Join(libraryRoot, ".cache", "thumbnails", sub, fmt.Sprintf("%d.jpg", assetID))
}

// WriteThumbnailJPEG decodes srcPath, scales so the longest edge is at most thumbnailMaxEdgePx,
// and writes a JPEG to destPath (parent dirs created). Used off the UI goroutine.
func WriteThumbnailJPEG(srcPath, destPath string) error {
	srcFi, err := os.Stat(srcPath)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(destPath), 0o755); err != nil {
		return err
	}
	if dstFi, err := os.Stat(destPath); err == nil {
		if dstFi.ModTime().After(srcFi.ModTime()) || dstFi.ModTime().Equal(srcFi.ModTime()) {
			return nil
		}
	}

	f, err := os.Open(srcPath)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()

	img, _, err := image.Decode(f)
	if err != nil {
		return err
	}
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	if w <= 0 || h <= 0 {
		return fmt.Errorf("invalid image bounds %dx%d", w, h)
	}
	var out image.Image = img
	if w > thumbnailMaxEdgePx || h > thumbnailMaxEdgePx {
		if w >= h {
			out = resize.Resize(thumbnailMaxEdgePx, 0, img, resize.Lanczos3)
		} else {
			out = resize.Resize(0, thumbnailMaxEdgePx, img, resize.Lanczos3)
		}
	}

	tmp := destPath + ".tmp"
	outF, err := os.Create(tmp)
	if err != nil {
		return err
	}
	err = jpeg.Encode(outF, out, &jpeg.Options{Quality: 82})
	_ = outF.Close()
	if err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return os.Rename(tmp, destPath)
}
