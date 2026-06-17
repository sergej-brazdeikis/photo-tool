package fixture

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"image"
	"image/color"
	"image/jpeg"
	"os"
	"path/filepath"
)

// WriteTinyJPEG writes a small non-uniform JPEG (unique per path) for scale fixtures.
func WriteTinyJPEG(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	const w, h = 48, 36
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	sum := sha256.Sum256([]byte(path))
	seed := binary.LittleEndian.Uint64(sum[:8])
	r0 := 35 + uint8(seed%90)
	g0 := 40 + uint8((seed>>8)%100)
	b0 := 45 + uint8((seed>>16)%95)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			dx := (x * 220) / max(w-1, 1)
			dy := (y * 200) / max(h-1, 1)
			r := min(255, int(r0)+dx+int((seed>>(x%8))&3))
			g := min(255, int(g0)+dy+int((seed>>(y%8))&3))
			b := min(255, int(b0)+(x+y)*65/max(w+h-2, 1))
			img.Set(x, y, color.NRGBA{R: uint8(r), G: uint8(g), B: uint8(b), A: 255})
		}
	}
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 85}); err != nil {
		return err
	}
	return os.WriteFile(path, buf.Bytes(), 0o644)
}
