package main

import (
	"flag"
	"fmt"
	"os"

	"photo-tool/tests/extended"
)

func main() {
	out := flag.String("out", "", "extended run directory")
	dir := flag.String("dir", "ui-real", "capture subdir (ui-real, ui-real-scale, ui-real-edge, or all)")
	flag.Parse()
	if *out == "" {
		fmt.Fprintln(os.Stderr, "usage: validate-ux-bundle -out RUN_DIR [-dir ui-real|all]")
		os.Exit(2)
	}
	if *dir == "all" {
		if err := extended.ValidateAllRealCaptureDirs(*out); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		if err := extended.ValidateCaptureDistinct(*out); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		fmt.Println("UX_JUDGE_INPUTS=ok")
		return
	}
	if err := extended.ValidateUXCaptureSubdir(*out, *dir); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if *dir == "ui-real" {
		if err := extended.ValidateRealAppCapture(*out); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		if err := extended.ValidateCaptureDistinct(*out); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}
	fmt.Println("UX_JUDGE_INPUTS=ok")
}
