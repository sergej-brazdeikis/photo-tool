package app

import (
	"os"
	"strings"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/theme"
)

// applyTestPhotoToolTheme applies [NewPhotoToolTheme] for Fyne tests, optionally overridden by env:
//
//	PHOTO_TOOL_TEST_THEME=light | white  → theme.VariantLight
//	PHOTO_TOOL_TEST_THEME=dark           → theme.VariantDark
//	unset or other                     → defaultVariant
//
// Use for tests that pick a single default (usually dark). Do not use in tests that must sweep both variants (e.g. NFR-01 matrix).
func applyTestPhotoToolTheme(t *testing.T, defaultVariant fyne.ThemeVariant) {
	t.Helper()
	test.ApplyTheme(t, NewPhotoToolTheme(testThemeVariantOrDefault(defaultVariant)))
}

func testThemeVariantOrDefault(defaultVariant fyne.ThemeVariant) fyne.ThemeVariant {
	switch strings.ToLower(strings.TrimSpace(os.Getenv("PHOTO_TOOL_TEST_THEME"))) {
	case "light", "white":
		return theme.VariantLight
	case "dark":
		return theme.VariantDark
	default:
		return defaultVariant
	}
}
