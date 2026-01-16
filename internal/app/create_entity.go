package app

import (
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea/v2"

	"github.com/kluzzebass/lazyhue/internal/hueclient"
	"github.com/kluzzebass/lazyhue/internal/ui/component"
)

// startCreateRoom initiates room creation using the modal.
func (m *Model) startCreateRoom() tea.Cmd {
	// Close help if showing
	m.showHelp = false

	// Find the bridge to create the room in
	bridgeID := m.findBridgeForCurrentContext()
	if bridgeID == "" {
		m.status = "No bridge available"
		return nil
	}

	// Set creation state
	m.creatingRoom = true
	m.creatingZone = false
	m.createBridgeID = bridgeID

	// Activate the create modal
	component.BlurAll(m.componentRoot)
	m.createModal.SetActive(true)
	m.createModal.Input().SetValue("")
	m.createModal.Input().Focus()

	// Switch to detail panel to show the modal
	m.previousPane = m.focusedPane
	m.focusedPane = PanelDetail

	m.status = "Enter room name..."
	m.updateDetailContent()

	return nil
}

// startCreateZone initiates zone creation using the modal.
func (m *Model) startCreateZone() tea.Cmd {
	// Close help if showing
	m.showHelp = false

	// Find the bridge to create the zone in
	bridgeID := m.findBridgeForCurrentContext()
	if bridgeID == "" {
		m.status = "No bridge available"
		return nil
	}

	// Set creation state
	m.creatingRoom = false
	m.creatingZone = true
	m.createBridgeID = bridgeID

	// Activate the create modal
	component.BlurAll(m.componentRoot)
	m.createModal.SetActive(true)
	m.createModal.Input().SetValue("")
	m.createModal.Input().Focus()

	// Switch to detail panel to show the modal
	m.previousPane = m.focusedPane
	m.focusedPane = PanelDetail

	m.status = "Enter zone name..."
	m.updateDetailContent()

	return nil
}

// cancelCreate cancels room/zone creation.
func (m *Model) cancelCreate() {
	m.creatingRoom = false
	m.creatingZone = false
	m.createBridgeID = ""
	m.createModal.SetActive(false)
	m.status = "Creation cancelled"
}

// confirmCreateWithValue performs the actual room/zone creation with the given name.
func (m *Model) confirmCreateWithValue(name string) tea.Cmd {
	if name == "" {
		m.status = "Name is required"
		return nil
	}

	// Get the bridge
	bridge := m.manager.GetBridge(m.createBridgeID)
	if bridge == nil {
		m.status = "Bridge not found"
		m.cancelCreate()
		return nil
	}

	// Call the appropriate create function
	var err error
	var entityType string
	if m.creatingRoom {
		entityType = "Room"
		err = bridge.CreateRoom(name, hueclient.RoomArchetypeOther, nil)
	} else if m.creatingZone {
		entityType = "Zone"
		err = bridge.CreateZone(name, hueclient.RoomArchetypeOther, nil)
	}

	if err != nil {
		m.status = fmt.Sprintf("Create failed: %v", err)
	} else {
		m.status = fmt.Sprintf("%s \"%s\" created", entityType, name)
	}

	// Clear creation state
	m.creatingRoom = false
	m.creatingZone = false
	m.createBridgeID = ""
	m.createModal.SetActive(false)

	// Rebuild tree to reflect creation
	m.rebuildTreeForActiveTab()

	// Log the creation
	m.activities = append(m.activities, &RequestActivity{
		timestamp: time.Now(),
		message:   fmt.Sprintf("%s \"%s\" created", entityType, name),
	})
	m.updateLogContent()

	return nil
}

// findBridgeForCurrentContext returns the bridge ID for the current selection context.
func (m *Model) findBridgeForCurrentContext() string {
	// First try to find from selected item
	if item := m.tree.SelectedItem(); item != nil {
		if bridgeID := m.findBridgeForEntity(item); bridgeID != "" {
			return bridgeID
		}
	}

	// Fall back to first connected bridge
	bridges := m.manager.AllBridges()
	for _, bridge := range bridges {
		if bridge.IsConnected() {
			return bridge.Info.ID
		}
	}

	return ""
}

// startBridgePairing starts the bridge pairing process.
func (m *Model) startBridgePairing() tea.Cmd {
	// Close help if showing
	m.showHelp = false

	// Set flag that user requested pairing
	m.pairingRequested = true

	// Start discovery - when it completes, it will check pairingRequested flag
	m.status = "Discovering bridges..."

	return discoverBridges()
}

// cancelPairing cancels the current pairing operation.
func (m *Model) cancelPairing() {
	if m.pairingCancel != nil {
		m.pairingCancel()
	}
	m.pairing = false
	m.pairingFor = nil
	m.pairingCancel = nil
	m.status = "Pairing cancelled"
}
