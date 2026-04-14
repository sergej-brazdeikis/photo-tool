package ingest

import (
	"slices"
	"testing"
)

func TestIsSupportedIngestExt(t *testing.T) {
	for _, ext := range []string{".jpg", ".JPEG", "jpg", ".heic", ".DNG", ".tif"} {
		if !IsSupportedIngestExt(ext) {
			t.Fatalf("expected supported: %q", ext)
		}
	}
	for _, ext := range []string{"", ".", ".raw", ".txt", ".mp4"} {
		if IsSupportedIngestExt(ext) {
			t.Fatalf("expected unsupported: %q", ext)
		}
	}
}

func TestPickerFilterExtensions_includesUpperCaseHEICDNG(t *testing.T) {
	exts := PickerFilterExtensions()
	if !slices.Contains(exts, ".heic") || !slices.Contains(exts, ".HEIC") {
		t.Fatalf("missing HEIC variants: %v", exts)
	}
	if !slices.Contains(exts, ".dng") || !slices.Contains(exts, ".DNG") {
		t.Fatalf("missing DNG variants: %v", exts)
	}
}
