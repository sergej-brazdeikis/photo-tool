package fixture_test

import (
	"fmt"
	"path/filepath"
	"testing"

	"photo-tool/internal/config"
	"photo-tool/internal/domain"
	"photo-tool/internal/fixture"
	"photo-tool/internal/ingest"
	"photo-tool/internal/store"
)

func TestScale_Ingest_dedupStorm(t *testing.T) {
	if testing.Short() {
		t.Skip("dedup storm skipped in -short")
	}
	root := t.TempDir()
	if err := config.EnsureLibraryLayout(root); err != nil {
		t.Fatal(err)
	}
	db, err := store.Open(root)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	srcDir := filepath.Join(root, "dup-src")
	dupPath := filepath.Join(srcDir, "dup.jpg")
	if err := fixture.WriteTinyJPEG(dupPath); err != nil {
		t.Fatal(err)
	}
	var sum domain.OperationSummary
	for i := 0; i < 20; i++ {
		ingest.RegisterInPlacePath(db, root, dupPath, &sum, false)
	}
	if sum.Added != 1 {
		t.Fatalf("added=%d want 1", sum.Added)
	}
	if sum.SkippedDuplicate < 19 {
		t.Fatalf("skipped=%d want ≥19", sum.SkippedDuplicate)
	}
}

func TestScale_Import_register100(t *testing.T) {
	if testing.Short() {
		t.Skip("import scale skipped in -short")
	}
	const n = 100
	root := t.TempDir()
	if err := config.EnsureLibraryLayout(root); err != nil {
		t.Fatal(err)
	}
	db, err := store.Open(root)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	srcDir := filepath.Join(root, "import-src")
	var sum domain.OperationSummary
	for i := 0; i < n; i++ {
		p := filepath.Join(srcDir, fmt.Sprintf("f%04d.jpg", i))
		if err := fixture.WriteTinyJPEG(p); err != nil {
			t.Fatal(err)
		}
		ingest.RegisterInPlacePath(db, root, p, &sum, false)
	}
	if sum.Added != n {
		t.Fatalf("added=%d want %d skipped=%d failed=%d", sum.Added, n, sum.SkippedDuplicate, sum.Failed)
	}
	total, err := store.CountAssetsForReview(db, domain.ReviewFilters{})
	if err != nil {
		t.Fatal(err)
	}
	if total != int64(n) {
		t.Fatalf("review count=%d want %d", total, n)
	}
}

func TestScale_Import_register1000(t *testing.T) {
	if testing.Short() {
		t.Skip("import 1k scale skipped in -short")
	}
	const n = 1000
	root := t.TempDir()
	if err := config.EnsureLibraryLayout(root); err != nil {
		t.Fatal(err)
	}
	db, err := store.Open(root)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	srcDir := filepath.Join(root, "import-src-1k")
	var sum domain.OperationSummary
	for i := 0; i < n; i++ {
		p := filepath.Join(srcDir, fmt.Sprintf("f%04d.jpg", i))
		if err := fixture.WriteTinyJPEG(p); err != nil {
			t.Fatal(err)
		}
		ingest.RegisterInPlacePath(db, root, p, &sum, false)
	}
	if sum.Added != n {
		t.Fatalf("added=%d want %d skipped=%d failed=%d", sum.Added, n, sum.SkippedDuplicate, sum.Failed)
	}
	total, err := store.CountAssetsForReview(db, domain.ReviewFilters{})
	if err != nil {
		t.Fatal(err)
	}
	if total != int64(n) {
		t.Fatalf("review count=%d want %d", total, n)
	}
}
