package types

// PackageMeta represents metadata included with a package.
type PackageMeta struct {
	MetaVersion string   `json:"apiVersion,omitempty"`
	Name        string   `json:"name,omitempty"`
	Version     string   `json:"version,omitempty"`
	K3sVersion  string   `json:"k3sVersion,omitempty"`
	Arch        string   `json:"arch,omitempty"`
	Manifest    Manifest `json:"manifest,omitempty"`
}

// NewEmptyMeta returns a new empty PackageMeta instance.
func NewEmptyMeta() *PackageMeta {
	return &PackageMeta{Manifest: NewEmptyManifest()}
}

// Manifest contains the listings of all the files in the package.
type Manifest struct {
	Bins         []string `json:"bins,omitempty"`
	Scripts      []string `json:"scripts,omitempty"`
	Images       []string `json:"images,omitempty"`
	K8sManifests []string `json:"k8sManifests,omitempty"`
	EULA         string   `json:"eula,omitempty"`
}

// HasEULA returns true if the manifest contains an end user license agreement.
func (m *Manifest) HasEULA() bool { return m.EULA != "" }

// NewEmptyManifest initializes a manifest with empty slices.
func NewEmptyManifest() Manifest {
	return Manifest{
		Bins:         make([]string, 0),
		Scripts:      make([]string, 0),
		Images:       make([]string, 0),
		K8sManifests: make([]string, 0),
	}
}
