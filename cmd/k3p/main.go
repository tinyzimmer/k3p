package main

import (
	"os"

	"github.com/tinyzimmer/k3p/pkg/cmd"
)

func main() {
	if err := cmd.GetRootCommand().Execute(); err != nil {
		os.Exit(1)
	}
}
