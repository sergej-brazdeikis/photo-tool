package app

import (
	"encoding/json"
	"image/png"
	"os"
	"path/filepath"
	"testing"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"photo-tool/internal/config"
	"photo-tool/internal/store"
)

// UX journey screenshots for local judge bundles. Run (or use scripts/assemble-judge-bundle.sh):
//
//	PHOTO_TOOL_UX_JOURNEY_TEST=1 PHOTO_TOOL_UX_CAPTURE_DIR=/path/to/bundle/ui go test ./internal/app -run TestUXJourneyCapture -count=1
//
// PHOTO_TOOL_UX_JOURNEY_TEST scopes loupe registration to this subprocess so a stray CAPTURE_DIR env does not affect other tests.
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

	now := time.Now().Unix()
	// Default Review filter is "No assigned collection" — grid needs at least one asset that matches.
	_, err = store.InsertAssetWithCamera(db, "ux-cap-free", "2026/04/15/ux-cap-free.jpg", now, now, "", "")
	if err != nil {
		t.Fatal(err)
	}
	aidInAlbum, err := store.InsertAssetWithCamera(db, "ux-cap-in-album", "2026/04/15/ux-cap-in.jpg", now, now, "", "")
	if err != nil {
		t.Fatal(err)
	}
	aidRejected, err := store.InsertAssetWithCamera(db, "ux-cap-rejected", "2026/04/15/ux-cap-rej.jpg", now, now, "", "")
	if err != nil {
		t.Fatal(err)
	}
	cid, err := store.CreateCollection(db, "UXCaptureAlbum", "")
	if err != nil {
		t.Fatal(err)
	}
	if err := store.LinkAssetsToCollection(db, cid, []int64{aidInAlbum}); err != nil {
		t.Fatal(err)
	}
	if _, err := store.RejectAsset(db, aidRejected, now+1); err != nil {
		t.Fatal(err)
	}
	writeTinyJPEG(t, filepath.Join(root, "2026", "04", "15", "ux-cap-free.jpg"))
	writeTinyJPEG(t, filepath.Join(root, "2026", "04", "15", "ux-cap-in.jpg"))
	writeTinyJPEG(t, filepath.Join(root, "2026", "04", "15", "ux-cap-rej.jpg"))

	win := test.NewTempWindow(t, nil)
	win.Resize(fyne.NewSize(1280, 800))
	test.ApplyTheme(t, NewPhotoToolTheme(theme.VariantDark))

	shell := NewMainShell(win, db, root, nil)
	win.SetContent(shell)

	type stepMeta struct {
		ID     string `json:"id"`
		File   string `json:"file"`
		Intent string `json:"intent"`
	}
	var steps []stepMeta

	capture := func(id, file, intent string) {
		t.Helper()
		if c := win.Content(); c != nil {
			win.Canvas().Refresh(c)
		}
		img := win.Canvas().Capture()
		if img == nil {
			t.Fatalf("capture %s: Canvas().Capture() returned nil (driver may not support capture)", file)
		}
		path := filepath.Join(dir, file)
		f, err := os.Create(path)
		if err != nil {
			t.Fatalf("create %s: %v", path, err)
		}
		if err := png.Encode(f, img); err != nil {
			_ = f.Close()
			t.Fatalf("png %s: %v", path, err)
		}
		if err := f.Close(); err != nil {
			t.Fatalf("close %s: %v", path, err)
		}
		steps = append(steps, stepMeta{ID: id, File: file, Intent: intent})
	}

	capture("upload_default", "01_upload_default.png", "Upload tab: drop zone and import actions")
	tapPanel(t, shell, "Review")
	capture("review_default", "02_review_default.png", "Review: filter strip and grid with at least one asset")

	if !uxCaptureOpenReviewLoupeAt(0) {
		t.Fatal("loupe: review grid not registered (need PHOTO_TOOL_UX_JOURNEY_TEST=1 and PHOTO_TOOL_UX_CAPTURE_DIR before shell build — this test sets the former via t.Setenv)")
	}
	time.Sleep(50 * time.Millisecond) // let loupe overlay layout settle
	capture("review_loupe", "03_review_loupe.png", "Review loupe: image area and primary chrome")

	closeBtn := findLoupeCloseButton(t, win)
	test.Tap(closeBtn)
	time.Sleep(30 * time.Millisecond)

	tapPanel(t, shell, "Collections")
	capture("collections_list", "04_collections_list.png", "Collections: album list chrome")
	lists := collectLists(shell)
	if len(lists) < 1 {
		t.Fatal("expected album list on Collections panel")
	}
	lists[0].Select(0)
	capture("collections_album_detail", "05_collections_album_detail.png", "Collections: open first album (grid / detail chrome)")

	tapPanel(t, shell, "Rejected")
	capture("rejected_default", "06_rejected_default.png", "Rejected / hidden surface")

	manifest := struct {
		Steps        []stepMeta `json:"steps"`
		CaptureTool  string     `json:"capture_tool"`
		GoTestTarget string     `json:"go_test_target"`
		Omissions    []string   `json:"omissions"`
	}{
		Steps:        steps,
		CaptureTool:  "PHOTO_TOOL_UX_JOURNEY_TEST=1 PHOTO_TOOL_UX_CAPTURE_DIR=<dir> go test ./internal/app -run TestUXJourneyCapture -count=1",
		GoTestTarget: "TestUXJourneyCapture",
		Omissions:    []string{},
	}
	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "steps.json"), data, 0o644); err != nil {
		t.Fatal(err)
	}
}

func collectButtonsDeep(o fyne.CanvasObject) []*widget.Button {
	var out []*widget.Button
	var walk func(fyne.CanvasObject)
	walk = func(x fyne.CanvasObject) {
		if x == nil {
			return
		}
		switch v := x.(type) {
		case *widget.Button:
			out = append(out, v)
		case *container.Scroll:
			walk(v.Content)
		case *widget.PopUp:
			walk(v.Content)
		case *widget.Accordion:
			for _, it := range v.Items {
				if it != nil {
					walk(it.Detail)
				}
			}
		case *fyne.Container:
			for _, ch := range v.Objects {
				walk(ch)
			}
		}
	}
	walk(o)
	return out
}

func findLoupeCloseButton(t *testing.T, win fyne.Window) *widget.Button {
	t.Helper()
	overs := win.Canvas().Overlays().List()
	for i := len(overs) - 1; i >= 0; i-- {
		o := overs[i]
		bs := collectButtonsDeep(o)
		var closeBtn *widget.Button
		hasLoupeChrome := false
		for _, b := range bs {
			switch b.Text {
			case "Close":
				closeBtn = b
			case "← Prev", "Reject photo":
				hasLoupeChrome = true
			}
		}
		if hasLoupeChrome && closeBtn != nil {
			return closeBtn
		}
	}
	t.Fatal("no review loupe Close in canvas overlays (loupe not open?)")
	return nil
}
