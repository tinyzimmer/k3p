package helm

import (
	"github.com/tinyzimmer/k3p/pkg/parser/raw"
	"github.com/tinyzimmer/k3p/pkg/types"
)

// ManifestParser implements a types.ManifestParser that extracts image names from
// templated helm charts.
type ManifestParser struct{ Raw *raw.ManifestParser }

// ParseImages implements the types.ManifestParser interface. It walks the configured directory,
// skipping those that are excluded. If a valid helm chart is found, it is templated,
// and then its contents are parsed by the raw image parser.
//
// The CLI itself should have flags that can be passed to the templating, and the interface will
// have to be redesigned to allow for those configurations to flow down.
func (p *ManifestParser) ParseImages() ([]string, error) {
	return nil, nil
}

func (p *ManifestParser) ParseManifests() ([]*types.Artifact, error) { return nil, nil }
