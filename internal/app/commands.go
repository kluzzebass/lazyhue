package app

import (
	"context"
	"fmt"
	"sync"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/kluzzebass/lazyhue/internal/hue"
)

// syncPoolSize controls how many API calls run concurrently per bridge.
// Adjust this to balance speed vs. bridge load.
const syncPoolSize = 4

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
		if err := bridge.SyncAllBulk(ctx); err != nil {
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

// syncTask represents a single sync operation to be executed by the worker pool.
type syncTask func()

// runWithWorkerPool executes tasks using a fixed-size worker pool.
func runWithWorkerPool(tasks []syncTask, poolSize int) {
	if len(tasks) == 0 {
		return
	}

	taskChan := make(chan syncTask, len(tasks))
	var wg sync.WaitGroup

	// Start workers
	for i := 0; i < poolSize; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for task := range taskChan {
				task()
			}
		}()
	}

	// Send tasks
	for _, task := range tasks {
		taskChan <- task
	}
	close(taskChan)

	// Wait for all workers to finish
	wg.Wait()
}

func syncAllConnectedBridges(bridges []*hue.Bridge) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		var bridgeWg sync.WaitGroup

		for _, bridge := range bridges {
			if !bridge.IsConnected() {
				continue
			}
			bridgeWg.Add(1)
			go func(b *hue.Bridge) {
				defer bridgeWg.Done()

				// Define all sync tasks for this bridge
				tasks := []syncTask{
					func() { _ = b.SyncLights(ctx) },
					func() { _ = b.SyncRooms(ctx) },
					func() { _ = b.SyncZones(ctx) },
					func() { _ = b.SyncGroupedLights(ctx) },
					func() { _ = b.SyncScenes(ctx) },
					func() { _ = b.SyncDevices(ctx) },
					func() { _ = b.SyncMotionSensors(ctx) },
					func() { _ = b.SyncTemperatures(ctx) },
					func() { _ = b.SyncLightLevels(ctx) },
					func() { _ = b.SyncDevicePowers(ctx) },
					func() { _ = b.SyncZigbeeConnectivity(ctx) },
					func() { _ = b.SyncEntertainmentConfigurations(ctx) },
				}

				// Run tasks with worker pool
				runWithWorkerPool(tasks, syncPoolSize)
			}(bridge)
		}

		bridgeWg.Wait()
		return LightsSyncedMsg{} // No specific bridge ID - all were synced
	}
}

// Ticker commands

func startSyncTicker() tea.Cmd {
	// With SSE for real-time updates, polling is just a fallback
	// Reduced from 2s to 30s to minimize unnecessary traffic
	return tea.Tick(30*time.Second, func(t time.Time) tea.Msg {
		return SyncTickMsg{}
	})
}

func startDiscoveryTicker() tea.Cmd {
	return tea.Tick(10*time.Second, func(t time.Time) tea.Msg {
		return DiscoveryTickMsg{}
	})
}

func startStateSaveTicker() tea.Cmd {
	// Save UI state every 5 seconds to handle abrupt termination
	return tea.Tick(5*time.Second, func(t time.Time) tea.Msg {
		return StateSaveTickMsg{}
	})
}

// listenForBridgeEvents listens for SSE events on the channel and returns them as messages.
func listenForBridgeEvents(eventChan <-chan BridgeEventMsg) tea.Cmd {
	return func() tea.Msg {
		// Block until we receive an event
		event, ok := <-eventChan
		if !ok {
			return nil // Channel closed
		}
		return event
	}
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
		// Get light name before toggle
		lightName := lightID
		if light, ok := bridge.GetState().GetLight(lightID); ok {
			lightName = bridge.GetState().GetLightName(light)
		}
		if err := bridge.ToggleLight(lightID); err != nil {
			return ActionErrorMsg{Action: "toggle light", Err: err}
		}
		return LightsSyncedMsg{BridgeID: bridge.Info.ID, Action: "toggle", Target: lightName}
	}
}

func toggleGroupedLight(bridge *hue.Bridge, groupID string) tea.Cmd {
	return func() tea.Msg {
		// Get group name
		groupName := bridge.GetState().GetGroupedLightName(groupID)
		if groupName == "" {
			groupName = groupID
		}
		if err := bridge.ToggleGroupedLight(groupID); err != nil {
			return ActionErrorMsg{Action: "toggle group", Err: err}
		}
		return LightsSyncedMsg{BridgeID: bridge.Info.ID, Action: "toggle", Target: groupName}
	}
}

func setLightOn(bridge *hue.Bridge, lightID string, on bool) tea.Cmd {
	return func() tea.Msg {
		lightName := lightID
		if light, ok := bridge.GetState().GetLight(lightID); ok {
			lightName = bridge.GetState().GetLightName(light)
		}
		if err := bridge.SetLightOn(lightID, on); err != nil {
			return ActionErrorMsg{Action: "set light", Err: err}
		}
		action := "turn on"
		if !on {
			action = "turn off"
		}
		return LightsSyncedMsg{BridgeID: bridge.Info.ID, Action: action, Target: lightName}
	}
}

func setGroupedLightOn(bridge *hue.Bridge, groupID string, on bool) tea.Cmd {
	return func() tea.Msg {
		groupName := bridge.GetState().GetGroupedLightName(groupID)
		if groupName == "" {
			groupName = groupID
		}
		if err := bridge.SetGroupedLightOn(groupID, on); err != nil {
			return ActionErrorMsg{Action: "set group", Err: err}
		}
		action := "turn on"
		if !on {
			action = "turn off"
		}
		return LightsSyncedMsg{BridgeID: bridge.Info.ID, Action: action, Target: groupName}
	}
}

func setLightBrightness(bridge *hue.Bridge, lightID string, brightness float64) tea.Cmd {
	return func() tea.Msg {
		lightName := lightID
		if light, ok := bridge.GetState().GetLight(lightID); ok {
			lightName = bridge.GetState().GetLightName(light)
		}
		if err := bridge.SetLightBrightness(lightID, brightness); err != nil {
			return ActionErrorMsg{Action: "set brightness", Err: err}
		}
		return LightsSyncedMsg{
			BridgeID: bridge.Info.ID,
			Action:   fmt.Sprintf("brightness %.0f%%", brightness),
			Target:   lightName,
		}
	}
}

func setGroupedLightBrightness(bridge *hue.Bridge, groupID string, brightness float64) tea.Cmd {
	return func() tea.Msg {
		groupName := bridge.GetState().GetGroupedLightName(groupID)
		if groupName == "" {
			groupName = groupID
		}
		if err := bridge.SetGroupedLightBrightness(groupID, brightness); err != nil {
			return ActionErrorMsg{Action: "set brightness", Err: err}
		}
		return LightsSyncedMsg{
			BridgeID: bridge.Info.ID,
			Action:   fmt.Sprintf("brightness %.0f%%", brightness),
			Target:   groupName,
		}
	}
}

func recallScene(bridge *hue.Bridge, sceneID string) tea.Cmd {
	return func() tea.Msg {
		sceneName := sceneID
		if scene, ok := bridge.GetState().GetScene(sceneID); ok {
			sceneName = scene.SceneName(sceneID)
		}
		if err := bridge.RecallScene(sceneID); err != nil {
			return ActionErrorMsg{Action: "recall scene", Err: err}
		}
		return LightsSyncedMsg{BridgeID: bridge.Info.ID, Action: "activate", Target: sceneName}
	}
}
