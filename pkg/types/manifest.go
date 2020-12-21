package types

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
	// Static assets
	Static []string `json:"static,omitempty"`
	// Etc assets
	Etc []string `json:"etc,omitempty"`
	// The End User License Agreement for the package, or an empty string if there is none
	EULA string `json:"eula,omitempty"`
}

// DeepCopy returns a copy of this Manifest.
func (m *Manifest) DeepCopy() *Manifest {
	out := &Manifest{
		Bins:         make([]string, len(m.Bins)),
		Scripts:      make([]string, len(m.Scripts)),
		Images:       make([]string, len(m.Images)),
		K8sManifests: make([]string, len(m.K8sManifests)),
		Static:       make([]string, len(m.Static)),
		Etc:          make([]string, len(m.Etc)),
		EULA:         m.EULA,
	}
	copy(out.Bins, m.Bins)
	copy(out.Scripts, m.Scripts)
	copy(out.Images, m.Images)
	copy(out.K8sManifests, m.K8sManifests)
	copy(out.Static, m.Static)
	copy(out.Etc, m.Etc)
	return out
}

// HasEULA returns true if the manifest contains an end user license agreement.
func (m *Manifest) HasEULA() bool { return m.EULA != "" }

// NewEmptyManifest initializes a manifest with empty slices.
func NewEmptyManifest() *Manifest {
	return &Manifest{
		Bins:         make([]string, 0),
		Scripts:      make([]string, 0),
		Images:       make([]string, 0),
		K8sManifests: make([]string, 0),
		Static:       make([]string, 0),
		Etc:          make([]string, 0),
	}
}
