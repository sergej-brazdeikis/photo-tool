//go:build darwin && !cgo

package app

import (
	"fmt"
	"os"
)

// NFR07AC3DisplayScalingPercent uses CoreGraphics when built with cgo; without
// cgo, only the GitHub Actions macOS CI surrogate (FYNE_SCALE + tier env) applies.
func NFR07AC3DisplayScalingPercent() (pct int, detail string, ok bool) {
	if tier, fyneWant, okS := nfr07AC3DarwinCISurrogate(); okS {
		if got := os.Getenv("FYNE_SCALE"); got != fyneWant {
			return 0, fmt.Sprintf("macOS CI AC3: tier %d%% requires FYNE_SCALE=%q (got %q)", tier, fyneWant, got), false
		}
		return tier, fmt.Sprintf("NFR-07 AC3 macOS CI: surrogate tier=%d%% FYNE_SCALE=%s (build without cgo — no CoreGraphics probe; runner cannot set Displays scaling)", tier, fyneWant), true
	}
	return 0, "darwin build without cgo cannot probe display scaling (enable CGO for NFR-07 AC3 on hardware)", false
}
