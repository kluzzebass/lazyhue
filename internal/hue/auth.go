package hue

import (
	"errors"
	"time"

	"github.com/openhue/openhue-go"
)

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
	
	if err != nil && retry {
		return AuthResult{Retry: true, Err: ErrLinkButtonNotPressed}
	}
	if err != nil {
		return AuthResult{Retry: false, Err: err}
	}
	
	return AuthResult{ApiKey: apiKey, Retry: false, Err: nil}
}

// AuthenticateWithPolling polls for authentication until success or timeout.
func (a *Authenticator) AuthenticateWithPolling(timeout time.Duration, pollInterval time.Duration) (string, error) {
	deadline := time.Now().Add(timeout)
	
	for time.Now().Before(deadline) {
		result := a.TryAuthenticate()
		
		if result.Err == nil {
			return result.ApiKey, nil
		}
		
		if !result.Retry {
			return "", result.Err
		}
		
		time.Sleep(pollInterval)
	}
	
	return "", errors.New("authentication timed out waiting for link button")
}

