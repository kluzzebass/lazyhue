package app2

import (
	"context"
	"time"

	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/kluzzebass/lazyhue/internal/hue"
)

// discoverBridges discovers all Hue bridges on the network.
func discoverBridges() tea.Cmd {
	return func() tea.Msg {
		discovery := hue.NewDiscoveryService(2 * time.Second)
		bridges := discovery.DiscoverAll()
		return bridgesDiscoveredMsg{bridges: bridges}
	}
}

// startPairing initiates the pairing process with a bridge.
func startPairing(ctx context.Context, info hue.BridgeInfo) tea.Cmd {
	return func() tea.Msg {
		auth, err := hue.NewAuthenticator(info.IPAddress)
		if err != nil {
			return pairingFailedMsg{BridgeID: info.ID, Err: err}
		}

		apiKey, err := auth.AuthenticateWithContext(ctx, 500*time.Millisecond)
		if err != nil {
			// Don't report cancellation as an error
			if err == hue.ErrPairingCancelled {
				return nil // Silently ignore - user already knows they cancelled
			}
			return pairingFailedMsg{BridgeID: info.ID, Err: err}
		}

		return pairingSuccessMsg{BridgeID: info.ID, ApiKey: apiKey}
	}
}

// pairingTick returns a command that ticks every second for the pairing countdown.
func pairingTick() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return pairingTickMsg{}
	})
}
