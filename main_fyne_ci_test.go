//go:build ci

package main

import (
	"testing"

	fyneapp "fyne.io/fyne/v2/app"

	ptapp "photo-tool/internal/app"
)

// Uses Fyne's software driver (go test -tags ci). Catches missing NewWithID at runtime.
func TestFyneNewWithID_uniqueIDForPreferences(t *testing.T) {
	a := fyneapp.NewWithID(ptapp.FyneAppID)
	t.Cleanup(func() { a.Quit() })
	if got := a.UniqueID(); got != ptapp.FyneAppID {
		t.Fatalf("UniqueID: got %q want %q", got, ptapp.FyneAppID)
	}
}
