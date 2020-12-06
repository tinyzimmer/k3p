package types

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"io"
	"io/ioutil"
)

// Artifact represents an object to be placed or extracted from a bundle.
type Artifact struct {
	// The type of the artifact
	Type ArtifactType
	// The name of the artifact (this can include subdirectories)
	Name string
	// The size of the artifact, only populated on retrieval, or if made
	// with the ArtifactFromReader method in the utils package.
	Size int64
	// The contents of the artifact
	Body io.ReadCloser
}

// Verify will verify the contents of this artifact against the given sha256sum.
// Note that this method will read the entire contents of the artifact into memory.
func (a *Artifact) Verify(sha256sum string) error {
	var buf bytes.Buffer
	defer func() { a.Body = ioutil.NopCloser(&buf) }()
	tee := io.TeeReader(a.Body, &buf)
	defer a.Body.Close() // will pop off the stack first
	h := sha256.New()
	if _, err := io.Copy(h, tee); err != nil {
		return err
	}
	localSum := fmt.Sprintf("%x", h.Sum(nil))
	if localSum != sha256sum {
		return fmt.Errorf("sha256 mismatch in %s %s", a.Type, a.Name)
	}
	return nil
}
