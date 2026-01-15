package app

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea/v2"

	"github.com/kluzzebass/lazyhue/internal/hueclient"
	"github.com/kluzzebass/lazyhue/internal/ui/component/field"
	"github.com/kluzzebass/lazyhue/internal/ui/panels"
)

// startRenameMode initiates rename mode for the selected entity.
// This focuses the name field in the detail panel and starts editing it.
func (m *Model) startRenameMode() tea.Cmd {
	// Close help if showing
	m.showHelp = false

	// Get the selected entity from the tree
	item := m.tree.SelectedItem()
	if item == nil {
		m.status = "No entity selected"
		return nil
	}

	// Find which bridge this entity belongs to
	bridgeID := m.findBridgeForEntity(item)
	if bridgeID == "" {
		m.status = "Could not find bridge for entity"
		return nil
	}

	// Determine the name field ID based on entity type
	var nameFieldID string

	switch item.Type {
	case panels.EntityBridge:
		// Bridges are renamed via their bridge device
		bridge := m.manager.GetBridge(bridgeID)
		if bridge == nil {
			m.status = "Bridge not found"
			return nil
		}
		state := bridge.GetState()
		if state == nil {
			m.status = "Bridge state not available"
			return nil
		}
		bridgeDevice, ok := state.GetBridgeDevice()
		if !ok || bridgeDevice.Id == nil {
			m.status = "Bridge device not found"
			return nil
		}
		nameFieldID = "bridge-name:" + *bridgeDevice.Id
	case panels.EntityRoom:
		nameFieldID = "room-name:" + item.ID
	case panels.EntityZone:
		nameFieldID = "zone-name:" + item.ID
	case panels.EntityScene:
		nameFieldID = "scene-name:" + item.ID
	case panels.EntityDevice:
		nameFieldID = "name:" + item.ID
	case panels.EntityLight:
		// Lights are renamed via their owning device - find the device ID
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
		// Lights use device name field with format "name:<deviceID>"
		nameFieldID = "name:" + *light.Owner.Rid
	default:
		m.status = fmt.Sprintf("Cannot rename %s", item.Type.String())
		return nil
	}

	// Switch to detail panel and update content
	m.previousPane = m.focusedPane
	m.focusedPane = PanelDetail
	m.updateDetailContent()

	// Focus the name field in the grid and start editing
	var cmd tea.Cmd
	if m.lightGrid.FocusByFieldID(nameFieldID) {
		// Get the component and start editing if it's a text field
		if comp := m.lightGrid.GetComponentByID(nameFieldID); comp != nil {
			if textComp, ok := comp.(*field.TextComponent); ok {
				cmd = textComp.StartEditing()
			}
		}
		// Re-render the grid to show edit mode
		var content strings.Builder
		if m.lightGrid.RowCount() > 0 {
			content.WriteString(m.lightGrid.View())
		}
		m.detailViewport.SetContent(content.String())
		// Scroll viewport to show the focused field
		m.scrollDetailViewportToFocusedRow()
		m.status = "Edit name and press Enter to confirm, Esc to cancel"
	} else {
		m.status = "Name field not found"
	}

	return cmd
}

// findBridgeForEntity finds the bridge ID that owns the given entity.
func (m *Model) findBridgeForEntity(item *panels.EntityItem) string {
	// Return the bridge ID stored in the entity item
	return item.BridgeID
}

// cancelRename cancels rename mode and restores state.
func (m *Model) cancelRename() {
	m.renaming = false
	m.renameEntityID = ""
	m.renameEntityType = 0
	m.renameBridgeID = ""
	m.renameOriginalName = ""
	m.renameModal.SetActive(false) // Deactivate modal to release input capture
	m.status = "Rename cancelled"
}

// confirmRenameWithValue saves the new name and exits rename mode.
// This is called via TextInputConfirmedMsg from the rename modal.
func (m *Model) confirmRenameWithValue(newName string) tea.Cmd {
	if newName == "" {
		m.status = "Name cannot be empty"
		return nil
	}

	if newName == m.renameOriginalName {
		// No change
		m.cancelRename()
		m.status = "No change"
		return nil
	}

	// Get the bridge
	bridge := m.manager.GetBridge(m.renameBridgeID)
	if bridge == nil {
		m.status = "Bridge not found"
		m.cancelRename()
		return nil
	}

	// Call the appropriate rename function
	var err error
	switch m.renameEntityType {
	case panels.EntityDevice:
		err = bridge.RenameDevice(m.renameEntityID, newName)
	case panels.EntityRoom:
		err = bridge.RenameRoom(m.renameEntityID, newName)
	case panels.EntityZone:
		err = bridge.RenameZone(m.renameEntityID, newName)
	case panels.EntityScene:
		err = bridge.RenameScene(m.renameEntityID, newName)
	default:
		m.status = fmt.Sprintf("Cannot rename %s", m.renameEntityType.String())
		m.cancelRename()
		return nil
	}

	if err != nil {
		m.status = fmt.Sprintf("Rename failed: %v", err)
	} else {
		m.status = fmt.Sprintf("Renamed to \"%s\"", newName)
	}

	// Exit rename mode
	m.renaming = false
	m.renameEntityID = ""
	m.renameEntityType = 0
	m.renameBridgeID = ""
	m.renameOriginalName = ""
	m.renameModal.SetActive(false) // Deactivate modal to release input capture

	return nil
}
