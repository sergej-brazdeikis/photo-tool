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
		fmt.Fprintln(os.Stderr, "usage: validate-ux-bundle -out RUN_DIR")
		os.Exit(2)
	}
	if err := extended.ValidateRealAppCapture(*out); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := extended.ValidateUXJudgeInputs(*out); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := extended.ValidateCaptureDistinct(*out); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println("UX_JUDGE_INPUTS=ok")
}
