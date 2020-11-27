package raw

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/tinyzimmer/k3p/pkg/log"
	"github.com/tinyzimmer/k3p/pkg/types"
	"github.com/tinyzimmer/k3p/pkg/util"
	"gopkg.in/yaml.v2"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	corescheme "k8s.io/client-go/kubernetes/scheme"
)

// ManifestParser implements a types.ManifestParser that extracts image names from
// raw kubernetes manifests.
type ManifestParser struct{ *types.BaseManifestParser }

// ParseImages implements the types.ManifestParser interface. It walks the configured directory,
// skipping those that are excluded. If a valid kubernetes yaml file is found, it is loaded
// and checked for container image references.
func (p *ManifestParser) ParseImages() ([]string, error) {
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
				log.Info("Skipping excluded directory", path)
				return filepath.SkipDir
			}
			return nil
		}

		// Check if the current file does not have a yaml extension
		if !strings.HasSuffix(info.Name(), "yaml") && !strings.HasSuffix(info.Name(), "yml") {
			log.Debug("Skipping non-yaml file", path)
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
				log.Debugf("Skipping invalid kubernetes object in %q: %s", path, err.Error()) // TODO: verbose logging
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

// ParseManifests implements the types.ManifestParser interface. It iterates the directories for yaml files,
// and checks to see if every object within them is a valid kubernetes object. If it is, it is returned to be added
// to the bundle.
func (p *ManifestParser) ParseManifests() ([]*types.Artifact, error) {
	artifacts := make([]*types.Artifact, 0)

	err := filepath.Walk(p.GetParseDir(), func(path string, info os.FileInfo, lastErr error) error {
		// Check previous error first to avoid panic
		if lastErr != nil {
			return lastErr
		}

		// If directory we either want to continue or skip it entirely
		if info.IsDir() {
			// Check if this entire directory is excluded
			if p.IsExcluded(info.Name()) {
				log.Info("Skipping excluded directory", path)
				return filepath.SkipDir
			}
			return nil
		}

		// Check if the current file does not have a yaml extension
		if !strings.HasSuffix(info.Name(), "yaml") && !strings.HasSuffix(info.Name(), "yml") {
			log.Debug("Skipping non-yaml file", path)
			return nil
		}

		// We have a yaml file, split and decode it
		data, err := ioutil.ReadFile(path)
		if err != nil {
			return err
		}

		// iterate all the yaml objects in the file
		rawYamls := strings.Split(string(data), "---")
		// assume file is valid until hitting a condition that it isn't
		fileIsValid := true
		for _, raw := range rawYamls {
			// Check if this is empty space
			if strings.TrimSpace(raw) == "" {
				continue
			}
			rawMap := map[string]interface{}{}
			if err := yaml.Unmarshal([]byte(raw), &rawMap); err != nil {
				log.Debug("Could not decode yaml object, skipping file:", err)
				fileIsValid = false
				break
			}
			if !util.IsK8sObject(rawMap) {
				log.Debug("Object does not appear to be a valid kubernetes manifest:", rawMap)
				fileIsValid = false
				break
			}
		}

		// if the file doesn't appear valid, continue
		if !fileIsValid {
			log.Warningf("Skipping %q since it contains invalid kubernetes yaml", path)
			return nil
		}

		log.Infof("Detected kubernetes manifest: %q", path)

		// queue up the artifact
		f, err := os.Open(path)
		if err != nil {
			return err
		}
		artifacts = append(artifacts, &types.Artifact{
			Name: p.StripParseDir(path),
			Type: types.ArtifactManifest,
			Body: f,
		})

		return nil
	})

	// Return any fatal walking errors
	if err != nil {
		return nil, fmt.Errorf("Error walking directory %q: %v", p.GetParseDir(), err)
	}

	return artifacts, nil
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
