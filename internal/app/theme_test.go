package app

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/theme"
)

func TestPhotoToolTheme_backgroundAndPrimaryDifferByVariant(t *testing.T) {
	dark := NewPhotoToolTheme(theme.VariantDark)
	light := NewPhotoToolTheme(theme.VariantLight)

	if c1, c2 := dark.Color(theme.ColorNameBackground, theme.VariantDark), light.Color(theme.ColorNameBackground, theme.VariantLight); c1 == c2 {
		t.Fatalf("background should differ: %#v vs %#v", c1, c2)
	}
	if c1, c2 := dark.Color(theme.ColorNamePrimary, theme.VariantDark), light.Color(theme.ColorNamePrimary, theme.VariantLight); c1 == c2 {
		t.Fatalf("primary should differ: %#v vs %#v", c1, c2)
	}
}

func TestPhotoToolTheme_forcedVariantIgnoresSecondArg(t *testing.T) {
	th := NewPhotoToolTheme(theme.VariantDark)
	// Second parameter is ignored; still dark palette.
	got := th.Color(theme.ColorNameBackground, theme.VariantLight)
	want := th.Color(theme.ColorNameBackground, theme.VariantDark)
	if got != want {
		t.Fatalf("expected forced variant: got %#v want %#v", got, want)
	}
}

func TestPhotoToolTheme_destructiveVsCautionDistinct(t *testing.T) {
	for _, v := range []fyne.ThemeVariant{theme.VariantDark, theme.VariantLight} {
		th := NewPhotoToolTheme(v)
		errC := th.Color(theme.ColorNameError, v)
		warnC := th.Color(theme.ColorNameWarning, v)
		if errC == warnC {
			t.Fatalf("variant %v: error and warning colors must differ", v)
		}
	}
}

// UX-DR1 / AC7: reject/caution must not read as the primary action (hue baseline for later grid chrome).
func TestPhotoToolTheme_cautionDistinctFromPrimary(t *testing.T) {
	for _, v := range []fyne.ThemeVariant{theme.VariantDark, theme.VariantLight} {
		th := NewPhotoToolTheme(v)
		prim := th.Color(theme.ColorNamePrimary, v)
		warn := th.Color(theme.ColorNameWarning, v)
		if prim == warn {
			t.Fatalf("variant %v: primary and warning must differ", v)
		}
	}
}

// Primary action must not read as destructive (UX-DR1 / AC5 semantic roles).
func TestPhotoToolTheme_primaryDistinctFromDestructive(t *testing.T) {
	for _, v := range []fyne.ThemeVariant{theme.VariantDark, theme.VariantLight} {
		th := NewPhotoToolTheme(v)
		prim := th.Color(theme.ColorNamePrimary, v)
		errC := th.Color(theme.ColorNameError, v)
		if prim == errC {
			t.Fatalf("variant %v: primary and destructive (error) must differ", v)
		}
	}
}

// Story 2.7 manual QA substitute: destructive (delete) buttons must stay visible on the app background in light and dark.
func TestPhotoToolTheme_destructiveDistinctFromBackground(t *testing.T) {
	for _, v := range []fyne.ThemeVariant{theme.VariantDark, theme.VariantLight} {
		th := NewPhotoToolTheme(v)
		errC := th.Color(theme.ColorNameError, v)
		bg := th.Color(theme.ColorNameBackground, v)
		if errC == bg {
			t.Fatalf("variant %v: destructive color must differ from background", v)
		}
	}
}

func TestLoadSaveThemeVariant_prefs(t *testing.T) {
	a := test.NewApp()
	p := a.Preferences()

	if got := LoadThemeVariantFromPrefs(p); got != theme.VariantDark {
		t.Fatalf("default: got %v want dark", got)
	}
	SaveThemeVariantToPrefs(p, theme.VariantLight)
	if got := LoadThemeVariantFromPrefs(p); got != theme.VariantLight {
		t.Fatalf("after save light: got %v", got)
	}
	SaveThemeVariantToPrefs(p, theme.VariantDark)
	if got := LoadThemeVariantFromPrefs(p); got != theme.VariantDark {
		t.Fatalf("after save dark: got %v", got)
	}
}

func TestLoadThemeVariantFromPrefs_invalidFallsBackToDark(t *testing.T) {
	a := test.NewApp()
	p := a.Preferences()
	p.SetString(prefKeyThemeVariant, "not-a-theme-token")
	if got := LoadThemeVariantFromPrefs(p); got != theme.VariantDark {
		t.Fatalf("invalid pref: got %v want dark", got)
	}
}

func TestPhotoToolTheme_focusDistinctFromBackground(t *testing.T) {
	for _, v := range []fyne.ThemeVariant{theme.VariantDark, theme.VariantLight} {
		th := NewPhotoToolTheme(v)
		f := th.Color(theme.ColorNameFocus, v)
		bg := th.Color(theme.ColorNameBackground, v)
		if f == bg {
			t.Fatalf("variant %v: focus color must differ from background for visible focus (AC10)", v)
		}
	}
}

// Focus ring must not use the same chroma as the primary action color (both are often blue-adjacent in light themes).
func TestPhotoToolTheme_focusDistinctFromPrimary(t *testing.T) {
	for _, v := range []fyne.ThemeVariant{theme.VariantDark, theme.VariantLight} {
		th := NewPhotoToolTheme(v)
		f := th.Color(theme.ColorNameFocus, v)
		prim := th.Color(theme.ColorNamePrimary, v)
		if f == prim {
			t.Fatalf("variant %v: focus and primary must differ (keyboard focus vs default action — AC10 baseline)", v)
		}
	}
}

// Reject/caution uses Warning importance; focus must not collapse into the same swatch when palette is retuned (AC10 + UX-DR5).
func TestPhotoToolTheme_focusDistinctFromWarning(t *testing.T) {
	for _, v := range []fyne.ThemeVariant{theme.VariantDark, theme.VariantLight} {
		th := NewPhotoToolTheme(v)
		f := th.Color(theme.ColorNameFocus, v)
		warn := th.Color(theme.ColorNameWarning, v)
		if f == warn {
			t.Fatalf("variant %v: focus and warning (caution) must differ", v)
		}
	}
}

// Destructive (Error) must not match focus — keyboard users tabbing past delete-adjacent controls need a visible ring (AC10 baseline).
func TestPhotoToolTheme_focusDistinctFromError(t *testing.T) {
	for _, v := range []fyne.ThemeVariant{theme.VariantDark, theme.VariantLight} {
		th := NewPhotoToolTheme(v)
		f := th.Color(theme.ColorNameFocus, v)
		errC := th.Color(theme.ColorNameError, v)
		if f == errC {
			t.Fatalf("variant %v: focus and destructive (error) must differ", v)
		}
	}
}

// AC8: border/divider roles must remain visible against background in both variants (shell separators, filter chrome).
func TestPhotoToolTheme_separatorDistinctFromBackground(t *testing.T) {
	for _, v := range []fyne.ThemeVariant{theme.VariantDark, theme.VariantLight} {
		th := NewPhotoToolTheme(v)
		sep := th.Color(theme.ColorNameSeparator, v)
		bg := th.Color(theme.ColorNameBackground, v)
		if sep == bg {
			t.Fatalf("variant %v: separator must differ from background", v)
		}
	}
}

// Story 2.2 AC4: filter strip Select widgets draw on input/surface tones; focus ring must stay visible.
func TestPhotoToolTheme_focusDistinctFromInputBackground(t *testing.T) {
	for _, v := range []fyne.ThemeVariant{theme.VariantDark, theme.VariantLight} {
		th := NewPhotoToolTheme(v)
		f := th.Color(theme.ColorNameFocus, v)
		in := th.Color(theme.ColorNameInputBackground, v)
		if f == in {
			t.Fatalf("variant %v: focus color must differ from input background (filter strip controls)", v)
		}
	}
}

func TestPhotoToolTheme_coreRolesNonZero(t *testing.T) {
	names := []fyne.ThemeColorName{
		theme.ColorNameBackground,
		theme.ColorNameInputBackground,
		theme.ColorNameButton,
		theme.ColorNameSeparator,
		theme.ColorNameInputBorder,
		theme.ColorNameForeground,
		theme.ColorNamePlaceHolder,
		theme.ColorNamePrimary,
		theme.ColorNameForegroundOnPrimary,
		theme.ColorNameError,
		theme.ColorNameForegroundOnError,
		theme.ColorNameWarning,
		theme.ColorNameForegroundOnWarning,
		theme.ColorNameFocus,
	}
	for _, v := range []fyne.ThemeVariant{theme.VariantDark, theme.VariantLight} {
		th := NewPhotoToolTheme(v)
		for _, n := range names {
			c := th.Color(n, v)
			r, g, b, a := c.RGBA()
			if r == 0 && g == 0 && b == 0 && a == 0 {
				t.Fatalf("variant %v name %q: unexpected zero color", v, n)
			}
		}
	}
}
