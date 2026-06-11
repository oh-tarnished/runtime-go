// WebSocket client implementation.
//
// This file provides the WebSocket client used when ClientType is WebsocketConnClient.
// It supports full-duplex messaging (Send/Receive), optional auto-reconnect on
// read errors, and ping/pong keepalive. Connection is always established by Dial;
// SkipConnectivityCheck is ignored for WebSocket.
package network

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/oh-tarnished/runtime-go/network/shared"
)

// WebSocketClient is a WebSocket connection client. It embeds ConnectionOptions
// and provides Connect, Close, Reconnect, SetAutoReconnect, Send, Receive,
// RetrySend, and Listen. Create via NewConnection(WebsocketConnClient) and
// AsWebSocketConnectionType. All methods are safe for concurrent use.
type WebSocketClient struct {
	conn              *websocket.Conn    // Underlying connection; nil when closed
	ConnectionOptions                    // URL, Timeout, Headers (used on handshake)
	dialer            *websocket.Dialer  // Used for Connect/Reconnect
	pathIndex         int                // Index into URL.Paths for buildFullURL
	mu                sync.RWMutex       // Protects conn and options
	ctx               context.Context    // Canceled on Close to stop ping goroutine
	cancel            context.CancelFunc // Cancel for ctx
	autoReconnect     bool               // When true, Listen attempts Reconnect on read error
	reconnectDelay    time.Duration      // Delay before reconnecting (default 5s)
}

// convertToHTTPHeader builds an http.Header from a string map for the WebSocket handshake.
func convertToHTTPHeader(headers map[string]string) http.Header {
	httpHeaders := http.Header{}
	for key, value := range headers {
		httpHeaders.Add(key, value)
	}
	return httpHeaders
}

// Connect establishes the WebSocket connection to the URL derived from opts and
// pathIndex. If opts.Timeout <= 0, DefaultTimeout is used for the handshake.
// Connect always performs a real Dial; SkipConnectivityCheck is not applied.
// On success, a ping/pong keepalive goroutine is started. Returns an error if
// the handshake fails (e.g. wrong scheme, host unreachable, non-101 response).
func (ws *WebSocketClient) Connect(opts ConnectionOptions) error {
	if opts.Timeout <= 0 {
		opts.Timeout = DefaultTimeout
		shared.Pulse.Logger.Debug("WebSocket Connect using default timeout", "timeout", opts.Timeout)
	}
	shared.Pulse.Logger.Debug("WebSocket Connect called", "host", opts.URL.Host, "scheme", opts.URL.Scheme, "timeout", opts.Timeout)
	ws.mu.Lock()
	defer ws.mu.Unlock()

	ws.ConnectionOptions = opts
	ws.dialer = &websocket.Dialer{
		HandshakeTimeout: opts.Timeout,
	}
	shared.Pulse.Logger.Debug("WebSocket dialer created", "handshakeTimeout", opts.Timeout)

	// Set default reconnect delay if not set
	if ws.reconnectDelay == 0 {
		ws.reconnectDelay = 5 * time.Second
	}
	shared.Pulse.Logger.Debug("WebSocket reconnect delay set", "delay", ws.reconnectDelay)

	// Create context for this connection
	ws.ctx, ws.cancel = context.WithCancel(context.Background())

	httpHeaders := convertToHTTPHeader(opts.Headers)
	shared.Pulse.Logger.Debug("WebSocket headers converted", "headerCount", len(httpHeaders))

	fullURL, err := buildFullURL(opts.URL, ws.pathIndex)
	if err != nil {
		shared.Pulse.Logger.Error("Failed to build WebSocket URL", "error", err, "pathIndex", ws.pathIndex)
		return fmt.Errorf("failed to build WebSocket URL: %w", err)
	}
	shared.Pulse.Logger.Debug("WebSocket URL built", "fullURL", fullURL)

	shared.Pulse.Logger.Infof("Connecting to WebSocket server url=%s", fullURL)
	conn, resp, err := ws.dialer.Dial(fullURL, httpHeaders)
	if resp != nil && resp.Body != nil {
		defer resp.Body.Close()
	}
	if err != nil {
		if resp != nil {
			shared.Pulse.Logger.Debug("WebSocket connection response status", "status", resp.Status, "statusCode", resp.StatusCode)
		}
		shared.Pulse.Logger.Error("Failed to dial WebSocket", "error", err, "url", fullURL, "host", opts.URL.Host)
		return fmt.Errorf("failed to dial WebSocket at %s: %w", opts.URL.Host, err)
	}

	ws.conn = conn
	shared.Pulse.Logger.Infof("Connected to WebSocket server successfully url=%s host=%s", fullURL, opts.URL.Host)

	// Start ping/pong handler
	shared.Pulse.Logger.Debug("Starting WebSocket ping/pong handler")
	ws.startPingPong()

	return nil
}

// Close sends a close frame, closes the connection, cancels the connection context
// (stopping the ping goroutine), and clears the client state. Safe to call multiple
// times; subsequent calls are no-ops if already closed.
func (ws *WebSocketClient) Close() error {
	ws.mu.Lock()
	defer ws.mu.Unlock()

	// Cancel context to stop all goroutines
	if ws.cancel != nil {
		ws.cancel()
	}

	if ws.conn != nil {
		shared.Pulse.Logger.Debug("Closing WebSocket connection", "host", ws.URL.Host)
		// Send close message
		err := ws.conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
		if err != nil {
			shared.Pulse.Logger.Debug("Error sending close message", "error", err)
		}
		err = ws.conn.Close()
		ws.conn = nil
		return err
	}
	shared.Pulse.Logger.Debug("WebSocket connection is already closed", "host", ws.URL.Host)
	return nil
}

// Reconnect closes the current connection (if any) and calls Connect with the
// same ConnectionOptions. Use after a transient failure or to refresh the connection.
func (ws *WebSocketClient) Reconnect() error {
	shared.Pulse.Logger.Debug("Reconnecting to WebSocket server...")

	if ws.conn != nil {
		_ = ws.Close()
	}

	return ws.Connect(ws.ConnectionOptions)
}

// SetAutoReconnect enables or disables automatic reconnection in Listen. When
// enabled, a read error in Listen triggers a sleep for delay (or the default 5s
// if delay <= 0), then Reconnect; on success, listening continues. When disabled,
// Listen returns on the first read error.
func (ws *WebSocketClient) SetAutoReconnect(enabled bool, delay time.Duration) {
	ws.mu.Lock()
	defer ws.mu.Unlock()
	ws.autoReconnect = enabled
	if delay > 0 {
		ws.reconnectDelay = delay
	}
}

// startPingPong starts a goroutine that sends a ping every 30 seconds and sets
// the read deadline on pong. The goroutine exits when ws.ctx is canceled (e.g. on Close).
func (ws *WebSocketClient) startPingPong() {
	ws.conn.SetPongHandler(func(string) error {
		shared.Pulse.Logger.Debug("Received pong from server")
		return ws.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	})

	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				ws.mu.RLock()
				conn := ws.conn
				ws.mu.RUnlock()

				if conn != nil {
					if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
						shared.Pulse.Logger.Debug("Failed to send ping", "error", err)
						return
					}
					shared.Pulse.Logger.Debug("Sent ping to server")
				}
			case <-ws.ctx.Done():
				return
			}
		}
	}()
}

// Send writes a single WebSocket frame. messageType is typically websocket.TextMessage
// or websocket.BinaryMessage. Returns an error if the connection is closed or the
// write fails.
func (ws *WebSocketClient) Send(messageType int, message []byte) error {
	shared.Pulse.Logger.Debug("WebSocket Send called", "messageType", messageType, "messageSize", len(message))
	ws.mu.RLock()
	defer ws.mu.RUnlock()

	if ws.conn == nil {
		shared.Pulse.Logger.Error("No WebSocket connection available for send")
		return fmt.Errorf("no WebSocket connection available")
	}

	err := ws.conn.WriteMessage(messageType, message)
	if err != nil {
		shared.Pulse.Logger.Error("Failed to send WebSocket message", "error", err, "messageType", messageType, "size", len(message))
		return fmt.Errorf("failed to send message: %w", err)
	}
	shared.Pulse.Logger.Debug("WebSocket message sent successfully", "messageType", messageType, "size", len(message))
	return nil
}

// Receive reads the next WebSocket message (one frame). It blocks until a message
// is available or the connection is closed. Returns the frame type (e.g. TextMessage),
// the payload, and an error on read failure or if the connection is closed.
func (ws *WebSocketClient) Receive() (messageType int, message []byte, err error) {
	ws.mu.RLock()
	conn := ws.conn
	ws.mu.RUnlock()

	if conn == nil {
		return 0, nil, fmt.Errorf("no WebSocket connection available")
	}

	messageType, message, err = conn.ReadMessage()
	if err != nil {
		shared.Pulse.Logger.Error("Failed to read WebSocket message", "error", err)
		return 0, nil, fmt.Errorf("failed to read message: %w", err)
	}

	shared.Pulse.Logger.Debug("Received WebSocket message", "messageLength", len(message))
	return messageType, message, nil
}

// RetrySend sends a message with up to maxRetries attempts. Between attempts it
// sleeps 2 seconds. Returns nil on first success, or an error after all retries fail.
func (ws *WebSocketClient) RetrySend(messageType int, message []byte, maxRetries int) error {
	var err error

	for i := 0; i < maxRetries; i++ {
		err = ws.Send(messageType, message)
		if err == nil {
			return nil
		}
		shared.Pulse.Logger.Debug("Send failed, retrying WebSocket message", "attempt", i+1, "maxRetries", maxRetries, "error", err)
		time.Sleep(2 * time.Second)
	}

	return fmt.Errorf("failed to send message after %d retries: %w", maxRetries, err)
}

// Listen runs a loop that reads messages and calls handleMessage for each. It
// returns a channel that receives a single error when the loop stops (context
// canceled, connection closed, or read error; if auto-reconnect is enabled and
// reconnection fails, that error is sent). The channel is closed after the error
// is sent. Listen is typically run in a goroutine; cancel ctx or close the client
// to stop it.
func (ws *WebSocketClient) Listen(ctx context.Context, handleMessage func(messageType int, message []byte)) <-chan error {
	errChan := make(chan error, 1)

	go func() {
		defer close(errChan)

		for {
			select {
			case <-ctx.Done():
				shared.Pulse.Logger.Debug("Listen context canceled")
				errChan <- ctx.Err()
				return
			case <-ws.ctx.Done():
				shared.Pulse.Logger.Debug("WebSocket context canceled")
				errChan <- ws.ctx.Err()
				return
			default:
				messageType, message, err := ws.Receive()
				if err != nil {
					shared.Pulse.Logger.Error("Error receiving WebSocket message", "error", err)

					// Attempt auto-reconnection if enabled
					if ws.autoReconnect {
						shared.Pulse.Logger.Debug("Attempting auto-reconnection", "delay", ws.reconnectDelay)
						time.Sleep(ws.reconnectDelay)
						if reconnectErr := ws.Reconnect(); reconnectErr != nil {
							shared.Pulse.Logger.Error("Auto-reconnection failed", "error", reconnectErr)
							errChan <- fmt.Errorf("reconnection failed: %w", reconnectErr)
							return
						}
						shared.Pulse.Logger.Debug("Auto-reconnection successful", "host", ws.URL.Host)
						continue
					}

					errChan <- err
					return
				}
				handleMessage(messageType, message)
			}
		}
	}()

	return errChan
}
