package util

import (
	"crypto/sha256"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"os"
	"path"
	"time"

	"github.com/tinyzimmer/k3p/pkg/log"
	"github.com/tinyzimmer/k3p/pkg/types"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

// TempDir cast as a var to be overridden by CLI flags.
var TempDir = os.TempDir()

// GetTempDir is a utility function for retrieving a new temporary directory within either
// the system default, or user-configured path.
func GetTempDir() (string, error) { return ioutil.TempDir(TempDir, "") }

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
		if err := writePkgFileToNode(target, pkg, types.ArtifactBin, bin, types.K3sBinDir, "0755"); err != nil {
			return err
		}
	}

	log.Info("Installing scripts to machine at", types.K3sScriptsDir)
	for _, script := range meta.Manifest.Scripts {
		if err := writePkgFileToNode(target, pkg, types.ArtifactScript, script, types.K3sBinDir, "0755"); err != nil {
			return err
		}
	}

	log.Info("Installing images to machine at", types.K3sImagesDir)
	for _, imgs := range meta.Manifest.Images {
		if err := writePkgFileToNode(target, pkg, types.ArtifactImages, imgs, types.K3sImagesDir, "0644"); err != nil {
			return err
		}
	}

	log.Info("Installing manifests to machine at", types.K3sManifestsDir)
	for _, mani := range meta.Manifest.K8sManifests {
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
