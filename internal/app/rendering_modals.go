package app

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss/v2"
	"github.com/kluzzebass/lazyhue/internal/ui/panels"
)

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
