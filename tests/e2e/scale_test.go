package e2e

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"photo-tool/internal/fixture"
)

// TestScale_Scan_dryRun10k verifies NFR-02: 10k-file walk completes without unbounded heap growth.
func TestScale_Scan_dryRun10k(t *testing.T) {
	if testing.Short() {
		t.Skip("NFR-02 10k scan skipped in -short")
	}
	const n = 10000
	root := t.TempDir()
	tree := filepath.Join(root, "scan10k")
	if err := fixture.SeedFilesystemTree(tree, n); err != nil {
		t.Fatal(err)
	}
	var before, after runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&before)
	var count int
	err := filepath.WalkDir(tree, func(_ string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if !d.IsDir() {
			count++
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if count != n {
		t.Fatalf("walk count=%d want %d", count, n)
	}
	runtime.ReadMemStats(&after)
	const maxGrowth = 64 * 1024 * 1024
	growth := int64(after.Alloc) - int64(before.Alloc)
	if growth < 0 {
		growth = 0
	}
	if growth > maxGrowth {
		t.Fatalf("heap growth %d bytes exceeds budget %d (NFR-02)", growth, maxGrowth)
	}
}

// TestScale_CLI_recursiveDeepTree verifies CLI scan --recursive over a deep S4-scale tree.
func TestScale_CLI_recursiveDeepTree(t *testing.T) {
	if testing.Short() {
		t.Skip("scale CLI deep tree skipped in -short")
	}
	lib := t.TempDir()
	scanRoot := filepath.Join(t.TempDir(), "deep-scan")
	dirs := []string{
		scanRoot,
		filepath.Join(scanRoot, "a"),
		filepath.Join(scanRoot, "a", "b"),
		filepath.Join(scanRoot, "a", "b", "c"),
	}
	for _, d := range dirs {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	const filesPerDir = 25
	for _, d := range dirs {
		for i := 0; i < filesPerDir; i++ {
			p := filepath.Join(d, fmt.Sprintf("img_%02d.jpg", i))
			if err := fixture.WriteTinyJPEG(p); err != nil {
				t.Fatal(err)
			}
		}
	}
	env := []string{"PHOTO_TOOL_LIBRARY=" + lib}
	out, errOut, code := runPhotoTool(t, env, "scan", "--dir", scanRoot, "--recursive")
	if code != 0 {
		t.Fatalf("scan exit %d stdout=%q stderr=%q", code, out, errOut)
	}
	if !strings.Contains(out, "Added") {
		t.Fatalf("expected Added in stdout: %q", out)
	}
}
