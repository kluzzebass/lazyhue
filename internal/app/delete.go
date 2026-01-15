package app

import (
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea/v2"

	"github.com/kluzzebass/lazyhue/internal/hueclient"
	"github.com/kluzzebass/lazyhue/internal/ui/panels"
)

// startBridgeDeleteConfirmation initiates bridge deletion confirmation.
func (m *Model) startBridgeDeleteConfirmation() tea.Cmd {
	// Close help if showing
	m.showHelp = false

	// Get the selected item from tree - must be a bridge
	item := m.tree.SelectedItem()
	if item == nil || item.Type != panels.EntityBridge {
		m.status = "No bridge selected"
		return nil
	}

	bridgeID := item.ID
	bridge := m.manager.GetBridge(bridgeID)
	if bridge == nil {
		m.status = "Bridge not found"
		return nil
	}

	// Set confirmation state
	m.confirmingDelete = true
	m.deleteBridgeID = bridgeID
	m.deleteBridgeName = bridge.Info.Name

	// Switch to detail panel to show confirmation
	m.previousPane = m.focusedPane
	m.focusedPane = PanelDetail

	m.status = "Confirm bridge deletion..."

	// Update UI to show confirmation dialog
	m.updateDetailContent()

	return nil
}

// cancelBridgeDelete cancels the bridge deletion.
func (m *Model) cancelBridgeDelete() {
	m.confirmingDelete = false
	m.deleteBridgeID = ""
	m.deleteBridgeName = ""
	m.status = "Bridge deletion cancelled"
}

// confirmBridgeDelete performs the actual bridge deletion.
func (m *Model) confirmBridgeDelete() tea.Cmd {
	if m.deleteBridgeID == "" {
		m.status = "No bridge to delete"
		m.confirmingDelete = false
		return nil
	}

	// Remove the bridge from the manager (also removes credentials)
	if !m.manager.RemoveBridge(m.deleteBridgeID) {
		m.status = "Failed to remove bridge"
		m.confirmingDelete = false
		return nil
	}

	// Save credentials after deletion
	if err := m.credentials.Save(); err != nil {
		m.status = fmt.Sprintf("Bridge removed but failed to save credentials: %v", err)
	} else {
		m.status = fmt.Sprintf("Bridge \"%s\" deleted", m.deleteBridgeName)
	}

	// Clear confirmation state
	bridgeName := m.deleteBridgeName
	m.confirmingDelete = false
	m.deleteBridgeID = ""
	m.deleteBridgeName = ""

	// Rebuild tree to reflect deletion
	m.rebuildTreeForActiveTab()

	// Log the deletion
	m.activities = append(m.activities, &RequestActivity{
		timestamp: time.Now(),
		message:   fmt.Sprintf("Bridge \"%s\" deleted", bridgeName),
	})
	m.updateLogContent()

	return nil
}

// startEntityDeleteConfirmation initiates entity deletion confirmation.
func (m *Model) startEntityDeleteConfirmation() tea.Cmd {
	// Close help if showing
	m.showHelp = false

	// Get the selected item from tree
	item := m.tree.SelectedItem()
	if item == nil {
		m.status = "No entity selected"
		return nil
	}

	// Only certain entity types can be deleted
	switch item.Type {
	case panels.EntityRoom, panels.EntityZone, panels.EntityScene, panels.EntitySmartScene, panels.EntityDevice, panels.EntityLight:
		// These can be deleted
	default:
		m.status = fmt.Sprintf("%s cannot be deleted", item.Type.String())
		return nil
	}

	// Find which bridge this entity belongs to
	bridgeID := m.findBridgeForEntity(item)
	if bridgeID == "" {
		m.status = "Could not find bridge for entity"
		return nil
	}

	// For lights, we need to find and delete the parent device
	deleteID := item.ID
	if item.Type == panels.EntityLight {
		var light hueclient.LightGet
		var hasLight bool
		if item.RawPtr != nil {
			if l, ok := item.RawPtr.(hueclient.LightGet); ok {
				light = l
				hasLight = true
			}
		}
		if !hasLight {
			bridge := m.manager.GetBridge(bridgeID)
			if bridge != nil {
				if state := bridge.GetState(); state != nil {
					light, hasLight = state.GetLight(item.ID)
				}
			}
		}
		if !hasLight {
			m.status = "Light not found"
			return nil
		}
		if light.Owner == nil || light.Owner.Rid == nil {
			m.status = "Light has no owning device"
			return nil
		}
		// Store the device ID for deletion
		deleteID = *light.Owner.Rid
	}

	// Set confirmation state
	m.confirmingDeleteEntity = true
	m.deleteEntityID = deleteID
	m.deleteEntityName = item.Name
	m.deleteEntityType = item.Type
	m.deleteEntityBridgeID = bridgeID

	// Switch to detail panel to show confirmation
	m.previousPane = m.focusedPane
	m.focusedPane = PanelDetail

	m.status = fmt.Sprintf("Confirm %s deletion...", item.Type.String())

	// Update UI to show confirmation dialog
	m.updateDetailContent()

	return nil
}

// cancelEntityDelete cancels the entity deletion.
func (m *Model) cancelEntityDelete() {
	m.confirmingDeleteEntity = false
	m.deleteEntityID = ""
	m.deleteEntityName = ""
	m.deleteEntityType = 0
	m.deleteEntityBridgeID = ""
	m.status = "Deletion cancelled"
}

// confirmEntityDelete performs the actual entity deletion.
func (m *Model) confirmEntityDelete() tea.Cmd {
	if m.deleteEntityID == "" {
		m.status = "No entity to delete"
		m.confirmingDeleteEntity = false
		return nil
	}

	// Get the bridge
	bridge := m.manager.GetBridge(m.deleteEntityBridgeID)
	if bridge == nil {
		m.status = "Bridge not found"
		m.confirmingDeleteEntity = false
		return nil
	}

	// Call the appropriate delete function
	var err error
	switch m.deleteEntityType {
	case panels.EntityRoom:
		err = bridge.DeleteRoom(m.deleteEntityID)
	case panels.EntityZone:
		err = bridge.DeleteZone(m.deleteEntityID)
	case panels.EntityScene:
		err = bridge.DeleteScene(m.deleteEntityID)
	case panels.EntitySmartScene:
		err = bridge.DeleteSmartScene(m.deleteEntityID)
	case panels.EntityDevice, panels.EntityLight:
		// For lights, deleteEntityID contains the parent device ID
		err = bridge.DeleteDevice(m.deleteEntityID)
	default:
		m.status = fmt.Sprintf("Cannot delete %s", m.deleteEntityType.String())
		m.confirmingDeleteEntity = false
		return nil
	}

	if err != nil {
		m.status = fmt.Sprintf("Delete failed: %v", err)
	} else {
		m.status = fmt.Sprintf("%s \"%s\" deleted", m.deleteEntityType.String(), m.deleteEntityName)
	}

	// Clear confirmation state
	entityName := m.deleteEntityName
	entityType := m.deleteEntityType
	m.confirmingDeleteEntity = false
	m.deleteEntityID = ""
	m.deleteEntityName = ""
	m.deleteEntityType = 0
	m.deleteEntityBridgeID = ""

	// Rebuild tree to reflect deletion
	m.rebuildTreeForActiveTab()

	// Log the deletion
	m.activities = append(m.activities, &RequestActivity{
		timestamp: time.Now(),
		message:   fmt.Sprintf("%s \"%s\" deleted", entityType.String(), entityName),
	})

	return nil
}
