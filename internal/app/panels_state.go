package app

import (
	"sort"

	"github.com/kluzzebass/lazyhue/internal/ui/panels"
)

// Panel accessors for type-specific operations

func (m *Model) bridgePanel() *panels.BridgePanel {
	return m.panelMap[PanelIDBridges].(*panels.BridgePanel)
}

func (m *Model) scenesPanel() *panels.TreePanel {
	return m.panelMap[PanelIDScenes].(*panels.TreePanel)
}

func (m *Model) groupsPanel() *panels.TabbedPanel {
	return m.panelMap[PanelIDGroups].(*panels.TabbedPanel)
}

func (m *Model) lightsPanel() *panels.ListPanel {
	return m.panelMap[PanelIDLights].(*panels.ListPanel)
}

func (m *Model) devicesPanel() *panels.ListPanel {
	return m.panelMap[PanelIDDevices].(*panels.ListPanel)
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
	m.refreshGroupsPanel()
	m.refreshLightsPanel()
	m.refreshDevicesPanel()
	m.refreshScenesPanel()
	m.updateDetailPanel() // Always refresh details with latest state
}

func (m *Model) refreshGroupsPanel() {
	bridge := m.manager.GetActiveBridge()
	if bridge == nil {
		m.groupsPanel().SetItemsByID("Rooms", nil)
		m.groupsPanel().SetItemsByID("Zones", nil)
		m.groupsPanel().SetItemsByID("Entertainment", nil)
		return
	}

	state := bridge.GetState()
	if state == nil {
		return
	}

	// Rooms tab
	m.groupsPanel().SetItemsByID("Rooms", panels.BuildRoomItems(state))

	// Zones tab
	m.groupsPanel().SetItemsByID("Zones", panels.BuildZoneItems(state))

	// Entertainment tab
	m.groupsPanel().SetItemsByID("Entertainment", panels.BuildEntertainmentItems(state))
}

func (m *Model) refreshLightsPanel() {
	bridge := m.manager.GetActiveBridge()
	if bridge == nil {
		m.lightsPanel().SetItems(nil)
		return
	}
	state := bridge.GetState()
	if state == nil {
		return
	}
	m.lightsPanel().SetItems(panels.BuildLightItems(state))
}

func (m *Model) refreshDevicesPanel() {
	bridge := m.manager.GetActiveBridge()
	if bridge == nil {
		m.devicesPanel().SetItems(nil)
		return
	}
	state := bridge.GetState()
	if state == nil {
		return
	}
	m.devicesPanel().SetItems(panels.BuildDeviceItems(state))
}

func (m *Model) refreshScenesPanel() {
	bridge := m.manager.GetActiveBridge()
	if bridge == nil {
		m.scenesPanel().SetRoots(nil)
		return
	}
	state := bridge.GetState()
	if state == nil {
		return
	}
	m.scenesPanel().SetRoots(panels.BuildSceneTree(state))
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
	// Get focused panel's bindings and pass to status bar
	panelID := m.focusedPanelID()
	if bindings, ok := m.panelBindings[panelID]; ok {
		m.statusBar.SetBindings(bindings)
	} else {
		m.statusBar.SetBindings(nil)
	}
}


