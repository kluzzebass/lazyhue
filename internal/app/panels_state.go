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

func (m *Model) hierarchyPanel() *panels.TreePanel {
	return m.panelMap[PanelIDHierarchy].(*panels.TreePanel)
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
}

func (m *Model) refreshAllPanels() {
	m.refreshHierarchyPanel()
	m.updateDetailPanel() // Always refresh details with latest state
}

func (m *Model) refreshHierarchyPanel() {
	bridge := m.manager.GetActiveBridge()
	if bridge == nil {
		m.hierarchyPanel().SetRoots(nil)
		return
	}

	state := bridge.GetState()
	if state == nil {
		return
	}

	m.hierarchyPanel().SetRoots(panels.BuildHierarchyTree(state))
}

func (m *Model) updateBridgePanel() {
	bridges := m.manager.AllBridges()
	sort.Slice(bridges, func(i, j int) bool {
		return bridges[i].Info.Name < bridges[j].Info.Name
	})
	m.bridgePanel().SetBridges(bridges, m.manager.GetActiveBridgeID())
}

func (m *Model) updateDetailPanel() {
	bridge := m.manager.GetActiveBridge()
	if bridge != nil && m.selectedItem != nil {
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


