package helm

import "github.com/tinyzimmer/k3p/pkg/parser/raw"

// ImageParser implements a types.ImageParser that extracts image names from
// templated helm charts.
type ImageParser struct{ Raw *raw.ImageParser }

// Parse implements the types.ImageParser interface. It walks the configured directory,
// skipping those that are excluded. If a valid helm chart is found, it is templated,
// and then its contents are parsed by the raw image parser.
//
// The CLI itself should have flags that can be passed to the templating, and the interface will
// have to be redesigned to allow for those configurations to flow down.
func (p *ImageParser) Parse() ([]string, error) {
	return nil, nil
}
