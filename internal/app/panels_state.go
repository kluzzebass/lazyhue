package app

import (
	"fmt"
	"sort"
	"time"

	"github.com/kluzzebass/lazyhue/internal/hue"
	"github.com/kluzzebass/lazyhue/internal/ui"
	"github.com/kluzzebass/lazyhue/internal/ui/panels"
)

// Panel accessors for type-specific operations

func (m *Model) bridgePanel() *panels.BridgePanel {
	return m.panelMap[PanelIDBridges].(*panels.BridgePanel)
}

func (m *Model) hierarchyPanel() *panels.HomeTabbedPanel {
	return m.panelMap[PanelIDHierarchy].(*panels.HomeTabbedPanel)
}

func (m *Model) focusedOnDetail() bool {
	return m.focusIndex == -1
}

func (m *Model) focusedOnLog() bool {
	return m.logPanelVisible && m.focusIndex == -2
}

func (m *Model) focusedPanelID() string {
	switch m.focusIndex {
	case -1:
		return PanelIDDetail
	case -2:
		return PanelIDLog
	default:
		if m.focusIndex >= 0 && m.focusIndex < len(m.panelOrder) {
			return m.panelOrder[m.focusIndex]
		}
		return PanelIDDetail
	}
}

func (m *Model) focusedPanel() panels.Panel {
	if m.focusIndex >= 0 && m.focusIndex < len(m.panelOrder) {
		return m.panelMap[m.panelOrder[m.focusIndex]]
	}
	return nil
}

// Panel refresh/update methods

func (m *Model) updateLayout() {
	// Calculate layout tree
	m.layoutTree.Layout(m.width, m.height)

	// Apply sizes to all panels from the tree
	for id, panel := range m.panelMap {
		bounds := m.layoutTree.Bounds(id)
		panel.SetSize(bounds.Width, bounds.Height)
	}

	// Size detail panel
	bounds := m.layoutTree.Bounds(PanelIDDetail)
	m.detailPanel.SetSize(bounds.Width, bounds.Height)

	// Size log panel
	bounds = m.layoutTree.Bounds(PanelIDLog)
	m.logPanel.SetSize(bounds.Width, bounds.Height)

	// Size status bar
	bounds = m.layoutTree.Bounds(PanelIDStatus)
	m.statusBar.SetWidth(bounds.Width)

	// Size help panel (uses full screen dimensions)
	m.helpPanel.SetSize(m.width, m.height)

	// Size pairing panel
	m.pairingPanel.SetSize(m.width, m.height)

	// Size popup panel
	m.popupPanel.SetSize(m.width, m.height)
}

func (m *Model) refreshAllPanels() {
	m.refreshHierarchyPanel()
	m.updateDetailPanel() // Always refresh details with latest state
}

func (m *Model) refreshHierarchyPanel() {
	bridge := m.manager.GetActiveBridge()
	if bridge == nil {
		m.hierarchyPanel().Clear()
		m.displayedBridgeID = ""
		return
	}

	state := bridge.GetState()

	// Update Home tab (tree)
	tree := panels.BuildHierarchyTree(state)
	m.hierarchyPanel().SetRoots(tree)

	// Update Lights tab (flat list)
	m.hierarchyPanel().SetLights(panels.BuildLightItems(state))

	// Update Devices tab (flat list)
	m.hierarchyPanel().SetDevices(panels.BuildDeviceItems(state))

	// Update Scenes tab (flat list)
	m.hierarchyPanel().SetScenes(panels.BuildSceneItems(state))

	m.displayedBridgeID = bridge.Info.ID
}

func (m *Model) updateBridgePanel() {
	bridges := m.manager.AllBridges()
	sort.Slice(bridges, func(i, j int) bool {
		return bridges[i].Info.Name < bridges[j].Info.Name
	})
	m.bridgePanel().SetBridges(bridges, m.manager.GetActiveBridgeID())

	// If no active bridge is set yet, sync it to the panel's selection
	// This ensures the hierarchy shows the correct bridge when data syncs
	if m.manager.GetActiveBridgeID() == "" {
		if selected := m.bridgePanel().SelectedBridge(); selected != nil && selected.IsConnected() {
			m.manager.SetActiveBridge(selected.Info.ID)
		}
	}

	// If no item is selected, sync selection from bridge panel
	// This ensures the details panel shows bridge info on startup
	if m.selectedItem == nil {
		if item, ok := m.bridgePanel().SelectedEntity(); ok {
			m.selectedItem = item
		}
	}
}

func (m *Model) updateDetailPanel() {
	if m.selectedItem == nil {
		return
	}

	// For bridge entities, use the bridge's own state
	if m.selectedItem.Type == panels.EntityBridge {
		m.detailPanel.SetItem(m.selectedItem, nil)
		return
	}

	// For other entities, refresh the RawPtr from current state
	bridge := m.manager.GetActiveBridge()
	if bridge != nil {
		state := bridge.GetState()
		// Refresh RawPtr from live state to get latest data
		if m.refreshSelectedItemFromState(state) {
			m.detailPanel.SetItem(m.selectedItem, state)
		}
	}
}

// refreshSelectedItemFromState updates the selected item's RawPtr with fresh data from state.
// Returns true if the data was successfully refreshed, false if the entity wasn't found.
func (m *Model) refreshSelectedItemFromState(state *hue.BridgeState) bool {
	if m.selectedItem == nil || state == nil {
		return false
	}

	switch m.selectedItem.Type {
	case panels.EntityLight:
		if light, ok := state.GetLight(m.selectedItem.ID); ok {
			m.selectedItem.RawPtr = light
			m.selectedItem.IsOn = panels.IsLightOn(light)
			m.selectedItem.Name = state.GetLightName(light)
			return true
		}
	case panels.EntityRoom:
		if room, ok := state.GetRoom(m.selectedItem.ID); ok {
			m.selectedItem.RawPtr = room
			m.selectedItem.IsOn = state.IsRoomOn(room)
			m.selectedItem.Name = hue.RoomName(room, m.selectedItem.Name)
			return true
		}
	case panels.EntityZone:
		if zone, ok := state.GetZone(m.selectedItem.ID); ok {
			m.selectedItem.RawPtr = zone
			m.selectedItem.IsOn = state.IsRoomOn(zone)
			m.selectedItem.Name = hue.RoomName(zone, m.selectedItem.Name)
			return true
		}
	case panels.EntityScene:
		if scene, ok := state.GetScene(m.selectedItem.ID); ok {
			m.selectedItem.RawPtr = scene
			// Scene active status
			if scene.Status != nil && scene.Status.Active != nil {
				m.selectedItem.IsOn = string(*scene.Status.Active) == "static" || string(*scene.Status.Active) == "dynamic_palette"
			}
			m.selectedItem.Name = hue.SceneName(scene, m.selectedItem.Name)
			return true
		}
	case panels.EntityDevice:
		if device, ok := state.GetDevice(m.selectedItem.ID); ok {
			m.selectedItem.RawPtr = device
			m.selectedItem.Name = hue.DeviceName(device, m.selectedItem.Name)
			return true
		}
	case panels.EntityEntertainment:
		if ent, ok := state.GetEntertainmentConfiguration(m.selectedItem.ID); ok {
			m.selectedItem.RawPtr = ent
			m.selectedItem.Name = hue.EntertainmentName(ent, m.selectedItem.Name)
			return true
		}
	case panels.EntityLightsCategory, panels.EntityDevicesCategory, panels.EntityScenesCategory:
		// Category items don't need state refresh - their RawPtr contains the data
		return true
	}
	return false
}

func (m *Model) syncSelectionFromFocusedPanel() {
	if panel := m.focusedPanel(); panel != nil {
		if item, ok := panel.SelectedEntity(); ok {
			m.selectedItem = item
			m.updateDetailPanel()
		}
	}

	// When a bridge is selected, update the hierarchy automatically
	if m.focusedPanelID() == PanelIDBridges {
		if bridge := m.bridgePanel().SelectedBridge(); bridge != nil {
			if bridge.IsConnected() {
				currentID := m.manager.GetActiveBridgeID()

				// Skip if same bridge AND hierarchy is already showing it
				// (displayedBridgeID might be empty if we just viewed a disconnected bridge)
				if currentID == bridge.Info.ID && m.displayedBridgeID == bridge.Info.ID {
					return
				}

				// Save current bridge's expanded state before switching
				if currentID != "" && currentID != bridge.Info.ID {
					m.saveExpandedState(currentID)
				}

				// Set as active and show its hierarchy
				m.manager.SetActiveBridge(bridge.Info.ID)
				m.updateBridgePanel() // Update checkmark indicator immediately
				m.refreshHierarchyPanel()
				m.updateDetailPanel() // Refresh details with new bridge's data

				// Restore new bridge's expanded state
				m.restoreExpandedState(bridge.Info.ID)
			} else {
				// Clear hierarchy for disconnected bridges
				m.hierarchyPanel().Clear()
				m.displayedBridgeID = ""
			}
		}
	}
}

// saveExpandedState saves the current hierarchy's expanded state for a bridge.
func (m *Model) saveExpandedState(bridgeID string) {
	// Save node expanded states
	states := m.hierarchyPanel().GetNodeStates()
	m.uiState.SetNodeStates(bridgeID, states)

	// Save active tab
	m.uiState.SetActiveTab(bridgeID, m.hierarchyPanel().ActiveTabIndex())

	// Save selected entity ID
	if m.selectedItem != nil {
		m.uiState.SetSelectedEntityID(bridgeID, m.selectedItem.ID)
	}

	// Save focused panel index
	m.uiState.FocusedPanelIndex = m.focusIndex

	_ = m.uiState.Save() // Best effort save
}

// restoreExpandedState restores the expanded state for a bridge.
func (m *Model) restoreExpandedState(bridgeID string) {
	// Restore node expanded states
	states := m.uiState.GetNodeStates(bridgeID)
	if len(states) > 0 {
		m.hierarchyPanel().SetNodeStates(states)
	}

	// Restore active tab
	m.hierarchyPanel().SetActiveTab(m.uiState.GetActiveTab(bridgeID))

	// Restore selected entity
	if entityID := m.uiState.GetSelectedEntityID(bridgeID); entityID != "" {
		if m.hierarchyPanel().SelectByID(entityID) {
			// Update selected item and detail panel
			if entity, ok := m.hierarchyPanel().SelectedEntity(); ok {
				m.selectedItem = entity
				m.updateDetailPanel()
			}
		}
	}
}

func (m *Model) setStatus(msg string, isError bool) {
	m.statusMsg = msg
	m.isError = isError
	m.statusExpiry = time.Time{} // No auto-clear
	m.statusBar.SetMessage(msg, isError)
}

// setStatusTemporary sets a status message that auto-clears after a duration.
func (m *Model) setStatusTemporary(msg string, isError bool, duration time.Duration) {
	m.statusMsg = msg
	m.isError = isError
	m.statusExpiry = time.Now().Add(duration)
	m.statusBar.SetMessage(msg, isError)
}

// checkStatusExpiry clears the status if it has expired.
func (m *Model) checkStatusExpiry() {
	if !m.statusExpiry.IsZero() && time.Now().After(m.statusExpiry) {
		m.clearStatus()
		m.statusExpiry = time.Time{}
	}
}

func (m *Model) clearStatus() {
	m.statusMsg = ""
	m.isError = false
	m.statusBar.ClearMessage()
}

func (m *Model) updateStatusContext() {
	panelID := m.focusedPanelID()
	bindings, ok := m.panelBindings[panelID]
	if !ok {
		m.statusBar.SetBindings(nil)
		return
	}

	// For hierarchy panel, filter bindings based on selected entity type
	if panelID == PanelIDHierarchy {
		bindings = m.filterBindingsForSelection(bindings)
	}

	m.statusBar.SetBindings(bindings)
}

// filterBindingsForSelection returns only relevant bindings for the current selection.
func (m *Model) filterBindingsForSelection(bindings []ui.Binding) []ui.Binding {
	// Check if we're on the tree view (tab 0) - only then show expand/collapse
	onTreeView := m.hierarchyPanel().ActiveTabIndex() == 0

	// Helper to conditionally add tree actions
	withTreeActions := func(actions ...ui.Action) []ui.Action {
		if onTreeView {
			actions = append(actions, ui.ActionExpand, ui.ActionCollapse)
		}
		return actions
	}

	// If on a group node (no item selected), only show navigation
	if m.selectedItem == nil {
		return filterBindingsByAction(bindings, withTreeActions(ui.ActionSelect)...)
	}

	switch m.selectedItem.Type {
	case panels.EntityLight:
		// Lights: controls (enter), toggle (space), on/off, brightness
		return filterBindingsByAction(bindings, withTreeActions(
			ui.ActionSelect, ui.ActionToggle, ui.ActionTurnOn, ui.ActionTurnOff,
			ui.ActionBrightnessUp, ui.ActionBrightnessDown)...)

	case panels.EntityScene:
		// Scenes: activate, rename
		return filterBindingsByAction(bindings, withTreeActions(
			ui.ActionSelect, ui.ActionRename)...)

	case panels.EntityDevice:
		// Devices: select (view details), edit, rename
		return filterBindingsByAction(bindings, withTreeActions(
			ui.ActionSelect, ui.ActionEditDevice, ui.ActionRename)...)

	case panels.EntityRoom, panels.EntityZone:
		// Rooms/Zones: toggle (grouped light), brightness, rename
		return filterBindingsByAction(bindings, withTreeActions(
			ui.ActionSelect, ui.ActionToggle, ui.ActionRename,
			ui.ActionBrightnessUp, ui.ActionBrightnessDown)...)

	default:
		return bindings
	}
}

// filterBindingsByAction returns only bindings matching the given actions.
func filterBindingsByAction(bindings []ui.Binding, actions ...ui.Action) []ui.Binding {
	actionSet := make(map[ui.Action]bool)
	for _, a := range actions {
		actionSet[a] = true
	}

	var filtered []ui.Binding
	for _, b := range bindings {
		if actionSet[b.Action] {
			filtered = append(filtered, b)
		}
	}
	return filtered
}

// buildEventDetails creates rich event details by looking up resource info from bridge state.
func (m *Model) buildEventDetails(msg BridgeEventMsg) panels.EventDetails {
	details := panels.EventDetails{
		ResourceType: msg.ResourceType,
		EventType:    msg.EventType,
	}

	bridge := m.manager.GetBridge(msg.BridgeID)
	if bridge == nil || bridge.GetState() == nil {
		return details
	}
	state := bridge.GetState()

	switch msg.ResourceType {
	case "light":
		if light, ok := state.GetLight(msg.ResourceID); ok {
			// Use GetLightName to get the user-assigned name from the owning device
			details.ResourceName = state.GetLightName(light)
			// Show current state
			if light.On != nil && light.On.On != nil {
				details.IsOn = *light.On.On
				if *light.On.On {
					if light.Dimming != nil && light.Dimming.Brightness != nil {
						details.Brightness = float64(*light.Dimming.Brightness)
						details.Details = fmt.Sprintf("on %.0f%%", *light.Dimming.Brightness)
					} else {
						details.Details = "on"
					}
					// Get color for indicator
					if light.Color != nil && light.Color.Xy != nil &&
						light.Color.Xy.X != nil && light.Color.Xy.Y != nil {
						r, g, b := ui.XyToRGB(float64(*light.Color.Xy.X), float64(*light.Color.Xy.Y), 1.0)
						details.IndicatorColor = fmt.Sprintf("#%02x%02x%02x", r, g, b)
					} else if light.ColorTemperature != nil && light.ColorTemperature.Mirek != nil {
						r, g, b := ui.MirekToRGB(*light.ColorTemperature.Mirek)
						details.IndicatorColor = fmt.Sprintf("#%02x%02x%02x", r, g, b)
					}
				} else {
					details.Details = "off"
				}
			}
		}

	case "grouped_light":
		if gl, ok := state.GetGroupedLight(msg.ResourceID); ok {
			// Try to find the room/zone name for this grouped light
			if name := state.GetGroupedLightName(msg.ResourceID); name != "" {
				details.ResourceName = name
			}
			if gl.On != nil && gl.On.On != nil {
				details.IsOn = *gl.On.On
				if *gl.On.On {
					if gl.Dimming != nil && gl.Dimming.Brightness != nil {
						details.Brightness = float64(*gl.Dimming.Brightness)
						details.Details = fmt.Sprintf("on %.0f%%", *gl.Dimming.Brightness)
					} else {
						details.Details = "on"
					}
				} else {
					details.Details = "off"
				}
			}
		}

	case "scene":
		if scene, ok := state.GetScene(msg.ResourceID); ok {
			details.ResourceName = hue.SceneName(scene, details.ResourceName)
			if scene.Status != nil && scene.Status.Active != nil {
				if *scene.Status.Active == "active" {
					details.Details = "activated"
				} else {
					details.Details = "deactivated"
				}
			}
		}

	case "motion":
		if motion, ok := state.GetMotion(msg.ResourceID); ok {
			// Try to get device name
			if motion.Owner != nil && motion.Owner.Rid != nil {
				if device, ok := state.GetDevice(*motion.Owner.Rid); ok {
					details.ResourceName = hue.DeviceName(device, details.ResourceName)
				}
			}
			if motion.Motion != nil && motion.Motion.Motion != nil {
				if *motion.Motion.Motion {
					details.Details = "motion detected"
				} else {
					details.Details = "clear"
				}
			}
		}

	case "temperature":
		if temp, ok := state.GetTemperature(msg.ResourceID); ok {
			// Try to get device name
			if temp.Owner != nil && temp.Owner.Rid != nil {
				if device, ok := state.GetDevice(*temp.Owner.Rid); ok {
					details.ResourceName = hue.DeviceName(device, details.ResourceName)
				}
			}
			if temp.Temperature != nil && temp.Temperature.Temperature != nil {
				details.Details = fmt.Sprintf("%.1f°C", *temp.Temperature.Temperature)
			}
		}

	case "light_level":
		if ll, ok := state.GetLightLevel(msg.ResourceID); ok {
			// Try to get device name
			if ll.Owner != nil && ll.Owner.Rid != nil {
				if device, ok := state.GetDevice(*ll.Owner.Rid); ok {
					details.ResourceName = hue.DeviceName(device, details.ResourceName)
				}
			}
			if ll.Light != nil && ll.Light.LightLevel != nil {
				// Convert from Hue's log scale to approximate lux
				lux := float64(*ll.Light.LightLevel-1) / 10000.0
				lux = 100 * (lux * lux * lux) // Rough approximation
				details.Details = fmt.Sprintf("%.0f lux", lux)
			}
		}
	}

	return details
}
