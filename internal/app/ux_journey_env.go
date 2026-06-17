package app

import (
	"os"
	"strings"
)

const (
	envUXJourneyTest     = "PHOTO_TOOL_UX_JOURNEY_TEST"
	envUXCaptureDir      = "PHOTO_TOOL_UX_CAPTURE_DIR"
	envUXUploadSeedPaths = "PHOTO_TOOL_UX_UPLOAD_SEED_PATHS"
	envUXCaptureFlows    = "PHOTO_TOOL_UX_CAPTURE_FLOWS"
	envUXCaptureAppMode  = "PHOTO_TOOL_UX_CAPTURE_APP_MODE"
	envUXFixtureScale    = "PHOTO_TOOL_UX_FIXTURE_SCALE"
	envGUIE2ELinux       = "PHOTO_TOOL_GUI_E2E_LINUX"
)

// UXJourneyCaptureRequested is true when journey capture should run (test or real binary).
func UXJourneyCaptureRequested() bool {
	return strings.TrimSpace(os.Getenv(envUXJourneyTest)) == "1" &&
		strings.TrimSpace(os.Getenv(envUXCaptureDir)) != ""
}

// UXJourneyRealBinaryMode is true when the production binary should run the journey and exit.
func UXJourneyRealBinaryMode() bool {
	return UXJourneyCaptureRequested() &&
		(strings.TrimSpace(os.Getenv(envGUIE2ELinux)) == "1" ||
			strings.TrimSpace(os.Getenv(envUXCaptureAppMode)) == "real_binary")
}

// UXCaptureDir returns PHOTO_TOOL_UX_CAPTURE_DIR.
func UXCaptureDir() string {
	return strings.TrimSpace(os.Getenv(envUXCaptureDir))
}

// UXCaptureFlowFilter returns flow names from PHOTO_TOOL_UX_CAPTURE_FLOWS or nil for all.
func UXCaptureFlowFilter() []string {
	raw := strings.TrimSpace(os.Getenv(envUXCaptureFlows))
	if raw == "" {
		return nil
	}
	var out []string
	for _, p := range strings.Split(raw, ",") {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func uxUploadSeedPathsFromEnv() []string {
	raw := os.Getenv(envUXUploadSeedPaths)
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	return strings.Split(raw, "\n")
}

// UXFixtureScaleTier returns PHOTO_TOOL_UX_FIXTURE_SCALE or S1 default.
func UXFixtureScaleTier() string {
	raw := strings.TrimSpace(os.Getenv(envUXFixtureScale))
	if raw == "" {
		return "S1"
	}
	return raw
}
