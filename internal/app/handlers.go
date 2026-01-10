package app

import (
	"context"
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/kluzzebass/lazyhue/internal/config"
	"github.com/kluzzebass/lazyhue/internal/hue"
	"github.com/kluzzebass/lazyhue/internal/ui"
	"github.com/kluzzebass/lazyhue/internal/ui/panels"
)

func (m *Model) handleKey(msg tea.KeyMsg) tea.Cmd {
	defer m.updateStatusContext()

	keyStr := msg.String()

	// Handle help overlay first - it captures all input when visible
	if m.helpPanel.IsVisible() {
		if keyStr == "?" || keyStr == "esc" || keyStr == "q" {
			m.helpPanel.Hide()
			m.statusBar.ClearPopupHints()
			return nil
		}
		// Pass navigation keys to help panel
		return m.helpPanel.Update(msg)
	}

	// Check for panel focus shortcuts (number keys)
	if keyStr == "0" {
		if m.focusIndex >= 0 {
			m.lastFocusIndex = m.focusIndex
		}
		m.focusIndex = -1 // Detail panel
		return nil
	}
	if keyStr == "3" && m.logPanelVisible {
		if m.focusIndex >= 0 {
			m.lastFocusIndex = m.focusIndex
		}
		m.focusIndex = -2 // Log panel
		return nil
	}
	for i, id := range m.panelOrder {
		if panel := m.panelMap[id]; panel != nil {
			if keyStr == panel.Key() {
				m.focusIndex = i
				m.syncSelectionFromFocusedPanel()
				return nil
			}
		}
	}

	// Check focused panel's bindings first
	panelID := m.focusedPanelID()
	if bindings, ok := m.panelBindings[panelID]; ok {
		for _, b := range bindings {
			if b.Matches(keyStr) && b.Action != ui.ActionNone {
				return m.dispatch(b.Action)
			}
		}
	}

	// Check global bindings
	for _, b := range m.globalBindings {
		if b.Matches(keyStr) && b.Action != ui.ActionNone {
			return m.dispatch(b.Action)
		}
	}

	return nil
}

func (m *Model) handleSelect() tea.Cmd {
	// Bridge panel: Enter on a bridge triggers pairing if not connected
	if m.focusedPanelID() == PanelIDBridges {
		if bridge := m.bridgePanel().SelectedBridge(); bridge != nil {
			if !bridge.IsConnected() {
				// Clear hierarchy when selecting a disconnected bridge
				m.hierarchyPanel().Clear()
				// Trigger pairing for unconnected bridge
				return m.startBridgePairing()
			}
			// For connected bridges, just switch to hierarchy
			m.manager.SetActiveBridge(bridge.Info.ID)
			m.updateBridgePanel()
			m.refreshAllPanels()
			if m.focusIndex < len(m.panelOrder)-1 {
				m.focusIndex++
			}
		}
		return nil
	}

	// Hierarchy panel: toggle group or activate scene
	if m.focusedPanelID() == PanelIDHierarchy {
		hp := m.hierarchyPanel()
		if hp.IsGroupSelected() {
			hp.ToggleSelected()
			return nil
		}
	}

	// Activate scene if one is selected
	if m.selectedItem != nil && m.selectedItem.Type == panels.EntityScene {
		bridge := m.manager.GetActiveBridge()
		if bridge != nil {
			return recallScene(bridge, m.selectedItem.ID)
		}
	}
	return nil
}

func (m *Model) handleMouse(msg tea.MouseMsg) tea.Cmd {
	defer m.updateStatusContext()

	// Use layout tree to find clicked panel
	leaf := m.layoutTree.At(msg.X, msg.Y)
	if leaf == nil {
		return nil
	}

	// Get panel bounds for relative coordinate calculation
	bounds := m.layoutTree.Bounds(leaf.ID)
	relX := msg.X - bounds.X
	relY := msg.Y - bounds.Y

	// Handle panel focus and item selection on click
	if msg.Action == tea.MouseActionPress && msg.Button == tea.MouseButtonLeft {
		switch leaf.ID {
		case PanelIDDetail:
			if m.focusIndex >= 0 {
				m.lastFocusIndex = m.focusIndex
			}
			m.focusIndex = -1
		case PanelIDLog:
			if m.logPanelVisible {
				if m.focusIndex >= 0 {
					m.lastFocusIndex = m.focusIndex
				}
				m.focusIndex = -2
			}
		default:
			for i, id := range m.panelOrder {
				if id == leaf.ID {
					m.focusIndex = i
					break
				}
			}
		}

		// Pass relative coordinates to panels for item selection
		switch leaf.ID {
		case PanelIDBridges:
			m.bridgePanel().HandleClick(relX, relY)
		case PanelIDHierarchy:
			m.hierarchyPanel().HandleClick(relX, relY)
		}

		m.syncSelectionFromFocusedPanel()
	}

	// Pass scroll events to panel under cursor
	if msg.Button == tea.MouseButtonWheelUp || msg.Button == tea.MouseButtonWheelDown {
		switch leaf.ID {
		case PanelIDDetail:
			m.detailPanel.Update(msg)
		case PanelIDLog:
			m.logPanel.Update(msg)
		case PanelIDBridges:
			m.bridgePanel().Update(msg)
			m.syncSelectionFromFocusedPanel()
		case PanelIDHierarchy:
			m.hierarchyPanel().Update(msg)
			m.syncSelectionFromFocusedPanel()
		}
	}

	return nil
}

func (m *Model) handleBridgesDiscovered(msg BridgesDiscoveredMsg) {
	if len(msg.Bridges) == 0 {
		if m.manager.BridgeCount() == 0 {
			m.setStatus("No bridges found. Press P to pair.", false)
		}
		return
	}

	for _, info := range msg.Bridges {
		// Check if we already have this bridge by ID
		existing := m.manager.GetBridge(info.ID)
		if existing != nil {
			needsSave := false

			// Update IP if changed
			if existing.Info.IPAddress != info.IPAddress {
				existing.Info.IPAddress = info.IPAddress
				needsSave = true
			}

			// Update name if changed
			if info.Name != "" && existing.Info.Name != info.Name {
				existing.Info.Name = info.Name
				needsSave = true
			}

			// Persist changes to credentials
			if needsSave {
				if cred, ok := m.credentials.Get(info.ID); ok {
					cred.IPAddress = existing.Info.IPAddress
					cred.Name = existing.Info.Name
					m.credentials.Set(cred)
					_ = m.credentials.Save()
				}
			}
			continue
		}

		// Also check by IP to avoid duplicates from discovery vs stored credentials
		existsByIP := false
		for _, bridge := range m.manager.AllBridges() {
			if bridge.Info.IPAddress == info.IPAddress {
				existsByIP = true
				break
			}
		}
		if existsByIP {
			continue
		}

		m.manager.AddBridge(info)
	}
}

func (m *Model) handlePairingSuccess(msg PairingSuccessMsg) {
	bridge := m.manager.GetBridge(msg.BridgeID)
	if bridge == nil {
		return
	}

	m.credentials.Set(config.BridgeCredential{
		BridgeID:  msg.BridgeID,
		Name:      bridge.Info.Name,
		IPAddress: bridge.Info.IPAddress,
		ApiKey:    msg.ApiKey,
	})
	_ = m.credentials.Save()
	m.setStatus("Pairing successful!", false)
}

func (m *Model) startBridgePairing() tea.Cmd {
	if m.pairing {
		return nil
	}

	var bridgeInfo hue.BridgeInfo
	if m.focusedPanelID() == PanelIDBridges {
		if selected := m.bridgePanel().SelectedBridge(); selected != nil {
			if selected.IsConnected() {
				m.setStatus("Bridge already connected", false)
				return nil
			}
			bridgeInfo = selected.Info
		}
	}

	if bridgeInfo.ID == "" {
		for _, bridge := range m.manager.AllBridges() {
			if !bridge.IsConnected() {
				bridgeInfo = bridge.Info
				break
			}
		}
	}

	if bridgeInfo.ID == "" {
		m.setStatus("No unpaired bridge found. Discovering...", false)
		return discoverBridges()
	}

	// Cancel any existing pairing attempt
	if m.pairingCancel != nil {
		m.pairingCancel()
	}

	// Create a new context with 60s timeout for this pairing attempt
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	m.pairingCancel = cancel

	m.pairing = true
	m.pairingFor = &bridgeInfo
	bridge := m.manager.GetBridge(bridgeInfo.ID)
	if bridge != nil {
		bridge.Status = hue.StatusPairing
		m.updateBridgePanel()
	}

	// Use the bridge name, falling back to IP if empty
	displayName := bridgeInfo.Name
	if displayName == "" {
		displayName = bridgeInfo.IPAddress
	}

	// Show pairing popup
	m.pairingPanel.Show(bridgeInfo.ID, displayName)

	// Start pairing and countdown tick
	return tea.Batch(
		startPairing(ctx, bridgeInfo),
		pairingTick(),
	)
}

// pairingTick returns a command that ticks every second for the pairing countdown.
func pairingTick() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return PairingTickMsg{}
	})
}

// forgetSelectedBridge shows a confirmation dialog before removing bridge credentials.
func (m *Model) forgetSelectedBridge() tea.Cmd {
	bridge := m.bridgePanel().SelectedBridge()
	if bridge == nil {
		return nil
	}

	// Only connected bridges can be forgotten (they have saved credentials)
	if !bridge.IsConnected() {
		m.setStatus("Bridge is not connected", false)
		return nil
	}

	bridgeID := bridge.Info.ID
	bridgeName := bridge.Info.Name

	// Show confirmation dialog
	m.popupPanel.SetRatio(0.5, 0.25)
	m.popupPanel.ShowConfirm(
		"Forget Bridge",
		"Are you sure you want to forget \""+bridgeName+"\"?",
		func(result panels.PopupResult) {
			if result.Confirmed {
				m.doForgetBridge(bridgeID, bridgeName)
			}
		},
	)

	return nil
}

// doForgetBridge actually removes the bridge credentials.
func (m *Model) doForgetBridge(bridgeID, bridgeName string) {
	bridge := m.manager.GetBridge(bridgeID)
	if bridge == nil {
		return
	}

	// Remove from credential store
	m.credentials.Delete(bridgeID)
	if err := m.credentials.Save(); err != nil {
		m.setStatus("Failed to save credentials: "+err.Error(), true)
		return
	}

	// Disconnect the bridge
	bridge.Disconnect()

	// Clear the hierarchy (this bridge's data is no longer valid)
	m.hierarchyPanel().Clear()

	m.updateBridgePanel()
	m.setStatus("Forgot bridge: "+bridgeName, false)
}

// showDeviceEditPopup shows a configuration popup for the selected device.
func (m *Model) showDeviceEditPopup() tea.Cmd {
	bridge := m.manager.GetActiveBridge()
	if bridge == nil || m.selectedItem == nil {
		return nil
	}

	if m.selectedItem.Type != panels.EntityDevice {
		m.setStatus("Select a device to edit", false)
		return nil
	}

	device, ok := panels.GetDeviceFromItem(*m.selectedItem)
	if !ok {
		return nil
	}

	deviceName := m.selectedItem.Name
	state := bridge.GetState()

	// Check if this device has a motion sensor
	motionID, motion, hasMotion := state.GetDeviceMotionSensor(device)
	if !hasMotion {
		m.setStatus("No editable features for this device", false)
		return nil
	}

	// Build form fields for motion sensor
	fields := []panels.FormField{}

	// Enabled toggle
	enabled := 0
	if motion.Enabled != nil && *motion.Enabled {
		enabled = 1
	}
	fields = append(fields, panels.FormField{
		ID:    "enabled",
		Label: "Motion sensor",
		Type:  panels.FormFieldToggle,
		Value: enabled,
	})

	// Sensitivity slider
	if motion.Sensitivity != nil && motion.Sensitivity.SensitivityMax != nil {
		currentSens := 0
		if motion.Sensitivity.Sensitivity != nil {
			currentSens = *motion.Sensitivity.Sensitivity
		}
		fields = append(fields, panels.FormField{
			ID:    "sensitivity",
			Label: "Sensitivity",
			Type:  panels.FormFieldSlider,
			Value: currentSens,
			Min:   0,
			Max:   *motion.Sensitivity.SensitivityMax,
		})
	}

	// Show the form popup
	m.popupPanel.SetRatio(0.45, 0.35)
	m.popupPanel.ShowForm("Configure: "+deviceName, fields, func(result panels.PopupResult, finalFields []panels.FormField) {
		if !result.Confirmed {
			return
		}

		// Apply changes for fields that were modified
		for _, field := range finalFields {
			if field.Value == field.Original {
				continue // No change
			}

			switch field.ID {
			case "enabled":
				newEnabled := field.Value != 0
				if err := bridge.SetMotionSensorEnabled(motionID, newEnabled); err != nil {
					m.setStatus("Failed to update sensor: "+err.Error(), true)
				} else {
					action := "sensor disabled"
					if newEnabled {
						action = "sensor enabled"
					}
					m.logPanel.AddEntry("request", deviceName+": "+action)
				}
			case "sensitivity":
				if err := bridge.SetMotionSensorSensitivity(motionID, field.Value); err != nil {
					m.setStatus("Failed to update sensitivity: "+err.Error(), true)
				} else {
					m.logPanel.AddEntry("request", fmt.Sprintf("%s: sensitivity %d", deviceName, field.Value))
				}
			}
		}

		m.setStatus("Settings saved", false)
		m.refreshHierarchyPanel()
		m.updateDetailPanel()
	})

	return nil
}

// showRenamePopup shows an input popup to rename the selected entity.
func (m *Model) showRenamePopup() tea.Cmd {
	bridge := m.manager.GetActiveBridge()
	if bridge == nil || m.selectedItem == nil {
		return nil
	}

	currentName := m.selectedItem.Name
	entityType := ""
	canRename := false

	switch m.selectedItem.Type {
	case panels.EntityDevice:
		entityType = "device"
		canRename = true
	case panels.EntityRoom:
		entityType = "room"
		canRename = true
	case panels.EntityZone:
		entityType = "zone"
		canRename = true
	case panels.EntityScene:
		entityType = "scene"
		canRename = true
	case panels.EntityLight:
		// Lights are renamed via their parent device
		m.setStatus("Rename the device to rename this light", false)
		return nil
	default:
		m.setStatus("This item cannot be renamed", false)
		return nil
	}

	if !canRename {
		return nil
	}

	itemID := m.selectedItem.ID
	itemType := m.selectedItem.Type

	fields := []panels.FormField{
		{
			ID:           "name",
			Label:        "Name",
			Type:         panels.FormFieldText,
			TextValue:    currentName,
			OriginalText: currentName,
		},
	}

	m.popupPanel.SetRatio(0.5, 0.25)
	m.popupPanel.ShowForm("Rename "+entityType, fields, func(result panels.PopupResult, finalFields []panels.FormField) {
		if !result.Confirmed {
			return
		}

		// Get the new name from the form field
		newName := ""
		for _, f := range finalFields {
			if f.ID == "name" {
				newName = f.TextValue
				break
			}
		}

		if newName == "" || newName == currentName {
			return
		}

		var err error

		switch itemType {
		case panels.EntityDevice:
			err = bridge.RenameDevice(itemID, newName)
		case panels.EntityRoom:
			err = bridge.RenameRoom(itemID, newName)
		case panels.EntityZone:
			err = bridge.RenameZone(itemID, newName)
		case panels.EntityScene:
			err = bridge.RenameScene(itemID, newName)
		}

		if err != nil {
			m.setStatus("Failed to rename: "+err.Error(), true)
			return
		}

		m.logPanel.AddEntry("request", fmt.Sprintf("%s: renamed to \"%s\"", currentName, newName))
		m.setStatus(fmt.Sprintf("Renamed to \"%s\"", newName), false)

		// Trigger a full state sync to get the new name
		go func() {
			if err := bridge.SyncAll(context.Background()); err == nil {
				// The state will be updated via the normal sync mechanism
			}
		}()

		m.refreshHierarchyPanel()
		m.updateDetailPanel()
	})

	return nil
}

