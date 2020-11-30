package types

// Builder is an interface for building application bundles to be distributed to systems.
type Builder interface {
	Build(*BuildOptions) error
}

// BuildOptions is a struct containing options to pass to the build operation.
type BuildOptions struct {
	K3sVersion  string
	Arch        string
	EULAFile    string
	ImageFile   string
	ManifestDir string
	HelmArgs    string
	Excludes    []string
	Output      string
}
