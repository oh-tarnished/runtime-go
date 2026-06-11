package config

import (
	"errors"
	"fmt"

	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/providers/rawbytes"
	"github.com/knadh/koanf/v2"
	"github.com/oh-tarnished/runtime-go/config/shared"

	yamlv3 "gopkg.in/yaml.v3"
	k8syaml "sigs.k8s.io/yaml"
)

// YamlParser is an enum for selecting the desired YAML parsing library.
type YamlParser string

const (
	DefaultYamlParser YamlParser = "yaml"      // Uses gopkg.in/yaml.v3
	KoanfYamlParser   YamlParser = "koanfyaml" // Uses knadh/koanf for stateful loading and merging.
	K8sYamlParser     YamlParser = "k8syaml"   // Uses sigs.k8s.io/yaml for Kubernetes-style YAML parsing.
)

// Yaml provides methods for handling YAML data, aware of the session's state.
type Yaml struct {
	io     *IO
	parser YamlParser
	koanf  *koanf.Koanf // Holds the shared instance from the Config session.
}

// newYaml creates a new Yaml instance with its required dependencies.
func newYaml(io *IO, parserType YamlParser, k *koanf.Koanf) *Yaml {
	shared.Pulse.Logger.Debugf("Initializing new Yaml handler with parser: '%s'", parserType)
	if parserType == KoanfYamlParser && k == nil {
		panic("Koanf instance is required for KoanfYamlParser")
	}
	return &Yaml{
		io:     io,
		parser: parserType,
		koanf:  k,
	}
}

// Load reads a YAML file and loads its contents into the session's Koanf state.
// This method is only effective when using the KoanfYamlParser.
func (y *Yaml) Load(path string) error {
	shared.Pulse.Logger.Debugf("Yaml.Load called for path: %s", path)
	if y.parser != KoanfYamlParser {
		err := errors.New("Load is only supported when using KoanfYamlParser")
		shared.Pulse.Logger.Errorf("Yaml.Load aborted: %v", err)
		return err
	}
	resolvedPath, err := y.io.resolvePath(path)
	if err != nil {
		shared.Pulse.Logger.Errorf("Yaml.Load failed to resolve path '%s': %v", path, err)
		return err
	}

	shared.Pulse.Logger.Debugf("Yaml.Load resolved path to '%s', loading into Koanf", resolvedPath)
	if err := y.koanf.Load(file.Provider(resolvedPath), yaml.Parser()); err != nil {
		shared.Pulse.Logger.Errorf("Yaml.Load failed for path '%s': %v", resolvedPath, err)
		return err
	}
	shared.Pulse.Logger.Debugf("Yaml.Load success for path: %s", resolvedPath)
	return nil
}

// Unmarshal extracts data from the Koanf session state into a struct.
func (y *Yaml) Unmarshal(koanfPath string, out any) error {
	shared.Pulse.Logger.Debugf("Yaml.Unmarshal called for Koanf path: '%s'", koanfPath)
	if y.parser != KoanfYamlParser {
		err := errors.New("Unmarshal from state is only supported when using KoanfYamlParser")
		shared.Pulse.Logger.Errorf("Yaml.Unmarshal aborted: %v", err)
		return err
	}

	if err := y.koanf.Unmarshal(koanfPath, out); err != nil {
		shared.Pulse.Logger.Errorf("Yaml.Unmarshal failed for Koanf path '%s': %v", koanfPath, err)
		return err
	}
	shared.Pulse.Logger.Debugf("Yaml.Unmarshal success for Koanf path: '%s'", koanfPath)
	return nil
}

// Read is a convenience method that reads a single file and unmarshals it.
func (y *Yaml) Read(path string, out any) error {
	shared.Pulse.Logger.Debugf("Yaml.Read called for path: %s", path)
	fileBytes, err := y.io.ReadFile(path)
	if err != nil {
		shared.Pulse.Logger.Errorf("Yaml.Read could not read file '%s': %v", path, err)
		return err
	}
	shared.Pulse.Logger.Debugf("Yaml.Read successfully read %d bytes from '%s'", len(fileBytes), path)

	switch y.parser {
	case KoanfYamlParser:
		shared.Pulse.Logger.Debugf("Yaml.Read using Koanf parser for '%s'", path)
		// FIX: Use rawbytes.Provider for efficiency as file is already in memory.
		if err := y.koanf.Load(rawbytes.Provider(fileBytes), yaml.Parser()); err != nil {
			shared.Pulse.Logger.Errorf("Yaml.Read(Koanf) failed to load bytes: %v", err)
			return err
		}
		// FIX: Unmarshal the entire config, not a sub-path named after the file path.
		return y.koanf.Unmarshal("", out)
	case K8sYamlParser:
		shared.Pulse.Logger.Debugf("Yaml.Read using K8s parser for '%s'", path)
		if err := k8syaml.Unmarshal(fileBytes, out); err != nil {
			shared.Pulse.Logger.Errorf("Yaml.Read(K8s) failed to unmarshal: %v", err)
			return err
		}
	default: // DefaultYamlParser
		shared.Pulse.Logger.Debugf("Yaml.Read using default parser for '%s'", path)
		if err := yamlv3.Unmarshal(fileBytes, out); err != nil {
			shared.Pulse.Logger.Errorf("Yaml.Read(Default) failed to unmarshal: %v", err)
			return err
		}
	}
	return nil
}

// Write marshals a Go struct to YAML and writes it to a file.
func (y *Yaml) Write(path string, data any) error {
	shared.Pulse.Logger.Debugf("Yaml.Write called for path: %s", path)
	var yamlBytes []byte
	var err error

	switch y.parser {
	case K8sYamlParser:
		shared.Pulse.Logger.Debugf("Yaml.Write using K8s parser for '%s'", path)
		yamlBytes, err = k8syaml.Marshal(data)
	default: // Default and Koanf use yaml.v3 for consistent writing.
		shared.Pulse.Logger.Debugf("Yaml.Write using default parser for '%s'", path)
		yamlBytes, err = yamlv3.Marshal(data)
	}

	if err != nil {
		shared.Pulse.Logger.Errorf("Yaml.Write failed to marshal data for path '%s': %v", path, err)
		return fmt.Errorf("failed to marshal data: %w", err)
	}

	shared.Pulse.Logger.Debugf("Yaml.Write marshaled %d bytes, writing to file '%s'", len(yamlBytes), path)
	if err := y.io.WriteFile(path, yamlBytes); err != nil {
		shared.Pulse.Logger.Errorf("Yaml.Write failed to write file '%s': %v", path, err)
		return err
	}
	shared.Pulse.Logger.Debugf("Yaml.Write success for path: %s", path)
	return nil
}
