// Package app contains the main Bubble Tea application model.
package app

import (
	"github.com/kluzzebass/lazyhue/internal/hue"
	"github.com/kluzzebass/lazyhue/internal/ui/panels"
)

// Bridge lifecycle messages

// BridgesDiscoveredMsg is sent when bridge discovery completes.
type BridgesDiscoveredMsg struct {
	Bridges []hue.BridgeInfo
}

// BridgeConnectedMsg is sent when a bridge successfully connects.
type BridgeConnectedMsg struct {
	BridgeID string
}

// BridgeDisconnectedMsg is sent when a bridge disconnects.
type BridgeDisconnectedMsg struct {
	BridgeID string
	Err      error
}

// PairingRequiredMsg is sent when a bridge needs authentication.
type PairingRequiredMsg struct {
	Bridge hue.BridgeInfo
}

// PairingProgressMsg is sent during the pairing polling loop.
type PairingProgressMsg struct {
	BridgeID string
	Message  string
}

// PairingTickMsg is sent every second to update the pairing countdown.
type PairingTickMsg struct{}

// PairingSuccessMsg is sent when pairing completes successfully.
type PairingSuccessMsg struct {
	BridgeID string
	ApiKey   string
}

// PairingFailedMsg is sent when pairing fails.
type PairingFailedMsg struct {
	BridgeID string
	Err      error
}

// State synchronization messages

// StateSyncedMsg is sent when bridge state is refreshed.
type StateSyncedMsg struct {
	BridgeID string
}

// StateSyncErrorMsg is sent when state sync fails.
type StateSyncErrorMsg struct {
	BridgeID string
	Err      error
}

// LightsSyncedMsg is sent when only lights are refreshed.
type LightsSyncedMsg struct {
	BridgeID string
	Action   string // Optional: what action was performed (for logging)
	Target   string // Optional: target resource name
}

// MotionSensorsSyncedMsg is sent when motion sensors are updated.
type MotionSensorsSyncedMsg struct {
	BridgeID string
	Action   string // Optional: what action was performed (for logging)
	Target   string // Optional: target resource name
}

// SyncTickMsg triggers periodic state sync.
type SyncTickMsg struct{}

// StateSaveTickMsg triggers periodic UI state save.
type StateSaveTickMsg struct{}

// SignalQuitMsg is sent when SIGTERM/SIGINT is received.
type SignalQuitMsg struct{}

// DiscoveryTickMsg triggers periodic bridge discovery.
type DiscoveryTickMsg struct{}

// BridgeEventMsg is sent when an SSE event is received from a bridge.
type BridgeEventMsg struct {
	BridgeID     string
	ResourceType string // e.g., "light", "scene", "motion"
	ResourceID   string
	EventType    string // e.g., "update", "add", "delete"
}

// indicatorRefreshMsg triggers a redraw to update activity indicators.
type indicatorRefreshMsg struct{}

// Action result messages

// LightToggledMsg is sent after toggling a light.
type LightToggledMsg struct {
	LightID  string
	NewState bool
}

// BrightnessChangedMsg is sent after changing brightness.
type BrightnessChangedMsg struct {
	EntityID   string
	Brightness float64
}

// SceneActivatedMsg is sent after activating a scene.
type SceneActivatedMsg struct {
	SceneID string
}

// ActionErrorMsg is sent when an action fails.
type ActionErrorMsg struct {
	Action string
	Err    error
}

// UI messages

// EntitySelectedMsg is sent when the user selects an entity.
type EntitySelectedMsg struct {
	Item panels.EntityItem
}

// StatusMsg displays a temporary status message.
type StatusMsg struct {
	Message string
	IsError bool
}

// ClearStatusMsg clears the status message.
type ClearStatusMsg struct{}
