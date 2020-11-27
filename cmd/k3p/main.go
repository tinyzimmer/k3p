package main

import (
	"os"

	"github.com/tinyzimmer/k3p/pkg/cmd"
	"github.com/tinyzimmer/k3p/pkg/log"
)

func main() {
	if err := cmd.GetRootCommand().Execute(); err != nil {
		log.Error(err)
		os.Exit(1)
	}
}
