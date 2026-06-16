package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"photo-tool/tests/extended"
)

func main() {
	out := flag.String("out", ".", "output directory for matrix.json and matrix.md")
	flag.Parse()

	gitShort := "unknown"
	if b, err := exec.Command("git", "rev-parse", "--short", "HEAD").Output(); err == nil {
		gitShort = strings.TrimSpace(string(b))
	}

	m, err := extended.GenerateMatrix(gitShort)
	if err != nil {
		fmt.Fprintf(os.Stderr, "generate matrix: %v\n", err)
		os.Exit(1)
	}
	if missing := extended.AllStoriesPresent(m); len(missing) > 0 {
		fmt.Fprintf(os.Stderr, "warning: missing story functional rows: %v\n", missing)
	}
	outAbs, err := filepath.Abs(*out)
	if err != nil {
		fmt.Fprintf(os.Stderr, "out path: %v\n", err)
		os.Exit(1)
	}
	if err := extended.WriteMatrix(outAbs, m); err != nil {
		fmt.Fprintf(os.Stderr, "write matrix: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(outAbs)
}
