package hue

import (
	"context"
	"sync"
	"time"

	"github.com/kluzzebass/lazyhue/internal/debug"
	"github.com/openhue/openhue-go"
)

// ConnectionStatus represents the state of a bridge connection.
type ConnectionStatus int

const (
	StatusDisconnected ConnectionStatus = iota
	StatusConnecting
	StatusConnected
	StatusPairing
	StatusError
)

func (s ConnectionStatus) String() string {
	switch s {
	case StatusDisconnected:
		return "disconnected"
	case StatusConnecting:
		return "connecting"
	case StatusConnected:
		return "connected"
	case StatusPairing:
		return "pairing"
	case StatusError:
		return "error"
	default:
		return "unknown"
	}
}

// Bridge represents a connection to a single Hue bridge.
type Bridge struct {
	Info     BridgeInfo
	Status   ConnectionStatus
	LastSync time.Time
	LastErr  error

	home        *openhue.Home
	extended    *ExtendedClient
	state       *BridgeState
	eventStream *EventStream
	eventCancel context.CancelFunc
	onEvent     func(bridgeID string) // Callback when events are received
	mu          sync.RWMutex
}

// NewBridge creates a new bridge connection.
func NewBridge(info BridgeInfo) *Bridge {
	return &Bridge{
		Info:   info,
		Status: StatusDisconnected,
		state:  NewBridgeState(),
	}
}

// Connect establishes a connection to the bridge using the provided API key.
func (b *Bridge) Connect(apiKey string) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.Status = StatusConnecting

	home, err := openhue.NewHome(b.Info.IPAddress, apiKey)
	if err != nil {
		b.Status = StatusError
		b.LastErr = err
		return err
	}

	// Create extended client for zones and other APIs not exposed by Home
	extended, err := NewExtendedClient(b.Info.IPAddress, apiKey)
	if err != nil {
		b.Status = StatusError
		b.LastErr = err
		return err
	}

	b.home = home
	b.extended = extended
	b.Status = StatusConnected
	b.LastErr = nil

	// Start event stream
	b.startEventStream(apiKey)

	return nil
}

// OnEvent sets a callback for when the bridge receives SSE events.
func (b *Bridge) OnEvent(fn func(bridgeID string)) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.onEvent = fn
}

// IsEventStreamConnected returns whether the SSE connection is active.
func (b *Bridge) IsEventStreamConnected() bool {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.eventStream != nil && b.eventStream.IsConnected()
}

func (b *Bridge) startEventStream(apiKey string) {
	// Stop any existing stream
	if b.eventCancel != nil {
		b.eventCancel()
	}

	es := NewEventStream(b.Info.IPAddress, apiKey)

	es.OnEvent(func(container EventContainer) {
		b.handleEvents(container)
	})

	es.OnConnect(func() {
		// Event stream connected - we can reduce polling frequency
	})

	es.OnDisconnect(func(err error) {
		// Event stream disconnected - might want to increase polling
	})

	ctx, cancel := context.WithCancel(context.Background())
	b.eventStream = es
	b.eventCancel = cancel

	// Start in background
	go es.Start(ctx)
}

// handleEvents processes events from the SSE stream.
// Applies partial updates directly to the in-memory state, avoiding API calls.
func (b *Bridge) handleEvents(container EventContainer) {
	if len(container.Events) == 0 {
		return
	}

	anyUpdated := false

	for _, event := range container.Events {
		updates, err := ParseResourceUpdates(event)
		if err != nil {
			continue
		}

		for _, update := range updates {
			debug.Log("SSE %s: %s %s", event.Type, update.Type, update.ID)

			// Apply the update directly to our cached state
			updated := b.applyResourceUpdate(update)
			if updated {
				anyUpdated = true
			}
		}
	}

	// Notify listener if anything was updated
	if anyUpdated {
		b.mu.RLock()
		callback := b.onEvent
		b.mu.RUnlock()

		if callback != nil {
			callback(b.Info.ID)
		}
	}
}

// applyResourceUpdate applies a single resource update to the bridge state.
// Returns true if the update was applied successfully.
func (b *Bridge) applyResourceUpdate(update ResourceUpdate) bool {
	if b.state == nil {
		return false
	}

	switch update.Type {
	case "light":
		var on *bool
		var brightness *float64
		if update.On != nil {
			on = &update.On.On
		}
		if update.Dimming != nil {
			brightness = &update.Dimming.Brightness
		}
		return b.state.ApplyLightUpdate(update.ID, on, brightness)

	case "grouped_light":
		var on *bool
		var brightness *float64
		if update.On != nil {
			on = &update.On.On
		}
		if update.Dimming != nil {
			brightness = &update.Dimming.Brightness
		}
		return b.state.ApplyGroupedLightUpdate(update.ID, on, brightness)

	case "motion":
		if update.Motion != nil {
			return b.state.ApplyMotionUpdate(update.ID, update.Motion.Motion)
		}
		return false

	case "scene":
		if update.Status != "" {
			return b.state.ApplySceneStatus(update.ID, update.Status)
		}
		return false

	default:
		// Unknown resource type - will be picked up by fallback polling
		return false
	}
}

// Disconnect closes the bridge connection.
func (b *Bridge) Disconnect() {
	b.mu.Lock()
	defer b.mu.Unlock()

	// Stop event stream
	if b.eventCancel != nil {
		b.eventCancel()
		b.eventCancel = nil
	}
	b.eventStream = nil

	b.home = nil
	b.Status = StatusDisconnected
}

// IsConnected returns true if the bridge is connected.
func (b *Bridge) IsConnected() bool {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.Status == StatusConnected && b.home != nil
}

// GetState returns the current cached state.
func (b *Bridge) GetState() *BridgeState {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.state
}

// Home returns the underlying openhue Home for direct API access.
func (b *Bridge) Home() *openhue.Home {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.home
}

// SyncAll fetches all entity types from the bridge.
func (b *Bridge) SyncAll(ctx context.Context) error {
	if err := b.SyncLights(ctx); err != nil {
		return err
	}
	if err := b.SyncRooms(ctx); err != nil {
		return err
	}
	if err := b.SyncZones(ctx); err != nil {
		return err
	}
	if err := b.SyncGroupedLights(ctx); err != nil {
		return err
	}
	if err := b.SyncScenes(ctx); err != nil {
		return err
	}
	if err := b.SyncDevices(ctx); err != nil {
		return err
	}
	// Sensor services (non-fatal if they fail - device may not have them)
	_ = b.SyncMotionSensors(ctx)
	_ = b.SyncTemperatures(ctx)
	_ = b.SyncLightLevels(ctx)
	_ = b.SyncDevicePowers(ctx)

	// Entertainment configurations (non-fatal)
	_ = b.SyncEntertainmentConfigurations(ctx)
	_ = b.SyncWifiConnectivity(ctx)
	_ = b.SyncZigbeeConnectivity(ctx)

	// Bridge resource, home, and auth apps (non-fatal)
	_ = b.SyncBridgeResource(ctx)
	_ = b.SyncBridgeHome(ctx)
	_ = b.SyncAuthApps(ctx)

	b.mu.Lock()
	b.LastSync = time.Now()
	b.mu.Unlock()
	return nil
}

// SyncLights fetches only lights from the bridge.
func (b *Bridge) SyncLights(ctx context.Context) error {
	b.mu.RLock()
	home := b.home
	b.mu.RUnlock()

	if home == nil {
		return ErrAuthFailed
	}

	lights, err := home.GetLights()
	if err != nil {
		return err
	}

	b.state.UpdateLights(lights)
	return nil
}

// SyncRooms fetches only rooms from the bridge.
func (b *Bridge) SyncRooms(ctx context.Context) error {
	b.mu.RLock()
	home := b.home
	b.mu.RUnlock()

	if home == nil {
		return ErrAuthFailed
	}

	rooms, err := home.GetRooms()
	if err != nil {
		return err
	}

	b.state.UpdateRooms(rooms)
	return nil
}

// SyncZones fetches only zones from the bridge.
func (b *Bridge) SyncZones(ctx context.Context) error {
	b.mu.RLock()
	extended := b.extended
	b.mu.RUnlock()

	if extended == nil {
		return ErrAuthFailed
	}

	zones, err := extended.GetZones(ctx)
	if err != nil {
		return err
	}

	b.state.UpdateZones(zones)
	return nil
}

// SyncGroupedLights fetches only grouped lights from the bridge.
func (b *Bridge) SyncGroupedLights(ctx context.Context) error {
	b.mu.RLock()
	home := b.home
	b.mu.RUnlock()

	if home == nil {
		return ErrAuthFailed
	}

	grouped, err := home.GetGroupedLights()
	if err != nil {
		return err
	}

	b.state.UpdateGroupedLights(grouped)
	return nil
}

// SyncScenes fetches only scenes from the bridge.
func (b *Bridge) SyncScenes(ctx context.Context) error {
	b.mu.RLock()
	home := b.home
	b.mu.RUnlock()

	if home == nil {
		return ErrAuthFailed
	}

	scenes, err := home.GetScenes()
	if err != nil {
		return err
	}

	b.state.UpdateScenes(scenes)
	return nil
}

// SyncDevices fetches only devices from the bridge.
func (b *Bridge) SyncDevices(ctx context.Context) error {
	b.mu.RLock()
	home := b.home
	b.mu.RUnlock()

	if home == nil {
		return ErrAuthFailed
	}

	devices, err := home.GetDevices()
	if err != nil {
		return err
	}

	b.state.UpdateDevices(devices)
	return nil
}

// SyncMotionSensors fetches only motion sensors from the bridge.
func (b *Bridge) SyncMotionSensors(ctx context.Context) error {
	b.mu.RLock()
	extended := b.extended
	b.mu.RUnlock()

	if extended == nil {
		return ErrAuthFailed
	}

	sensors, err := extended.GetMotionSensors(ctx)
	if err != nil {
		return err
	}

	b.state.UpdateMotionSensors(sensors)
	return nil
}

// SyncTemperatures fetches only temperature sensors from the bridge.
func (b *Bridge) SyncTemperatures(ctx context.Context) error {
	b.mu.RLock()
	extended := b.extended
	b.mu.RUnlock()

	if extended == nil {
		return ErrAuthFailed
	}

	temps, err := extended.GetTemperatures(ctx)
	if err != nil {
		return err
	}

	b.state.UpdateTemperatures(temps)
	return nil
}

// SyncLightLevels fetches only light level sensors from the bridge.
func (b *Bridge) SyncLightLevels(ctx context.Context) error {
	b.mu.RLock()
	extended := b.extended
	b.mu.RUnlock()

	if extended == nil {
		return ErrAuthFailed
	}

	levels, err := extended.GetLightLevels(ctx)
	if err != nil {
		return err
	}

	b.state.UpdateLightLevels(levels)
	return nil
}

// SyncDevicePowers fetches only device power (battery) statuses from the bridge.
func (b *Bridge) SyncDevicePowers(ctx context.Context) error {
	b.mu.RLock()
	extended := b.extended
	b.mu.RUnlock()

	if extended == nil {
		return ErrAuthFailed
	}

	powers, err := extended.GetDevicePowers(ctx)
	if err != nil {
		return err
	}

	b.state.UpdateDevicePowers(powers)
	return nil
}

// SyncBridgeResource fetches the bridge resource.
func (b *Bridge) SyncBridgeResource(ctx context.Context) error {
	b.mu.RLock()
	extended := b.extended
	b.mu.RUnlock()

	if extended == nil {
		return ErrAuthFailed
	}

	bridges, err := extended.GetBridges(ctx)
	if err != nil {
		return err
	}

	if len(bridges) > 0 {
		b.state.UpdateBridgeResource(&bridges[0])
	}
	return nil
}

// SyncBridgeHome fetches the bridge home resource.
func (b *Bridge) SyncBridgeHome(ctx context.Context) error {
	b.mu.RLock()
	extended := b.extended
	b.mu.RUnlock()

	if extended == nil {
		return ErrAuthFailed
	}

	home, err := extended.GetBridgeHome(ctx)
	if err != nil {
		return err
	}

	b.state.UpdateBridgeHome(home)
	return nil
}

// SyncAuthApps fetches the authenticated applications list.
func (b *Bridge) SyncAuthApps(ctx context.Context) error {
	b.mu.RLock()
	extended := b.extended
	b.mu.RUnlock()

	if extended == nil {
		return ErrAuthFailed
	}

	apps, err := extended.GetAuthenticatedApps(ctx)
	if err != nil {
		return err
	}

	b.state.UpdateAuthApps(apps)
	return nil
}

// SyncEntertainmentConfigurations fetches entertainment configuration data.
func (b *Bridge) SyncEntertainmentConfigurations(ctx context.Context) error {
	b.mu.RLock()
	extended := b.extended
	b.mu.RUnlock()

	if extended == nil {
		return ErrAuthFailed
	}

	configs, err := extended.GetEntertainmentConfigurations(ctx)
	if err != nil {
		return err
	}

	b.state.UpdateEntertainmentConfigurations(configs)
	return nil
}

// SyncWifiConnectivity fetches WiFi connectivity status (Bridge Pro only).
func (b *Bridge) SyncWifiConnectivity(ctx context.Context) error {
	b.mu.RLock()
	extended := b.extended
	b.mu.RUnlock()

	if extended == nil {
		return ErrAuthFailed
	}

	wifi, err := extended.GetWifiConnectivity(ctx)
	if err != nil {
		return err
	}

	b.state.UpdateWifiConnectivity(wifi)
	return nil
}

// SyncZigbeeConnectivity fetches Zigbee connectivity status for all devices.
func (b *Bridge) SyncZigbeeConnectivity(ctx context.Context) error {
	b.mu.RLock()
	extended := b.extended
	b.mu.RUnlock()

	if extended == nil {
		return ErrAuthFailed
	}

	zigbee, err := extended.GetZigbeeConnectivity(ctx)
	if err != nil {
		return err
	}

	b.state.UpdateZigbeeConnectivity(zigbee)
	return nil
}
