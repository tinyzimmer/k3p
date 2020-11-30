package archive

import (
	v1 "github.com/tinyzimmer/k3p/pkg/archive/v1"
	"github.com/tinyzimmer/k3p/pkg/types"
)

// GetReadWriter returns a bundle read/writer for the given working directory.
func GetReadWriter(dir string) types.BundleReadWriter { return v1.New(dir) }
