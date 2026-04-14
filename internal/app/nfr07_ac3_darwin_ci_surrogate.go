//go:build darwin

package app

import "os"

// nfr07AC3DarwinCISurrogate reports GitHub Actions macOS matrix jobs that pin
// NFR-07 tiers with FYNE_SCALE (GHA cannot drive System Settings display scaling).
func nfr07AC3DarwinCISurrogate() (tier int, fyneScale string, ok bool) {
	if os.Getenv("GITHUB_ACTIONS") != "true" {
		return 0, "", false
	}
	switch os.Getenv("PHOTO_TOOL_NFR07_MACOS_CI_TIER") {
	case "125":
		return 125, "1.25", true
	case "150":
		return 150, "1.5", true
	default:
		return 0, "", false
	}
}
