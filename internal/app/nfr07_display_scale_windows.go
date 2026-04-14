//go:build windows

package app

import (
	"fmt"
	"syscall"
)

var (
	user32            = syscall.NewLazyDLL("user32.dll")
	procGetDpiForSystem = user32.NewProc("GetDpiForSystem")
)

// NFR07AC3DisplayScalingPercent maps Windows system DPI to the NFR-07 tiers
// (125% → 120 DPI, 150% → 144 DPI at 96 DPI = 100% baseline).
func NFR07AC3DisplayScalingPercent() (pct int, detail string, ok bool) {
	r, _, _ := procGetDpiForSystem.Call()
	if r == 0 {
		return 0, "GetDpiForSystem returned 0", false
	}
	dpi := uint32(r)
	switch dpi {
	case 120:
		return 125, fmt.Sprintf("windows system DPI=%d (125%% tier)", dpi), true
	case 144:
		return 150, fmt.Sprintf("windows system DPI=%d (150%% tier)", dpi), true
	default:
		p := int(dpi*100/96 + 0.5)
		return p, fmt.Sprintf("windows system DPI=%d (~%d%% vs 96 DPI baseline; AC3 requires 120 or 144)", dpi, p), false
	}
}
