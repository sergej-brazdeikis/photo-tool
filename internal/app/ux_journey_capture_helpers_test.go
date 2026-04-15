package app

import (
	"testing"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
)

func uxCaptureSettle() {
	time.Sleep(45 * time.Millisecond)
}

func tapButtonInOverlays(t *testing.T, win fyne.Window, want string) {
	t.Helper()
	overs := win.Canvas().Overlays().List()
	for i := len(overs) - 1; i >= 0; i-- {
		for _, b := range collectButtonsDeep(overs[i]) {
			if b.Text == want {
				test.Tap(b)
				return
			}
		}
	}
	t.Fatalf("button %q not found in any overlay", want)
}

func tapLoupeShareButton(t *testing.T, win fyne.Window) {
	t.Helper()
	overs := win.Canvas().Overlays().List()
	for i := len(overs) - 1; i >= 0; i-- {
		for _, b := range collectButtonsDeep(overs[i]) {
			if b.Text == "Share…" {
				test.Tap(b)
				return
			}
		}
	}
	t.Fatal("Share… not found in overlays")
}

func overlayHasButtonText(o fyne.CanvasObject, want string) bool {
	for _, b := range collectButtonsDeep(o) {
		if b.Text == want {
			return true
		}
	}
	return false
}

// tapButtonInTopOverlayIfSharePreview taps `want` only on the top overlay when it looks like the loupe Share preview
// (has "Create link" — avoids tapping unrelated Cancel buttons).
func tapButtonInSharePreviewOverlay(t *testing.T, win fyne.Window, want string) {
	t.Helper()
	top := win.Canvas().Overlays().Top()
	if top == nil || !overlayHasButtonText(top, "Create link") {
		t.Fatalf("top overlay is not Share preview (no Create link button)")
	}
	for _, b := range collectButtonsDeep(top) {
		if b.Text == want {
			test.Tap(b)
			return
		}
	}
	t.Fatalf("button %q not found on Share preview overlay", want)
}

func setSelectAt(t *testing.T, root fyne.CanvasObject, idx int, option string) {
	t.Helper()
	sels := collectSelectWidgets(root)
	if len(sels) <= idx {
		t.Fatalf("select index %d: have %d selects", idx, len(sels))
	}
	sels[idx].SetSelected(option)
	sels[idx].Refresh()
	uxCaptureSettle()
}
