package types

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"

	"github.com/tinyzimmer/k3p/pkg/util"
)

// ArtifactType declares a type of artifact to be included in a bundle.
type ArtifactType string

const (
	// ArtifactBin represents a binary artifact.
	ArtifactBin ArtifactType = "bin"
	// ArtifactImages represents a container image artifact.
	ArtifactImages ArtifactType = "images"
	// ArtifactScript represents a script artifact.
	ArtifactScript ArtifactType = "script"
	// ArtifactManifest represents a kubernetes manifest artifact.
	ArtifactManifest ArtifactType = "manifest"
)

// Artifact represents an object to be placed or extracted from a bundle.
// It includes a helper Verify() method for validating the contents against
// a provided sha256sum.
type Artifact struct {
	Type ArtifactType
	Name string
	Size int64 // only populated on retrieval
	Body io.ReadCloser
}

// Verify will verify the contents of this artifact against the given sha256sum.
func (a *Artifact) Verify(sha256sum string) error {
	var buf bytes.Buffer
	defer func() { a.Body = ioutil.NopCloser(&buf) }()
	tee := io.TeeReader(a.Body, &buf)
	defer a.Body.Close() // will pop off the stack first

	localSum, err := util.CalculateSHA256Sum(tee)
	if err != nil {
		return err
	}
	if localSum != sha256sum {
		return fmt.Errorf("sha256 mismatch in %s %s", a.Type, a.Name)
	}
	return nil
}

// PackageManifest represents the complete contents of a packaged k3s system.
type PackageManifest struct {
	Bins, Scripts, Images, Manifests []*Artifact
}

// NewPackageManifest initializes a manifest with empty slices.
func NewPackageManifest() *PackageManifest {
	return &PackageManifest{
		Bins:      make([]*Artifact, 0),
		Scripts:   make([]*Artifact, 0),
		Images:    make([]*Artifact, 0),
		Manifests: make([]*Artifact, 0),
	}
}

// BundleReadWriter is an interface to be implemented for use by a package bundler/extracter.
// Different versions of how manifests are built can implement this interface.
type BundleReadWriter interface {
	// Put should store the provided artifact inside the bundle.
	Put(*Artifact) error
	// Read should populate the given artifact with the contents inside the bundle.
	Get(*Artifact) error
	// GetManifest should retrieve readable copies of the entire contents of the bundle.
	GetManifest() (*PackageManifest, error)
	// ArchiveTo should tar the contents of the bundle (with any required meta) to the given
	// path.
	ArchiveTo(path string) error
	// Close should cleanup the interface's working directory.
	Close() error
}
