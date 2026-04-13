package paths

import (
	"path/filepath"
	"testing"
	"time"
)

func TestCanonicalDayDir(t *testing.T) {
	root := "/data/lib"
	ts := time.Date(2024, 3, 9, 15, 4, 5, 0, time.UTC)
	got := CanonicalDayDir(root, ts)
	want := filepath.Join(root, "2024", "03", "09")
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestSuggestedFilename(t *testing.T) {
	ts := time.Date(2024, 3, 9, 15, 4, 5, 0, time.UTC)
	hash := "ABCDEF0123456789ABCDEF0123456789ABCDEF0123456789ABCDEF0123456789"
	got := SuggestedFilename(ts, hash, ".JPG")
	want := "20240309-150405_abcdef012345.jpg"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}
