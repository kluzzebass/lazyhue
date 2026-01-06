package app

import (
	"context"
	"sync"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/kluzzebass/lazyhue/internal/hue"
)

// Bridge discovery and connection commands

func discoverBridges() tea.Cmd {
	return func() tea.Msg {
		discovery := hue.NewDiscoveryService(2 * time.Second)
		bridges := discovery.DiscoverAll()
		return BridgesDiscoveredMsg{Bridges: bridges}
	}
}

func connectBridge(bridge *hue.Bridge, apiKey string) tea.Cmd {
	return func() tea.Msg {
		if err := bridge.Connect(apiKey); err != nil {
			return BridgeDisconnectedMsg{BridgeID: bridge.Info.ID, Err: err}
		}
		return BridgeConnectedMsg{BridgeID: bridge.Info.ID}
	}
}

func syncBridgeState(bridge *hue.Bridge) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		if err := bridge.SyncAll(ctx); err != nil {
			return StateSyncErrorMsg{BridgeID: bridge.Info.ID, Err: err}
		}
		return StateSyncedMsg{BridgeID: bridge.Info.ID}
	}
}

func syncLights(bridge *hue.Bridge) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		if err := bridge.SyncLights(ctx); err != nil {
			return StateSyncErrorMsg{BridgeID: bridge.Info.ID, Err: err}
		}
		return LightsSyncedMsg{BridgeID: bridge.Info.ID}
	}
}

func syncLightsAndGroups(bridge *hue.Bridge) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		if err := bridge.SyncLights(ctx); err != nil {
			return StateSyncErrorMsg{BridgeID: bridge.Info.ID, Err: err}
		}
		if err := bridge.SyncGroupedLights(ctx); err != nil {
			return StateSyncErrorMsg{BridgeID: bridge.Info.ID, Err: err}
		}
		// Sync sensor services for live updates (non-fatal if they fail)
		_ = bridge.SyncMotionSensors(ctx)
		_ = bridge.SyncTemperatures(ctx)
		_ = bridge.SyncLightLevels(ctx)
		_ = bridge.SyncDevicePowers(ctx)
		return LightsSyncedMsg{BridgeID: bridge.Info.ID}
	}
}

func syncAllConnectedBridges(bridges []*hue.Bridge) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		var wg sync.WaitGroup

		for _, bridge := range bridges {
			if !bridge.IsConnected() {
				continue
			}
			wg.Add(1)
			go func(b *hue.Bridge) {
				defer wg.Done()
				// Sync lights and grouped lights
				_ = b.SyncLights(ctx)
				_ = b.SyncGroupedLights(ctx)
				// Sync sensor services (non-fatal)
				_ = b.SyncMotionSensors(ctx)
				_ = b.SyncTemperatures(ctx)
				_ = b.SyncLightLevels(ctx)
				_ = b.SyncDevicePowers(ctx)
			}(bridge)
		}

		wg.Wait()
		return LightsSyncedMsg{} // No specific bridge ID - all were synced
	}
}

// Ticker commands

func startSyncTicker() tea.Cmd {
	return tea.Tick(2*time.Second, func(t time.Time) tea.Msg {
		return SyncTickMsg{}
	})
}

func startDiscoveryTicker() tea.Cmd {
	return tea.Tick(10*time.Second, func(t time.Time) tea.Msg {
		return DiscoveryTickMsg{}
	})
}

// Pairing command

func startPairing(ctx context.Context, info hue.BridgeInfo) tea.Cmd {
	return func() tea.Msg {
		auth, err := hue.NewAuthenticator(info.IPAddress)
		if err != nil {
			return PairingFailedMsg{BridgeID: info.ID, Err: err}
		}

		apiKey, err := auth.AuthenticateWithContext(ctx, 500*time.Millisecond)
		if err != nil {
			// Don't report cancellation as an error
			if err == hue.ErrPairingCancelled {
				return nil // Silently ignore - user already knows they cancelled
			}
			return PairingFailedMsg{BridgeID: info.ID, Err: err}
		}

		return PairingSuccessMsg{BridgeID: info.ID, ApiKey: apiKey}
	}
}

// Light control commands

func toggleLight(bridge *hue.Bridge, lightID string) tea.Cmd {
	return func() tea.Msg {
		if err := bridge.ToggleLight(lightID); err != nil {
			return ActionErrorMsg{Action: "toggle light", Err: err}
		}
		return LightsSyncedMsg{BridgeID: bridge.Info.ID}
	}
}

func toggleGroupedLight(bridge *hue.Bridge, groupID string) tea.Cmd {
	return func() tea.Msg {
		if err := bridge.ToggleGroupedLight(groupID); err != nil {
			return ActionErrorMsg{Action: "toggle group", Err: err}
		}
		return LightsSyncedMsg{BridgeID: bridge.Info.ID}
	}
}

func setLightOn(bridge *hue.Bridge, lightID string, on bool) tea.Cmd {
	return func() tea.Msg {
		if err := bridge.SetLightOn(lightID, on); err != nil {
			return ActionErrorMsg{Action: "set light", Err: err}
		}
		return LightsSyncedMsg{BridgeID: bridge.Info.ID}
	}
}

func setGroupedLightOn(bridge *hue.Bridge, groupID string, on bool) tea.Cmd {
	return func() tea.Msg {
		if err := bridge.SetGroupedLightOn(groupID, on); err != nil {
			return ActionErrorMsg{Action: "set group", Err: err}
		}
		return LightsSyncedMsg{BridgeID: bridge.Info.ID}
	}
}

func setLightBrightness(bridge *hue.Bridge, lightID string, brightness float64) tea.Cmd {
	return func() tea.Msg {
		if err := bridge.SetLightBrightness(lightID, brightness); err != nil {
			return ActionErrorMsg{Action: "set brightness", Err: err}
		}
		return LightsSyncedMsg{BridgeID: bridge.Info.ID}
	}
}

func setGroupedLightBrightness(bridge *hue.Bridge, groupID string, brightness float64) tea.Cmd {
	return func() tea.Msg {
		if err := bridge.SetGroupedLightBrightness(groupID, brightness); err != nil {
			return ActionErrorMsg{Action: "set brightness", Err: err}
		}
		return LightsSyncedMsg{BridgeID: bridge.Info.ID}
	}
}

func recallScene(bridge *hue.Bridge, sceneID string) tea.Cmd {
	return func() tea.Msg {
		if err := bridge.RecallScene(sceneID); err != nil {
			return ActionErrorMsg{Action: "recall scene", Err: err}
		}
		return LightsSyncedMsg{BridgeID: bridge.Info.ID}
	}
}
