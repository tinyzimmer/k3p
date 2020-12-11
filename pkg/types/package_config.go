package types

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"gopkg.in/yaml.v2"
)

// PackageConfig represents an optional configuration structure that can be added
// to packages. It can be used to interpolate manifests at installation time against
// user-provided values. There is currently only one construct available in the form of
// string variables, but this can be extended further in the future.
type PackageConfig struct {
	Variables []PackageVariable `json:"variables,omitempty"`
}

// DeepCopy creates a copy of this PackageConfig.
func (p *PackageConfig) DeepCopy() *PackageConfig {
	out := &PackageConfig{Variables: make([]PackageVariable, len(p.Variables))}
	copy(out.Variables, p.Variables)
	return out
}

// PackageVariable represents a value that can be requested for modification from the
// user. At installation time, these values can be provided either via a configuration,
// CLI flags, or by accepting the defaults. This is a very crude implementation and
// I'm, open to refactoring.
type PackageVariable struct {
	// The name of the variable as it appears in manifests and values files
	Name string `json:"name" yaml:"name"`
	// Optionally override the default prompt when the value is not provided
	Prompt string `json:"prompt,omitempty" yaml:"prompt,omitempty"`
	// An optional default to provide for the value.
	Default string `json:"default,omitempty" yaml:"default,omitempty"`
}

// PackageConfigFromFile will unmarshal a file containing a package configuration.
func PackageConfigFromFile(path string) (*PackageConfig, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	body, err := ioutil.ReadAll(f)
	if err != nil {
		return nil, err
	}
	var cfg PackageConfig
	if strings.HasSuffix(path, ".json") {
		err = json.Unmarshal(body, &cfg)
	} else if strings.HasSuffix(path, ".yaml") || strings.HasSuffix(path, ".yml") {
		err = yaml.Unmarshal(body, &cfg)
	} else {
		return nil, fmt.Errorf("%s is not a valid yaml or json file", path)
	}
	return &cfg, err
}
