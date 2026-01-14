package app2

import "log/slog"

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
	slog.Debug("restoreState: starting")

	// Restore active tab
	activeTab := m.uiState.GetActiveTab("global")
	slog.Debug("restoreState: setting active tab", "tab", activeTab)
	m.tree.SetActiveTab(activeTab)
	m.rebuildTreeForActiveTab()

	// Restore node expansion states
	states := m.uiState.GetNodeStates("global")
	slog.Debug("restoreState: got expansion states", "count", len(states))

	// Log a sample of the states
	expandedCount := 0
	for id, expanded := range states {
		if expanded {
			expandedCount++
			if expandedCount <= 5 {
				slog.Debug("restoreState: expanded node", "id", id)
			}
		}
	}
	slog.Debug("restoreState: total expanded nodes", "count", expandedCount)

	if len(states) > 0 {
		m.tree.SetExpandedStates(states)
		slog.Debug("restoreState: applied expansion states")

		// Verify by checking current tree state
		currentStates := m.tree.GetExpandedStates()
		currentExpanded := 0
		for _, exp := range currentStates {
			if exp {
				currentExpanded++
			}
		}
		slog.Debug("restoreState: tree now has expanded nodes", "count", currentExpanded)
	}

	// Restore selected entity
	selectedID := m.uiState.GetSelectedEntityID("global")
	slog.Debug("restoreState: selected entity ID", "id", selectedID)
	if selectedID != "" {
		// Try to select the entity in the tree
		if m.tree.SelectByID(selectedID) {
			slog.Debug("restoreState: SelectByID succeeded")
			// Update detail panel to show the selected entity
			m.updateDetailContent()
		} else {
			// Entity not found yet - store as pending selection to retry after tree rebuilds
			slog.Debug("restoreState: SelectByID failed - storing as pending selection")
			m.pendingSelectionID = selectedID
		}
	}

	slog.Debug("restoreState: completed")
}
