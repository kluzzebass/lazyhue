package hue

import (
	"context"
	"crypto/tls"
	"errors"
	"net/http"
	"os"
	"time"

	"github.com/kluzzebass/lazyhue/internal/hueclient"
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
	client     *hueclient.ClientWithResponses
	deviceType string
}

// NewAuthenticator creates an authenticator for the given bridge IP.
func NewAuthenticator(bridgeIP string) (*Authenticator, error) {
	// Build device type as "lazyhue#hostname" (Hue convention)
	deviceType := "lazyhue"
	if hostname, err := os.Hostname(); err == nil && hostname != "" {
		deviceType = "lazyhue#" + hostname
	}

	// Create HTTP client with TLS skip (bridge uses self-signed cert)
	httpClient := &http.Client{
		Timeout: 10 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}

	client, err := hueclient.NewClientWithResponses(
		"https://"+bridgeIP,
		hueclient.WithHTTPClient(httpClient),
	)
	if err != nil {
		return nil, err
	}

	return &Authenticator{
		client:     client,
		deviceType: deviceType,
	}, nil
}

// TryAuthenticate attempts to authenticate once.
// Returns the API key on success, or an error indicating whether to retry.
func (a *Authenticator) TryAuthenticate() AuthResult {
	generateClientKey := true
	body := hueclient.AuthenticateJSONRequestBody{
		Devicetype:        &a.deviceType,
		Generateclientkey: &generateClientKey,
	}

	resp, err := a.client.AuthenticateWithResponse(context.Background(), body)
	if err != nil {
		// Network errors are retryable
		return AuthResult{Retry: true, Err: err}
	}

	// Check for HTTP-level errors
	if resp.StatusCode() != http.StatusOK {
		if resp.JSON401 != nil {
			return AuthResult{Retry: false, Err: ErrAuthFailed}
		}
		return AuthResult{Retry: true, Err: errors.New("unexpected HTTP status: " + resp.Status())}
	}

	// Parse the response
	if resp.JSON200 == nil || len(*resp.JSON200) == 0 {
		return AuthResult{Retry: true, Err: errors.New("empty response from bridge")}
	}

	response := (*resp.JSON200)[0]

	// Check for success
	if response.Success != nil && response.Success.Username != nil && *response.Success.Username != "" {
		return AuthResult{ApiKey: *response.Success.Username, Retry: false, Err: nil}
	}

	// Check for error
	if response.Error != nil {
		// Error type 101 = link button not pressed
		if response.Error.Type != nil && *response.Error.Type == 101 {
			return AuthResult{Retry: true, Err: ErrLinkButtonNotPressed}
		}
		desc := "unknown error"
		if response.Error.Description != nil {
			desc = *response.Error.Description
		}
		errType := 0
		if response.Error.Type != nil {
			errType = *response.Error.Type
		}
		return AuthResult{Retry: true, Err: errors.New(desc + " (type " + string(rune(errType+'0')) + ")")}
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
