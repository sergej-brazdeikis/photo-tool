package app

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/theme"

	"photo-tool/internal/config"
	"photo-tool/internal/store"
)

// UX journey screenshots — software-driver tier (functional regression).
// Real binary tier: PHOTO_TOOL_GUI_E2E_LINUX=1 bin/photo-tool with PHOTO_TOOL_UX_* env.
func TestUXJourneyCapture(t *testing.T) {
	dir := os.Getenv("PHOTO_TOOL_UX_CAPTURE_DIR")
	if dir == "" {
		t.Skip("set PHOTO_TOOL_UX_CAPTURE_DIR to a writable directory to capture judge bundle PNGs")
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	t.Setenv("PHOTO_TOOL_UX_JOURNEY_TEST", "1")
	t.Cleanup(clearUXCaptureReviewGrid)

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
	win.Resize(fyne.NewSize(1280, 800))
	applyTestPhotoToolTheme(t, theme.VariantLight)
	t.Cleanup(func() {
		win.SetContent(nil)
		time.Sleep(450 * time.Millisecond)
	})

	outDir := dir
	if sub := os.Getenv("PHOTO_TOOL_UX_CAPTURE_SOFTWARE_SUBDIR"); sub != "" {
		outDir = filepath.Join(dir, sub)
	}

	err = RunUXJourneyCapture(JourneyCaptureOptions{
		Win:           win,
		OutDir:        outDir,
		DB:            db,
		LibraryRoot:   root,
		AppMode:       "software_driver",
		CaptureTool:   "go test ./internal/app -run TestUXJourneyCapture",
		GoTestTarget:  "TestUXJourneyCapture",
		UploadSeeds:   uxUploadSeedPathsFromEnv(),
		FlowFilter:    UXCaptureFlowFilter(),
		UseLightTheme: true,
	})
	if err != nil {
		t.Fatal(err)
	}
}
