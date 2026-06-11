// Package config provides a unified, session-based library for Go applications to
// handle common operational tasks. It simplifies file system I/O, layered configuration
// management, file watching, and archive handling through a single, easy-to-use
// session object.
package config

import (
	"errors"
	"strings"

	_ "github.com/joho/godotenv/autoload"
	"github.com/knadh/koanf/providers/env/v2"
	"github.com/knadh/koanf/providers/structs"
	"github.com/knadh/koanf/v2"

	"github.com/oh-tarnished/runtime-go/config/shared"
)

// Config holds all the necessary settings for initializing a new Config session.
type ConfigIO struct {
	// BasePath is the root directory for all relative file operations (read, write, etc.).
	BasePath string
	// WatcherPath is an optional path to create a default, embedded watcher for.
	// If this is set, a watcher will be available at `session.Watcher`.
	WatcherPath string
	// YamlParser selects the engine to use for YAML file operations.
	YamlParser YamlParser
	// JsonParser selects the engine to use for JSON file operations.
	JsonParser JsonParser
	// TomlParser selects the engine to use for TOML file operations.
	TomlParser TomlParser
}

// Config is the primary session object for all library operations.
// It holds the state and provides access to all sub-modules like IO, Yaml, and Watcher.
type Config struct {
	// IO provides thread-safe methods for basic file system operations.
	IO *IO
	// Yaml provides methods for handling YAML files.
	Yaml *Yaml
	// Json provides methods for handling JSON files.
	Json *Json
	// Toml provides methods for handling TOML files.
	Toml *Toml
	// Watcher is the default, embedded file watcher. It is only initialized if
	// a WatcherPath is provided in the initial Config.
	Watcher *Watcher
	// Compression provides methods for compressing and decompressing data.
	Compression *Compression
	// koanf is the internal instance used for stateful configuration management.
	koanf *koanf.Koanf
	// ServiceName is the name of the service.
	ServiceName string
}

// New creates and initializes a new Config session based on the provided configuration.
func New(config ConfigIO, serviceName ...string) (*Config, error) {
	shared.Pulse.Logger.Debugf("New Config session requested with BasePath: '%s'", config.BasePath)
	io, err := newIO(config.BasePath)
	if err != nil {
		shared.Pulse.Logger.Errorf("Failed to initialize IO module: %v", err)
		return nil, err
	}

	var watcher *Watcher
	if config.WatcherPath != "" {
		shared.Pulse.Logger.Debugf("WatcherPath provided, creating embedded watcher for: '%s'", config.WatcherPath)
		watcher, err = newWatcher(config.WatcherPath, nil)
		if err != nil {
			shared.Pulse.Logger.Errorf("Failed to create default watcher: %v", err)
			return nil, err
		}
	}

	k := koanf.New(".")
	yaml := newYaml(io, config.YamlParser, k)
	json := newJson(io, config.JsonParser, k)
	toml := newToml(io, config.TomlParser, k)

	var serviceNameStr string
	if len(serviceName) > 0 {
		serviceNameStr = serviceName[0]
	}
	s := &Config{
		IO:          io,
		Yaml:        yaml,
		Json:        json,
		Toml:        toml,
		Watcher:     watcher,
		Compression: newCompression(io),
		koanf:       k,
		ServiceName: serviceNameStr,
	}
	shared.Pulse.Logger.Infof("New Config session created successfully for BasePath: '%s'", config.BasePath)
	return s, nil
}

// String returns the string value for a given dot-delimited configuration path.
func (s *Config) String(path string) string {
	shared.Pulse.Logger.Debugf("Reading string from Koanf path: '%s'", path)
	return s.koanf.String(path)
}

// Int returns the integer value for a given dot-delimited configuration path.
func (s *Config) Int(path string) int {
	shared.Pulse.Logger.Debugf("Reading int from Koanf path: '%s'", path)
	return s.koanf.Int(path)
}

// Bool returns the boolean value for a given dot-delimited configuration path.
func (s *Config) Bool(path string) bool {
	shared.Pulse.Logger.Debugf("Reading bool from Koanf path: '%s'", path)
	return s.koanf.Bool(path)
}

// Koanf returns the underlying Koanf instance, allowing for advanced configuration management.
func (s *Config) Koanf() *koanf.Koanf {
	shared.Pulse.Logger.Debug("Providing access to underlying Koanf instance.")
	return s.koanf
}

// NewWatcher creates a new file system watcher.
// The provided path is resolved relative to the session's BasePath, allowing you to
// create multiple, independent watchers for different files or directories.
func (s *Config) NewWatcher(path string, options *WatcherOptions) (*Watcher, error) {
	shared.Pulse.Logger.Debugf("NewWatcher factory called for path: '%s'", path)
	if s.IO == nil {
		err := errors.New("IO module is not initialized")
		shared.Pulse.Logger.Errorf("NewWatcher failed: %v", err)
		return nil, err
	}
	resolvedPath, err := s.IO.resolvePath(path)
	if err != nil {
		shared.Pulse.Logger.Errorf("NewWatcher failed to resolve path '%s': %v", path, err)
		return nil, err
	}
	shared.Pulse.Logger.Debugf("NewWatcher path resolved to: '%s'", resolvedPath)
	return newWatcher(resolvedPath, options)
}

// LoadEnv loads environment variables with a given prefix into the Koanf state.
// It uses a default transformation:
// 1. Converts keys to lowercase.
// 2. Removes the given prefix.
// 3. Replaces double underscores (__) with dots (.) for explicit nesting.
// 4. For simple keys (no double underscores), replaces single underscores with dots.
// 5. Preserves underscores in field names when using double underscore notation.
// For example, with prefix "APP_":
// - APP_SERVER_HOST becomes "server.host"
// - APP_DATABASE_USER becomes "database.user"
// - APP_DATABASE__SETTINGS__MAX_CONNECTIONS becomes "database.settings.max_connections"
func (s *Config) LoadEnv(prefix string) error {
	shared.Pulse.Logger.Infof("Loading environment variables with prefix: '%s' for service: '%s'", prefix, s.ServiceName)

	// Define the default transformation function.
	transform := func(k, v string) (string, any) {
		// Use the provided prefix, or fall back to service name with underscore
		effectivePrefix := prefix
		if effectivePrefix == "" {
			effectivePrefix = s.ServiceName + "_"
		}

		key := strings.ToLower(strings.TrimPrefix(k, effectivePrefix))

		// Check if this uses double underscore notation (for nested fields with underscores in names)
		if strings.Contains(key, "__") {
			// Convert double underscores to dots for explicit nesting
			key = strings.ReplaceAll(key, "__", ".")
		} else {
			// For simple keys, convert single underscores to dots
			key = strings.ReplaceAll(key, "_", ".")
		}

		shared.Pulse.Logger.Debugf("Transforming env var: '%s' -> '%s' = '%v'", k, key, v)
		return key, v
	}

	// The first argument to env.Provider is the delimiter Koanf uses for its keys.
	if err := s.koanf.Load(env.Provider(".", env.Opt{
		Prefix:        prefix,
		TransformFunc: transform,
	}), nil); err != nil {
		shared.Pulse.Logger.Errorf("Failed to load environment variables: %v", err)
		return err
	}

	return nil
}

// LoadEnvWithOpts loads environment variables using a custom env.Opt struct.
// This gives you full control over the prefix and transformation logic.
func (s *Config) LoadEnvWithOpts(opts env.Opt) error {
	shared.Pulse.Logger.Infof("Loading environment variables with custom options (Prefix: '%s')", opts.Prefix)

	if err := s.koanf.Load(env.Provider(".", opts), nil); err != nil {
		shared.Pulse.Logger.Errorf("Failed to load environment variables with custom options: %v", err)
		return err
	}
	return nil
}

// LoadDefaults loads configuration from a Go struct into the Koanf state.
// It uses the `koanf` struct tags to map fields to configuration keys. This is
// typically used to set the initial, default values for the configuration.
func (s *Config) LoadDefaults(defaults any) error {
	shared.Pulse.Logger.Info("Loading default values from struct")
	if err := s.koanf.Load(structs.Provider(defaults, "koanf"), nil); err != nil {
		shared.Pulse.Logger.Errorf("Failed to load defaults from struct: %v", err)
		return err
	}
	return nil
}

func (s *Config) Close() error {
	shared.Pulse.Logger.Info("Closing Config session and releasing resources")
	shared.Close()
	return nil
}
