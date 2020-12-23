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
	// The format with which images were bundles in the archive.
	ImageBundleFormat ImageBundleFormat `json:"imageBundleFormat,omitempty"`
	// A listing of the contents of the package
	Manifest *Manifest `json:"manifest,omitempty"`
	// A configuration containing installation variables
	PackageConfig *PackageConfig `json:"config,omitempty"`
	// The raw, untemplated package config
	PackageConfigRaw []byte `json:"configRaw,omitempty"`
}

// DeepCopy creates a copy of this PackageMeta instance.
// TODO: DeepCopy functions need to be generated
func (p *PackageMeta) DeepCopy() *PackageMeta {
	meta := &PackageMeta{
		MetaVersion:       p.MetaVersion,
		Name:              p.Name,
		Version:           p.Version,
		K3sVersion:        p.K3sVersion,
		Arch:              p.Arch,
		ImageBundleFormat: p.ImageBundleFormat,
		PackageConfigRaw:  make([]byte, len(p.PackageConfigRaw)),
	}
	copy(meta.PackageConfigRaw, p.PackageConfigRaw)
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
// way to deal with helm values. For convenience, the pointer to the PackageMeta is returned.
func (p *PackageMeta) Sanitize() *PackageMeta {
	if p.PackageConfig == nil {
		return p
	}
	newHelmValues := make(map[string]interface{})
	for key, value := range p.PackageConfig.HelmValues {
		newHelmValues[key] = sanitizeValue(value)
	}
	p.PackageConfig.HelmValues = newHelmValues
	return p
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

// GetRegistryImageName returns the name that would have been used for a container image
// containing the registry contents.
// TODO: Needing to keep this logic here and BuildRegistryOptions is not a good design probably.
func (p *PackageMeta) GetRegistryImageName() string {
	return fmt.Sprintf("%s-private-registry-data:%s", p.Name, p.Version)
}

// NewEmptyMeta returns a new empty PackageMeta instance.
func NewEmptyMeta() *PackageMeta {
	return &PackageMeta{Manifest: NewEmptyManifest()}
}
