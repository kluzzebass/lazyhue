package app

import (
	"fmt"
	"time"
)

// navigateToEntity navigates to a specific entity by type and ID.
func (m *Model) navigateToEntity(entityType, entityID, bridgeID string) {
	// Find the entity in the tree
	node := m.tree.FindEntity(entityType, entityID)
	if node == nil {
		m.status = fmt.Sprintf("Entity not found: %s %s", entityType, entityID)
		return
	}

	// Add current position to history before navigating (if we have a current selection)
	if currentNode := m.tree.SelectedNode(); currentNode != nil && currentNode.Item != nil {
		// Only add to history if we're not already in the middle of history navigation
		if m.historyIndex == -1 || m.historyIndex == len(m.navigationHistory)-1 {
			// At the end of history or no history - add new entry
			entry := NavigationEntry{
				EntityType: currentNode.Item.Type.String(),
				EntityID:   currentNode.Item.ID,
				BridgeID:   currentNode.Item.BridgeID,
				Timestamp:  time.Now(),
			}
			m.navigationHistory = append(m.navigationHistory, entry)
			m.historyIndex = len(m.navigationHistory) - 1
		} else {
			// In the middle of history - truncate forward history and add new entry
			m.navigationHistory = m.navigationHistory[:m.historyIndex+1]
			entry := NavigationEntry{
				EntityType: currentNode.Item.Type.String(),
				EntityID:   currentNode.Item.ID,
				BridgeID:   currentNode.Item.BridgeID,
				Timestamp:  time.Now(),
			}
			m.navigationHistory = append(m.navigationHistory, entry)
			m.historyIndex = len(m.navigationHistory) - 1
		}
	}

	// Select the node in the tree
	m.tree.SelectNode(node)
	m.saveState() // Persist selection

	// Update detail panel
	m.updateDetailContent()

	// Auto-focus detail panel for easier interaction
	m.previousPane = m.focusedPane
	m.focusedPane = PanelDetail

	m.status = fmt.Sprintf("Navigated to %s", node.Item.Name)
}

// navigateBack navigates backward in history.
func (m *Model) navigateBack() {
	if len(m.navigationHistory) == 0 || m.historyIndex <= 0 {
		m.status = "No previous navigation"
		return
	}

	// Move back in history
	m.historyIndex--
	entry := m.navigationHistory[m.historyIndex]

	// Find and select the entity
	node := m.tree.FindEntity(entry.EntityType, entry.EntityID)
	if node == nil {
		m.status = fmt.Sprintf("Entity no longer exists: %s", entry.EntityID)
		// Remove invalid entry and try again
		m.navigationHistory = append(m.navigationHistory[:m.historyIndex], m.navigationHistory[m.historyIndex+1:]...)
		if m.historyIndex >= len(m.navigationHistory) {
			m.historyIndex = len(m.navigationHistory) - 1
		}
		return
	}

	m.tree.SelectNode(node)
	m.saveState() // Persist selection
	m.updateDetailContent()
	m.previousPane = m.focusedPane
	m.focusedPane = PanelDetail

	m.status = fmt.Sprintf("Back: %s (%d/%d)", node.Item.Name, m.historyIndex+1, len(m.navigationHistory))
}

// navigateForward navigates forward in history.
func (m *Model) navigateForward() {
	if len(m.navigationHistory) == 0 || m.historyIndex >= len(m.navigationHistory)-1 {
		m.status = "No forward navigation"
		return
	}

	// Move forward in history
	m.historyIndex++
	entry := m.navigationHistory[m.historyIndex]

	// Find and select the entity
	node := m.tree.FindEntity(entry.EntityType, entry.EntityID)
	if node == nil {
		m.status = fmt.Sprintf("Entity no longer exists: %s", entry.EntityID)
		// Remove invalid entry and try again
		m.navigationHistory = append(m.navigationHistory[:m.historyIndex], m.navigationHistory[m.historyIndex+1:]...)
		if m.historyIndex >= len(m.navigationHistory) {
			m.historyIndex = len(m.navigationHistory) - 1
		}
		return
	}

	m.tree.SelectNode(node)
	m.saveState() // Persist selection
	m.updateDetailContent()
	m.previousPane = m.focusedPane
	m.focusedPane = PanelDetail

	m.status = fmt.Sprintf("Forward: %s (%d/%d)", node.Item.Name, m.historyIndex+1, len(m.navigationHistory))
}

// scrollDetailViewportToFocusedRow scrolls the detail viewport to ensure the focused grid row is visible.
func (m *Model) scrollDetailViewportToFocusedRow() {
	if m.lightGrid.FocusRow() < 0 {
		return
	}

	// Get the Y position of the focused row
	focusY := m.lightGrid.FocusedRowYPosition()

	// Get viewport dimensions
	viewportHeight := m.detailViewport.Height()
	currentOffset := m.detailViewport.YOffset

	// Add some padding to keep the row visible with context
	const padding = 2

	// Check if focused row is above visible area
	if focusY < currentOffset+padding {
		newOffset := focusY - padding
		if newOffset < 0 {
			newOffset = 0
		}
		m.detailViewport.SetYOffset(newOffset)
	}

	// Check if focused row is below visible area
	if focusY >= currentOffset+viewportHeight-padding {
		newOffset := focusY - viewportHeight + padding + 1
		m.detailViewport.SetYOffset(newOffset)
	}
}
