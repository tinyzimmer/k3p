package util

import (
	"crypto/sha256"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"os"
	"path"
	"runtime"
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
func SyncPackageToNode(target types.Node, pkg types.Package) error {
	meta := pkg.GetMeta()

	log.Info("Installing binaries to machine at", types.K3sBinDir)
	for _, bin := range meta.Manifest.Bins {
		if err := writePkgFileToNode(target, pkg, types.ArtifactBin, path.Base(bin), types.K3sBinDir, "0755"); err != nil {
			return err
		}
	}

	log.Info("Installing scripts to machine at", types.K3sScriptsDir)
	for _, script := range meta.Manifest.Scripts {
		if err := writePkgFileToNode(target, pkg, types.ArtifactScript, path.Base(script), types.K3sBinDir, "0755"); err != nil {
			return err
		}
	}

	log.Info("Installing images to machine at", types.K3sImagesDir)
	for _, imgs := range meta.Manifest.Images {
		if err := writePkgFileToNode(target, pkg, types.ArtifactImages, path.Base(imgs), types.K3sImagesDir, "0644"); err != nil {
			return err
		}
	}

	log.Info("Installing manifests to machine at", types.K3sManifestsDir)
	for _, mani := range meta.Manifest.K8sManifests {
		// strip the prefix if it matches the base of the k3s dir
		mani = strings.TrimPrefix(mani, path.Base(types.K3sManifestsDir)+"/")
		if err := writePkgFileToNode(target, pkg, types.ArtifactManifest, mani, types.K3sManifestsDir, "0644"); err != nil {
			return err
		}
	}
	return nil
}

func writePkgFileToNode(target types.Node, pkg types.Package, t types.ArtifactType, name, destDir, mode string) error {
	artifact := &types.Artifact{Type: t, Name: name}
	if err := pkg.Get(artifact); err != nil {
		return err
	}
	// artifact.Name will reflect the correct basename to use after a Get (in case it contains an internal leading directory)
	// this needs to be fixed
	return target.WriteFile(artifact.Body, path.Join(destDir, artifact.Name), mode, artifact.Size)
}

// ArtifactFromReader will create a new types.Artifact object with the name and type provided. Its contents
// are populated by the reader. The purpose of this method is an easy way to determine the size of the
// object, while taking care that it may be too large to place in memory. The reader is dumped to disk,
// and then its size is queried from the filesystem. The Body of the returned artifact points to the file
// on the system.
//
// A finalizer is placed on the resulting artifact to ensure the temporary directory is cleaned up once the
// artifact leaves runtime scope.
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

	// Build out the artifact
	artifact := &types.Artifact{
		Type: t,
		Name: name,
		Size: stat.Size(),
		Body: f,
	}

	// Set a finalizer on the artifact to remove the tempdir
	runtime.SetFinalizer(artifact, func(_ *types.Artifact) { os.RemoveAll(tmpDir) })

	return artifact, nil
}
