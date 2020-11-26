package types

// ImageParser is an interface for extracting a list of images from a directory
// containing kubernetes manifests.
type ImageParser interface {
	Parse() (images []string, err error)
}

// BaseImageParser represents the base elements for a parser interface.
type BaseImageParser struct {
	ParseDir    string
	ExcludeDirs []string
}

// SetParseDir will overwrite the currently configured parse directory. Could
// be useful for instances where a parser uses pre-processing to a temporary
// directory before invoking inherited methods.
func (b *BaseImageParser) SetParseDir(dir string) { b.ParseDir = dir }

// GetParseDir returns the directory to be parsed for container images.
func (b *BaseImageParser) GetParseDir() string { return b.ParseDir }

// IsExcluded returns true if the given directory should be excluded from parsing.
func (b *BaseImageParser) IsExcluded(dirName string) bool {
	for _, ex := range b.ExcludeDirs {
		if ex == dirName {
			return true
		}
	}
	return false
}
