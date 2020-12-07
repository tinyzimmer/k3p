package types

// PackageMeta represents metadata included with a package.
type PackageMeta struct {
	// The version of this manifest, only v1 currently
	MetaVersion string `json:"apiVersion,omitempty"`
	// The name of the package
	Name string `json:"name,omitempty"`
	// The version of the package
	Version string `json:"version,omitempty"`
	// The K3s version inside the package
	K3sVersion string `json:"k3sVersion,omitempty"`
	// The architecture the package was built for
	Arch string `json:"arch,omitempty"`
	// A listing of the contents of the package
	Manifest Manifest `json:"manifest,omitempty"`
}

// GetName returns the name of the package.
func (p *PackageMeta) GetName() string { return p.Name }

// GetVersion returns the version of the package.
func (p *PackageMeta) GetVersion() string { return p.Version }

// GetK3sVersion returns the K3s version for the package.
func (p *PackageMeta) GetK3sVersion() string { return p.K3sVersion }

// GetArch returns the CPU architecture fo rthe package.
func (p *PackageMeta) GetArch() string { return p.Arch }

// GetManifest returns the manifest of the package.
func (p *PackageMeta) GetManifest() Manifest { return p.Manifest }

// NewEmptyMeta returns a new empty PackageMeta instance.
func NewEmptyMeta() *PackageMeta {
	return &PackageMeta{Manifest: NewEmptyManifest()}
}

// Manifest contains the listings of all the files in the package.
type Manifest struct {
	// Binaries inside the package
	Bins []string `json:"bins,omitempty"`
	// Scripts inside the package
	Scripts []string `json:"scripts,omitempty"`
	// Images inside the package
	Images []string `json:"images,omitempty"`
	// Kubernetes manifests inside the package
	K8sManifests []string `json:"k8sManifests,omitempty"`
	// The End User License Agreement for the package, or an empty string if there is none
	EULA string `json:"eula,omitempty"`
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
