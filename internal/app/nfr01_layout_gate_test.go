package app

import (
	"database/sql"
	"image"
	"image/color"
	"os"
	"path/filepath"
	"strconv"
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
// canvas.Image with ImageFillContain (shipping layout path for Story 2.11 AC1).
func newNFR01GateShippingLoupeBody(t *testing.T) (fyne.CanvasObject, *canvas.Image) {
	t.Helper()
	prevBtn := widget.NewButton("← Prev", nil)
	nextBtn := widget.NewButton("Next →", nil)
	closeBtn := widget.NewButton("Close", nil)
	rejectBtn := widget.NewButton("Reject photo", nil)
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

	top := container.NewVBox(
		container.NewHBox(prevBtn, layout.NewSpacer(), ratingBox, layout.NewSpacer(), rejectBtn, deleteBtn, nextBtn, closeBtn),
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
		"← Prev", "Next →", "Close", "Reject photo", "Move to library trash…",
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
