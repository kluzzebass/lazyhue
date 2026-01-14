package app2

// State persistence helpers for app2

// saveState persists the current UI state.
func (m *Model) saveState() {
	// Save active tab
	activeTab := m.tree.ActiveTabIndex()
	// Since we no longer have per-bridge state, just save to a global key
	m.uiState.SetActiveTab("global", activeTab)

	// Save selected entity ID
	if node := m.tree.SelectedNode(); node != nil && node.Item != nil {
		m.uiState.SetSelectedEntityID("global", node.Item.ID)
	}

	// Save node expansion states
	m.uiState.SetNodeStates("global", m.tree.GetExpandedStates())

	// Best effort save to disk
	_ = m.uiState.Save()
}

// restoreState restores UI state.
func (m *Model) restoreState() {
	// Restore active tab
	activeTab := m.uiState.GetActiveTab("global")
	m.tree.SetActiveTab(activeTab)
	m.rebuildTreeForActiveTab()

	// Restore node expansion states
	states := m.uiState.GetNodeStates("global")
	if len(states) > 0 {
		m.tree.SetExpandedStates(states)
	}

	// Restore selected entity
	selectedID := m.uiState.GetSelectedEntityID("global")
	if selectedID != "" {
		if m.tree.SelectByID(selectedID) {
			m.updateDetailContent()
		} else {
			// Entity not found yet - store as pending selection to retry after tree rebuilds
			m.pendingSelectionID = selectedID
		}
	}
}
