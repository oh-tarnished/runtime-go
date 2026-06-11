// Package network provides GraphQL, HTTP, and WebSocket clients with consistent
// connection options, structured logging, and optional connectivity verification.
//
// # Overview
//
// The package exposes three client types, all created via NewConnection(clientType)
// and configured with WithOpts(opts):
//
//   - GraphQL: run queries and mutations against a GraphQL endpoint; supports
//     typed operations and raw query strings.
//   - HTTP: perform GET, POST, PUT, PATCH, DELETE with retries and context support.
//   - WebSocket: full-duplex connection with Send/Receive, optional auto-reconnect
//     and ping/pong keepalive.
//
// # Connection and connectivity
//
// By default, Connect (and thus WithOpts) verifies that the target is reachable
// before returning:
//
//   - GraphQL: runs a small introspection query (or GraphQLConnectivityQuery if set).
//   - HTTP: sends HEAD to the URL; if the server returns 405, falls back to GET
//     and drains the body (up to 1 MiB) so the connection can be reused.
//   - WebSocket: performs a real Dial; there is no "skip" for WebSocket.
//
// Set ConnectionOptions.SkipConnectivityCheck to true to skip the HTTP/GraphQL
// reachability check when the server is known to be up or the first request will
// act as the check. Set GraphQLConnectivityQuery to override the default query
// used for the GraphQL connectivity check (e.g. for servers that limit introspection).
//
// # Timeouts and defaults
//
// If ConnectionOptions.Timeout is zero or negative, DefaultTimeout (10s) is used
// for Connect and for request/dial timeouts. RetryDelay defaults to 2s when
// retries are used.
//
// # Example
//
//	opts := network.ConnectionOptions{
//	    URL: network.URLOptions{
//	        Scheme: network.HTTPS,
//	        Host:   "api.example.com",
//	        Paths:  []string{"/graphql"},
//	    },
//	    Timeout:                 10 * time.Second,
//	    SkipConnectivityCheck:   true,
//	    GraphQLConnectivityQuery: network.DefaultGraphQLConnectivityQuery,
//	}
//	conn, err := network.NewConnection(network.GraphQLConnClient)
//	if err != nil { ... }
//	_, err = conn.WithOpts(opts)
//	if err != nil { ... }
//	client, _ := conn.AsGraphQLConnectionType()
package network

import (
	"fmt"
	"net/url"
	"time"

	"github.com/oh-tarnished/runtime-go/network/shared"
)

// ClientType identifies which kind of network client to create.
// Pass it to NewConnection to obtain a GraphQL, HTTP, or WebSocket client.
type ClientType string

// DefaultTimeout is used when ConnectionOptions.Timeout is zero or negative.
// It applies to connection establishment, HTTP requests, and GraphQL operations.
var DefaultTimeout = 10 * time.Second

const (
	GraphQLConnClient   ClientType = "graphql"   // GraphQL client for queries and mutations
	HTTPConnClient      ClientType = "http"      // HTTP client for REST-style requests
	WebsocketConnClient ClientType = "websocket" // WebSocket client for full-duplex messaging
)

// ConnectionOptions holds settings shared by all client types.
// Pass it to WithOpts or to a client's Connect method.
type ConnectionOptions struct {
	// URL is the target endpoint (scheme, host, paths, optional query params).
	// For HTTP/GraphQL, use http or https; for WebSocket, use ws or wss.
	URL URLOptions
	// Timeout applies to connection establishment and to individual requests.
	// If zero or negative, DefaultTimeout is used.
	Timeout time.Duration
	// Headers are sent on every request (and on WebSocket handshake).
	Headers map[string]string
	// Retries is the maximum number of retries for HTTP requests (see Request).
	Retries int
	// RetryDelay is the pause between retries. If zero, a default of 2s is used.
	RetryDelay time.Duration

	// SkipConnectivityCheck, when true, skips the initial HTTP/GraphQL reachability
	// check. Use when the server is known to be up or the first request will act as
	// the check. Ignored for WebSocket, where connection is established by Dial.
	SkipConnectivityCheck bool

	// GraphQLConnectivityQuery overrides the default query used to verify the
	// GraphQL server is reachable. If empty, DefaultGraphQLConnectivityQuery
	// is used. Only meaningful for GraphQL clients; set for strict servers that
	// limit introspection.
	GraphQLConnectivityQuery string
}

// URLOptions describes the target URL: scheme, host, path(s), and optional query params.
// Host may include a port (e.g. "localhost:8080"). Paths is a list of path segments;
// the client selects one by index (e.g. pathIndex 0 for the first path).
type URLOptions struct {
	Scheme URLScheme         // Protocol: http, https, ws, or wss
	Host   string            // Hostname and optional port (e.g. "api.example.com:443")
	Paths  []string          // Paths to choose from (e.g. ["/graphql", "/v2/graphql"])
	Params map[string]string // Optional query parameters
}

// URLScheme is the protocol part of a URL.
type URLScheme string

const (
	HTTP  URLScheme = "http"  // Plain HTTP
	HTTPS URLScheme = "https" // TLS HTTP
	WS    URLScheme = "ws"    // Plain WebSocket
	WSS   URLScheme = "wss"   // TLS WebSocket
)

// Client is the common interface implemented by GraphQL, HTTP, and WebSocket clients.
// Use it when you need to treat connections uniformly (e.g. Connect, Close, Reconnect).
// For type-specific operations, cast via AsGraphQLConnectionType, AsHTTPConnectionType,
// or AsWebSocketConnectionType.
type Client interface {
	Connect(opts ConnectionOptions) error
	Close() error
	Reconnect() error
}

// Network wraps a single client (GraphQL, HTTP, or WebSocket) and exposes connection
// lifecycle and type-cast helpers. Create with NewConnection; configure with WithOpts.
type Network struct {
	client  Client
	options ConnectionOptions
}

// NewConnection creates a new network client of the given type. The client is not
// connected until WithOpts (or the underlying client's Connect) is called.
// Returns an error if clientType is not one of GraphQLConnClient, HTTPConnClient,
// or WebsocketConnClient.
func NewConnection(clientType ClientType) (*Network, error) {
	shared.Pulse.Logger.Debugf("NewConnection called clientType=%s", clientType)
	var client Client
	switch clientType {
	case GraphQLConnClient:
		shared.Pulse.Logger.Info("Creating GraphQL client")
		client = &GraphQLClient{}
	case HTTPConnClient:
		shared.Pulse.Logger.Info("Creating HTTP client")
		client = &HTTPClient{}
	case WebsocketConnClient:
		shared.Pulse.Logger.Info("Creating WebSocket client")
		client = &WebSocketClient{}
	default:
		shared.Pulse.Logger.Errorf("Unsupported client type clientType=%s", clientType)
		return nil, fmt.Errorf("client type not supported: %s", clientType)
	}
	shared.Pulse.Logger.Infof("Network connection created clientType=%s", clientType)
	return &Network{client: client, options: defaultOpts}, nil
}

// Close closes the underlying client connection and releases shared resources.
// Safe to call multiple times; subsequent calls are no-ops if already closed.
func (n *Network) Close() error {
	shared.Pulse.Logger.Debugf("Closing network connection host=%s", n.options.URL.Host)
	_ = shared.Close() // best-effort telemetry flush; non-fatal in test/offline environments
	return n.client.Close()
}

// Reconnect re-establishes the connection using the same options as the last
// successful Connect/WithOpts. For HTTP/GraphQL this may run the connectivity
// check again unless SkipConnectivityCheck was set.
func (n *Network) Reconnect() error {
	shared.Pulse.Logger.Debugf("Reconnecting to network host=%s", n.options.URL.Host)
	return n.client.Reconnect()
}

// Client returns the underlying Client implementation (GraphQL, HTTP, or WebSocket).
// Use the As*ConnectionType methods to obtain the concrete type for type-specific APIs.
func (n *Network) Client() Client {
	shared.Pulse.Logger.Debugf("Getting network client host=%s", n.options.URL.Host)
	return n.client
}

// WithOpts applies the given connection options and establishes the connection
// (including the optional connectivity check). Returns the receiver for chaining
// and an error if connection fails. On success, the client is ready for use.
func (n *Network) WithOpts(opts ConnectionOptions) (*Network, error) {
	shared.Pulse.Logger.Debugf("WithOpts called host=%s timeout=%v retries=%d", opts.URL.Host, opts.Timeout, opts.Retries)
	n.options = opts
	err := n.client.Connect(opts)
	if err != nil {
		shared.Pulse.Logger.Errorf("Failed to connect with options error=%v host=%s", err, opts.URL.Host)
		return n, err
	}
	shared.Pulse.Logger.Infof("Connected successfully host=%s", opts.URL.Host)
	return n, nil
}

// defaultOpts defines the default connection options
var defaultOpts = ConnectionOptions{}

// AsHTTPConnectionType returns the underlying client as an *HTTPClient.
// Returns an error if this Network was not created with HTTPConnClient.
func (n *Network) AsHTTPConnectionType() (*HTTPClient, error) {
	httpClient, ok := n.Client().(*HTTPClient)
	if !ok {
		shared.Pulse.Logger.Error("Failed to cast to HTTPClient")
		return nil, fmt.Errorf("failed to cast to HTTPClient")
	}
	return httpClient, nil
}

// AsGraphQLConnectionType returns the underlying client as an *GraphQLClient.
// Returns an error if this Network was not created with GraphQLConnClient.
func (n *Network) AsGraphQLConnectionType() (*GraphQLClient, error) {
	graphQLClient, ok := n.Client().(*GraphQLClient)
	if !ok {
		shared.Pulse.Logger.Error("Failed to cast to GraphQLClient")
		return nil, fmt.Errorf("failed to cast to GraphQLClient")
	}
	return graphQLClient, nil
}

// AsWebSocketConnectionType returns the underlying client as an *WebSocketClient.
// Returns an error if this Network was not created with WebsocketConnClient.
func (n *Network) AsWebSocketConnectionType() (*WebSocketClient, error) {
	websocketClient, ok := n.Client().(*WebSocketClient)
	if !ok {
		shared.Pulse.Logger.Error("Failed to cast to WebsocketClient")
		return nil, fmt.Errorf("failed to cast to WebSocketClient")
	}
	return websocketClient, nil
}

// buildFullURL constructs a full URL from URLOptions using the path at pathIndex.
// It validates scheme (http, https, ws, wss), non-empty host, non-empty paths,
// and pathIndex in range. Query parameters from Params are appended when present.
func buildFullURL(urlOptions URLOptions, pathIndex int) (string, error) {
	shared.Pulse.Logger.Debugf("buildFullURL called scheme=%s host=%s pathIndex=%d pathCount=%d", urlOptions.Scheme, urlOptions.Host, pathIndex, len(urlOptions.Paths))

	// Validate the scheme
	if urlOptions.Scheme != "http" && urlOptions.Scheme != "https" && urlOptions.Scheme != "ws" && urlOptions.Scheme != "wss" {
		shared.Pulse.Logger.Errorf("Invalid URL scheme scheme=%s", urlOptions.Scheme)
		return "", fmt.Errorf("invalid URL scheme: %s. Must be 'http', 'https', 'ws', or 'wss'", urlOptions.Scheme)
	}

	// Validate host
	if urlOptions.Host == "" {
		shared.Pulse.Logger.Error("Host cannot be empty")
		return "", fmt.Errorf("host cannot be empty")
	}

	// Validate paths
	if len(urlOptions.Paths) == 0 {
		shared.Pulse.Logger.Error("Paths array cannot be empty")
		return "", fmt.Errorf("paths array cannot be empty")
	}

	// Build the base URL without re-encoding slashes
	u := url.URL{
		Scheme: string(urlOptions.Scheme),
		Host:   urlOptions.Host,
	}

	// Validate pathIndex bounds
	if pathIndex < 0 || pathIndex >= len(urlOptions.Paths) {
		shared.Pulse.Logger.Errorf("PathIndex out of bounds pathIndex=%d pathsLength=%d", pathIndex, len(urlOptions.Paths))
		return "", fmt.Errorf("pathIndex %d out of bounds for paths array of length %d", pathIndex, len(urlOptions.Paths))
	}

	// Concatenate the path properly to avoid double encoding
	path := urlOptions.Paths[pathIndex]
	if len(path) > 0 && path[0] != '/' {
		path = "/" + path // Ensure the path starts with a forward slash
	}
	u.Path = path

	// Add query parameters if they exist
	if len(urlOptions.Params) > 0 {
		query := u.Query()
		for key, value := range urlOptions.Params {
			query.Set(key, value)
		}
		u.RawQuery = query.Encode()
	}

	finalURL := u.String()
	shared.Pulse.Logger.Debugf("URL built successfully url=%s hasParams=%v", finalURL, len(urlOptions.Params) > 0)
	return finalURL, nil
}
