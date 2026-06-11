// Package config provides a session-based toolkit for managing application
// configuration, file I/O, and file watching.
//
// # Quick start
//
// Create a session with New, then use its sub-modules for all I/O and
// config operations. Close the session when your application exits.
//
//	sess, err := config.New(config.Options{
//	    BasePath:    "~/.myapp",
//	    ServiceName: "MYAPP",
//	    YamlParser:  config.KoanfYamlParser,
//	})
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer sess.Close()
//
// # Layered configuration
//
// Load configuration from multiple sources in priority order; later sources
// override earlier ones. Call Unmarshal once after all loads are complete:
//
//	// 1. Struct defaults (lowest priority)
//	sess.LoadDefaults(myDefaults)
//
//	// 2. YAML file
//	sess.Yaml.Load("config.yaml")
//
//	// 3. Environment variables override everything
//	sess.LoadEnv("MYAPP_")
//
//	var cfg AppConfig
//	sess.Yaml.Unmarshal("", &cfg)
//
// # Parsers
//
// Each format (YAML, JSON, TOML) ships with two parser modes:
//
//   - Default (e.g. [DefaultYamlParser]): one-shot unmarshal via the
//     format's canonical library (yaml.v3, encoding/json, BurntSushi/toml).
//     Use for simple "read file → struct" cases.
//
//   - Koanf (e.g. [KoanfYamlParser]): stateful, mergeable loading via
//     knadh/koanf. Use when you need to overlay multiple files or merge
//     environment variables into a single config state.
//
// # File watching
//
// Watch a file or directory and receive change events on a channel:
//
//	w, err := sess.NewWatcher("config.yaml", nil)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	w.Start()
//	defer w.Stop()
//
//	for event := range w.Events() {
//	    fmt.Printf("changed: %s (%d bytes)\n", event.Path, len(event.Content))
//	}
//
// # Archives
//
// Create and extract gzipped tar archives through the Compression sub-module:
//
//	err := sess.Compression.CreateTarGz("backup.tar.gz", []string{"data.yaml", "keys.json"})
//
//	err = sess.Compression.ExtractTarGz("backup.tar.gz", "/tmp/restore", 0)
package config
