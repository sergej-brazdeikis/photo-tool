package app

import (
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
