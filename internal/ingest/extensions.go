package ingest

import (
	"sort"
	"strings"
)

// supportedIngestExts is the single source of truth for extensions the ingest pipeline may attempt
// (GUI file picker + CLI scan). Keep in sync with exifmeta/readability expectations — formats that
// reliably fail at ReadCapture still increment Failed, not silent skip.
var supportedIngestExts = map[string]struct{}{
	".jpg": {}, ".jpeg": {}, ".png": {}, ".gif": {},
	".webp": {}, ".tif": {}, ".tiff": {},
	".heic": {}, ".dng": {},
}

// extensionsWithUppercaseVariant are included twice in [PickerFilterExtensions] (lower + upper)
// so the desktop picker matches common on-disk casing.
var extensionsWithUppercaseVariant = map[string]struct{}{
	".heic": {}, ".dng": {},
}

// IsSupportedIngestExt reports whether a path's extension is eligible for GUI ingest, CLI scan, and import.
func IsSupportedIngestExt(ext string) bool {
	e := strings.ToLower(strings.TrimSpace(ext))
	if e == "" {
		return false
	}
	if !strings.HasPrefix(e, ".") {
		e = "." + e
	}
	_, ok := supportedIngestExts[e]
	return ok
}

// IsSupportedScanExt is an alias for [IsSupportedIngestExt] (CLI scan uses the same rules as upload).
func IsSupportedScanExt(ext string) bool {
	return IsSupportedIngestExt(ext)
}

// PickerFilterExtensions returns distinct extensions for Fyne's [storage.NewExtensionFileFilter].
func PickerFilterExtensions() []string {
	out := make([]string, 0, len(supportedIngestExts)+len(extensionsWithUppercaseVariant))
	seen := make(map[string]struct{})
	add := func(s string) {
		if _, ok := seen[s]; ok {
			return
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	for ext := range supportedIngestExts {
		add(ext)
		if _, d := extensionsWithUppercaseVariant[ext]; d {
			add(strings.ToUpper(ext))
		}
	}
	sort.Strings(out)
	return out
}
