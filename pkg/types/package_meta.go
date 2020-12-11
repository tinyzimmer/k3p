package types

import (
	"fmt"
	"reflect"
)

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
	Manifest *Manifest `json:"manifest,omitempty"`
	// A configuration containing installation variables
	PackageConfig *PackageConfig `json:"config,omitempty"`
}

// DeepCopy creates a copy of this PackageMeta instance.
func (p *PackageMeta) DeepCopy() *PackageMeta {
	meta := &PackageMeta{
		MetaVersion: p.MetaVersion,
		Name:        p.Name,
		Version:     p.Version,
		K3sVersion:  p.K3sVersion,
		Arch:        p.Arch,
	}
	if p.Manifest != nil {
		meta.Manifest = p.Manifest.DeepCopy()
	}
	if p.PackageConfig != nil {
		meta.PackageConfig = p.PackageConfig.DeepCopy()
	}
	return meta
}

// Sanitize will iterate the PackageConfig and convert any `map[interface{}]interface{}`
// to `map[string]interface{}`. This is required for serializing meta until I find a better
// way to deal with helm values.
func (p *PackageMeta) Sanitize() {
	if p.PackageConfig == nil {
		return
	}
	newHelmValues := make(map[string]interface{})
	for key, value := range p.PackageConfig.HelmValues {
		newHelmValues[key] = sanitizeValue(value)
	}
	p.PackageConfig.HelmValues = newHelmValues
}

func sanitizeValue(val interface{}) interface{} {
	switch reflect.TypeOf(val).Kind() {
	case reflect.Map:
		if m, ok := val.(map[interface{}]interface{}); ok {
			newMap := make(map[string]interface{})
			for k, v := range m {
				kStr := fmt.Sprintf("%v", k)
				newMap[kStr] = sanitizeValue(v)
			}
			return newMap
		}
		if m, ok := val.(map[string]interface{}); ok {
			// if the keys are already strings, we still need to descend
			newMap := make(map[string]interface{})
			for k, v := range m {
				newMap[k] = sanitizeValue(v)
			}
			return newMap
		}
		// otherwise just return the regular map, but this may not catch
		// all cases yet
		return val
	default:
		return val
	}
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
func (p *PackageMeta) GetManifest() *Manifest { return p.Manifest }

// GetPackageConfig returns the package config if of the package or nil if there is none.
func (p *PackageMeta) GetPackageConfig() *PackageConfig { return p.PackageConfig }

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
	// Static assets
	Static []string `json:"static,omitempty"`
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
		EULA:         m.EULA,
	}
	copy(out.Bins, m.Bins)
	copy(out.Scripts, m.Scripts)
	copy(out.Images, m.Images)
	copy(out.K8sManifests, m.K8sManifests)
	copy(out.Static, m.Static)
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
	}
}
