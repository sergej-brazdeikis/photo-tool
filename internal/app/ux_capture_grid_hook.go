package app

import "sync"

// PHOTO_TOOL_UX_CAPTURE_DIR plus PHOTO_TOOL_UX_JOURNEY_TEST=1 registers the main Review grid so
// [TestUXJourneyCapture] can open the loupe without walking Fyne List internals. Rejected grids use nil onLoupeOpen and do not register.
var uxCaptureReviewGridMu sync.Mutex
var uxCaptureReviewGrid *reviewAssetGrid

func registerUXCaptureReviewGrid(g *reviewAssetGrid) {
	uxCaptureReviewGridMu.Lock()
	defer uxCaptureReviewGridMu.Unlock()
	uxCaptureReviewGrid = g
}

func clearUXCaptureReviewGrid() {
	uxCaptureReviewGridMu.Lock()
	defer uxCaptureReviewGridMu.Unlock()
	uxCaptureReviewGrid = nil
}

func uxCaptureOpenReviewLoupeAt(idx int) bool {
	uxCaptureReviewGridMu.Lock()
	g := uxCaptureReviewGrid
	uxCaptureReviewGridMu.Unlock()
	if g == nil || g.onLoupeOpen == nil {
		return false
	}
	g.onLoupeOpen(idx)
	return true
}
