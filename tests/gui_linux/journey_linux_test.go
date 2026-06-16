//go:build linux

package gui_linux_test

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"photo-tool/internal/config"
)

func moduleRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("go.mod not found")
		}
		dir = parent
	}
}

func TestLinuxGUIE2E_smokeBinaryStarts(t *testing.T) {
	if os.Getenv("PHOTO_TOOL_GUI_E2E_LINUX") != "1" {
		t.Skip("set PHOTO_TOOL_GUI_E2E_LINUX=1 to run real GUI smoke")
	}
	if os.Getenv("DISPLAY") == "" && os.Getenv("WAYLAND_DISPLAY") == "" {
		t.Skip("DISPLAY or WAYLAND_DISPLAY required for real GUI")
	}

	repo := moduleRoot(t)
	lib := t.TempDir()
	if err := config.EnsureLibraryLayout(lib); err != nil {
		t.Fatal(err)
	}
	bin := filepath.Join(t.TempDir(), "photo-tool")
	build := exec.Command("go", "build", "-o", bin, ".")
	build.Dir = repo
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build: %v\n%s", err, out)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, bin)
	cmd.Env = append(os.Environ(), config.EnvLibraryRoot+"="+lib)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = cmd.Process.Kill()
		_, _ = cmd.Process.Wait()
	})
	time.Sleep(1500 * time.Millisecond)
	if cmd.ProcessState != nil && cmd.ProcessState.Exited() {
		t.Fatalf("binary exited early: %s", stderr.String())
	}
}

func TestLinuxGUIE2E_journeyRealAppCapture(t *testing.T) {
	if os.Getenv("PHOTO_TOOL_GUI_E2E_LINUX") != "1" {
		t.Skip("set PHOTO_TOOL_GUI_E2E_LINUX=1 to run real GUI journey")
	}
	if os.Getenv("DISPLAY") == "" && os.Getenv("WAYLAND_DISPLAY") == "" {
		t.Skip("DISPLAY or WAYLAND_DISPLAY required")
	}

	repo := moduleRoot(t)
	lib := t.TempDir()
	captureDir := filepath.Join(t.TempDir(), "ui-real")
	if err := config.EnsureLibraryLayout(lib); err != nil {
		t.Fatal(err)
	}
	bin := filepath.Join(t.TempDir(), "photo-tool")
	build := exec.Command("go", "build", "-o", bin, ".")
	build.Dir = repo
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build: %v\n%s", err, out)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, bin)
	cmd.Env = append(os.Environ(),
		config.EnvLibraryRoot+"="+lib,
		"PHOTO_TOOL_UX_JOURNEY_TEST=1",
		"PHOTO_TOOL_UX_CAPTURE_DIR="+captureDir,
		"PHOTO_TOOL_GUI_E2E_LINUX=1",
		"PHOTO_TOOL_UX_CAPTURE_APP_MODE=real_binary",
	)
	var combined bytes.Buffer
	cmd.Stdout = &combined
	cmd.Stderr = &combined
	if err := cmd.Run(); err != nil {
		t.Fatalf("journey binary: %v\n%s", err, combined.String())
	}

	stepsPath := filepath.Join(captureDir, "steps.json")
	data, err := os.ReadFile(stepsPath)
	if err != nil {
		t.Fatalf("steps.json: %v", err)
	}
	if !strings.Contains(string(data), `"app_mode": "real_binary"`) &&
		!strings.Contains(string(data), `"app_mode":"real_binary"`) {
		t.Fatalf("steps.json missing real_binary app_mode: %s", string(data))
	}
	entries, err := os.ReadDir(captureDir)
	if err != nil {
		t.Fatal(err)
	}
	var pngs int
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".png") {
			pngs++
		}
	}
	if pngs < 5 {
		t.Fatalf("expected at least 5 PNGs, got %d", pngs)
	}
}
