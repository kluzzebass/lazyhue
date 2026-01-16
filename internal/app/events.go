package app

import (
	"context"
	"time"

	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/kluzzebass/lazyhue/internal/hue"
)

// loadBridgesFromCredentials loads all bridges from stored credentials and connects them.
func (m *Model) loadBridgesFromCredentials() tea.Cmd {
	var cmds []tea.Cmd

	for _, cred := range m.credentials.Bridges {
		name := cred.Name
		if name == "" {
			name = cred.BridgeID // Fallback for old credentials without name
		}
		info := hue.BridgeInfo{
			ID:        cred.BridgeID,
			Name:      name,
			IPAddress: cred.IPAddress,
		}
		m.manager.AddBridge(info)
		if bridge := m.manager.GetBridge(cred.BridgeID); bridge != nil {
			cmds = append(cmds, connectBridge(bridge, cred.ApiKey))
		}
	}

	return tea.Batch(cmds...)
}

// connectBridge connects to a bridge with the given API key.
func connectBridge(bridge *hue.Bridge, apiKey string) tea.Cmd {
	return func() tea.Msg {
		if err := bridge.Connect(apiKey); err != nil {
			return errMsg{err: err}
		}
		return bridgeConnectedMsg{bridgeID: bridge.Info.ID}
	}
}

// syncBridgeState syncs the state of a specific bridge.
func (m *Model) syncBridgeState(bridgeID string) tea.Cmd {
	return func() tea.Msg {
		bridge := m.manager.GetBridge(bridgeID)
		if bridge == nil {
			return errMsg{err: nil}
		}

		ctx := context.Background()
		if err := bridge.SyncAllBulk(ctx); err != nil {
			return errMsg{err: err}
		}
		return stateSyncedMsg{bridgeID: bridgeID}
	}
}

// startStateSaveTicker returns a command that periodically triggers state saves.
func (m *Model) startStateSaveTicker() tea.Cmd {
	return tea.Tick(5*time.Second, func(t time.Time) tea.Msg {
		return stateSaveTickMsg{}
	})
}

// startDiscoveryTicker returns a command that periodically triggers bridge discovery.
func startDiscoveryTicker() tea.Cmd {
	return tea.Tick(10*time.Second, func(t time.Time) tea.Msg {
		return discoveryTickMsg{}
	})
}

// startSSEListener starts listening for SSE events from a bridge.
// The bridge handles the SSE connection internally, we just set the callback.
func (m *Model) startSSEListener(ctx context.Context, bridge *hue.Bridge) {
	// Set event callback to forward events to our channel
	bridge.OnEvent(func(bridgeID string, update hue.ResourceUpdate, eventType string) {
		select {
		case m.eventChan <- bridgeEventMsg{
			bridgeID:   bridgeID,
			update:     update,
			eventType:  eventType,
			receivedAt: time.Now(),
		}:
		case <-ctx.Done():
			return
		}
	})
}

// listenForEvents returns a command that waits for events from the event channel.
func (m *Model) listenForEvents() tea.Cmd {
	return func() tea.Msg {
		event, ok := <-m.eventChan
		if !ok {
			return nil // Channel closed
		}
		return event
	}
}

// startRequestListener sets the request callback for a bridge.
func (m *Model) startRequestListener(ctx context.Context, bridge *hue.Bridge) {
	bridge.OnRequest(func(bridgeID, message string) {
		select {
		case m.requestChan <- requestMsg{
			bridgeID: bridgeID,
			message:  message,
		}:
		case <-ctx.Done():
			return
		}
	})
}

// listenForRequests returns a command that waits for request messages from the request channel.
func (m *Model) listenForRequests() tea.Cmd {
	return func() tea.Msg {
		msg, ok := <-m.requestChan
		if !ok {
			return nil // Channel closed
		}
		return msg
	}
}

// startErrorListener sets the error callback for a bridge.
func (m *Model) startErrorListener(ctx context.Context, bridge *hue.Bridge) {
	bridge.OnError(func(bridgeID, message string) {
		select {
		case m.errorChan <- errorMsg{
			bridgeID: bridgeID,
			message:  message,
		}:
		case <-ctx.Done():
			return
		}
	})
}

// listenForErrors returns a command that waits for error messages from the error channel.
func (m *Model) listenForErrors() tea.Cmd {
	return func() tea.Msg {
		msg, ok := <-m.errorChan
		if !ok {
			return nil // Channel closed
		}
		return msg
	}
}
