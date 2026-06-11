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

// Options configures a new Config session. Pass it to New.
type Options struct {
	// BasePath is the root directory for all relative file operations.
	// Tilde (~) expansion is supported. Defaults to the current directory.
	BasePath string
	// ServiceName is used as the env-variable prefix fallback in LoadEnv when
	// no explicit prefix is provided (e.g. "MYAPP" → looks for "MYAPP_*").
	ServiceName string
	// WatcherPath, when non-empty, creates an embedded file watcher at this
	// path. The watcher is accessible at Config.Watcher after New returns.
	WatcherPath string
	// YamlParser selects the engine for YAML file operations.
	// Defaults to DefaultYamlParser (yaml.v3) when unset.
	YamlParser YamlParser
	// JsonParser selects the engine for JSON file operations.
	// Defaults to DefaultJsonParser (encoding/json) when unset.
	JsonParser JsonParser
	// TomlParser selects the engine for TOML file operations.
	// Defaults to DefaultTomlParser (BurntSushi/toml) when unset.
	TomlParser TomlParser
}

// ConfigIO is an alias for Options retained for source compatibility.
//
// Deprecated: use Options.
type ConfigIO = Options

// Config is the primary session object returned by New.
// Access sub-modules (IO, Yaml, Json, Toml, Compression) as fields; call the
// receiver methods (LoadEnv, LoadDefaults, String, Int, Bool) for merged-state
// lookups. Call Close when the session is no longer needed.
type Config struct {
	// IO provides thread-safe file system operations (read, write, list, mkdir).
	IO *IO
	// Yaml provides YAML read/write and optional Koanf-state loading.
	Yaml *Yaml
	// Json provides JSON read/write and optional Koanf-state loading.
	Json *Json
	// Toml provides TOML read/write and optional Koanf-state loading.
	Toml *Toml
	// Watcher is an embedded file-change watcher, available only when
	// Options.WatcherPath was non-empty at construction time.
	Watcher *Watcher
	// Compression provides gzipped-tar create/extract operations.
	Compression *Compression
	// ServiceName is the service identifier set via Options.ServiceName.
	// Used as the env-variable prefix fallback in LoadEnv.
	ServiceName string

	koanf *koanf.Koanf
}

// New creates and initializes a new Config session from the provided options.
//
// The session's IO, Yaml, Json, Toml, and Compression sub-modules are always
// initialized. The embedded Watcher is only created when opts.WatcherPath is
// non-empty. Call Close when the session is no longer needed.
func New(opts Options) (*Config, error) {
	shared.Pulse.Logger.Debugf("New Config session requested with BasePath: '%s'", opts.BasePath)
	io, err := newIO(opts.BasePath)
	if err != nil {
		shared.Pulse.Logger.Errorf("Failed to initialize IO module: %v", err)
		return nil, err
	}

	var watcher *Watcher
	if opts.WatcherPath != "" {
		shared.Pulse.Logger.Debugf("WatcherPath provided, creating embedded watcher for: '%s'", opts.WatcherPath)
		watcher, err = newWatcher(opts.WatcherPath, nil)
		if err != nil {
			shared.Pulse.Logger.Errorf("Failed to create default watcher: %v", err)
			return nil, err
		}
	}

	k := koanf.New(".")
	s := &Config{
		IO:          io,
		Yaml:        newYaml(io, opts.YamlParser, k),
		Json:        newJson(io, opts.JsonParser, k),
		Toml:        newToml(io, opts.TomlParser, k),
		Watcher:     watcher,
		Compression: newCompression(io),
		koanf:       k,
		ServiceName: opts.ServiceName,
	}
	shared.Pulse.Logger.Infof("New Config session created successfully for BasePath: '%s'", opts.BasePath)
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
	_ = shared.Close() // best-effort telemetry flush; non-fatal in test/offline environments
	return nil
}
