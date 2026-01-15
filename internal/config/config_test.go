package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	if cfg == nil {
		t.Fatal("DefaultConfig() returned nil")
	}
	if cfg.PollInterval != "2s" {
		t.Errorf("DefaultConfig().PollInterval = %q, want %q", cfg.PollInterval, "2s")
	}
	if cfg.DefaultBridge != "" {
		t.Errorf("DefaultConfig().DefaultBridge = %q, want empty", cfg.DefaultBridge)
	}
}

func TestNewCredentialStore(t *testing.T) {
	store := NewCredentialStore()
	if store == nil {
		t.Fatal("NewCredentialStore() returned nil")
	}
	if store.Bridges == nil {
		t.Error("NewCredentialStore().Bridges is nil")
	}
	if len(store.Bridges) != 0 {
		t.Errorf("NewCredentialStore().Bridges has %d entries, want 0", len(store.Bridges))
	}
}

func TestCredentialStore_GetSetDelete(t *testing.T) {
	store := NewCredentialStore()

	// Test Get on empty store
	_, ok := store.Get("nonexistent")
	if ok {
		t.Error("Get on empty store should return false")
	}

	// Test Set
	cred := BridgeCredential{
		BridgeID:  "bridge-1",
		Name:      "Test Bridge",
		IPAddress: "192.168.1.100",
		ApiKey:    "test-api-key",
	}
	store.Set(cred)

	// Test Get after Set
	got, ok := store.Get("bridge-1")
	if !ok {
		t.Error("Get after Set should return true")
	}
	if got.BridgeID != cred.BridgeID {
		t.Errorf("Got BridgeID = %q, want %q", got.BridgeID, cred.BridgeID)
	}
	if got.Name != cred.Name {
		t.Errorf("Got Name = %q, want %q", got.Name, cred.Name)
	}
	if got.IPAddress != cred.IPAddress {
		t.Errorf("Got IPAddress = %q, want %q", got.IPAddress, cred.IPAddress)
	}
	if got.ApiKey != cred.ApiKey {
		t.Errorf("Got ApiKey = %q, want %q", got.ApiKey, cred.ApiKey)
	}
	// LastUsed should be set
	if got.LastUsed == "" {
		t.Error("Set should populate LastUsed")
	}

	// Test Delete
	store.Delete("bridge-1")
	_, ok = store.Get("bridge-1")
	if ok {
		t.Error("Get after Delete should return false")
	}
}

func TestCredentialStore_UpdateLastUsed(t *testing.T) {
	store := NewCredentialStore()

	// UpdateLastUsed on nonexistent bridge should not panic
	store.UpdateLastUsed("nonexistent")

	// Add a bridge with an old timestamp
	cred := BridgeCredential{
		BridgeID: "bridge-1",
		Name:     "Test Bridge",
		LastUsed: "2020-01-01T00:00:00Z",
	}
	store.Bridges[cred.BridgeID] = cred // Direct assignment to avoid Set() updating timestamp

	// Update last used
	store.UpdateLastUsed("bridge-1")

	got, _ := store.Get("bridge-1")
	// Should have a more recent timestamp
	if got.LastUsed == "2020-01-01T00:00:00Z" {
		t.Error("UpdateLastUsed should update the timestamp")
	}
	// Verify it's a valid RFC3339 timestamp
	_, err := time.Parse(time.RFC3339, got.LastUsed)
	if err != nil {
		t.Errorf("UpdateLastUsed produced invalid timestamp: %v", err)
	}
}

func TestNewUIStateStore(t *testing.T) {
	store := NewUIStateStore()
	if store == nil {
		t.Fatal("NewUIStateStore() returned nil")
	}
	if store.Bridges == nil {
		t.Error("NewUIStateStore().Bridges is nil")
	}
	if len(store.Bridges) != 0 {
		t.Errorf("NewUIStateStore().Bridges has %d entries, want 0", len(store.Bridges))
	}
}

func TestUIStateStore_NodeStates(t *testing.T) {
	store := NewUIStateStore()

	// Get from empty store
	states := store.GetNodeStates("bridge-1")
	if states != nil {
		t.Error("GetNodeStates from empty store should return nil")
	}

	// Set node states
	nodeStates := map[string]bool{
		"rooms/room-1":  true,
		"lights/light-1": false,
	}
	store.SetNodeStates("bridge-1", nodeStates)

	// Get node states
	got := store.GetNodeStates("bridge-1")
	if got == nil {
		t.Fatal("GetNodeStates after Set returned nil")
	}
	if got["rooms/room-1"] != true {
		t.Error("Expected rooms/room-1 to be true")
	}
	if got["lights/light-1"] != false {
		t.Error("Expected lights/light-1 to be false")
	}
}

func TestUIStateStore_ActiveTab(t *testing.T) {
	store := NewUIStateStore()

	// Get from empty store (default 0)
	tab := store.GetActiveTab("bridge-1")
	if tab != 0 {
		t.Errorf("GetActiveTab from empty store = %d, want 0", tab)
	}

	// Set active tab
	store.SetActiveTab("bridge-1", 2)

	// Get active tab
	got := store.GetActiveTab("bridge-1")
	if got != 2 {
		t.Errorf("GetActiveTab after Set = %d, want 2", got)
	}
}

func TestUIStateStore_SelectedEntityID(t *testing.T) {
	store := NewUIStateStore()

	// Get from empty store (default empty string)
	id := store.GetSelectedEntityID("bridge-1")
	if id != "" {
		t.Errorf("GetSelectedEntityID from empty store = %q, want empty", id)
	}

	// Set selected entity
	store.SetSelectedEntityID("bridge-1", "light-abc-123")

	// Get selected entity
	got := store.GetSelectedEntityID("bridge-1")
	if got != "light-abc-123" {
		t.Errorf("GetSelectedEntityID after Set = %q, want %q", got, "light-abc-123")
	}
}

func TestUIStateStore_Delete(t *testing.T) {
	store := NewUIStateStore()

	// Set some state
	store.SetActiveTab("bridge-1", 3)
	store.SetSelectedEntityID("bridge-1", "entity-1")

	// Delete
	store.Delete("bridge-1")

	// Verify deleted
	if store.GetActiveTab("bridge-1") != 0 {
		t.Error("GetActiveTab after Delete should return default")
	}
	if store.GetSelectedEntityID("bridge-1") != "" {
		t.Error("GetSelectedEntityID after Delete should return empty")
	}
}

// Test JSON serialization/deserialization
func TestConfig_JSON(t *testing.T) {
	cfg := &Config{
		PollInterval:  "5s",
		DefaultBridge: "my-bridge",
	}

	data, err := json.Marshal(cfg)
	if err != nil {
		t.Fatalf("Failed to marshal config: %v", err)
	}

	var decoded Config
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal config: %v", err)
	}

	if decoded.PollInterval != cfg.PollInterval {
		t.Errorf("PollInterval mismatch: got %q, want %q", decoded.PollInterval, cfg.PollInterval)
	}
	if decoded.DefaultBridge != cfg.DefaultBridge {
		t.Errorf("DefaultBridge mismatch: got %q, want %q", decoded.DefaultBridge, cfg.DefaultBridge)
	}
}

func TestCredentialStore_JSON(t *testing.T) {
	store := NewCredentialStore()
	store.Bridges["bridge-1"] = BridgeCredential{
		BridgeID:  "bridge-1",
		Name:      "Test Bridge",
		IPAddress: "192.168.1.100",
		ApiKey:    "secret-key",
		LastUsed:  "2024-01-01T00:00:00Z",
	}

	data, err := json.Marshal(store)
	if err != nil {
		t.Fatalf("Failed to marshal credential store: %v", err)
	}

	var decoded CredentialStore
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal credential store: %v", err)
	}

	if len(decoded.Bridges) != 1 {
		t.Errorf("Expected 1 bridge, got %d", len(decoded.Bridges))
	}
	got, ok := decoded.Bridges["bridge-1"]
	if !ok {
		t.Fatal("bridge-1 not found in decoded store")
	}
	if got.ApiKey != "secret-key" {
		t.Errorf("ApiKey mismatch: got %q, want %q", got.ApiKey, "secret-key")
	}
}

func TestUIStateStore_JSON(t *testing.T) {
	store := NewUIStateStore()
	store.LastSelectedBridgeID = "bridge-1"
	store.FocusedPanelIndex = 2
	store.Bridges["bridge-1"] = BridgeUIState{
		NodeStates:       map[string]bool{"rooms/room-1": true},
		ActiveTab:        1,
		SelectedEntityID: "light-1",
	}

	data, err := json.Marshal(store)
	if err != nil {
		t.Fatalf("Failed to marshal UI state store: %v", err)
	}

	var decoded UIStateStore
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal UI state store: %v", err)
	}

	if decoded.LastSelectedBridgeID != "bridge-1" {
		t.Errorf("LastSelectedBridgeID mismatch: got %q, want %q", decoded.LastSelectedBridgeID, "bridge-1")
	}
	if decoded.FocusedPanelIndex != 2 {
		t.Errorf("FocusedPanelIndex mismatch: got %d, want %d", decoded.FocusedPanelIndex, 2)
	}
	got, ok := decoded.Bridges["bridge-1"]
	if !ok {
		t.Fatal("bridge-1 not found in decoded store")
	}
	if got.ActiveTab != 1 {
		t.Errorf("ActiveTab mismatch: got %d, want %d", got.ActiveTab, 1)
	}
}

// Test file operations with temp directory
func TestConfig_LoadSave(t *testing.T) {
	// Create temp dir
	tmpDir := t.TempDir()

	// Override configDir for test by using a temp HOME
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", origHome)

	// Load should return defaults when file doesn't exist
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if cfg.PollInterval != "2s" {
		t.Errorf("Load() on missing file returned PollInterval = %q, want %q", cfg.PollInterval, "2s")
	}

	// Save config
	cfg.PollInterval = "10s"
	cfg.DefaultBridge = "test-bridge"
	if err := cfg.Save(); err != nil {
		t.Fatalf("Save() error: %v", err)
	}

	// Verify file exists
	configPath := filepath.Join(tmpDir, ".config", "lazyhue", "config.json")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Error("Config file was not created")
	}

	// Load again and verify
	cfg2, err := Load()
	if err != nil {
		t.Fatalf("Load() after Save() error: %v", err)
	}
	if cfg2.PollInterval != "10s" {
		t.Errorf("Loaded PollInterval = %q, want %q", cfg2.PollInterval, "10s")
	}
	if cfg2.DefaultBridge != "test-bridge" {
		t.Errorf("Loaded DefaultBridge = %q, want %q", cfg2.DefaultBridge, "test-bridge")
	}
}

func TestCredentialStore_LoadSave(t *testing.T) {
	tmpDir := t.TempDir()
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", origHome)

	// Load should return empty store when file doesn't exist
	store, err := LoadCredentials()
	if err != nil {
		t.Fatalf("LoadCredentials() error: %v", err)
	}
	if len(store.Bridges) != 0 {
		t.Errorf("LoadCredentials() on missing file returned %d bridges, want 0", len(store.Bridges))
	}

	// Add credential and save
	store.Set(BridgeCredential{
		BridgeID:  "test-bridge",
		Name:      "Test",
		IPAddress: "192.168.1.1",
		ApiKey:    "secret",
	})
	if err := store.Save(); err != nil {
		t.Fatalf("Save() error: %v", err)
	}

	// Verify file exists with restricted permissions
	credPath := filepath.Join(tmpDir, ".config", "lazyhue", "credentials.json")
	info, err := os.Stat(credPath)
	if os.IsNotExist(err) {
		t.Error("Credentials file was not created")
	}
	// Check permissions (should be 0600)
	if info.Mode().Perm() != 0600 {
		t.Errorf("Credentials file permissions = %o, want 0600", info.Mode().Perm())
	}

	// Load again and verify
	store2, err := LoadCredentials()
	if err != nil {
		t.Fatalf("LoadCredentials() after Save() error: %v", err)
	}
	cred, ok := store2.Get("test-bridge")
	if !ok {
		t.Error("Loaded store missing test-bridge")
	}
	if cred.ApiKey != "secret" {
		t.Errorf("Loaded ApiKey = %q, want %q", cred.ApiKey, "secret")
	}
}

func TestUIStateStore_LoadSave(t *testing.T) {
	tmpDir := t.TempDir()
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", origHome)

	// Load should return empty store when file doesn't exist
	store, err := LoadUIState()
	if err != nil {
		t.Fatalf("LoadUIState() error: %v", err)
	}
	if len(store.Bridges) != 0 {
		t.Errorf("LoadUIState() on missing file returned %d bridges, want 0", len(store.Bridges))
	}

	// Set state and save
	store.SetActiveTab("bridge-1", 3)
	store.LastSelectedBridgeID = "bridge-1"
	if err := store.Save(); err != nil {
		t.Fatalf("Save() error: %v", err)
	}

	// Load again and verify
	store2, err := LoadUIState()
	if err != nil {
		t.Fatalf("LoadUIState() after Save() error: %v", err)
	}
	if store2.LastSelectedBridgeID != "bridge-1" {
		t.Errorf("Loaded LastSelectedBridgeID = %q, want %q", store2.LastSelectedBridgeID, "bridge-1")
	}
	if store2.GetActiveTab("bridge-1") != 3 {
		t.Errorf("Loaded ActiveTab = %d, want %d", store2.GetActiveTab("bridge-1"), 3)
	}
}

func TestLoad_InvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", origHome)

	// Create config dir
	configDir := filepath.Join(tmpDir, ".config", "lazyhue")
	os.MkdirAll(configDir, 0755)

	// Write invalid JSON
	configPath := filepath.Join(configDir, "config.json")
	os.WriteFile(configPath, []byte("{invalid json"), 0644)

	// Load should return error for invalid JSON
	_, err := Load()
	if err == nil {
		t.Error("Load() should return error for invalid JSON")
	}
	if !strings.Contains(err.Error(), "invalid") {
		t.Errorf("Error should mention 'invalid', got: %v", err)
	}
}
