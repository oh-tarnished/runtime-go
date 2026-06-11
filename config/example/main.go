package main

import (
	"fmt"
	"log"

	"github.com/oh-tarnished/runtime-go/config"
)

type Config struct {
	Server struct {
		Host string `koanf:"host"`
		Port int    `koanf:"port"`
	} `koanf:"server"`
	Database struct {
		Host string `koanf:"host"`
		User string `koanf:"user"`
	} `koanf:"database"`
	Features struct {
		CacheEnabled bool `koanf:"cache_enabled"`
	} `koanf:"features"`
	Log struct {
		Level string `koanf:"level"`
	} `koanf:"log"`
}

func main() {
	// 1. Initialize a single config session.
	s, err := config.New(config.Options{
		BasePath:    "./sample",
		ServiceName: "MYAPP",
		YamlParser:  config.KoanfYamlParser,
		JsonParser:  config.KoanfJsonParser,
		TomlParser:  config.KoanfTomlParser,
	})
	if err != nil {
		log.Fatalf("Failed to create session: %v", err)
	}

	defer s.Close()

	// 	// 2. Setup: Create config files.
	configDir := "config"
	// 	s.IO.CreateDirectory(configDir)

	// 	s.IO.WriteFile(configDir+"/base.yaml", []byte(`
	// server:
	//   host: "0.0.0.0"
	//   port: 8080
	// database:
	//   user: "yaml_default"
	// features:
	//   cache_enabled: true
	// log:
	//   level: "info"
	// `))
	// 	s.IO.WriteFile(configDir+"/prod.json", []byte(`{ "server": { "port": 443 } }`))

	// 	// Add a TOML file with nested fields including underscores
	// 	s.IO.WriteFile(configDir+"/database.toml", []byte(`
	// [database]
	// host = "db.toml.example.com"
	// user = "toml_user"
	// password = "toml_secret"

	// [database.settings]
	// max_connections = 50
	// timeout = 60

	// [features]
	// cache_enabled = false
	// debug_mode = true
	// `))

	// 3. Initial Load: Load files first, then load environment variables to override.
	fmt.Println("--- Performing Initial Configuration Load ---")
	s.Yaml.Load(configDir + "/base.yaml")
	s.Json.Load(configDir + "/prod.json")
	s.Toml.Load(configDir + "/secrets.toml")

	// NEW: Load environment variables with prefix "APP_". This will be the highest priority.
	if err := s.LoadEnv(""); err != nil {
		log.Fatalf("Error loading environment variables: %v", err)
	}

	// 4. Read the initial merged values from the session.
	fmt.Println("\n--- Final Merged Configuration ---")
	fmt.Printf("Server Port: %d (from JSON file)\n", s.Int("server.port"))
	fmt.Printf("Log Level: '%s' (from Environment)\n", s.String("log.level"))
	fmt.Printf("Database User: '%s' (from Environment)\n", s.String("database.user"))
	fmt.Printf("Database Host: '%s' (from TOML file)\n", s.String("database.host"))
	fmt.Printf("Database Max Connections: %d (from TOML file)\n", s.Int("database.settings.max_connections"))
	fmt.Printf("Features Cache Enabled: %t (from YAML, overridden by TOML)\n", s.Bool("features.cache_enabled"))
	fmt.Printf("Features Debug Mode: %t (from TOML file)\n", s.Bool("features.debug_mode"))

	// 5. Watcher Setup (no changes to this logic)
	// fmt.Printf("\n--- Starting Watcher on '%s' ---\n", configDir)
	// watcher, err := s.NewWatcher(configDir, nil)
	// if err != nil {
	// 	log.Fatalf("Failed to create watcher: %v", err)
	// }
	// watcher.Start()
	// defer watcher.Stop()

	// // Goroutine to handle reloads (note: env variables are not watched, only files)
	// go func() {
	// 	for event := range watcher.Events() {
	// 		fmt.Printf("🔥 File change detected: %s\n", filepath.Base(event.Path))
	// 		// ... reload logic for files ...
	// 	}
	// }()

	// time.Sleep(1 * time.Second)
	fmt.Println("\nExample finished. Run with environment variables set to see overrides.")
	fmt.Println("Try: APP_SERVER_HOST=custom-host APP_DATABASE__SETTINGS__MAX_CONNECTIONS=100 go run main.go")
}
