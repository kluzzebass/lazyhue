package hue

import (
	"bytes"
	"io"
	"log/slog"
	"net/http"
	"time"
)

// LoggingTransport wraps an http.RoundTripper and logs requests and responses.
type LoggingTransport struct {
	Transport http.RoundTripper
}

// RoundTrip implements http.RoundTripper.
func (t *LoggingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	start := time.Now()

	// Log request
	attrs := []any{
		"method", req.Method,
		"url", req.URL.String(),
	}

	// Capture request body if present
	var reqBody []byte
	if req.Body != nil && req.Body != http.NoBody {
		reqBody, _ = io.ReadAll(req.Body)
		req.Body = io.NopCloser(bytes.NewBuffer(reqBody))
		if len(reqBody) > 0 {
			attrs = append(attrs, "requestBody", string(reqBody))
		}
	}

	slog.Debug("hue api request", attrs...)

	// Perform the request
	transport := t.Transport
	if transport == nil {
		transport = http.DefaultTransport
	}
	resp, err := transport.RoundTrip(req)

	duration := time.Since(start)

	if err != nil {
		slog.Debug("hue api error",
			"method", req.Method,
			"url", req.URL.String(),
			"duration", duration,
			"error", err,
		)
		return resp, err
	}

	// Log response
	respAttrs := []any{
		"method", req.Method,
		"url", req.URL.String(),
		"status", resp.StatusCode,
		"duration", duration,
	}

	// Capture response body for non-2xx responses or small responses
	if resp.Body != nil && resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(resp.Body)
		resp.Body = io.NopCloser(bytes.NewBuffer(respBody))
		if len(respBody) > 0 {
			respAttrs = append(respAttrs, "responseBody", string(respBody))
		}
	}

	slog.Debug("hue api response", respAttrs...)

	return resp, nil
}
