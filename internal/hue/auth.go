package hue

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"time"
)

// ErrPairingCancelled indicates that pairing was cancelled by the user.
var ErrPairingCancelled = errors.New("pairing cancelled")

var (
	// ErrLinkButtonNotPressed indicates the user hasn't pressed the bridge button.
	ErrLinkButtonNotPressed = errors.New("link button not pressed")
	// ErrAuthFailed indicates authentication failed for a non-retryable reason.
	ErrAuthFailed = errors.New("authentication failed")
)

// AuthResult contains the result of an authentication attempt.
type AuthResult struct {
	ApiKey string
	Retry  bool
	Err    error
}

// Authenticator handles the bridge pairing flow.
type Authenticator struct {
	bridgeIP   string
	deviceType string
}

// NewAuthenticator creates an authenticator for the given bridge IP.
func NewAuthenticator(bridgeIP string) (*Authenticator, error) {
	// Build device type as "lazyhue#hostname" (Hue convention)
	deviceType := "lazyhue"
	if hostname, err := os.Hostname(); err == nil && hostname != "" {
		deviceType = "lazyhue#" + hostname
	}

	return &Authenticator{
		bridgeIP:   bridgeIP,
		deviceType: deviceType,
	}, nil
}

// TryAuthenticate attempts to authenticate once.
// Returns the API key on success, or an error indicating whether to retry.
// NOTE: The authentication endpoint is a v1 API, not part of CLIP v2.
// This uses direct HTTP calls instead of the generated client.
func (a *Authenticator) TryAuthenticate() AuthResult {
	// Build request body
	body := map[string]interface{}{
		"devicetype":        a.deviceType,
		"generateclientkey": true,
	}

	// Create HTTP client with TLS skip (bridge uses self-signed cert)
	httpClient := &http.Client{
		Timeout: 10 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}

	// Marshal body
	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return AuthResult{Retry: false, Err: err}
	}

	// Create request to v1 API endpoint
	// The authentication endpoint is at /api, not /clip/v2
	serverURL := "https://" + a.bridgeIP + "/api"
	req, err := http.NewRequestWithContext(context.Background(), "POST", serverURL, bytes.NewReader(bodyBytes))
	if err != nil {
		return AuthResult{Retry: false, Err: err}
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil {
		return AuthResult{Retry: true, Err: err}
	}
	defer resp.Body.Close()

	// Read response
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return AuthResult{Retry: true, Err: err}
	}

	// Parse response
	var responses []struct {
		Success *struct {
			Username string `json:"username"`
		} `json:"success,omitempty"`
		Error *struct {
			Type        int    `json:"type"`
			Description string `json:"description"`
		} `json:"error,omitempty"`
	}

	if err := json.Unmarshal(respBody, &responses); err != nil {
		return AuthResult{Retry: true, Err: errors.New("failed to parse response: " + err.Error())}
	}

	if len(responses) == 0 {
		return AuthResult{Retry: true, Err: errors.New("empty response from bridge")}
	}

	response := responses[0]

	// Check for success
	if response.Success != nil && response.Success.Username != "" {
		return AuthResult{ApiKey: response.Success.Username, Retry: false, Err: nil}
	}

	// Check for error
	if response.Error != nil {
		// Error type 101 = link button not pressed
		if response.Error.Type == 101 {
			return AuthResult{Retry: true, Err: ErrLinkButtonNotPressed}
		}
		return AuthResult{Retry: true, Err: errors.New(response.Error.Description)}
	}

	return AuthResult{Retry: true, Err: errors.New("unexpected response format")}
}

// AuthenticateWithPolling polls for authentication until success or timeout.
func (a *Authenticator) AuthenticateWithPolling(timeout time.Duration, pollInterval time.Duration) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	return a.AuthenticateWithContext(ctx, pollInterval)
}

// AuthenticateWithContext polls for authentication until success, context cancellation, or timeout.
func (a *Authenticator) AuthenticateWithContext(ctx context.Context, pollInterval time.Duration) (string, error) {
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	// Try immediately first
	result := a.TryAuthenticate()
	if result.Err == nil {
		return result.ApiKey, nil
	}

	for {
		select {
		case <-ctx.Done():
			if ctx.Err() == context.Canceled {
				return "", ErrPairingCancelled
			}
			return "", errors.New("authentication timed out waiting for link button")
		case <-ticker.C:
			result := a.TryAuthenticate()
			if result.Err == nil {
				return result.ApiKey, nil
			}
			if !result.Retry {
				return "", result.Err
			}
		}
	}
}
