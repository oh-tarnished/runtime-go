package config

import (
	"bytes"
	"errors"
	"fmt"

	"github.com/BurntSushi/toml"
	koanfToml "github.com/knadh/koanf/parsers/toml"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/providers/rawbytes"
	"github.com/knadh/koanf/v2"
	"github.com/oh-tarnished/runtime-go/config/shared"
)

// TomlParser is an enum for selecting the desired TOML parsing library.
type TomlParser string

const (
	DefaultTomlParser TomlParser = "toml"      // Uses BurntSushi/toml
	KoanfTomlParser   TomlParser = "koanftoml" // Uses knadh/koanf for stateful loading and merging.
)

// Toml provides methods for handling TOML data, aware of the session's state.
type Toml struct {
	io     *IO
	parser TomlParser
	koanf  *koanf.Koanf // Holds the shared instance from the Config session.
}

// newToml creates a new Toml instance with its required dependencies.
func newToml(io *IO, parserType TomlParser, k *koanf.Koanf) *Toml {
	shared.Pulse.Logger.Debugf("Initializing new Toml handler with parser: '%s'", parserType)
	if parserType == KoanfTomlParser && k == nil {
		panic("Koanf instance is required for KoanfTomlParser")
	}
	return &Toml{
		io:     io,
		parser: parserType,
		koanf:  k,
	}
}

// Load reads a TOML file and loads its contents into the session's Koanf state.
// This method is only effective when using the KoanfTomlParser.
func (t *Toml) Load(path string) error {
	shared.Pulse.Logger.Debugf("Toml.Load called for path: %s", path)
	if t.parser != KoanfTomlParser {
		err := errors.New("Load is only supported when using KoanfTomlParser")
		shared.Pulse.Logger.Errorf("Toml.Load aborted: %v", err)
		return err
	}
	resolvedPath, err := t.io.resolvePath(path)
	if err != nil {
		shared.Pulse.Logger.Errorf("Toml.Load failed to resolve path '%s': %v", path, err)
		return err
	}

	shared.Pulse.Logger.Debugf("Toml.Load resolved path to '%s', loading into Koanf", resolvedPath)
	if err := t.koanf.Load(file.Provider(resolvedPath), koanfToml.Parser()); err != nil {
		shared.Pulse.Logger.Errorf("Toml.Load failed for path '%s': %v", resolvedPath, err)
		return err
	}
	shared.Pulse.Logger.Debugf("Toml.Load success for path: %s", resolvedPath)
	return nil
}

// Unmarshal extracts data from the Koanf session state into a struct.
func (t *Toml) Unmarshal(koanfPath string, out any) error {
	shared.Pulse.Logger.Debugf("Toml.Unmarshal called for Koanf path: '%s'", koanfPath)
	if t.parser != KoanfTomlParser {
		err := errors.New("Unmarshal from state is only supported when using KoanfTomlParser")
		shared.Pulse.Logger.Errorf("Toml.Unmarshal aborted: %v", err)
		return err
	}

	if err := t.koanf.Unmarshal(koanfPath, out); err != nil {
		shared.Pulse.Logger.Errorf("Toml.Unmarshal failed for Koanf path '%s': %v", koanfPath, err)
		return err
	}
	shared.Pulse.Logger.Debugf("Toml.Unmarshal success for Koanf path: '%s'", koanfPath)
	return nil
}

// Read is a convenience method that reads a single file and unmarshals it.
// For KoanfTomlParser, it will CLEAR previous state before loading the new file.
func (t *Toml) Read(path string, out any) error {
	shared.Pulse.Logger.Debugf("Toml.Read called for path: %s", path)
	fileBytes, err := t.io.ReadFile(path)
	if err != nil {
		shared.Pulse.Logger.Errorf("Toml.Read could not read file '%s': %v", path, err)
		return err
	}
	shared.Pulse.Logger.Debugf("Toml.Read successfully read %d bytes from '%s'", len(fileBytes), path)

	switch t.parser {
	case KoanfTomlParser:
		shared.Pulse.Logger.Debugf("Toml.Read using Koanf parser for '%s'", path)
		if err := t.koanf.Load(rawbytes.Provider(fileBytes), koanfToml.Parser()); err != nil {
			shared.Pulse.Logger.Errorf("Toml.Read(Koanf) failed to load bytes: %v", err)
			return err
		}
		return t.koanf.Unmarshal("", out)
	default: // DefaultTomlParser
		shared.Pulse.Logger.Debugf("Toml.Read using default parser for '%s'", path)
		if err := toml.Unmarshal(fileBytes, out); err != nil {
			shared.Pulse.Logger.Errorf("Toml.Read(Default) failed to unmarshal: %v", err)
			return err
		}
		return nil
	}
}

// Write marshals a Go struct to TOML and writes it to a file.
func (t *Toml) Write(path string, data any) error {
	shared.Pulse.Logger.Debugf("Toml.Write called for path: %s", path)
	var buf bytes.Buffer
	if err := toml.NewEncoder(&buf).Encode(data); err != nil {
		shared.Pulse.Logger.Errorf("Toml.Write failed to marshal data for path '%s': %v", path, err)
		return fmt.Errorf("failed to marshal data to toml: %w", err)
	}

	shared.Pulse.Logger.Debugf("Toml.Write marshaled %d bytes, writing to file '%s'", buf.Len(), path)
	if err := t.io.WriteFile(path, buf.Bytes()); err != nil {
		shared.Pulse.Logger.Errorf("Toml.Write failed to write file '%s': %v", path, err)
		return err
	}
	shared.Pulse.Logger.Debugf("Toml.Write success for path: %s", path)
	return nil
}
