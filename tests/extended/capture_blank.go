package extended

import (
	"image"
	"image/color"
)

// captureLooksBlank detects unpainted GL buffers (same heuristic as internal/app journeyCaptureLooksBlank).
func captureLooksBlank(img image.Image) bool {
	if img == nil {
		return true
	}
	b := img.Bounds()
	if b.Dx() < 16 || b.Dy() < 16 {
		return true
	}
	var sum uint64
	n := 0
	for y := 0; y < 4; y++ {
		for x := 0; x < 4; x++ {
			px := b.Min.X + (b.Dx()-1)*x/3
			py := b.Min.Y + (b.Dy()-1)*y/3
			r, g, bl, a := captureColorRGBA(img.At(px, py))
			if a < 128 {
				continue
			}
			sum += uint64((299*r + 587*g + 114*bl) / 1000)
			n++
		}
	}
	if n == 0 {
		return true
	}
	return sum/uint64(n) < 12
}

func captureColorRGBA(c color.Color) (r, g, b, a uint32) {
	rr, gg, bb, aa := c.RGBA()
	return rr >> 8, gg >> 8, bb >> 8, aa >> 8
}
