package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"photo-tool/internal/config"
	"photo-tool/internal/fixture"
	"photo-tool/internal/store"
)

func main() {
	out := flag.String("out", "", "library root directory")
	tier := flag.String("tier", "S1", "scale tier S0–S8")
	srcDir := flag.String("src-dir", "", "directory for upload seed JPEGs (default: out/.fixture-src)")
	fsOnly := flag.Int("fs-tree", 0, "if >0, write N JPEGs for CLI scan only (no DB); ignores tier asset count")
	flag.Parse()
	if *out == "" {
		fmt.Fprintln(os.Stderr, "usage: seed-library -out LIBRARY_ROOT [-tier S4] [-src-dir DIR]")
		os.Exit(2)
	}
	root := filepath.Clean(*out)
	if *fsOnly > 0 {
		if err := fixture.SeedFilesystemTree(root, *fsOnly); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		fmt.Printf("FIXTURE_FS_TREE=ok n=%d\n", *fsOnly)
		return
	}
	if err := config.EnsureLibraryLayout(root); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	sd := *srcDir
	if sd == "" {
		sd = filepath.Join(root, ".fixture-src")
	}
	db, err := store.Open(root)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer func() { _ = db.Close() }()
	m, _, err := fixture.SeedLibrary(db, fixture.SeedOptions{
		Tier:   fixture.ParseTier(*tier),
		Root:   root,
		SrcDir: sd,
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Printf("FIXTURE_SEED=ok tier=%s assets=%d albums=%d tags=%d rejected=%d ms=%d\n",
		m.Tier, m.Assets, m.Albums, m.Tags, m.Rejected, m.GenerationMS)
}
