package types

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"io"
	"io/ioutil"
)

// Package is an interface to be implemented for use by a package bundler/extracter.
// Different versions of how packages are built can implement this interface.
type Package interface {
	// Put should store the provided artifact inside the archive. The interface is responsible
	// for appending the details of the artifact to the metadata.
	Put(*Artifact) error
	// PutMeta should merge the provided meta with any tracked internally by the interface.
	PutMeta(meta *PackageMeta) error
	// Read should populate the given artifact with the contents inside the archive.
	Get(*Artifact) error
	// GetMeta should return the metadata associated with the package. This will contain information
	// on the full contents of the package.
	GetMeta() *PackageMeta
	// ArchiveTo should tar the contents of the archive (with any required meta) to the given
	// path.
	ArchiveTo(path string) error
	// Reader returns an io.Reader containing the tar contents of the archive.
	Reader() io.ReadCloser
	// Size returns the archived size of the package.
	Size() (int64, error)
	// Close should perform any necessary cleanup.
	Close() error
}

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
