package app

import (
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"

	"fyne.io/fyne/v2"

	"photo-tool/internal/ingest"
)

type droppedPaths struct {
	Supported   []string
	Unsupported []string
}

// tryAddUniquePath appends abs to paths if not already present (after filepath.Clean).
// Returns whether paths changed.
func tryAddUniquePath(paths *[]string, abs string) bool {
	abs = filepath.Clean(abs)
	for _, p := range *paths {
		if p == abs {
			return false
		}
	}
	*paths = append(*paths, abs)
	return true
}

func uriLocalPath(u fyne.URI) (string, error) {
	if u == nil {
		return "", fmt.Errorf("nil URI")
	}
	scheme := strings.ToLower(strings.TrimSpace(u.Scheme()))
	if scheme != "" && scheme != "file" {
		return "", fmt.Errorf("not a local file (%s)", scheme)
	}
	p := strings.TrimSpace(u.Path())
	if p == "" {
		return "", fmt.Errorf("empty path")
	}
	return filepath.Clean(p), nil
}

// dropRejectReason turns [uriLocalPath] failures into short user-facing lines (Story 2.12 AC5).
func dropRejectReason(err error) string {
	if err == nil {
		return ""
	}
	s := err.Error()
	switch {
	case s == "nil URI":
		return "A dropped item could not be read."
	case s == "empty path":
		return "A dropped item had no file path."
	case strings.HasPrefix(s, "not a local file ("):
		return "That drop is not a file on this computer (for example a browser or app link). Save or export the image, then drop the saved file or use Add images…"
	default:
		return s
	}
}

// droppedSkipSummaryForDialog joins per-item skip reasons for ShowInformation (Story 2.12 AC5).
// Long lists are capped so dialogs stay readable; the tail reminds how to recover.
func droppedSkipSummaryForDialog(lines []string) string {
	const capN = 8
	if len(lines) == 0 {
		return ""
	}
	if len(lines) <= capN {
		return strings.Join(lines, "\n")
	}
	head := strings.Join(lines[:capN], "\n")
	rest := len(lines) - capN
	return head + fmt.Sprintf("\n\n… and %d more — use Add images… if you need to pick files manually.", rest)
}

// classifyDroppedURIs turns OS drop URIs into supported paths and human-facing skip lines.
// stat follows [os.Stat] — inject in tests.
func rectContainsPoint(absPos, topLeft fyne.Position, size fyne.Size) bool {
	return absPos.X >= topLeft.X && absPos.Y >= topLeft.Y &&
		absPos.X < topLeft.X+size.Width && absPos.Y < topLeft.Y+size.Height
}

func classifyDroppedURIs(uris []fyne.URI, stat func(string) (fs.FileInfo, error)) droppedPaths {
	seen := make(map[string]struct{})
	var out droppedPaths
	for _, u := range uris {
		path, err := uriLocalPath(u)
		if err != nil {
			out.Unsupported = append(out.Unsupported, dropRejectReason(err))
			continue
		}
		if _, ok := seen[path]; ok {
			continue
		}
		seen[path] = struct{}{}

		fi, err := stat(path)
		if err != nil {
			out.Unsupported = append(out.Unsupported, fmt.Sprintf("%s: not accessible", filepath.Base(path)))
			continue
		}
		if fi.IsDir() {
			out.Unsupported = append(out.Unsupported, fmt.Sprintf("%s: folders are not supported (drop files only)", filepath.Base(path)))
			continue
		}
		if !ingest.IsSupportedIngestExt(filepath.Ext(path)) {
			out.Unsupported = append(out.Unsupported, fmt.Sprintf("%s: unsupported type", filepath.Base(path)))
			continue
		}
		out.Supported = append(out.Supported, path)
	}
	return out
}

func dropHitTest(absPos fyne.Position, target fyne.CanvasObject) bool {
	if target == nil {
		return false
	}
	d := fyne.CurrentApp().Driver()
	if d == nil {
		return false
	}
	tp := d.AbsolutePositionForObject(target)
	ts := target.Size()
	return rectContainsPoint(absPos, tp, ts)
}
