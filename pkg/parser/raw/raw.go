package raw

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/tinyzimmer/k3p/pkg/types"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	corescheme "k8s.io/client-go/kubernetes/scheme"
)

// ImageParser implements a types.ImageParser that extracts image names from
// raw kubernetes manifests.
type ImageParser struct{ *types.BaseImageParser }

// Parse implements the types.ImageParser interface. It walks the configured directory,
// skipping those that are excluded. If a valid kubernetes yaml file is found, it is loaded
// and checked for container image references.
func (p *ImageParser) Parse() ([]string, error) {
	// create a new scheme
	sch := runtime.NewScheme()

	// currently only supports core APIs, could consider some way of dynamically adding CRD support
	// full list: https://github.com/kubernetes/client-go/blob/master/kubernetes/scheme/register.go
	_ = corescheme.AddToScheme(sch)

	// Assign a decode function
	decode := serializer.NewCodecFactory(sch).UniversalDeserializer().Decode

	// Initialize a slice for images
	images := make([]string, 0)

	err := filepath.Walk(p.GetParseDir(), func(path string, info os.FileInfo, lastErr error) error {
		// Check previous error first to avoid panic
		if lastErr != nil {
			return lastErr
		}

		// If directory we either want to continue or skip it entirely
		if info.IsDir() {
			// Check if this entire directory is excluded
			if p.IsExcluded(info.Name()) {
				log.Println("Skipping excluded directory", path)
				return filepath.SkipDir
			}
			return nil
		}

		// Check if the current file does not have a yaml extension
		if !strings.HasSuffix(info.Name(), "yaml") && !strings.HasSuffix(info.Name(), "yml") {
			// log.Println("Skipping non-yaml file", path) // TODO: Setup verbose logging
			return nil
		}

		// We have a yaml file, split and decode it
		data, err := ioutil.ReadFile(path)
		if err != nil {
			return err
		}

		// iterate all the yaml objects in the file
		rawYamls := strings.Split(string(data), "---")
		for _, raw := range rawYamls {
			// Check if this is empty space
			if strings.TrimSpace(raw) == "" {
				continue
			}
			// Decode the object
			obj, _, err := decode([]byte(raw), nil, nil)
			if err != nil {
				// log.Printf("Skipping invalid kubernetes object in %q: %s", path, err.Error()) // TODO: verbose logging
				continue
			}
			// Append any images to the local images to be downloaded
			if objImgs := parseObjectForImages(obj); len(objImgs) > 0 {
				images = appendIfMissing(images, objImgs...)
			}
		}

		return nil
	})

	// Return any fatal walking errors
	if err != nil {
		return nil, fmt.Errorf("Error walking directory %q: %v", p.GetParseDir(), err)
	}

	return images, nil
}

func appendIfMissing(inSlc []string, args ...string) []string {
	outSlc := make([]string, len(inSlc))
	copy(outSlc, inSlc)
ArgLoop:
	for _, arg := range args {
		for _, present := range outSlc {
			if present == arg {
				continue ArgLoop
			}
		}
		outSlc = append(outSlc, arg)
	}
	return outSlc
}
