package util

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"os"
	"path"
	"strings"
	"time"

	"github.com/tinyzimmer/k3p/pkg/log"
	"github.com/tinyzimmer/k3p/pkg/types"

	"github.com/docker/docker/pkg/namesgenerator"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

// TempDir cast as a var to be overridden by CLI flags.
var TempDir = os.TempDir()

// GetTempDir is a utility function for retrieving a new temporary directory within either
// the system default, or user-configured path.
func GetTempDir() (string, error) { return ioutil.TempDir(TempDir, "") }

// GetRandomName returns a random name using the docker name generator
func GetRandomName() string { return namesgenerator.GetRandomName(0) }

// CalculateSHA256Sum calculates the sha256sum of the contents of the given reader.
func CalculateSHA256Sum(rdr io.Reader) (string, error) {
	h := sha256.New()
	if _, err := io.Copy(h, rdr); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

// IsK8sObject returns true if the given map contains what appears to be a valid Kubernetes object.
// This function needs to compensate for not having a reliable representation of the full cluster scheme
// once deployed. So for now, it just checks for the existence of the common fields (kind, apiVersion, metadata).
func IsK8sObject(data map[string]interface{}) bool {
	for _, key := range []string{"kind", "apiVersion", "metadata"} {
		if _, ok := data[key]; !ok {
			return false
		}
	}
	return true
}

var letterBytes = []byte("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")

// GenerateToken will generate a token of the given length.
func GenerateToken(length int) string {
	b := make([]byte, length)
	for i := range b {
		b[i] = letterBytes[rand.Int63()%int64(len(letterBytes))]
	}
	return string(b)
}

// SyncPackageToNode is a convenience method for extracting the contents of a package manifest
// to a k3s node.
func SyncPackageToNode(target types.Node, pkg types.Package, vars map[string]string) error {
	meta := pkg.GetMeta()

	if len(meta.Manifest.Bins) > 0 {
		log.Info("Installing binaries to", types.K3sBinDir)
		for _, bin := range meta.Manifest.Bins {
			if err := writePkgFileToNode(target, pkg, types.ArtifactBin, path.Base(bin), types.K3sBinDir, "0755", vars); err != nil {
				return err
			}
		}
	}

	if len(meta.Manifest.Scripts) > 0 {
		log.Info("Installing scripts to", types.K3sScriptsDir)
		for _, script := range meta.Manifest.Scripts {
			if err := writePkgFileToNode(target, pkg, types.ArtifactScript, path.Base(script), types.K3sScriptsDir, "0755", vars); err != nil {
				return err
			}
		}
	}

	if len(meta.Manifest.Images) > 0 {
		log.Info("Installing images to", types.K3sImagesDir)
		for _, imgs := range meta.Manifest.Images {
			if err := writePkgFileToNode(target, pkg, types.ArtifactImages, path.Base(imgs), types.K3sImagesDir, "0644", vars); err != nil {
				return err
			}
		}
	}

	if len(meta.Manifest.K8sManifests) > 0 {
		log.Info("Installing manifests to", types.K3sManifestsDir)
		for _, mani := range meta.Manifest.K8sManifests {
			if err := writePkgFileToNode(target, pkg, types.ArtifactManifest, mani, types.K3sManifestsDir, "0644", vars); err != nil {
				return err
			}
		}
	}

	if len(meta.Manifest.Static) > 0 {
		log.Info("Installing static content to", types.K3sStaticDir)
		for _, static := range meta.Manifest.Static {
			static = strings.TrimPrefix(static, "static/") // ugly hack, should fix to come back without the prefix
			if err := writePkgFileToNode(target, pkg, types.ArtifactStatic, static, types.K3sStaticDir, "0644", vars); err != nil {
				return err
			}
		}
	}

	if len(vars) > 0 {
		out, err := json.MarshalIndent(vars, "", "  ")
		if err != nil {
			return err
		}
		rdr := ioutil.NopCloser(bytes.NewReader(out))
		if err := target.WriteFile(rdr, types.InstalledConfigFile, "0644", int64(len(out))); err != nil {
			return err
		}
	}

	return nil
}

func writePkgFileToNode(target types.Node, pkg types.Package, t types.ArtifactType, name, destDir, mode string, vars map[string]string) error {
	artifact := &types.Artifact{Type: t, Name: name}
	if err := pkg.Get(artifact); err != nil {
		return err
	}
	if t == types.ArtifactManifest && len(vars) > 0 {
		if err := artifact.ApplyVariables(vars); err != nil {
			return err
		}
	}
	return target.WriteFile(artifact.Body, path.Join(destDir, artifact.Name), mode, artifact.Size)
}

type tmpReadCloser struct {
	tmpDir string
	f      *os.File
}

func (r *tmpReadCloser) Read(p []byte) (int, error) { return r.f.Read(p) }

func (r *tmpReadCloser) Close() error {
	if err := r.f.Close(); err != nil {
		return err
	}
	return os.RemoveAll(r.tmpDir)
}

// ArtifactFromReader will create a new types.Artifact object with the name and type provided. Its contents
// are populated by the reader. The purpose of this method is an easy way to determine the size of the
// object, while taking care that it may be too large to place in memory. The reader is dumped to disk,
// and then its size is queried from the filesystem. The Body of the returned artifact points to the file
// on the system.
func ArtifactFromReader(t types.ArtifactType, name string, rdr io.ReadCloser) (*types.Artifact, error) {
	defer rdr.Close()

	// Get a tempdir just for this artifact
	tmpDir, err := GetTempDir()
	if err != nil {
		return nil, err
	}
	tmpFile := path.Join(tmpDir, "file")

	// Copy the reader to the temp file
	f, err := os.Create(tmpFile)
	if err != nil {
		return nil, err
	}

	if _, err := io.Copy(f, rdr); err != nil {
		return nil, err
	}

	// Close the file to ensure the contents are flushed
	if err := f.Close(); err != nil {
		return nil, err
	}

	// Stat the file to get the size
	stat, err := os.Stat(tmpFile)
	if err != nil {
		return nil, err
	}

	// Reopen the file for reading
	f, err = os.Open(tmpFile)
	if err != nil {
		return nil, err
	}

	return &types.Artifact{
		Type: t,
		Name: name,
		Size: stat.Size(),
		Body: &tmpReadCloser{
			tmpDir: tmpDir, f: f,
		},
	}, nil
}
