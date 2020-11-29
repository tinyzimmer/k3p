package types

import (
	"path"
	"strings"

	"k8s.io/apimachinery/pkg/runtime"
)

// ManifestParser is an interface for extracting a list of images from a directory
// containing kubernetes manifests.
type ManifestParser interface {
	ParseImages() ([]string, error)
	ParseManifests() ([]*Artifact, error)
}

// BaseManifestParser represents the base elements for a parser interface.
type BaseManifestParser struct {
	ParseDir     string
	ExcludeDirs  []string
	HelmArgs     string
	Deserializer runtime.Decoder
}

// GetParseDir returns the directory to be parsed for container images.
func (b *BaseManifestParser) GetParseDir() string { return b.ParseDir }

// GetHelmArgs returns the helm args to use when templating and packaging
// charts
func (b *BaseManifestParser) GetHelmArgs() string { return b.HelmArgs }

// StripParseDir is a convenience method for stripping the parse directory from the beginning
// of a path.
func (b *BaseManifestParser) StripParseDir(s string) string {
	return strings.Replace(s, b.ParseDir+"/", "", 1)
}

// IsExcluded returns true if the given directory should be excluded from parsing.
func (b *BaseManifestParser) IsExcluded(dirName string) bool {
	for _, ex := range b.ExcludeDirs {
		if ex == path.Base(dirName) {
			return true
		}
	}
	return false
}

// Decode will decode the given bytes into a kubernetes runtime object.
func (b *BaseManifestParser) Decode(data []byte) (runtime.Object, error) {
	obj, _, err := b.Deserializer.Decode(data, nil, nil)
	return obj, err
}
