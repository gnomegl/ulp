package main

import (
	"os"

	"github.com/gnomegl/ulp/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
