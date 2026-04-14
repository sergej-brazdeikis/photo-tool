package cli

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/spf13/cobra"

	"photo-tool/internal/config"
	"photo-tool/internal/domain"
	"photo-tool/internal/store"
)

func testScanCommand() *cobra.Command {
	cmd := &cobra.Command{Use: "scan", RunE: RunScan}
	cmd.Flags().String("dir", "", "directory to scan")
	_ = cmd.MarkFlagRequired("dir")
	cmd.Flags().Bool("recursive", false, "include subdirectories")
	cmd.Flags().Bool("dry-run", false, "preview only")
	return cmd
}

func TestRunScan_exitErrorWhenFileFails(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("chmod 0 unreadable file behavior is Unix-specific")
	}
	libRoot := filepath.Join(t.TempDir(), "lib")
	scanDir := filepath.Join(t.TempDir(), "scan")
	t.Setenv(config.EnvLibraryRoot, libRoot)
	if err := config.EnsureLibraryLayout(libRoot); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(scanDir, 0o755); err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(scanDir, "locked.jpg")
	if err := writeJPEGGray(p, 0xCD); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(p, 0); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(p, 0o644) })

	cmd := testScanCommand()
	cmd.SetArgs([]string{"--dir", scanDir})
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	err := cmd.Execute()
	if err == nil {
		t.Fatal("want non-nil error when Failed > 0")
	}
	if !strings.Contains(err.Error(), "file(s) failed") {
		t.Fatalf("err: %v", err)
	}
	if !strings.Contains(buf.String(), "Failed: 1") {
		t.Fatalf("out:\n%s", buf.String())
	}
}

func TestRunScan_nonRecursive_ingestsFlatFiles(t *testing.T) {
	libRoot := filepath.Join(t.TempDir(), "lib")
	scanDir := filepath.Join(t.TempDir(), "scan")
	t.Setenv(config.EnvLibraryRoot, libRoot)
	if err := config.EnsureLibraryLayout(libRoot); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(scanDir, 0o755); err != nil {
		t.Fatal(err)
	}
	mt := time.Date(2019, 7, 8, 9, 10, 11, 0, time.UTC)
	p := filepath.Join(scanDir, "one.jpg")
	if err := writeJPEGGray(p, 0x55); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(p, mt, mt); err != nil {
		t.Fatal(err)
	}

	cmd := testScanCommand()
	cmd.SetArgs([]string{"--dir", scanDir})
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "Added: 1") || !strings.Contains(out, "Skipped duplicate: 0") {
		t.Fatalf("output:\n%s", out)
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

func TestRunScan_recursive_skipsNestedWhenFalse(t *testing.T) {
	libRoot := filepath.Join(t.TempDir(), "lib")
	scanDir := filepath.Join(t.TempDir(), "scan")
	t.Setenv(config.EnvLibraryRoot, libRoot)
	if err := config.EnsureLibraryLayout(libRoot); err != nil {
		t.Fatal(err)
	}
	sub := filepath.Join(scanDir, "nested")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	mt := time.Date(2018, 1, 2, 3, 4, 5, 0, time.UTC)
	p := filepath.Join(sub, "deep.jpg")
	if err := writeJPEGGray(p, 0x66); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(p, mt, mt); err != nil {
		t.Fatal(err)
	}

	cmd := testScanCommand()
	cmd.SetArgs([]string{"--dir", scanDir})
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "Added: 0") {
		t.Fatalf("want no files without recursive, got:\n%s", buf.String())
	}
}

func TestRunScan_secondPass_skipsDuplicates(t *testing.T) {
	libRoot := filepath.Join(t.TempDir(), "lib")
	scanDir := filepath.Join(t.TempDir(), "scan")
	t.Setenv(config.EnvLibraryRoot, libRoot)
	if err := config.EnsureLibraryLayout(libRoot); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(scanDir, 0o755); err != nil {
		t.Fatal(err)
	}
	mt := time.Date(2015, 3, 4, 5, 6, 7, 0, time.UTC)
	for i, name := range []string{"a.jpg", "b.jpg"} {
		p := filepath.Join(scanDir, name)
		if err := writeJPEGGray(p, byte(0x30+i)); err != nil {
			t.Fatal(err)
		}
		if err := os.Chtimes(p, mt, mt); err != nil {
			t.Fatal(err)
		}
	}

	cmd := testScanCommand()
	cmd.SetArgs([]string{"--dir", scanDir})
	var buf1 bytes.Buffer
	cmd.SetOut(&buf1)
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf1.String(), "Added: 2") {
		t.Fatalf("first pass want Added: 2, got:\n%s", buf1.String())
	}

	cmd2 := testScanCommand()
	cmd2.SetArgs([]string{"--dir", scanDir})
	var buf2 bytes.Buffer
	cmd2.SetOut(&buf2)
	if err := cmd2.Execute(); err != nil {
		t.Fatal(err)
	}
	out2 := buf2.String()
	if !strings.Contains(out2, "Added: 0") || !strings.Contains(out2, "Skipped duplicate: 2") {
		t.Fatalf("second pass want duplicates only, got:\n%s", out2)
	}
}

func TestRunScan_dryRun_afterLive_skipsAllDuplicates(t *testing.T) {
	libRoot := filepath.Join(t.TempDir(), "lib")
	scanDir := filepath.Join(t.TempDir(), "scan")
	t.Setenv(config.EnvLibraryRoot, libRoot)
	if err := config.EnsureLibraryLayout(libRoot); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(scanDir, 0o755); err != nil {
		t.Fatal(err)
	}
	mt := time.Date(2014, 8, 9, 10, 11, 12, 0, time.UTC)
	for i, name := range []string{"x.jpg", "y.jpg"} {
		p := filepath.Join(scanDir, name)
		if err := writeJPEGGray(p, byte(0x40+i)); err != nil {
			t.Fatal(err)
		}
		if err := os.Chtimes(p, mt, mt); err != nil {
			t.Fatal(err)
		}
	}

	cmdLive := testScanCommand()
	cmdLive.SetArgs([]string{"--dir", scanDir})
	if err := cmdLive.Execute(); err != nil {
		t.Fatal(err)
	}

	cmdDry := testScanCommand()
	cmdDry.SetArgs([]string{"--dir", scanDir, "--dry-run"})
	var buf bytes.Buffer
	cmdDry.SetOut(&buf)
	if err := cmdDry.Execute(); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "Added: 0") || !strings.Contains(out, "Skipped duplicate: 2") {
		t.Fatalf("dry-run after live want0 added, 2 skipped; got:\n%s", out)
	}
}

func TestScanDirInsideLibrary(t *testing.T) {
	tmp := t.TempDir()
	lib := filepath.Join(tmp, "library")
	inside := filepath.Join(lib, "nested")
	outside := filepath.Join(tmp, "outside")
	for _, p := range []string{inside, outside} {
		if err := os.MkdirAll(p, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if !scanDirInsideLibrary(lib, lib) {
		t.Fatal("library root should count as inside")
	}
	if !scanDirInsideLibrary(lib, inside) {
		t.Fatal("nested dir under library should count as inside")
	}
	if scanDirInsideLibrary(lib, outside) {
		t.Fatal("sibling path must not count as inside library")
	}
}

// TestRunScan_dryRun_countsMatch_liveSeparateLibraries asserts AC2 dry-run classification matches a
// live run on an empty DB when both see the same source tree (parity across two fresh libraries).
func TestRunScan_dryRun_countsMatch_liveSeparateLibraries(t *testing.T) {
	scanDir := filepath.Join(t.TempDir(), "scan")
	if err := os.MkdirAll(scanDir, 0o755); err != nil {
		t.Fatal(err)
	}
	mt := time.Date(2013, 2, 3, 4, 5, 6, 0, time.UTC)
	for i, name := range []string{"a.jpg", "b.jpg", "c.jpg"} {
		p := filepath.Join(scanDir, name)
		if err := writeJPEGGray(p, byte(0x50+i)); err != nil {
			t.Fatal(err)
		}
		if err := os.Chtimes(p, mt, mt); err != nil {
			t.Fatal(err)
		}
	}

	libDry := filepath.Join(t.TempDir(), "lib-dry")
	libLive := filepath.Join(t.TempDir(), "lib-live")
	for _, lr := range []string{libDry, libLive} {
		if err := config.EnsureLibraryLayout(lr); err != nil {
			t.Fatal(err)
		}
	}

	t.Setenv(config.EnvLibraryRoot, libDry)
	cmdDry := testScanCommand()
	cmdDry.SetArgs([]string{"--dir", scanDir, "--dry-run"})
	var bufDry bytes.Buffer
	cmdDry.SetOut(&bufDry)
	if err := cmdDry.Execute(); err != nil {
		t.Fatal(err)
	}
	sumDry := scanSummaryFromOutput(t, bufDry.String())

	t.Setenv(config.EnvLibraryRoot, libLive)
	cmdLive := testScanCommand()
	cmdLive.SetArgs([]string{"--dir", scanDir})
	var bufLive bytes.Buffer
	cmdLive.SetOut(&bufLive)
	if err := cmdLive.Execute(); err != nil {
		t.Fatal(err)
	}
	sumLive := scanSummaryFromOutput(t, bufLive.String())

	if sumDry != sumLive {
		t.Fatalf("dry %+v vs live %+v", sumDry, sumLive)
	}
	if sumLive.Added != 3 || sumLive.SkippedDuplicate != 0 || sumLive.Failed != 0 {
		t.Fatalf("unexpected live summary: %+v", sumLive)
	}
}

// TestRunScan_dryRun_countsMatch_liveSeparateLibraries_recursive is session-2 hardening: flat-dir parity
// does not exercise filepath.WalkDir; recursive mode must still classify identically for dry vs live.
func TestRunScan_dryRun_countsMatch_liveSeparateLibraries_recursive(t *testing.T) {
	scanDir := filepath.Join(t.TempDir(), "scan")
	nested := filepath.Join(scanDir, "nested")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}
	mt := time.Date(2011, 9, 8, 7, 6, 5, 0, time.UTC)
	files := []struct {
		rel  string
		gray byte
	}{
		{"root.jpg", 0x60},
		{filepath.Join("nested", "a.jpg"), 0x61},
		{filepath.Join("nested", "b.jpg"), 0x62},
	}
	for _, f := range files {
		p := filepath.Join(scanDir, f.rel)
		if err := writeJPEGGray(p, f.gray); err != nil {
			t.Fatal(err)
		}
		if err := os.Chtimes(p, mt, mt); err != nil {
			t.Fatal(err)
		}
	}

	libDry := filepath.Join(t.TempDir(), "lib-dry-rec")
	libLive := filepath.Join(t.TempDir(), "lib-live-rec")
	for _, lr := range []string{libDry, libLive} {
		if err := config.EnsureLibraryLayout(lr); err != nil {
			t.Fatal(err)
		}
	}

	t.Setenv(config.EnvLibraryRoot, libDry)
	cmdDry := testScanCommand()
	cmdDry.SetArgs([]string{"--dir", scanDir, "--recursive", "--dry-run"})
	var bufDry bytes.Buffer
	cmdDry.SetOut(&bufDry)
	if err := cmdDry.Execute(); err != nil {
		t.Fatal(err)
	}
	sumDry := scanSummaryFromOutput(t, bufDry.String())

	t.Setenv(config.EnvLibraryRoot, libLive)
	cmdLive := testScanCommand()
	cmdLive.SetArgs([]string{"--dir", scanDir, "--recursive"})
	var bufLive bytes.Buffer
	cmdLive.SetOut(&bufLive)
	if err := cmdLive.Execute(); err != nil {
		t.Fatal(err)
	}
	sumLive := scanSummaryFromOutput(t, bufLive.String())

	if sumDry != sumLive {
		t.Fatalf("recursive dry %+v vs live %+v", sumDry, sumLive)
	}
	if sumLive.Added != 3 || sumLive.SkippedDuplicate != 0 || sumLive.Failed != 0 {
		t.Fatalf("unexpected live summary: %+v", sumLive)
	}
}

func TestRunScan_dryRun_noAssetRows(t *testing.T) {
	libRoot := filepath.Join(t.TempDir(), "lib")
	scanDir := filepath.Join(t.TempDir(), "scan")
	t.Setenv(config.EnvLibraryRoot, libRoot)
	if err := config.EnsureLibraryLayout(libRoot); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(scanDir, 0o755); err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(scanDir, "x.jpg")
	mt := time.Date(2017, 6, 5, 4, 3, 2, 0, time.UTC)
	if err := writeJPEGGray(p, 0x77); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(p, mt, mt); err != nil {
		t.Fatal(err)
	}

	cmd := testScanCommand()
	cmd.SetArgs([]string{"--dir", scanDir, "--dry-run"})
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "Added: 1") {
		t.Fatalf("got:\n%s", buf.String())
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
		t.Fatalf("dry-run wrote rows: %d", n)
	}
}

// BenchmarkScanWalkDir_processingPattern documents NFR-02: traversal does not collect all paths;
// each directory entry is handled and released before the next (filepath.WalkDir streaming pattern).
func BenchmarkScanWalkDir_processingPattern(b *testing.B) {
	root := b.TempDir()
	sub := filepath.Join(root, "nested")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		b.Fatal(err)
	}
	mt := time.Date(2016, 5, 4, 3, 2, 1, 0, time.UTC)
	for i := range 500 {
		p := filepath.Join(sub, fmt.Sprintf("f%d.jpg", i))
		if err := writeJPEGGray(p, byte(i)); err != nil {
			b.Fatal(err)
		}
		if err := os.Chtimes(p, mt, mt); err != nil {
			b.Fatal(err)
		}
	}
	b.ResetTimer()
	for range b.N {
		_ = filepath.WalkDir(root, func(string, fs.DirEntry, error) error { return nil })
	}
}

func scanSummaryFromOutput(t *testing.T, out string) domain.OperationSummary {
	t.Helper()
	var s domain.OperationSummary
	for _, line := range strings.Split(strings.ReplaceAll(out, "\r\n", "\n"), "\n") {
		line = strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(line, "Added: "):
			s.Added = mustAtoiSuffix(t, line, "Added: ")
		case strings.HasPrefix(line, "Skipped duplicate: "):
			s.SkippedDuplicate = mustAtoiSuffix(t, line, "Skipped duplicate: ")
		case strings.HasPrefix(line, "Updated: "):
			s.Updated = mustAtoiSuffix(t, line, "Updated: ")
		case strings.HasPrefix(line, "Failed: "):
			s.Failed = mustAtoiSuffix(t, line, "Failed: ")
		}
	}
	return s
}

func mustAtoiSuffix(t *testing.T, line, prefix string) int {
	t.Helper()
	n, err := strconv.Atoi(strings.TrimSpace(strings.TrimPrefix(line, prefix)))
	if err != nil {
		t.Fatalf("parse int from %q: %v", line, err)
	}
	return n
}

func writeJPEGGray(path string, y byte) error {
	img := image.NewGray(image.Rect(0, 0, 2, 2))
	img.Set(0, 0, color.Gray{Y: y})
	img.Set(1, 0, color.Gray{Y: y})
	img.Set(0, 1, color.Gray{Y: y ^ 1})
	img.Set(1, 1, color.Gray{Y: y})
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return jpeg.Encode(f, img, &jpeg.Options{Quality: 80})
}
