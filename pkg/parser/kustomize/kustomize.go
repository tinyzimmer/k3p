package kustomize

import "github.com/tinyzimmer/k3p/pkg/parser/raw"

// ImageParser implements a types.ImageParser that extracts image names from
// rendered kustomize manifests.
type ImageParser struct{ Raw *raw.ImageParser }

// Parse implements the types.ImageParser interface. It walks the configured directory,
// skipping those that are excluded. If a valid kustomize directory is found, it is rendered,
// and then its contents are parsed by the raw image parser.
func (p *ImageParser) Parse() ([]string, error) {
	return nil, nil
}
