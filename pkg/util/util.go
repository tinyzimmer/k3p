package util

import (
	"crypto/sha256"
	"fmt"
	"io"
	"io/ioutil"
	"os"
)

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
