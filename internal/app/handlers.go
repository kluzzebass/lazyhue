package app

import (
	"context"
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/kluzzebass/lazyhue/internal/config"
	"github.com/kluzzebass/lazyhue/internal/hue"
	"github.com/kluzzebass/lazyhue/internal/hueclient"
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

	// IMPORTANT: For detail panel, let the form handle navigation keys
	// before global bindings consume them. This is handled in Update(),
	// so we skip global bindings for navigation keys when detail is focused.
	if m.focusedOnDetail() {
		// Let navigation keys pass through to be handled by updateFocusedPanel
		switch keyStr {
		case "up", "down", "left", "right", "j", "k", "h", "l", "enter", " ", "esc":
			return nil // Will be handled by updateFocusedPanel in Update()
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

	// Open light controls if a light is selected
	if m.selectedItem != nil && m.selectedItem.Type == panels.EntityLight {
		return m.showLightEditPopup()
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
			// Pass click to detail panel with relative coordinates
			relMsg := tea.MouseMsg{
				X:      relX,
				Y:      relY,
				Type:   tea.MouseLeft,
				Button: msg.Button,
				Action: msg.Action,
			}
			_, cmd := m.detailPanel.Update(relMsg)
			if cmd != nil {
				return cmd
			}
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

	// Handle mouse motion/drag for detail panel
	if msg.Action == tea.MouseActionMotion && leaf.ID == PanelIDDetail {
		relMsg := tea.MouseMsg{
			X:      relX,
			Y:      relY,
			Type:   tea.MouseMotion,
			Button: msg.Button,
			Action: msg.Action,
		}
		m.detailPanel.Update(relMsg)
	}

	// Pass scroll events to panel under cursor
	if msg.Button == tea.MouseButtonWheelUp || msg.Button == tea.MouseButtonWheelDown {
		switch leaf.ID {
		case PanelIDDetail:
			// Pass scroll with relative coordinates
			relMsg := tea.MouseMsg{
				X:      relX,
				Y:      relY,
				Type:   msg.Type,
				Button: msg.Button,
				Action: msg.Action,
			}
			m.detailPanel.Update(relMsg)
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
	m.setStatusTemporary("Pairing successful!", false, 3*time.Second)
}

// handleDetailFieldSave handles saves from the details panel's editable fields.
func (m *Model) handleDetailFieldSave(entityType panels.EntityType, entityID string, fieldID string, value interface{}) {
	bridge := m.manager.GetActiveBridge()
	if bridge == nil {
		m.setStatus("No active bridge", true)
		return
	}

	var err error

	switch entityType {
	case panels.EntityLight:
		switch fieldID {
		case "on":
			if v, ok := value.(int); ok {
				err = bridge.SetLightOn(entityID, v != 0)
			}
		case "brightness":
			if v, ok := value.(int); ok {
				err = bridge.SetLightBrightness(entityID, float64(v))
			}
		case "colortemp":
			if v, ok := value.(int); ok {
				err = bridge.SetLightColorTemperature(entityID, v)
			}
		case "color":
			if v, ok := value.([2]float64); ok {
				err = bridge.SetLightColor(entityID, v[0], v[1])
			}
		case "effect":
			if v, ok := value.(int); ok {
				// Get the effect name from the options
				if light, ok := bridge.GetState().GetLight(entityID); ok {
					if light.Effects != nil && light.Effects.EffectValues != nil && v < len(*light.Effects.EffectValues) {
						effect := (*light.Effects.EffectValues)[v]
						err = bridge.SetLightEffect(entityID, effect)
					}
				}
			}
		}

	case panels.EntityRoom:
		switch fieldID {
		case "archetype":
			if v, ok := value.(int); ok {
				archetypes := hue.RoomArchetypeList()
				if v < len(archetypes) {
					archetype := hueclient.RoomArchetype(archetypes[v])
					err = bridge.SetRoomArchetype(entityID, archetype)
				}
			}
		}

	case panels.EntityZone:
		switch fieldID {
		case "archetype":
			if v, ok := value.(int); ok {
				archetypes := hue.RoomArchetypeList()
				if v < len(archetypes) {
					archetype := hueclient.RoomArchetype(archetypes[v])
					err = bridge.SetRoomArchetype(entityID, archetype)
				}
			}
		}
	}

	if err != nil {
		m.setStatus(fmt.Sprintf("Failed to update: %v", err), true)
	} else {
		// Refresh the display
		m.refreshHierarchyPanel()
	}
}

func (m *Model) startBridgePairing() tea.Cmd {
	if m.pairing {
		return nil
	}

	var bridgeInfo hue.BridgeInfo
	if m.focusedPanelID() == PanelIDBridges {
		if selected := m.bridgePanel().SelectedBridge(); selected != nil {
			if selected.IsConnected() {
				m.setStatusTemporary("Bridge already connected", false, 3*time.Second)
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
		m.setStatusTemporary("Bridge is not connected", false, 3*time.Second)
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
	m.setStatusTemporary("Forgot bridge: "+bridgeName, false, 3*time.Second)
}

// showDeviceEditPopup shows a configuration popup for the selected device.
func (m *Model) showDeviceEditPopup() tea.Cmd {
	bridge := m.manager.GetActiveBridge()
	if bridge == nil || m.selectedItem == nil {
		return nil
	}

	if m.selectedItem.Type != panels.EntityDevice {
		m.setStatusTemporary("Select a device to edit", false, 3*time.Second)
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
		m.setStatusTemporary("No editable features for this device", false, 3*time.Second)
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

		m.setStatusTemporary("Settings saved", false, 3*time.Second)
		m.refreshHierarchyPanel()
		m.updateDetailPanel()
	})

	return startBlinkTicker()
}

// showLightEditPopup shows a configuration popup for the selected light.
func (m *Model) showLightEditPopup() tea.Cmd {
	bridge := m.manager.GetActiveBridge()
	if bridge == nil || m.selectedItem == nil {
		return nil
	}

	if m.selectedItem.Type != panels.EntityLight {
		m.setStatusTemporary("Select a light to edit", false, 3*time.Second)
		return nil
	}

	light, ok := panels.GetLightFromItem(*m.selectedItem)
	if !ok {
		return nil
	}

	lightID := m.selectedItem.ID
	lightName := m.selectedItem.Name

	// Build form fields based on light capabilities
	fields := []panels.FormField{}

	// On/Off toggle
	isOn := 0
	if light.On != nil && light.On.On != nil && *light.On.On {
		isOn = 1
	}
	fields = append(fields, panels.FormField{
		ID:    "on",
		Label: "Power",
		Type:  panels.FormFieldToggle,
		Value: isOn,
	})

	// Brightness (if dimmable)
	if light.Dimming != nil {
		brightness := 0
		if light.Dimming.Brightness != nil {
			brightness = int(*light.Dimming.Brightness)
		}
		fields = append(fields, panels.FormField{
			ID:    "brightness",
			Label: "Brightness",
			Type:  panels.FormFieldBrightness,
			Value: brightness,
			Min:   1,
			Max:   100,
		})
	}

	// Color temperature (if supported)
	if light.ColorTemperature != nil && light.ColorTemperature.MirekSchema != nil {
		mirek := 366 // Default middle value
		if light.ColorTemperature.Mirek != nil {
			mirek = *light.ColorTemperature.Mirek
		}
		mirekMin := 153
		mirekMax := 500
		if light.ColorTemperature.MirekSchema.MirekMinimum != nil {
			mirekMin = *light.ColorTemperature.MirekSchema.MirekMinimum
		}
		if light.ColorTemperature.MirekSchema.MirekMaximum != nil {
			mirekMax = *light.ColorTemperature.MirekSchema.MirekMaximum
		}
		fields = append(fields, panels.FormField{
			ID:    "color_temp",
			Label: "Color temp",
			Type:  panels.FormFieldColorTemp,
			Value: mirek,
			Min:   mirekMin,
			Max:   mirekMax,
		})
	}

	// Color (if supported)
	if light.Color != nil && light.Color.Xy != nil {
		x, y := float64(0.3127), float64(0.3290) // Default to white
		if light.Color.Xy.X != nil {
			x = float64(*light.Color.Xy.X)
		}
		if light.Color.Xy.Y != nil {
			y = float64(*light.Color.Xy.Y)
		}
		fields = append(fields, panels.FormField{
			ID:     "color",
			Label:  "Color",
			Type:   panels.FormFieldColor,
			ColorX: x,
			ColorY: y,
		})
	}

	// Effects (if supported)
	if light.Effects != nil && light.Effects.EffectValues != nil && len(*light.Effects.EffectValues) > 0 {
		// Build options from available effects
		effectOptions := []panels.FormSelectOption{}
		currentEffect := 0

		for i, effect := range *light.Effects.EffectValues {
			effectOptions = append(effectOptions, panels.FormSelectOption{
				Label: hue.EffectDisplayName(string(effect)),
				Value: i,
			})
			// Check if this is the current effect
			if light.Effects.Status != nil && *light.Effects.Status == effect {
				currentEffect = i
			}
		}

		fields = append(fields, panels.FormField{
			ID:      "effect",
			Label:   "Effect",
			Type:    panels.FormFieldSelect,
			Value:   currentEffect,
			Options: effectOptions,
		})
	}

	// Calculate popup size based on fields
	height := 0.30 + float64(len(fields))*0.06
	if height > 0.7 {
		height = 0.7
	}
	m.popupPanel.SetRatio(0.50, height)

	// Track the light being edited for live sync
	m.editingLightID = lightID
	m.syncLivePopup = func() {
		// Re-read light state and update popup fields
		if currentLight, ok := bridge.GetState().GetLight(lightID); ok {
			// Update on/off
			isOn := 0
			if currentLight.On != nil && currentLight.On.On != nil && *currentLight.On.On {
				isOn = 1
			}
			m.popupPanel.UpdateFormField("on", isOn, 0, 0)

			// Update brightness
			if currentLight.Dimming != nil && currentLight.Dimming.Brightness != nil {
				m.popupPanel.UpdateFormField("brightness", int(*currentLight.Dimming.Brightness), 0, 0)
			}

			// Update color temperature
			if currentLight.ColorTemperature != nil && currentLight.ColorTemperature.Mirek != nil {
				m.popupPanel.UpdateFormField("color_temp", *currentLight.ColorTemperature.Mirek, 0, 0)
			}

			// Update color
			if currentLight.Color != nil && currentLight.Color.Xy != nil {
				x, y := float64(0), float64(0)
				if currentLight.Color.Xy.X != nil {
					x = float64(*currentLight.Color.Xy.X)
				}
				if currentLight.Color.Xy.Y != nil {
					y = float64(*currentLight.Color.Xy.Y)
				}
				m.popupPanel.UpdateFormField("color", 0, x, y)
			}

			// Update effect
			if currentLight.Effects != nil && currentLight.Effects.Status != nil && currentLight.Effects.EffectValues != nil {
				for i, effect := range *currentLight.Effects.EffectValues {
					if effect == *currentLight.Effects.Status {
						m.popupPanel.UpdateFormField("effect", i, 0, 0)
						break
					}
				}
			}
		}
	}

	// Show form in live mode - changes apply immediately
	m.popupPanel.ShowFormLive("Controls: "+lightName, fields,
		func(result panels.PopupResult, finalFields []panels.FormField) {
			// On close, clear live sync and refresh
			m.editingLightID = ""
			m.syncLivePopup = nil
			m.refreshHierarchyPanel()
			m.updateDetailPanel()
		},
		func(field panels.FormField) {
			// onChange callback - apply changes immediately and update UI
			switch field.ID {
			case "on":
				newOn := field.Value != 0
				if err := bridge.SetLightOn(lightID, newOn); err != nil {
					m.setStatus("Failed to update light: "+err.Error(), true)
				}
			case "brightness":
				if err := bridge.SetLightBrightness(lightID, float64(field.Value)); err != nil {
					m.setStatus("Failed to set brightness: "+err.Error(), true)
				}
			case "color_temp":
				if err := bridge.SetLightColorTemperature(lightID, field.Value); err != nil {
					m.setStatus("Failed to set color temp: "+err.Error(), true)
				}
			case "color":
				if err := bridge.SetLightColor(lightID, field.ColorX, field.ColorY); err != nil {
					m.setStatus("Failed to set color: "+err.Error(), true)
				}
			case "effect":
				// Get the effect from the options based on the Value (index)
				if currentLight, ok := bridge.GetState().GetLight(lightID); ok {
					if currentLight.Effects != nil && currentLight.Effects.EffectValues != nil {
						effects := *currentLight.Effects.EffectValues
						if field.Value >= 0 && field.Value < len(effects) {
							effect := effects[field.Value]
							if err := bridge.SetLightEffect(lightID, effect); err != nil {
								m.setStatus("Failed to set effect: "+err.Error(), true)
							}
						}
					}
				}
			}
			// Live update the UI panels
			m.refreshHierarchyPanel()
			m.updateDetailPanel()
		},
	)

	return startBlinkTicker()
}

// showRenamePopup shows an input popup to rename the selected entity.
func (m *Model) showRenamePopup() tea.Cmd {
	bridge := m.manager.GetActiveBridge()
	if bridge == nil || m.selectedItem == nil {
		return nil
	}

	currentName := m.selectedItem.Name
	itemID := m.selectedItem.ID
	itemType := m.selectedItem.Type
	entityType := ""

	switch m.selectedItem.Type {
	case panels.EntityBridge:
		// Bridges are renamed via their device
		state := bridge.GetState()
		bridgeDevice, ok := state.GetBridgeDevice()
		if !ok || bridgeDevice.Id == nil {
			m.setStatusTemporary("Cannot find bridge device", false, 3*time.Second)
			return nil
		}
		itemID = *bridgeDevice.Id
		itemType = panels.EntityDevice
		currentName = bridgeDevice.DeviceName("")
		entityType = "bridge"
	case panels.EntityDevice:
		entityType = "device"
	case panels.EntityRoom:
		entityType = "room"
	case panels.EntityZone:
		entityType = "zone"
	case panels.EntityScene:
		entityType = "scene"
	case panels.EntityLight:
		// Lights are renamed via their parent device - find and rename it
		light, ok := m.selectedItem.RawPtr.(hueclient.LightGet)
		if !ok || light.Owner == nil || light.Owner.Rid == nil {
			m.setStatusTemporary("Cannot find parent device", false, 3*time.Second)
			return nil
		}
		// Get the device and rename it instead
		state := bridge.GetState()
		device, ok := state.GetDevice(*light.Owner.Rid)
		if !ok {
			m.setStatusTemporary("Cannot find parent device", false, 3*time.Second)
			return nil
		}
		itemID = *light.Owner.Rid
		itemType = panels.EntityDevice
		currentName = device.DeviceName("")
		entityType = "device"
	default:
		m.setStatusTemporary("This item cannot be renamed", false, 3*time.Second)
		return nil
	}

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
		m.setStatusTemporary(fmt.Sprintf("Renamed to \"%s\"", newName), false, 3*time.Second)

		// Trigger a full state sync to get the new name
		go func() {
			if err := bridge.SyncAll(context.Background()); err == nil {
				// The state will be updated via the normal sync mechanism
			}
		}()

		m.refreshHierarchyPanel()
		m.updateDetailPanel()
	})

	return startBlinkTicker()
}

// showTestForm displays a demo form with all available field types.
func (m *Model) showTestForm() tea.Cmd {
	fields := []panels.FormField{
		{
			ID:    "toggle",
			Label: "Toggle",
			Type:  panels.FormFieldToggle,
			Value: 1,
		},
		{
			ID:    "slider",
			Label: "Slider",
			Type:  panels.FormFieldSlider,
			Value: 5,
			Min:   0,
			Max:   10,
		},
		{
			ID:    "brightness",
			Label: "Brightness",
			Type:  panels.FormFieldBrightness,
			Value: 75,
			Min:   0,
			Max:   100,
		},
		{
			ID:    "colortemp",
			Label: "Color Temp",
			Type:  panels.FormFieldColorTemp,
			Value: 326, // Mirek value (warm-ish)
			Min:   153, // Cool (6500K)
			Max:   500, // Warm (2000K)
		},
		{
			ID:     "color",
			Label:  "Color",
			Type:   panels.FormFieldColor,
			ColorX: 0.5,
			ColorY: 0.3,
		},
		{
			ID:    "select",
			Label: "Select",
			Type:  panels.FormFieldSelect,
			Value: 13, // M
			Options: []panels.FormSelectOption{
				{Label: "Alpha", Value: 1},
				{Label: "Bravo", Value: 2},
				{Label: "Charlie", Value: 3},
				{Label: "Delta", Value: 4},
				{Label: "Echo", Value: 5},
				{Label: "Foxtrot", Value: 6},
				{Label: "Golf", Value: 7},
				{Label: "Hotel", Value: 8},
				{Label: "India", Value: 9},
				{Label: "Juliet", Value: 10},
				{Label: "Kilo", Value: 11},
				{Label: "Lima", Value: 12},
				{Label: "Mike", Value: 13},
				{Label: "November", Value: 14},
				{Label: "Oscar", Value: 15},
				{Label: "Papa", Value: 16},
				{Label: "Quebec", Value: 17},
				{Label: "Romeo", Value: 18},
				{Label: "Sierra", Value: 19},
				{Label: "Tango", Value: 20},
				{Label: "Uniform", Value: 21},
				{Label: "Victor", Value: 22},
				{Label: "Whiskey", Value: 23},
				{Label: "X-ray", Value: 24},
				{Label: "Yankee", Value: 25},
				{Label: "Zulu", Value: 26},
			},
		},
		{
			ID:        "text",
			Label:     "Text",
			Type:      panels.FormFieldText,
			TextValue: "Hello World",
		},
		{
			ID:    "radio",
			Label: "Radio (horiz)",
			Type:  panels.FormFieldRadio,
			Value: 2,
			Options: []panels.FormSelectOption{
				{Label: "Small", Value: 1},
				{Label: "Medium", Value: 2},
				{Label: "Large", Value: 3},
			},
			Vertical: false,
		},
		{
			ID:    "radio_vert",
			Label: "Radio (vert)",
			Type:  panels.FormFieldRadio,
			Value: 2,
			Options: []panels.FormSelectOption{
				{Label: "Low", Value: 1},
				{Label: "Medium", Value: 2},
				{Label: "High", Value: 3},
				{Label: "Ultra", Value: 4},
			},
			Vertical: true,
		},
		{
			ID:         "hsl",
			Label:      "HSL Color",
			Type:       panels.FormFieldHSL,
			Hue:        180,
			Saturation: 75,
			Lightness:  50,
		},
		{
			ID:    "rgb",
			Label: "RGB Color",
			Type:  panels.FormFieldRGB,
			Red:   100,
			Green: 150,
			Blue:  200,
		},
	}

	m.popupPanel.SetRatio(0.6, 0.6)
	m.popupPanel.ShowFormLive("Form Field Demo", fields,
		func(result panels.PopupResult, finalFields []panels.FormField) {
			if result.Confirmed {
				m.setStatusTemporary("Form saved!", false, 3*time.Second)
			} else {
				m.setStatusTemporary("Form cancelled", false, 3*time.Second)
			}
		},
		func(field panels.FormField) {
			// Live mode callback - log changes
			var value string
			switch field.Type {
			case panels.FormFieldToggle:
				if field.Value != 0 {
					value = "on"
				} else {
					value = "off"
				}
			case panels.FormFieldColor:
				value = fmt.Sprintf("(%.2f, %.2f)", field.ColorX, field.ColorY)
			case panels.FormFieldHSL:
				value = fmt.Sprintf("H:%d S:%d L:%d", field.Hue, field.Saturation, field.Lightness)
			case panels.FormFieldRGB:
				value = fmt.Sprintf("R:%d G:%d B:%d", field.Red, field.Green, field.Blue)
			case panels.FormFieldText:
				value = field.TextValue
			default:
				value = fmt.Sprintf("%d", field.Value)
			}
			m.logPanel.AddEntry("live", fmt.Sprintf("%s: %s", field.Label, value))
		},
	)

	return startBlinkTicker()
}

// startBlinkTicker returns a command that starts the blink animation ticker.
func startBlinkTicker() tea.Cmd {
	return tea.Tick(400*time.Millisecond, func(t time.Time) tea.Msg {
		return blinkTickMsg{}
	})
}

// showCreateRoomPopup shows a form to create a new room.
func (m *Model) showCreateRoomPopup() tea.Cmd {
	bridge := m.manager.GetActiveBridge()
	if bridge == nil {
		m.setStatusTemporary("No active bridge", false, 3*time.Second)
		return nil
	}

	// Build archetype options
	archetypeOptions := []panels.FormSelectOption{}
	for i, archetype := range hue.RoomArchetypeList() {
		archetypeOptions = append(archetypeOptions, panels.FormSelectOption{
			Label: hue.RoomArchetypeDisplayName(archetype),
			Value: i,
		})
	}

	fields := []panels.FormField{
		{
			ID:        "name",
			Label:     "Name",
			Type:      panels.FormFieldText,
			TextValue: "",
		},
		{
			ID:      "archetype",
			Label:   "Type",
			Type:    panels.FormFieldSelect,
			Value:   0, // Default to first (living_room)
			Options: archetypeOptions,
		},
	}

	m.popupPanel.SetRatio(0.5, 0.35)
	m.popupPanel.ShowForm("Create Room", fields,
		func(result panels.PopupResult, finalFields []panels.FormField) {
			if !result.Confirmed {
				return
			}

			// Extract values
			name := ""
			archetypeIdx := 0
			for _, f := range finalFields {
				switch f.ID {
				case "name":
					name = f.TextValue
				case "archetype":
					archetypeIdx = f.Value
				}
			}

			if name == "" {
				m.setStatusTemporary("Room name is required", true, 3*time.Second)
				return
			}

			// Get archetype from index
			archetypes := hue.RoomArchetypeList()
			if archetypeIdx < 0 || archetypeIdx >= len(archetypes) {
				archetypeIdx = 0
			}
			archetype := hueclient.RoomArchetype(archetypes[archetypeIdx])

			// Create room with no devices initially
			if err := bridge.CreateRoom(name, archetype, nil); err != nil {
				m.setStatus("Failed to create room: "+err.Error(), true)
				return
			}

			m.setStatusTemporary("Created room: "+name, false, 3*time.Second)
			// Sync to get the new room
			go func() {
				bridge.SyncRooms(context.Background())
			}()
		},
	)

	return startBlinkTicker()
}

// showCreateZonePopup shows a form to create a new zone.
func (m *Model) showCreateZonePopup() tea.Cmd {
	bridge := m.manager.GetActiveBridge()
	if bridge == nil {
		m.setStatusTemporary("No active bridge", false, 3*time.Second)
		return nil
	}

	// Build archetype options
	archetypeOptions := []panels.FormSelectOption{}
	for i, archetype := range hue.RoomArchetypeList() {
		archetypeOptions = append(archetypeOptions, panels.FormSelectOption{
			Label: hue.RoomArchetypeDisplayName(archetype),
			Value: i,
		})
	}

	fields := []panels.FormField{
		{
			ID:        "name",
			Label:     "Name",
			Type:      panels.FormFieldText,
			TextValue: "",
		},
		{
			ID:      "archetype",
			Label:   "Type",
			Type:    panels.FormFieldSelect,
			Value:   0,
			Options: archetypeOptions,
		},
	}

	m.popupPanel.SetRatio(0.5, 0.35)
	m.popupPanel.ShowForm("Create Zone", fields,
		func(result panels.PopupResult, finalFields []panels.FormField) {
			if !result.Confirmed {
				return
			}

			// Extract values
			name := ""
			archetypeIdx := 0
			for _, f := range finalFields {
				switch f.ID {
				case "name":
					name = f.TextValue
				case "archetype":
					archetypeIdx = f.Value
				}
			}

			if name == "" {
				m.setStatusTemporary("Zone name is required", true, 3*time.Second)
				return
			}

			// Get archetype from index
			archetypes := hue.RoomArchetypeList()
			if archetypeIdx < 0 || archetypeIdx >= len(archetypes) {
				archetypeIdx = 0
			}
			archetype := hueclient.RoomArchetype(archetypes[archetypeIdx])

			// Create zone with no services initially
			if err := bridge.CreateZone(name, archetype, nil); err != nil {
				m.setStatus("Failed to create zone: "+err.Error(), true)
				return
			}

			m.setStatusTemporary("Created zone: "+name, false, 3*time.Second)
			// Sync to get the new zone
			go func() {
				bridge.SyncZones(context.Background())
			}()
		},
	)

	return startBlinkTicker()
}

// showDeleteConfirmation shows a confirmation dialog for deleting the selected entity.
func (m *Model) showDeleteConfirmation() tea.Cmd {
	bridge := m.manager.GetActiveBridge()
	if bridge == nil || m.selectedItem == nil {
		return nil
	}

	itemID := m.selectedItem.ID
	itemName := m.selectedItem.Name
	itemType := m.selectedItem.Type

	// Only rooms and zones can be deleted
	var entityType string
	switch itemType {
	case panels.EntityRoom:
		entityType = "room"
	case panels.EntityZone:
		entityType = "zone"
	default:
		m.setStatusTemporary("Cannot delete this item", false, 3*time.Second)
		return nil
	}

	m.popupPanel.SetRatio(0.5, 0.25)
	m.popupPanel.ShowConfirm(
		"Delete "+entityType,
		"Are you sure you want to delete \""+itemName+"\"?",
		func(result panels.PopupResult) {
			if !result.Confirmed {
				return
			}

			var err error
			switch itemType {
			case panels.EntityRoom:
				err = bridge.DeleteRoom(itemID)
			case panels.EntityZone:
				err = bridge.DeleteZone(itemID)
			}

			if err != nil {
				m.setStatus("Failed to delete: "+err.Error(), true)
				return
			}

			m.setStatusTemporary("Deleted: "+itemName, false, 3*time.Second)

			// Sync to update the UI
			go func() {
				if itemType == panels.EntityRoom {
					bridge.SyncRooms(context.Background())
				} else {
					bridge.SyncZones(context.Background())
				}
			}()
		},
	)

	return nil
}

// showEditRoomPopup shows a form to edit a room's properties.
func (m *Model) showEditRoomPopup() tea.Cmd {
	bridge := m.manager.GetActiveBridge()
	if bridge == nil || m.selectedItem == nil {
		return nil
	}

	if m.selectedItem.Type != panels.EntityRoom {
		return nil
	}

	room, ok := panels.GetRoomFromItem(*m.selectedItem)
	if !ok {
		return nil
	}

	roomID := m.selectedItem.ID
	roomName := m.selectedItem.Name
	state := bridge.GetState()

	// Get current archetype
	currentArchetype := "other"
	if room.Metadata != nil && room.Metadata.Archetype != nil {
		currentArchetype = string(*room.Metadata.Archetype)
	}

	// Build archetype options and find current index
	archetypeOptions := []panels.FormSelectOption{}
	currentArchetypeIdx := 0
	for i, archetype := range hue.RoomArchetypeList() {
		archetypeOptions = append(archetypeOptions, panels.FormSelectOption{
			Label: hue.RoomArchetypeDisplayName(archetype),
			Value: i,
		})
		if archetype == currentArchetype {
			currentArchetypeIdx = i
		}
	}

	// Build set of device IDs currently in this room
	currentDeviceIDs := make(map[string]bool)
	if room.Children != nil {
		for _, child := range *room.Children {
			if child.Rid != nil && child.Rtype != nil && *child.Rtype == hueclient.ResourceIdentifierRtypeDevice {
				currentDeviceIDs[*child.Rid] = true
			}
		}
	}

	// Get all devices and create toggle fields for devices in this room
	allDevices := state.GetAllDevices()
	fields := []panels.FormField{
		{
			ID:      "archetype",
			Label:   "Type",
			Type:    panels.FormFieldSelect,
			Value:   currentArchetypeIdx,
			Options: archetypeOptions,
		},
	}

	// Store device IDs in order for later reference
	deviceIDs := []string{}
	for _, device := range allDevices {
		if device.Id == nil {
			continue
		}
		deviceID := *device.Id
		deviceName := device.DeviceName(deviceID[:8])

		// Only show devices that are currently in this room
		if currentDeviceIDs[deviceID] {
			deviceIDs = append(deviceIDs, deviceID)
			fields = append(fields, panels.FormField{
				ID:             "device_" + deviceID,
				Label:          deviceName,
				Type:           panels.FormFieldToggle,
				Value:          1, // 1 = in room (checked), 0 = removed
				ToggleOnLabel:  "In Room",
				ToggleOffLabel: "Removed",
			})
		}
	}

	// Calculate popup height based on number of fields
	height := 0.25 + float64(len(fields))*0.05
	if height > 0.8 {
		height = 0.8
	}

	m.popupPanel.SetRatio(0.55, height)
	m.popupPanel.ShowForm("Edit Room: "+roomName, fields,
		func(result panels.PopupResult, finalFields []panels.FormField) {
			if !result.Confirmed {
				return
			}

			// Extract archetype
			archetypeIdx := 0
			for _, f := range finalFields {
				if f.ID == "archetype" {
					archetypeIdx = f.Value
					break
				}
			}

			// Get archetype from index
			archetypes := hue.RoomArchetypeList()
			if archetypeIdx < 0 || archetypeIdx >= len(archetypes) {
				archetypeIdx = 0
			}
			archetype := hueclient.RoomArchetype(archetypes[archetypeIdx])

			// Collect device IDs that should remain in the room (those still toggled on)
			remainingDeviceIDs := []string{}
			for _, f := range finalFields {
				if len(f.ID) > 7 && f.ID[:7] == "device_" {
					deviceID := f.ID[7:]
					if f.Value != 0 { // Value 1 = keep in room, Value 0 = removed
						remainingDeviceIDs = append(remainingDeviceIDs, deviceID)
					}
				}
			}

			// Update archetype
			if err := bridge.SetRoomArchetype(roomID, archetype); err != nil {
				m.setStatus("Failed to update room type: "+err.Error(), true)
				return
			}

			// Update devices if changed
			if len(remainingDeviceIDs) != len(currentDeviceIDs) {
				if err := bridge.UpdateRoomDevices(roomID, remainingDeviceIDs); err != nil {
					m.setStatus("Failed to update room devices: "+err.Error(), true)
					return
				}
			}

			m.setStatusTemporary("Updated room: "+roomName, false, 3*time.Second)
			go func() {
				bridge.SyncRooms(context.Background())
			}()
		},
	)

	return startBlinkTicker()
}

// showEditZonePopup shows a form to edit a zone's properties.
func (m *Model) showEditZonePopup() tea.Cmd {
	bridge := m.manager.GetActiveBridge()
	if bridge == nil || m.selectedItem == nil {
		return nil
	}

	if m.selectedItem.Type != panels.EntityZone {
		return nil
	}

	// Get the zone from the state (zones use the same RoomGet type)
	state := bridge.GetState()
	zone, ok := state.GetZone(m.selectedItem.ID)
	if !ok {
		return nil
	}

	zoneID := m.selectedItem.ID
	zoneName := m.selectedItem.Name

	// Get current archetype
	currentArchetype := "other"
	if zone.Metadata != nil && zone.Metadata.Archetype != nil {
		currentArchetype = string(*zone.Metadata.Archetype)
	}

	// Build archetype options and find current index
	archetypeOptions := []panels.FormSelectOption{}
	currentIdx := 0
	for i, archetype := range hue.RoomArchetypeList() {
		archetypeOptions = append(archetypeOptions, panels.FormSelectOption{
			Label: hue.RoomArchetypeDisplayName(archetype),
			Value: i,
		})
		if archetype == currentArchetype {
			currentIdx = i
		}
	}

	fields := []panels.FormField{
		{
			ID:      "archetype",
			Label:   "Type",
			Type:    panels.FormFieldSelect,
			Value:   currentIdx,
			Options: archetypeOptions,
		},
	}

	m.popupPanel.SetRatio(0.5, 0.30)
	m.popupPanel.ShowForm("Edit Zone: "+zoneName, fields,
		func(result panels.PopupResult, finalFields []panels.FormField) {
			if !result.Confirmed {
				return
			}

			// Extract archetype
			archetypeIdx := 0
			for _, f := range finalFields {
				if f.ID == "archetype" {
					archetypeIdx = f.Value
				}
			}

			// Get archetype from index
			archetypes := hue.RoomArchetypeList()
			if archetypeIdx < 0 || archetypeIdx >= len(archetypes) {
				archetypeIdx = 0
			}
			archetype := hueclient.RoomArchetype(archetypes[archetypeIdx])

			// Use SetRoomArchetype for zones too (same API)
			if err := bridge.SetZoneArchetype(zoneID, archetype); err != nil {
				m.setStatus("Failed to update zone: "+err.Error(), true)
				return
			}

			m.setStatusTemporary("Updated zone: "+zoneName, false, 3*time.Second)
			go func() {
				bridge.SyncZones(context.Background())
			}()
		},
	)

	return startBlinkTicker()
}
