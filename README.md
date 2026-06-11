# runtime-go

[![CI](https://github.com/oh-tarnished/runtime-go/actions/workflows/ci.yml/badge.svg)](https://github.com/oh-tarnished/runtime-go/actions/workflows/ci.yml)
[![License](https://img.shields.io/badge/license-Apache%202.0-blue.svg)](LICENSE)

A battle-tested Go utility belt for building production-grade services. Each module is an independent Go module — import only what you need.

## Modules

| Module | Import path | Description |
| ------ | ----------- | ----------- |
| [ulid](ulid/) | `github.com/oh-tarnished/runtime-go/ulid` | ULID generation with protobuf support, timestamp extraction, monotonic stitching |
| [resourcename](resourcename/) | `github.com/oh-tarnished/runtime-go/resourcename` | AIP-style hierarchical resource name marshalling/unmarshalling via struct tags |
| [config](config/) | `github.com/oh-tarnished/runtime-go/config` | Layered config from YAML/JSON/TOML files, env vars, structs, and live file watching |
| [network](network/) | `github.com/oh-tarnished/runtime-go/network` | HTTP, GraphQL, and WebSocket clients with retries, reconnect, and connectivity checks |
| [grpc](grpc/) | `github.com/oh-tarnished/runtime-go/grpc` | `HybridServer` — gRPC + HTTP/JSON gateway + HTTP/3 + MCP + OpenTelemetry in one |
| [system](system/) | `github.com/oh-tarnished/runtime-go/system` | Linux system utilities: user queries, power management (requires elevated privileges) |

## Requirements

- Go 1.22 or later
- Each module declares its own minimum version in its `go.mod`

## Installation

Each module is versioned and imported independently:

```bash
# ULID generation
go get github.com/oh-tarnished/runtime-go/ulid

# Resource name marshal/unmarshal
go get github.com/oh-tarnished/runtime-go/resourcename

# Layered config
go get github.com/oh-tarnished/runtime-go/config

# HTTP / GraphQL / WebSocket clients
go get github.com/oh-tarnished/runtime-go/network

# gRPC + HTTP gateway hybrid server
go get github.com/oh-tarnished/runtime-go/grpc

# Linux system utilities
go get github.com/oh-tarnished/runtime-go/system
```

## Quick start

### ulid

```go
import "github.com/oh-tarnished/runtime-go/ulid"

// Generate a new ULID (panics on entropy failure — appropriate for app startup paths)
id := ulid.Generate()
fmt.Println(id.String())       // 01JXKZ...
fmt.Println(id.Time())         // timestamp extracted from the ULID
fmt.Println(id.TimestampProto()) // *timestamppb.Timestamp

// Parse and validate
id, err := ulid.Parse("01JXKZ...")
```

### resourcename

```go
import "github.com/oh-tarnished/runtime-go/resourcename"

type User struct {
    _  struct{} `resource:"//api.example.com/users/{id}"`
    ID string   `resource:"id"`
}

// Struct → resource string
name, err := resourcename.MarshalResource(&User{ID: "u42"})
// "//api.example.com/users/u42"

// Resource string → struct
var u User
err = resourcename.UnmarshalResource("//api.example.com/users/u42", &u)
```

### config

```go
import "github.com/oh-tarnished/runtime-go/config"

cfg, err := config.New(config.ConfigIO{
    BasePath:    "./config",
    YamlParser:  config.GoYaml,
})

cfg.LoadDefaults(myDefaults)
cfg.LoadEnv("APP_")
cfg.Yaml.Load("base.yaml")

port := cfg.Int("server.port")
```

### network

```go
import "github.com/oh-tarnished/runtime-go/network"

conn, err := network.NewConnection(network.HTTPConnClient)
conn, err = conn.WithOpts(network.ConnectionOptions{
    URL: network.URLOptions{
        Scheme: network.HTTPS,
        Host:   "api.example.com",
        Paths:  []string{"/v1/resource"},
    },
    Timeout: 10 * time.Second,
    Retries: 3,
})

client, _ := conn.AsHTTPConnectionType()
data, err := client.RequestSync(ctx, network.GET, conn.Options().URL, nil, nil, 0, 0)
```

### grpc

```go
import (
    "github.com/oh-tarnished/runtime-go/grpc"
    "github.com/oh-tarnished/runtime-go/grpc/options"
)

srv := grpc.NewHybridServer(options.Options{
    ServiceName: "my-service",
    Environment: options.Production,
    GRPC:        options.Endpoint{Host: "0.0.0.0", Port: 50051},
    HTTP:        options.Endpoint{Host: "0.0.0.0", Port: 8080},
})

srv.RegisterGRPC(myGRPCServiceFunc)
srv.RegisterHTTP(myHTTPGatewayFunc)
srv.Start()
```

## Development

This repository uses a [Go workspace](https://go.dev/ref/mod#go-work) so all modules can be developed together without publishing:

```bash
git clone https://github.com/oh-tarnished/runtime-go
cd runtime-go

# Run all tests across the workspace
go work sync
go test github.com/oh-tarnished/runtime-go/ulid/...
go test github.com/oh-tarnished/runtime-go/resourcename/...
go test github.com/oh-tarnished/runtime-go/network/...
```

### Linting

```bash
# Install golangci-lint
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Lint all modules
for dir in ulid resourcename config network grpc system; do
    (cd $dir && golangci-lint run ./...)
done
```

### Adding a new module

1. Create the directory and run `go mod init github.com/oh-tarnished/runtime-go/<name>`
2. Add the module to `go.work`: `go work use ./<name>`
3. Add a row to the module table in this README

## Security

See [SECURITY.md](SECURITY.md) for the responsible disclosure policy.

## Contributing

Issues and pull requests are welcome. Please open an issue before submitting a large change so we can discuss the approach first.

## License

Apache 2.0 — see [LICENSE](LICENSE).

© 2026 oh-tarnished
