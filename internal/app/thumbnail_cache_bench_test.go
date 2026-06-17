package app

import (
	"os"
	"path/filepath"
	"testing"

	"photo-tool/internal/config"
	"photo-tool/internal/domain"
	"photo-tool/internal/fixture"
	"photo-tool/internal/store"
)

// BenchmarkReviewThumbnailCache500 measures decode vs cache-hit for 500 review thumbnails.
func BenchmarkReviewThumbnailCache500(b *testing.B) {
	root := b.TempDir()
	if err := config.EnsureLibraryLayout(root); err != nil {
		b.Fatal(err)
	}
	db, err := store.Open(root)
	if err != nil {
		b.Fatal(err)
	}
	b.Cleanup(func() { _ = db.Close() })
	if _, _, err := fixture.SeedLibrary(db, fixture.SeedOptions{Tier: fixture.TierS5, Root: root}); err != nil {
		b.Fatal(err)
	}
	rows, err := store.ListAssetsForReview(db, domain.ReviewFilters{}, 500, 0)
	if err != nil {
		b.Fatal(err)
	}
	if len(rows) < 100 {
		b.Fatalf("need assets, got %d", len(rows))
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, row := range rows {
			src := filepath.Join(root, filepath.FromSlash(row.RelPath))
			cache := ThumbnailCachePath(root, row.ID, row.ContentHash)
			if _, err := os.Stat(cache); err != nil {
				if err := WriteThumbnailJPEG(src, cache); err != nil {
					b.Fatal(err)
				}
			}
			if _, err := decodeImageFile(cache); err != nil {
				b.Fatal(err)
			}
		}
	}
}
