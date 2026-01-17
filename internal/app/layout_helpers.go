package app

import (
	"fmt"

	"github.com/kluzzebass/lazyhue/internal/ui/layout"
)

// getPanelKey returns the keyboard shortcut for a panel based on its position in panelOrder.
func (m *Model) getPanelKey(panelID string) string {
	for i, id := range m.panelOrder {
		if id == panelID {
			return fmt.Sprintf("%d", i+1)
		}
	}
	return ""
}

// rebuildLayout rebuilds the layout tree based on current visibility settings.
func (m *Model) rebuildLayout() {
	var layoutRoot layout.Node

	// Tree panel: 40% of width, capped at the maximum content width
	// This prevents wasted space on wide terminals
	maxTreeWidth := max(m.tree.MaxContentWidth(), 30) // 30 = minimum usable width
	treeSizeSpec := layout.FlexWithConstraints(0.4, 0, maxTreeWidth)

	if m.showActivity {
		// Full layout with activity log
		layoutRoot = layout.HSplit(
			layout.Child{Size: treeSizeSpec, Node: layout.NewLeaf(PanelTree)},
			layout.Child{Size: layout.Flex(0.6), Node: layout.VSplit(
				layout.Child{Size: layout.Flex(0.67), Node: layout.NewLeaf(PanelDetail)},
				layout.Child{Size: layout.Flex(0.33), Node: layout.NewLeaf(PanelLog)},
			)},
		)
	} else {
		// Layout without activity log - just tree and detail
		layoutRoot = layout.HSplit(
			layout.Child{Size: treeSizeSpec, Node: layout.NewLeaf(PanelTree)},
			layout.Child{Size: layout.Flex(0.6), Node: layout.NewLeaf(PanelDetail)},
		)
	}

	m.layout = layout.NewTree(layoutRoot)

	// Re-apply current dimensions
	if m.width > 0 && m.height > 0 {
		helpHeight := 1
		contentHeight := m.height - helpHeight
		m.layout.Layout(m.width, contentHeight)

		// Update panel sizes
		treeBounds := m.layout.Bounds(PanelTree)
		m.tree.SetSize(treeBounds.Width, treeBounds.Height)

		detailBounds := m.layout.Bounds(PanelDetail)
		detailWidth := detailBounds.Width - 2
		if detailWidth < 1 {
			detailWidth = 1
		}
		detailHeight := detailBounds.Height - 2
		if detailHeight < 1 {
			detailHeight = 1
		}
		m.detailViewport.SetWidth(detailWidth)
		m.detailViewport.SetHeight(detailHeight)

		if m.showActivity {
			logBounds := m.layout.Bounds(PanelLog)
			logWidth := logBounds.Width - 2
			if logWidth < 1 {
				logWidth = 1
			}
			logHeight := logBounds.Height - 2
			if logHeight < 1 {
				logHeight = 1
			}
			m.logViewport.SetWidth(logWidth)
			m.logViewport.SetHeight(logHeight)
		}

		// Re-render content with new dimensions
		m.updateDetailContent()
		m.updateLogContent()
	}
}

// rebuildTreeForActiveTab rebuilds the tree for the active tab, showing all bridges.
func (m *Model) rebuildTreeForActiveTab() {
	tabID := m.tree.ActiveTabID()

	switch tabID {
	case "home":
		m.buildHomeTree(nil)
	case "lights":
		m.buildLightsTree(nil)
	case "devices":
		m.buildDevicesTree(nil)
	case "scenes":
		m.buildScenesTree(nil)
	default:
		m.buildHomeTree(nil)
	}

	// Try to apply pending selection if we have one
	if m.pendingSelectionID != "" {
		if m.tree.SelectByID(m.pendingSelectionID) {
			m.pendingSelectionID = ""
			m.updateDetailContent()
		}
	}
}
