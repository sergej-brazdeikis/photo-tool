package ingest

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"photo-tool/internal/config"
	"photo-tool/internal/domain"
	"photo-tool/internal/store"
)

func TestRegisterInPlacePath_addsRowUnderLibrary(t *testing.T) {
	libRoot := filepath.Join(t.TempDir(), "lib")
	if err := config.EnsureLibraryLayout(libRoot); err != nil {
		t.Fatal(err)
	}
	day := filepath.Join(libRoot, "2021", "03", "04")
	if err := os.MkdirAll(day, 0o755); err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(day, "inplace.jpg")
	mt := time.Date(2021, 3, 4, 5, 6, 7, 0, time.UTC)
	if err := writeJPEGGray(p, 0x77); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(p, mt, mt); err != nil {
		t.Fatal(err)
	}

	db, err := store.Open(libRoot)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	var sum domain.OperationSummary
	RegisterInPlacePath(db, libRoot, p, &sum, false)
	if sum.Added != 1 || sum.SkippedDuplicate != 0 || sum.Updated != 0 || sum.Failed != 0 {
		t.Fatalf("summary: %+v", sum)
	}
	if n := assetCount(t, db); n != 1 {
		t.Fatalf("rows: %d", n)
	}
}

func TestRegisterInPlacePath_secondPass_skipsDuplicate(t *testing.T) {
	libRoot := filepath.Join(t.TempDir(), "lib")
	if err := config.EnsureLibraryLayout(libRoot); err != nil {
		t.Fatal(err)
	}
	day := filepath.Join(libRoot, "2020", "08", "09")
	if err := os.MkdirAll(day, 0o755); err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(day, "dup.jpg")
	mt := time.Date(2020, 8, 9, 10, 11, 12, 0, time.UTC)
	if err := writeJPEGGray(p, 0x55); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(p, mt, mt); err != nil {
		t.Fatal(err)
	}

	db, err := store.Open(libRoot)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	var s1 domain.OperationSummary
	RegisterInPlacePath(db, libRoot, p, &s1, false)
	if s1.Added != 1 {
		t.Fatalf("first: %+v", s1)
	}
	var s2 domain.OperationSummary
	RegisterInPlacePath(db, libRoot, p, &s2, false)
	if s2.SkippedDuplicate != 1 || s2.Added != 0 {
		t.Fatalf("second: %+v", s2)
	}
}

func TestRegisterInPlacePath_dryRun_noInsert(t *testing.T) {
	libRoot := filepath.Join(t.TempDir(), "lib")
	if err := config.EnsureLibraryLayout(libRoot); err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(libRoot, "flat.jpg")
	mt := time.Date(2018, 2, 3, 4, 5, 6, 0, time.UTC)
	if err := writeJPEGGray(p, 0x44); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(p, mt, mt); err != nil {
		t.Fatal(err)
	}

	db, err := store.Open(libRoot)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	var sum domain.OperationSummary
	RegisterInPlacePath(db, libRoot, p, &sum, true)
	if sum.Added != 1 || sum.Failed != 0 {
		t.Fatalf("summary: %+v", sum)
	}
	if n := assetCount(t, db); n != 0 {
		t.Fatalf("dry-run inserted: %d", n)
	}
}

func TestRegisterInPlacePath_backfillCaptureTime(t *testing.T) {
	libRoot := filepath.Join(t.TempDir(), "lib")
	if err := config.EnsureLibraryLayout(libRoot); err != nil {
		t.Fatal(err)
	}
	day := filepath.Join(libRoot, "2019", "10", "11")
	if err := os.MkdirAll(day, 0o755); err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(day, "meta.jpg")
	mt := time.Date(2019, 10, 11, 12, 13, 14, 0, time.UTC)
	if err := writeJPEGGray(p, 0x99); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(p, mt, mt); err != nil {
		t.Fatal(err)
	}

	db, err := store.Open(libRoot)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	var s0 domain.OperationSummary
	RegisterInPlacePath(db, libRoot, p, &s0, false)
	if s0.Added != 1 {
		t.Fatalf("seed: %+v", s0)
	}

	wrong := int64(111)
	_, err = db.Exec(`UPDATE assets SET capture_time_unix = ? WHERE rel_path = ?`, wrong, "2019/10/11/meta.jpg")
	if err != nil {
		t.Fatal(err)
	}

	var s1 domain.OperationSummary
	RegisterInPlacePath(db, libRoot, p, &s1, false)
	if s1.Updated != 1 || s1.SkippedDuplicate != 0 || s1.Added != 0 {
		t.Fatalf("backfill: %+v", s1)
	}

	var cap int64
	if err := db.QueryRow(`SELECT capture_time_unix FROM assets WHERE rel_path = ?`, "2019/10/11/meta.jpg").Scan(&cap); err != nil {
		t.Fatal(err)
	}
	if cap != mt.Unix() {
		t.Fatalf("capture: got %d want %d", cap, mt.Unix())
	}
}

func TestRegisterInPlacePath_softDeletedRowReservesHash(t *testing.T) {
	libRoot := filepath.Join(t.TempDir(), "lib")
	if err := config.EnsureLibraryLayout(libRoot); err != nil {
		t.Fatal(err)
	}
	a := filepath.Join(libRoot, "a.jpg")
	b := filepath.Join(libRoot, "b.jpg")
	mt := time.Date(2015, 4, 3, 2, 1, 0, 0, time.UTC)
	if err := writeJPEGGray(a, 0x71); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(a, mt, mt); err != nil {
		t.Fatal(err)
	}
	if err := copyFile(t, a, b); err != nil {
		t.Fatal(err)
	}

	db, err := store.Open(libRoot)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	var s1 domain.OperationSummary
	RegisterInPlacePath(db, libRoot, a, &s1, false)
	if s1.Added != 1 {
		t.Fatalf("seed: %+v", s1)
	}
	if _, err := db.Exec(`UPDATE assets SET deleted_at_unix = 1 WHERE rel_path = ?`, "a.jpg"); err != nil {
		t.Fatal(err)
	}

	var s2 domain.OperationSummary
	RegisterInPlacePath(db, libRoot, b, &s2, false)
	if s2.SkippedDuplicate != 1 || s2.Added != 0 || s2.Failed != 0 {
		t.Fatalf("tombstone still holds content_hash globally: %+v", s2)
	}
	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM assets WHERE deleted_at_unix IS NULL`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Fatalf("active rows: %d", n)
	}
}

func TestRegisterInPlacePath_sameHashOtherPath_skipsDuplicate(t *testing.T) {
	libRoot := filepath.Join(t.TempDir(), "lib")
	if err := config.EnsureLibraryLayout(libRoot); err != nil {
		t.Fatal(err)
	}
	a := filepath.Join(libRoot, "a.jpg")
	b := filepath.Join(libRoot, "b.jpg")
	mt := time.Date(2017, 1, 2, 3, 4, 5, 0, time.UTC)
	if err := writeJPEGGray(a, 0x31); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(a, mt, mt); err != nil {
		t.Fatal(err)
	}
	if err := copyFile(t, a, b); err != nil {
		t.Fatal(err)
	}

	db, err := store.Open(libRoot)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	var s1 domain.OperationSummary
	RegisterInPlacePath(db, libRoot, a, &s1, false)
	if s1.Added != 1 {
		t.Fatalf("first: %+v", s1)
	}
	var s2 domain.OperationSummary
	RegisterInPlacePath(db, libRoot, b, &s2, false)
	if s2.SkippedDuplicate != 1 || s2.Added != 0 {
		t.Fatalf("second path same bytes: %+v", s2)
	}
}

func copyFile(t *testing.T, src, dst string) error {
	t.Helper()
	b, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, b, 0o644)
}

func TestRegisterInPlacePath_replacedFileAtPath_fails(t *testing.T) {
	libRoot := filepath.Join(t.TempDir(), "lib")
	if err := config.EnsureLibraryLayout(libRoot); err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(libRoot, "replaced.jpg")
	mt := time.Date(2016, 5, 4, 3, 2, 1, 0, time.UTC)
	if err := writeJPEGGray(p, 0x21); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(p, mt, mt); err != nil {
		t.Fatal(err)
	}

	db, err := store.Open(libRoot)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	var s1 domain.OperationSummary
	RegisterInPlacePath(db, libRoot, p, &s1, false)
	if s1.Added != 1 {
		t.Fatalf("seed: %+v", s1)
	}

	if err := writeJPEGGray(p, 0xEF); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(p, mt, mt); err != nil {
		t.Fatal(err)
	}

	var s2 domain.OperationSummary
	RegisterInPlacePath(db, libRoot, p, &s2, false)
	if s2.Failed != 1 || s2.Added != 0 {
		t.Fatalf("replaced content: %+v", s2)
	}
}

func TestRegisterInPlace_dryRun_matchesLiveSeparateLibraries(t *testing.T) {
	libRoot := filepath.Join(t.TempDir(), "lib-tree")
	if err := config.EnsureLibraryLayout(libRoot); err != nil {
		t.Fatal(err)
	}
	sub := filepath.Join(libRoot, "nested")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	mt := time.Date(2015, 6, 7, 8, 9, 0, 0, time.UTC)
	files := []struct {
		rel  string
		gray byte
	}{
		{"root.jpg", 0x10},
		{filepath.Join("nested", "x.jpg"), 0x11},
	}
	for _, f := range files {
		p := filepath.Join(libRoot, f.rel)
		if err := writeJPEGGray(p, f.gray); err != nil {
			t.Fatal(err)
		}
		if err := os.Chtimes(p, mt, mt); err != nil {
			t.Fatal(err)
		}
	}

	libDry := filepath.Join(t.TempDir(), "lib-dry-import")
	libLive := filepath.Join(t.TempDir(), "lib-live-import")
	for _, lr := range []string{libDry, libLive} {
		if err := os.MkdirAll(lr, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := copyTree(t, libRoot, lr); err != nil {
			t.Fatal(err)
		}
		if err := config.EnsureLibraryLayout(lr); err != nil {
			t.Fatal(err)
		}
	}

	dbDry, err := store.Open(libDry)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = dbDry.Close() })
	var sumDry domain.OperationSummary
	_ = filepath.WalkDir(libDry, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || !IsSupportedScanExt(filepath.Ext(path)) {
			return nil
		}
		RegisterInPlacePath(dbDry, libDry, path, &sumDry, true)
		return nil
	})

	dbLive, err := store.Open(libLive)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = dbLive.Close() })
	var sumLive domain.OperationSummary
	_ = filepath.WalkDir(libLive, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || !IsSupportedScanExt(filepath.Ext(path)) {
			return nil
		}
		RegisterInPlacePath(dbLive, libLive, path, &sumLive, false)
		return nil
	})

	if sumDry != sumLive {
		t.Fatalf("dry %+v vs live %+v", sumDry, sumLive)
	}
	if sumLive.Added != 2 || sumLive.Failed != 0 {
		t.Fatalf("live: %+v", sumLive)
	}
}

func copyTree(t *testing.T, srcRoot, dstRoot string) error {
	t.Helper()
	return filepath.WalkDir(srcRoot, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(srcRoot, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		dst := filepath.Join(dstRoot, rel)
		if d.IsDir() {
			return os.MkdirAll(dst, 0o755)
		}
		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			return err
		}
		return os.WriteFile(dst, b, 0o644)
	})
}
