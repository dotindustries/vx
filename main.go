package main

import (
	"os"

	"go.dot.industries/vx/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
