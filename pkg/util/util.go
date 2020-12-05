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

// SyncManifestToNode is a convenience method for extracting the contents of a package manifest
// to a k3s node.
func SyncManifestToNode(target types.Node, manifest *types.PackageManifest) error {
	log.Info("Installing binaries to remote machine at", types.K3sBinDir)
	for _, bin := range manifest.Bins {
		if err := target.WriteFile(bin.Body, path.Join(types.K3sBinDir, bin.Name), "0755", bin.Size); err != nil {
			return err
		}
	}

	log.Info("Installing scripts to remote machine at", types.K3sScriptsDir)
	for _, script := range manifest.Scripts {
		if err := target.WriteFile(script.Body, path.Join(types.K3sScriptsDir, script.Name), "0755", script.Size); err != nil {
			return err
		}
	}

	log.Info("Installing images to remote machine at", types.K3sImagesDir)
	for _, imgs := range manifest.Images {
		if err := target.WriteFile(imgs.Body, path.Join(types.K3sImagesDir, imgs.Name), "0644", imgs.Size); err != nil {
			return err
		}
	}

	log.Info("Installing manifests to remote machine at", types.K3sManifestsDir)
	for _, mani := range manifest.Manifests {
		if err := target.WriteFile(mani.Body, path.Join(types.K3sManifestsDir, mani.Name), "0644", mani.Size); err != nil {
			return err
		}
	}
	return nil
}
