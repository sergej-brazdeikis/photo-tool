package app

import (
	"fmt"
	"sync"
)

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

// uxCaptureSelectReviewGridAt toggles bulk selection for the grid row at idx (0-based).
func uxCaptureSelectReviewGridAt(idx int) error {
	uxCaptureReviewGridMu.Lock()
	g := uxCaptureReviewGrid
	uxCaptureReviewGridMu.Unlock()
	if g == nil {
		return fmt.Errorf("review grid not registered for UX capture")
	}
	row, ok, err := g.rowAt(idx)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("review grid row %d not found", idx)
	}
	g.toggleSelected(row.ID)
	return nil
}
