package node

import (
	"io"
	"io/ioutil"
	"os"
	"path"
	"strconv"
	"strings"

	"github.com/tinyzimmer/k3p/pkg/log"
	"github.com/tinyzimmer/k3p/pkg/types"
)

// SyncManifestToNode is a convenience method for extracting the contents of a package manifest
// to a k3s node.
func SyncManifestToNode(system types.Node, manifest *types.PackageManifest) error {
	log.Info("Installing binaries to remote machine at", types.K3sBinDir)
	for _, bin := range manifest.Bins {
		if err := system.WriteFile(bin.Body, path.Join(types.K3sBinDir, bin.Name), "0755", bin.Size); err != nil {
			return err
		}
	}

	log.Info("Installing scripts to remote machine at", types.K3sScriptsDir)
	for _, script := range manifest.Scripts {
		if err := system.WriteFile(script.Body, path.Join(types.K3sScriptsDir, script.Name), "0755", script.Size); err != nil {
			return err
		}
	}

	log.Info("Installing images to remote machine at", types.K3sImagesDir)
	for _, imgs := range manifest.Images {
		if err := system.WriteFile(imgs.Body, path.Join(types.K3sImagesDir, imgs.Name), "0644", imgs.Size); err != nil {
			return err
		}
	}

	log.Info("Installing manifests to remote machine at", types.K3sManifestsDir)
	for _, mani := range manifest.Manifests {
		if err := system.WriteFile(mani.Body, path.Join(types.K3sManifestsDir, mani.Name), "0644", mani.Size); err != nil {
			return err
		}
	}
	return nil
}

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

func (m *mockNode) Execute(cmd, logPrefix string) error { return nil }

func (m *mockNode) GetFile(f string) (io.ReadCloser, error) {
	return os.Open(m.rootedDir(f))
}

func (m *mockNode) WriteFile(rdr io.ReadCloser, dest, mode string, size int64) error {
	log.Debugf("Writing file to local system at %q with mode %q\n", dest, mode)
	defer rdr.Close()
	if err := m.MkdirAll(dest); err != nil {
		return err
	}
	u, err := strconv.ParseUint("0755", 0, 16)
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
