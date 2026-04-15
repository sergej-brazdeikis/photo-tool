package app

import (
	"path/filepath"
	"strings"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/theme"

	"photo-tool/internal/config"
	"photo-tool/internal/store"
)

// UX invariant: shipping shell must not include the Story 2.1 theme demo strip (CI-enforceable).
const semanticStylePreviewMarker = "Semantic roles (preview)"

func TestReleaseShell_hasNoSemanticStylePreviewStrip(t *testing.T) {
	test.NewTempApp(t)

	root := filepath.Join(t.TempDir(), "lib")
	if err := config.EnsureLibraryLayout(root); err != nil {
		t.Fatal(err)
	}
	db, err := store.Open(root)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	win := test.NewTempWindow(t, nil)
	win.Resize(fyne.NewSize(1120, 600))
	test.ApplyTheme(t, NewPhotoToolTheme(theme.VariantDark))

	shell := NewMainShell(win, db, root, nil)
	win.SetContent(shell)

	for _, lb := range collectLabelsDeep(shell) {
		if strings.Contains(lb.Text, semanticStylePreviewMarker) {
			t.Fatalf("shipping shell must not show theme demo label; found: %q", lb.Text)
		}
	}
	for _, b := range collectButtons(shell) {
		if b.Text == "Destructive (preview)" || b.Text == "Reject / caution (preview)" {
			t.Fatalf("shipping shell must not include demo buttons; found %q", b.Text)
		}
	}
}
