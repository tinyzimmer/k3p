package node

import (
	"io"

	"github.com/tinyzimmer/k3p/pkg/types"
)

// Docker initializes a new node using a local container for the instance.
func Docker(opts *types.DockerNodeOptions) types.Node {
	return &docker{}
}

type docker struct{}

func (d *docker) MkdirAll(dir string) error                  { return nil }
func (d *docker) GetFile(path string) (io.ReadCloser, error) { return nil, nil }
func (d *docker) WriteFile(rdr io.ReadCloser, destination string, mode string, size int64) error {
	return nil
}
func (d *docker) Execute(*types.ExecuteOptions) error { return nil }
func (d *docker) Close() error                        { return nil }
