package app

import (
	"database/sql"
	"fmt"
	"strconv"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"photo-tool/internal/store"
)

func uxCaptureSettle() {
	d := 45 * time.Millisecond
	if isJourneyRealBinary() {
		d = 120 * time.Millisecond
	}
	time.Sleep(d)
}

func journeySettle() {
	uxCaptureSettle()
}

// journeyAfterNav waits for native GL repaint after primary nav or dialog open.
func journeyAfterNav(win fyne.Window) {
	if !isJourneyRealBinary() {
		journeySettle()
		return
	}
	journeyOnMain(func() {
		journeyRealBinaryRepaintDirect(win)
	})
	_ = journeyWaitPainted(win, 3*time.Second)
}

func journeyFindButton(root fyne.CanvasObject, want string) (*widget.Button, error) {
	for _, b := range collectButtonsForJourney(root) {
		if b.Text == want {
			return b, nil
		}
	}
	return nil, fmt.Errorf("button %q not found", want)
}

func journeyTapButton(root fyne.CanvasObject, want string) error {
	var err error
	journeyOnMain(func() {
		b, findErr := journeyFindButton(root, want)
		if findErr != nil {
			err = findErr
			return
		}
		if b.OnTapped == nil {
			err = fmt.Errorf("button %q has nil OnTapped", want)
			return
		}
		b.OnTapped()
		journeySettle()
	})
	return err
}

func journeyTapPanel(shell fyne.CanvasObject, label string) error {
	if err := journeyTapButton(shell, label); err != nil {
		return err
	}
	switch label {
	case "Review":
		uxCaptureActivateReviewGrid()
	case "Rejected":
		uxCaptureActivateRejectedGrid()
	}
	return nil
}

func journeySelectReviewGridAt(idx int) error {
	var err error
	journeyOnMain(func() {
		err = uxCaptureSelectReviewGridAt(idx)
		if err == nil {
			journeySettle()
		}
	})
	return err
}

func journeyOpenReviewLoupeAt(idx int) error {
	var err error
	journeyOnMain(func() {
		if !uxCaptureOpenReviewLoupeAt(idx) {
			err = fmt.Errorf("loupe: review grid not registered")
			return
		}
		journeySettle()
	})
	return err
}

func journeyCloseLoupe(win fyne.Window) error {
	var err error
	journeyOnMain(func() {
		closeBtn, findErr := journeyFindLoupeClose(win)
		if findErr != nil {
			err = findErr
			return
		}
		if closeBtn.OnTapped != nil {
			closeBtn.OnTapped()
		}
		journeySettle()
	})
	return err
}

func journeyListSelectAt(root fyne.CanvasObject, listIdx, rowIdx int) error {
	var err error
	journeyOnMain(func() {
		lists := collectListsForJourney(root)
		if len(lists) <= listIdx {
			err = fmt.Errorf("list index %d: have %d lists", listIdx, len(lists))
			return
		}
		lists[listIdx].Select(rowIdx)
		journeySettle()
	})
	return err
}

func journeyScrollListAt(root fyne.CanvasObject, listIdx, rowIdx int) error {
	var err error
	journeyOnMain(func() {
		lists := collectListsForJourney(root)
		if len(lists) <= listIdx {
			err = fmt.Errorf("list index %d: have %d lists", listIdx, len(lists))
			return
		}
		lists[listIdx].ScrollTo(rowIdx)
		journeySettle()
	})
	return err
}

// journeyCollectionSelectNamed opens the collections album list row whose label contains name.
func journeyCollectionSelectNamed(root fyne.CanvasObject, name string) error {
	var err error
	journeyOnMain(func() {
		lists := collectListsForJourney(root)
		if len(lists) == 0 {
			err = fmt.Errorf("collections list not found")
			return
		}
		list := lists[0]
		n := list.Length()
		for i := 0; i < n; i++ {
			list.Select(i)
			journeySettle()
			for _, lb := range collectLabelsDeepForJourney(root) {
				if strings.Contains(lb.Text, name) {
					return
				}
			}
		}
		err = fmt.Errorf("collection %q not found in list", name)
	})
	return err
}

func journeyWaitForUploadPostImport(root fyne.CanvasObject, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		var ready bool
		journeyOnMain(func() {
			for _, lb := range collectLabelsDeepForJourney(root) {
				if lb.Text == "Batch preview" {
					ready = true
					return
				}
			}
			for _, b := range collectButtonsDeep(root) {
				if b.Text == "Confirm" && b.Visible() {
					ready = true
					return
				}
			}
		})
		if ready {
			return nil
		}
		time.Sleep(50 * time.Millisecond)
	}
	return fmt.Errorf("timeout waiting for FR-06 post-import UI")
}

func journeyRadioGroupSelect(root fyne.CanvasObject, selected string) error {
	var err error
	journeyOnMain(func() {
		rg := firstRadioGroupForJourney(root)
		if rg == nil {
			err = fmt.Errorf("radio group not found")
			return
		}
		rg.Selected = selected
		if rg.OnChanged != nil {
			rg.OnChanged(selected)
		}
		rg.Refresh()
		journeySettle()
	})
	return err
}

func journeyResizeWin(win fyne.Window, w, h float32) {
	journeyOnMain(func() {
		win.Resize(fyne.NewSize(w, h))
		journeySettle()
	})
}

func journeySetSelectAt(root fyne.CanvasObject, idx int, option string) error {
	var err error
	journeyOnMain(func() {
		sels := collectSelectWidgetsForJourney(root)
		if len(sels) <= idx {
			err = fmt.Errorf("select index %d: have %d selects", idx, len(sels))
			return
		}
		sels[idx].SetSelected(option)
		sels[idx].Refresh()
		journeySettle()
	})
	return err
}

func journeyTapButtonInOverlays(win fyne.Window, want string) error {
	var err error
	journeyOnMain(func() {
		overs := win.Canvas().Overlays().List()
		for i := len(overs) - 1; i >= 0; i-- {
			for _, b := range collectButtonsDeep(overs[i]) {
				if b.Text == want {
					if b.OnTapped != nil {
						b.OnTapped()
						journeySettle()
						return
					}
				}
			}
		}
		err = fmt.Errorf("button %q not found in overlays", want)
	})
	return err
}

func journeyTapLoupeShare(win fyne.Window) error {
	var err error
	journeyOnMain(func() {
		overs := win.Canvas().Overlays().List()
		for i := len(overs) - 1; i >= 0; i-- {
			for _, b := range collectButtonsDeep(overs[i]) {
				if b.Text == "Share…" {
					if b.OnTapped != nil {
						b.OnTapped()
						journeySettle()
						return
					}
				}
			}
		}
		err = fmt.Errorf("Share… not found in overlays")
	})
	return err
}

func journeyTapSharePreviewOverlay(win fyne.Window, want string) error {
	var err error
	journeyOnMain(func() {
		top := win.Canvas().Overlays().Top()
		if top == nil {
			err = fmt.Errorf("no overlay top")
			return
		}
		hasShare := false
		for _, b := range collectButtonsDeep(top) {
			switch b.Text {
			case "Create link":
				hasShare = true
			default:
				if strings.HasPrefix(b.Text, "Create package link") {
					hasShare = true
				}
			}
		}
		if !hasShare {
			err = fmt.Errorf("top overlay is not share preview")
			return
		}
	})
	if err != nil {
		return err
	}
	journeyAfterNav(win)
	return journeyTapButtonInOverlays(win, want)
}

func journeyBulkTagAdd(root fyne.CanvasObject, label string) error {
	var err error
	journeyOnMain(func() {
		entries := collectEntriesForJourney(root)
		if len(entries) == 0 {
			err = fmt.Errorf("tag entry not found")
			return
		}
		entries[0].SetText(label)
		entries[0].Refresh()
		journeySettle()
	})
	if err != nil {
		return err
	}
	return journeyTapButton(root, "Add tag to selection")
}

func journeyTapSharePackage(root fyne.CanvasObject) error {
	for _, label := range []string{"Share (selection)…", "Share (filtered)…"} {
		if err := journeyTapButton(root, label); err == nil {
			return nil
		}
	}
	return fmt.Errorf("share package button not found (Share (selection)… / Share (filtered)…)")
}

func journeyTapShareFiltered(root fyne.CanvasObject) error {
	return journeyTapButton(root, "Share (filtered)…")
}

func journeyClearReviewGridSelection() {
	journeyOnMain(func() {
		uxCaptureClearReviewGridSelection()
		journeySettle()
	})
}

// journeyApplyZeroMatchFilters selects empty album UXCapNone (scale fixture) for zero-match empty state.
func journeyApplyZeroMatchFilters(shell fyne.CanvasObject) error {
	return journeySetSelectAt(shell, 0, "UXCapNone")
}

// journeyApplyZeroMatchDeadlockFilters stacks non-default filters that still match zero assets.
func journeyApplyZeroMatchDeadlockFilters(shell fyne.CanvasObject) error {
	if err := journeySetSelectAt(shell, 0, "UXCapNone"); err != nil {
		return err
	}
	if err := journeySetSelectAt(shell, 1, "5"); err != nil {
		return err
	}
	return journeySetSelectAt(shell, 2, "UXCapTag")
}

// journeyWaitCollectionThumbnails allows async collection detail decodes to finish before capture.
func journeyWaitCollectionThumbnails(win fyne.Window) {
	if !isJourneyRealBinary() {
		journeySettle()
		return
	}
	time.Sleep(600 * time.Millisecond)
	journeyAfterNav(win)
}

func journeySyncReviewGridSelection(win fyne.Window) {
	journeyOnMain(func() {
		uxCaptureReviewGridMu.Lock()
		g := uxCaptureReviewGrid
		uxCaptureReviewGridMu.Unlock()
		if g != nil && g.list != nil {
			g.list.Refresh()
		}
		if isJourneyRealBinary() {
			journeyRealBinaryRepaintDirect(win)
		}
		journeySettle()
	})
	journeyAfterNav(win)
}

func journeyKeyboardLoupeRating(win fyne.Window, n int) error {
	if n < 1 || n > 5 {
		return fmt.Errorf("loupe rating %d out of range 1–5", n)
	}
	var err error
	journeyOnMain(func() {
		type canvasTypedRune interface {
			OnTypedRune() func(rune)
		}
		tc, ok := win.Canvas().(canvasTypedRune)
		if !ok {
			err = fmt.Errorf("canvas does not support typed rune")
			return
		}
		fn := tc.OnTypedRune()
		if fn == nil {
			want := strconv.Itoa(n) + "★"
			overs := win.Canvas().Overlays().List()
			for i := len(overs) - 1; i >= 0; i-- {
				for _, b := range collectButtonsDeep(overs[i]) {
					if b.Text == want && b.OnTapped != nil {
						b.OnTapped()
						journeySettle()
						return
					}
				}
			}
			err = fmt.Errorf("loupe keyboard handler not registered")
			return
		}
		fn(rune('0' + n))
		journeySettle()
	})
	return err
}

func journeyResetFilters(shell fyne.CanvasObject) error {
	if err := journeyTapButton(shell, "Reset filters"); err == nil {
		return nil
	}
	if err := journeySetSelectAt(shell, 0, reviewCollectionSentinel); err != nil {
		return err
	}
	if err := journeySetSelectAt(shell, 1, reviewRatingAny); err != nil {
		return err
	}
	return journeySetSelectAt(shell, 2, reviewTagAny)
}

func journeyFindLoupeClose(win fyne.Window) (*widget.Button, error) {
	overs := win.Canvas().Overlays().List()
	for i := len(overs) - 1; i >= 0; i-- {
		bs := collectButtonsDeep(overs[i])
		var closeBtn *widget.Button
		hasLoupe := false
		for _, b := range bs {
			switch b.Text {
			case "Close":
				closeBtn = b
			case "← Prev", "Reject photo":
				hasLoupe = true
			}
		}
		if hasLoupe && closeBtn != nil {
			return closeBtn, nil
		}
	}
	return nil, fmt.Errorf("loupe Close not found")
}

func journeyTapLoupeButton(win fyne.Window, label string) error {
	var err error
	journeyOnMain(func() {
		overs := win.Canvas().Overlays().List()
		for i := len(overs) - 1; i >= 0; i-- {
			for _, b := range collectButtonsDeep(overs[i]) {
				if b.Text == label && b.OnTapped != nil {
					b.OnTapped()
					journeySettle()
					return
				}
			}
		}
		err = fmt.Errorf("loupe button %q not found", label)
	})
	return err
}

func journeyLoupeShareRejectedBlock(db *sql.DB, win fyne.Window) error {
	id := getUXCaptureLoupeAssetID()
	if id <= 0 {
		return fmt.Errorf("loupe share rejected block: no asset in loupe")
	}
	if _, err := store.RejectAsset(db, id, time.Now().Unix()); err != nil {
		return err
	}
	return journeyTapLoupeButton(win, "Share…")
}

func journeySimulateUploadDrop(path string) error {
	if uxJourneyUploadDropFn == nil {
		return fmt.Errorf("upload drop simulator not registered")
	}
	if path == "" {
		return fmt.Errorf("empty drop path")
	}
	var err error
	journeyOnMain(func() {
		uxJourneyUploadDropFn([]fyne.URI{storage.NewFileURI(path)})
		journeySettle()
	})
	return err
}

func journeyToggleAppTheme(win fyne.Window, light bool) {
	journeyOnMain(func() {
		v := theme.VariantDark
		if light {
			v = theme.VariantLight
		}
		if app := fyne.CurrentApp(); app != nil {
			app.Settings().SetTheme(NewPhotoToolTheme(v))
		}
		if c := win.Content(); c != nil {
			c.Refresh()
		}
		journeySettle()
	})
}
