package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/spf13/cobra"

	"photo-tool/internal/config"
	"photo-tool/internal/store"
)

func testImportCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "import",
		RunE:          RunImport,
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	cmd.Flags().String("dir", "", "directory to import")
	_ = cmd.MarkFlagRequired("dir")
	cmd.Flags().Bool("recursive", false, "include subdirectories")
	cmd.Flags().Bool("dry-run", false, "preview only")
	return cmd
}

func TestRunImport_exitErrorWhenFileFails(t *testing.T) {
	libRoot := filepath.Join(t.TempDir(), "lib")
	t.Setenv(config.EnvLibraryRoot, libRoot)
	if err := config.EnsureLibraryLayout(libRoot); err != nil {
		t.Fatal(err)
	}
	day := filepath.Join(libRoot, "2022", "04", "05")
	if err := os.MkdirAll(day, 0o755); err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(day, "conflict.jpg")
	mt := time.Date(2022, 4, 5, 6, 7, 8, 0, time.UTC)
	if err := writeJPEGGray(p, 0x3C); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(p, mt, mt); err != nil {
		t.Fatal(err)
	}

	cmd1 := testImportCommand()
	cmd1.SetArgs([]string{"--dir", day})
	if err := cmd1.Execute(); err != nil {
		t.Fatal(err)
	}
	if err := writeJPEGGray(p, 0xEF); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(p, mt, mt); err != nil {
		t.Fatal(err)
	}

	cmd2 := testImportCommand()
	cmd2.SetArgs([]string{"--dir", day})
	var buf bytes.Buffer
	cmd2.SetOut(&buf)
	err := cmd2.Execute()
	if err == nil {
		t.Fatal("want non-nil error when Failed > 0")
	}
	if !strings.Contains(err.Error(), "file(s) failed") {
		t.Fatalf("err: %v", err)
	}
	if !strings.Contains(buf.String(), "Failed: 1") {
		t.Fatalf("out:\n%s", buf.String())
	}
	assertOperationReceiptLineOrder(t, buf.String())
}

func TestRunImport_rejectsDirOutsideLibrary(t *testing.T) {
	libRoot := filepath.Join(t.TempDir(), "lib")
	ext := filepath.Join(t.TempDir(), "external")
	for _, p := range []string{libRoot, ext} {
		if err := os.MkdirAll(p, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv(config.EnvLibraryRoot, libRoot)
	if err := config.EnsureLibraryLayout(libRoot); err != nil {
		t.Fatal(err)
	}

	cmd := testImportCommand()
	cmd.SetArgs([]string{"--dir", ext})
	if err := cmd.Execute(); err == nil {
		t.Fatal("want error for dir outside library")
	} else if !strings.Contains(err.Error(), "must be under library root") {
		t.Fatalf("err: %v", err)
	}
}

func TestRunImport_nonRecursive_registersFlatFile(t *testing.T) {
	libRoot := filepath.Join(t.TempDir(), "lib")
	t.Setenv(config.EnvLibraryRoot, libRoot)
	if err := config.EnsureLibraryLayout(libRoot); err != nil {
		t.Fatal(err)
	}
	day := filepath.Join(libRoot, "2022", "04", "05")
	if err := os.MkdirAll(day, 0o755); err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(day, "cli.jpg")
	mt := time.Date(2022, 4, 5, 6, 7, 8, 0, time.UTC)
	if err := writeJPEGGray(p, 0x3C); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(p, mt, mt); err != nil {
		t.Fatal(err)
	}

	cmd := testImportCommand()
	cmd.SetArgs([]string{"--dir", day})
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "Added: 1") {
		t.Fatalf("out:\n%s", buf.String())
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
		t.Fatalf("assets: %d", n)
	}
}

func TestRunImport_dryRun_noRows(t *testing.T) {
	libRoot := filepath.Join(t.TempDir(), "lib")
	t.Setenv(config.EnvLibraryRoot, libRoot)
	if err := config.EnsureLibraryLayout(libRoot); err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(libRoot, "solo.jpg")
	mt := time.Date(2023, 1, 2, 3, 4, 5, 0, time.UTC)
	if err := writeJPEGGray(p, 0x2A); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(p, mt, mt); err != nil {
		t.Fatal(err)
	}

	cmd := testImportCommand()
	cmd.SetArgs([]string{"--dir", libRoot, "--dry-run"})
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "Added: 1") {
		t.Fatalf("out:\n%s", buf.String())
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
		t.Fatalf("dry-run wrote %d rows", n)
	}
}

func TestRunImport_recursive_registersNestedFiles(t *testing.T) {
	libRoot := filepath.Join(t.TempDir(), "lib")
	t.Setenv(config.EnvLibraryRoot, libRoot)
	if err := config.EnsureLibraryLayout(libRoot); err != nil {
		t.Fatal(err)
	}
	sub := filepath.Join(libRoot, "deep", "nest")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	mt := time.Date(2021, 8, 9, 10, 11, 12, 0, time.UTC)
	for _, rel := range []string{"root.jpg", filepath.Join("deep", "nest", "leaf.jpg")} {
		p := filepath.Join(libRoot, rel)
		if err := writeJPEGGray(p, 0x5A+byte(len(rel))); err != nil {
			t.Fatal(err)
		}
		if err := os.Chtimes(p, mt, mt); err != nil {
			t.Fatal(err)
		}
	}

	cmd := testImportCommand()
	cmd.SetArgs([]string{"--dir", libRoot, "--recursive"})
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "Added: 2") {
		t.Fatalf("out:\n%s", buf.String())
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
	if n != 2 {
		t.Fatalf("assets: %d", n)
	}
}

func TestRunImport_nonRecursive_skipsNestedFiles(t *testing.T) {
	libRoot := filepath.Join(t.TempDir(), "lib")
	t.Setenv(config.EnvLibraryRoot, libRoot)
	if err := config.EnsureLibraryLayout(libRoot); err != nil {
		t.Fatal(err)
	}
	sub := filepath.Join(libRoot, "deep")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(sub, "hidden.jpg")
	mt := time.Date(2024, 3, 4, 5, 6, 7, 0, time.UTC)
	if err := writeJPEGGray(p, 0x7E); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(p, mt, mt); err != nil {
		t.Fatal(err)
	}

	cmd := testImportCommand()
	cmd.SetArgs([]string{"--dir", libRoot})
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "Added: 0") {
		t.Fatalf("non-recursive should not read subdirs; out:\n%s", buf.String())
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
		t.Fatalf("assets: %d", n)
	}
}

func TestRunImport_dryRun_backfillClassifiesUpdatedWithoutDBWrite(t *testing.T) {
	libRoot := filepath.Join(t.TempDir(), "lib")
	t.Setenv(config.EnvLibraryRoot, libRoot)
	if err := config.EnsureLibraryLayout(libRoot); err != nil {
		t.Fatal(err)
	}
	day := filepath.Join(libRoot, "2020", "01", "02")
	if err := os.MkdirAll(day, 0o755); err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(day, "meta-dry.jpg")
	mt := time.Date(2020, 1, 2, 3, 4, 5, 0, time.UTC)
	if err := writeJPEGGray(p, 0xD1); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(p, mt, mt); err != nil {
		t.Fatal(err)
	}

	cmd := testImportCommand()
	cmd.SetArgs([]string{"--dir", day})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}

	db, err := store.Open(libRoot)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	wrong := int64(424242)
	_, err = db.Exec(`UPDATE assets SET capture_time_unix = ? WHERE rel_path = ?`, wrong, "2020/01/02/meta-dry.jpg")
	if err != nil {
		t.Fatal(err)
	}

	cmd2 := testImportCommand()
	cmd2.SetArgs([]string{"--dir", day, "--dry-run"})
	var buf bytes.Buffer
	cmd2.SetOut(&buf)
	if err := cmd2.Execute(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "Updated: 1") {
		t.Fatalf("dry-run should classify backfill; out:\n%s", buf.String())
	}

	var cap int64
	if err := db.QueryRow(`SELECT capture_time_unix FROM assets WHERE rel_path = ?`, "2020/01/02/meta-dry.jpg").Scan(&cap); err != nil {
		t.Fatal(err)
	}
	if cap != wrong {
		t.Fatalf("dry-run must not write DB; capture_time_unix: got %d want %d", cap, wrong)
	}
}

func TestRunImport_backfillsStaleCaptureTime(t *testing.T) {
	libRoot := filepath.Join(t.TempDir(), "lib")
	t.Setenv(config.EnvLibraryRoot, libRoot)
	if err := config.EnsureLibraryLayout(libRoot); err != nil {
		t.Fatal(err)
	}
	day := filepath.Join(libRoot, "2020", "01", "02")
	if err := os.MkdirAll(day, 0o755); err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(day, "meta.jpg")
	mt := time.Date(2020, 1, 2, 3, 4, 5, 0, time.UTC)
	if err := writeJPEGGray(p, 0xC3); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(p, mt, mt); err != nil {
		t.Fatal(err)
	}

	cmd := testImportCommand()
	cmd.SetArgs([]string{"--dir", day})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}

	db, err := store.Open(libRoot)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	wrong := int64(999)
	_, err = db.Exec(`UPDATE assets SET capture_time_unix = ? WHERE rel_path = ?`, wrong, "2020/01/02/meta.jpg")
	if err != nil {
		t.Fatal(err)
	}

	cmd2 := testImportCommand()
	cmd2.SetArgs([]string{"--dir", day})
	var buf bytes.Buffer
	cmd2.SetOut(&buf)
	if err := cmd2.Execute(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "Updated: 1") {
		t.Fatalf("out:\n%s", buf.String())
	}

	var cap int64
	if err := db.QueryRow(`SELECT capture_time_unix FROM assets WHERE rel_path = ?`, "2020/01/02/meta.jpg").Scan(&cap); err != nil {
		t.Fatal(err)
	}
	if cap != mt.Unix() {
		t.Fatalf("capture_time_unix: got %d want %d", cap, mt.Unix())
	}
}

func TestRunImport_rejectsSymlinkDirOutsideLibrary(t *testing.T) {
	libRoot := filepath.Join(t.TempDir(), "lib")
	ext := filepath.Join(t.TempDir(), "outside")
	if err := os.MkdirAll(libRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(ext, 0o755); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(libRoot, "escape")
	if err := os.Symlink(ext, link); err != nil {
		t.Skip("symlink not supported:", err)
	}
	t.Setenv(config.EnvLibraryRoot, libRoot)
	if err := config.EnsureLibraryLayout(libRoot); err != nil {
		t.Fatal(err)
	}

	cmd := testImportCommand()
	cmd.SetArgs([]string{"--dir", link})
	if err := cmd.Execute(); err == nil {
		t.Fatal("want error when import dir resolves outside library")
	} else if !strings.Contains(err.Error(), "must be under library root") {
		t.Fatalf("err: %v", err)
	}
}
