package paths

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"
)

// CanonicalDayDir returns {root}/{YYYY}/{MM}/{DD} for capture time in UTC.
// Folder naming matches PRD storage layout (calendar components).
func CanonicalDayDir(root string, captureUTC time.Time) string {
	t := captureUTC.UTC()
	y, m, d := t.Date()
	return filepath.Join(root,
		fmt.Sprintf("%04d", y),
		fmt.Sprintf("%02d", int(m)),
		fmt.Sprintf("%02d", d),
	)
}

// SuggestedFilename builds "{YYYYMMDD-HHMMSS}_{hashPrefix}{ext}" using UTC.
// ext must include the leading dot (e.g. ".jpg"). hashHexFull is lowercased;
// prefix uses the first 12 hex runes for shorter filenames (full hash lives in DB).
func SuggestedFilename(captureUTC time.Time, hashHexFull, ext string) string {
	ext = strings.ToLower(ext)
	if ext != "" && ext[0] != '.' {
		ext = "." + ext
	}
	ts := captureUTC.UTC().Format("20060102-150405")
	h := strings.ToLower(hashHexFull)
	if len(h) > 12 {
		h = h[:12]
	}
	return fmt.Sprintf("%s_%s%s", ts, h, ext)
}
