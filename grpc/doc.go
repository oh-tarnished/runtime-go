/*
Package grpc provides a HybridServer for creating servers that serve both gRPC
and a JSON/HTTP gateway. It wraps common functionality like TLS configuration,
service registration, and graceful shutdown to reduce boilerplate code.

# Overview

The main component in this package is the HybridServer. It's configured using a
functional options pattern. You create a server with base options, then apply
additional configuration using `With...` functions.

The server supports:
  - gRPC, HTTP/1.1, and experimental HTTP/3 protocols.
  - Automatic TLS configuration from certificate files.
  - Graceful shutdown on SIGINT and SIGTERM signals.
  - Standard gRPC Health Checking and Server Reflection services.

Configuration can also be overridden by environment variables, which is useful
for deploying in different environments.

# Usage

The following example shows how to set up a complete HybridServer. It assumes
you have a `greeter.proto` file that has been compiled with both
`protoc-gen-go-grpc` and `protoc-gen-grpc-gateway`.

### 1. Implement Your Service and Registrars

First, implement your gRPC service's logic. Then, create registrar functions
that this package can use to add your service to the server.

```go
// service/greeter.go
package service

import (

	"context"
	"your/project/pb" // Assumes compiled protobufs are in this path.

	"[github.com/oh-tarnished/runtime-go/grpc](https://github.com/oh-tarnished/runtime-go/grpc)"

)

// server implements the Greeter service.

	type server struct {
		pb.UnimplementedGreeterServer
	}

// SayHello is the implementation of the RPC.

	func (s *server) SayHello(ctx context.Context, req *pb.HelloRequest) (*pb.HelloReply, error) {
		return &pb.HelloReply{Message: "Hello, " + req.GetName()}, nil
	}

// RegisterGRPCService registers the gRPC service with the server.

	func RegisterGRPCService(s *grpc.GRPCServer) {
		pb.RegisterGreeterServer(s, &server{})
	}

// RegisterHTTPGateway registers the HTTP gateway handler for the Greeter service.

	func RegisterHTTPGateway(mux *grpc.ServeMux, endpoint string, opts []grpc.DialOption) error {
		return pb.RegisterGreeterHandlerFromEndpoint(context.Background(), mux, endpoint, opts)
	}
*/
package grpc
