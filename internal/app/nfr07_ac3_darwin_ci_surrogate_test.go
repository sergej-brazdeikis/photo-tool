//go:build darwin

package app

import (
	"strings"
	"testing"
)

func TestNFR07AC3DarwinCISurrogate_requiresGitHubActions(t *testing.T) {
	t.Setenv("GITHUB_ACTIONS", "")
	t.Setenv("PHOTO_TOOL_NFR07_MACOS_CI_TIER", "125")
	if _, _, ok := nfr07AC3DarwinCISurrogate(); ok {
		t.Fatal("expected no surrogate without GITHUB_ACTIONS")
	}
}

func TestNFR07AC3DisplayScalingPercent_macOSCISurrogate(t *testing.T) {
	t.Setenv("GITHUB_ACTIONS", "true")
	t.Setenv("PHOTO_TOOL_NFR07_MACOS_CI_TIER", "125")
	t.Setenv("FYNE_SCALE", "1.25")
	pct, detail, ok := NFR07AC3DisplayScalingPercent()
	if !ok || pct != 125 {
		t.Fatalf("got ok=%v pct=%d detail=%q", ok, pct, detail)
	}
	if detail == "" {
		t.Fatal("empty detail")
	}
	if !strings.Contains(detail, "surrogate tier=125%") {
		t.Fatalf("detail %q should document CI surrogate tier", detail)
	}
	if !strings.Contains(detail, "CoreGraphics") && !strings.Contains(detail, "no CoreGraphics probe") {
		t.Fatalf("detail %q should mention CoreGraphics observation or nocgo limitation", detail)
	}
	t.Setenv("FYNE_SCALE", "1.0")
	_, _, ok = NFR07AC3DisplayScalingPercent()
	if ok {
		t.Fatal("expected fail when FYNE_SCALE does not match tier")
	}
}

func TestNFR07AC3DarwinCISurrogate_tier150(t *testing.T) {
	t.Setenv("GITHUB_ACTIONS", "true")
	t.Setenv("PHOTO_TOOL_NFR07_MACOS_CI_TIER", "150")
	if tier, fyne, ok := nfr07AC3DarwinCISurrogate(); !ok || tier != 150 || fyne != "1.5" {
		t.Fatalf("got tier=%d fyne=%q ok=%v", tier, fyne, ok)
	}
}
