package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

const credentialsFileName = "credentials.json"

// BridgeCredential stores authentication info for a single bridge.
type BridgeCredential struct {
	BridgeID   string `json:"bridge_id"`
	BridgeName string `json:"bridge_name"`
	IPAddress  string `json:"ip_address"`
	ApiKey     string `json:"api_key"`
	LastUsed   string `json:"last_used"`
}

// CredentialStore manages credentials for multiple bridges.
type CredentialStore struct {
	Bridges map[string]BridgeCredential `json:"bridges"`
}

// NewCredentialStore creates an empty credential store.
func NewCredentialStore() *CredentialStore {
	return &CredentialStore{
		Bridges: make(map[string]BridgeCredential),
	}
}

// LoadCredentials reads the credentials file, returning empty store if not found.
func LoadCredentials() (*CredentialStore, error) {
	dir, err := configDir()
	if err != nil {
		return NewCredentialStore(), nil
	}

	path := filepath.Join(dir, credentialsFileName)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return NewCredentialStore(), nil
		}
		return nil, err
	}

	var store CredentialStore
	if err := json.Unmarshal(data, &store); err != nil {
		return nil, err
	}

	if store.Bridges == nil {
		store.Bridges = make(map[string]BridgeCredential)
	}
	return &store, nil
}

// Save writes the credentials to disk with restricted permissions.
func (s *CredentialStore) Save() error {
	dir, err := ensureConfigDir()
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}

	path := filepath.Join(dir, credentialsFileName)
	return os.WriteFile(path, data, 0600)
}

// Get retrieves credentials for a bridge by ID.
func (s *CredentialStore) Get(bridgeID string) (BridgeCredential, bool) {
	cred, ok := s.Bridges[bridgeID]
	return cred, ok
}

// Set stores credentials for a bridge.
func (s *CredentialStore) Set(cred BridgeCredential) {
	cred.LastUsed = time.Now().Format(time.RFC3339)
	s.Bridges[cred.BridgeID] = cred
}

// Delete removes credentials for a bridge.
func (s *CredentialStore) Delete(bridgeID string) {
	delete(s.Bridges, bridgeID)
}

// UpdateLastUsed updates the last used timestamp for a bridge.
func (s *CredentialStore) UpdateLastUsed(bridgeID string) {
	if cred, ok := s.Bridges[bridgeID]; ok {
		cred.LastUsed = time.Now().Format(time.RFC3339)
		s.Bridges[bridgeID] = cred
	}
}

