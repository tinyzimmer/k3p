package node

import (
	"io"
	"io/ioutil"
	"os"
	"path"
	"strconv"
	"strings"

	"github.com/tinyzimmer/k3p/pkg/types"
)

// Mock returns a mock node rooting all files from a temp directory.
func Mock() types.Node {
	tmpDir, err := ioutil.TempDir("", "")
	if err != nil {
		panic(err) // Mock would not be used in normal execution so this is fine
	}
	return &mockNode{root: tmpDir}
}

type mockNode struct{ root string }

func (m *mockNode) rootedDir(f string) string {
	return path.Join(m.root, strings.TrimPrefix(f, "/"))
}

func (m *mockNode) Close() error { return os.RemoveAll(m.root) }

func (m *mockNode) Execute(opts *types.ExecuteOptions) error { return nil }

func (m *mockNode) GetFile(f string) (io.ReadCloser, error) {
	return os.Open(m.rootedDir(f))
}

func (m *mockNode) WriteFile(rdr io.ReadCloser, dest, mode string, size int64) error {
	defer rdr.Close()
	if err := m.MkdirAll(path.Dir(dest)); err != nil {
		return err
	}
	u, err := strconv.ParseUint(mode, 0, 16)
	if err != nil {
		return err
	}
	f, err := os.OpenFile(m.rootedDir(dest), os.O_RDWR|os.O_CREATE, os.FileMode(u))
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(f, rdr)
	return err
}

func (m *mockNode) MkdirAll(path string) error { return os.MkdirAll(m.rootedDir(path), 0755) }
