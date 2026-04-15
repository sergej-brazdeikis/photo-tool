package app

import (
	"database/sql"
	"image"
	"image/color"
	"image/jpeg"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"

	"photo-tool/internal/config"
	"photo-tool/internal/store"
)

func writeJPEGUploadTest(t *testing.T, path string, y byte) {
	t.Helper()
	img := image.NewGray(image.Rect(0, 0, 2, 2))
	for i := range 4 {
		x := i % 2
		yy := i / 2
		img.Set(x, yy, color.Gray{Y: y})
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err := jpeg.Encode(f, img, &jpeg.Options{Quality: 80}); err != nil {
		t.Fatal(err)
	}
}

func firstRadioGroup(t *testing.T, root fyne.CanvasObject) *widget.RadioGroup {
	t.Helper()
	var found *widget.RadioGroup
	var walk func(fyne.CanvasObject)
	walk = func(x fyne.CanvasObject) {
		if x == nil || found != nil {
			return
		}
		switch v := x.(type) {
		case *widget.RadioGroup:
			found = v
		case *container.Scroll:
			walk(v.Content)
		case *fyne.Container:
			for _, ch := range v.Objects {
				walk(ch)
			}
		}
	}
	walk(root)
	if found == nil {
		t.Fatal("no RadioGroup in view")
	}
	return found
}

func firstEntry(t *testing.T, root fyne.CanvasObject) *widget.Entry {
	t.Helper()
	var out []*widget.Entry
	var walk func(fyne.CanvasObject)
	walk = func(x fyne.CanvasObject) {
		if x == nil {
			return
		}
		switch v := x.(type) {
		case *widget.Entry:
			out = append(out, v)
		case *widget.Form:
			for _, it := range v.Items {
				if it != nil {
					walk(it.Widget)
				}
			}
		case *container.Scroll:
			walk(v.Content)
		case *fyne.Container:
			for _, ch := range v.Objects {
				walk(ch)
			}
		}
	}
	walk(root)
	if len(out) == 0 {
		t.Fatal("no Entry in view")
	}
	return out[0]
}

func collectionCount(t *testing.T, db *sql.DB) int {
	t.Helper()
	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM collections`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

func linkCount(t *testing.T, db *sql.DB) int {
	t.Helper()
	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM asset_collections`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

// FR-06 / P0: Confirm with "Skip collection" does not create a collection row.
func TestUpload_flow_confirmSkipCollection_noDBCollection(t *testing.T) {
	test.NewTempApp(t)

	libRoot := filepath.Join(t.TempDir(), "lib")
	srcDir := filepath.Join(t.TempDir(), "src")
	if err := config.EnsureLibraryLayout(libRoot); err != nil {
		t.Fatal(err)
	}
	db, err := store.Open(libRoot)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	src := filepath.Join(srcDir, "one.jpg")
	mt := time.Date(2019, 3, 4, 5, 6, 7, 0, time.UTC)
	writeJPEGUploadTest(t, src, 0x2A)
	if err := os.Chtimes(src, mt, mt); err != nil {
		t.Fatal(err)
	}

	win := test.NewTempWindow(t, nil)
	view := NewUploadViewWithOptions(win, db, libRoot, UploadViewOptions{
		SeedPaths:             []string{src},
		SkipCompletionDialogs: true,
		SynchronousIngest:     true,
	})

	test.Tap(findButtonByText(t, view, "Import selected files"))
	test.Tap(findButtonByText(t, view, "Confirm"))

	if n := collectionCount(t, db); n != 0 {
		t.Fatalf("collections: got %d want 0", n)
	}
	var assetN int
	if err := db.QueryRow(`SELECT COUNT(*) FROM assets`).Scan(&assetN); err != nil {
		t.Fatal(err)
	}
	if assetN != 1 {
		t.Fatalf("assets: got %d want 1", assetN)
	}
}

// FR-06: Assign + Confirm links ingested assets to a new collection.
func TestUpload_flow_assignAndConfirm_createsCollectionAndLinks(t *testing.T) {
	test.NewTempApp(t)

	libRoot := filepath.Join(t.TempDir(), "lib")
	srcDir := filepath.Join(t.TempDir(), "src")
	if err := config.EnsureLibraryLayout(libRoot); err != nil {
		t.Fatal(err)
	}
	db, err := store.Open(libRoot)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	src := filepath.Join(srcDir, "link.jpg")
	writeJPEGUploadTest(t, src, 0x3B)
	if err := os.Chtimes(src, time.Now(), time.Now()); err != nil {
		t.Fatal(err)
	}

	win := test.NewTempWindow(t, nil)
	view := NewUploadViewWithOptions(win, db, libRoot, UploadViewOptions{
		SeedPaths:             []string{src},
		SkipCompletionDialogs: true,
		SynchronousIngest:     true,
	})

	test.Tap(findButtonByText(t, view, "Import selected files"))

	rg := firstRadioGroup(t, view)
	rg.Selected = "Assign to collection"
	if rg.OnChanged != nil {
		rg.OnChanged(rg.Selected)
	}
	rg.Refresh()

	firstEntry(t, view).SetText("FlowTestAlbum")

	test.Tap(findButtonByText(t, view, "Confirm"))

	if n := collectionCount(t, db); n != 1 {
		t.Fatalf("collections: got %d want 1", n)
	}
	if n := linkCount(t, db); n != 1 {
		t.Fatalf("asset_collections: got %d want 1", n)
	}
}

// FR-06: Cancel after import does not create a collection even if assign was chosen.
func TestUpload_flow_cancelAfterAssign_noCollection(t *testing.T) {
	test.NewTempApp(t)

	libRoot := filepath.Join(t.TempDir(), "lib")
	srcDir := filepath.Join(t.TempDir(), "src")
	if err := config.EnsureLibraryLayout(libRoot); err != nil {
		t.Fatal(err)
	}
	db, err := store.Open(libRoot)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	src := filepath.Join(srcDir, "cancel.jpg")
	writeJPEGUploadTest(t, src, 0x4C)
	if err := os.Chtimes(src, time.Now(), time.Now()); err != nil {
		t.Fatal(err)
	}

	win := test.NewTempWindow(t, nil)
	view := NewUploadViewWithOptions(win, db, libRoot, UploadViewOptions{
		SeedPaths:             []string{src},
		SkipCompletionDialogs: true,
		SynchronousIngest:     true,
	})

	test.Tap(findButtonByText(t, view, "Import selected files"))

	rg := firstRadioGroup(t, view)
	rg.Selected = "Assign to collection"
	if rg.OnChanged != nil {
		rg.OnChanged(rg.Selected)
	}
	rg.Refresh()
	firstEntry(t, view).SetText("ShouldNotExist")

	test.Tap(findButtonByText(t, view, "Cancel"))

	if n := collectionCount(t, db); n != 0 {
		t.Fatalf("collections: got %d want 0", n)
	}
}

// Journey A: after ingest, Import / Add / Clear stay disabled until Confirm or Cancel (no second batch mid-step).
func TestUpload_flow_duringCollectionStep_importAddClearDisabled(t *testing.T) {
	test.NewTempApp(t)

	libRoot := filepath.Join(t.TempDir(), "lib")
	srcDir := filepath.Join(t.TempDir(), "src")
	if err := config.EnsureLibraryLayout(libRoot); err != nil {
		t.Fatal(err)
	}
	db, err := store.Open(libRoot)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	src := filepath.Join(srcDir, "gate.jpg")
	writeJPEGUploadTest(t, src, 0x6E)
	if err := os.Chtimes(src, time.Now(), time.Now()); err != nil {
		t.Fatal(err)
	}

	win := test.NewTempWindow(t, nil)
	view := NewUploadViewWithOptions(win, db, libRoot, UploadViewOptions{
		SeedPaths:             []string{src},
		SkipCompletionDialogs: true,
		SynchronousIngest:     true,
	})

	test.Tap(findButtonByText(t, view, "Import selected files"))

	importBtn := findButtonByText(t, view, "Import selected files")
	addBtn := findButtonByText(t, view, "Add images…")
	clearBtn := findButtonByText(t, view, "Clear list")
	if !importBtn.Disabled() || !addBtn.Disabled() || !clearBtn.Disabled() {
		t.Fatalf("want Import/Add/Clear disabled during collection step; import=%v add=%v clear=%v",
			importBtn.Disabled(), addBtn.Disabled(), clearBtn.Disabled())
	}

	test.Tap(findButtonByText(t, view, "Confirm"))

	// Batch reset: list cleared — Add/Clear available again; Import stays off until there are paths.
	if addBtn.Disabled() || clearBtn.Disabled() {
		t.Fatalf("want Add/Clear enabled after confirm; add=%v clear=%v", addBtn.Disabled(), clearBtn.Disabled())
	}
	if !importBtn.Disabled() {
		t.Fatal("want Import disabled when upload list is empty after reset")
	}
}

// FR-06: All ingest failures + assign intent → no collection row (no asset IDs to link).
func TestUpload_flow_allFailedIngest_assignDoesNotCreateCollection(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("chmod 0 unreadable file behavior is Unix-specific")
	}
	test.NewTempApp(t)

	libRoot := filepath.Join(t.TempDir(), "lib")
	srcDir := filepath.Join(t.TempDir(), "src")
	if err := config.EnsureLibraryLayout(libRoot); err != nil {
		t.Fatal(err)
	}
	db, err := store.Open(libRoot)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	src := filepath.Join(srcDir, "unreadable.jpg")
	writeJPEGUploadTest(t, src, 0x5D)
	if err := os.Chmod(src, 0); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(src, 0o644) })

	win := test.NewTempWindow(t, nil)
	view := NewUploadViewWithOptions(win, db, libRoot, UploadViewOptions{
		SeedPaths:             []string{src},
		SkipCompletionDialogs: true,
		SynchronousIngest:     true,
	})

	test.Tap(findButtonByText(t, view, "Import selected files"))

	rg := firstRadioGroup(t, view)
	rg.Selected = "Assign to collection"
	if rg.OnChanged != nil {
		rg.OnChanged(rg.Selected)
	}
	rg.Refresh()
	firstEntry(t, view).SetText("NoAssetsAlbum")

	test.Tap(findButtonByText(t, view, "Confirm"))

	if n := collectionCount(t, db); n != 0 {
		t.Fatalf("collections: got %d want 0", n)
	}
}

// AC4: User renames the default collection label before confirm → persisted name matches input.
func TestUpload_flow_renameFromDefault_persistsCollectionName(t *testing.T) {
	test.NewTempApp(t)

	libRoot := filepath.Join(t.TempDir(), "lib")
	srcDir := filepath.Join(t.TempDir(), "src")
	if err := config.EnsureLibraryLayout(libRoot); err != nil {
		t.Fatal(err)
	}
	db, err := store.Open(libRoot)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	src := filepath.Join(srcDir, "rename.jpg")
	writeJPEGUploadTest(t, src, 0x81)
	if err := os.Chtimes(src, time.Now(), time.Now()); err != nil {
		t.Fatal(err)
	}

	win := test.NewTempWindow(t, nil)
	view := NewUploadViewWithOptions(win, db, libRoot, UploadViewOptions{
		SeedPaths:             []string{src},
		SkipCompletionDialogs: true,
		SynchronousIngest:     true,
	})

	test.Tap(findButtonByText(t, view, "Import selected files"))

	rg := firstRadioGroup(t, view)
	rg.Selected = "Assign to collection"
	if rg.OnChanged != nil {
		rg.OnChanged(rg.Selected)
	}
	rg.Refresh()

	ent := firstEntry(t, view)
	if !strings.HasPrefix(ent.Text, "Upload ") {
		t.Fatalf("want default Upload YYYYMMDD, got %q", ent.Text)
	}
	ent.SetText("RenamedFromDefault")

	test.Tap(findButtonByText(t, view, "Confirm"))

	var got string
	if err := db.QueryRow(`SELECT name FROM collections ORDER BY id DESC LIMIT 1`).Scan(&got); err != nil {
		t.Fatal(err)
	}
	if got != "RenamedFromDefault" {
		t.Fatalf("collection name: got %q want RenamedFromDefault", got)
	}
}

// Duplicate paths in one batch resolve to one asset; confirm still links the album (FR-04 / junction idempotency).
func TestUpload_flow_duplicatePathsInBatch_singleLinkRow(t *testing.T) {
	test.NewTempApp(t)

	libRoot := filepath.Join(t.TempDir(), "lib")
	srcDir := filepath.Join(t.TempDir(), "src")
	if err := config.EnsureLibraryLayout(libRoot); err != nil {
		t.Fatal(err)
	}
	db, err := store.Open(libRoot)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	src1 := filepath.Join(srcDir, "one.jpg")
	src2 := filepath.Join(srcDir, "two.jpg")
	writeJPEGUploadTest(t, src1, 0x91)
	b, err := os.ReadFile(src1)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(src2, b, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(src2, time.Now(), time.Now()); err != nil {
		t.Fatal(err)
	}

	win := test.NewTempWindow(t, nil)
	view := NewUploadViewWithOptions(win, db, libRoot, UploadViewOptions{
		SeedPaths:             []string{src1, src2},
		SkipCompletionDialogs: true,
		SynchronousIngest:     true,
	})

	test.Tap(findButtonByText(t, view, "Import selected files"))

	rg := firstRadioGroup(t, view)
	rg.Selected = "Assign to collection"
	if rg.OnChanged != nil {
		rg.OnChanged(rg.Selected)
	}
	rg.Refresh()
	firstEntry(t, view).SetText("DupBatchAlbum")

	test.Tap(findButtonByText(t, view, "Confirm"))

	var assetsN, linksN int
	if err := db.QueryRow(`SELECT COUNT(*) FROM assets`).Scan(&assetsN); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow(`SELECT COUNT(*) FROM asset_collections`).Scan(&linksN); err != nil {
		t.Fatal(err)
	}
	if assetsN != 1 {
		t.Fatalf("assets: got %d want 1", assetsN)
	}
	if linksN != 1 {
		t.Fatalf("asset_collections: got %d want 1", linksN)
	}
}
