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
	NodeStates       map[string]bool `json:"node_states"`
	ActiveTab        int             `json:"active_tab"`         // Which tab is selected (0=Home, 1=Lights, etc.)
	SelectedEntityID string          `json:"selected_entity_id"` // ID of selected entity
}

// UIStateStore manages UI state for multiple bridges.
type UIStateStore struct {
	Bridges              map[string]BridgeUIState `json:"bridges"`
	LastSelectedBridgeID string                   `json:"last_selected_bridge_id,omitempty"`
	FocusedPanelIndex    int                      `json:"focused_panel_index"` // Which panel is focused
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
	existing := s.Bridges[bridgeID]
	existing.NodeStates = states
	s.Bridges[bridgeID] = existing
}

// GetActiveTab returns the active tab for a bridge.
func (s *UIStateStore) GetActiveTab(bridgeID string) int {
	if state, ok := s.Bridges[bridgeID]; ok {
		return state.ActiveTab
	}
	return 0
}

// SetActiveTab stores the active tab for a bridge.
func (s *UIStateStore) SetActiveTab(bridgeID string, tab int) {
	existing := s.Bridges[bridgeID]
	existing.ActiveTab = tab
	s.Bridges[bridgeID] = existing
}

// GetSelectedEntityID returns the selected entity ID for a bridge.
func (s *UIStateStore) GetSelectedEntityID(bridgeID string) string {
	if state, ok := s.Bridges[bridgeID]; ok {
		return state.SelectedEntityID
	}
	return ""
}

// SetSelectedEntityID stores the selected entity ID for a bridge.
func (s *UIStateStore) SetSelectedEntityID(bridgeID string, entityID string) {
	existing := s.Bridges[bridgeID]
	existing.SelectedEntityID = entityID
	s.Bridges[bridgeID] = existing
}

// Delete removes state for a bridge.
func (s *UIStateStore) Delete(bridgeID string) {
	delete(s.Bridges, bridgeID)
}

