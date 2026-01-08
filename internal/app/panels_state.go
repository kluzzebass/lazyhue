package app

import (
	"sort"

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
	return m.focusIndex < 0
}

func (m *Model) focusedPanelID() string {
	if m.focusIndex >= 0 && m.focusIndex < len(m.panelOrder) {
		return m.panelOrder[m.focusIndex]
	}
	return PanelIDDetail
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
	if state == nil {
		return
	}

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

	// For other entities, use the active bridge's state
	bridge := m.manager.GetActiveBridge()
	if bridge != nil {
		m.detailPanel.SetItem(m.selectedItem, bridge.GetState())
	}
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

				// If clicking the same bridge, do nothing
				if currentID == bridge.Info.ID {
					return
				}

				// Save current bridge's expanded state before switching
				if currentID != "" {
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
	m.statusBar.SetMessage(msg, isError)
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
	// If on a group node (no item selected), only show navigation
	if m.selectedItem == nil {
		return filterBindingsByAction(bindings,
			ui.ActionSelect, ui.ActionExpand, ui.ActionCollapse)
	}

	switch m.selectedItem.Type {
	case panels.EntityLight:
		// Lights: toggle, on/off, brightness
		return filterBindingsByAction(bindings,
			ui.ActionSelect, ui.ActionToggle, ui.ActionTurnOn, ui.ActionTurnOff,
			ui.ActionBrightnessUp, ui.ActionBrightnessDown,
			ui.ActionExpand, ui.ActionCollapse)

	case panels.EntityScene:
		// Scenes: activate only
		return filterBindingsByAction(bindings,
			ui.ActionSelect, ui.ActionExpand, ui.ActionCollapse)

	case panels.EntityDevice:
		// Devices: just select (view details)
		return filterBindingsByAction(bindings,
			ui.ActionSelect, ui.ActionExpand, ui.ActionCollapse)

	case panels.EntityRoom, panels.EntityZone:
		// Rooms/Zones: toggle (grouped light), brightness
		return filterBindingsByAction(bindings,
			ui.ActionSelect, ui.ActionToggle,
			ui.ActionBrightnessUp, ui.ActionBrightnessDown,
			ui.ActionExpand, ui.ActionCollapse)

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


