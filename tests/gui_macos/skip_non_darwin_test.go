//go:build !darwin

package gui_macos_test

import "testing"

func TestMacOSGUIE2ESkippedOnNonDarwin(t *testing.T) {
	t.Skip("black-box GUI E2E runs only on macOS with CGO; see tests/gui_macos/README.md")
}
