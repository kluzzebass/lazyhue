package hue

import (
	"context"
	"sync"

	"github.com/kluzzebass/lazyhue/internal/config"
)

// Manager coordinates multiple bridge connections.
type Manager struct {
	bridges     map[string]*Bridge
	credentials *config.CredentialStore
	mu          sync.RWMutex
}

// NewManager creates a bridge manager with the given credentials.
func NewManager(creds *config.CredentialStore) *Manager {
	return &Manager{
		bridges:     make(map[string]*Bridge),
		credentials: creds,
	}
}

// AddBridge adds a bridge to the manager.
func (m *Manager) AddBridge(info BridgeInfo) *Bridge {
	m.mu.Lock()
	defer m.mu.Unlock()

	bridge := NewBridge(info)
	m.bridges[info.ID] = bridge

	return bridge
}

// GetBridge returns a bridge by ID.
func (m *Manager) GetBridge(id string) *Bridge {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.bridges[id]
}

// AllBridges returns all managed bridges.
func (m *Manager) AllBridges() []*Bridge {
	m.mu.RLock()
	defer m.mu.RUnlock()

	bridges := make([]*Bridge, 0, len(m.bridges))
	for _, b := range m.bridges {
		bridges = append(bridges, b)
	}
	return bridges
}

// ConnectedBridges returns only connected bridges.
func (m *Manager) ConnectedBridges() []*Bridge {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var connected []*Bridge
	for _, b := range m.bridges {
		if b.IsConnected() {
			connected = append(connected, b)
		}
	}
	return connected
}

// ConnectAll attempts to connect all bridges using stored credentials.
func (m *Manager) ConnectAll(ctx context.Context) error {
	m.mu.RLock()
	bridges := make([]*Bridge, 0, len(m.bridges))
	for _, b := range m.bridges {
		bridges = append(bridges, b)
	}
	m.mu.RUnlock()

	for _, bridge := range bridges {
		cred, ok := m.credentials.Get(bridge.Info.ID)
		if !ok {
			continue
		}

		if err := bridge.Connect(cred.ApiKey); err != nil {
			// Continue trying other bridges
			continue
		}

		// Update last used
		m.credentials.UpdateLastUsed(bridge.Info.ID)
	}

	return nil
}

// SyncAll refreshes state for all connected bridges.
func (m *Manager) SyncAll(ctx context.Context) map[string]error {
	bridges := m.ConnectedBridges()
	errors := make(map[string]error)

	var wg sync.WaitGroup
	var errMu sync.Mutex

	for _, bridge := range bridges {
		wg.Add(1)
		go func(b *Bridge) {
			defer wg.Done()
			if err := b.SyncAllBulk(ctx); err != nil {
				errMu.Lock()
				errors[b.Info.ID] = err
				errMu.Unlock()
			}
		}(bridge)
	}

	wg.Wait()
	return errors
}

// BridgeCount returns the number of managed bridges.
func (m *Manager) BridgeCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.bridges)
}

// RemoveBridge removes a bridge from the manager and its credentials.
// Returns true if the bridge was found and removed.
func (m *Manager) RemoveBridge(id string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if bridge exists
	bridge, exists := m.bridges[id]
	if !exists {
		return false
	}

	// Disconnect the bridge if it's connected
	if bridge.IsConnected() {
		bridge.Disconnect()
	}

	// Remove from bridges map
	delete(m.bridges, id)

	// Remove credentials
	m.credentials.Delete(id)

	return true
}

