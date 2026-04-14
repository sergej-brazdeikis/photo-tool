package cli

import (
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"photo-tool/internal/domain"
	"photo-tool/internal/ingest"
)

// RunImport wires the import subcommand: register files already under the library tree (no copy).
func RunImport(cmd *cobra.Command, _ []string) error {
	dir, err := cmd.Flags().GetString("dir")
	if err != nil {
		return err
	}
	recursive, err := cmd.Flags().GetBool("recursive")
	if err != nil {
		return err
	}
	dryRun, err := cmd.Flags().GetBool("dry-run")
	if err != nil {
		return err
	}

	importDir, err := filepath.Abs(filepath.Clean(dir))
	if err != nil {
		return fmt.Errorf("resolve import dir: %w", err)
	}
	st, err := os.Stat(importDir)
	if err != nil {
		return fmt.Errorf("stat import dir: %w", err)
	}
	if !st.IsDir() {
		return fmt.Errorf("not a directory: %s", importDir)
	}

	db, libRoot, cleanup, err := openLibrary()
	if err != nil {
		return err
	}
	defer cleanup()

	// Containment uses EvalSymlinks so a directory symlink cannot point outside the library while still
	// looking nested. RegisterInPlacePath uses the configured library root (unresolved) and walk paths
	// from importDir so filepath.Rel stays in the same name space as the user's paths and the DB.
	libResolved, err := filepath.EvalSymlinks(libRoot)
	if err != nil {
		return fmt.Errorf("resolve library root: %w", err)
	}
	importResolved, err := filepath.EvalSymlinks(importDir)
	if err != nil {
		return fmt.Errorf("resolve import dir: %w", err)
	}
	if !scanDirInsideLibrary(libResolved, importResolved) {
		return fmt.Errorf("import dir must be under library root (%s): %s", libRoot, importDir)
	}

	var sum domain.OperationSummary
	var nSeen int
	logEvery := 1000

	walkFile := func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			sum.Failed++
			slog.Error("import: walk", "path", path, "err", walkErr)
			return nil
		}
		if d.IsDir() {
			return nil
		}
		if !ingest.IsSupportedScanExt(filepath.Ext(path)) {
			return nil
		}
		ingest.RegisterInPlacePath(db, libRoot, path, &sum, dryRun)
		nSeen++
		if logEvery > 0 && nSeen%logEvery == 0 {
			slog.Info("import progress", "files", nSeen, "added", sum.Added, "skipped_duplicate", sum.SkippedDuplicate, "updated", sum.Updated, "failed", sum.Failed)
		}
		return nil
	}

	if recursive {
		if err := filepath.WalkDir(importDir, walkFile); err != nil {
			return err
		}
	} else {
		entries, err := os.ReadDir(importDir)
		if err != nil {
			return fmt.Errorf("read dir: %w", err)
		}
		for _, e := range entries {
			if e.IsDir() {
				continue
			}
			path := filepath.Join(importDir, e.Name())
			_ = walkFile(path, e, nil)
		}
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Added: %d\n", sum.Added)
	fmt.Fprintf(cmd.OutOrStdout(), "Skipped duplicate: %d\n", sum.SkippedDuplicate)
	fmt.Fprintf(cmd.OutOrStdout(), "Updated: %d\n", sum.Updated)
	fmt.Fprintf(cmd.OutOrStdout(), "Failed: %d\n", sum.Failed)
	return errIfOperationFailures("import", sum)
}
