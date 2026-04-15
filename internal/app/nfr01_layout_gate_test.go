package app

import (
	"bytes"
	"database/sql"
	"image"
	"image/color"
	"image/jpeg"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"photo-tool/internal/config"
	"photo-tool/internal/domain"
	"photo-tool/internal/store"
)

func assertObjectInWindowCanvas(t *testing.T, win fyne.Window, obj fyne.CanvasObject) {
	t.Helper()
	if obj == nil {
		t.Fatal("nil object")
	}
	drv := fyne.CurrentApp().Driver()
	pos := drv.AbsolutePositionForObject(obj)
	sz := obj.Size()
	csz := win.Canvas().Size()
	const tol float32 = 4
	if sz.Width < 2 || sz.Height < 2 {
		t.Fatalf("near-zero size %v for object", sz)
	}
	if pos.X < -tol || pos.Y < -tol {
		t.Fatalf("negative position %v (size %v)", pos, sz)
	}
	if pos.X+sz.Width > csz.Width+tol || pos.Y+sz.Height > csz.Height+tol {
		t.Fatalf("outside canvas: pos=%v size=%v canvas=%v", pos, sz, csz)
	}
}

// nfr01GateTestPatternImage returns a wide landscape raster so ImageFillContain letterboxes
// inside the ~90% loupe band (AC1: no unintended crop — decoded image path, not a solid rectangle).
func nfr01GateTestPatternImage(t *testing.T) image.Image {
	t.Helper()
	const w, h = 800, 200
	rgba := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			rgba.Set(x, y, color.NRGBA{R: uint8(x * 255 / max(w-1, 1)), G: uint8(y * 255 / max(h-1, 1)), B: 120, A: 255})
		}
	}
	return rgba
}

// newNFR01GateShippingLoupeBody mirrors review_loupe.go modal chrome + loupeImageLayout +
// canvas.Image with ImageFillContain (shipping layout path for Story 2.11 AC1), including
// the rating-row HBox order through Share… → Reject → Delete → Next → Close.
func newNFR01GateShippingLoupeBody(t *testing.T) (fyne.CanvasObject, *canvas.Image) {
	t.Helper()
	prevBtn := widget.NewButton("← Prev", nil)
	nextBtn := widget.NewButton("Next →", nil)
	closeBtn := widget.NewButton("Close", nil)
	rejectBtn := widget.NewButton("Reject photo", nil)
	shareBtn := widget.NewButton("Share…", nil)
	deleteBtn := widget.NewButton("Move to library trash…", nil)
	ratingBox := container.NewHBox()
	for i := 1; i <= 5; i++ {
		ratingBox.Add(widget.NewButton(strconv.Itoa(i)+"★", nil))
	}
	tagEntry := widget.NewSelectEntry([]string{"sample"})
	tagAdd := widget.NewButton("Add tag", nil)
	tagRem := widget.NewButton("Remove tag", nil)
	tagsLbl := widget.NewLabel("Tags: —")
	newAlbumLoupeBtn := widget.NewButton("New album…", nil)
	albumChecksBox := container.NewVBox(widget.NewLabel("No albums yet — use New album…"))
	albumScroll := container.NewVScroll(albumChecksBox)
	albumScroll.SetMinSize(fyne.NewSize(80, 100))
	albumHeader := container.NewHBox(widget.NewLabel("Albums"), layout.NewSpacer(), newAlbumLoupeBtn)

	// Rating row order must match review_loupe.go (Share sits between rating cluster and Reject for min-width stress).
	top := container.NewVBox(
		container.NewHBox(prevBtn, layout.NewSpacer(), ratingBox, layout.NewSpacer(), shareBtn, rejectBtn, deleteBtn, nextBtn, closeBtn),
		container.NewHBox(tagEntry, tagAdd, tagRem),
		tagsLbl,
		widget.NewSeparator(),
		albumHeader,
		albumScroll,
	)

	cimg := canvas.NewImageFromImage(nfr01GateTestPatternImage(t))
	cimg.FillMode = canvas.ImageFillContain
	errLbl := widget.NewLabel("")
	errLbl.Hide()
	imgStack := container.NewStack(cimg, container.NewCenter(errLbl))
	imgArea := container.New(&loupeImageLayout{}, imgStack)
	return container.NewBorder(top, nil, nil, nil, imgArea), cimg
}

func findLabelByTextDeep(t *testing.T, root fyne.CanvasObject, want string) *widget.Label {
	t.Helper()
	for _, lb := range collectLabelsDeep(root) {
		if lb.Text == want {
			return lb
		}
	}
	t.Fatalf("label %q not found (deep walk)", want)
	return nil
}

func writeTinyJPEG(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	img := image.NewRGBA(image.Rect(0, 0, 4, 4))
	for y := 0; y < 4; y++ {
		for x := 0; x < 4; x++ {
			img.Set(x, y, color.NRGBA{R: 200, G: 100, B: 50, A: 255})
		}
	}
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 80}); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, buf.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
}

func collectLists(o fyne.CanvasObject) []*widget.List {
	var out []*widget.List
	var walk func(fyne.CanvasObject)
	walk = func(x fyne.CanvasObject) {
		if x == nil {
			return
		}
		switch v := x.(type) {
		case *widget.List:
			out = append(out, v)
		case *container.Scroll:
			walk(v.Content)
		case *fyne.Container:
			for _, ch := range v.Objects {
				walk(ch)
			}
		}
	}
	walk(o)
	return out
}

func assertPrimaryNavVisible(t *testing.T, win fyne.Window, shell fyne.CanvasObject) {
	t.Helper()
	for _, label := range PrimaryNavLabels() {
		b := findButtonByText(t, shell, label)
		assertObjectInWindowCanvas(t, win, b)
	}
}

func assertReviewFilterStripOnScreen(t *testing.T, win fyne.Window, shell fyne.CanvasObject) {
	t.Helper()
	sels := collectSelectWidgets(shell)
	if len(sels) < 3 {
		t.Fatalf("expected ≥3 Select (filter strip), got %d", len(sels))
	}
	strip := sels[:3]
	for _, s := range strip {
		assertObjectInWindowCanvas(t, win, s)
	}
	drv := fyne.CurrentApp().Driver()
	p0 := drv.AbsolutePositionForObject(strip[0])
	p2 := drv.AbsolutePositionForObject(strip[2])
	if p0.X > p2.X {
		t.Fatal("filter strip selects not left-to-right")
	}
}

// assertReviewBulkActionsOnScreen checks primary Epic 2 bulk affordances stay on the window canvas
// (Story 2.11 AC1 — grid may scroll internally; shell must not hide these controls).
func assertReviewBulkActionsOnScreen(t *testing.T, win fyne.Window, shell fyne.CanvasObject) {
	t.Helper()
	for _, label := range []string{
		"Reject selected photos",
		"Delete selected…",
		"Add tag to selection",
		"Assign selection to album",
	} {
		b := findButtonByText(t, shell, label)
		assertObjectInWindowCanvas(t, win, b)
	}
}

func assertNFR01GateLoupeChromeOnScreen(t *testing.T, win fyne.Window, content fyne.CanvasObject, cimg *canvas.Image) {
	t.Helper()
	if cimg.FillMode != canvas.ImageFillContain {
		t.Fatalf("loupe image fill mode: got %v want ImageFillContain", cimg.FillMode)
	}
	for _, text := range []string{
		"← Prev", "Next →", "Close", "Share…", "Reject photo", "Move to library trash…",
		"1★", "5★",
		"Add tag", "Remove tag", "New album…",
	} {
		b := findButtonByText(t, content, text)
		assertObjectInWindowCanvas(t, win, b)
	}
	assertObjectInWindowCanvas(t, win, findLabelByText(t, content, "Albums"))
	assertObjectInWindowCanvas(t, win, cimg)
}

func tapPanel(t *testing.T, shell fyne.CanvasObject, label string) {
	t.Helper()
	b := findButtonByText(t, shell, label)
	if b.OnTapped == nil {
		t.Fatalf("button %q has nil OnTapped", label)
	}
	b.OnTapped()
}

func themeVariantName(v fyne.ThemeVariant) string {
	if v == theme.VariantLight {
		return "light"
	}
	return "dark"
}

// exerciseNFR01NonReviewRoutes asserts Collections album-detail browsing chrome and Rejected
// filter/grid chrome at Story 2.11 corner sizes (S-min + 169-min) — structural CI only; real
// assets and OS scaling remain in the human matrix where noted in nfr-01-layout-matrix-evidence.md.
func exerciseNFR01NonReviewRoutes(t *testing.T, db *sql.DB, root string, w, h int, variant fyne.ThemeVariant) {
	t.Helper()
	win := test.NewTempWindow(t, nil)
	win.Resize(fyne.NewSize(float32(w), float32(h)))
	test.ApplyTheme(t, NewPhotoToolTheme(variant))
	shell := newMainShell(win, db, root, false, nil)
	win.SetContent(shell)

	tapPanel(t, shell, "Collections")
	assertPrimaryNavVisible(t, win, shell)
	lists := collectLists(shell)
	if len(lists) < 1 {
		t.Fatal("expected album list on Collections panel")
	}
	lists[0].Select(0)
	assertPrimaryNavVisible(t, win, shell)
	findButtonByText(t, shell, "Back")
	findButtonByText(t, shell, "Edit album")
	findButtonByText(t, shell, "Delete album…")
	findLabelByTextDeep(t, shell, "NFR11GateAlbum")
	findLabelByTextDeep(t, shell, "Unrated")
	assertNFR01GateThumbnailGridListsOnCanvas(t, win, shell, "collection detail")

	tapPanel(t, shell, "Rejected")
	assertPrimaryNavVisible(t, win, shell)
	assertReviewFilterStripOnScreen(t, win, shell)
	findButtonByText(t, shell, "Delete selected…")
	var sawHiddenCount bool
	for _, lb := range collectLabelsDeep(shell) {
		if strings.HasPrefix(lb.Text, "Hidden assets: ") && strings.Contains(lb.Text, "1") {
			sawHiddenCount = true
			break
		}
	}
	if !sawHiddenCount {
		t.Fatal(`expected Rejected count line to include hidden asset count "1"`)
	}
	assertNFR01GateThumbnailGridListsOnCanvas(t, win, shell, "Rejected")
}

// assertNFR01GateThumbnailGridListsOnCanvas is a structural UX-DR16 supplement: the browsing grid
// List must appear on the window canvas (not a numeric “majority” rubric — that stays manual).
func assertNFR01GateThumbnailGridListsOnCanvas(t *testing.T, win fyne.Window, shell fyne.CanvasObject, where string) {
	t.Helper()
	const minDim float32 = 32
	lists := collectLists(shell)
	if len(lists) < 1 {
		t.Fatalf("%s: expected ≥1 widget.List (thumbnail grid), got 0", where)
	}
	var onCanvas int
	for _, li := range lists {
		sz := li.Size()
		if sz.Width < minDim || sz.Height < minDim {
			continue
		}
		assertObjectInWindowCanvas(t, win, li)
		onCanvas++
	}
	if onCanvas < 1 {
		t.Fatalf("%s: no thumbnail grid List on canvas with size ≥%.0f×%.0f (got %d lists)", where, minDim, minDim, len(lists))
	}
}

func exerciseNFR01MatrixCell(t *testing.T, db *sql.DB, root string, cell domain.NFR01MatrixCell, variant fyne.ThemeVariant) {
	t.Helper()
	win := test.NewTempWindow(t, nil)
	win.Resize(fyne.NewSize(float32(cell.Width), float32(cell.Height)))
	test.ApplyTheme(t, NewPhotoToolTheme(variant))

	if cell.IsLoupe {
		content, cimg := newNFR01GateShippingLoupeBody(t)
		win.SetContent(content)
		assertNFR01GateLoupeChromeOnScreen(t, win, content, cimg)
		return
	}

	// Full shipping shell including Story 2.1 semantic style preview strip (AC1).
	shell := newMainShell(win, db, root, false, nil)
	win.SetContent(shell)
	tapPanel(t, shell, "Review")
	assertPrimaryNavVisible(t, win, shell)
	assertReviewFilterStripOnScreen(t, win, shell)
	assertReviewBulkActionsOnScreen(t, win, shell)
	// Story 2.12 AC6 / NFR-01: library-empty primary CTA must stay on-screen at matrix sizes.
	assertObjectInWindowCanvas(t, win, findButtonByText(t, shell, "Go to Upload"))

	// Non-Review hops: primary nav on-canvas only here. Collections **detail** + **Rejected**
	// grid chrome are structurally asserted in `TestNFR01LayoutGate_nonReviewRoutes_collectionsDetailAndRejected`;
	// Upload flow chrome and UX-DR16 numeric thresholds remain manual / evidence notes.
	for _, panel := range []string{"Upload", "Collections", "Rejected"} {
		tapPanel(t, shell, panel)
		assertPrimaryNavVisible(t, win, shell)
	}
	tapPanel(t, shell, "Review")
	assertReviewFilterStripOnScreen(t, win, shell)
	assertReviewBulkActionsOnScreen(t, win, shell)
	assertObjectInWindowCanvas(t, win, findButtonByText(t, shell, "Go to Upload"))
}

func exerciseNFR07Epic2DefaultSubsetAndAC2(t *testing.T, db *sql.DB, root string) {
	t.Helper()
	subset := make(map[string]struct{})
	for _, id := range domain.NFR07Epic2DefaultSubsetCellIDs() {
		subset[id] = struct{}{}
	}
	for _, cell := range domain.NFR01Epic2MatrixCells() {
		if _, ok := subset[cell.CellID]; !ok {
			continue
		}
		cell := cell
		for _, variant := range []fyne.ThemeVariant{theme.VariantDark, theme.VariantLight} {
			variant := variant
			name := cell.CellID + "_" + themeVariantName(variant)
			t.Run(name, func(t *testing.T) {
				exerciseNFR01MatrixCell(t, db, root, cell, variant)
			})
		}
	}
	t.Run("AC2_resize_sweep", func(t *testing.T) {
		for _, variant := range []fyne.ThemeVariant{theme.VariantDark, theme.VariantLight} {
			variant := variant
			t.Run(themeVariantName(variant), func(t *testing.T) {
				exerciseNFR01ResizeSweepAC2(t, db, root, variant)
			})
		}
	})
}

func exerciseNFR01ResizeSweepAC2(t *testing.T, db *sql.DB, root string, variant fyne.ThemeVariant) {
	t.Helper()
	win := test.NewTempWindow(t, nil)
	test.ApplyTheme(t, NewPhotoToolTheme(variant))
	shell := newMainShell(win, db, root, false, nil)
	win.SetContent(shell)

	for _, wh := range domain.NFR01AC2ResizeSweepPath() {
		sz := fyne.NewSize(float32(wh[0]), float32(wh[1]))
		win.Resize(sz)
		// AC2 / evidence protocol: idle dwell after each resize so transient layout can settle.
		time.Sleep(1100 * time.Millisecond)

		win.SetContent(shell)
		tapPanel(t, shell, "Review")
		assertPrimaryNavVisible(t, win, shell)
		assertReviewFilterStripOnScreen(t, win, shell)
		assertReviewBulkActionsOnScreen(t, win, shell)
		assertObjectInWindowCanvas(t, win, findButtonByText(t, shell, "Go to Upload"))

		loupeContent, cimg := newNFR01GateShippingLoupeBody(t)
		win.SetContent(loupeContent)
		assertNFR01GateLoupeChromeOnScreen(t, win, loupeContent, cimg)
	}
}

func TestNFR01LayoutGate_nonReviewRoutes_collectionsDetailAndRejected(t *testing.T) {
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
	aidInAlbum, err := store.InsertAssetWithCamera(db, "nfr11-gate-in-album", "2026/04/15/gate-in.jpg", now, now, "", "")
	if err != nil {
		t.Fatal(err)
	}
	aidRejected, err := store.InsertAssetWithCamera(db, "nfr11-gate-rejected", "2026/04/15/gate-rej.jpg", now, now, "", "")
	if err != nil {
		t.Fatal(err)
	}
	cid, err := store.CreateCollection(db, "NFR11GateAlbum", "")
	if err != nil {
		t.Fatal(err)
	}
	if err := store.LinkAssetsToCollection(db, cid, []int64{aidInAlbum}); err != nil {
		t.Fatal(err)
	}
	if _, err := store.RejectAsset(db, aidRejected, now+1); err != nil {
		t.Fatal(err)
	}

	writeTinyJPEG(t, filepath.Join(root, "2026", "04", "15", "gate-in.jpg"))
	writeTinyJPEG(t, filepath.Join(root, "2026", "04", "15", "gate-rej.jpg"))

	corners := []struct {
		name string
		w, h int
	}{
		{"S-min", 1024, 1024},
		{"169-min", 1366, 768},
	}
	for _, sz := range corners {
		sz := sz
		for _, variant := range []fyne.ThemeVariant{theme.VariantDark, theme.VariantLight} {
			variant := variant
			t.Run(sz.name+"_"+themeVariantName(variant), func(t *testing.T) {
				exerciseNFR01NonReviewRoutes(t, db, root, sz.w, sz.h, variant)
			})
		}
	}
}

func TestNFR01LayoutGate_UXDR19_reviewFilterStripTabAtSMin(t *testing.T) {
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

	for _, variant := range []fyne.ThemeVariant{theme.VariantDark, theme.VariantLight} {
		variant := variant
		t.Run(themeVariantName(variant), func(t *testing.T) {
			win := test.NewTempWindow(t, nil)
			win.Resize(fyne.NewSize(1024, 1024))
			test.ApplyTheme(t, NewPhotoToolTheme(variant))
			shell := newMainShell(win, db, root, false, nil)
			win.SetContent(shell)
			tapPanel(t, shell, "Review")

			sels := collectSelectWidgets(shell)
			if len(sels) < 3 {
				t.Fatalf("expected ≥3 Select (filter strip), got %d", len(sels))
			}
			strip := sels[:3]
			c := win.Canvas()
			// Shell adds focusable nav buttons before Review; Tab until the first strip Select is focused.
			var atStrip bool
			for step := 0; step < 48; step++ {
				if f := c.Focused(); f == strip[0] {
					atStrip = true
					break
				}
				test.FocusNext(c)
			}
			if !atStrip {
				t.Fatal("Tab order never reached filter strip first Select within step budget (UX-DR19 shell prelude)")
			}
			for i := range strip {
				if f := c.Focused(); f != strip[i] {
					t.Fatalf("strip focus mismatch at %d: got %T want strip[%d]", i, f, i)
				}
				if i < len(strip)-1 {
					test.FocusNext(c)
				}
			}
		})
	}
}

func TestNFR01LayoutGate_matrixCells(t *testing.T) {
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

	for _, cell := range domain.NFR01Epic2MatrixCells() {
		cell := cell
		for _, variant := range []fyne.ThemeVariant{theme.VariantDark, theme.VariantLight} {
			variant := variant
			name := cell.CellID + "_" + themeVariantName(variant)
			t.Run(name, func(t *testing.T) {
				exerciseNFR01MatrixCell(t, db, root, cell, variant)
			})
		}
	}
}

func TestNFR01LayoutGate_resizeSweep_AC2(t *testing.T) {
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

	for _, variant := range []fyne.ThemeVariant{theme.VariantDark, theme.VariantLight} {
		variant := variant
		t.Run(themeVariantName(variant), func(t *testing.T) {
			exerciseNFR01ResizeSweepAC2(t, db, root, variant)
		})
	}
}

// TestNFR01LayoutGate_NFR07FYNEProxy re-runs the Epic 2 default NFR-07 subset plus the AC2
// sweep with FYNE_SCALE set. This exercises HiDPI-style logical scaling in the Fyne test
// driver; OS Settings display scale should still be spot-checked on real hardware when feasible.
func TestNFR01LayoutGate_NFR07FYNEProxy(t *testing.T) {
	for _, scale := range []string{"1.25", "1.5"} {
		scale := scale
		t.Run("FYNE_SCALE_"+scale, func(t *testing.T) {
			t.Setenv("FYNE_SCALE", scale)
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

			exerciseNFR07Epic2DefaultSubsetAndAC2(t, db, root)
		})
	}
}

// TestNFR01LayoutGate_NFR07_AC3 re-runs the Epic 2 NFR-07 subset using the OS-reported
// display scaling tier (FYNE_SCALE unset), except on GitHub Actions macOS where the workflow
// pins FYNE_SCALE to match PHOTO_TOOL_NFR07_MACOS_CI_TIER (runners cannot drive Displays scaling).
// On darwin+cgo, NFR07AC3DisplayScalingPercent still records CoreGraphics in the log line.
// Satisfies Story 2.11 AC3 when this test is not skipped; see nfr-07-os-scaling-checklist.md.
func TestNFR01LayoutGate_NFR07_AC3(t *testing.T) {
	if tier, fyneWant, ok := nfr07AC3DarwinCISurrogate(); ok {
		if got := os.Getenv("FYNE_SCALE"); got != fyneWant {
			t.Fatalf("NFR-07 AC3 macOS CI: FYNE_SCALE must be %q for tier %d%% (got %q)", fyneWant, tier, got)
		}
		pct, detail, okPct := NFR07AC3DisplayScalingPercent()
		if !okPct || pct != tier {
			t.Fatalf("NFR-07 AC3 macOS CI: NFR07AC3DisplayScalingPercent ok=%v pct=%d want tier=%d detail=%q", okPct, pct, tier, detail)
		}
		t.Logf("NFR-07 AC3: tier=%d%% (%s)", pct, detail)

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

		exerciseNFR07Epic2DefaultSubsetAndAC2(t, db, root)
		return
	}

	prevFYNE, hadFYNE := os.LookupEnv("FYNE_SCALE")
	t.Cleanup(func() {
		if hadFYNE {
			_ = os.Setenv("FYNE_SCALE", prevFYNE)
		} else {
			_ = os.Unsetenv("FYNE_SCALE")
		}
	})
	_ = os.Unsetenv("FYNE_SCALE")

	pct, detail, ok := NFR07AC3DisplayScalingPercent()
	if !ok {
		t.Skipf("NFR-07 AC3: host display scaling is not in the 125%%/150%% tier (%s). Rerun on macOS or Windows with System Settings display scaling at 125%% or 150%%, use the GitHub Actions macOS matrix (FYNE_SCALE + PHOTO_TOOL_NFR07_MACOS_CI_TIER), or apply the Windows CI LogPixels matrix step.", detail)
	}
	t.Logf("NFR-07 AC3: tier=%d%% (%s)", pct, detail)

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

	exerciseNFR07Epic2DefaultSubsetAndAC2(t, db, root)
}
