package parser

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v2"
	"helm.sh/helm/pkg/chartutil"

	"github.com/tinyzimmer/k3p/pkg/log"
	"github.com/tinyzimmer/k3p/pkg/types"
	"github.com/tinyzimmer/k3p/pkg/util"
)

// NewManifestParser returns an interface for parsing container images from the given directory.
func NewManifestParser(parseDir string, excludeDirs []string, cfg *types.PackageConfig) types.ManifestParser {
	return &ManifestParser{
		BaseManifestParser: NewBaseManifestParser(parseDir, excludeDirs, cfg),
	}
}

// ManifestParser implements a types.ManifestParser that extracts image names from
// raw kubernetes manifests.
// Helm functionality was included in this implementation as well for simplicity sake, but I'd
// ultimately like this divided into separate structs.
type ManifestParser struct{ *BaseManifestParser }

// ParseImages implements the types.ManifestParser interface. It walks the configured directory,
// skipping those that are excluded. If a valid kubernetes yaml file is found, it is loaded
// and checked for container image references.
func (p *ManifestParser) ParseImages() ([]string, error) {
	// Initialize a slice for images
	images := make([]string, 0)

	renderVars := make(map[string]string)
	if p.PackageConfig != nil {
		renderVars = p.PackageConfig.DefaultVars()
	}

	err := filepath.Walk(p.GetParseDir(), func(file string, info os.FileInfo, lastErr error) error {
		// Check previous error first to avoid panic
		if lastErr != nil {
			return lastErr
		}

		// If directory we either want to check if it's a helm chart or should be skipped entirely
		if info.IsDir() {
			// Check if this entire directory is excluded
			if p.IsExcluded(info.Name()) {
				log.Info("Skipping excluded directory", file)
				return filepath.SkipDir
			}
			// Check if it's a helm chart
			if ok, err := chartutil.IsChartDir(file); err == nil && ok {
				log.Info("Detected helm chart at", file)
				containerImages, err := p.detectImagesFromHelmChart(file)
				if err != nil {
					return err
				}
				if len(containerImages) > 0 {
					images = appendIfMissing(images, containerImages...)
				}
				return filepath.SkipDir
			}
			return nil
		}

		// Check if we have a packaged helm chart
		if strings.HasSuffix(info.Name(), ".tgz") && isHelmArchive(file) {
			log.Info("Detected helm chart at", file)
			containerImages, err := p.detectImagesFromHelmChart(file)
			if err != nil {
				return err
			}
			if len(containerImages) > 0 {
				images = appendIfMissing(images, containerImages...)
			}
			return nil
		}

		// Check if the current file does not have a yaml extension
		if !strings.HasSuffix(info.Name(), "yaml") && !strings.HasSuffix(info.Name(), "yml") {
			log.Debug("Skipping non-yaml file", file)
			return nil
		}

		// We have a yaml file, parse it for images
		containerImages, err := p.parseFileForImages(file, renderVars)
		if err != nil {
			return err
		}
		if len(containerImages) > 0 {
			images = appendIfMissing(images, containerImages...)
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

	renderVars := make(map[string]string)
	if p.PackageConfig != nil {
		renderVars = p.PackageConfig.DefaultVars()
	}

	err := filepath.Walk(p.GetParseDir(), func(file string, info os.FileInfo, lastErr error) error {
		// Check previous error first to avoid panic
		if lastErr != nil {
			return lastErr
		}

		// If directory we want to check if it's a helm chart or should be skipped entirely
		if info.IsDir() {
			// Check if this entire directory is excluded
			if p.IsExcluded(info.Name()) {
				log.Info("Skipping excluded directory", file)
				return filepath.SkipDir
			}
			if ok, err := chartutil.IsChartDir(file); err == nil && ok {
				log.Infof("Packaging helm chart: %q\n", file)
				helmArtifacts, err := p.packageHelmChartToArtifacts(file)
				if err != nil {
					return err
				}
				artifacts = append(artifacts, helmArtifacts...)
				return filepath.SkipDir
			}
			return nil
		}

		// Check if we have an already packaged helm chart
		if strings.HasSuffix(info.Name(), ".tgz") && isHelmArchive(file) {
			log.Info("Detected helm chart at", file)
			helmArtifacts, err := p.packageHelmChartToArtifacts(file)
			if err != nil {
				return err
			}
			artifacts = append(artifacts, helmArtifacts...)
			return nil
		}

		// Check if the current file does not have a yaml extension
		if !strings.HasSuffix(info.Name(), "yaml") && !strings.HasSuffix(info.Name(), "yml") {
			log.Debug("Skipping non-yaml file", file)
			return nil
		}

		// We have a yaml file, split and decode it
		data, err := ioutil.ReadFile(file)
		if err != nil {
			return err
		}

		if len(renderVars) > 0 {
			data, err = util.RenderBody(data, renderVars)
			if err != nil {
				return err
			}
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
			log.Debugf("Skipping %q for manifest parsing since it contains invalid kubernetes yaml\n", file)
			return nil
		}

		log.Infof("Detected kubernetes manifest: %q\n", file)

		// queue up the artifact
		f, err := os.Open(file)
		if err != nil {
			return err
		}
		artifacts = append(artifacts, &types.Artifact{
			Name: p.StripParseDir(file),
			Type: types.ArtifactManifest,
			Body: f,
			Size: info.Size(),
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
