package app

import (
	"fmt"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"
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
	time.Sleep(200 * time.Millisecond)
	journeySettle()
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
	return journeyTapButton(shell, label)
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
