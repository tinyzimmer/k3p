package types

// ManifestParser is an interface for extracting a list of images/manifests from a directory.
type ManifestParser interface {
	// ParseImages should traverse the configured directories and search for container images
	// to download.
	ParseImages() ([]string, error)
	// ParseManifests should traverse the configured directories and produce artifacts for
	// every kubernetes manifest it finds.
	ParseManifests() ([]*Artifact, error)
}
