package app

import (
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"os"
)

// decodeImageFile loads a raster with the standard library decoders (JPEG/PNG/GIF).
// Other formats fall back to canvas.Image File loading in callers.
func decodeImageFile(path string) (image.Image, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()
	img, _, err := image.Decode(f)
	return img, err
}
