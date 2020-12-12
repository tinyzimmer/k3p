package types

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"reflect"
	"strings"

	"github.com/tinyzimmer/k3p/pkg/log"
	"gopkg.in/yaml.v2"
)

// PackageConfig represents an optional configuration structure that can be added
// to packages. It can be used to configure manifests at installation time against distributor
// and user-provided values. Because of the way this is implemented currently, it is not possible
// for the installing user to override any --disable placed in the ServerConfig. This may need to be
// refactored, but at the same time if a package maintainer wants to disable something, it's probably
// for good reason.
type PackageConfig struct {
	// Variables that will be used to interpolate manifests at installation time.
	Variables []PackageVariable `json:"variables,omitempty" yaml:"variables,omitempty"`
	// ServerConfig represents configurations to apply to k3s server nodes. They can be bundled into
	// the package and optionally overwritten by the user at installation. Any long-form flags accepted
	// by k3s server without the leading "--" can be used as keys to Flags along with their cooresponding values.
	// A list can be provided for the value to signal specifying the flag multiple times.
	ServerConfig map[string]string `json:"serverConfig,omitempty" yaml:"serverConfig,omitempty"`
	// AgentConfig represents configurations to apply to k3s agent nodes. They can be bundled into
	// the package and optionally overwritten by the user at installation. Any long-form flags accepted
	// by k3s agent without the leading "--" can be used as keys to Flags along with their cooresponding values.
	// A list can be provided for the value to signal specifying the flag multiple times.
	AgentConfig map[string]string `json:"agentConfig,omitempty" yaml:"agentConfig,omitempty"`
	// HelmValues is a map of chart names to either a list of filenames containing values for that chart, or a single
	// map of inline value declarations.
	HelmValues map[string]interface{} `json:"helmValues,omitempty" yaml:"helmValues,omitempty"`
}

// ExpandHelmValues will iterate the helm values in this package configuration, and deserialize any encountered files
// into mapped representations.
func (p *PackageConfig) ExpandHelmValues(rootDir string) error {
	if len(p.HelmValues) == 0 {
		return nil
	}
	expandedValues := make(map[string]interface{})
	for chartName, values := range p.HelmValues {
		switch reflect.TypeOf(values).Kind() {
		case reflect.Slice:
			valueFiles := values.([]interface{})
			for _, file := range valueFiles {
				filename, ok := file.(string)
				if !ok {
					return fmt.Errorf("cannot open non string filename %+v", file)
				}
				relPath := path.Join(rootDir, path.Dir(filename), path.Base(filename))
				log.Debugf("Reading values file from %q\n", relPath)
				body, err := ioutil.ReadFile(relPath)
				if err != nil {
					return err
				}
				var vals map[string]interface{}
				if err := yaml.Unmarshal(body, &vals); err != nil {
					return fmt.Errorf("error unmarshaling %q: %s", relPath, err.Error())
				}
				expandedValues[chartName] = vals
			}
		case reflect.Map:
			expandedValues[chartName] = values
		default:
			return fmt.Errorf("Unable to deserialize type %v into helm values", reflect.TypeOf(values).Kind())
		}
	}
	p.HelmValues = expandedValues
	return nil
}

// ServerArgs merges the given overrides on top of the server arguments represented
// by this package configuration.
func (p *PackageConfig) ServerArgs(overrides []string) []string {
	if len(p.ServerConfig) == 0 {
		return overrides
	}
	out := make([]string, len(overrides))
	copy(out, overrides)
	for flag, val := range p.ServerConfig {
		if !flagKeyExists(overrides, flag) {
			out = appendFlag(out, flag, val)
		}
	}
	return out
}

// AgentArgs merges the given overrides on top of the agent arguments represented
// by this package configuration.
func (p *PackageConfig) AgentArgs(overrides []string) []string {
	if len(p.AgentConfig) == 0 {
		return overrides
	}
	out := make([]string, len(overrides))
	copy(out, overrides)
	for flag, val := range p.AgentConfig {
		if !flagKeyExists(overrides, flag) {
			out = appendFlag(out, flag, val)
		}
	}
	return out
}

func appendFlag(flags []string, key string, val interface{}) []string {
	if valStr, ok := val.(string); ok && valStr == "" {
		return append(flags, fmt.Sprintf("--%s", key))
	}
	switch reflect.TypeOf(val).Kind() {
	case reflect.String:
		return append(flags, fmt.Sprintf("--%s=%s", key, val))
	case reflect.Slice:
		valSlc := val.([]interface{})
		for _, v := range valSlc {
			flags = appendFlag(flags, key, v)
		}
		return flags
	case reflect.Int, reflect.Int32, reflect.Int64:
		return append(flags, fmt.Sprintf("--%s=%d", key, val))
	}
	log.Warningf("Invalid type for value %+v in %s, ignoring", val, key)
	return flags
}

func flagKeyExists(fields []string, key string) bool {
	for _, field := range fields {
		trim := strings.TrimPrefix(field, "--")
		if trim == key {
			return true
		}
	}
	return false
}

// DeepCopy creates a copy of this PackageConfig.
func (p *PackageConfig) DeepCopy() *PackageConfig {
	out := &PackageConfig{
		Variables:    make([]PackageVariable, len(p.Variables)),
		ServerConfig: make(map[string]string),
		AgentConfig:  make(map[string]string),
		HelmValues:   make(map[string]interface{}),
	}
	copy(out.Variables, p.Variables)
	for k, v := range p.ServerConfig {
		out.ServerConfig[k] = v
	}
	for k, v := range p.AgentConfig {
		out.AgentConfig[k] = v
	}
	for k, v := range p.HelmValues {
		out.HelmValues[k] = v
	}
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

// InstallConfig represents the values that were collected at installation time. It is used
// to serialize the configuration used to disk for future node-add/join operations.
type InstallConfig struct {
	// Options passed at installation
	InstallOptions *InstallOptions
}

// DeepCopy creates a copy of this InstallConfig.
func (i *InstallConfig) DeepCopy() *InstallConfig {
	return &InstallConfig{
		InstallOptions: i.InstallOptions.DeepCopy(),
	}
}
