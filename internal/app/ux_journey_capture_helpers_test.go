package app

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
)

func tapButtonInOverlays(t *testing.T, win fyne.Window, want string) {
	t.Helper()
	if err := journeyTapButtonInOverlays(win, want); err != nil {
		t.Fatal(err)
	}
}

func tapLoupeShareButton(t *testing.T, win fyne.Window) {
	t.Helper()
	if err := journeyTapLoupeShare(win); err != nil {
		t.Fatal(err)
	}
}

func tapButtonInSharePreviewOverlay(t *testing.T, win fyne.Window, want string) {
	t.Helper()
	if err := journeyTapSharePreviewOverlay(win, want); err != nil {
		t.Fatal(err)
	}
}

func setSelectAt(t *testing.T, root fyne.CanvasObject, idx int, option string) {
	t.Helper()
	if err := journeySetSelectAt(root, idx, option); err != nil {
		t.Fatal(err)
	}
}

func findLoupeCloseButton(t *testing.T, win fyne.Window) {
	t.Helper()
	closeBtn, err := journeyFindLoupeClose(win)
	if err != nil {
		t.Fatal(err)
	}
	test.Tap(closeBtn)
	uxCaptureSettle()
}
