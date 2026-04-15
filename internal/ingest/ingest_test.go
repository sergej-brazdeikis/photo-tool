package ingest

import (
	"bytes"
	"database/sql"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/dsoprea/go-exif/v3"
	exifcommon "github.com/dsoprea/go-exif/v3/common"

	"photo-tool/internal/config"
	"photo-tool/internal/domain"
	"photo-tool/internal/exifmeta"
	"photo-tool/internal/store"
)

func TestIsUniqueContentHash_moderncSQLite(t *testing.T) {
	libRoot := filepath.Join(t.TempDir(), "lib")
	if err := config.EnsureLibraryLayout(libRoot); err != nil {
		t.Fatal(err)
	}
	db, err := store.Open(libRoot)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	const hash = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	_, err = db.Exec(`
INSERT INTO assets (content_hash, rel_path, capture_time_unix, created_at_unix)
VALUES (?, '2000/01/01/a.jpg', 1, 1)`, hash)
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(`
INSERT INTO assets (content_hash, rel_path, capture_time_unix, created_at_unix)
VALUES (?, '2000/01/01/b.jpg', 1, 1)`, hash)
	if err == nil {
		t.Fatal("expected unique violation on content_hash")
	}
	if !isUniqueContentHash(err) {
		t.Fatalf("expected content_hash unique via sqlite.Error, got %v", err)
	}
	wrapped := fmt.Errorf("insert asset: %w", err)
	if !isUniqueContentHash(wrapped) {
		t.Fatalf("expected errors.As through wrap, got %v", wrapped)
	}
}

func TestIsUniqueContentHash_falseForRelPathUnique(t *testing.T) {
	libRoot := filepath.Join(t.TempDir(), "lib")
	if err := config.EnsureLibraryLayout(libRoot); err != nil {
		t.Fatal(err)
	}
	db, err := store.Open(libRoot)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	const rel = "2000/01/01/same_rel.jpg"
	_, err = db.Exec(`
INSERT INTO assets (content_hash, rel_path, capture_time_unix, created_at_unix)
VALUES (?, ?, 1, 1)`, "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", rel)
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(`
INSERT INTO assets (content_hash, rel_path, capture_time_unix, created_at_unix)
VALUES (?, ?, 1, 1)`, "cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc", rel)
	if err == nil {
		t.Fatal("expected unique violation on rel_path")
	}
	if isUniqueContentHash(err) {
		t.Fatalf("rel_path unique must not be treated as content_hash duplicate: %v", err)
	}
}

func TestIngest_exifDateTimeOriginal_drivesCanonicalPath(t *testing.T) {
	const exifLocal = "2017:08:09 14:30:00"
	exifBlob := mustBuildExifWithDateTimeOriginal(t, exifLocal)
	jpegBytes := jpegWithAPP1Exif(t, exifBlob, 0x66)

	libRoot := filepath.Join(t.TempDir(), "lib")
	srcDir := filepath.Join(t.TempDir(), "in")
	if err := config.EnsureLibraryLayout(libRoot); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(srcDir, 0o755); err != nil {
		t.Fatal(err)
	}
	src := filepath.Join(srcDir, "exif_time.jpg")
	if err := os.WriteFile(src, jpegBytes, 0o644); err != nil {
		t.Fatal(err)
	}
	wrongMtime := time.Date(2030, 1, 1, 12, 0, 0, 0, time.UTC)
	if err := os.Chtimes(src, wrongMtime, wrongMtime); err != nil {
		t.Fatal(err)
	}

	cap, err := exifmeta.ReadCapture(src)
	if err != nil {
		t.Fatal(err)
	}
	if cap.Source != exifmeta.SourceExifDateTimeOriginal {
		t.Fatalf("ReadCapture source: got %q want exif DateTimeOriginal", cap.Source)
	}
	wantUTC, err := time.ParseInLocation("2006:01:02 15:04:05", exifLocal, time.Local)
	if err != nil {
		t.Fatal(err)
	}
	if !cap.UTC.Equal(wantUTC.UTC()) {
		t.Fatalf("capture UTC: got %v want %v", cap.UTC, wantUTC.UTC())
	}

	db, err := store.Open(libRoot)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	sum := Ingest(db, libRoot, []string{src})
	if sum.Added != 1 || sum.Failed != 0 {
		t.Fatalf("ingest summary: %+v", sum)
	}

	var relPath string
	var capUnix int64
	if err := db.QueryRow(`SELECT rel_path, capture_time_unix FROM assets LIMIT 1`).Scan(&relPath, &capUnix); err != nil {
		t.Fatal(err)
	}
	if capUnix != wantUTC.Unix() {
		t.Fatalf("row capture_time_unix: got %d want %d", capUnix, wantUTC.Unix())
	}
	wantRelPrefix := "2017/08/09/"
	if len(filepath.ToSlash(relPath)) < len(wantRelPrefix) || filepath.ToSlash(relPath)[:len(wantRelPrefix)] != wantRelPrefix {
		t.Fatalf("rel_path: got %q want prefix %q", relPath, wantRelPrefix)
	}
}

func TestIngest_concurrentSameSource_oneRow_fileSurvives(t *testing.T) {
	libRoot := filepath.Join(t.TempDir(), "lib")
	srcDir := filepath.Join(t.TempDir(), "in")
	if err := config.EnsureLibraryLayout(libRoot); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(srcDir, 0o755); err != nil {
		t.Fatal(err)
	}
	src := filepath.Join(srcDir, "race.jpg")
	mt := time.Date(2020, 6, 1, 15, 4, 5, 0, time.UTC)
	if err := writeJPEGGray(src, 0x55); err != nil {
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

	const n = 16
	var wg sync.WaitGroup
	ch := make(chan domain.OperationSummary, n)
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ch <- Ingest(db, libRoot, []string{src})
		}()
	}
	wg.Wait()
	close(ch)

	var acc domain.OperationSummary
	for s := range ch {
		acc.Added += s.Added
		acc.SkippedDuplicate += s.SkippedDuplicate
		acc.Updated += s.Updated
		acc.Failed += s.Failed
	}
	if acc.Added != 1 {
		t.Fatalf("concurrent added: got %d want 1 (acc=%+v)", acc.Added, acc)
	}
	if acc.SkippedDuplicate != n-1 {
		t.Fatalf("concurrent skipped_duplicate: got %d want %d (acc=%+v)", acc.SkippedDuplicate, n-1, acc)
	}
	if acc.Failed != 0 || acc.Updated != 0 {
		t.Fatalf("unexpected counts: %+v", acc)
	}
	if got := assetCount(t, db); got != 1 {
		t.Fatalf("asset rows: got %d want 1", got)
	}
	var relPath string
	if err := db.QueryRow(`SELECT rel_path FROM assets LIMIT 1`).Scan(&relPath); err != nil {
		t.Fatal(err)
	}
	destAbs := filepath.Join(libRoot, filepath.FromSlash(relPath))
	if _, err := os.Stat(destAbs); err != nil {
		t.Fatalf("library file missing after concurrent ingest: %v", err)
	}
}

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

func TestIngest_batch_mixedSuccessAndFailure(t *testing.T) {
	libRoot := filepath.Join(t.TempDir(), "lib")
	srcDir := filepath.Join(t.TempDir(), "in")
	if err := config.EnsureLibraryLayout(libRoot); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(srcDir, 0o755); err != nil {
		t.Fatal(err)
	}
	mt := time.Date(2018, 7, 4, 12, 0, 0, 0, time.UTC)
	ok1 := filepath.Join(srcDir, "ok1.jpg")
	if err := writeJPEGGray(ok1, 0x77); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(ok1, mt, mt); err != nil {
		t.Fatal(err)
	}
	missing := filepath.Join(srcDir, "nope.jpg")
	ok2 := filepath.Join(srcDir, "ok2.jpg")
	if err := writeJPEGGray(ok2, 0x88); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(ok2, mt, mt); err != nil {
		t.Fatal(err)
	}

	db, err := store.Open(libRoot)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	sum := Ingest(db, libRoot, []string{ok1, missing, ok2})
	if sum.Added != 2 || sum.Failed != 1 || sum.SkippedDuplicate != 0 || sum.Updated != 0 {
		t.Fatalf("mixed batch summary: %+v", sum)
	}
	if n := assetCount(t, db); n != 2 {
		t.Fatalf("asset rows: got %d want 2", n)
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
	id := IngestPath(db, libRoot, src, &sum, true, nil)
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
	_ = IngestPath(db, libRoot, src, &sum3, true, nil)
	if sum3.Added != 0 || sum3.SkippedDuplicate != 1 {
		t.Fatalf("dry-run after live: %+v", sum3)
	}
}

func TestIngestPaths_dryRun_matchesLiveWhenSameBytesTwoPaths(t *testing.T) {
	libRoot := filepath.Join(t.TempDir(), "lib")
	srcDir := filepath.Join(t.TempDir(), "in")
	if err := config.EnsureLibraryLayout(libRoot); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(srcDir, 0o755); err != nil {
		t.Fatal(err)
	}
	mt := time.Date(2022, 3, 4, 5, 6, 7, 0, time.UTC)
	a := filepath.Join(srcDir, "a.jpg")
	b := filepath.Join(srcDir, "b.jpg")
	for _, p := range []string{a, b} {
		if err := writeJPEGGray(p, 0x77); err != nil {
			t.Fatal(err)
		}
		if err := os.Chtimes(p, mt, mt); err != nil {
			t.Fatal(err)
		}
	}

	db, err := store.Open(libRoot)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	dry := IngestPaths(db, libRoot, []string{a, b}, true)
	live := IngestPaths(db, libRoot, []string{a, b}, false)
	if dry != live {
		t.Fatalf("dry %+v vs live %+v", dry, live)
	}
	if live.Added != 1 || live.SkippedDuplicate != 1 || live.Failed != 0 {
		t.Fatalf("live summary: %+v", live)
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

// mustBuildExifWithDateTimeOriginal mirrors internal/exifmeta capture_test helpers: EXIF sub-IFD DateTimeOriginal.
func mustBuildExifWithDateTimeOriginal(t *testing.T, dateTimeOriginal string) []byte {
	t.Helper()
	im, err := exifcommon.NewIfdMappingWithStandard()
	if err != nil {
		t.Fatal(err)
	}
	ti := exif.NewTagIndex()
	if err := exif.LoadStandardTags(ti); err != nil {
		t.Fatal(err)
	}
	rootIb := exif.NewIfdBuilder(im, ti, exifcommon.IfdStandardIfdIdentity, exifcommon.EncodeDefaultByteOrder)
	exifIb, err := exif.GetOrCreateIbFromRootIb(rootIb, "IFD/Exif")
	if err != nil {
		t.Fatal(err)
	}
	if err := exifIb.SetStandardWithName("DateTimeOriginal", dateTimeOriginal); err != nil {
		t.Fatal(err)
	}
	ibe := exif.NewIfdByteEncoder()
	blob, err := ibe.EncodeToExif(rootIb)
	if err != nil {
		t.Fatal(err)
	}
	return blob
}

// jpegWithAPP1Exif returns a minimal JPEG whose first marker after SOI is APP1 carrying exifBlob (TIFF EXIF payload).
func jpegWithAPP1Exif(t *testing.T, exifBlob []byte, grayY byte) []byte {
	t.Helper()
	img := image.NewGray(image.Rect(0, 0, 2, 2))
	img.Set(0, 0, color.Gray{Y: grayY})
	img.Set(1, 0, color.Gray{Y: grayY})
	img.Set(0, 1, color.Gray{Y: grayY ^ 1})
	img.Set(1, 1, color.Gray{Y: grayY})
	var jpegBody bytes.Buffer
	if err := jpeg.Encode(&jpegBody, img, &jpeg.Options{Quality: 80}); err != nil {
		t.Fatal(err)
	}
	raw := jpegBody.Bytes()
	if len(raw) < 2 || raw[0] != 0xff || raw[1] != 0xd8 {
		t.Fatal(errors.New("jpeg.Encode did not produce SOI"))
	}
	payload := append([]byte("Exif\x00"), exifBlob...)
	segLen := len(payload) + 2
	var out bytes.Buffer
	out.Write([]byte{0xff, 0xd8, 0xff, 0xe1, byte(segLen >> 8), byte(segLen & 0xff)})
	out.Write(payload)
	out.Write(raw[2:])
	return out.Bytes()
}
