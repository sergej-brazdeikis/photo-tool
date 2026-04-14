//go:build darwin && cgo

package app

/*
#cgo LDFLAGS: -framework CoreGraphics
#include <CoreGraphics/CoreGraphics.h>
#include <stdlib.h>
*/
import "C"

import (
	"fmt"
	"os"
	"strings"
)

// nfr07DarwinCoreGraphicsUIProbe returns the effective UI scale (100% baseline)
// implied by the largest pixel-to-point ratio among active, non-mirrored
// displays. detail is always non-empty when got is true.
func nfr07DarwinCoreGraphicsUIProbe() (uiPct int, maxRatio, ui float64, got bool, detail string) {
	var maxRatioLocal float64
	var gotLocal bool

	var n C.uint32_t
	if C.CGGetActiveDisplayList(0, nil, &n) == C.kCGErrorSuccess && n > 0 {
		ids := make([]C.CGDirectDisplayID, n)
		if C.CGGetActiveDisplayList(n, (*C.CGDirectDisplayID)(&ids[0]), &n) == C.kCGErrorSuccess {
			for _, id := range ids {
				if C.CGDisplayIsActive(id) == 0 {
					continue
				}
				if C.CGDisplayMirrorsDisplay(id) != 0 {
					continue
				}
				bounds := C.CGDisplayBounds(id)
				ptW := float64(bounds.size.width)
				pw := float64(C.CGDisplayPixelsWide(id))
				if ptW <= 0 || pw <= 0 {
					continue
				}
				r := pw / ptW
				if !gotLocal || r > maxRatioLocal {
					maxRatioLocal = r
					gotLocal = true
				}
			}
		}
	}
	if !gotLocal {
		id := C.CGMainDisplayID()
		if C.CGDisplayIsActive(id) == 0 {
			return 0, 0, 0, false, "CoreGraphics: main display is inactive — wake the display for a reliable pixel/point probe"
		}
		bounds := C.CGDisplayBounds(id)
		ptW := float64(bounds.size.width)
		pw := float64(C.CGDisplayPixelsWide(id))
		if ptW <= 0 || pw <= 0 {
			return 0, 0, 0, false, "CoreGraphics: invalid main display bounds"
		}
		maxRatioLocal = pw / ptW
		gotLocal = true
	}

	uiLocal := maxRatioLocal
	if maxRatioLocal >= 1.7 {
		uiLocal = maxRatioLocal / 2.0
	}
	p := int(uiLocal*100.0 + 0.5)
	summary := fmt.Sprintf("CoreGraphics pixel/point max=%.3f uiScale=%.3f (~%d%% UI vs 1x baseline)", maxRatioLocal, uiLocal, p)
	return p, maxRatioLocal, uiLocal, true, summary
}

// nfr07DarwinTierFromUIPct maps effective UI percent to NFR-07 125%/150% tiers.
func nfr07DarwinTierFromUIPct(uiPct int) (tier int, ok bool) {
	switch {
	case uiPct >= 122 && uiPct <= 128:
		return 125, true
	case uiPct >= 146 && uiPct <= 154:
		return 150, true
	default:
		return 0, false
	}
}

// NFR07AC3DisplayScalingPercent uses the maximum pixel-to-point ratio across
// active (non-mirrored) displays. On Retina, normalized ui = ratio/2 when
// ratio >= 1.7 so a 2.5x mode reads as ~125% UI vs a 2x baseline.
func NFR07AC3DisplayScalingPercent() (pct int, detail string, ok bool) {
	cgPct, _, _, cgGot, cgLine := nfr07DarwinCoreGraphicsUIProbe()

	if tier, fyneWant, okS := nfr07AC3DarwinCISurrogate(); okS {
		if got := os.Getenv("FYNE_SCALE"); got != fyneWant {
			return 0, fmt.Sprintf("macOS CI AC3: tier %d%% requires FYNE_SCALE=%q (got %q); %s", tier, fyneWant, got, cgLine), false
		}
		var b strings.Builder
		b.WriteString(fmt.Sprintf("NFR-07 AC3 macOS CI: surrogate tier=%d%% FYNE_SCALE=%s", tier, fyneWant))
		if !cgGot {
			b.WriteString(fmt.Sprintf("; CoreGraphics unavailable (%s)", cgLine))
			return tier, b.String(), true
		}
		b.WriteString("; ")
		b.WriteString(cgLine)
		if cgTier, match := nfr07DarwinTierFromUIPct(cgPct); match {
			if cgTier == tier {
				b.WriteString(fmt.Sprintf("; CoreGraphics matches NFR-07 tier %d%%", tier))
			} else {
				b.WriteString(fmt.Sprintf("; CoreGraphics NFR-07 tier %d%% (workflow tier=%d%%)", cgTier, tier))
			}
		} else {
			b.WriteString(fmt.Sprintf("; CoreGraphics ~%d%% UI (runner observation; workflow surrogate enforces NFR-07 %d%% tier)", cgPct, tier))
		}
		return tier, b.String(), true
	}

	if !cgGot {
		return 0, cgLine, false
	}

	switch {
	case cgPct >= 122 && cgPct <= 128:
		return 125, fmt.Sprintf("macOS display %s (125%% tier)", cgLine), true
	case cgPct >= 146 && cgPct <= 154:
		return 150, fmt.Sprintf("macOS display %s (150%% tier)", cgLine), true
	default:
		return cgPct, fmt.Sprintf("macOS display %s (AC3 requires ~125%% or ~150%% effective UI)", cgLine), false
	}
}
