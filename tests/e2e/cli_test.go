package e2e

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"photo-tool/internal/config"
	"photo-tool/internal/store"
)

// e2eBinPath returns path to photo-tool binary: PHOTO_TOOL_E2E_BIN if set, else builds once to test temp.
var e2eBinOnce sync.Once
var e2eBinCached string
var e2eBinErr error

func photoToolBin(t *testing.T) string {
	t.Helper()
	if p := os.Getenv("PHOTO_TOOL_E2E_BIN"); p != "" {
		return p
	}
	e2eBinOnce.Do(func() {
		dir, err := os.MkdirTemp("", "photo-tool-e2e-bin-*")
		if err != nil {
			e2eBinErr = err
			return
		}
		out := filepath.Join(dir, "photo-tool"+exeSuffix())
		cmd := exec.Command("go", "build", "-o", out, ".")
		cmd.Dir = moduleRoot(t)
		var stderr bytes.Buffer
		cmd.Stderr = &stderr
		if err := cmd.Run(); err != nil {
			e2eBinErr = err
			_ = os.RemoveAll(dir)
			return
		}
		e2eBinCached = out
	})
	if e2eBinErr != nil {
		t.Fatalf("build photo-tool for e2e: %v", e2eBinErr)
	}
	return e2eBinCached
}

func exeSuffix() string {
	if runtime.GOOS == "windows" {
		return ".exe"
	}
	return ""
}

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
			t.Fatal("go.mod not found from cwd")
		}
		dir = parent
	}
}

func writeJPEGGray(t *testing.T, path string, y byte) {
	t.Helper()
	img := image.NewGray(image.Rect(0, 0, 2, 2))
	img.Set(0, 0, color.Gray{Y: y})
	img.Set(1, 0, color.Gray{Y: y})
	img.Set(0, 1, color.Gray{Y: y ^ 1})
	img.Set(1, 1, color.Gray{Y: y})
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err := jpeg.Encode(f, img, &jpeg.Options{Quality: 80}); err != nil {
		t.Fatal(err)
	}
}

func runPhotoTool(t *testing.T, env []string, args ...string) (stdout, stderr string, exitCode int) {
	t.Helper()
	bin := photoToolBin(t)
	cmd := exec.Command(bin, args...)
	cmd.Env = append(os.Environ(), env...)
	var outBuf, errBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf
	err := cmd.Run()
	exitCode = 0
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			exitCode = ee.ExitCode()
		} else {
			t.Fatalf("run phototool: %v", err)
		}
	}
	return outBuf.String(), errBuf.String(), exitCode
}

func TestCLI_scan_ingestsJPEG(t *testing.T) {
	libRoot := filepath.Join(t.TempDir(), "lib")
	scanDir := filepath.Join(t.TempDir(), "scan")
	if err := config.EnsureLibraryLayout(libRoot); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(scanDir, 0o755); err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(scanDir, "e2e-one.jpg")
	mt := time.Date(2020, 6, 15, 12, 0, 0, 0, time.UTC)
	writeJPEGGray(t, p, 0x42)
	if err := os.Chtimes(p, mt, mt); err != nil {
		t.Fatal(err)
	}

	env := []string{config.EnvLibraryRoot + "=" + libRoot}
	out, errOut, code := runPhotoTool(t, env, "scan", "--dir", scanDir)
	if code != 0 {
		t.Fatalf("exit %d stderr=%q stdout=%q", code, errOut, out)
	}
	if !strings.Contains(out, "Added: 1") || !strings.Contains(out, "Skipped duplicate: 0") {
		t.Fatalf("stdout:\n%s", out)
	}

	db, err := store.Open(libRoot)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM assets`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("assets count: got %d want 1", n)
	}
}

func TestCLI_scan_dryRun_noDatabaseWrites(t *testing.T) {
	libRoot := filepath.Join(t.TempDir(), "lib")
	scanDir := filepath.Join(t.TempDir(), "scan")
	if err := config.EnsureLibraryLayout(libRoot); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(scanDir, 0o755); err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(scanDir, "dry.jpg")
	writeJPEGGray(t, p, 0x99)
	if err := os.Chtimes(p, time.Now(), time.Now()); err != nil {
		t.Fatal(err)
	}

	env := []string{config.EnvLibraryRoot + "=" + libRoot}
	out, errOut, code := runPhotoTool(t, env, "scan", "--dir", scanDir, "--dry-run")
	if code != 0 {
		t.Fatalf("exit %d stderr=%q stdout=%q", code, errOut, out)
	}
	if !strings.Contains(out, "Added: 1") {
		t.Fatalf("stdout:\n%s", out)
	}

	db, err := store.Open(libRoot)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM assets`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Fatalf("dry-run wrote assets: got %d want 0", n)
	}
}

func TestCLI_import_registersFileUnderLibrary(t *testing.T) {
	libRoot := filepath.Join(t.TempDir(), "lib")
	if err := config.EnsureLibraryLayout(libRoot); err != nil {
		t.Fatal(err)
	}
	day := filepath.Join(libRoot, "2021", "08", "09")
	p := filepath.Join(day, "registered.jpg")
	mt := time.Date(2021, 8, 9, 10, 11, 12, 0, time.UTC)
	writeJPEGGray(t, p, 0x5E)
	if err := os.Chtimes(p, mt, mt); err != nil {
		t.Fatal(err)
	}

	env := []string{config.EnvLibraryRoot + "=" + libRoot}
	out, errOut, code := runPhotoTool(t, env, "import", "--dir", day)
	if code != 0 {
		t.Fatalf("exit %d stderr=%q stdout=%q", code, errOut, out)
	}
	if !strings.Contains(out, "Added: 1") {
		t.Fatalf("stdout:\n%s", out)
	}

	db, err := store.Open(libRoot)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM assets`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("assets: got %d want 1", n)
	}
}

func TestCLI_scan_missingDir_nonZero(t *testing.T) {
	libRoot := filepath.Join(t.TempDir(), "lib")
	if err := config.EnsureLibraryLayout(libRoot); err != nil {
		t.Fatal(err)
	}
	env := []string{config.EnvLibraryRoot + "=" + libRoot}
	_, errOut, code := runPhotoTool(t, env, "scan")
	if code == 0 {
		t.Fatal("want non-zero exit when --dir missing")
	}
	combined := errOut
	if !strings.Contains(combined, "dir") && !strings.Contains(strings.ToLower(combined), "required") {
		t.Fatalf("stderr: %q", errOut)
	}
}

func TestCLI_import_dirOutsideLibrary_nonZero(t *testing.T) {
	libRoot := filepath.Join(t.TempDir(), "lib")
	outside := filepath.Join(t.TempDir(), "outside")
	if err := config.EnsureLibraryLayout(libRoot); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(outside, 0o755); err != nil {
		t.Fatal(err)
	}

	env := []string{config.EnvLibraryRoot + "=" + libRoot}
	_, errOut, code := runPhotoTool(t, env, "import", "--dir", outside)
	if code == 0 {
		t.Fatal("want non-zero exit for import dir outside library")
	}
	if !strings.Contains(errOut, "library") {
		t.Fatalf("stderr: %q", errOut)
	}
}
