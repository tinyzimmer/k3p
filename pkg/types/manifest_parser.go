package types

// ManifestParser is an interface for extracting a list of images/manifests from a directory
type ManifestParser interface {
	ParseImages() ([]string, error)
	ParseManifests() ([]*Artifact, error)
}
