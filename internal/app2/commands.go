package app2

import (
	"context"
	"time"

	tea "github.com/charmbracelet/bubbletea/v2"

	"github.com/kluzzebass/lazyhue/internal/hue"
)

// connectBridge connects a single bridge and sends a message.
func (m *Model) connectBridge(bridge *hue.Bridge, apiKey string) tea.Cmd {
	return func() tea.Msg {
		if err := bridge.Connect(apiKey); err != nil {
			return errMsg{err: err}
		}
		return bridgeConnectedMsg{bridgeID: bridge.Info.ID}
	}
}

// loadBridgesFromCredentials loads saved credentials and connects to all bridges.
func (m *Model) loadBridgesFromCredentials() tea.Cmd {
	// Load bridges from stored credentials
	var cmds []tea.Cmd
	for _, cred := range m.credentials.Bridges {
		name := cred.Name
		if name == "" {
			name = cred.BridgeID
		}
		info := hue.BridgeInfo{
			ID:        cred.BridgeID,
			Name:      name,
			IPAddress: cred.IPAddress,
		}
		m.manager.AddBridge(info)
		if bridge := m.manager.GetBridge(cred.BridgeID); bridge != nil && cred.ApiKey != "" {
			cmds = append(cmds, m.connectBridge(bridge, cred.ApiKey))
		}
	}

	// Also discover bridges on the network
	discovery := hue.NewDiscoveryService(5 * time.Second)
	bridges, err := discovery.Discover()
	if err == nil {
		for _, info := range bridges {
			// Add if not already in manager
			if m.manager.GetBridge(info.ID) == nil {
				m.manager.AddBridge(info)
			}
		}
	}

	return tea.Batch(cmds...)
}

// syncBridgeState syncs state for a bridge.
func (m *Model) syncBridgeState(bridgeID string) tea.Cmd {
	return func() tea.Msg {
		bridge := m.manager.GetBridge(bridgeID)
		if bridge == nil {
			return nil
		}

		ctx := context.Background()
		if err := bridge.SyncAll(ctx); err != nil {
			return errMsg{err}
		}

		return stateSyncedMsg{bridgeID: bridgeID}
	}
}

// startSSEListener starts listening to SSE events from a bridge.
// This runs in a goroutine and sends events to m.eventChan.
func (m *Model) startSSEListener(ctx context.Context, bridge *hue.Bridge) {
	bridge.OnEvent(func(bridgeID, resourceType, resourceID, eventType string) {
		select {
		case <-ctx.Done():
			return
		case m.eventChan <- bridgeEventMsg{
			bridgeID:     bridgeID,
			resourceType: resourceType,
			resourceID:   resourceID,
			eventType:    eventType,
		}:
		}
	})
}

// listenForEvents returns a command that listens for events from the channel.
func (m *Model) listenForEvents() tea.Cmd {
	return func() tea.Msg {
		return <-m.eventChan
	}
}
