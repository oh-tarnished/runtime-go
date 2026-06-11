// HTTP client implementation.
//
// This file provides the HTTP client used when ClientType is HTTPConnClient.
// It supports GET, POST, PUT, PATCH, DELETE with retries, context cancellation,
// and optional connectivity verification (HEAD request, with GET fallback on 405).
package network

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/oh-tarnished/runtime-go/network/shared"
)

// maxConnectivityResponseBodyBytes is the maximum number of bytes read from the
// connectivity-check response body when draining it (after GET fallback). Limits
// exposure to buggy or malicious servers that send huge bodies.
const maxConnectivityResponseBodyBytes = 1 << 20 // 1 MiB

// HTTPClient is an HTTP REST client. It embeds ConnectionOptions and provides
// Connect, Close, Reconnect, Request (async with retries), and RequestSync.
// Create via NewConnection(HTTPConnClient) and AsHTTPConnectionType.
type HTTPClient struct {
	client            *http.Client
	ConnectionOptions // URL, Timeout, Headers, Retries, RetryDelay, SkipConnectivityCheck
}

// HTTPMethod is the HTTP verb for a request.
type HTTPMethod string

const (
	GET    HTTPMethod = "GET"
	POST   HTTPMethod = "POST"
	PUT    HTTPMethod = "PUT"
	PATCH  HTTPMethod = "PATCH"
	DELETE HTTPMethod = "DELETE"
)

// Connect configures the HTTP client and optionally verifies the server is
// reachable. If opts.Timeout <= 0, DefaultTimeout is used. If SkipConnectivityCheck
// is true, no request is sent. Otherwise a HEAD request is sent to the configured
// URL; if the server returns 405 Method Not Allowed, a GET is sent instead and the
// response body is drained (up to 1 MiB) so the connection can be reused. Connect
// returns an error if the reachability check fails.
func (h *HTTPClient) Connect(opts ConnectionOptions) error {
	if opts.Timeout <= 0 {
		opts.Timeout = DefaultTimeout
		shared.Pulse.Logger.Debug("HTTP Connect using default timeout", "timeout", opts.Timeout)
	}
	shared.Pulse.Logger.Debug("HTTP Connect called", "host", opts.URL.Host, "scheme", opts.URL.Scheme, "timeout", opts.Timeout)

	// Assign options, including timeout
	h.ConnectionOptions = opts

	// Initialize the HTTP client with the given timeout
	h.client = &http.Client{
		Timeout: opts.Timeout,
	}
	shared.Pulse.Logger.Debug("HTTP client initialized", "timeout", opts.Timeout)

	// Validate and build the full URL
	fullURL, err := buildFullURL(opts.URL, 0)
	if err != nil {
		shared.Pulse.Logger.Error("Failed to build HTTP URL", "error", err, "urlOptions", opts.URL)
		return fmt.Errorf("failed to build HTTP URL: %w", err)
	}
	shared.Pulse.Logger.Debug("HTTP URL validated", "fullURL", fullURL)

	// Check the scheme of the URL
	if opts.URL.Scheme != "http" && opts.URL.Scheme != "https" {
		shared.Pulse.Logger.Error("Invalid URL scheme", "scheme", opts.URL.Scheme)
		return fmt.Errorf("invalid URL scheme: %s. Must be 'http' or 'https'", opts.URL.Scheme)
	}

	if opts.SkipConnectivityCheck {
		shared.Pulse.Logger.Infof("HTTP client configured (connectivity check skipped) url=%s host=%s", fullURL, opts.URL.Host)
		return nil
	}

	// Verify connection to host:port before reporting connected
	connectCtx, connectCancel := context.WithTimeout(context.Background(), opts.Timeout)
	defer connectCancel()
	req, err := http.NewRequestWithContext(connectCtx, http.MethodHead, fullURL, nil)
	if err != nil {
		shared.Pulse.Logger.Error("Failed to create HTTP connectivity request", "error", err, "url", fullURL)
		return fmt.Errorf("failed to create connectivity request: %w", err)
	}
	resp, err := h.client.Do(req)
	if err == nil && resp != nil && resp.StatusCode == http.StatusMethodNotAllowed {
		// Server doesn't support HEAD; try GET (then discard body)
		if resp.Body != nil {
			_ = resp.Body.Close()
		}
		req, _ = http.NewRequestWithContext(connectCtx, http.MethodGet, fullURL, nil)
		resp, err = h.client.Do(req)
	}
	if err != nil {
		shared.Pulse.Logger.Error("Failed to connect to HTTP server at host", "error", err, "url", fullURL, "host", opts.URL.Host)
		return fmt.Errorf("failed to connect to HTTP server at %s: %w", opts.URL.Host, err)
	}
	if resp != nil && resp.Body != nil {
		_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, maxConnectivityResponseBodyBytes))
		_ = resp.Body.Close()
	}
	shared.Pulse.Logger.Infof("Connected to HTTP server successfully url=%s host=%s headers=%d", fullURL, opts.URL.Host, len(opts.Headers))
	return nil
}

// Close clears the HTTP client and releases resources. The client is not usable
// until Connect is called again.
func (h *HTTPClient) Close() error {
	host := h.URL.Host
	h.client = nil
	shared.Pulse.Logger.Debug("HTTP connection closed", "host", host)
	return nil
}

// Reconnect re-applies the current ConnectionOptions (calls Connect again).
// Useful after a long idle or when the server was temporarily unavailable.
func (h *HTTPClient) Reconnect() error {
	shared.Pulse.Logger.Debug("Reconnecting to HTTP server...")
	return h.Connect(h.ConnectionOptions)
}

// buildRequest allocates an HTTP request with the given method, URL, body, and headers.
func (h *HTTPClient) buildRequest(ctx context.Context, method HTTPMethod, fullURL string, body []byte, headers map[string]string) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, string(method), fullURL, bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	// Set headers
	for key, value := range headers {
		req.Header.Set(key, value)
	}
	return req, nil
}

// validateStatusCode maps HTTP status codes to errors. 2xx returns nil; 4xx/5xx
// return descriptive errors (e.g. 400 Bad Request, 404 Not Found, 500 server error).
func validateStatusCode(statusCode int) error {
	switch {
	case statusCode >= 200 && statusCode < 300:
		return nil
	case statusCode == 400:
		return fmt.Errorf("client error: status code %d (Bad Request)", statusCode)
	case statusCode == 401:
		return fmt.Errorf("client error: status code %d (Unauthorized)", statusCode)
	case statusCode == 403:
		return fmt.Errorf("client error: status code %d (Forbidden)", statusCode)
	case statusCode == 404:
		return fmt.Errorf("client error: status code %d (Not Found)", statusCode)
	case statusCode == 429:
		return fmt.Errorf("client error: status code %d (Too Many Requests)", statusCode)
	case statusCode >= 500 && statusCode < 600:
		return fmt.Errorf("server error: status code %d", statusCode)
	default:
		return fmt.Errorf("unexpected status code: %d", statusCode)
	}
}

// prettyPrintJSON unmarshals data as JSON and re-encodes it with indentation.
func prettyPrintJSON(data []byte) (string, error) {
	var prettyJSON bytes.Buffer
	if err := json.Indent(&prettyJSON, data, "", "  "); err != nil {
		return "", fmt.Errorf("error formatting JSON: %w", err)
	}
	return prettyJSON.String(), nil
}

// sendRequest performs the HTTP request, reads the body, and validates the status
// code. Returns the response body and an error for non-2xx status codes.
func (h *HTTPClient) sendRequest(req *http.Request) ([]byte, error) {
	resp, err := h.client.Do(req)
	if err != nil {
		shared.Pulse.Logger.Error("Error making request", "error", err)
		return nil, fmt.Errorf("error making request: %w", err)
	}
	defer resp.Body.Close()

	// Read the response body
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Validate JSON if response is JSON
	contentType := resp.Header.Get("Content-Type")
	if strings.Contains(contentType, "application/json") {
		if _, err := prettyPrintJSON(data); err != nil {
			shared.Pulse.Logger.Debug("Response contains invalid JSON", "error", err)
		}
	}

	// Validate the status code (this will return an error if it's a client/server error)
	if err := validateStatusCode(resp.StatusCode); err != nil {
		shared.Pulse.Logger.Error("Request failed", "error", err, "statusCode", resp.StatusCode)
		// Even though there's an error, return the response body for logging
		return data, err
	}

	shared.Pulse.Logger.Debug("Request successful", "statusCode", resp.StatusCode, "responseLength", len(data))
	return data, nil
}

// HTTPResponse holds the result of an HTTP request: raw body bytes and an error
// (from network failure or non-2xx status). Exactly one of Data or Error is
// typically set; on retry exhaustion both may be set (Error describes the failure).
type HTTPResponse struct {
	Data  []byte // Response body on success
	Error error  // Non-nil on failure or non-2xx
}

// Request performs an HTTP request asynchronously with optional retries. It builds
// the URL from urlOptions and pathIndex, sends the request, and retries up to
// maxRetries times with RetryDelay between attempts. The returned channel receives
// exactly one HTTPResponse and is then closed. Context cancellation aborts the
// request and any retries.
func (h *HTTPClient) Request(ctx context.Context, method HTTPMethod, urlOptions URLOptions, body []byte, headers map[string]string, pathIndex int, maxRetries int) <-chan HTTPResponse {
	shared.Pulse.Logger.Debug("HTTP Request initiated", "method", method, "host", urlOptions.Host, "pathIndex", pathIndex, "maxRetries", maxRetries, "bodySize", len(body))
	resultChan := make(chan HTTPResponse, 1)

	go func() {
		defer close(resultChan)

		// Determine retry delay
		retryDelay := h.RetryDelay
		if retryDelay == 0 {
			retryDelay = 2 * time.Second // Default retry delay
		}
		shared.Pulse.Logger.Debug("HTTP retry configuration", "retryDelay", retryDelay, "maxRetries", maxRetries)

		var lastErr error
		for attempt := 0; attempt <= maxRetries; attempt++ {
			if attempt > 0 {
				shared.Pulse.Logger.Debug("Retrying HTTP request", "attempt", attempt, "maxRetries", maxRetries)
				// Wait before retry, respecting context cancellation
				select {
				case <-time.After(retryDelay):
				case <-ctx.Done():
					resultChan <- HTTPResponse{Error: ctx.Err()}
					return
				}
			}

			// Create a context with timeout for this specific request
			requestCtx, cancel := context.WithTimeout(ctx, h.Timeout)

			// Build the full URL for the request
			fullURL, err := buildFullURL(urlOptions, pathIndex)
			if err != nil {
				cancel()
				shared.Pulse.Logger.Error("Error building full URL", "error", err, "pathIndex", pathIndex)
				resultChan <- HTTPResponse{Error: fmt.Errorf("failed to build URL: %w", err)}
				return
			}
			shared.Pulse.Logger.Debug("HTTP request URL built", "fullURL", fullURL, "attempt", attempt+1)

			// Build the actual HTTP request
			req, err := h.buildRequest(requestCtx, method, fullURL, body, headers)
			if err != nil {
				cancel()
				shared.Pulse.Logger.Error("Error building request", "error", err, "method", method, "url", fullURL)
				resultChan <- HTTPResponse{Error: fmt.Errorf("failed to build request: %w", err)}
				return
			}
			shared.Pulse.Logger.Debug("HTTP request object created", "method", method, "headers", len(req.Header))

			// Send the HTTP request and capture the response
			data, err := h.sendRequest(req)
			cancel()

			if err == nil {
				// Success
				resultChan <- HTTPResponse{Data: data}
				return
			}

			lastErr = err
			shared.Pulse.Logger.Error("Error sending request", "error", err, "attempt", attempt+1, "retriesLeft", maxRetries-attempt)
		}

		// All retries exhausted
		shared.Pulse.Logger.Debug("Max retries reached. Failing request.", "maxRetries", maxRetries)
		resultChan <- HTTPResponse{Error: fmt.Errorf("request failed after %d retries: %w", maxRetries, lastErr)}
	}()

	return resultChan
}

// RequestSync is a blocking version of Request. It waits for the single response
// and returns the body and error. Convenient when callers do not need a channel.
func (h *HTTPClient) RequestSync(ctx context.Context, method HTTPMethod, urlOptions URLOptions, body []byte, headers map[string]string, pathIndex int, maxRetries int) ([]byte, error) {
	response := <-h.Request(ctx, method, urlOptions, body, headers, pathIndex, maxRetries)
	return response.Data, response.Error
}
