package app2

// State persistence helpers for app2

// saveState persists the current UI state for the active bridge.
func (m *Model) saveState() {
	if m.activeBridgeID == "" {
		return
	}

	// Save active tab
	activeTab := m.tree.ActiveTabIndex()
	m.uiState.SetActiveTab(m.activeBridgeID, activeTab)

	// Save selected entity ID
	if node := m.tree.SelectedNode(); node != nil && node.Item != nil {
		m.uiState.SetSelectedEntityID(m.activeBridgeID, node.Item.ID)
	}

	// Save node expansion states
	m.uiState.SetNodeStates(m.activeBridgeID, m.tree.GetExpandedStates())

	// Save last selected bridge
	m.uiState.LastSelectedBridgeID = m.activeBridgeID

	// Best effort save to disk
	_ = m.uiState.Save()
}

// restoreState restores UI state for a bridge after its state has been synced.
func (m *Model) restoreState(bridgeID string) {
	// Restore active tab
	activeTab := m.uiState.GetActiveTab(bridgeID)
	m.tree.SetActiveTab(activeTab)
	m.rebuildTreeForActiveTab()

	// Restore node expansion states
	states := m.uiState.GetNodeStates(bridgeID)
	if len(states) > 0 {
		m.tree.SetExpandedStates(states)
	}

	// Restore selected entity
	selectedID := m.uiState.GetSelectedEntityID(bridgeID)
	if selectedID != "" {
		// Try to select the entity in the tree
		if m.tree.SelectByID(selectedID) {
			// Update detail panel to show the selected entity
			m.updateDetailContent()
		}
	}
}
