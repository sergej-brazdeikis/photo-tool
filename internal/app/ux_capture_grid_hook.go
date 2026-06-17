package app

import (
	"fmt"
	"sync"
)

// PHOTO_TOOL_UX_CAPTURE_DIR plus PHOTO_TOOL_UX_JOURNEY_TEST=1 registers the main Review grid so
// [TestUXJourneyCapture] can open the loupe without walking Fyne List internals. Rejected grids use nil onLoupeOpen and do not register.
var uxCaptureReviewGridMu sync.Mutex
var uxCaptureReviewGrid *reviewAssetGrid
var uxCaptureReviewPanelGrid *reviewAssetGrid
var uxCaptureRejectedPanelGrid *reviewAssetGrid
var uxCaptureLoupeAssetID int64

func setUXCaptureLoupeAssetID(id int64) {
	uxCaptureReviewGridMu.Lock()
	uxCaptureLoupeAssetID = id
	uxCaptureReviewGridMu.Unlock()
}

func getUXCaptureLoupeAssetID() int64 {
	uxCaptureReviewGridMu.Lock()
	defer uxCaptureReviewGridMu.Unlock()
	return uxCaptureLoupeAssetID
}

func registerUXCaptureReviewGrid(g *reviewAssetGrid) {
	if g == nil {
		return
	}
	uxCaptureReviewGridMu.Lock()
	defer uxCaptureReviewGridMu.Unlock()
	if g.rejectedMode {
		uxCaptureRejectedPanelGrid = g
		return
	}
	uxCaptureReviewPanelGrid = g
	uxCaptureReviewGrid = g
}

func uxCaptureActivateReviewGrid() {
	uxCaptureReviewGridMu.Lock()
	defer uxCaptureReviewGridMu.Unlock()
	if uxCaptureReviewPanelGrid != nil {
		uxCaptureReviewGrid = uxCaptureReviewPanelGrid
	}
}

func uxCaptureActivateRejectedGrid() {
	uxCaptureReviewGridMu.Lock()
	defer uxCaptureReviewGridMu.Unlock()
	if uxCaptureRejectedPanelGrid != nil {
		uxCaptureReviewGrid = uxCaptureRejectedPanelGrid
	}
}

func clearUXCaptureReviewGrid() {
	uxCaptureReviewGridMu.Lock()
	defer uxCaptureReviewGridMu.Unlock()
	uxCaptureReviewGrid = nil
	uxCaptureReviewPanelGrid = nil
	uxCaptureRejectedPanelGrid = nil
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

// uxCaptureScrollReviewGridTo scrolls the review list so asset index idx is visible (0-based asset index).
func uxCaptureScrollReviewGridTo(idx int) error {
	uxCaptureReviewGridMu.Lock()
	g := uxCaptureReviewGrid
	uxCaptureReviewGridMu.Unlock()
	if g == nil || g.list == nil {
		return fmt.Errorf("review grid not registered for UX capture")
	}
	row := idx / reviewGridColumns
	g.list.ScrollTo(row)
	return nil
}

// uxCaptureSelectReviewGridRange selects assets [from, to) for bulk actions.
func uxCaptureSelectReviewGridRange(from, to int) error {
	for i := from; i < to; i++ {
		if err := uxCaptureSelectReviewGridAt(i); err != nil {
			return err
		}
	}
	return nil
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

// uxCaptureClearReviewGridSelection clears bulk thumbnail selection on the active review/rejected grid.
func uxCaptureClearReviewGridSelection() {
	uxCaptureReviewGridMu.Lock()
	g := uxCaptureReviewGrid
	uxCaptureReviewGridMu.Unlock()
	if g != nil {
		g.clearSelected()
	}
}
