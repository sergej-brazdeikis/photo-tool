//go:build !linux

package gui_linux

import "testing"

func TestLinuxGUIE2ESkippedOnNonLinux(t *testing.T) {
	t.Skip("Linux GUI E2E runs only on linux; see tests/gui_linux/README.md")
}
