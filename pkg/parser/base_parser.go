package parser

import (
	"path"
	"strings"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	corescheme "k8s.io/client-go/kubernetes/scheme"
)

// BaseManifestParser represents the base elements for a parser interface. It contains
// convenience methods for common directory and file operations. The original intention
// of this here was to be used as a base for different processors (e.g. raw, helm, kustomize, jsonnet, etc.),
// however in working towards a POC it made sense to keep things simple and combine raw and helm into a single
// interface.
type BaseManifestParser struct {
	ParseDir     string
	ExcludeDirs  []string
	HelmArgs     string
	Deserializer runtime.Decoder
}

// NewBaseManifestParser returns a new base parser with the given arguments.
func NewBaseManifestParser(parseDir string, excludeDirs []string, helmArgs string) *BaseManifestParser {
	// create a new scheme
	sch := runtime.NewScheme()

	// currently only supports core APIs, could consider some way of dynamically adding CRD support
	// full list: https://github.com/kubernetes/client-go/blob/master/kubernetes/scheme/register.go
	_ = corescheme.AddToScheme(sch)

	return &BaseManifestParser{
		ParseDir:     parseDir,
		ExcludeDirs:  excludeDirs,
		HelmArgs:     helmArgs,
		Deserializer: serializer.NewCodecFactory(sch).UniversalDeserializer(),
	}
}

// GetParseDir returns the directory to be parsed for container images.
func (b *BaseManifestParser) GetParseDir() string { return b.ParseDir }

// GetHelmArgs returns the helm args to use when templating and packaging
// charts
func (b *BaseManifestParser) GetHelmArgs() string { return b.HelmArgs }

// StripParseDir is a convenience method for stripping the parse directory from the beginning
// of a path.
func (b *BaseManifestParser) StripParseDir(s string) string {
	return strings.Replace(s, b.ParseDir+"/", "", 1)
}

// IsExcluded returns true if the given directory should be excluded from parsing.
func (b *BaseManifestParser) IsExcluded(dirName string) bool {
	for _, ex := range b.ExcludeDirs {
		if ex == path.Base(dirName) {
			return true
		}
	}
	return false
}

// Decode will decode the given bytes into a kubernetes runtime object.
func (b *BaseManifestParser) Decode(data []byte) (runtime.Object, error) {
	obj, _, err := b.Deserializer.Decode(data, nil, nil)
	return obj, err
}
