package cli

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/spf13/cobra"
)

// Execute parses CLI args for subcommands (e.g. scan). The desktop UI is launched from [main] when
// invoked with no arguments — this package stays free of Fyne imports.
func Execute() error {
	rootCmd := &cobra.Command{
		Use:           "phototool",
		Short:         "Photo Tool — local photo library",
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	scanCmd := &cobra.Command{
		Use:   "scan",
		Short: "Scan a directory into the library",
		Long: `Discover supported images under --dir with the same extensions and ingest pipeline as the GUI.

Only files directly in --dir are scanned unless --recursive is set. Use --dry-run to classify outcomes
(added / skipped duplicate / updated / failed) without copying files or writing the database.

Exits with status 1 when any files fail during the run (after printing the summary).`,
		RunE: RunScan,
	}
	scanCmd.Flags().String("dir", "", "directory to scan")
	_ = scanCmd.MarkFlagRequired("dir")
	scanCmd.Flags().Bool("recursive", false, "include subdirectories")
	scanCmd.Flags().Bool("dry-run", false, "preview only (no copy or database writes)")

	importCmd := &cobra.Command{
		Use:   "import",
		Short: "Register files already in the library (no copy)",
		Long: `Sync the database with image files that already live under the library root (register-in-place).

Unlike scan, import does not copy from external folders: --dir must be inside the configured library tree
(after resolving symlinks, so a symlink cannot bypass that rule). You may pass the library root itself
as --dir (use --recursive to include nested canonical day folders).
Use this after manual file operations so assets rows, hashes, and capture time stay aligned.

Use --dry-run to classify outcomes (added / skipped duplicate / updated / failed) without writing the database.

Exits with status 1 when any files fail during the run (after printing the summary).`,
		RunE: RunImport,
	}
	importCmd.Flags().String("dir", "", "directory under the library to import")
	_ = importCmd.MarkFlagRequired("dir")
	importCmd.Flags().Bool("recursive", false, "include subdirectories")
	importCmd.Flags().Bool("dry-run", false, "preview only (no database writes)")

	rootCmd.AddCommand(scanCmd)
	rootCmd.AddCommand(importCmd)

	return rootCmd.Execute()
}

// MainExit runs [Execute] and maps errors to exit codes for main().
func MainExit() {
	if err := Execute(); err != nil {
		slog.Error("phototool", "err", err)
		_, _ = fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}
