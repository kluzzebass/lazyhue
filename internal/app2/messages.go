package app2

// Messages for the application.
type (
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
)
