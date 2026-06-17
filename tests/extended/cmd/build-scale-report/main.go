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
		fmt.Fprintln(os.Stderr, "usage: build-scale-report -out RUN_DIR")
		os.Exit(2)
	}
	rep, err := extended.WriteScaleReport(*out)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println(rep.MachineLine)
}
