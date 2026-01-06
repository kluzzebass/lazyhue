package hue

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/openhue/openhue-go"
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
	bridgeIP string
	auth     openhue.Authenticator
}

// NewAuthenticator creates an authenticator for the given bridge IP.
func NewAuthenticator(bridgeIP string) (*Authenticator, error) {
	auth, err := openhue.NewAuthenticator(bridgeIP)
	if err != nil {
		return nil, err
	}
	return &Authenticator{
		bridgeIP: bridgeIP,
		auth:     auth,
	}, nil
}

// TryAuthenticate attempts to authenticate once.
// Returns the API key on success, or an error indicating whether to retry.
func (a *Authenticator) TryAuthenticate() AuthResult {
	apiKey, retry, err := a.auth.Authenticate()
	
	if err == nil {
		return AuthResult{ApiKey: apiKey, Retry: false, Err: nil}
	}
	
	// If openhue says to retry, or if it's a transient error, keep trying
	// Most errors during pairing are transient (network, TLS, etc.)
	if retry {
		return AuthResult{Retry: true, Err: ErrLinkButtonNotPressed}
	}
	
	// Check if error message indicates link button not pressed
	errStr := err.Error()
	if strings.Contains(errStr, "link button") || strings.Contains(errStr, "not pressed") {
		return AuthResult{Retry: true, Err: ErrLinkButtonNotPressed}
	}
	
	// For other errors, still retry but pass the actual error
	// Only stop on explicit auth failures
	return AuthResult{Retry: true, Err: err}
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

