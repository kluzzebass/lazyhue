package app2

import (
	"github.com/kluzzebass/lazyhue/internal/hue"
)

// Messages for the application.
type (
	// bridgesDiscoveredMsg is sent when bridge discovery completes.
	bridgesDiscoveredMsg struct {
		bridges []hue.BridgeInfo
	}

	// bridgeConnectedMsg is sent when a bridge connects successfully.
	bridgeConnectedMsg struct {
		bridgeID string
	}

	// stateSyncedMsg is sent when bridge state is synced.
	stateSyncedMsg struct {
		bridgeID string
	}

	// bridgeEventMsg wraps SSE events from bridges.
	bridgeEventMsg struct {
		bridgeID     string
		resourceType string
		resourceID   string
		eventType    string
	}

	// errMsg wraps errors.
	errMsg struct {
		err error
	}

	// bridgeBlinkTickMsg is sent when a bridge blink timer expires.
	bridgeBlinkTickMsg struct {
		bridgeID string
	}

	// requestMsg wraps API request messages from bridges.
	requestMsg struct {
		bridgeID string
		message  string
	}

	// errorMsg wraps API error messages from bridges.
	errorMsg struct {
		bridgeID string
		message  string
	}

	// stateSaveTickMsg triggers periodic UI state save.
	stateSaveTickMsg struct{}

	// discoveryTickMsg triggers periodic bridge discovery.
	discoveryTickMsg struct{}

	// pairingTickMsg is sent every second during pairing for countdown updates.
	pairingTickMsg struct{}

	// pairingSuccessMsg is sent when bridge pairing succeeds.
	pairingSuccessMsg struct {
		BridgeID string
		ApiKey   string
	}

	// pairingFailedMsg is sent when bridge pairing fails.
	pairingFailedMsg struct {
		BridgeID string
		Err      error
	}
)
