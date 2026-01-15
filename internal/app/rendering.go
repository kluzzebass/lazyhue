package app

import (
	"fmt"
	"math"
	"sort"
	"strings"

	"image/color"

	"github.com/charmbracelet/lipgloss/v2"
	"github.com/kluzzebass/lazyhue/internal/hue"
	"github.com/kluzzebass/lazyhue/internal/hueclient"
	"github.com/kluzzebass/lazyhue/internal/ui"
	"github.com/kluzzebass/lazyhue/internal/ui/component/field"
	gridlayout "github.com/kluzzebass/lazyhue/internal/ui/component/layout"
	"github.com/kluzzebass/lazyhue/internal/ui/panels"
)

// getSortedProductArchetypes returns a sorted list of product archetype keys from the display names map.
// This ensures the dropdown and value handling use the same order.
func getSortedProductArchetypes() []string {
	archetypes := make([]string, 0, len(hue.ProductArchetypeDisplayNames))
	for key := range hue.ProductArchetypeDisplayNames {
		archetypes = append(archetypes, key)
	}
	sort.Strings(archetypes)
	return archetypes
}


// getSortedPowerupPresets returns a sorted list of power-on preset keys from the display names map.
// This ensures the dropdown and value handling use the same order.
func getSortedPowerupPresets() []string {
	presets := make([]string, 0, len(hue.PowerupPresetDisplayNames))
	for key := range hue.PowerupPresetDisplayNames {
		presets = append(presets, key)
	}
	sort.Strings(presets)
	return presets
}

// buildHelpContent builds the help content dynamically from bindings.
func (m *Model) buildHelpContent() string {
	var content strings.Builder

	// Header
	content.WriteString(m.styles.Title.Render("Keyboard Shortcuts"))
	content.WriteString("\n\n")

	// Always show Tree panel bindings if there's a selection
	// This is more useful than showing focused panel bindings
	node := m.tree.SelectedNode()
	var entityType panels.EntityType
	if node != nil && node.Item != nil {
		entityType = node.Item.Type
	}

	// Get tree bindings based on selected entity
	var treeBindings []Binding
	var treeTitle string

	if node != nil && node.Item != nil {
		treeBindings = make([]Binding, 0, len(m.panelBindings[PanelTree]))

		switch entityType {
		case panels.EntityBridge:
			treeTitle = "Tree (Bridge selected)"
			// Include all bindings including 'x' for bridge deletion
			treeBindings = append(treeBindings, m.panelBindings[PanelTree]...)
		case panels.EntityRoom:
			treeTitle = "Tree (Room selected)"
			// Include all bindings including 'x' for room deletion
			treeBindings = append(treeBindings, m.panelBindings[PanelTree]...)
		case panels.EntityZone:
			treeTitle = "Tree (Zone selected)"
			// Include all bindings including 'x' for zone deletion
			treeBindings = append(treeBindings, m.panelBindings[PanelTree]...)
		default:
			// For other entities (lights, devices, scenes), exclude 'x' deletion binding
			for _, b := range m.panelBindings[PanelTree] {
				isDeleteBinding := false
				for _, key := range b.Keys {
					if key == "x" {
						isDeleteBinding = true
						break
					}
				}
				if !isDeleteBinding {
					treeBindings = append(treeBindings, b)
				}
			}

			// Set appropriate title
			switch entityType {
			case panels.EntityLight:
				treeTitle = "Tree (Light selected)"
			case panels.EntityScene:
				treeTitle = "Tree (Scene selected)"
			case panels.EntityDevice:
				treeTitle = "Tree (Device selected)"
			default:
				treeTitle = "Tree"
			}
		}
	}

	// Show tree bindings
	if len(treeBindings) > 0 && treeTitle != "" {
		content.WriteString(m.styles.Subtitle.Render(treeTitle + ":"))
		content.WriteString("\n")
		for _, b := range treeBindings {
			content.WriteString(fmt.Sprintf("  %-12s %s\n", b.Display, b.Desc))
		}
		content.WriteString("\n")
	}

	// Get bindings for currently focused panel (if not tree)
	var focusedPanelBindings []Binding
	var focusedPanelTitle string

	if m.focusedPane != PanelTree {
		focusedPanelBindings, _, focusedPanelTitle = m.getContextBindings()

		// Show focused panel bindings
		if len(focusedPanelBindings) > 0 && focusedPanelTitle != "" {
			content.WriteString(m.styles.Subtitle.Render(focusedPanelTitle + ":"))
			content.WriteString("\n")
			for _, b := range focusedPanelBindings {
				content.WriteString(fmt.Sprintf("  %-12s %s\n", b.Display, b.Desc))
			}
			content.WriteString("\n")
		}
	}

	// Get global bindings
	globalBindings := m.globalBindings

	// Global bindings - group by category
	navBindings := []Binding{}
	panelNavBindings := []Binding{}
	tabBindings := []Binding{}
	uiBindings := []Binding{}

	for _, b := range globalBindings {
		if contains(b.Keys, "up") || contains(b.Keys, "down") || contains(b.Keys, "g") ||
		   contains(b.Keys, "G") || contains(b.Keys, "pgup") || contains(b.Keys, "pgdown") {
			navBindings = append(navBindings, b)
		} else if contains(b.Keys, "tab") || contains(b.Keys, "shift+tab") ||
		          contains(b.Keys, "0") || contains(b.Keys, "1") || contains(b.Keys, "2") {
			panelNavBindings = append(panelNavBindings, b)
		} else if contains(b.Keys, "[") || contains(b.Keys, "]") {
			tabBindings = append(tabBindings, b)
		} else {
			uiBindings = append(uiBindings, b)
		}
	}

	// Navigation
	if len(navBindings) > 0 {
		content.WriteString(m.styles.Subtitle.Render("Navigation:"))
		content.WriteString("\n")
		for _, b := range navBindings {
			content.WriteString(fmt.Sprintf("  %-12s %s\n", b.Display, b.Desc))
		}
		content.WriteString("\n")
	}

	// Panel navigation
	if len(panelNavBindings) > 0 {
		content.WriteString(m.styles.Subtitle.Render("Panels:"))
		content.WriteString("\n")
		for _, b := range panelNavBindings {
			content.WriteString(fmt.Sprintf("  %-12s %s\n", b.Display, b.Desc))
		}
		content.WriteString("\n")
	}

	// Tab navigation
	if len(tabBindings) > 0 {
		content.WriteString(m.styles.Subtitle.Render("Tabs:"))
		content.WriteString("\n")
		for _, b := range tabBindings {
			content.WriteString(fmt.Sprintf("  %-12s %s\n", b.Display, b.Desc))
		}
		content.WriteString("\n")
	}

	// UI controls
	if len(uiBindings) > 0 {
		content.WriteString(m.styles.Subtitle.Render("UI:"))
		content.WriteString("\n")
		for _, b := range uiBindings {
			content.WriteString(fmt.Sprintf("  %-12s %s\n", b.Display, b.Desc))
		}
		content.WriteString("\n")
	}

	// Footer
	content.WriteString(m.styles.Dimmed.Render("Press Esc or ? to close"))
	content.WriteString("\n")

	return content.String()
}

// contains checks if a slice contains a string.
func contains(slice []string, str string) bool {
	for _, s := range slice {
		if s == str {
			return true
		}
	}
	return false
}

// buildRenameContent builds the content for the rename input view.
func (m *Model) buildRenameContent() string {
	var content strings.Builder

	// Header
	content.WriteString(m.styles.Subtitle.Render(fmt.Sprintf("Rename %s", m.renameEntityType.String())))
	content.WriteString("\n\n")

	// Original name
	content.WriteString(fmt.Sprintf("  Current: %s\n\n", m.renameOriginalName))

	// Text input - manually add cursor since virtual cursor isn't rendering
	renameInput := m.renameModal.Input()
	inputValue := renameInput.Value()
	cursorPos := renameInput.Position()

	// Build the input display with a visible cursor
	// Convert to runes to handle multi-byte UTF-8 characters properly
	runes := []rune(inputValue)
	var displayValue string

	if renameInput.Focused() {
		// Insert a block cursor at the cursor position
		if cursorPos >= len(runes) {
			// Cursor at end - append block
			displayValue = inputValue + "█"
		} else {
			// Cursor in middle - insert block between characters
			beforeCursor := string(runes[:cursorPos])
			afterCursor := string(runes[cursorPos:])
			displayValue = beforeCursor + "█" + afterCursor
		}
	} else {
		displayValue = inputValue
	}

	content.WriteString("  ")
	content.WriteString(renameInput.Prompt)
	content.WriteString(displayValue)
	content.WriteString("\n\n")

	// Instructions
	content.WriteString(m.styles.Dimmed.Render("  Enter to save, Esc to cancel"))
	content.WriteString("\n")

	return content.String()
}

// buildCreateContent builds the create room/zone input dialog.
func (m *Model) buildCreateContent() string {
	var content strings.Builder

	// Header
	entityType := "Room"
	if m.creatingZone {
		entityType = "Zone"
	}
	content.WriteString(m.styles.Subtitle.Render(fmt.Sprintf("Create %s", entityType)))
	content.WriteString("\n\n")

	// Get bridge name
	bridgeName := ""
	if bridge := m.manager.GetBridge(m.createBridgeID); bridge != nil {
		bridgeName = bridge.Info.Name
	}
	content.WriteString(fmt.Sprintf("  Bridge: %s\n\n", m.styles.Dimmed.Render(bridgeName)))

	// Text input - manually add cursor since virtual cursor isn't rendering
	createInput := m.createModal.Input()
	inputValue := createInput.Value()
	cursorPos := createInput.Position()

	// Build the input display with a visible cursor
	// Convert to runes to handle multi-byte UTF-8 characters properly
	runes := []rune(inputValue)
	var displayValue string

	if createInput.Focused() {
		// Insert a block cursor at the cursor position
		if cursorPos >= len(runes) {
			// Cursor at end - append block
			displayValue = inputValue + "█"
		} else {
			// Cursor in middle - insert block between characters
			beforeCursor := string(runes[:cursorPos])
			afterCursor := string(runes[cursorPos:])
			displayValue = beforeCursor + "█" + afterCursor
		}
	} else {
		displayValue = inputValue
	}

	content.WriteString("  ")
	content.WriteString(createInput.Prompt)
	content.WriteString(displayValue)
	content.WriteString("\n\n")

	// Instructions
	content.WriteString(m.styles.Dimmed.Render("  Enter to create, Esc to cancel"))
	content.WriteString("\n")

	return content.String()
}

// buildDeleteConfirmationContent builds the bridge deletion confirmation dialog.
func (m *Model) buildDeleteConfirmationContent() string {
	var content strings.Builder

	// Header with warning color
	warningStyle := lipgloss.NewStyle().Foreground(m.styles.Theme.Error).Bold(true)
	content.WriteString(warningStyle.Render("Delete Bridge"))
	content.WriteString("\n\n")

	// Bridge name and ID
	content.WriteString(fmt.Sprintf("  Bridge: %s\n", m.styles.Highlight.Render(m.deleteBridgeName)))
	content.WriteString(fmt.Sprintf("      ID: %s\n\n", m.styles.Dimmed.Render(m.deleteBridgeID)))

	// Warning message
	content.WriteString(m.styles.Dimmed.Render("  This will remove the bridge and its credentials."))
	content.WriteString("\n")
	content.WriteString(m.styles.Dimmed.Render("  You will need to re-pair to use this bridge again."))
	content.WriteString("\n\n")

	// Confirmation prompt
	content.WriteString("  ")
	content.WriteString(warningStyle.Render("Delete this bridge?"))
	content.WriteString("\n\n")

	// Instructions
	yesStyle := lipgloss.NewStyle().Foreground(m.styles.Theme.Error).Bold(true)
	noStyle := lipgloss.NewStyle().Foreground(m.styles.Theme.Success).Bold(true)
	content.WriteString("  ")
	content.WriteString(yesStyle.Render("Y"))
	content.WriteString(" = Yes    ")
	content.WriteString(noStyle.Render("N"))
	content.WriteString(" = No (Esc)")
	content.WriteString("\n")

	return content.String()
}

// buildEntityDeleteConfirmationContent builds the entity deletion confirmation dialog.
func (m *Model) buildEntityDeleteConfirmationContent() string {
	var content strings.Builder

	// Header with warning color
	warningStyle := lipgloss.NewStyle().Foreground(m.styles.Theme.Error).Bold(true)

	// For lights, we're actually deleting the device (factory reset)
	if m.deleteEntityType == panels.EntityLight {
		content.WriteString(warningStyle.Render("Delete Light (Factory Reset Device)"))
		content.WriteString("\n\n")

		// Entity name and device ID
		content.WriteString(fmt.Sprintf("  Light: %s\n", m.styles.Highlight.Render(m.deleteEntityName)))
		content.WriteString(fmt.Sprintf("  Device ID: %s\n\n", m.styles.Dimmed.Render(m.deleteEntityID)))

		// Warning message
		content.WriteString(warningStyle.Render("  WARNING: This will factory reset the device!"))
		content.WriteString("\n")
		content.WriteString(m.styles.Dimmed.Render("  The device will be removed from this bridge and"))
		content.WriteString("\n")
		content.WriteString(m.styles.Dimmed.Render("  must be re-paired to use again."))
		content.WriteString("\n\n")

		// Confirmation prompt
		content.WriteString("  ")
		content.WriteString(warningStyle.Render("Factory reset this device?"))
		content.WriteString("\n\n")
	} else {
		content.WriteString(warningStyle.Render(fmt.Sprintf("Delete %s", m.deleteEntityType.String())))
		content.WriteString("\n\n")

		// Entity name and ID
		content.WriteString(fmt.Sprintf("  %s: %s\n", m.deleteEntityType.String(), m.styles.Highlight.Render(m.deleteEntityName)))
		content.WriteString(fmt.Sprintf("     ID: %s\n\n", m.styles.Dimmed.Render(m.deleteEntityID)))

		// Warning message
		content.WriteString(m.styles.Dimmed.Render(fmt.Sprintf("  This will permanently delete this %s.", m.deleteEntityType.String())))
		content.WriteString("\n\n")

		// Confirmation prompt
		content.WriteString("  ")
		content.WriteString(warningStyle.Render(fmt.Sprintf("Delete this %s?", m.deleteEntityType.String())))
		content.WriteString("\n\n")
	}

	// Instructions
	yesStyle := lipgloss.NewStyle().Foreground(m.styles.Theme.Error).Bold(true)
	noStyle := lipgloss.NewStyle().Foreground(m.styles.Theme.Success).Bold(true)
	content.WriteString("  ")
	content.WriteString(yesStyle.Render("Y"))
	content.WriteString(" = Yes    ")
	content.WriteString(noStyle.Render("N"))
	content.WriteString(" = No (Esc)")
	content.WriteString("\n")

	return content.String()
}

// buildPairingContent builds the bridge pairing dialog.
func (m *Model) buildPairingContent() string {
	var content strings.Builder

	// Header
	titleStyle := lipgloss.NewStyle().Foreground(m.styles.Theme.Primary).Bold(true)
	content.WriteString(titleStyle.Render("Pair with Bridge"))
	content.WriteString("\n\n")

	if m.pairingFor != nil {
		// Bridge info
		content.WriteString(fmt.Sprintf("  Bridge: %s\n", m.styles.Highlight.Render(m.pairingFor.Name)))
		content.WriteString(fmt.Sprintf("      IP: %s\n\n", m.styles.Dimmed.Render(m.pairingFor.IPAddress)))

		// Instructions
		content.WriteString(m.styles.Subtitle.Render("  Press the button on your bridge"))
		content.WriteString("\n\n")

		// Progress indicator with countdown
		remaining := m.pairingRemaining
		if remaining < 0 {
			remaining = 0
		}

		// Progress bar
		totalWidth := 40
		filled := int(float64(totalWidth) * float64(remaining) / 60.0)
		if filled < 0 {
			filled = 0
		}
		if filled > totalWidth {
			filled = totalWidth
		}

		bar := strings.Repeat("█", filled) + strings.Repeat("░", totalWidth-filled)
		content.WriteString("  ")
		content.WriteString(bar)
		content.WriteString("\n\n")

		// Countdown
		content.WriteString(fmt.Sprintf("  Time remaining: %d seconds\n\n", remaining))

		// Cancel instruction
		content.WriteString(m.styles.Dimmed.Render("  Press Esc to cancel"))
		content.WriteString("\n")
	} else {
		content.WriteString("  No bridge selected for pairing\n")
	}

	return content.String()
}

// updateLogContent updates the log viewport content from activities.
func (m *Model) updateLogContent() {
	var content strings.Builder
	logBounds := m.layout.Bounds(PanelLog)
	contentWidth := logBounds.Width - 2 // Account for border
	if contentWidth < 1 {
		contentWidth = 1
	}

	// Check if user was at the bottom before updating content
	wasAtBottom := m.logViewport.AtBottom()

	for i, activity := range m.activities {
		line := activity.Render(&m.styles, contentWidth)
		content.WriteString(line)
		if i < len(m.activities)-1 {
			content.WriteString("\n")
		}
	}
	m.logViewport.SetContent(content.String())

	// Only auto-scroll to bottom if user was already at the bottom
	if wasAtBottom {
		m.logViewport.GotoBottom()
	}
}

// updateDetailContent updates the detail viewport content based on the selected node.
func (m *Model) updateDetailContent() {
	// If showing help, display help content
	if m.showHelp {
		m.detailViewport.SetContent(m.buildHelpContent())
		return
	}

	// If showing delete confirmation, display confirmation dialog
	if m.confirmingDelete {
		m.detailViewport.SetContent(m.buildDeleteConfirmationContent())
		return
	}

	// If showing entity delete confirmation, display confirmation dialog
	if m.confirmingDeleteEntity {
		m.detailViewport.SetContent(m.buildEntityDeleteConfirmationContent())
		return
	}

	// If in rename mode, show rename input
	if m.renaming {
		m.detailViewport.SetContent(m.buildRenameContent())
		return
	}

	// If in create room/zone mode, show create form
	if m.creatingRoom || m.creatingZone {
		m.detailViewport.SetContent(m.buildCreateContent())
		return
	}

	// If in pairing mode, show pairing dialog
	if m.pairing {
		m.detailViewport.SetContent(m.buildPairingContent())
		return
	}

	node := m.tree.SelectedNode()
	if node == nil || node.Item == nil {
		m.detailViewport.SetContent("No selection")
		return
	}

	// Get bridge state from the entity's bridge (not active bridge)
	var state *hue.BridgeState
	bridgeID := node.Item.BridgeID
	if bridgeID != "" {
		bridge := m.manager.GetBridge(bridgeID)
		if bridge != nil {
			state = bridge.GetState()
		}
	}

	// Build content based on entity type
	var content strings.Builder
	switch node.Item.Type {
	case panels.EntityLight:
		// Get light for form fields
		var light hueclient.LightGet
		var ok bool
		if node.Item.RawPtr != nil {
			if l, typeOk := node.Item.RawPtr.(hueclient.LightGet); typeOk {
				light = l
				ok = true
			}
		}
		if !ok && state != nil {
			light, ok = state.GetLight(node.Item.ID)
		}

		if ok {
			// Get the actual light ID from the light object
			actualLightID := ""
			if light.Id != nil {
				actualLightID = *light.Id
			} else {
				actualLightID = node.Item.ID
			}

			// Build grid rows if light changed or grid is empty
			if m.selectedLightID != actualLightID || len(m.lightGrid.Children()) == 0 {
				// Build grid rows for the light controls
				m.buildLightGridRows(light)
				m.selectedLightID = actualLightID
			}

			// Render the grid
			if len(m.lightGrid.Children()) > 0 {
				gridContent := m.lightGrid.View()
				content.WriteString(gridContent)
			}
		}
	case panels.EntityRoom:
		m.selectedLightID = ""
		var room hueclient.RoomGet
		var ok bool
		if node.Item.RawPtr != nil {
			if r, typeOk := node.Item.RawPtr.(hueclient.RoomGet); typeOk {
				room = r
				ok = true
			}
		}
		if !ok && state != nil {
			room, ok = state.GetRoom(node.Item.ID)
		}
		if ok {
			rows := m.buildRoomGridRows(room, false, state)
			m.lightGrid.SetRows(rows)
			if len(m.lightGrid.Children()) > 0 {
				content.WriteString(m.lightGrid.View())
			}
		}

	case panels.EntityZone:
		m.selectedLightID = ""
		var zone hueclient.RoomGet
		var ok bool
		if node.Item.RawPtr != nil {
			if z, typeOk := node.Item.RawPtr.(hueclient.RoomGet); typeOk {
				zone = z
				ok = true
			}
		}
		if !ok && state != nil {
			zone, ok = state.GetZone(node.Item.ID)
		}
		if ok {
			rows := m.buildRoomGridRows(zone, true, state)
			m.lightGrid.SetRows(rows)
			if len(m.lightGrid.Children()) > 0 {
				content.WriteString(m.lightGrid.View())
			}
		}

	case panels.EntityScene:
		m.selectedLightID = ""
		var scene hueclient.SceneGet
		var ok bool
		if node.Item.RawPtr != nil {
			if s, typeOk := node.Item.RawPtr.(hueclient.SceneGet); typeOk {
				scene = s
				ok = true
			}
		}
		if !ok && state != nil {
			scene, ok = state.GetScene(node.Item.ID)
		}
		if ok {
			rows := m.buildSceneGridRows(scene, state)
			m.lightGrid.SetRows(rows)
			if len(m.lightGrid.Children()) > 0 {
				content.WriteString(m.lightGrid.View())
			}
		}

	case panels.EntitySmartScene:
		m.selectedLightID = ""
		var scene hueclient.SmartSceneGet
		var ok bool
		if node.Item.RawPtr != nil {
			if s, typeOk := node.Item.RawPtr.(hueclient.SmartSceneGet); typeOk {
				scene = s
				ok = true
			}
		}
		if !ok && state != nil {
			scene, ok = state.GetSmartScene(node.Item.ID)
		}
		if ok {
			rows := m.buildSmartSceneGridRows(scene, state, bridgeID)
			m.lightGrid.SetRows(rows)
			if len(m.lightGrid.Children()) > 0 {
				content.WriteString(m.lightGrid.View())
			}
		}

	case panels.EntityDevice:
		m.selectedLightID = ""
		var device hueclient.DeviceGet
		var ok bool
		if node.Item.RawPtr != nil {
			if d, typeOk := node.Item.RawPtr.(hueclient.DeviceGet); typeOk {
				device = d
				ok = true
			}
		}
		if !ok && state != nil {
			device, ok = state.GetDevice(node.Item.ID)
		}
		if ok {
			rows := m.buildDeviceGridRows(device, state)
			m.lightGrid.SetRows(rows)
			if len(m.lightGrid.Children()) > 0 {
				content.WriteString(m.lightGrid.View())
			}
		}

	case panels.EntityBridge:
		m.selectedLightID = ""
		rows := m.buildBridgeGridRows(bridgeID)
		m.lightGrid.SetRows(rows)
		if len(m.lightGrid.Children()) > 0 {
			content.WriteString(m.lightGrid.View())
		}

	case panels.EntityEntertainment:
		m.selectedLightID = ""
		var ent hue.EntertainmentConfiguration
		var ok bool
		if node.Item.RawPtr != nil {
			if e, typeOk := node.Item.RawPtr.(hue.EntertainmentConfiguration); typeOk {
				ent = e
				ok = true
			}
		}
		if !ok && state != nil {
			ent, ok = state.GetEntertainmentConfiguration(node.Item.ID)
		}
		if ok {
			rows := m.buildEntertainmentGridRows(ent, state)
			m.lightGrid.SetRows(rows)
			if len(m.lightGrid.Children()) > 0 {
				content.WriteString(m.lightGrid.View())
			}
		}

	case panels.EntityRoomsCategory:
		m.selectedLightID = ""
		if data, ok := node.Item.RawPtr.(panels.RoomsCategoryData); ok {
			rows := m.buildRoomsCategoryGridRows(data, state)
			m.lightGrid.SetRows(rows)
			if len(m.lightGrid.Children()) > 0 {
				content.WriteString(m.lightGrid.View())
			}
		}

	case panels.EntityZonesCategory:
		m.selectedLightID = ""
		if data, ok := node.Item.RawPtr.(panels.ZonesCategoryData); ok {
			rows := m.buildZonesCategoryGridRows(data, state)
			m.lightGrid.SetRows(rows)
			if len(m.lightGrid.Children()) > 0 {
				content.WriteString(m.lightGrid.View())
			}
		}

	case panels.EntityLightsCategory:
		m.selectedLightID = ""
		if data, ok := node.Item.RawPtr.(panels.LightsCategoryData); ok {
			rows := m.buildLightsCategoryGridRows(data, state)
			m.lightGrid.SetRows(rows)
			if len(m.lightGrid.Children()) > 0 {
				content.WriteString(m.lightGrid.View())
			}
		}

	case panels.EntityDevicesCategory:
		m.selectedLightID = ""
		if data, ok := node.Item.RawPtr.(panels.DevicesCategoryData); ok {
			rows := m.buildDevicesCategoryGridRows(data, state)
			m.lightGrid.SetRows(rows)
			if len(m.lightGrid.Children()) > 0 {
				content.WriteString(m.lightGrid.View())
			}
		}

	case panels.EntityScenesCategory:
		m.selectedLightID = ""
		if data, ok := node.Item.RawPtr.(panels.ScenesCategoryData); ok {
			rows := m.buildScenesCategoryGridRows(data, state)
			m.lightGrid.SetRows(rows)
			if len(m.lightGrid.Children()) > 0 {
				content.WriteString(m.lightGrid.View())
			}
		}

	case panels.EntitySmartScenesCategory:
		m.selectedLightID = ""
		if data, ok := node.Item.RawPtr.(panels.SmartScenesCategoryData); ok {
			rows := m.buildSmartScenesCategoryGridRows(data, state)
			m.lightGrid.SetRows(rows)
			if len(m.lightGrid.Children()) > 0 {
				content.WriteString(m.lightGrid.View())
			}
		}

	case panels.EntityEntertainmentCategory:
		m.selectedLightID = ""
		if data, ok := node.Item.RawPtr.(panels.EntertainmentCategoryData); ok {
			rows := m.buildEntertainmentCategoryGridRows(data)
			m.lightGrid.SetRows(rows)
			if len(m.lightGrid.Children()) > 0 {
				content.WriteString(m.lightGrid.View())
			}
		}

	default:
		// Clear grid for unknown entities
		m.lightGrid.SetRows(nil)
		m.selectedLightID = ""
		content.WriteString(fmt.Sprintf("Type: %s\n", node.Item.Type))
		content.WriteString(fmt.Sprintf("ID: %s\n", node.Item.ID))
	}

	m.detailViewport.SetContent(content.String())
}

// renderDetailPanel renders the detail panel with border and header.
func (m *Model) renderDetailPanel(width, height int, focused bool, key string) string {
	borderColor := m.styles.Theme.Border
	if focused {
		borderColor = m.styles.Theme.Accent
	}

	// Get the panel title based on current mode
	title := "Details"
	if m.showHelp {
		title = "Help"
	} else if m.renaming {
		title = "Rename"
	} else {
		node := m.tree.SelectedNode()
		if node != nil && node.Item != nil && node.Item.Name != "" {
			title = node.Item.Name
		}
	}

	topBorder := m.renderPanelHeader(width, key, title, borderColor)

	innerWidth := width - 2
	if innerWidth < 1 {
		innerWidth = 1
	}
	innerHeight := height - 2
	if innerHeight < 1 {
		innerHeight = 1
	}

	// Get content from viewport (includes details + form if light is selected)
	content := m.detailViewport.View()
	contentLines := strings.Split(content, "\n")
	// Remove trailing empty line if present
	if len(contentLines) > 0 && contentLines[len(contentLines)-1] == "" {
		contentLines = contentLines[:len(contentLines)-1]
	}
	if len(contentLines) > innerHeight {
		contentLines = contentLines[:innerHeight]
	}

	border := lipgloss.RoundedBorder()
	borderStyleColor := lipgloss.NewStyle().Foreground(borderColor)

	leftBorder := borderStyleColor.Render(border.Left)
	bottomBorder := borderStyleColor.Render(border.BottomLeft) +
		borderStyleColor.Render(strings.Repeat(border.Bottom, innerWidth)) +
		borderStyleColor.Render(border.BottomRight)

	// Calculate scrollbar
	scrollPos := m.detailViewport.YOffset
	totalHeight := m.detailViewport.TotalLineCount()
	viewHeight := m.detailViewport.VisibleLineCount()
	rightBorders := ui.BuildRightBorderWithScrollbar(border, innerHeight, borderColor, scrollPos, totalHeight, viewHeight)

	var lines []string
	lines = append(lines, topBorder)

	for i, line := range contentLines {
		if lipgloss.Width(line) > innerWidth {
			line = lipgloss.Place(innerWidth, 1, lipgloss.Left, lipgloss.Top, line)
		}
		paddedLine := lipgloss.Place(innerWidth, 1, lipgloss.Left, lipgloss.Top, line)
		rightBorder := borderStyleColor.Render(border.Right)
		if i < len(rightBorders) {
			rightBorder = rightBorders[i]
		}
		lines = append(lines, leftBorder+paddedLine+rightBorder)
	}

	// Fill to exact height (1 for top border + content + 1 for bottom border)
	targetHeight := height
	contentIdx := len(contentLines)
	for len(lines) < targetHeight-1 {
		paddedLine := strings.Repeat(" ", innerWidth)
		rightBorder := borderStyleColor.Render(border.Right)
		if contentIdx < len(rightBorders) {
			rightBorder = rightBorders[contentIdx]
		}
		lines = append(lines, leftBorder+paddedLine+rightBorder)
		contentIdx++
	}

	lines = append(lines, bottomBorder)

	return strings.Join(lines, "\n")
}

// renderLogPanel renders the log panel with border and header.
func (m *Model) renderLogPanel(width, height int, focused bool, key string) string {
	borderColor := m.styles.Theme.Border
	if focused {
		borderColor = m.styles.Theme.Accent
	}

	topBorder := m.renderPanelHeader(width, key, "Activity", borderColor)

	innerWidth := width - 2
	if innerWidth < 1 {
		innerWidth = 1
	}
	innerHeight := height - 2
	if innerHeight < 1 {
		innerHeight = 1
	}

	content := m.logViewport.View()
	// Split content - viewport may add trailing newline
	contentLines := strings.Split(content, "\n")
	// Remove trailing empty line if present
	if len(contentLines) > 0 && contentLines[len(contentLines)-1] == "" {
		contentLines = contentLines[:len(contentLines)-1]
	}
	// Viewport should return exactly innerHeight lines, but cap it just in case
	if len(contentLines) > innerHeight {
		contentLines = contentLines[:innerHeight]
	}

	border := lipgloss.RoundedBorder()
	borderStyleColor := lipgloss.NewStyle().Foreground(borderColor)

	leftBorder := borderStyleColor.Render(border.Left)
	bottomBorder := borderStyleColor.Render(border.BottomLeft) +
		borderStyleColor.Render(strings.Repeat(border.Bottom, innerWidth)) +
		borderStyleColor.Render(border.BottomRight)

	// Calculate scrollbar
	scrollPos := m.logViewport.YOffset
	totalHeight := m.logViewport.TotalLineCount()
	viewHeight := m.logViewport.VisibleLineCount()
	rightBorders := ui.BuildRightBorderWithScrollbar(border, innerHeight, borderColor, scrollPos, totalHeight, viewHeight)

	var lines []string
	lines = append(lines, topBorder)

	// Render exactly the lines the viewport provides (should be innerHeight)
	// Limit to innerHeight to prevent overflow
	// Activity.Render() already handles truncation, so just pad to width
	for i := 0; i < len(contentLines) && i < innerHeight; i++ {
		line := contentLines[i]
		paddedLine := lipgloss.Place(innerWidth, 1, lipgloss.Left, lipgloss.Top, line)
		rightBorder := borderStyleColor.Render(border.Right)
		if i < len(rightBorders) {
			rightBorder = rightBorders[i]
		}
		lines = append(lines, leftBorder+paddedLine+rightBorder)
	}

	// Fill to exact height (1 for top border + content + 1 for bottom border)
	targetHeight := height
	contentIdx := len(contentLines)
	for len(lines) < targetHeight-1 {
		paddedLine := strings.Repeat(" ", innerWidth)
		rightBorder := borderStyleColor.Render(border.Right)
		if contentIdx < len(rightBorders) {
			rightBorder = rightBorders[contentIdx]
		}
		lines = append(lines, leftBorder+paddedLine+rightBorder)
		contentIdx++
	}

	lines = append(lines, bottomBorder)

	return strings.Join(lines, "\n")
}

// renderPanelHeader renders a panel header with hotkey and title.
func (m *Model) renderPanelHeader(width int, keyStr, title string, borderColor color.Color) string {
	border := lipgloss.RoundedBorder()

	keyRendered := ""
	keyWidth := 0
	if keyStr != "" {
		keyStyle := lipgloss.NewStyle().
			Foreground(m.styles.Theme.Primary).
			Bold(true)
		keyRendered = keyStyle.Render("[" + keyStr + "]")
		keyWidth = lipgloss.Width(keyRendered)
	}

	titleStyle := lipgloss.NewStyle().
		Foreground(m.styles.Theme.Primary).
		Bold(true)
	titleRendered := titleStyle.Render(title)
	titleWidth := lipgloss.Width(titleRendered)

	leftPadding := 1
	middlePadding := 1
	remainingWidth := width - keyWidth - titleWidth - leftPadding - middlePadding - 2

	if remainingWidth < 0 {
		remainingWidth = 0
	}

	borderStyle := lipgloss.NewStyle().Foreground(borderColor)

	if keyRendered != "" {
		return borderStyle.Render(border.TopLeft) +
			borderStyle.Render(strings.Repeat(border.Top, leftPadding)) +
			keyRendered +
			borderStyle.Render(strings.Repeat(border.Top, middlePadding)) +
			titleRendered +
			borderStyle.Render(strings.Repeat(border.Top, remainingWidth)) +
			borderStyle.Render(border.TopRight)
	}

	return borderStyle.Render(border.TopLeft) +
		borderStyle.Render(strings.Repeat(border.Top, leftPadding)) +
		titleRendered +
		borderStyle.Render(strings.Repeat(border.Top, remainingWidth+middlePadding)) +
		borderStyle.Render(border.TopRight)
}

// buildLightGridRows builds grid rows for a light's full details panel.
// This uses the Grid layout with sections for product info, controls, state, etc.
// Sections are ordered by immediacy: Controls > Settings > Info > Technical
func (m *Model) buildLightGridRows(light hueclient.LightGet) {
	var rows []gridlayout.GridRow

	// Get owning device for product info
	device := m.getDeviceForLight(light)

	// 1. Controls section - instant adjustments (power, brightness, color, effects, identify)
	rows = append(rows, m.buildControlsRows(light)...)

	// 2. Gradient section - interactive color gradient editing (if supported)
	lightID := ""
	if light.Id != nil {
		lightID = *light.Id
	}
	rows = append(rows, m.buildGradientRows(light, lightID)...)

	// 3. Settings section - persistent configuration (name, power-on behavior)
	rows = append(rows, m.buildLightSettingsRows(light, device)...)

	// 4. State Info section - current readings
	rows = append(rows, m.buildStateRows(light)...)

	// 5. Product Info section
	rows = append(rows, m.buildProductInfoRows(device)...)

	// 6. Classification section
	rows = append(rows, m.buildClassificationRows(light, device)...)

	// 7. Dynamics section
	rows = append(rows, m.buildDynamicsRows(light)...)

	// 8. Capabilities section
	rows = append(rows, m.buildCapabilitiesRows(light)...)

	// 9. Effects list section
	rows = append(rows, m.buildEffectsListRows(light)...)

	// 10. Signaling section
	rows = append(rows, m.buildSignalingRows(light)...)

	// 11. Device Services section
	rows = append(rows, m.buildDeviceServicesRows(light, device)...)

	// 12. IDs section - least urgent, at the end
	rows = append(rows, m.buildIDsRows(light)...)

	m.lightGrid.SetRows(rows)
}

// handleNewFieldChange handles field change messages from the new form component.
// This is the message-based equivalent of handleLightFieldChange.
func (m *Model) handleNewFieldChange(msg field.FieldChangedMsg) {
	m.status = fmt.Sprintf("Field changed: %s", msg.FieldID)

	// Get the currently selected node
	node := m.tree.SelectedNode()
	if node == nil || node.Item == nil {
		m.status = "Field change ignored: no node selected"
		return
	}

	bridgeID := node.Item.BridgeID
	bridge := m.manager.GetBridge(bridgeID)
	if bridge == nil {
		m.status = fmt.Sprintf("Bridge not found: %s", bridgeID)
		return
	}

	// For light-specific fields, we need a light ID
	lightID := m.selectedLightID

	var err error

	// Handle fields with embedded IDs (name:deviceID, archetype:deviceID, powerup-preset:lightID, room-archetype:roomID, zone-archetype:zoneID)
	if strings.HasPrefix(msg.FieldID, "name:") {
		deviceID := strings.TrimPrefix(msg.FieldID, "name:")
		if v, ok := msg.Value.(field.TextValue); ok {
			err = bridge.SetDeviceName(deviceID, v.Text)
		}
	} else if strings.HasPrefix(msg.FieldID, "archetype:") {
		deviceID := strings.TrimPrefix(msg.FieldID, "archetype:")
		if v, ok := msg.Value.(field.SelectValue); ok {
			archetypeKeys := getSortedProductArchetypes()
			if v.Index >= 0 && v.Index < len(archetypeKeys) {
				archetype := hueclient.ProductArchetype(archetypeKeys[v.Index])
				err = bridge.SetDeviceArchetype(deviceID, archetype)
			}
		}
	} else if strings.HasPrefix(msg.FieldID, "powerup-preset:") {
		lightIDFromField := strings.TrimPrefix(msg.FieldID, "powerup-preset:")
		if v, ok := msg.Value.(field.SelectValue); ok {
			presetKeys := getSortedPowerupPresets()
			if v.Index >= 0 && v.Index < len(presetKeys) {
				preset := hueclient.PowerupPreset(presetKeys[v.Index])
				err = bridge.SetLightPowerupPreset(lightIDFromField, preset)
			}
		}
	} else if strings.HasPrefix(msg.FieldID, "room-archetype:") {
		roomID := strings.TrimPrefix(msg.FieldID, "room-archetype:")
		if v, ok := msg.Value.(field.SelectValue); ok {
			archetypeKeys := hue.RoomArchetypeList()
			if v.Index >= 0 && v.Index < len(archetypeKeys) {
				archetype := hueclient.RoomArchetype(archetypeKeys[v.Index])
				m.status = fmt.Sprintf("Setting room archetype to %s", hue.RoomArchetypeDisplayNames[archetypeKeys[v.Index]])
				err = bridge.SetRoomArchetype(roomID, archetype)
				// Don't call updateDetailContent - SSE event will refresh UI
				// This prevents the selector from reverting to old value
				if err != nil {
					m.status = fmt.Sprintf("Error: %v", err)
				}
				return
			}
		}
	} else if strings.HasPrefix(msg.FieldID, "zone-archetype:") {
		zoneID := strings.TrimPrefix(msg.FieldID, "zone-archetype:")
		if v, ok := msg.Value.(field.SelectValue); ok {
			archetypeKeys := hue.RoomArchetypeList()
			if v.Index >= 0 && v.Index < len(archetypeKeys) {
				archetype := hueclient.RoomArchetype(archetypeKeys[v.Index])
				m.status = fmt.Sprintf("Setting zone archetype to %s", hue.RoomArchetypeDisplayNames[archetypeKeys[v.Index]])
				err = bridge.SetZoneArchetype(zoneID, archetype)
				// Don't call updateDetailContent - SSE event will refresh UI
				if err != nil {
					m.status = fmt.Sprintf("Error: %v", err)
				}
				return
			}
		}
	} else if strings.HasPrefix(msg.FieldID, "room-name:") {
		roomID := strings.TrimPrefix(msg.FieldID, "room-name:")
		if v, ok := msg.Value.(field.TextValue); ok {
			m.status = fmt.Sprintf("Renaming room to \"%s\"", v.Text)
			err = bridge.RenameRoom(roomID, v.Text)
			// Don't call updateDetailContent - SSE event will refresh UI
			if err != nil {
				m.status = fmt.Sprintf("Error: %v", err)
			}
			return
		}
	} else if strings.HasPrefix(msg.FieldID, "zone-name:") {
		zoneID := strings.TrimPrefix(msg.FieldID, "zone-name:")
		if v, ok := msg.Value.(field.TextValue); ok {
			m.status = fmt.Sprintf("Renaming zone to \"%s\"", v.Text)
			err = bridge.RenameZone(zoneID, v.Text)
			// Don't call updateDetailContent - SSE event will refresh UI
			if err != nil {
				m.status = fmt.Sprintf("Error: %v", err)
			}
			return
		}
	} else if strings.HasPrefix(msg.FieldID, "device-room:") || strings.HasPrefix(msg.FieldID, "light-room:") {
		// Handle both device and light room assignment (both use device ID)
		deviceID := strings.TrimPrefix(msg.FieldID, "device-room:")
		if strings.HasPrefix(msg.FieldID, "light-room:") {
			deviceID = strings.TrimPrefix(msg.FieldID, "light-room:")
		}
		if v, ok := msg.Value.(field.SelectValue); ok {
			bridgeState := bridge.GetState()
			if bridgeState != nil {
				allRooms := bridgeState.AllRooms()
				var newRoomID string
				var newRoomName string

				if v.Index == 0 {
					// "No Room" selected - remove from current room
					newRoomID = ""
					newRoomName = "No Room"
				} else if v.Index > 0 && v.Index <= len(allRooms) {
					// Room selected (index 1 = first room in AllRooms)
					room := allRooms[v.Index-1]
					if room.Id != nil {
						newRoomID = *room.Id
					}
					if room.Metadata != nil && room.Metadata.Name != nil {
						newRoomName = *room.Metadata.Name
					}
				}

				m.status = fmt.Sprintf("Moving device to %s...", newRoomName)
				err = bridge.MoveDeviceToRoom(deviceID, newRoomID)
				if err == nil {
					m.status = fmt.Sprintf("Device moved to %s", newRoomName)
					m.rebuildTreeForActiveTab()
				}
				if err != nil {
					m.status = fmt.Sprintf("Error: %v", err)
				}
				return
			}
		}
	} else if strings.HasPrefix(msg.FieldID, "light-zones:") {
		// Handle zone membership checkbox - format is "light-zones:{lightID}"
		lightID := strings.TrimPrefix(msg.FieldID, "light-zones:")
		if v, ok := msg.Value.(field.CheckboxValue); ok {
			bridgeState := bridge.GetState()
			if bridgeState != nil {
				allZones := bridgeState.AllZones()
				currentZones := bridgeState.GetLightZones(lightID)

				// Build current zone membership set
				currentZoneIDs := make(map[string]bool)
				for _, zone := range currentZones {
					if zone.Id != nil {
						currentZoneIDs[*zone.Id] = true
					}
				}

				// Process changes
				var added, removed int
				for i, zone := range allZones {
					if zone.Id == nil {
						continue
					}
					zoneID := *zone.Id
					wasInZone := currentZoneIDs[zoneID]
					nowInZone := v.Selected[i]

					if nowInZone && !wasInZone {
						// Add to zone
						if err = bridge.AddLightToZone(lightID, zoneID); err != nil {
							m.status = fmt.Sprintf("Error: %v", err)
							return
						}
						added++
					} else if !nowInZone && wasInZone {
						// Remove from zone
						if err = bridge.RemoveLightFromZone(lightID, zoneID); err != nil {
							m.status = fmt.Sprintf("Error: %v", err)
							return
						}
						removed++
					}
				}

				if added > 0 || removed > 0 {
					m.status = fmt.Sprintf("Zone membership updated (+%d/-%d)", added, removed)
				}
			}
			return
		}
	} else if strings.HasPrefix(msg.FieldID, "room-create-scene:") {
		roomID := strings.TrimPrefix(msg.FieldID, "room-create-scene:")
		if _, ok := msg.Value.(field.ButtonValue); ok {
			// Get room name for the scene name
			sceneName := "New Scene"
			bridgeState := bridge.GetState()
			if bridgeState != nil {
				if room, ok := bridgeState.GetRoom(roomID); ok {
					roomName := bridgeState.GetRoomName(room)
					sceneName = fmt.Sprintf("%s Scene", roomName)
				}
			}
			m.status = fmt.Sprintf("Creating scene \"%s\"...", sceneName)
			err = bridge.CreateSceneFromCurrentState(roomID, false, sceneName)
			if err == nil {
				m.status = fmt.Sprintf("Scene \"%s\" created", sceneName)
				m.rebuildTreeForActiveTab()
			}
			if err != nil {
				m.status = fmt.Sprintf("Error: %v", err)
			}
			return
		}
	} else if strings.HasPrefix(msg.FieldID, "zone-create-scene:") {
		zoneID := strings.TrimPrefix(msg.FieldID, "zone-create-scene:")
		if _, ok := msg.Value.(field.ButtonValue); ok {
			// Get zone name for the scene name
			sceneName := "New Scene"
			bridgeState := bridge.GetState()
			if bridgeState != nil {
				if zone, ok := bridgeState.GetZone(zoneID); ok {
					zoneName := zone.RoomName("Zone")
					sceneName = fmt.Sprintf("%s Scene", zoneName)
				}
			}
			m.status = fmt.Sprintf("Creating scene \"%s\"...", sceneName)
			err = bridge.CreateSceneFromCurrentState(zoneID, true, sceneName)
			if err == nil {
				m.status = fmt.Sprintf("Scene \"%s\" created", sceneName)
				m.rebuildTreeForActiveTab()
			}
			if err != nil {
				m.status = fmt.Sprintf("Error: %v", err)
			}
			return
		}
	} else if strings.HasPrefix(msg.FieldID, "scene-name:") {
		sceneID := strings.TrimPrefix(msg.FieldID, "scene-name:")
		if v, ok := msg.Value.(field.TextValue); ok {
			m.status = fmt.Sprintf("Renaming scene to \"%s\"", v.Text)
			err = bridge.RenameScene(sceneID, v.Text)
			// Don't call updateDetailContent - SSE event will refresh UI
			if err != nil {
				m.status = fmt.Sprintf("Error: %v", err)
			}
			return
		}
	} else if strings.HasPrefix(msg.FieldID, "bridge-name:") {
		deviceID := strings.TrimPrefix(msg.FieldID, "bridge-name:")
		if v, ok := msg.Value.(field.TextValue); ok {
			m.status = fmt.Sprintf("Renaming bridge to \"%s\"", v.Text)
			err = bridge.RenameDevice(deviceID, v.Text)
			// Don't call updateDetailContent - SSE event will refresh UI
			if err != nil {
				m.status = fmt.Sprintf("Error: %v", err)
			}
			return
		}
	} else if strings.HasPrefix(msg.FieldID, "identify:") {
		// General identify handler for lights and devices
		deviceID := strings.TrimPrefix(msg.FieldID, "identify:")
		if _, ok := msg.Value.(field.ButtonValue); ok {
			m.status = "Identifying device..."
			err = bridge.IdentifyDevice(deviceID)
			if err != nil {
				m.status = fmt.Sprintf("Error: %v", err)
			} else {
				m.status = "Device identification triggered"
			}
			return
		}
	} else if strings.HasPrefix(msg.FieldID, "bridge-identify:") {
		// Bridge-specific identify handler
		deviceID := strings.TrimPrefix(msg.FieldID, "bridge-identify:")
		if _, ok := msg.Value.(field.ButtonValue); ok {
			m.status = "Identifying bridge..."
			err = bridge.IdentifyDevice(deviceID)
			if err != nil {
				m.status = fmt.Sprintf("Error: %v", err)
			} else {
				m.status = "Bridge identification triggered"
			}
			return
		}
	} else if strings.HasPrefix(msg.FieldID, "device-name:") {
		deviceID := strings.TrimPrefix(msg.FieldID, "device-name:")
		if v, ok := msg.Value.(field.TextValue); ok {
			m.status = fmt.Sprintf("Renaming device to \"%s\"", v.Text)
			err = bridge.RenameDevice(deviceID, v.Text)
			// Don't call updateDetailContent - SSE event will refresh UI
			if err != nil {
				m.status = fmt.Sprintf("Error: %v", err)
			}
			return
		}
	} else if strings.HasPrefix(msg.FieldID, "scene-recall:") {
		sceneID := strings.TrimPrefix(msg.FieldID, "scene-recall:")
		if _, ok := msg.Value.(field.ButtonValue); ok {
			m.status = "Activating scene..."
			err = bridge.RecallScene(sceneID)
			if err == nil {
				m.status = "Scene activated"
			}
		}
	} else if strings.HasPrefix(msg.FieldID, "smartscene-toggle:") {
		sceneID := strings.TrimPrefix(msg.FieldID, "smartscene-toggle:")
		if v, ok := msg.Value.(field.ToggleValue); ok {
			if v.On {
				m.status = "Activating smart scene..."
				err = bridge.ActivateSmartScene(sceneID)
				if err == nil {
					m.status = "Smart scene activated"
				}
			} else {
				m.status = "Deactivating smart scene..."
				err = bridge.DeactivateSmartScene(sceneID)
				if err == nil {
					m.status = "Smart scene deactivated"
				}
			}
			if err != nil {
				m.status = fmt.Sprintf("Error: %v", err)
			}
			return
		}
	} else if strings.HasPrefix(msg.FieldID, "motion-enabled:") {
		motionID := strings.TrimPrefix(msg.FieldID, "motion-enabled:")
		if v, ok := msg.Value.(field.ToggleValue); ok {
			action := "disabled"
			if v.On {
				action = "enabled"
			}
			m.status = fmt.Sprintf("Motion sensor %s", action)
			err = bridge.SetMotionSensorEnabled(motionID, v.On)
			if err != nil {
				m.status = fmt.Sprintf("Error: %v", err)
			}
			return
		}
	} else if strings.HasPrefix(msg.FieldID, "motion-sensitivity:") {
		motionID := strings.TrimPrefix(msg.FieldID, "motion-sensitivity:")
		if v, ok := msg.Value.(field.SliderValue); ok {
			m.status = fmt.Sprintf("Motion sensitivity: %d", v.Value)
			err = bridge.SetMotionSensorSensitivity(motionID, v.Value)
			if err != nil {
				m.status = fmt.Sprintf("Error: %v", err)
			}
			return
		}
	} else if strings.HasPrefix(msg.FieldID, "temp-enabled:") {
		tempID := strings.TrimPrefix(msg.FieldID, "temp-enabled:")
		if v, ok := msg.Value.(field.ToggleValue); ok {
			action := "disabled"
			if v.On {
				action = "enabled"
			}
			m.status = fmt.Sprintf("Temperature sensor %s", action)
			err = bridge.SetTemperatureSensorEnabled(tempID, v.On)
			if err != nil {
				m.status = fmt.Sprintf("Error: %v", err)
			}
			return
		}
	} else if strings.HasPrefix(msg.FieldID, "ll-enabled:") {
		llID := strings.TrimPrefix(msg.FieldID, "ll-enabled:")
		if v, ok := msg.Value.(field.ToggleValue); ok {
			action := "disabled"
			if v.On {
				action = "enabled"
			}
			m.status = fmt.Sprintf("Light level sensor %s", action)
			err = bridge.SetLightLevelSensorEnabled(llID, v.On)
			if err != nil {
				m.status = fmt.Sprintf("Error: %v", err)
			}
			return
		}
	} else if strings.HasPrefix(msg.FieldID, "room-power:") || strings.HasPrefix(msg.FieldID, "zone-power:") {
		// Room or zone grouped light power control
		var glID string
		if strings.HasPrefix(msg.FieldID, "room-power:") {
			glID = strings.TrimPrefix(msg.FieldID, "room-power:")
		} else {
			glID = strings.TrimPrefix(msg.FieldID, "zone-power:")
		}
		if v, ok := msg.Value.(field.ToggleValue); ok {
			err = bridge.SetGroupedLightOn(glID, v.On)
		}
	} else if strings.HasPrefix(msg.FieldID, "room-brightness:") || strings.HasPrefix(msg.FieldID, "zone-brightness:") {
		// Room or zone grouped light brightness control
		var glID string
		if strings.HasPrefix(msg.FieldID, "room-brightness:") {
			glID = strings.TrimPrefix(msg.FieldID, "room-brightness:")
		} else {
			glID = strings.TrimPrefix(msg.FieldID, "zone-brightness:")
		}
		if v, ok := msg.Value.(field.SliderValue); ok {
			err = bridge.SetGroupedLightBrightness(glID, float64(v.Value))
		}
	} else if strings.HasPrefix(msg.FieldID, "gradient-mode:") {
		// Gradient mode selection
		gradientLightID := strings.TrimPrefix(msg.FieldID, "gradient-mode:")
		if v, ok := msg.Value.(field.SelectValue); ok {
			state := bridge.GetState()
			if state != nil {
				if light, ok := state.GetLight(gradientLightID); ok {
					if light.Gradient != nil && light.Gradient.ModeValues != nil {
						modes := *light.Gradient.ModeValues
						if v.Index >= 0 && v.Index < len(modes) {
							m.status = fmt.Sprintf("Gradient mode: %s", formatGradientMode(modes[v.Index]))
							err = bridge.SetLightGradientMode(gradientLightID, modes[v.Index])
						}
					}
				}
			}
		}
	} else if strings.HasPrefix(msg.FieldID, "gradient-points:") {
		// Gradient points changed
		gradientLightID := strings.TrimPrefix(msg.FieldID, "gradient-points:")
		if v, ok := msg.Value.(field.GradientValue); ok {
			points := convertToAPIPoints(v.Points)
			m.status = fmt.Sprintf("Gradient points: %d", len(points))
			err = bridge.SetLightGradientPoints(gradientLightID, points)
		}
	} else {
		// Handle regular fields
		switch msg.FieldID {
		case "on":
			if v, ok := msg.Value.(field.ToggleValue); ok {
				err = bridge.SetLightOn(lightID, v.On)
			}
		case "brightness":
			if v, ok := msg.Value.(field.SliderValue); ok {
				err = bridge.SetLightBrightness(lightID, float64(v.Value))
			}
		case "colortemp":
			if v, ok := msg.Value.(field.SliderValue); ok {
				err = bridge.SetLightColorTemperature(lightID, v.Value)
			}
		case "color":
			if v, ok := msg.Value.(field.ColorValue); ok {
				err = bridge.SetLightColor(lightID, v.X, v.Y)
			}
		case "effect":
			if v, ok := msg.Value.(field.SelectValue); ok {
				// Get the effect from the light's effect values
				if state := bridge.GetState(); state != nil {
					if light, ok := state.GetLight(lightID); ok {
						if light.Effects != nil && light.Effects.EffectValues != nil {
							effects := *light.Effects.EffectValues
							if v.Index >= 0 && v.Index < len(effects) {
								effect := effects[v.Index]
								err = bridge.SetLightEffect(lightID, effect)
							}
						}
					}
				}
			}
		}
	}

	if err != nil {
		m.status = fmt.Sprintf("Error: %v", err)
	} else {
		// Update detail content immediately to reflect changes (optimistic update)
		m.updateDetailContent()
		// Rebuild tree to update indicators
		m.rebuildTreeForActiveTab()
	}
}

// getDeviceForLight looks up the device that owns a light.
func (m *Model) getDeviceForLight(light hueclient.LightGet) *hueclient.DeviceGet {
	if light.Owner == nil || light.Owner.Rid == nil {
		return nil
	}
	node := m.tree.SelectedNode()
	if node == nil || node.Item == nil {
		return nil
	}
	bridge := m.manager.GetBridge(node.Item.BridgeID)
	if bridge == nil {
		return nil
	}
	state := bridge.GetState()
	if state == nil {
		return nil
	}
	if device, ok := state.GetDevice(*light.Owner.Rid); ok {
		return &device
	}
	return nil
}

// renderLightIndicator renders a brightness indicator for a light.
func (m *Model) renderLightIndicator(light hueclient.LightGet, isOn bool) string {
	if !isOn {
		// Return dimmed off indicator
		return m.styles.Dimmed.Render(ui.BrightnessIndicator(0))
	}

	brightness := 100.0
	if light.Dimming != nil && light.Dimming.Brightness != nil {
		brightness = float64(*light.Dimming.Brightness)
	}

	hexColor := ui.GetLightColor(light)
	return ui.RenderBrightnessIndicatorFromHex(brightness, hexColor)
}

// infoLabelWidth is the width for labels in info rows.
const infoLabelWidth = 14

// buildProductInfoRows builds rows for the product info section.
func (m *Model) buildProductInfoRows(device *hueclient.DeviceGet) []gridlayout.GridRow {
	if device == nil || device.ProductData == nil {
		return nil
	}

	var rows []gridlayout.GridRow
	pd := device.ProductData

	// Section header
	header := field.NewHeaderComponent("product-header", "Product", &m.styles, m.zones)
	rows = append(rows, gridlayout.GridRow{
		Type:    gridlayout.RowTypeSection,
		Section: header,
	})

	if pd.ProductName != nil {
		rows = append(rows, gridlayout.NewInfoRow("Product", *pd.ProductName, infoLabelWidth))
	}
	if pd.ManufacturerName != nil {
		rows = append(rows, gridlayout.NewInfoRow("Manufacturer", *pd.ManufacturerName, infoLabelWidth))
	}
	if pd.ModelId != nil {
		rows = append(rows, gridlayout.NewInfoRow("Model", *pd.ModelId, infoLabelWidth))
	}
	if pd.SoftwareVersion != nil {
		rows = append(rows, gridlayout.NewInfoRow("Firmware", *pd.SoftwareVersion, infoLabelWidth))
	}
	if pd.HardwarePlatformType != nil {
		rows = append(rows, gridlayout.NewInfoRow("Hardware", *pd.HardwarePlatformType, infoLabelWidth))
	}

	return rows
}

// buildClassificationRows builds rows for the classification section.
func (m *Model) buildClassificationRows(light hueclient.LightGet, device *hueclient.DeviceGet) []gridlayout.GridRow {
	var rows []gridlayout.GridRow

	// Section header
	header := field.NewHeaderComponent("class-header", "Classification", &m.styles, m.zones)
	rows = append(rows, gridlayout.GridRow{
		Type:    gridlayout.RowTypeSection,
		Section: header,
	})

	// Type (read-only)
	if light.Type != nil {
		rows = append(rows, gridlayout.NewInfoRow("Type", string(*light.Type), infoLabelWidth))
	}

	// Mode (read-only)
	if light.Mode != nil {
		rows = append(rows, gridlayout.NewInfoRow("Mode", string(*light.Mode), infoLabelWidth))
	}

	// Alternate name (read-only) - shows light's own name if different from device name
	if light.Metadata != nil && light.Metadata.Name != nil {
		altName := *light.Metadata.Name
		currentName := ""
		if device != nil && device.Metadata != nil && device.Metadata.Name != nil {
			currentName = *device.Metadata.Name
		}
		if altName != currentName {
			rows = append(rows, gridlayout.NewInfoRow("Alternate name", altName, infoLabelWidth))
		}
	}

	return rows
}

// buildLightSettingsRows builds rows for the light settings section.
// Only includes editable fields: name, archetype, power-on behavior.
func (m *Model) buildLightSettingsRows(light hueclient.LightGet, device *hueclient.DeviceGet) []gridlayout.GridRow {
	var rows []gridlayout.GridRow

	// Section header
	header := field.NewHeaderComponent("settings-header", "Settings", &m.styles, m.zones)
	rows = append(rows, gridlayout.GridRow{
		Type:    gridlayout.RowTypeSection,
		Section: header,
	})

	// Device name (editable)
	deviceID := ""
	if device != nil && device.Id != nil {
		deviceID = *device.Id
	}

	if device != nil && device.Metadata != nil && device.Metadata.Name != nil {
		textInput := field.NewTextComponent(
			"name:"+deviceID, "Name", *device.Metadata.Name,
			&m.styles, m.zones,
		)
		rows = append(rows, gridlayout.GridRow{
			Type: gridlayout.RowTypeNormal,
			Cells: []gridlayout.GridCell{
				{Component: gridlayout.NewLabelWithWidth("Name", infoLabelWidth)},
				{Component: textInput},
			},
		})
	}

	// Archetype (editable)
	if device != nil && device.ProductData != nil && device.ProductData.ProductArchetype != nil {
		deviceID := ""
		if device.Id != nil {
			deviceID = *device.Id
		}

		archetypeKeys := getSortedProductArchetypes()
		var options []field.Option
		currentIndex := 0
		currentArchetype := string(*device.ProductData.ProductArchetype)

		for i, key := range archetypeKeys {
			displayName := hue.ProductArchetypeDisplayNames[key]
			options = append(options, field.Option{Label: displayName, Value: i})
			if key == currentArchetype {
				currentIndex = i
			}
		}

		selectComp := field.NewSelectComponent(
			"archetype:"+deviceID, "Archetype", currentIndex, options,
			&m.styles, m.zones,
		)
		rows = append(rows, gridlayout.GridRow{
			Type: gridlayout.RowTypeNormal,
			Cells: []gridlayout.GridCell{
				{Component: gridlayout.NewLabelWithWidth("Archetype", infoLabelWidth)},
				{Component: selectComp},
			},
		})
	}

	// Powerup preset (editable)
	if light.Powerup != nil && light.Powerup.Preset != nil {
		presetKeys := getSortedPowerupPresets()
		var options []field.Option
		currentIndex := 0
		currentPreset := string(*light.Powerup.Preset)

		for i, key := range presetKeys {
			displayName := hue.PowerupPresetDisplayNames[key]
			options = append(options, field.Option{Label: displayName, Value: i})
			if key == currentPreset {
				currentIndex = i
			}
		}

		lightID := ""
		if light.Id != nil {
			lightID = *light.Id
		}

		selectComp := field.NewSelectComponent(
			"powerup-preset:"+lightID, "Power-on", currentIndex, options,
			&m.styles, m.zones,
		)
		rows = append(rows, gridlayout.GridRow{
			Type: gridlayout.RowTypeNormal,
			Cells: []gridlayout.GridCell{
				{Component: gridlayout.NewLabelWithWidth("Power-on", infoLabelWidth)},
				{Component: selectComp},
			},
		})
	}

	// Room assignment dropdown (lights move with their device)
	if deviceID != "" {
		// Get state from the selected bridge
		node := m.tree.SelectedNode()
		if node != nil && node.Item != nil {
			bridge := m.manager.GetBridge(node.Item.BridgeID)
			if bridge != nil {
				state := bridge.GetState()
				if state != nil {
					currentRoom, hasRoom := state.GetDeviceRoom(deviceID)
					currentRoomID := ""
					if hasRoom && currentRoom.Id != nil {
						currentRoomID = *currentRoom.Id
					}

					// Build room options: "No Room" + all rooms
					allRooms := state.AllRooms()
					options := make([]field.Option, 0, len(allRooms)+1)
					options = append(options, field.Option{Label: "No Room", Value: 0})
					selectedIndex := 0

					for i, room := range allRooms {
						roomName := "Unknown"
						roomID := ""
						if room.Metadata != nil && room.Metadata.Name != nil {
							roomName = *room.Metadata.Name
						}
						if room.Id != nil {
							roomID = *room.Id
						}
						options = append(options, field.Option{Label: roomName, Value: i + 1, Color: m.styles.Theme.EntityRoom})
						if roomID == currentRoomID {
							selectedIndex = i + 1
						}
					}

					roomSelect := field.NewSelectComponent(
						"light-room:"+deviceID, "Room", selectedIndex, options,
						&m.styles, m.zones,
					)
					rows = append(rows, gridlayout.GridRow{
						Type: gridlayout.RowTypeNormal,
						Cells: []gridlayout.GridCell{
							{Component: gridlayout.NewLabelWithWidth("Room", infoLabelWidth)},
							{Component: roomSelect},
						},
					})
				}
			}
		}
	}

	// Zone membership (checkboxes)
	if light.Id != nil {
		zoneNode := m.tree.SelectedNode()
		if zoneNode != nil && zoneNode.Item != nil {
			zoneBridge := m.manager.GetBridge(zoneNode.Item.BridgeID)
			if zoneBridge != nil {
				zoneState := zoneBridge.GetState()
				if zoneState != nil {
					lightID := *light.Id
					allZones := zoneState.AllZones()
					if len(allZones) > 0 {
						// Build a set of zone IDs this light is in
						lightZones := zoneState.GetLightZones(lightID)
						lightZoneIDs := make(map[string]bool)
						for _, zone := range lightZones {
							if zone.Id != nil {
								lightZoneIDs[*zone.Id] = true
							}
						}

						// Build options and selected map for checkbox component
						var options []field.Option
						selected := make(map[int]bool)
						for i, z := range allZones {
							zoneID := ""
							zoneName := "Unknown"
							if z.Id != nil {
								zoneID = *z.Id
							}
							if z.Metadata != nil && z.Metadata.Name != nil {
								zoneName = *z.Metadata.Name
							}
							options = append(options, field.Option{Label: zoneName, Value: i, Color: m.styles.Theme.EntityZone})
							if lightZoneIDs[zoneID] {
								selected[i] = true
							}
						}

						// Add checkbox group for zones
						zoneCheckbox := field.NewCheckboxComponent(
							"light-zones:"+lightID, "Zones", selected, options, true,
							&m.styles, m.zones,
						)
						rows = append(rows, gridlayout.GridRow{
							Type: gridlayout.RowTypeNormal,
							Cells: []gridlayout.GridCell{
								{Component: gridlayout.NewLabelWithWidth("Zones", infoLabelWidth)},
								{Component: zoneCheckbox},
							},
						})
					}
				}
			}
		}
	}

	return rows
}

// buildIDsRows builds rows for the IDs section.
func (m *Model) buildIDsRows(light hueclient.LightGet) []gridlayout.GridRow {
	var rows []gridlayout.GridRow

	// Section header
	header := field.NewHeaderComponent("ids-header", "IDs", &m.styles, m.zones)
	rows = append(rows, gridlayout.GridRow{
		Type:    gridlayout.RowTypeSection,
		Section: header,
	})

	dimStyle := m.styles.Dimmed

	if light.Id != nil {
		rows = append(rows, gridlayout.NewStyledInfoRow("Light ID", *light.Id, infoLabelWidth, dimStyle))
	}
	if light.Owner != nil && light.Owner.Rid != nil {
		rows = append(rows, gridlayout.NewStyledInfoRow("Device ID", *light.Owner.Rid, infoLabelWidth, dimStyle))
	}
	if light.IdV1 != nil {
		rows = append(rows, gridlayout.NewStyledInfoRow("V1 ID", *light.IdV1, infoLabelWidth, dimStyle))
	}

	return rows
}

// buildControlsRows builds rows for the controls section (existing functionality).
func (m *Model) buildControlsRows(light hueclient.LightGet) []gridlayout.GridRow {
	var rows []gridlayout.GridRow

	// Section header
	header := field.NewHeaderComponent("controls-header", "Controls", &m.styles, m.zones)
	rows = append(rows, gridlayout.GridRow{
		Type:    gridlayout.RowTypeSection,
		Section: header,
	})

	// On/Off toggle
	onValue := false
	if light.On != nil && light.On.On != nil && *light.On.On {
		onValue = true
	}
	toggle := field.NewToggleComponent("on", "Power", onValue, &m.styles, m.zones)
	toggle.SetLabels("On", "Off")
	rows = append(rows, gridlayout.GridRow{
		Type: gridlayout.RowTypeNormal,
		Cells: []gridlayout.GridCell{
			{Component: gridlayout.NewLabelWithWidth("Power", infoLabelWidth)},
			{Component: toggle},
		},
	})

	// Brightness (if dimmable)
	if light.Dimming != nil && light.Dimming.Brightness != nil {
		brightness := int(*light.Dimming.Brightness)
		slider := field.NewBrightnessSliderComponent(
			"brightness", "Brightness", brightness,
			&m.styles, m.zones,
		)
		rows = append(rows, gridlayout.GridRow{
			Type: gridlayout.RowTypeNormal,
			Cells: []gridlayout.GridCell{
				{Component: gridlayout.NewLabelWithWidth("Brightness", infoLabelWidth)},
				{Component: slider},
			},
		})
	}

	// Color temperature (if supported)
	if light.ColorTemperature != nil && light.ColorTemperature.MirekSchema != nil {
		mirek := 250
		if light.ColorTemperature.Mirek != nil {
			mirek = *light.ColorTemperature.Mirek
		}
		minMirek := 153
		maxMirek := 500
		if light.ColorTemperature.MirekSchema.MirekMinimum != nil {
			minMirek = *light.ColorTemperature.MirekSchema.MirekMinimum
		}
		if light.ColorTemperature.MirekSchema.MirekMaximum != nil {
			maxMirek = *light.ColorTemperature.MirekSchema.MirekMaximum
		}
		slider := field.NewColorTempSliderComponent(
			"colortemp", "Color Temp", mirek, minMirek, maxMirek,
			&m.styles, m.zones,
		)
		rows = append(rows, gridlayout.GridRow{
			Type: gridlayout.RowTypeNormal,
			Cells: []gridlayout.GridCell{
				{Component: gridlayout.NewLabelWithWidth("Color Temp", infoLabelWidth)},
				{Component: slider},
			},
		})
	}

	// Color (if supported)
	if light.Color != nil && light.Color.Xy != nil {
		x, y := 0.3127, 0.329
		if light.Color.Xy.X != nil {
			x = float64(*light.Color.Xy.X)
		}
		if light.Color.Xy.Y != nil {
			y = float64(*light.Color.Xy.Y)
		}
		colorWheel := field.NewColorWheelComponent(
			"color", "Color", x, y,
			&m.styles, m.zones,
		)
		rows = append(rows, gridlayout.GridRow{
			Type: gridlayout.RowTypeNormal,
			Cells: []gridlayout.GridCell{
				{Component: gridlayout.NewLabelWithWidth("Color", infoLabelWidth)},
				{Component: colorWheel},
			},
		})
	}

	// Effect (if supported)
	if light.Effects != nil && light.Effects.EffectValues != nil {
		var options []field.Option
		currentEffect := -1
		if light.Effects.Status != nil {
			for i, effect := range *light.Effects.EffectValues {
				effectStr := string(effect)
				displayName := hue.EffectDisplayName(effectStr)
				options = append(options, field.Option{
					Label: displayName,
					Value: i,
				})
				if effect == *light.Effects.Status {
					currentEffect = i
				}
			}
		}
		if len(options) > 0 {
			if currentEffect == -1 {
				currentEffect = 0
			}
			selectComp := field.NewSelectComponent(
				"effect", "Effect", currentEffect, options,
				&m.styles, m.zones,
			)
			rows = append(rows, gridlayout.GridRow{
				Type: gridlayout.RowTypeNormal,
				Cells: []gridlayout.GridCell{
					{Component: gridlayout.NewLabelWithWidth("Effect", infoLabelWidth)},
					{Component: selectComp},
				},
			})
		}
	}

	// Identify button (uses device ID from light owner)
	if light.Owner != nil && light.Owner.Rid != nil {
		identifyBtn := field.NewButtonComponent(
			"identify:"+*light.Owner.Rid, "Identify", "Identify",
			&m.styles, m.zones,
		)
		rows = append(rows, gridlayout.GridRow{
			Type: gridlayout.RowTypeNormal,
			Cells: []gridlayout.GridCell{
				{Component: gridlayout.NewLabelWithWidth("Identify", infoLabelWidth)},
				{Component: identifyBtn},
			},
		})
	}

	return rows
}

// buildStateRows builds rows for technical state info not shown in controls.
// Only includes details that add value beyond what the controls already display.
func (m *Model) buildStateRows(light hueclient.LightGet) []gridlayout.GridRow {
	var rows []gridlayout.GridRow

	// Collect items that provide info beyond the controls
	var hasContent bool

	// Min dim level (not shown in brightness slider)
	if light.Dimming != nil && light.Dimming.MinDimLevel != nil {
		hasContent = true
	}

	// Gamut type (not shown in color wheel)
	if light.Color != nil && light.Color.GamutType != nil {
		hasContent = true
	}

	// CT range (not shown in color temp slider)
	if light.ColorTemperature != nil && light.ColorTemperature.MirekSchema != nil {
		schema := light.ColorTemperature.MirekSchema
		if schema.MirekMinimum != nil && schema.MirekMaximum != nil {
			hasContent = true
		}
	}

	// Only add section if we have content
	if !hasContent {
		return rows
	}

	// Section header
	header := field.NewHeaderComponent("state-header", "Limits", &m.styles, m.zones)
	rows = append(rows, gridlayout.GridRow{
		Type:    gridlayout.RowTypeSection,
		Section: header,
	})

	// Min dim level
	if light.Dimming != nil && light.Dimming.MinDimLevel != nil {
		rows = append(rows, gridlayout.NewInfoRow("Min Dim", fmt.Sprintf("%.0f%%", *light.Dimming.MinDimLevel), infoLabelWidth))
	}

	// Gamut type
	if light.Color != nil && light.Color.GamutType != nil {
		rows = append(rows, gridlayout.NewInfoRow("Gamut", string(*light.Color.GamutType), infoLabelWidth))
	}

	// CT range
	if light.ColorTemperature != nil && light.ColorTemperature.MirekSchema != nil {
		schema := light.ColorTemperature.MirekSchema
		if schema.MirekMinimum != nil && schema.MirekMaximum != nil {
			minK := 1000000 / int(*schema.MirekMaximum)
			maxK := 1000000 / int(*schema.MirekMinimum)
			rows = append(rows, gridlayout.NewInfoRow("CT Range", fmt.Sprintf("%dK - %dK", minK, maxK), infoLabelWidth))
		}
	}

	return rows
}

// buildDynamicsRows builds rows for the dynamics section.
func (m *Model) buildDynamicsRows(light hueclient.LightGet) []gridlayout.GridRow {
	if light.Dynamics == nil {
		return nil
	}

	var rows []gridlayout.GridRow

	// Section header
	header := field.NewHeaderComponent("dynamics-header", "Dynamics", &m.styles, m.zones)
	rows = append(rows, gridlayout.GridRow{
		Type:    gridlayout.RowTypeSection,
		Section: header,
	})

	if light.Dynamics.Status != nil {
		rows = append(rows, gridlayout.NewInfoRow("Status", string(*light.Dynamics.Status), infoLabelWidth))
	}
	if light.Dynamics.Speed != nil {
		rows = append(rows, gridlayout.NewInfoRow("Speed", fmt.Sprintf("%.2f", *light.Dynamics.Speed), infoLabelWidth))
	}

	return rows
}

// buildCapabilitiesRows builds rows for the capabilities section.
func (m *Model) buildCapabilitiesRows(light hueclient.LightGet) []gridlayout.GridRow {
	var rows []gridlayout.GridRow

	// Section header
	header := field.NewHeaderComponent("caps-header", "Capabilities", &m.styles, m.zones)
	rows = append(rows, gridlayout.GridRow{
		Type:    gridlayout.RowTypeSection,
		Section: header,
	})

	// Collect capabilities
	var caps []string
	if light.Dimming != nil {
		caps = append(caps, "Dimming")
	}
	if light.Color != nil {
		caps = append(caps, "Color")
	}
	if light.ColorTemperature != nil {
		caps = append(caps, "Color Temperature")
	}
	if light.Gradient != nil {
		caps = append(caps, "Gradient")
	}
	if light.Effects != nil {
		caps = append(caps, "Effects")
	}
	if light.TimedEffects != nil {
		caps = append(caps, "Timed Effects")
	}

	if len(caps) == 0 {
		caps = append(caps, "On/Off only")
	}

	for _, cap := range caps {
		rows = append(rows, gridlayout.NewListItemRow("•", cap))
	}

	return rows
}

// buildEffectsListRows builds rows for the available effects section.
func (m *Model) buildEffectsListRows(light hueclient.LightGet) []gridlayout.GridRow {
	if light.Effects == nil || light.Effects.EffectValues == nil || len(*light.Effects.EffectValues) == 0 {
		return nil
	}

	var rows []gridlayout.GridRow

	// Section header
	header := field.NewHeaderComponent("effects-header", "Available Effects", &m.styles, m.zones)
	rows = append(rows, gridlayout.GridRow{
		Type:    gridlayout.RowTypeSection,
		Section: header,
	})

	for _, effect := range *light.Effects.EffectValues {
		displayName := hue.EffectDisplayName(string(effect))
		rows = append(rows, gridlayout.NewListItemRow("•", displayName))
	}

	return rows
}

// buildGradientRows builds rows for the gradient section.
func (m *Model) buildGradientRows(light hueclient.LightGet, lightID string) []gridlayout.GridRow {
	if light.Gradient == nil {
		return nil
	}

	var rows []gridlayout.GridRow

	// Section header
	header := field.NewHeaderComponent("gradient-header", "Gradient", &m.styles, m.zones)
	rows = append(rows, gridlayout.GridRow{
		Type:    gridlayout.RowTypeSection,
		Section: header,
	})

	// Mode selector (if ModeValues available)
	if light.Gradient.ModeValues != nil && len(*light.Gradient.ModeValues) > 0 {
		options := make([]field.Option, len(*light.Gradient.ModeValues))
		currentIndex := 0
		for i, mode := range *light.Gradient.ModeValues {
			options[i] = field.Option{Label: formatGradientMode(mode), Value: i}
			if light.Gradient.Mode != nil && *light.Gradient.Mode == mode {
				currentIndex = i
			}
		}
		modeSelect := field.NewSelectComponent(
			"gradient-mode:"+lightID, "Mode", currentIndex, options,
			&m.styles, m.zones,
		)
		rows = append(rows, gridlayout.GridRow{
			Type: gridlayout.RowTypeNormal,
			Cells: []gridlayout.GridCell{
				{Component: gridlayout.NewLabelWithWidth("Mode", infoLabelWidth)},
				{Component: modeSelect},
			},
		})
	} else if light.Gradient.Mode != nil {
		// Read-only mode display if no mode values available
		rows = append(rows, gridlayout.NewInfoRow("Mode", formatGradientMode(*light.Gradient.Mode), infoLabelWidth))
	}

	// Pixel count info
	if light.Gradient.PixelCount != nil {
		rows = append(rows, gridlayout.NewInfoRow("Pixels", fmt.Sprintf("%d", *light.Gradient.PixelCount), infoLabelWidth))
	}

	// Max points capability info
	if light.Gradient.PointsCapable != nil {
		rows = append(rows, gridlayout.NewInfoRow("Max Points", fmt.Sprintf("%d", *light.Gradient.PointsCapable), infoLabelWidth))
	}

	// Gradient editor
	if light.Gradient.Points != nil && len(*light.Gradient.Points) > 0 {
		maxPoints := 5 // default
		if light.Gradient.PointsCapable != nil {
			maxPoints = *light.Gradient.PointsCapable
		}

		points := convertGradientPoints(*light.Gradient.Points)
		editor := field.NewGradientEditorComponent(
			"gradient-points:"+lightID, "Colors", points, maxPoints,
			&m.styles, m.zones,
		)
		rows = append(rows, gridlayout.GridRow{
			Type: gridlayout.RowTypeNormal,
			Cells: []gridlayout.GridCell{
				{Component: gridlayout.NewLabelWithWidth("Colors", infoLabelWidth)},
				{Component: editor},
			},
		})
	}

	return rows
}

// formatGradientMode converts a gradient mode to a display-friendly string.
func formatGradientMode(mode hueclient.SupportedGradientMode) string {
	switch mode {
	case hueclient.InterpolatedPalette:
		return "Interpolated"
	case hueclient.InterpolatedPaletteMirrored:
		return "Mirrored"
	case hueclient.RandomPixelated:
		return "Pixelated"
	default:
		return string(mode)
	}
}

// convertGradientPoints converts hueclient colors to field gradient points.
func convertGradientPoints(colors []hueclient.Color) []field.GradientPoint {
	points := make([]field.GradientPoint, 0, len(colors))
	for _, c := range colors {
		if c.Xy != nil && c.Xy.X != nil && c.Xy.Y != nil {
			points = append(points, field.GradientPoint{
				X: float64(*c.Xy.X),
				Y: float64(*c.Xy.Y),
			})
		}
	}
	return points
}

// convertToAPIPoints converts field gradient points to hueclient colors.
func convertToAPIPoints(points []field.GradientPoint) []hueclient.Color {
	colors := make([]hueclient.Color, len(points))
	for i, pt := range points {
		x, y := float32(pt.X), float32(pt.Y)
		colors[i] = hueclient.Color{
			Xy: &hueclient.GamutPosition{X: &x, Y: &y},
		}
	}
	return colors
}

// buildSignalingRows builds rows for the signaling section.
func (m *Model) buildSignalingRows(light hueclient.LightGet) []gridlayout.GridRow {
	if light.Signaling == nil || light.Signaling.SignalValues == nil || len(*light.Signaling.SignalValues) == 0 {
		return nil
	}

	var rows []gridlayout.GridRow

	// Section header
	header := field.NewHeaderComponent("signaling-header", "Signaling Modes", &m.styles, m.zones)
	rows = append(rows, gridlayout.GridRow{
		Type:    gridlayout.RowTypeSection,
		Section: header,
	})

	for _, sig := range *light.Signaling.SignalValues {
		displayName := hue.SignalingModeDisplayName(string(sig))
		rows = append(rows, gridlayout.NewListItemRow("•", displayName))
	}

	return rows
}

// buildPowerupRows builds rows for the power-on behavior section.
func (m *Model) buildPowerupRows(light hueclient.LightGet) []gridlayout.GridRow {
	if light.Powerup == nil {
		return nil
	}

	var rows []gridlayout.GridRow

	// Section header
	header := field.NewHeaderComponent("powerup-header", "Power-on Behavior", &m.styles, m.zones)
	rows = append(rows, gridlayout.GridRow{
		Type:    gridlayout.RowTypeSection,
		Section: header,
	})

	// Powerup preset (editable)
	if light.Powerup.Preset != nil {
		presetKeys := getSortedPowerupPresets()
		var options []field.Option
		currentIndex := 0
		currentPreset := string(*light.Powerup.Preset)

		for i, key := range presetKeys {
			displayName := hue.PowerupPresetDisplayNames[key]
			options = append(options, field.Option{Label: displayName, Value: i})
			if key == currentPreset {
				currentIndex = i
			}
		}

		lightID := ""
		if light.Id != nil {
			lightID = *light.Id
		}

		selectComp := field.NewSelectComponent(
			"powerup-preset:"+lightID, "Preset", currentIndex, options,
			&m.styles, m.zones,
		)
		rows = append(rows, gridlayout.GridRow{
			Type: gridlayout.RowTypeNormal,
			Cells: []gridlayout.GridCell{
				{Component: gridlayout.NewLabelWithWidth("Preset", infoLabelWidth)},
				{Component: selectComp},
			},
		})
	}

	return rows
}

// buildDeviceServicesRows builds rows for the device services section.
func (m *Model) buildDeviceServicesRows(light hueclient.LightGet, device *hueclient.DeviceGet) []gridlayout.GridRow {
	if device == nil || device.Services == nil || len(*device.Services) == 0 {
		return nil
	}

	var rows []gridlayout.GridRow

	// Section header
	header := field.NewHeaderComponent("services-header", "Device Services", &m.styles, m.zones)
	rows = append(rows, gridlayout.GridRow{
		Type:    gridlayout.RowTypeSection,
		Section: header,
	})

	for _, svc := range *device.Services {
		rtype := "unknown"
		if svc.Rtype != nil {
			rtype = string(*svc.Rtype)
		}
		displayName := hue.DeviceServiceDisplayName(rtype)
		if svc.Rid != nil && light.Id != nil && *svc.Rid == *light.Id {
			displayName = displayName + " " + m.styles.Dimmed.Render("(this)")
		}
		rows = append(rows, gridlayout.NewListItemRow("•", displayName))
	}

	return rows
}

// buildRoomGridRows builds grid rows for a room or zone details panel.
// Sections are ordered by immediacy: Controls > Settings > Status > Lists > IDs
func (m *Model) buildRoomGridRows(room hueclient.RoomGet, isZone bool, state *hue.BridgeState) []gridlayout.GridRow {
	var rows []gridlayout.GridRow

	if state == nil {
		return rows
	}

	roomID := ""
	if room.Id != nil {
		roomID = *room.Id
	}

	var lights []hueclient.LightGet
	if isZone {
		lights = state.ZoneLights(room)
	} else {
		lights = state.RoomLights(room)
	}

	// 1. Controls section - instant adjustments (grouped light power, brightness)
	if gl, ok := state.RoomGroupedLight(room); ok {
		controlsHeader := field.NewHeaderComponent("controls-header", "Controls", &m.styles, m.zones)
		rows = append(rows, gridlayout.GridRow{
			Type:    gridlayout.RowTypeSection,
			Section: controlsHeader,
		})

		// Use different field ID prefix for rooms vs zones
		fieldPrefix := "room-"
		if isZone {
			fieldPrefix = "zone-"
		}

		glID := ""
		if gl.Id != nil {
			glID = *gl.Id
		}

		// Power toggle
		onValue := gl.On != nil && gl.On.On != nil && *gl.On.On
		toggle := field.NewToggleComponent(fieldPrefix+"power:"+glID, "Power", onValue, &m.styles, m.zones)
		toggle.SetLabels("On", "Off")
		rows = append(rows, gridlayout.GridRow{
			Type: gridlayout.RowTypeNormal,
			Cells: []gridlayout.GridCell{
				{Component: gridlayout.NewLabelWithWidth("Power", infoLabelWidth)},
				{Component: toggle},
			},
		})

		// Brightness slider
		if gl.Dimming != nil && gl.Dimming.Brightness != nil {
			brightness := int(*gl.Dimming.Brightness)
			slider := field.NewBrightnessSliderComponent(
				fieldPrefix+"brightness:"+glID, "Brightness", brightness,
				&m.styles, m.zones,
			)
			rows = append(rows, gridlayout.GridRow{
				Type: gridlayout.RowTypeNormal,
				Cells: []gridlayout.GridCell{
					{Component: gridlayout.NewLabelWithWidth("Brightness", infoLabelWidth)},
					{Component: slider},
				},
			})
		}

		// Create Scene button (captures current light states)
		createSceneBtn := field.NewButtonComponent(
			fieldPrefix+"create-scene:"+roomID, "Create Scene", "Create Scene",
			&m.styles, m.zones,
		)
		rows = append(rows, gridlayout.GridRow{
			Type: gridlayout.RowTypeNormal,
			Cells: []gridlayout.GridCell{
				{Component: gridlayout.NewLabelWithWidth("Scene", infoLabelWidth)},
				{Component: createSceneBtn},
			},
		})
	}

	// 2. Settings section (name, archetype)
	settingsHeader := field.NewHeaderComponent("settings-header", "Settings", &m.styles, m.zones)
	rows = append(rows, gridlayout.GridRow{
		Type:    gridlayout.RowTypeSection,
		Section: settingsHeader,
	})

	if room.Metadata != nil && room.Metadata.Name != nil {
		// Use different field ID prefix for rooms vs zones
		namePrefix := "room-name:"
		if isZone {
			namePrefix = "zone-name:"
		}

		textInput := field.NewTextComponent(
			namePrefix+roomID, "Name", *room.Metadata.Name,
			&m.styles, m.zones,
		)
		rows = append(rows, gridlayout.GridRow{
			Type: gridlayout.RowTypeNormal,
			Cells: []gridlayout.GridCell{
				{Component: gridlayout.NewLabelWithWidth("Name", infoLabelWidth)},
				{Component: textInput},
			},
		})
	}

	// Archetype (editable)
	if room.Metadata != nil && room.Metadata.Archetype != nil {
		archetypeKeys := hue.RoomArchetypeList()
		var options []field.Option
		currentIndex := 0
		currentArchetype := string(*room.Metadata.Archetype)

		for i, key := range archetypeKeys {
			displayName := hue.RoomArchetypeDisplayNames[key]
			options = append(options, field.Option{Label: displayName, Value: i})
			if key == currentArchetype {
				currentIndex = i
			}
		}

		// Use different field ID prefix for rooms vs zones
		fieldPrefix := "room-archetype:"
		if isZone {
			fieldPrefix = "zone-archetype:"
		}

		selectComp := field.NewSelectComponent(
			fieldPrefix+roomID, "Archetype", currentIndex, options,
			&m.styles, m.zones,
		)
		rows = append(rows, gridlayout.GridRow{
			Type: gridlayout.RowTypeNormal,
			Cells: []gridlayout.GridCell{
				{Component: gridlayout.NewLabelWithWidth("Archetype", infoLabelWidth)},
				{Component: selectComp},
			},
		})
	}

	// 3. Status section
	statusHeader := field.NewHeaderComponent("status-header", "Status", &m.styles, m.zones)
	rows = append(rows, gridlayout.GridRow{
		Type:    gridlayout.RowTypeSection,
		Section: statusHeader,
	})

	onCount := 0
	for _, l := range lights {
		if panels.IsLightOn(l) {
			onCount++
		}
	}
	rows = append(rows, gridlayout.NewInfoRow("Lights", fmt.Sprintf("%d/%d on", onCount, len(lights)), infoLabelWidth))

	// Motion sensor status - check devices in the room for motion sensors
	if room.Children != nil {
		var motionSensors int
		var motionDetected bool
		for _, child := range *room.Children {
			if child.Rid == nil || child.Rtype == nil || *child.Rtype != hueclient.ResourceIdentifierRtypeDevice {
				continue
			}
			device, ok := state.GetDevice(*child.Rid)
			if !ok {
				continue
			}
			if hasMotion, detecting := state.GetDeviceMotionState(device); hasMotion {
				motionSensors++
				if detecting {
					motionDetected = true
				}
			}
		}
		if motionSensors > 0 {
			motionStyle := lipgloss.NewStyle()
			var motionStatus string
			if motionDetected {
				motionStyle = motionStyle.Foreground(m.styles.Theme.Warning)
				motionStatus = motionStyle.Render("● motion detected")
			} else {
				motionStyle = motionStyle.Foreground(m.styles.Theme.TextMuted)
				motionStatus = motionStyle.Render("○ clear")
			}
			if motionSensors > 1 {
				motionStatus += m.styles.Dimmed.Render(fmt.Sprintf(" (%d sensors)", motionSensors))
			}
			rows = append(rows, gridlayout.NewInfoRow("Motion", motionStatus, infoLabelWidth))
		}
	}

	// Lights section
	if len(lights) > 0 {
		lightsHeaderText := fmt.Sprintf("Lights %s%d%s", m.styles.Dimmed.Render("["), len(lights), m.styles.Dimmed.Render("]"))
		lightsHeader := field.NewHeaderComponent("lights-header", lightsHeaderText, &m.styles, m.zones)
		rows = append(rows, gridlayout.GridRow{
			Type:    gridlayout.RowTypeSection,
			Section: lightsHeader,
		})

		lightStyle := lipgloss.NewStyle().Foreground(m.styles.Theme.EntityLight)
		for _, light := range lights {
			name := state.GetLightName(light)
			isOn := panels.IsLightOn(light)
			indicator := m.renderLightIndicator(light, isOn)

			var detailParts []string
			if isOn {
				if light.Dimming != nil && light.Dimming.Brightness != nil {
					detailParts = append(detailParts, fmt.Sprintf("%.0f%%", *light.Dimming.Brightness))
				}
				if light.ColorTemperature != nil && light.ColorTemperature.Mirek != nil {
					mirek := *light.ColorTemperature.Mirek
					kelvin := 1000000 / mirek
					detailParts = append(detailParts, fmt.Sprintf("%dK", kelvin))
				}
			} else {
				detailParts = append(detailParts, "off")
			}

			suffix := ""
			if len(detailParts) > 0 {
				suffix = " " + m.styles.Dimmed.Render(strings.Join(detailParts, ", "))
			}
			rows = append(rows, gridlayout.NewListItemRow(indicator, lightStyle.Render(name)+suffix))
		}
	}

	// Non-light devices
	if room.Children != nil && len(*room.Children) > 0 {
		var nonLightDevices []hueclient.DeviceGet
		for _, child := range *room.Children {
			if child.Rid == nil || child.Rtype == nil || *child.Rtype != hueclient.ResourceIdentifierRtypeDevice {
				continue
			}
			device, ok := state.GetDevice(*child.Rid)
			if !ok {
				continue
			}
			isLight := false
			if device.Services != nil {
				for _, svc := range *device.Services {
					if svc.Rtype != nil && *svc.Rtype == hueclient.ResourceIdentifierRtypeLight {
						isLight = true
						break
					}
				}
			}
			if isLight {
				continue
			}
			nonLightDevices = append(nonLightDevices, device)
		}

		if len(nonLightDevices) > 0 {
			devicesHeaderText := fmt.Sprintf("Devices %s%d%s", m.styles.Dimmed.Render("["), len(nonLightDevices), m.styles.Dimmed.Render("]"))
			devicesHeader := field.NewHeaderComponent("devices-header", devicesHeaderText, &m.styles, m.zones)
			rows = append(rows, gridlayout.GridRow{
				Type:    gridlayout.RowTypeSection,
				Section: devicesHeader,
			})
			deviceStyle := lipgloss.NewStyle().Foreground(m.styles.Theme.EntityDevice)
			for _, device := range nonLightDevices {
				name := device.DeviceName("")
				indicator := "•"
				suffix := ""

				// Check for motion sensor
				if hasMotion, detecting := state.GetDeviceMotionState(device); hasMotion {
					if detecting {
						indicator = lipgloss.NewStyle().Foreground(m.styles.Theme.Warning).Render("●")
						suffix = " " + m.styles.Dimmed.Render("motion")
					} else {
						indicator = lipgloss.NewStyle().Foreground(m.styles.Theme.TextMuted).Render("○")
						suffix = " " + m.styles.Dimmed.Render("clear")
					}
				}

				rows = append(rows, gridlayout.NewListItemRow(indicator, deviceStyle.Render(name)+suffix))
			}
		}
	}

	// Scenes
	scenes := state.RoomScenes(roomID)
	if len(scenes) > 0 {
		scenesHeaderText := fmt.Sprintf("Scenes %s%d%s", m.styles.Dimmed.Render("["), len(scenes), m.styles.Dimmed.Render("]"))
		scenesHeader := field.NewHeaderComponent("scenes-header", scenesHeaderText, &m.styles, m.zones)
		rows = append(rows, gridlayout.GridRow{
			Type:    gridlayout.RowTypeSection,
			Section: scenesHeader,
		})
		sceneStyle := lipgloss.NewStyle().Foreground(m.styles.Theme.EntityScene)
		for _, scene := range scenes {
			rows = append(rows, gridlayout.NewListItemRow("•", sceneStyle.Render(scene.SceneName(""))))
		}
	}

	// IDs section
	idsHeader := field.NewHeaderComponent("ids-header", "IDs", &m.styles, m.zones)
	rows = append(rows, gridlayout.GridRow{
		Type:    gridlayout.RowTypeSection,
		Section: idsHeader,
	})

	dimStyle := m.styles.Dimmed
	if room.Id != nil {
		rows = append(rows, gridlayout.NewStyledInfoRow("ID", *room.Id, infoLabelWidth, dimStyle))
	}
	if room.IdV1 != nil && *room.IdV1 != "" {
		rows = append(rows, gridlayout.NewStyledInfoRow("V1 ID", *room.IdV1, infoLabelWidth, dimStyle))
	}
	typeStr := "Room"
	if isZone {
		typeStr = "Zone"
	}
	rows = append(rows, gridlayout.NewStyledInfoRow("Type", typeStr, infoLabelWidth, dimStyle))

	return rows
}

// buildSceneGridRows builds grid rows for a scene details panel.
// Sections are ordered by immediacy: Controls > Settings > Info > Actions > IDs
func (m *Model) buildSceneGridRows(scene hueclient.SceneGet, state *hue.BridgeState) []gridlayout.GridRow {
	var rows []gridlayout.GridRow

	sceneID := ""
	if scene.Id != nil {
		sceneID = *scene.Id
	}

	// 1. Controls section - instant action (recall/activate scene)
	controlsHeader := field.NewHeaderComponent("controls-header", "Controls", &m.styles, m.zones)
	rows = append(rows, gridlayout.GridRow{
		Type:    gridlayout.RowTypeSection,
		Section: controlsHeader,
	})

	if sceneID != "" {
		activateBtn := field.NewButtonComponent(
			"scene-recall:"+sceneID, "Activate", "Activate",
			&m.styles, m.zones,
		)
		rows = append(rows, gridlayout.GridRow{
			Type: gridlayout.RowTypeNormal,
			Cells: []gridlayout.GridCell{
				{Component: gridlayout.NewLabelWithWidth("Activate", infoLabelWidth)},
				{Component: activateBtn},
			},
		})
	}

	// 2. Settings section (name)
	settingsHeader := field.NewHeaderComponent("settings-header", "Settings", &m.styles, m.zones)
	rows = append(rows, gridlayout.GridRow{
		Type:    gridlayout.RowTypeSection,
		Section: settingsHeader,
	})

	if scene.Metadata != nil && scene.Metadata.Name != nil {
		textInput := field.NewTextComponent(
			"scene-name:"+sceneID, "Name", *scene.Metadata.Name,
			&m.styles, m.zones,
		)
		rows = append(rows, gridlayout.GridRow{
			Type: gridlayout.RowTypeNormal,
			Cells: []gridlayout.GridCell{
				{Component: gridlayout.NewLabelWithWidth("Name", infoLabelWidth)},
				{Component: textInput},
			},
		})
	}

	// 3. Info section (status, group, speed, etc.)
	infoHeader := field.NewHeaderComponent("info-header", "Info", &m.styles, m.zones)
	rows = append(rows, gridlayout.GridRow{
		Type:    gridlayout.RowTypeSection,
		Section: infoHeader,
	})

	if scene.Status != nil && scene.Status.Active != nil {
		status := string(*scene.Status.Active)
		statusStyle := m.styles.Dimmed
		if status == "active" {
			statusStyle = m.styles.Success
		}
		rows = append(rows, gridlayout.NewStyledInfoRow("Status", status, infoLabelWidth, statusStyle))
	}

	// Group (room/zone)
	if scene.Group != nil && scene.Group.Rid != nil && state != nil {
		groupName := ""
		groupType := "unknown"
		var groupStyle lipgloss.Style
		if scene.Group.Rtype != nil {
			groupType = string(*scene.Group.Rtype)
		}
		if room, ok := state.GetRoom(*scene.Group.Rid); ok {
			groupName = state.GetRoomName(room)
			groupStyle = lipgloss.NewStyle().Foreground(m.styles.Theme.EntityRoom)
		} else if zone, ok := state.GetZone(*scene.Group.Rid); ok {
			groupName = zone.RoomName("")
			groupStyle = lipgloss.NewStyle().Foreground(m.styles.Theme.EntityZone)
		}
		if groupName != "" {
			rows = append(rows, gridlayout.NewStyledInfoRow("Group", groupStyle.Render(groupName), infoLabelWidth, lipgloss.NewStyle()))
		}
		rows = append(rows, gridlayout.NewStyledInfoRow("Group Type", groupType, infoLabelWidth, m.styles.Dimmed))
	}

	// Speed and auto dynamic
	if scene.Speed != nil {
		rows = append(rows, gridlayout.NewInfoRow("Speed", fmt.Sprintf("%.2f", *scene.Speed), infoLabelWidth))
	}
	if scene.AutoDynamic != nil {
		rows = append(rows, gridlayout.NewInfoRow("Auto Dynamic", fmt.Sprintf("%v", *scene.AutoDynamic), infoLabelWidth))
	}

	// Actions
	if scene.Actions != nil && len(*scene.Actions) > 0 {
		actionsHeaderText := fmt.Sprintf("Actions %s%d%s", m.styles.Dimmed.Render("["), len(*scene.Actions), m.styles.Dimmed.Render("]"))
		actionsHeader := field.NewHeaderComponent("actions-header", actionsHeaderText, &m.styles, m.zones)
		rows = append(rows, gridlayout.GridRow{
			Type:    gridlayout.RowTypeSection,
			Section: actionsHeader,
		})

		actionLightStyle := lipgloss.NewStyle().Foreground(m.styles.Theme.EntityLight)
		for _, action := range *scene.Actions {
			targetName := "unknown"
			if action.Target != nil && action.Target.Rid != nil {
				targetName = *action.Target.Rid
				if state != nil {
					if light, ok := state.GetLight(*action.Target.Rid); ok {
						targetName = state.GetLightName(light)
					}
				}
			}

			actionDesc := ""
			if action.Action != nil {
				var parts []string
				if action.Action.On != nil && action.Action.On.On != nil {
					if *action.Action.On.On {
						parts = append(parts, "on")
					} else {
						parts = append(parts, "off")
					}
				}
				if action.Action.Dimming != nil && action.Action.Dimming.Brightness != nil {
					parts = append(parts, fmt.Sprintf("%.0f%%", *action.Action.Dimming.Brightness))
				}
				if action.Action.ColorTemperature != nil && action.Action.ColorTemperature.Mirek != nil {
					kelvin := 1000000 / *action.Action.ColorTemperature.Mirek
					parts = append(parts, fmt.Sprintf("%dK", kelvin))
				}
				if len(parts) > 0 {
					actionDesc = " → " + strings.Join(parts, ", ")
				}
			}

			rows = append(rows, gridlayout.NewListItemRow("•", actionLightStyle.Render(targetName)+m.styles.Dimmed.Render(actionDesc)))
		}
	}

	// IDs section
	idsHeader := field.NewHeaderComponent("ids-header", "IDs", &m.styles, m.zones)
	rows = append(rows, gridlayout.GridRow{
		Type:    gridlayout.RowTypeSection,
		Section: idsHeader,
	})

	dimStyle := m.styles.Dimmed
	if scene.Id != nil {
		rows = append(rows, gridlayout.NewStyledInfoRow("ID", *scene.Id, infoLabelWidth, dimStyle))
	}
	if scene.IdV1 != nil {
		rows = append(rows, gridlayout.NewStyledInfoRow("V1 ID", *scene.IdV1, infoLabelWidth, dimStyle))
	}

	return rows
}

// buildDeviceGridRows builds grid rows for a device details panel.
// Sections are ordered by immediacy: Controls > Settings > Sensors > Product > Zigbee > IDs
func (m *Model) buildDeviceGridRows(device hueclient.DeviceGet, state *hue.BridgeState) []gridlayout.GridRow {
	var rows []gridlayout.GridRow

	deviceID := ""
	if device.Id != nil {
		deviceID = *device.Id
	}

	// 1. Controls section - instant actions (identify)
	controlsHeader := field.NewHeaderComponent("controls-header", "Controls", &m.styles, m.zones)
	rows = append(rows, gridlayout.GridRow{
		Type:    gridlayout.RowTypeSection,
		Section: controlsHeader,
	})

	if deviceID != "" {
		identifyBtn := field.NewButtonComponent(
			"identify:"+deviceID, "Identify", "Identify",
			&m.styles, m.zones,
		)
		rows = append(rows, gridlayout.GridRow{
			Type: gridlayout.RowTypeNormal,
			Cells: []gridlayout.GridCell{
				{Component: gridlayout.NewLabelWithWidth("Identify", infoLabelWidth)},
				{Component: identifyBtn},
			},
		})
	}

	// 2. Settings section (name, sensor toggles)
	settingsHeader := field.NewHeaderComponent("settings-header", "Settings", &m.styles, m.zones)
	rows = append(rows, gridlayout.GridRow{
		Type:    gridlayout.RowTypeSection,
		Section: settingsHeader,
	})

	if device.Metadata != nil && device.Metadata.Name != nil {
		textInput := field.NewTextComponent(
			"device-name:"+deviceID, "Name", *device.Metadata.Name,
			&m.styles, m.zones,
		)
		rows = append(rows, gridlayout.GridRow{
			Type: gridlayout.RowTypeNormal,
			Cells: []gridlayout.GridCell{
				{Component: gridlayout.NewLabelWithWidth("Name", infoLabelWidth)},
				{Component: textInput},
			},
		})
	}

	// Room assignment dropdown
	if state != nil && deviceID != "" {
		currentRoom, hasRoom := state.GetDeviceRoom(deviceID)
		currentRoomID := ""
		if hasRoom && currentRoom.Id != nil {
			currentRoomID = *currentRoom.Id
		}

		// Build room options: "No Room" + all rooms
		allRooms := state.AllRooms()
		options := make([]field.Option, 0, len(allRooms)+1)
		options = append(options, field.Option{Label: "No Room", Value: 0})
		selectedIndex := 0

		for i, room := range allRooms {
			roomName := "Unknown"
			roomID := ""
			if room.Metadata != nil && room.Metadata.Name != nil {
				roomName = *room.Metadata.Name
			}
			if room.Id != nil {
				roomID = *room.Id
			}
			options = append(options, field.Option{Label: roomName, Value: i + 1, Color: m.styles.Theme.EntityRoom})
			if roomID == currentRoomID {
				selectedIndex = i + 1
			}
		}

		roomSelect := field.NewSelectComponent(
			"device-room:"+deviceID, "Room", selectedIndex, options,
			&m.styles, m.zones,
		)
		rows = append(rows, gridlayout.GridRow{
			Type: gridlayout.RowTypeNormal,
			Cells: []gridlayout.GridCell{
				{Component: gridlayout.NewLabelWithWidth("Room", infoLabelWidth)},
				{Component: roomSelect},
			},
		})
	}

	// Sensor enable/disable toggles in Settings
	if state != nil {
		// Motion sensor toggle and sensitivity
		if motionID, motion, found := state.GetDeviceMotionSensor(device); found {
			enabled := motion.Enabled != nil && *motion.Enabled
			motionToggle := field.NewToggleComponent(
				"motion-enabled:"+motionID, "Motion Sensor", enabled,
				&m.styles, m.zones,
			)
			rows = append(rows, gridlayout.GridRow{
				Type: gridlayout.RowTypeNormal,
				Cells: []gridlayout.GridCell{
					{Component: gridlayout.NewLabelWithWidth("Motion Sensor", infoLabelWidth)},
					{Component: motionToggle},
				},
			})

			// Sensitivity slider
			if motion.Sensitivity != nil && motion.Sensitivity.SensitivityMax != nil {
				sensitivity := 0
				if motion.Sensitivity.Sensitivity != nil {
					sensitivity = *motion.Sensitivity.Sensitivity
				}
				maxSensitivity := *motion.Sensitivity.SensitivityMax
				sensitivitySlider := field.NewSliderComponent(
					"motion-sensitivity:"+motionID, "Sensitivity",
					sensitivity, 0, maxSensitivity, 1,
					&m.styles, m.zones,
				)
				rows = append(rows, gridlayout.GridRow{
					Type: gridlayout.RowTypeNormal,
					Cells: []gridlayout.GridCell{
						{Component: gridlayout.NewLabelWithWidth("Sensitivity", infoLabelWidth)},
						{Component: sensitivitySlider},
					},
				})
			}
		}

		// Temperature sensor toggle
		if tempID, tempSensor, found := state.GetDeviceTemperatureSensor(device); found {
			enabled := tempSensor.Enabled != nil && *tempSensor.Enabled
			tempToggle := field.NewToggleComponent(
				"temp-enabled:"+tempID, "Temp Sensor", enabled,
				&m.styles, m.zones,
			)
			rows = append(rows, gridlayout.GridRow{
				Type: gridlayout.RowTypeNormal,
				Cells: []gridlayout.GridCell{
					{Component: gridlayout.NewLabelWithWidth("Temp Sensor", infoLabelWidth)},
					{Component: tempToggle},
				},
			})
		}

		// Light level sensor toggle
		if llID, llSensor, found := state.GetDeviceLightLevelSensor(device); found {
			enabled := llSensor.Enabled != nil && *llSensor.Enabled
			llToggle := field.NewToggleComponent(
				"ll-enabled:"+llID, "Light Sensor", enabled,
				&m.styles, m.zones,
			)
			rows = append(rows, gridlayout.GridRow{
				Type: gridlayout.RowTypeNormal,
				Cells: []gridlayout.GridCell{
					{Component: gridlayout.NewLabelWithWidth("Light Sensor", infoLabelWidth)},
					{Component: llToggle},
				},
			})
		}
	}

	// 3. Product section
	if device.ProductData != nil {
		pd := device.ProductData
		header := field.NewHeaderComponent("product-header", "Product", &m.styles, m.zones)
		rows = append(rows, gridlayout.GridRow{
			Type:    gridlayout.RowTypeSection,
			Section: header,
		})

		if pd.ProductName != nil {
			rows = append(rows, gridlayout.NewInfoRow("Product", *pd.ProductName, infoLabelWidth))
		}
		if pd.ManufacturerName != nil {
			rows = append(rows, gridlayout.NewInfoRow("Manufacturer", *pd.ManufacturerName, infoLabelWidth))
		}
		if pd.ModelId != nil {
			rows = append(rows, gridlayout.NewInfoRow("Model", *pd.ModelId, infoLabelWidth))
		}
		if pd.SoftwareVersion != nil {
			rows = append(rows, gridlayout.NewInfoRow("Firmware", *pd.SoftwareVersion, infoLabelWidth))
		}
		if pd.ProductArchetype != nil {
			rows = append(rows, gridlayout.NewStyledInfoRow("Archetype", string(*pd.ProductArchetype), infoLabelWidth, m.styles.Dimmed))
		}
	}

	// Sensors section
	if state != nil {
		var sensorRows []gridlayout.GridRow

		// Motion sensor (read-only status)
		if _, motion, found := state.GetDeviceMotionSensor(device); found {
			isDetecting := false
			if motion.Motion != nil {
				if motion.Motion.MotionReport != nil && motion.Motion.MotionReport.Motion != nil {
					isDetecting = *motion.Motion.MotionReport.Motion
				} else if motion.Motion.Motion != nil {
					isDetecting = *motion.Motion.Motion
				}
			}
			status := "clear"
			statusStyle := m.styles.Dimmed
			if isDetecting {
				status = "detected"
				statusStyle = m.styles.Success
			}
			sensorRows = append(sensorRows, gridlayout.NewStyledInfoRow("Motion", status, infoLabelWidth, statusStyle))
		}

		// Temperature (read-only value)
		if _, tempSensor, found := state.GetDeviceTemperatureSensor(device); found {
			tempValue := "N/A"
			if tempSensor.Temperature != nil {
				if tempSensor.Temperature.TemperatureReport != nil && tempSensor.Temperature.TemperatureReport.Temperature != nil {
					tempValue = fmt.Sprintf("%.1f°C", *tempSensor.Temperature.TemperatureReport.Temperature)
				} else if tempSensor.Temperature.Temperature != nil {
					tempValue = fmt.Sprintf("%.1f°C", *tempSensor.Temperature.Temperature)
				}
			}
			sensorRows = append(sensorRows, gridlayout.NewInfoRow("Temperature", tempValue, infoLabelWidth))
		}

		// Light level (read-only value)
		if _, llSensor, found := state.GetDeviceLightLevelSensor(device); found {
			llValue := "N/A"
			if llSensor.Light != nil {
				level := 0
				if llSensor.Light.LightLevelReport != nil && llSensor.Light.LightLevelReport.LightLevel != nil {
					level = *llSensor.Light.LightLevelReport.LightLevel
				} else if llSensor.Light.LightLevel != nil {
					level = *llSensor.Light.LightLevel
				}
				if level > 1 {
					lux := math.Pow(10, float64(level-1)/10000.0)
					llValue = fmt.Sprintf("%.0f lux", lux)
				} else {
					llValue = "0 lux"
				}
			}
			sensorRows = append(sensorRows, gridlayout.NewInfoRow("Light Level", llValue, infoLabelWidth))
		}

		// Battery
		if hasBattery, battLevel, battState := state.GetDeviceBattery(device); hasBattery {
			value := fmt.Sprintf("%d%%", battLevel)
			if battState != "" {
				value += fmt.Sprintf(" (%s)", battState)
			}
			sensorRows = append(sensorRows, gridlayout.NewInfoRow("Battery", value, infoLabelWidth))
		}

		if len(sensorRows) > 0 {
			sensorsHeader := field.NewHeaderComponent("sensors-header", "Sensors", &m.styles, m.zones)
			rows = append(rows, gridlayout.GridRow{
				Type:    gridlayout.RowTypeSection,
				Section: sensorsHeader,
			})
			rows = append(rows, sensorRows...)
		}

		// Zigbee connectivity
		if zc, ok := state.GetDeviceZigbeeConnectivity(device); ok {
			zigbeeHeader := field.NewHeaderComponent("zigbee-header", "Zigbee", &m.styles, m.zones)
			rows = append(rows, gridlayout.GridRow{
				Type:    gridlayout.RowTypeSection,
				Section: zigbeeHeader,
			})

			statusStyle := m.styles.Dimmed
			if zc.Status == "connected" {
				statusStyle = m.styles.Success
			} else if zc.Status == "connectivity_issue" || zc.Status == "disconnected" {
				statusStyle = m.styles.Error
			}
			rows = append(rows, gridlayout.NewStyledInfoRow("Status", zc.Status, infoLabelWidth, statusStyle))

			if zc.MacAddress != "" {
				rows = append(rows, gridlayout.NewStyledInfoRow("MAC", zc.MacAddress, infoLabelWidth, m.styles.Dimmed))
			}
		}
	}

	// IDs section
	idsHeader := field.NewHeaderComponent("ids-header", "IDs", &m.styles, m.zones)
	rows = append(rows, gridlayout.GridRow{
		Type:    gridlayout.RowTypeSection,
		Section: idsHeader,
	})

	dimStyle := m.styles.Dimmed
	if device.Id != nil {
		rows = append(rows, gridlayout.NewStyledInfoRow("ID", *device.Id, infoLabelWidth, dimStyle))
	}
	if device.Type != nil {
		rows = append(rows, gridlayout.NewStyledInfoRow("Type", string(*device.Type), infoLabelWidth, dimStyle))
	}

	return rows
}

// buildBridgeGridRows builds grid rows for a bridge details panel.
func (m *Model) buildBridgeGridRows(bridgeID string) []gridlayout.GridRow {
	var rows []gridlayout.GridRow

	bridge := m.manager.GetBridge(bridgeID)
	if bridge == nil {
		return rows
	}

	state := bridge.GetState()

	// 1. Controls section - instant actions (identify)
	if state != nil {
		if bridgeDevice, ok := state.GetBridgeDevice(); ok {
			deviceID := ""
			if bridgeDevice.Id != nil {
				deviceID = *bridgeDevice.Id
			}

			controlsHeader := field.NewHeaderComponent("controls-header", "Controls", &m.styles, m.zones)
			rows = append(rows, gridlayout.GridRow{
				Type:    gridlayout.RowTypeSection,
				Section: controlsHeader,
			})

			// Identify button
			if deviceID != "" {
				identifyBtn := field.NewButtonComponent(
					"bridge-identify:"+deviceID, "Identify", "Identify",
					&m.styles, m.zones,
				)
				rows = append(rows, gridlayout.GridRow{
					Type: gridlayout.RowTypeNormal,
					Cells: []gridlayout.GridCell{
						{Component: gridlayout.NewLabelWithWidth("Identify", infoLabelWidth)},
						{Component: identifyBtn},
					},
				})
			}

			// 2. Settings section (name)
			settingsHeader := field.NewHeaderComponent("settings-header", "Settings", &m.styles, m.zones)
			rows = append(rows, gridlayout.GridRow{
				Type:    gridlayout.RowTypeSection,
				Section: settingsHeader,
			})

			// Name field
			if bridgeDevice.Metadata != nil && bridgeDevice.Metadata.Name != nil && deviceID != "" {
				textInput := field.NewTextComponent(
					"bridge-name:"+deviceID, "Name", *bridgeDevice.Metadata.Name,
					&m.styles, m.zones,
				)
				rows = append(rows, gridlayout.GridRow{
					Type: gridlayout.RowTypeNormal,
					Cells: []gridlayout.GridCell{
						{Component: gridlayout.NewLabelWithWidth("Name", infoLabelWidth)},
						{Component: textInput},
					},
				})
			}
		}
	}

	// Connection section
	connHeader := field.NewHeaderComponent("conn-header", "Connection", &m.styles, m.zones)
	rows = append(rows, gridlayout.GridRow{
		Type:    gridlayout.RowTypeSection,
		Section: connHeader,
	})

	statusStyle := m.styles.Dimmed
	statusText := "disconnected"
	switch bridge.Status {
	case hue.StatusConnected:
		statusStyle, statusText = m.styles.Success, "connected"
	case hue.StatusConnecting:
		statusStyle, statusText = m.styles.Warning, "connecting"
	case hue.StatusPairing:
		statusStyle, statusText = m.styles.Warning, "pairing"
	case hue.StatusError:
		statusStyle, statusText = m.styles.Error, "error"
	}
	rows = append(rows, gridlayout.NewStyledInfoRow("Status", statusText, infoLabelWidth, statusStyle))
	rows = append(rows, gridlayout.NewInfoRow("IP Address", bridge.Info.IPAddress, infoLabelWidth))
	if !bridge.LastSync.IsZero() {
		rows = append(rows, gridlayout.NewInfoRow("Last Sync", bridge.LastSync.Format("15:04:05"), infoLabelWidth))
	}

	// Hardware section
	if state != nil {
		if bridgeDevice, ok := state.GetBridgeDevice(); ok && bridgeDevice.ProductData != nil {
			pd := bridgeDevice.ProductData
			hwHeader := field.NewHeaderComponent("hw-header", "Hardware", &m.styles, m.zones)
			rows = append(rows, gridlayout.GridRow{
				Type:    gridlayout.RowTypeSection,
				Section: hwHeader,
			})

			if pd.ProductName != nil {
				rows = append(rows, gridlayout.NewInfoRow("Product", *pd.ProductName, infoLabelWidth))
			}
			if pd.ModelId != nil {
				rows = append(rows, gridlayout.NewInfoRow("Model", *pd.ModelId, infoLabelWidth))
			}
			if pd.SoftwareVersion != nil {
				rows = append(rows, gridlayout.NewInfoRow("Firmware", *pd.SoftwareVersion, infoLabelWidth))
			}
		}
	}

	// Summary section
	if state != nil {
		summaryHeader := field.NewHeaderComponent("summary-header", "Summary", &m.styles, m.zones)
		rows = append(rows, gridlayout.GridRow{
			Type:    gridlayout.RowTypeSection,
			Section: summaryHeader,
		})

		rooms := state.AllRooms()
		zones := state.AllZones()
		lights := state.AllLights()
		scenes := state.AllScenes()
		devices := state.AllDevices()

		rows = append(rows, gridlayout.NewInfoRow("Rooms", fmt.Sprintf("%d", len(rooms)), infoLabelWidth))
		rows = append(rows, gridlayout.NewInfoRow("Zones", fmt.Sprintf("%d", len(zones)), infoLabelWidth))
		rows = append(rows, gridlayout.NewInfoRow("Lights", fmt.Sprintf("%d", len(lights)), infoLabelWidth))
		rows = append(rows, gridlayout.NewInfoRow("Scenes", fmt.Sprintf("%d", len(scenes)), infoLabelWidth))
		rows = append(rows, gridlayout.NewInfoRow("Devices", fmt.Sprintf("%d", len(devices)), infoLabelWidth))

		lightsOn := 0
		for _, light := range lights {
			if panels.IsLightOn(light) {
				lightsOn++
			}
		}
		rows = append(rows, gridlayout.NewInfoRow("Lights On", fmt.Sprintf("%d/%d", lightsOn, len(lights)), infoLabelWidth))
	}

	// IDs section
	idsHeader := field.NewHeaderComponent("ids-header", "IDs", &m.styles, m.zones)
	rows = append(rows, gridlayout.GridRow{
		Type:    gridlayout.RowTypeSection,
		Section: idsHeader,
	})
	rows = append(rows, gridlayout.NewStyledInfoRow("Bridge ID", bridge.Info.ID, infoLabelWidth, m.styles.Dimmed))

	return rows
}

// buildEntertainmentGridRows builds grid rows for an entertainment configuration details panel.
func (m *Model) buildEntertainmentGridRows(cfg hue.EntertainmentConfiguration, state *hue.BridgeState) []gridlayout.GridRow {
	var rows []gridlayout.GridRow

	// Status section
	header := field.NewHeaderComponent("status-header", "Status", &m.styles, m.zones)
	rows = append(rows, gridlayout.GridRow{
		Type:    gridlayout.RowTypeSection,
		Section: header,
	})

	if cfg.Status != "" {
		statusStyle := m.styles.Dimmed
		if cfg.Status == "active" {
			statusStyle = m.styles.Success
		}
		rows = append(rows, gridlayout.NewStyledInfoRow("Status", cfg.Status, infoLabelWidth, statusStyle))
	}
	if cfg.ConfigurationType != "" {
		rows = append(rows, gridlayout.NewInfoRow("Type", cfg.ConfigurationType, infoLabelWidth))
	}

	// Lights section
	if len(cfg.Lights) > 0 {
		cfgLightsHeaderText := fmt.Sprintf("Lights %s%d%s", m.styles.Dimmed.Render("["), len(cfg.Lights), m.styles.Dimmed.Render("]"))
		lightsHeader := field.NewHeaderComponent("lights-header", cfgLightsHeaderText, &m.styles, m.zones)
		rows = append(rows, gridlayout.GridRow{
			Type:    gridlayout.RowTypeSection,
			Section: lightsHeader,
		})

		cfgLightStyle := lipgloss.NewStyle().Foreground(m.styles.Theme.EntityLight)
		for _, light := range cfg.Lights {
			if light.Service != nil && light.Service.RID != "" {
				lightName := light.Service.RID
				if state != nil {
					if l, ok := state.GetLight(light.Service.RID); ok {
						lightName = state.GetLightName(l)
					}
				}
				rows = append(rows, gridlayout.NewListItemRow("•", cfgLightStyle.Render(lightName)))
			}
		}
	}

	// Channels section
	if len(cfg.Channels) > 0 {
		channelsHeaderText := fmt.Sprintf("Channels %s%d%s", m.styles.Dimmed.Render("["), len(cfg.Channels), m.styles.Dimmed.Render("]"))
		channelsHeader := field.NewHeaderComponent("channels-header", channelsHeaderText, &m.styles, m.zones)
		rows = append(rows, gridlayout.GridRow{
			Type:    gridlayout.RowTypeSection,
			Section: channelsHeader,
		})

		for _, ch := range cfg.Channels {
			posStr := ""
			if ch.Position != nil {
				posStr = fmt.Sprintf(" (%.1f, %.1f, %.1f)", ch.Position.X, ch.Position.Y, ch.Position.Z)
			}
			rows = append(rows, gridlayout.NewListItemRow("•", fmt.Sprintf("Channel %d%s", ch.ChannelID, m.styles.Dimmed.Render(posStr))))
		}
	}

	// IDs section
	idsHeader := field.NewHeaderComponent("ids-header", "IDs", &m.styles, m.zones)
	rows = append(rows, gridlayout.GridRow{
		Type:    gridlayout.RowTypeSection,
		Section: idsHeader,
	})
	rows = append(rows, gridlayout.NewStyledInfoRow("ID", cfg.ID, infoLabelWidth, m.styles.Dimmed))

	return rows
}

// buildRoomsCategoryGridRows builds a list of all rooms in a category.
func (m *Model) buildRoomsCategoryGridRows(data panels.RoomsCategoryData, state *hue.BridgeState) []gridlayout.GridRow {
	var rows []gridlayout.GridRow

	// Header with count
	roomsHeaderText := fmt.Sprintf("Rooms %s%d%s", m.styles.Dimmed.Render("["), len(data.Rooms), m.styles.Dimmed.Render("]"))
	header := field.NewHeaderComponent("rooms-header", roomsHeaderText, &m.styles, m.zones)
	rows = append(rows, gridlayout.GridRow{
		Type:    gridlayout.RowTypeSection,
		Section: header,
	})

	// Count on/off rooms
	onCount := 0
	for _, room := range data.Rooms {
		lights := state.RoomLights(room)
		for _, light := range lights {
			if panels.IsLightOn(light) {
				onCount++
				break
			}
		}
	}

	rows = append(rows, gridlayout.NewInfoRow("Active", fmt.Sprintf("%d of %d", onCount, len(data.Rooms)), infoLabelWidth))
	rows = append(rows, gridlayout.NewEmptyRow())

	// List each room
	roomStyle := lipgloss.NewStyle().Foreground(m.styles.Theme.EntityRoom)
	for _, room := range data.Rooms {
		name := "Unknown"
		if room.Metadata != nil && room.Metadata.Name != nil {
			name = *room.Metadata.Name
		}

		// Get room's lights and calculate state
		lights := state.RoomLights(room)
		brightness, indicatorColor := hue.CalculateRoomAggregate(lights)

		// Build indicator
		indicator := m.styles.Dimmed.Render("○")
		if brightness > 0 {
			indicator = ui.RenderBrightnessIndicatorFromHex(brightness, indicatorColor)
		}

		rows = append(rows, gridlayout.NewListItemRow(indicator, roomStyle.Render(name)))
	}

	return rows
}

// buildZonesCategoryGridRows builds a list of all zones in a category.
func (m *Model) buildZonesCategoryGridRows(data panels.ZonesCategoryData, state *hue.BridgeState) []gridlayout.GridRow {
	var rows []gridlayout.GridRow

	// Header with count
	zonesHeaderText := fmt.Sprintf("Zones %s%d%s", m.styles.Dimmed.Render("["), len(data.Zones), m.styles.Dimmed.Render("]"))
	header := field.NewHeaderComponent("zones-header", zonesHeaderText, &m.styles, m.zones)
	rows = append(rows, gridlayout.GridRow{
		Type:    gridlayout.RowTypeSection,
		Section: header,
	})

	// Count on/off zones
	onCount := 0
	for _, zone := range data.Zones {
		lights := state.ZoneLights(zone)
		for _, light := range lights {
			if panels.IsLightOn(light) {
				onCount++
				break
			}
		}
	}

	rows = append(rows, gridlayout.NewInfoRow("Active", fmt.Sprintf("%d of %d", onCount, len(data.Zones)), infoLabelWidth))
	rows = append(rows, gridlayout.NewEmptyRow())

	// List each zone
	zoneStyle := lipgloss.NewStyle().Foreground(m.styles.Theme.EntityZone)
	for _, zone := range data.Zones {
		name := "Unknown"
		if zone.Metadata != nil && zone.Metadata.Name != nil {
			name = *zone.Metadata.Name
		}

		// Get zone's lights and calculate state
		lights := state.ZoneLights(zone)
		brightness, indicatorColor := hue.CalculateRoomAggregate(lights)

		// Build indicator
		indicator := m.styles.Dimmed.Render("○")
		if brightness > 0 {
			indicator = ui.RenderBrightnessIndicatorFromHex(brightness, indicatorColor)
		}

		rows = append(rows, gridlayout.NewListItemRow(indicator, zoneStyle.Render(name)))
	}

	return rows
}

// buildLightsCategoryGridRows builds a list of all lights in a category.
func (m *Model) buildLightsCategoryGridRows(data panels.LightsCategoryData, _ *hue.BridgeState) []gridlayout.GridRow {
	var rows []gridlayout.GridRow

	// Header with parent context
	headerText := fmt.Sprintf("Lights %s%d%s", m.styles.Dimmed.Render("["), len(data.Lights), m.styles.Dimmed.Render("]"))
	if data.ParentName != "" {
		headerText = fmt.Sprintf("Lights in %s %s%d%s", data.ParentName, m.styles.Dimmed.Render("["), len(data.Lights), m.styles.Dimmed.Render("]"))
	}
	header := field.NewHeaderComponent("lights-header", headerText, &m.styles, m.zones)
	rows = append(rows, gridlayout.GridRow{
		Type:    gridlayout.RowTypeSection,
		Section: header,
	})

	// Count on/off lights
	onCount := 0
	for _, light := range data.Lights {
		if panels.IsLightOn(light) {
			onCount++
		}
	}

	rows = append(rows, gridlayout.NewInfoRow("On", fmt.Sprintf("%d of %d", onCount, len(data.Lights)), infoLabelWidth))
	rows = append(rows, gridlayout.NewEmptyRow())

	// List each light
	for _, light := range data.Lights {
		name := "Unknown"
		if light.Metadata != nil && light.Metadata.Name != nil {
			name = *light.Metadata.Name
		}

		isOn := panels.IsLightOn(light)
		brightness := 0.0
		indicatorColor := ""
		if isOn {
			if light.Dimming != nil && light.Dimming.Brightness != nil {
				brightness = float64(*light.Dimming.Brightness)
			} else {
				brightness = 100.0
			}
			indicatorColor = ui.GetLightColor(light)
		}

		// Build indicator
		indicator := m.styles.Dimmed.Render("○")
		if brightness > 0 {
			indicator = ui.RenderBrightnessIndicatorFromHex(brightness, indicatorColor)
		}

		catLightStyle := lipgloss.NewStyle().Foreground(m.styles.Theme.EntityLight)
		rows = append(rows, gridlayout.NewListItemRow(indicator, catLightStyle.Render(name)))
	}

	return rows
}

// buildDevicesCategoryGridRows builds a list of all devices in a category.
func (m *Model) buildDevicesCategoryGridRows(data panels.DevicesCategoryData, state *hue.BridgeState) []gridlayout.GridRow {
	var rows []gridlayout.GridRow

	// Header with parent context
	headerText := fmt.Sprintf("Devices %s%d%s", m.styles.Dimmed.Render("["), len(data.Devices), m.styles.Dimmed.Render("]"))
	if data.ParentName != "" {
		headerText = fmt.Sprintf("Devices in %s %s%d%s", data.ParentName, m.styles.Dimmed.Render("["), len(data.Devices), m.styles.Dimmed.Render("]"))
	}
	header := field.NewHeaderComponent("devices-header", headerText, &m.styles, m.zones)
	rows = append(rows, gridlayout.GridRow{
		Type:    gridlayout.RowTypeSection,
		Section: header,
	})

	rows = append(rows, gridlayout.NewInfoRow("Total", fmt.Sprintf("%d", len(data.Devices)), infoLabelWidth))
	rows = append(rows, gridlayout.NewEmptyRow())

	// List each device
	catDeviceStyle := lipgloss.NewStyle().Foreground(m.styles.Theme.EntityDevice)
	for _, device := range data.Devices {
		name := device.DeviceName("Unknown")

		// Check if device has motion sensor and its state
		hasMotion, isDetecting := state.GetDeviceMotionState(device)

		// Build indicator based on motion state
		indicator := m.styles.Dimmed.Render("○")
		if hasMotion && isDetecting {
			indicator = lipgloss.NewStyle().Foreground(m.styles.Theme.Warning).Render("●")
		}

		rows = append(rows, gridlayout.NewListItemRow(indicator, catDeviceStyle.Render(name)))
	}

	return rows
}

// buildScenesCategoryGridRows builds a list of all scenes in a category.
func (m *Model) buildScenesCategoryGridRows(data panels.ScenesCategoryData, _ *hue.BridgeState) []gridlayout.GridRow {
	var rows []gridlayout.GridRow

	// Header with parent context
	headerText := fmt.Sprintf("Scenes %s%d%s", m.styles.Dimmed.Render("["), len(data.Scenes), m.styles.Dimmed.Render("]"))
	if data.ParentName != "" {
		headerText = fmt.Sprintf("Scenes in %s %s%d%s", data.ParentName, m.styles.Dimmed.Render("["), len(data.Scenes), m.styles.Dimmed.Render("]"))
	}
	header := field.NewHeaderComponent("scenes-header", headerText, &m.styles, m.zones)
	rows = append(rows, gridlayout.GridRow{
		Type:    gridlayout.RowTypeSection,
		Section: header,
	})

	rows = append(rows, gridlayout.NewInfoRow("Total", fmt.Sprintf("%d", len(data.Scenes)), infoLabelWidth))
	rows = append(rows, gridlayout.NewEmptyRow())

	// List each scene
	catSceneStyle := lipgloss.NewStyle().Foreground(m.styles.Theme.EntityScene)
	for _, scene := range data.Scenes {
		name := scene.SceneName("Unknown")
		rows = append(rows, gridlayout.NewListItemRow("•", catSceneStyle.Render(name)))
	}

	return rows
}

// buildSmartSceneGridRows builds the detail grid rows for a smart scene.
func (m *Model) buildSmartSceneGridRows(scene hueclient.SmartSceneGet, state *hue.BridgeState, bridgeID string) []gridlayout.GridRow {
	var rows []gridlayout.GridRow

	sceneID := ""
	if scene.Id != nil {
		sceneID = *scene.Id
	}

	// 1. Controls section - activate/deactivate toggle and delete
	controlsHeader := field.NewHeaderComponent("controls-header", "Controls", &m.styles, m.zones)
	rows = append(rows, gridlayout.GridRow{
		Type:    gridlayout.RowTypeSection,
		Section: controlsHeader,
	})

	if sceneID != "" {
		// Activate/Deactivate toggle
		isActive := scene.State == "active"
		toggle := field.NewToggleComponent(
			"smartscene-toggle:"+sceneID, "Active", isActive,
			&m.styles, m.zones,
		)
		rows = append(rows, gridlayout.GridRow{
			Type: gridlayout.RowTypeNormal,
			Cells: []gridlayout.GridCell{
				{Component: gridlayout.NewLabelWithWidth("Active", infoLabelWidth)},
				{Component: toggle},
			},
		})

		}

	// 2. Info section
	infoHeader := field.NewHeaderComponent("info-header", "Info", &m.styles, m.zones)
	rows = append(rows, gridlayout.GridRow{
		Type:    gridlayout.RowTypeSection,
		Section: infoHeader,
	})

	// Name
	if scene.Metadata.Name != nil {
		rows = append(rows, gridlayout.NewInfoRow("Name", *scene.Metadata.Name, infoLabelWidth))
	}

	// State
	stateStr := string(scene.State)
	stateStyle := m.styles.Dimmed
	if scene.State == "active" {
		stateStyle = m.styles.Success
	}
	rows = append(rows, gridlayout.NewStyledInfoRow("State", stateStr, infoLabelWidth, stateStyle))

	// Group (room/zone)
	if scene.Group.Rid != nil && state != nil {
		groupName := ""
		groupType := "unknown"
		var groupStyle lipgloss.Style
		if scene.Group.Rtype != nil {
			groupType = string(*scene.Group.Rtype)
			switch *scene.Group.Rtype {
			case hueclient.ResourceIdentifierRtypeRoom:
				if room, ok := state.GetRoom(*scene.Group.Rid); ok {
					groupName = state.GetRoomName(room)
					groupStyle = lipgloss.NewStyle().Foreground(m.styles.Theme.EntityRoom)
				}
			case hueclient.ResourceIdentifierRtypeZone:
				if zone, ok := state.GetZone(*scene.Group.Rid); ok {
					groupName = zone.RoomName("Unknown")
					groupStyle = lipgloss.NewStyle().Foreground(m.styles.Theme.EntityZone)
				}
			}
		}
		if groupName != "" {
			rows = append(rows, gridlayout.NewStyledInfoRow("Group", groupStyle.Render(groupName), infoLabelWidth, lipgloss.NewStyle()))
		}
		rows = append(rows, gridlayout.NewStyledInfoRow("Group Type", groupType, infoLabelWidth, m.styles.Dimmed))
	}

	// Transition duration (convert from ms to seconds)
	if scene.TransitionDuration > 0 {
		durationSec := float64(scene.TransitionDuration) / 1000.0
		rows = append(rows, gridlayout.NewInfoRow("Transition", fmt.Sprintf("%.1fs", durationSec), infoLabelWidth))
	}

	// Active timeslot
	if scene.ActiveTimeslot != nil {
		slotInfo := fmt.Sprintf("Slot %d (%s)", scene.ActiveTimeslot.TimeslotId, string(scene.ActiveTimeslot.Weekday))
		rows = append(rows, gridlayout.NewInfoRow("Active Slot", slotInfo, infoLabelWidth))
	}

	// Week timeslots summary
	if len(scene.WeekTimeslots) > 0 {
		rows = append(rows, gridlayout.NewInfoRow("Days Configured", fmt.Sprintf("%d", len(scene.WeekTimeslots)), infoLabelWidth))
	}

	if scene.Id != nil {
		rows = append(rows, gridlayout.NewEmptyRow())
		rows = append(rows, gridlayout.NewInfoRow("ID", *scene.Id, infoLabelWidth))
	}

	return rows
}

// buildSmartScenesCategoryGridRows builds a list of all smart scenes in a category.
func (m *Model) buildSmartScenesCategoryGridRows(data panels.SmartScenesCategoryData, _ *hue.BridgeState) []gridlayout.GridRow {
	var rows []gridlayout.GridRow

	// Header with parent context
	headerText := fmt.Sprintf("Smart Scenes %s%d%s", m.styles.Dimmed.Render("["), len(data.SmartScenes), m.styles.Dimmed.Render("]"))
	if data.ParentName != "" {
		headerText = fmt.Sprintf("Smart Scenes on %s %s%d%s", data.ParentName, m.styles.Dimmed.Render("["), len(data.SmartScenes), m.styles.Dimmed.Render("]"))
	}
	header := field.NewHeaderComponent("smartscenes-header", headerText, &m.styles, m.zones)
	rows = append(rows, gridlayout.GridRow{
		Type:    gridlayout.RowTypeSection,
		Section: header,
	})

	rows = append(rows, gridlayout.NewInfoRow("Total", fmt.Sprintf("%d", len(data.SmartScenes)), infoLabelWidth))
	rows = append(rows, gridlayout.NewEmptyRow())

	// List each smart scene with status
	smartSceneStyle := lipgloss.NewStyle().Foreground(m.styles.Theme.EntityScene)
	for _, scene := range data.SmartScenes {
		name := "Unknown"
		if scene.Metadata.Name != nil {
			name = *scene.Metadata.Name
		}
		statusIndicator := m.styles.Dimmed.Render("○") // inactive
		if scene.State == "active" {
			statusIndicator = m.styles.Success.Render("●") // active
		}
		rows = append(rows, gridlayout.NewListItemRow(statusIndicator, smartSceneStyle.Render(name)))
	}

	return rows
}

// buildEntertainmentCategoryGridRows builds a list of all entertainment configurations in a category.
func (m *Model) buildEntertainmentCategoryGridRows(data panels.EntertainmentCategoryData) []gridlayout.GridRow {
	var rows []gridlayout.GridRow

	// Header with count
	entHeaderText := fmt.Sprintf("Entertainment Areas %s%d%s", m.styles.Dimmed.Render("["), len(data.Configurations), m.styles.Dimmed.Render("]"))
	header := field.NewHeaderComponent("entertainment-header", entHeaderText, &m.styles, m.zones)
	rows = append(rows, gridlayout.GridRow{
		Type:    gridlayout.RowTypeSection,
		Section: header,
	})

	rows = append(rows, gridlayout.NewInfoRow("Total", fmt.Sprintf("%d", len(data.Configurations)), infoLabelWidth))
	rows = append(rows, gridlayout.NewEmptyRow())

	// List each entertainment configuration
	entStyle := lipgloss.NewStyle().Foreground(m.styles.Theme.EntityEntertainment)
	for _, cfg := range data.Configurations {
		rows = append(rows, gridlayout.NewListItemRow("•", entStyle.Render(cfg.Name)))
	}

	return rows
}
