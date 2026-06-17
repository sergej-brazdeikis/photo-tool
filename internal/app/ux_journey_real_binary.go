package app

import (
	"fmt"
	"image"
	"image/color"
	"sync"
	"time"

	"fyne.io/fyne/v2"
)

var (
	journeyRealBinaryMu sync.Mutex
	journeyRealBinary   bool
)

func setJourneyRealBinary(enabled bool) {
	journeyRealBinaryMu.Lock()
	journeyRealBinary = enabled
	journeyRealBinaryMu.Unlock()
}

func isJourneyRealBinary() bool {
	journeyRealBinaryMu.Lock()
	defer journeyRealBinaryMu.Unlock()
	return journeyRealBinary
}

// journeyOnMain runs fn on the Fyne UI thread (from journey worker goroutine only).
func journeyOnMain(fn func()) {
	if isJourneyRealBinary() {
		fyne.DoAndWait(fn)
		return
	}
	fn()
}

// journeyRealBinaryRepaintDirect flushes GL paint; caller must be on UI thread.
func journeyRealBinaryRepaintDirect(win fyne.Window) {
	if win == nil {
		return
	}
	cur := win.Canvas().Size()
	if cur.Width < 2 || cur.Height < 2 {
		return
	}
	win.Resize(fyne.NewSize(cur.Width+1, cur.Height))
	time.Sleep(60 * time.Millisecond)
	win.Resize(cur)
	if c := win.Content(); c != nil {
		win.Canvas().Refresh(c)
		c.Refresh()
	}
	time.Sleep(120 * time.Millisecond)
}

// journeyWarmRealBinaryWindow primes the GL framebuffer before the first capture in a journey.
func journeyWarmRealBinaryWindow(win fyne.Window) {
	if !isJourneyRealBinary() || win == nil {
		return
	}
	journeyOnMain(func() {
		for i := 0; i < 2; i++ {
			journeyRealBinaryRepaintDirect(win)
		}
	})
	time.Sleep(400 * time.Millisecond)
	journeyAfterNav(win)
}

// journeyWaitPainted polls until Canvas().Capture() returns a non-blank frame or timeout.
func journeyWaitPainted(win fyne.Window, timeout time.Duration) error {
	if !isJourneyRealBinary() || win == nil {
		return nil
	}
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		var blank bool
		journeyOnMain(func() {
			journeyRealBinaryRepaintDirect(win)
			var captured image.Image
			func() {
				defer func() { _ = recover() }()
				captured = win.Canvas().Capture()
			}()
			blank = journeyCaptureLooksBlank(captured)
		})
		if !blank {
			return nil
		}
		time.Sleep(80 * time.Millisecond)
	}
	return fmt.Errorf("journeyWaitPainted: GL frame still blank after %v", timeout)
}

// journeyCaptureLooksBlank detects unpainted GL buffers (uniform near-black frames).
func journeyCaptureLooksBlank(img image.Image) bool {
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
			r, g, bl, a := colorRGBA(img.At(px, py))
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

func colorRGBA(c color.Color) (r, g, b, a uint32) {
	rr, gg, bb, aa := c.RGBA()
	return rr >> 8, gg >> 8, bb >> 8, aa >> 8
}
