package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/theme"

	"photo-tool/internal/config"
	"photo-tool/internal/store"
)

func TestUXJourneyScaleSpotCapture(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(dir, "ui-real-scale")
	t.Setenv("PHOTO_TOOL_UX_CAPTURE_DIR", out)
	t.Setenv("PHOTO_TOOL_UX_JOURNEY_TEST", "1")
	t.Setenv("PHOTO_TOOL_UX_FIXTURE_SCALE", "S4")
	t.Setenv("PHOTO_TOOL_UX_CAPTURE_FLOWS", "scale_spot")
	t.Cleanup(clearUXCaptureReviewGrid)

	test.NewTempApp(t)
	root := filepath.Join(dir, "lib")
	if err := config.EnsureLibraryLayout(root); err != nil {
		t.Fatal(err)
	}
	db, err := store.Open(root)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	win := test.NewTempWindow(t, nil)
	win.Resize(fyne.NewSize(1280, 800))
	applyTestPhotoToolTheme(t, theme.VariantLight)

	if err := RunUXJourneyScaleSpot(win, db, root); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(out, "steps.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "scale_spot") {
		t.Fatalf("expected scale_spot steps in manifest")
	}
}

func TestUXJourneyEdgeCaptureS4(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(dir, "ui-real-edge")
	t.Setenv("PHOTO_TOOL_UX_CAPTURE_DIR", out)
	t.Setenv("PHOTO_TOOL_UX_JOURNEY_TEST", "1")
	t.Setenv("PHOTO_TOOL_UX_FIXTURE_SCALE", "S4")
	t.Setenv("PHOTO_TOOL_UX_CAPTURE_FLOWS", "edge")
	t.Cleanup(clearUXCaptureReviewGrid)

	test.NewTempApp(t)
	root := filepath.Join(dir, "lib")
	if err := config.EnsureLibraryLayout(root); err != nil {
		t.Fatal(err)
	}
	db, err := store.Open(root)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	win := test.NewTempWindow(t, nil)
	win.Resize(fyne.NewSize(1280, 800))
	applyTestPhotoToolTheme(t, theme.VariantLight)

	if err := RunUXJourneyEdgePack(win, db, root); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(out, "steps.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "review_loupe_share_rejected_block") {
		t.Fatalf("expected review_loupe_share_rejected_block in manifest: %s", string(data))
	}
}
