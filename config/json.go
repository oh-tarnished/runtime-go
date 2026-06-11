package config

import (
	"encoding/json"
	"errors"
	"fmt"

	koanfJson "github.com/knadh/koanf/parsers/json"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/providers/rawbytes"
	"github.com/knadh/koanf/v2"
	"github.com/oh-tarnished/runtime-go/config/shared"
)

// JsonParser is an enum for selecting the desired JSON parsing library.
type JsonParser string

const (
	DefaultJsonParser JsonParser = "json"      // Uses encoding/json for standard JSON parsing.
	KoanfJsonParser   JsonParser = "koanfjson" // Uses knadh/koanf for stateful loading and merging.
)

// Json provides methods for handling JSON data, aware of the session's state.
type Json struct {
	io     *IO
	parser JsonParser
	koanf  *koanf.Koanf // Holds the shared instance from the Config session.
}

// newJson creates a new Json instance with its required dependencies.
func newJson(io *IO, parserType JsonParser, k *koanf.Koanf) *Json {
	shared.Pulse.Logger.Debugf("Initializing new Json handler with parser: '%s'", parserType)
	if parserType == KoanfJsonParser && k == nil {
		// This should not happen if called from New, but it's a good safeguard.
		panic("Koanf instance is required for KoanfJsonParser")
	}
	return &Json{
		io:     io,
		parser: parserType,
		koanf:  k,
	}
}

// Load reads a JSON file and loads its contents into the session's Koanf state.
// This method is only effective when using the KoanfJsonParser.
func (j *Json) Load(path string) error {
	shared.Pulse.Logger.Debugf("Json.Load called for path: %s", path)
	if j.parser != KoanfJsonParser {
		err := errors.New("Load is only supported when using KoanfJsonParser")
		shared.Pulse.Logger.Errorf("Json.Load aborted: %v", err)
		return err
	}
	resolvedPath, err := j.io.resolvePath(path)
	if err != nil {
		shared.Pulse.Logger.Errorf("Json.Load failed to resolve path '%s': %v", path, err)
		return err
	}

	shared.Pulse.Logger.Debugf("Json.Load resolved path to '%s', loading into Koanf", resolvedPath)
	if err := j.koanf.Load(file.Provider(resolvedPath), koanfJson.Parser()); err != nil {
		shared.Pulse.Logger.Errorf("Json.Load failed for path '%s': %v", resolvedPath, err)
		return err
	}
	shared.Pulse.Logger.Debugf("Json.Load success for path: %s", resolvedPath)
	return nil
}

// Unmarshal extracts data from the Koanf session state into a struct.
func (j *Json) Unmarshal(koanfPath string, out any) error {
	shared.Pulse.Logger.Debugf("Json.Unmarshal called for Koanf path: '%s'", koanfPath)
	if j.parser != KoanfJsonParser {
		err := errors.New("Unmarshal from state is only supported when using KoanfJsonParser")
		shared.Pulse.Logger.Errorf("Json.Unmarshal aborted: %v", err)
		return err
	}

	if err := j.koanf.Unmarshal(koanfPath, out); err != nil {
		shared.Pulse.Logger.Errorf("Json.Unmarshal failed for Koanf path '%s': %v", koanfPath, err)
		return err
	}
	shared.Pulse.Logger.Debugf("Json.Unmarshal success for Koanf path: '%s'", koanfPath)
	return nil
}

// Read is a convenience method that reads a single file and unmarshals it.
// For KoanfJsonParser, it will CLEAR previous state before loading the new file.
func (j *Json) Read(path string, out any) error {
	shared.Pulse.Logger.Debugf("Json.Read called for path: %s", path)
	fileBytes, err := j.io.ReadFile(path)
	if err != nil {
		shared.Pulse.Logger.Errorf("Json.Read could not read file '%s': %v", path, err)
		return err
	}
	shared.Pulse.Logger.Debugf("Json.Read successfully read %d bytes from '%s'", len(fileBytes), path)

	switch j.parser {
	case KoanfJsonParser:
		shared.Pulse.Logger.Debugf("Json.Read using Koanf parser for '%s'", path)
		if err := j.koanf.Load(rawbytes.Provider(fileBytes), koanfJson.Parser()); err != nil {
			shared.Pulse.Logger.Errorf("Json.Read(Koanf) failed to load bytes: %v", err)
			return err
		}
		return j.koanf.Unmarshal("", out)
	default: // DefaultJsonParser
		shared.Pulse.Logger.Debugf("Json.Read using default parser for '%s'", path)
		if err := json.Unmarshal(fileBytes, out); err != nil {
			shared.Pulse.Logger.Errorf("Json.Read(Default) failed to unmarshal: %v", err)
			return err
		}
		return nil
	}
}

// Write marshals a Go struct to JSON and writes it to a file.
func (j *Json) Write(path string, data any) error {
	shared.Pulse.Logger.Debugf("Json.Write called for path: %s", path)
	jsonBytes, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		shared.Pulse.Logger.Errorf("Json.Write failed to marshal data for path '%s': %v", path, err)
		return fmt.Errorf("failed to marshal data to json: %w", err)
	}

	shared.Pulse.Logger.Debugf("Json.Write marshaled %d bytes, writing to file '%s'", len(jsonBytes), path)
	if err := j.io.WriteFile(path, jsonBytes); err != nil {
		shared.Pulse.Logger.Errorf("Json.Write failed to write file '%s': %v", path, err)
		return err
	}
	shared.Pulse.Logger.Debugf("Json.Write success for path: %s", path)
	return nil
}
