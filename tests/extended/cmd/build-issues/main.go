package main

import (
	"flag"
	"fmt"
	"os"

	"photo-tool/tests/extended"
)

func main() {
	out := flag.String("out", "", "extended run directory")
	flag.Parse()
	if *out == "" {
		fmt.Fprintln(os.Stderr, "usage: build-issues -out RUN_DIR")
		os.Exit(2)
	}
	issues, err := extended.BuildIssueQueue(*out)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Printf("issues=%d\n", len(issues))
	if len(issues) == 0 {
		fmt.Println("EXTENDED_ISSUE_QUEUE=empty")
		return
	}
	fmt.Println("EXTENDED_ISSUE_QUEUE=open")
}
