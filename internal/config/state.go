package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

const stateFileName = "state.json"

// BridgeUIState stores UI state for a single bridge.
type BridgeUIState struct {
	// NodeStates maps node paths to their expanded state (true = expanded, false = collapsed)
	NodeStates map[string]bool `json:"node_states"`
}

// UIStateStore manages UI state for multiple bridges.
type UIStateStore struct {
	Bridges              map[string]BridgeUIState `json:"bridges"`
	LastSelectedBridgeID string                   `json:"last_selected_bridge_id,omitempty"`
}

// NewUIStateStore creates an empty UI state store.
func NewUIStateStore() *UIStateStore {
	return &UIStateStore{
		Bridges: make(map[string]BridgeUIState),
	}
}

// LoadUIState reads the state file, returning empty store if not found.
func LoadUIState() (*UIStateStore, error) {
	dir, err := configDir()
	if err != nil {
		return NewUIStateStore(), nil
	}

	path := filepath.Join(dir, stateFileName)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return NewUIStateStore(), nil
		}
		return nil, err
	}

	var store UIStateStore
	if err := json.Unmarshal(data, &store); err != nil {
		return nil, err
	}

	if store.Bridges == nil {
		store.Bridges = make(map[string]BridgeUIState)
	}
	return &store, nil
}

// Save writes the state to disk.
func (s *UIStateStore) Save() error {
	dir, err := ensureConfigDir()
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}

	path := filepath.Join(dir, stateFileName)
	return os.WriteFile(path, data, 0644)
}

// GetNodeStates returns the node states for a bridge (path -> expanded).
func (s *UIStateStore) GetNodeStates(bridgeID string) map[string]bool {
	if state, ok := s.Bridges[bridgeID]; ok {
		return state.NodeStates
	}
	return nil
}

// SetNodeStates stores the node states for a bridge.
func (s *UIStateStore) SetNodeStates(bridgeID string, states map[string]bool) {
	s.Bridges[bridgeID] = BridgeUIState{NodeStates: states}
}

// Delete removes state for a bridge.
func (s *UIStateStore) Delete(bridgeID string) {
	delete(s.Bridges, bridgeID)
}

