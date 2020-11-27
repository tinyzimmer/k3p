package types

import "strings"

// ManifestParser is an interface for extracting a list of images from a directory
// containing kubernetes manifests.
type ManifestParser interface {
	ParseImages() ([]string, error)
	ParseManifests() ([]*Artifact, error)
}

// BaseManifestParser represents the base elements for a parser interface.
type BaseManifestParser struct {
	ParseDir    string
	ExcludeDirs []string
}

// SetParseDir will overwrite the currently configured parse directory. Could
// be useful for instances where a parser uses pre-processing to a temporary
// directory before invoking inherited methods.
func (b *BaseManifestParser) SetParseDir(dir string) { b.ParseDir = dir }

// GetParseDir returns the directory to be parsed for container images.
func (b *BaseManifestParser) GetParseDir() string { return b.ParseDir }

// StripParseDir is a convenience method for stripping the parse directory from the beginning
// of a path.
func (b *BaseManifestParser) StripParseDir(s string) string {
	return strings.Replace(s, b.ParseDir+"/", "", 1)
}

// IsExcluded returns true if the given directory should be excluded from parsing.
func (b *BaseManifestParser) IsExcluded(dirName string) bool {
	for _, ex := range b.ExcludeDirs {
		if ex == dirName {
			return true
		}
	}
	return false
}
