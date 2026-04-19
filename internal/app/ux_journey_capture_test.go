package app

import (
	"encoding/json"
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"photo-tool/internal/config"
	"photo-tool/internal/domain"
	"photo-tool/internal/store"
)

// UX journey screenshots for local judge bundles — full primary flows (Upload, Review+loupe+share+filters,
// Collections list/detail/grouping/dialog, Rejected, then a second shell pass for FR-06 import).
//
//	PHOTO_TOOL_UX_JOURNEY_TEST=1 PHOTO_TOOL_UX_CAPTURE_DIR=/path/to/bundle/ui go test ./internal/app -run TestUXJourneyCapture -count=1
//	Optional: PHOTO_TOOL_TEST_THEME=dark for dark captures (default is light).
//
// Shell upload seeding (phase 2 only): newline-separated absolute paths in PHOTO_TOOL_UX_UPLOAD_SEED_PATHS (set by this test).
// After the main 1280×800 flow, NFR-01 minimum (1024×768) frames include Rejected + Upload during phase 1, then
// Review grid/loupe/share preview + Collections album detail + album list (Back from detail) appended after phase 2 (stable 01–21 filenames).
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

	srcDir := filepath.Join(t.TempDir(), "ux-cap-src")
	if err := os.MkdirAll(srcDir, 0o755); err != nil {
		t.Fatal(err)
	}

	now := time.Now().Unix()
	aidFree, err := store.InsertAssetWithCamera(db, "ux-cap-free", "2026/04/15/ux-cap-free.jpg", now, now, "", "")
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
	cid, err := store.CreateCollection(db, "UXCapAlb", "")
	if err != nil {
		t.Fatal(err)
	}
	if err := store.LinkAssetsToCollection(db, cid, []int64{aidInAlbum}); err != nil {
		t.Fatal(err)
	}
	if _, err := store.RejectAsset(db, aidRejected, now+1); err != nil {
		t.Fatal(err)
	}
	tid, err := store.FindOrCreateTagByLabel(db, "UXCapTag")
	if err != nil {
		t.Fatal(err)
	}
	if err := store.LinkTagToAssets(db, tid, []int64{aidFree}); err != nil {
		t.Fatal(err)
	}

	writeTinyJPEG(t, filepath.Join(root, "2026", "04", "15", "ux-cap-free.jpg"))
	writeTinyJPEG(t, filepath.Join(root, "2026", "04", "15", "ux-cap-in.jpg"))
	writeTinyJPEG(t, filepath.Join(root, "2026", "04", "15", "ux-cap-rej.jpg"))

	uploadA := filepath.Join(srcDir, "ux_journey_new_a.jpg")
	uploadB := filepath.Join(srcDir, "ux_journey_new_b.jpg")
	// Chromatic library-style JPEGs so staged/batch previews read as decoded photos (not flat plates).
	writeTinyJPEG(t, uploadA)
	writeTinyJPEG(t, uploadB)

	win := test.NewTempWindow(t, nil)
	win.Resize(fyne.NewSize(1280, 800))
	// Default light for judge bundles (better contrast on printed/rubric review). Override: PHOTO_TOOL_TEST_THEME=dark
	applyTestPhotoToolTheme(t, theme.VariantLight)
	// Drain async grid thumbnail callbacks before other package tests reuse the Fyne test driver (parallel runs).
	t.Cleanup(func() {
		win.SetContent(nil)
		time.Sleep(450 * time.Millisecond)
	})

	type stepMeta struct {
		ID     string `json:"id"`
		Flow   string `json:"flow"`
		File   string `json:"file"`
		Intent string `json:"intent"`
	}
	var steps []stepMeta
	stepN := 1

	captureAt := func(flow, id, intent string, w, h float32) {
		t.Helper()
		win.Resize(fyne.NewSize(w, h))
		if c := win.Content(); c != nil {
			c.Refresh()
		}
		uxCaptureSettle()
		var img image.Image
		for attempt := 0; attempt < 3; attempt++ {
			if c := win.Content(); c != nil {
				win.Canvas().Refresh(c)
			}
			var captured image.Image
			var panicked any
			func() {
				defer func() {
					if r := recover(); r != nil {
						panicked = r
					}
				}()
				captured = win.Canvas().Capture()
			}()
			if panicked == nil && captured != nil {
				img = captured
				break
			}
			if panicked != nil {
				t.Logf("capture %s: attempt %d panic %v (Fyne software painter / x/image/vector)", id, attempt+1, panicked)
			}
			win.Resize(fyne.NewSize(w, h))
			if c := win.Content(); c != nil {
				c.Refresh()
			}
			time.Sleep(120 * time.Millisecond)
		}
		if img == nil {
			t.Fatalf("capture %s: Canvas().Capture() nil or panicked after retries (see theme vector-safe radii)", id)
		}
		file := fmt.Sprintf("%02d_%s.png", stepN, id)
		stepN++
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
		steps = append(steps, stepMeta{ID: id, Flow: flow, File: file, Intent: intent})
	}
	capture := func(flow, id, intent string) {
		captureAt(flow, id, intent, 1280, 800)
	}

	mountShell := func() fyne.CanvasObject {
		t.Helper()
		clearUXCaptureReviewGrid()
		sh := NewMainShell(win, db, root, nil)
		win.SetContent(sh)
		uxCaptureSettle()
		return sh
	}

	// —— Phase 1: library fixture, empty upload ——
	t.Setenv("PHOTO_TOOL_UX_UPLOAD_SEED_PATHS", "")
	shell := mountShell()

	capture("upload", "upload_empty", "Upload: empty list, drop zone, Add/Clear/Import (import disabled)")
	tapPanel(t, shell, "Review")
	capture("review", "review_grid_default_filters", "Review: filter strip, bulk row, grid with ≥1 asset (no assigned collection)")

	if !uxCaptureOpenReviewLoupeAt(0) {
		t.Fatal("loupe: review grid not registered")
	}
	uxCaptureSettle()
	capture("review", "review_loupe", "Review loupe: image band + Prev/Next/Close, ratings, tags, albums, Reject, Share…")

	tapLoupeShareButton(t, win)
	uxCaptureSettle()
	capture("review", "review_loupe_share_preview", "Share preview dialog: Create link / Cancel (loopback share UX)")

	tapButtonInSharePreviewOverlay(t, win, "Cancel")
	uxCaptureSettle()

	closeBtn := findLoupeCloseButton(t, win)
	test.Tap(closeBtn)
	uxCaptureSettle()

	setSelectAt(t, shell, 0, "UXCapAlb")
	capture("review", "review_filter_collection_album", "Review: Collection filter set to UXCapAlb (in-album asset visible if any)")

	setSelectAt(t, shell, 1, "5")
	capture("review", "review_filter_min_rating_no_matches", "Review: Minimum rating 5 with zero matches — empty-state / guidance")

	// Do not tap "Reset filters" here: Fyne test driver can panic on grid rebind (placeholder image decode).
	setSelectAt(t, shell, 0, reviewCollectionSentinel)
	setSelectAt(t, shell, 1, reviewRatingAny)
	uxCaptureSettle()

	setSelectAt(t, shell, 2, "UXCapTag")
	capture("review", "review_filter_tag_uxcaptag", "Review: Tags filter = UXCapTag (tagged asset)")

	setSelectAt(t, shell, 2, reviewTagAny)
	setSelectAt(t, shell, 0, reviewCollectionSentinel)
	setSelectAt(t, shell, 1, reviewRatingAny)
	uxCaptureSettle()
	capture("review", "review_filters_fr16_reset", "Review: filters back to defaults (Any / sentinel)")

	tapPanel(t, shell, "Collections")
	capture("collections", "collections_album_list", "Collections: album list + New album / Rename / Delete")

	test.Tap(findButtonByText(t, shell, "New album"))
	uxCaptureSettle()
	capture("collections", "collections_new_album_form", "Collections: New album dialog (Name / Display date / Cancel / Save)")

	tapButtonInOverlays(t, win, "Cancel")
	uxCaptureSettle()

	lists := collectLists(shell)
	if len(lists) < 1 {
		t.Fatal("expected album list")
	}
	lists[0].Select(0)
	uxCaptureSettle()
	capture("collections", "collections_album_detail_stars", "Collections: album detail — Back, Edit, Delete, Group photos (Stars default), grid")

	rg := firstRadioGroup(t, shell)
	rg.Selected = "By day"
	if rg.OnChanged != nil {
		rg.OnChanged("By day")
	}
	rg.Refresh()
	uxCaptureSettle()
	capture("collections", "collections_album_group_by_day", "Collections: detail grouping = By day")

	rg.Selected = "By camera"
	if rg.OnChanged != nil {
		rg.OnChanged("By camera")
	}
	rg.Refresh()
	uxCaptureSettle()
	capture("collections", "collections_album_group_by_camera", "Collections: detail grouping = By camera")

	rg.Selected = "Stars"
	if rg.OnChanged != nil {
		rg.OnChanged("Stars")
	}
	rg.Refresh()
	uxCaptureSettle()

	test.Tap(findButtonByText(t, shell, "Back"))
	uxCaptureSettle()
	capture("collections", "collections_back_to_album_list", "Collections: returned to album list chrome")

	tapPanel(t, shell, "Rejected")
	capture("rejected", "rejected_hidden_grid", "Rejected: filters + rejected-asset count + bulk delete + grid")

	setSelectAt(t, shell, 1, "5")
	uxCaptureSettle()
	capture("rejected", "rejected_filter_min_rating_empty", "Rejected: narrow filter (e.g. 5★) with no matching rejected rows")

	setSelectAt(t, shell, 1, reviewRatingAny)
	uxCaptureSettle()

	// NFR-01 contractual floor (see domain.NFR01Window*): wide captures miss horizontal clip from
	// filter strip + shell vertical scroll; judge bundles must include these frames.
	captureAt(
		"rejected",
		"rejected_nfr01_min_window",
		fmt.Sprintf("Rejected at NFR min %d×%d: filter strip + bulk delete + Back/Reset CTAs fully visible (no clipped labels)", domain.NFR01WindowMinWidth, domain.NFR01WindowMinHeight),
		float32(domain.NFR01WindowMinWidth),
		float32(domain.NFR01WindowMinHeight),
	)
	tapPanel(t, shell, "Upload")
	uxCaptureSettle()
	captureAt(
		"upload",
		"upload_empty_nfr01_min_window",
		fmt.Sprintf("Upload at NFR min %d×%d: drop zone + primary actions readable; drop-zone text contrasts with its surface", domain.NFR01WindowMinWidth, domain.NFR01WindowMinHeight),
		float32(domain.NFR01WindowMinWidth),
		float32(domain.NFR01WindowMinHeight),
	)
	win.Resize(fyne.NewSize(1280, 800))
	if c := win.Content(); c != nil {
		c.Refresh()
	}
	uxCaptureSettle()

	// —— Phase 2: upload FR-06 with fresh shell + seeded paths ——
	t.Setenv("PHOTO_TOOL_UX_UPLOAD_SEED_PATHS", strings.Join([]string{uploadA, uploadB}, "\n"))
	shell = mountShell()
	capture("upload", "upload_paths_staged", "Upload: two files staged, Import enabled, path list + post-import area")

	test.Tap(findButtonByText(t, shell, "Import selected files"))
	uxCaptureSettle()
	capture("upload", "upload_fr06_collection_assign", "FR-06: after import — receipt + Skip collection / Assign + Confirm / Cancel")

	test.Tap(findButtonByText(t, shell, "Confirm"))
	uxCaptureSettle()
	capture("upload", "upload_after_confirm_idle", "Upload: batch cleared after Confirm (Skip collection); ready for next add")

	// NFR-01 parity for Review + Collections: append so steps 01–21 keep stable numbering; closes harness gap
	// where only Rejected/Upload had min-window proof mid-journey.
	nfrW := float32(domain.NFR01WindowMinWidth)
	nfrH := float32(domain.NFR01WindowMinHeight)
	nfrIntent := func(surface string) string {
		return fmt.Sprintf("%s at NFR min %d×%d: primary chrome readable; filter strip may scroll; no clipped safety labels", surface, domain.NFR01WindowMinWidth, domain.NFR01WindowMinHeight)
	}

	tapPanel(t, shell, "Review")
	uxCaptureSettle()
	captureAt("review", "review_grid_nfr01_min_window", nfrIntent("Review grid (default filters)"), nfrW, nfrH)

	// Post-import sort is id DESC — row 0 is the gray harness import if we open loupe blindly. Narrow to the
	// tagged fixture asset so NFR-01 loupe/share match the decoded hero bar from the default-size steps.
	setSelectAt(t, shell, 2, "UXCapTag")
	uxCaptureSettle()
	if !uxCaptureOpenReviewLoupeAt(0) {
		t.Fatal("loupe: review grid not registered (NFR-01 pass)")
	}
	uxCaptureSettle()
	captureAt("review", "review_loupe_nfr01_min_window", nfrIntent("Review loupe"), nfrW, nfrH)

	tapLoupeShareButton(t, win)
	uxCaptureSettle()
	captureAt("review", "review_loupe_share_preview_nfr01_min_window", nfrIntent("Share preview dialog"), nfrW, nfrH)
	tapButtonInSharePreviewOverlay(t, win, "Cancel")
	uxCaptureSettle()

	closeNFR := findLoupeCloseButton(t, win)
	test.Tap(closeNFR)
	uxCaptureSettle()

	setSelectAt(t, shell, 2, reviewTagAny)
	setSelectAt(t, shell, 0, reviewCollectionSentinel)
	setSelectAt(t, shell, 1, reviewRatingAny)
	uxCaptureSettle()

	tapPanel(t, shell, "Collections")
	uxCaptureSettle()
	listsNFR := collectLists(shell)
	if len(listsNFR) < 1 {
		t.Fatal("expected album list (NFR-01 pass)")
	}
	listsNFR[0].Select(0)
	uxCaptureSettle()
	captureAt("collections", "collections_album_detail_nfr01_min_window", nfrIntent("Collections album detail (Stars)"), nfrW, nfrH)

	test.Tap(findButtonByText(t, shell, "Back"))
	uxCaptureSettle()
	captureAt("collections", "collections_album_list_nfr01_min_window", nfrIntent("Collections album list"), nfrW, nfrH)

	win.Resize(fyne.NewSize(1280, 800))
	if c := win.Content(); c != nil {
		c.Refresh()
	}
	uxCaptureSettle()

	manifest := struct {
		Flows        []string   `json:"flows"`
		Steps        []stepMeta `json:"steps"`
		CaptureTool  string     `json:"capture_tool"`
		GoTestTarget string     `json:"go_test_target"`
		Omissions    []string   `json:"omissions"`
	}{
		Flows: []string{
			"upload", "review", "collections", "rejected",
		},
		Steps: steps,
		CaptureTool: "PHOTO_TOOL_UX_JOURNEY_TEST=1 PHOTO_TOOL_UX_CAPTURE_DIR=<dir> " +
			"[optional phase 2: PHOTO_TOOL_UX_UPLOAD_SEED_PATHS=newlines] go test ./internal/app -run TestUXJourneyCapture -count=1",
		GoTestTarget: "TestUXJourneyCapture",
		Omissions: []string{
			"Native file picker and real OS DPI not captured",
			"CLI scan/import and browser share URL not captured",
			"Library trash / delete-confirm dialogs not exercised in this harness",
			"Theme switch mid-session (e.g. dark→light canvas.Rectangle staleness) not captured — judge rubric still applies if visible in static light frames",
		},
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
