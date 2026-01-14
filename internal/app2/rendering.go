package app2

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"image/color"

	"github.com/charmbracelet/lipgloss/v2"
	"github.com/kluzzebass/lazyhue/internal/hue"
	"github.com/kluzzebass/lazyhue/internal/hueclient"
	"github.com/kluzzebass/lazyhue/internal/ui2"
	"github.com/kluzzebass/lazyhue/internal/ui2/component/field"
	gridlayout "github.com/kluzzebass/lazyhue/internal/ui2/component/layout"
	"github.com/kluzzebass/lazyhue/internal/ui2/panels"
)

// #region agent log
func debugLog(location, message string, data map[string]interface{}) {
	logData := map[string]interface{}{
		"sessionId": "debug-session",
		"runId":     "run1",
		"location":  location,
		"message":   message,
		"data":      data,
		"timestamp": time.Now().UnixMilli(),
	}
	if jsonData, err := json.Marshal(logData); err == nil {
		if f, err := os.OpenFile("/Users/kluzz/Code/lazyhue/.cursor/debug.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644); err == nil {
			f.WriteString(string(jsonData) + "\n")
			f.Close()
		}
	}
}

// #endregion

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

	for i, activity := range m.activities {
		line := activity.Render(&m.styles, contentWidth)
		content.WriteString(line)
		if i < len(m.activities)-1 {
			content.WriteString("\n")
		}
	}
	m.logViewport.SetContent(content.String())
	// Scroll to bottom
	m.logViewport.LineDown(len(m.activities))
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
	default:
		// Clear grid for non-light entities
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

	topBorder := m.renderPanelHeader(width, key, title, focused, borderColor)

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
	rightBorder := borderStyleColor.Render(border.Right)
	bottomBorder := borderStyleColor.Render(border.BottomLeft) +
		borderStyleColor.Render(strings.Repeat(border.Bottom, innerWidth)) +
		borderStyleColor.Render(border.BottomRight)

	var lines []string
	lines = append(lines, topBorder)

	for _, line := range contentLines {
		if lipgloss.Width(line) > innerWidth {
			line = lipgloss.Place(innerWidth, 1, lipgloss.Left, lipgloss.Top, line)
		}
		paddedLine := lipgloss.Place(innerWidth, 1, lipgloss.Left, lipgloss.Top, line)
		lines = append(lines, leftBorder+paddedLine+rightBorder)
	}

	// Fill to exact height (1 for top border + content + 1 for bottom border)
	targetHeight := height
	for len(lines) < targetHeight-1 {
		paddedLine := strings.Repeat(" ", innerWidth)
		lines = append(lines, leftBorder+paddedLine+rightBorder)
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

	topBorder := m.renderPanelHeader(width, key, "Activity", focused, borderColor)

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
	rightBorder := borderStyleColor.Render(border.Right)
	bottomBorder := borderStyleColor.Render(border.BottomLeft) +
		borderStyleColor.Render(strings.Repeat(border.Bottom, innerWidth)) +
		borderStyleColor.Render(border.BottomRight)

	var lines []string
	lines = append(lines, topBorder)

	// Render exactly the lines the viewport provides (should be innerHeight)
	// Limit to innerHeight to prevent overflow
	// Activity.Render() already handles truncation, so just pad to width
	for i := 0; i < len(contentLines) && i < innerHeight; i++ {
		line := contentLines[i]
		paddedLine := lipgloss.Place(innerWidth, 1, lipgloss.Left, lipgloss.Top, line)
		lines = append(lines, leftBorder+paddedLine+rightBorder)
	}

	// Fill to exact height (1 for top border + content + 1 for bottom border)
	targetHeight := height
	for len(lines) < targetHeight-1 {
		paddedLine := strings.Repeat(" ", innerWidth)
		lines = append(lines, leftBorder+paddedLine+rightBorder)
	}

	lines = append(lines, bottomBorder)

	return strings.Join(lines, "\n")
}

// renderPanelHeader renders a panel header with hotkey and title.
func (m *Model) renderPanelHeader(width int, keyStr, title string, focused bool, borderColor color.Color) string {
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
func (m *Model) buildLightGridRows(light hueclient.LightGet) {
	var rows []gridlayout.GridRow

	// Get owning device for product info
	device := m.getDeviceForLight(light)

	// 1. Controls section (editable) - most important, at the top
	rows = append(rows, m.buildControlsRows(light)...)

	// 2. Power-on Behavior section (editable)
	rows = append(rows, m.buildPowerupRows(light)...)

	// 3. State Info section
	rows = append(rows, m.buildStateRows(light)...)

	// 4. Dynamics section
	rows = append(rows, m.buildDynamicsRows(light)...)

	// 5. Product Info section
	rows = append(rows, m.buildProductInfoRows(device)...)

	// 6. Classification section
	rows = append(rows, m.buildClassificationRows(light, device)...)

	// 7. Name section (editable)
	rows = append(rows, m.buildNameRows(light, device)...)

	// 8. IDs section
	rows = append(rows, m.buildIDsRows(light)...)

	// 9. Capabilities section
	rows = append(rows, m.buildCapabilitiesRows(light)...)

	// 10. Effects list section
	rows = append(rows, m.buildEffectsListRows(light)...)

	// 11. Gradient section
	rows = append(rows, m.buildGradientRows(light)...)

	// 12. Signaling section
	rows = append(rows, m.buildSignalingRows(light)...)

	// 13. Device Services section
	rows = append(rows, m.buildDeviceServicesRows(light, device)...)

	m.lightGrid.SetRows(rows)
}

// handleNewFieldChange handles field change messages from the new form component.
// This is the message-based equivalent of handleLightFieldChange.
func (m *Model) handleNewFieldChange(msg field.FieldChangedMsg) {
	// Get the currently selected light and bridge
	node := m.tree.SelectedNode()
	if node == nil || node.Item.Type != panels.EntityLight {
		return
	}

	bridgeID := node.Item.BridgeID
	lightID := m.selectedLightID
	if lightID == "" {
		return
	}

	bridge := m.manager.GetBridge(bridgeID)
	if bridge == nil {
		m.status = fmt.Sprintf("Bridge not found: %s", bridgeID)
		return
	}

	var err error

	// Handle fields with embedded IDs (name:deviceID, archetype:deviceID, powerup-preset:lightID)
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
		return m.styles.Dimmed.Render(ui2.BrightnessIndicator(0))
	}

	brightness := 100.0
	if light.Dimming != nil && light.Dimming.Brightness != nil {
		brightness = float64(*light.Dimming.Brightness)
	}

	hexColor := ui2.GetLightColor(light)
	return ui2.RenderBrightnessIndicatorFromHex(brightness, hexColor)
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

	// Archetype (editable if device available)
	if device != nil && device.ProductData != nil && device.ProductData.ProductArchetype != nil {
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

		deviceID := ""
		if device.Id != nil {
			deviceID = *device.Id
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

	// Type (read-only)
	if light.Type != nil {
		rows = append(rows, gridlayout.NewInfoRow("Type", string(*light.Type), infoLabelWidth))
	}

	// Mode (read-only)
	if light.Mode != nil {
		rows = append(rows, gridlayout.NewInfoRow("Mode", string(*light.Mode), infoLabelWidth))
	}

	return rows
}

// buildNameRows builds rows for the name section.
func (m *Model) buildNameRows(light hueclient.LightGet, device *hueclient.DeviceGet) []gridlayout.GridRow {
	var rows []gridlayout.GridRow

	// Section header
	header := field.NewHeaderComponent("name-header", "Name", &m.styles, m.zones)
	rows = append(rows, gridlayout.GridRow{
		Type:    gridlayout.RowTypeSection,
		Section: header,
	})

	// Device name (editable)
	if device != nil && device.Metadata != nil && device.Metadata.Name != nil {
		deviceID := ""
		if device.Id != nil {
			deviceID = *device.Id
		}

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

	// Show deprecated light name if different from device name
	if light.Metadata != nil && light.Metadata.Name != nil {
		deprecatedName := *light.Metadata.Name
		currentName := ""
		if device != nil && device.Metadata != nil && device.Metadata.Name != nil {
			currentName = *device.Metadata.Name
		}
		if deprecatedName != currentName {
			dimStyle := m.styles.Dimmed
			rows = append(rows, gridlayout.NewStyledInfoRow("Deprecated", deprecatedName, infoLabelWidth, dimStyle))
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

	return rows
}

// buildStateRows builds rows for the current state section.
func (m *Model) buildStateRows(light hueclient.LightGet) []gridlayout.GridRow {
	var rows []gridlayout.GridRow

	// Section header
	header := field.NewHeaderComponent("state-header", "State", &m.styles, m.zones)
	rows = append(rows, gridlayout.GridRow{
		Type:    gridlayout.RowTypeSection,
		Section: header,
	})

	// On/off status with indicator
	status := "off"
	isOn := light.On != nil && light.On.On != nil && *light.On.On
	if isOn {
		status = "on"
	}
	indicator := m.renderLightIndicator(light, isOn)
	rows = append(rows, gridlayout.NewInfoRow("Status", indicator+" "+status, infoLabelWidth))

	// Brightness
	if light.Dimming != nil {
		if light.Dimming.Brightness != nil {
			rows = append(rows, gridlayout.NewInfoRow("Brightness", fmt.Sprintf("%.0f%%", *light.Dimming.Brightness), infoLabelWidth))
		}
		if light.Dimming.MinDimLevel != nil {
			rows = append(rows, gridlayout.NewInfoRow("Min Dim", fmt.Sprintf("%.0f%%", *light.Dimming.MinDimLevel), infoLabelWidth))
		}
	}

	// Color
	if light.Color != nil && light.Color.Xy != nil {
		xy := light.Color.Xy
		if xy.X != nil && xy.Y != nil {
			rows = append(rows, gridlayout.NewInfoRow("Color XY", fmt.Sprintf("(%.4f, %.4f)", *xy.X, *xy.Y), infoLabelWidth))
		}
		if light.Color.GamutType != nil {
			rows = append(rows, gridlayout.NewInfoRow("Gamut", string(*light.Color.GamutType), infoLabelWidth))
		}
	}

	// Color temperature
	if light.ColorTemperature != nil {
		if light.ColorTemperature.Mirek != nil {
			mirek := *light.ColorTemperature.Mirek
			kelvin := 1000000 / int(mirek)
			rows = append(rows, gridlayout.NewInfoRow("Color Temp", fmt.Sprintf("%d mirek (~%dK)", mirek, kelvin), infoLabelWidth))
		}
		if light.ColorTemperature.MirekSchema != nil {
			schema := light.ColorTemperature.MirekSchema
			if schema.MirekMinimum != nil && schema.MirekMaximum != nil {
				minK := 1000000 / int(*schema.MirekMaximum)
				maxK := 1000000 / int(*schema.MirekMinimum)
				rows = append(rows, gridlayout.NewInfoRow("CT Range", fmt.Sprintf("%dK - %dK", minK, maxK), infoLabelWidth))
			}
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
func (m *Model) buildGradientRows(light hueclient.LightGet) []gridlayout.GridRow {
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

	if light.Gradient.Mode != nil {
		rows = append(rows, gridlayout.NewInfoRow("Mode", string(*light.Gradient.Mode), infoLabelWidth))
	}
	if light.Gradient.PixelCount != nil {
		rows = append(rows, gridlayout.NewInfoRow("Pixels", fmt.Sprintf("%d", *light.Gradient.PixelCount), infoLabelWidth))
	}
	if light.Gradient.Points != nil && len(*light.Gradient.Points) > 0 {
		rows = append(rows, gridlayout.NewInfoRow("Points", fmt.Sprintf("%d", len(*light.Gradient.Points)), infoLabelWidth))
	}

	return rows
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
