package cli

import (
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"photo-tool/internal/domain"
	"photo-tool/internal/ingest"
)

// RunScan wires the scan subcommand. Directory traversal uses filepath.WalkDir so paths are
// processed one at a time (NFR-02: no slice of every path for 10k+ trees).
func RunScan(cmd *cobra.Command, _ []string) error {
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

	scanDir, err := filepath.Abs(filepath.Clean(dir))
	if err != nil {
		return fmt.Errorf("resolve scan dir: %w", err)
	}
	st, err := os.Stat(scanDir)
	if err != nil {
		return fmt.Errorf("stat scan dir: %w", err)
	}
	if !st.IsDir() {
		return fmt.Errorf("not a directory: %s", scanDir)
	}

	db, libRoot, cleanup, err := openLibrary()
	if err != nil {
		return err
	}
	defer cleanup()

	if scanDirInsideLibrary(libRoot, scanDir) {
		slog.Warn("scan dir is under library root; expect mostly skipped_duplicate for canonical files", "dir", scanDir, "library", libRoot)
	}

	var sum domain.OperationSummary
	var nSeen int
	logEvery := 1000

	walkFile := func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			sum.Failed++
			slog.Error("scan: walk", "path", path, "err", walkErr)
			return nil
		}
		if d.IsDir() {
			return nil
		}
		if !ingest.IsSupportedScanExt(filepath.Ext(path)) {
			return nil
		}
		ingest.IngestPath(db, libRoot, path, &sum, dryRun)
		nSeen++
		if logEvery > 0 && nSeen%logEvery == 0 {
			slog.Info("scan progress", "files", nSeen, "added", sum.Added, "skipped_duplicate", sum.SkippedDuplicate, "failed", sum.Failed)
		}
		return nil
	}

	if recursive {
		if err := filepath.WalkDir(scanDir, walkFile); err != nil {
			return err
		}
	} else {
		entries, err := os.ReadDir(scanDir)
		if err != nil {
			return fmt.Errorf("read dir: %w", err)
		}
		for _, e := range entries {
			if e.IsDir() {
				continue
			}
			path := filepath.Join(scanDir, e.Name())
			_ = walkFile(path, e, nil)
		}
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Added: %d\n", sum.Added)
	fmt.Fprintf(cmd.OutOrStdout(), "Skipped duplicate: %d\n", sum.SkippedDuplicate)
	fmt.Fprintf(cmd.OutOrStdout(), "Updated: %d\n", sum.Updated)
	fmt.Fprintf(cmd.OutOrStdout(), "Failed: %d\n", sum.Failed)
	return errIfOperationFailures("scan", sum)
}

func scanDirInsideLibrary(libraryRoot, scanDir string) bool {
	lr := filepath.Clean(libraryRoot)
	sd := filepath.Clean(scanDir)
	rel, err := filepath.Rel(lr, sd)
	if err != nil {
		return false
	}
	return rel == "." || !strings.HasPrefix(rel, "..")
}
