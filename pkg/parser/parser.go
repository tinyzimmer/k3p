package parser

import (
	"github.com/tinyzimmer/k3p/pkg/parser/helm"
	"github.com/tinyzimmer/k3p/pkg/parser/kustomize"
	"github.com/tinyzimmer/k3p/pkg/parser/raw"
	"github.com/tinyzimmer/k3p/pkg/types"
)

// Type represents a type of image parser to use when traversing directories
// for container images.
type Type string

const (
	// TypeRaw represents a raw image parser that interacts with regular kubernetes yaml
	TypeRaw Type = "raw"
	// TypeHelm (TODO) represents a helm image parser that will template charts and then likely
	// callback to the raw parser to find images.
	TypeHelm Type = "helm"
	// TypeKustomize (TODO) represents a kustomize image parser that, like the helm parser, will render
	// the raw manifests and then likely call back to the raw parser.
	TypeKustomize Type = "kustomize"
)

const ()

// NewManifestParser returns an interface for parsing container images from the given directory.
// TOOO: Currently only supports a raw manifest parser, with opts for helm/kustomize planned
// in the future.
func NewManifestParser(parseDir string, excludeDirs []string, parserType Type) types.ManifestParser {
	base := &types.BaseManifestParser{
		ParseDir:    parseDir,
		ExcludeDirs: excludeDirs,
	}

	rawParser := &raw.ManifestParser{BaseManifestParser: base}

	switch parserType {
	case TypeHelm:
		return &helm.ManifestParser{Raw: rawParser}
	case TypeKustomize:
		return &kustomize.ManifestParser{Raw: rawParser}
	default:
		return rawParser
	}
}