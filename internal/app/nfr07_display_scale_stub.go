//go:build !windows && !darwin

package app

// NFR07AC3DisplayScalingPercent returns (125 or 150, true) when the host OS
// reports display scaling in those tiers. Tier-1 structural tests use windows
// and darwin builds; other GOOS skip AC3.
func NFR07AC3DisplayScalingPercent() (pct int, detail string, ok bool) {
	return 0, "unsupported GOOS for NFR-07 AC3 display-scale probe", false
}
