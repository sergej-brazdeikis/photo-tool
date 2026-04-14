package app

import (
	"image"
	"image/jpeg"
	"os"
	"path/filepath"
	"testing"

	"photo-tool/internal/config"
)

func TestThumbnailCachePath_sharded(t *testing.T) {
	got := ThumbnailCachePath("/lib", 7, "abcdef")
	want := filepath.Join("/lib", ".cache", "thumbnails", "ab", "7.jpg")
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestWriteThumbnailJPEG_invalidSource(t *testing.T) {
	root := filepath.Join(t.TempDir(), "lib")
	if err := config.EnsureLibraryLayout(root); err != nil {
		t.Fatal(err)
	}
	dest := ThumbnailCachePath(root, 1, "aa")
	err := WriteThumbnailJPEG(filepath.Join(root, "nope.jpg"), dest)
	if err == nil {
		t.Fatal("expected error for missing file")
	}
	if _, err := os.Stat(dest); err == nil {
		t.Fatal("should not create cache file")
	}
}

func TestWriteThumbnailJPEG_roundTrip(t *testing.T) {
	root := filepath.Join(t.TempDir(), "lib")
	if err := config.EnsureLibraryLayout(root); err != nil {
		t.Fatal(err)
	}
	src := filepath.Join(root, "in.jpg")
	img := image.NewRGBA(image.Rect(0, 0, 8, 8))
	f, err := os.Create(src)
	if err != nil {
		t.Fatal(err)
	}
	if err := jpeg.Encode(f, img, &jpeg.Options{Quality: 80}); err != nil {
		_ = f.Close()
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	dest := ThumbnailCachePath(root, 9, "cafef00d")
	if err := WriteThumbnailJPEG(src, dest); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(dest); err != nil {
		t.Fatal(err)
	}
}
