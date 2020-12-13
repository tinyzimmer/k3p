package types

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"reflect"
	"regexp"
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
	// The raw untemplated contents of the config - only populated by loaders from this package and archivers
	Raw []byte `json:"raw,omitempty" yaml:"raw,omitempty"`
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
	return PackageConfigFromReader(f)
}

// PackageConfigFromFileWithVars will unmarshal a file containing a package configuration, taking
// any provided variables into account while rendering.
func PackageConfigFromFileWithVars(path string, vars map[string]string) (*PackageConfig, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return PackageConfigFromReaderWithVars(f, vars)
}

// PackageConfigFromReader will unmarshal the contents of the given reader into a package configuration.
func PackageConfigFromReader(rdr io.Reader) (*PackageConfig, error) {
	return PackageConfigFromReaderWithVars(rdr, nil)
}

// PackageConfigFromReaderWithVars will unmarshal the contents of the given reader, taking the
// provided pre-processed variables into account with rendering.
func PackageConfigFromReaderWithVars(rdr io.Reader, vars map[string]string) (*PackageConfig, error) {
	rawBody, err := ioutil.ReadAll(rdr)
	if err != nil {
		return nil, err
	}

	// We do a multi-pass load to account for templates and variables that can be throughout the config

	// Treat the entire contents as one yaml object during the first pass
	rawJoined := []byte(strings.Replace(string(rawBody), "---", "\n", -1))
	cfg := PackageConfig{Raw: rawJoined}

	if vars != nil {
		cfg.Raw, err = render(cfg.Raw, vars)
		if err != nil {
			return nil, err
		}
	}

	// First try to unmarshal as is. If we encounter errors we assume there are variables to be
	// parsed first.
	log.Debug("Attempting first pass load of configuration")
	if err = yaml.Unmarshal(cfg.Raw, &cfg); err == nil {
		log.Debug("Loaded configuration on first pass, returning")
		return &cfg, nil
	}
	log.Debug("Could not load the configuration on the first pass:", err.Error())

	// If there is a variable section, it will need to be split logically from the rest of the configuration.
	// This will be documented.
	log.Debug("Running second pass load of the configuration in parts to look for variables")
	splitBody := strings.Split(string(rawBody), "---")
	for _, part := range splitBody {
		if err = yaml.Unmarshal([]byte(part), &cfg); err == nil {
			if len(cfg.Variables) > 0 {
				log.Debugf("Loaded the following variables for processing: %+v\n", cfg.Variables)
			}
			continue
		}
		log.Debug("Could not load section during second pass:", err.Error())
	}
	if len(cfg.Variables) == 0 {
		return nil, errors.New("Could not load configuration as raw yaml and no variables found for templating")
	}

	// Add any additional variables to the map variables
	// TODO: We'd only reach this point if vars was nil anyway i think
	if vars == nil {
		vars = make(map[string]string)
	}
	for _, v := range cfg.Variables {
		vars[v.Name] = v.Default
	}

	// Try to render the raw body
	log.Debug("Running third pass template render with variables from previous pass")
	templatedBody, err := render(cfg.Raw, vars)
	if err != nil {
		return nil, err
	}

	// Finally try to load the templated configuration
	log.Debug("Running final pass yaml decode of rendered body")
	return &cfg, yaml.Unmarshal(templatedBody, &cfg)
}

// reBlockStart is a regex matching the start of a root-level yaml block
var reBlockStart = regexp.MustCompile("^[a-zA-Z0-9]")

// RawHelmValuesForChart will attempt to return the raw, untemplated helm values for the given chart.
// TODO: The fact that this has to exist probably means I took a wrong turn somewhere.
func (p *PackageConfig) RawHelmValuesForChart(chartName string) ([]byte, error) {
	scanner := bufio.NewScanner(bytes.NewReader(p.Raw))
	// Scan to the helm values block
	for scanner.Scan() {
		text := scanner.Text()
		// Scan to the helm values block
		if strings.HasPrefix(text, "helmValues:") {
			break
		}
	}
	// Scan to the indented chart name
	var indent string
	for scanner.Scan() {
		text := scanner.Text()
		if strings.HasPrefix(strings.TrimSpace(text), chartName) {
			// indent signals both that we found something, and where to seek to
			// in the block
			indent = strings.Replace(text, chartName+":", "", 1)
			break
		}
	}
	if indent == "" {
		return nil, fmt.Errorf("could not parse a raw helm values block for %q", chartName)
	}
	// Scan everything until the next block matching the indent level of the chart name,
	// or the end of the helm values block
	var chartValues bytes.Buffer
	re, err := regexp.Compile(fmt.Sprintf("^%s[a-zA-Z0-9]", indent))

	if err != nil {
		return nil, err
	}
	for scanner.Scan() {
		text := scanner.Text()
		if !re.MatchString(text) && !reBlockStart.MatchString(text) {
			fmt.Fprintln(&chartValues, strings.Replace(text, indent, "", 1))
			continue
		}
		break
	}
	return chartValues.Bytes(), nil
}

// ApplyVariables will template this entire configuration with the given variables
func (p *PackageConfig) ApplyVariables(vars map[string]string) error {
	out, err := render(p.Raw, vars)
	if err != nil {
		return err
	}
	return yaml.Unmarshal(out, p)
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

// DefaultVars returns the default variables for this configuration to be used for manifest
// discovery while building packages.
func (p *PackageConfig) DefaultVars() map[string]string {
	vars := make(map[string]string)
	for _, v := range p.Variables {
		vars[v.Name] = v.Default
	}
	return vars
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
		Raw:          make([]byte, len(p.Raw)),
	}
	copy(out.Variables, p.Variables)
	copy(out.Raw, p.Raw)
	for k, v := range p.ServerConfig {
		out.ServerConfig[k] = v
	}
	for k, v := range p.AgentConfig {
		out.AgentConfig[k] = v
	}
	for k, v := range p.HelmValues {
		// This technically does not do the whole job, need to generate
		// proper deepcopy functions
		out.HelmValues[k] = v
	}
	return out
}
