package types

import (
	"io"
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
	// Archive should produce an Archive interface that can be used to read from the final package stream.
	// This method should ensure any metadata and finalize the archive. Any changes made to the package after
	// Archive is called will require another call to receive the latest changes.
	Archive() (Archive, error)
	// Close should perform any necessary cleanup on both this interface, and archives created from it.
	Close() error
}

// Archive is an interface to be implemented by packagers/extracers. It contains the final contents
// of the archive and methods for interacting with it.
type Archive interface {
	// Reader should return a simple io.ReadCloser for the archive.
	Reader() io.ReadCloser
	// WriteTo should dump the contents of the archive to the given file.
	WriteTo(path string) error
	// CompressTo should compress the contents of the archive to the given zst file.
	CompressTo(path string) error
	// Size should return the size of the archive.
	Size() int64
}
