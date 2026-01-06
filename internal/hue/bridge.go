package hue

import (
	"context"
	"sync"
	"time"

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

	home     *openhue.Home
	extended *ExtendedClient
	state    *BridgeState
	mu       sync.RWMutex
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
	return nil
}

// Disconnect closes the bridge connection.
func (b *Bridge) Disconnect() {
	b.mu.Lock()
	defer b.mu.Unlock()

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

	// Update bridge name from device metadata
	if name := b.state.BridgeName(); name != "" {
		b.mu.Lock()
		b.Info.Name = name
		b.mu.Unlock()
	}

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

