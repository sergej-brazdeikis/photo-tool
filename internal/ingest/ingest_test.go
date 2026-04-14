package ingest

import (
	"database/sql"
	"image"
	"image/color"
	"image/jpeg"
	"os"
	"path/filepath"
	"testing"
	"time"

	"photo-tool/internal/config"
	"photo-tool/internal/domain"
	"photo-tool/internal/store"
)

func TestIngest_duplicatePath_skipsSecond(t *testing.T) {
	libRoot := filepath.Join(t.TempDir(), "lib")
	srcDir := filepath.Join(t.TempDir(), "in")
	if err := config.EnsureLibraryLayout(libRoot); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(srcDir, 0o755); err != nil {
		t.Fatal(err)
	}
	src := filepath.Join(srcDir, "a.jpg")
	mt := time.Date(2020, 6, 1, 15, 4, 5, 0, time.UTC)
	if err := writeJPEGGray(src, 0x11); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(src, mt, mt); err != nil {
		t.Fatal(err)
	}

	db, err := store.Open(libRoot)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	sum1 := Ingest(db, libRoot, []string{src})
	if sum1.Added != 1 || sum1.SkippedDuplicate != 0 || sum1.Updated != 0 || sum1.Failed != 0 {
		t.Fatalf("first ingest summary: %+v", sum1)
	}
	if n := assetCount(t, db); n != 1 {
		t.Fatalf("asset rows after first: got %d want 1", n)
	}

	sum2 := Ingest(db, libRoot, []string{src})
	if sum2.Added != 0 || sum2.SkippedDuplicate != 1 || sum2.Updated != 0 || sum2.Failed != 0 {
		t.Fatalf("second ingest summary: %+v", sum2)
	}
	if n := assetCount(t, db); n != 1 {
		t.Fatalf("asset rows after second: got %d want 1", n)
	}

	var relPath string
	var capUnix, createdUnix int64
	var hash string
	if err := db.QueryRow(
		`SELECT content_hash, rel_path, capture_time_unix, created_at_unix FROM assets LIMIT 1`,
	).Scan(&hash, &relPath, &capUnix, &createdUnix); err != nil {
		t.Fatal(err)
	}
	if len(hash) != 64 {
		t.Fatalf("content_hash length: got %d want 64", len(hash))
	}
	if capUnix != mt.Unix() {
		t.Fatalf("capture_time_unix: got %d want %d", capUnix, mt.Unix())
	}
	if createdUnix <= 0 {
		t.Fatalf("created_at_unix: %d", createdUnix)
	}
	wantRelPrefix := "2020/06/01/"
	if filepath.ToSlash(relPath)[:len(wantRelPrefix)] != wantRelPrefix {
		t.Fatalf("rel_path prefix: got %q want prefix %q", relPath, wantRelPrefix)
	}
	destAbs := filepath.Join(libRoot, filepath.FromSlash(relPath))
	if _, err := os.Stat(destAbs); err != nil {
		t.Fatalf("dest file: %v", err)
	}
}

func TestIngest_batch_multipleFiles(t *testing.T) {
	libRoot := filepath.Join(t.TempDir(), "lib")
	srcDir := filepath.Join(t.TempDir(), "in")
	if err := config.EnsureLibraryLayout(libRoot); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(srcDir, 0o755); err != nil {
		t.Fatal(err)
	}
	mt := time.Date(2019, 12, 31, 23, 0, 1, 0, time.UTC)
	paths := make([]string, 3)
	for i := range paths {
		p := filepath.Join(srcDir, "batch", string(rune('a'+i))+".jpg")
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := writeJPEGGray(p, byte(0x20+i)); err != nil {
			t.Fatal(err)
		}
		if err := os.Chtimes(p, mt, mt); err != nil {
			t.Fatal(err)
		}
		paths[i] = p
	}

	db, err := store.Open(libRoot)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	sum := Ingest(db, libRoot, paths)
	if sum.Added != 3 || sum.SkippedDuplicate != 0 || sum.Updated != 0 || sum.Failed != 0 {
		t.Fatalf("batch summary: %+v", sum)
	}
	if n := assetCount(t, db); n != 3 {
		t.Fatalf("asset rows: got %d want 3", n)
	}
}

func TestIngestWithAssetIDs_duplicateMatchesFirstID(t *testing.T) {
	libRoot := filepath.Join(t.TempDir(), "lib")
	srcDir := filepath.Join(t.TempDir(), "in")
	if err := config.EnsureLibraryLayout(libRoot); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(srcDir, 0o755); err != nil {
		t.Fatal(err)
	}
	src := filepath.Join(srcDir, "a.jpg")
	mt := time.Date(2020, 6, 1, 15, 4, 5, 0, time.UTC)
	if err := writeJPEGGray(src, 0x22); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(src, mt, mt); err != nil {
		t.Fatal(err)
	}

	db, err := store.Open(libRoot)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	sum, ids := IngestWithAssetIDs(db, libRoot, []string{src, src})
	if sum.Added != 1 || sum.SkippedDuplicate != 1 || sum.Failed != 0 {
		t.Fatalf("summary: %+v", sum)
	}
	if ids[0] == 0 || ids[1] != ids[0] {
		t.Fatalf("asset ids: %v", ids)
	}
}

func TestIngest_dryRun_noWrites_emptyDB(t *testing.T) {
	libRoot := filepath.Join(t.TempDir(), "lib")
	srcDir := filepath.Join(t.TempDir(), "in")
	if err := config.EnsureLibraryLayout(libRoot); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(srcDir, 0o755); err != nil {
		t.Fatal(err)
	}
	src := filepath.Join(srcDir, "a.jpg")
	mt := time.Date(2020, 6, 1, 15, 4, 5, 0, time.UTC)
	if err := writeJPEGGray(src, 0x33); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(src, mt, mt); err != nil {
		t.Fatal(err)
	}

	db, err := store.Open(libRoot)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	var sum domain.OperationSummary
	id := IngestPath(db, libRoot, src, &sum, true)
	if id != 0 {
		t.Fatalf("dry-run id: got %d want 0", id)
	}
	if sum.Added != 1 || sum.SkippedDuplicate != 0 || sum.Failed != 0 {
		t.Fatalf("dry-run summary: %+v", sum)
	}
	if n := assetCount(t, db); n != 0 {
		t.Fatalf("dry-run must not insert: got %d asset rows", n)
	}

	sum2 := Ingest(db, libRoot, []string{src})
	if sum2.Added != 1 {
		t.Fatalf("live ingest: %+v", sum2)
	}
	if n := assetCount(t, db); n != 1 {
		t.Fatalf("after live: got %d want 1", n)
	}

	var sum3 domain.OperationSummary
	_ = IngestPath(db, libRoot, src, &sum3, true)
	if sum3.Added != 0 || sum3.SkippedDuplicate != 1 {
		t.Fatalf("dry-run after live: %+v", sum3)
	}
}

func TestIngestPaths_dryRun_matchesLiveWhenUnique(t *testing.T) {
	libRoot := filepath.Join(t.TempDir(), "lib")
	srcDir := filepath.Join(t.TempDir(), "in")
	if err := config.EnsureLibraryLayout(libRoot); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(srcDir, 0o755); err != nil {
		t.Fatal(err)
	}
	src := filepath.Join(srcDir, "a.jpg")
	mt := time.Date(2021, 1, 2, 3, 4, 5, 0, time.UTC)
	if err := writeJPEGGray(src, 0x44); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(src, mt, mt); err != nil {
		t.Fatal(err)
	}

	db, err := store.Open(libRoot)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	dry := IngestPaths(db, libRoot, []string{src}, true)
	live := IngestPaths(db, libRoot, []string{src}, false)
	if dry.Added != live.Added || dry.SkippedDuplicate != live.SkippedDuplicate || dry.Failed != live.Failed {
		t.Fatalf("dry %+v vs live %+v", dry, live)
	}
}

func TestIngestWithAssetIDs_failedPathZeroID(t *testing.T) {
	libRoot := filepath.Join(t.TempDir(), "lib")
	if err := config.EnsureLibraryLayout(libRoot); err != nil {
		t.Fatal(err)
	}
	db, err := store.Open(libRoot)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	sum, ids := IngestWithAssetIDs(db, libRoot, []string{"/no/such/file/ever.jpg"})
	if sum.Failed != 1 || sum.Added != 0 || len(ids) != 1 || ids[0] != 0 {
		t.Fatalf("sum=%+v ids=%v", sum, ids)
	}
}

func assetCount(t *testing.T, db *sql.DB) int {
	t.Helper()
	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM assets`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

func writeJPEGGray(path string, y byte) error {
	img := image.NewGray(image.Rect(0, 0, 2, 2))
	img.Set(0, 0, color.Gray{Y: y})
	img.Set(1, 0, color.Gray{Y: y})
	img.Set(0, 1, color.Gray{Y: y ^ 1})
	img.Set(1, 1, color.Gray{Y: y})
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return jpeg.Encode(f, img, &jpeg.Options{Quality: 80})
}
