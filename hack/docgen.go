package main

import (
	"os"

	"github.com/spf13/cobra/doc"

	"github.com/tinyzimmer/k3p/pkg/cmd"
	"github.com/tinyzimmer/k3p/pkg/log"
)

func main() {
	if err := os.MkdirAll("doc", 0755); err != nil {
		log.Fatal(err)
	}
	if err := doc.GenMarkdownTree(cmd.GetRootCommand(), "doc"); err != nil {
		log.Fatal(err)
	}
}
