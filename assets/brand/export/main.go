package main

import (
	"os"

	"github.com/0xamirreza/portpilot/assets/brand"
)

func main() {
	if len(os.Args) != 2 {
		os.Exit(2)
	}
	if err := brand.WriteIcons(os.Args[1]); err != nil {
		panic(err)
	}
}
