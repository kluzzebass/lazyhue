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

	home  *openhue.Home
	state *BridgeState
	mu    sync.RWMutex
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

	b.home = home
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
	if err := b.SyncGroupedLights(ctx); err != nil {
		return err
	}
	if err := b.SyncScenes(ctx); err != nil {
		return err
	}
	if err := b.SyncDevices(ctx); err != nil {
		return err
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

