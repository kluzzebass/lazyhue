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
	"github.com/kluzzebass/lazyhue/internal/ui2/components"
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

// buildHelpContent builds the help content for the detail panel.
func (m *Model) buildHelpContent() string {
	var content strings.Builder

	// Header
	content.WriteString(m.styles.Title.Render("Keyboard Shortcuts"))
	content.WriteString("\n\n")

	// Navigation
	content.WriteString(m.styles.Subtitle.Render("Navigation:"))
	content.WriteString("\n")
	content.WriteString("  ↑/k, ↓/j     Move up/down\n")
	content.WriteString("  ←/h, →/l     Collapse/expand (tree) or prev/next tab\n")
	content.WriteString("  PgUp/PgDn    Page up/down\n")
	content.WriteString("  g/Home       Go to top\n")
	content.WriteString("  G/End        Go to bottom\n")
	content.WriteString("\n")

	// Panel navigation
	content.WriteString(m.styles.Subtitle.Render("Panels:"))
	content.WriteString("\n")
	content.WriteString("  Tab          Next panel\n")
	content.WriteString("  Shift+Tab    Previous panel\n")
	content.WriteString("  0, 1, 2      Jump to panel by number\n")
	content.WriteString("  Esc          Back / close\n")
	content.WriteString("\n")

	// Bridge navigation
	content.WriteString(m.styles.Subtitle.Render("Bridges:"))
	content.WriteString("\n")
	content.WriteString("  [            Previous bridge tab\n")
	content.WriteString("  ]            Next bridge tab\n")
	content.WriteString("\n")

	// Actions
	content.WriteString(m.styles.Subtitle.Render("Actions:"))
	content.WriteString("\n")
	content.WriteString("  Enter        Select / expand\n")
	content.WriteString("  Space        Toggle on/off\n")
	content.WriteString("  r            Rename selected item\n")
	content.WriteString("  +/-          Brightness up/down\n")
	content.WriteString("  o/O          Turn on/off\n")
	content.WriteString("\n")

	// UI
	content.WriteString(m.styles.Subtitle.Render("UI:"))
	content.WriteString("\n")
	content.WriteString("  ?            Toggle this help\n")
	content.WriteString("  a            Toggle activity log\n")
	content.WriteString("  q            Quit\n")
	content.WriteString("\n")

	// Footer
	content.WriteString(m.styles.Dimmed.Render("Press Esc or ? to close"))
	content.WriteString("\n")

	return content.String()
}

// buildRenameContent builds the content for the rename input view.
func (m *Model) buildRenameContent() string {
	var content strings.Builder

	// Header
	content.WriteString(m.styles.Subtitle.Render(fmt.Sprintf("Rename %s", m.renameEntityType.String())))
	content.WriteString("\n\n")

	// Original name
	content.WriteString(fmt.Sprintf("  Current: %s\n\n", m.renameOriginalName))

	// Text input
	content.WriteString("  ")
	content.WriteString(m.renameInput.View())
	content.WriteString("\n\n")

	// Instructions
	content.WriteString(m.styles.Dimmed.Render("  Enter to save, Esc to cancel"))
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

	// If in rename mode, show rename input
	if m.renaming {
		m.detailViewport.SetContent(m.buildRenameContent())
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
			// Get the actual light ID from the light object (use *light.Id, not node.Item.ID)
			actualLightID := ""
			if light.Id != nil {
				actualLightID = *light.Id
			} else {
				// Fallback to node.Item.ID if light.Id is nil (shouldn't happen)
				actualLightID = node.Item.ID
			}

			// Only rebuild form fields if light ID changed or form is empty
			// But never rebuild while dropdown is open (user is actively interacting)
			if !m.lightForm.DropdownOpen && (m.selectedLightID != actualLightID || len(m.lightForm.Fields) == 0) {
				// Build control fields
				controlFields := m.buildLightFormFields(light)

				// Add Controls header
				allFields := []components.FormField{
					{
						Type:  components.FormFieldHeader,
						Label: "Controls",
					},
				}
				allFields = append(allFields, controlFields...)

				// Build and append detail fields
				detailFields := m.buildLightDetailFormFields(light, state)
				allFields = append(allFields, detailFields...)

				// Calculate max label width across all fields
				maxLabelWidth := 0
				for _, field := range allFields {
					if len(field.Label) > maxLabelWidth {
						maxLabelWidth = len(field.Label)
					}
				}
				m.lightForm.MaxLabelWidth = maxLabelWidth
				m.lightForm.SectionHeader = "" // No section header since we now use FormFieldHeader

				m.lightForm.SetFields(allFields)
				m.selectedLightID = actualLightID

				// Capture bridgeID for callback closure
				callbackBridgeID := bridgeID
				// Set onChange callback
				// #region agent log
				debugLog("rendering.go:78", "setting OnChange callback", map[string]interface{}{
					"nodeItemID": node.Item.ID, "actualLightID": actualLightID, "bridgeID": bridgeID, "lightId": func() string {
						if light.Id != nil {
							return *light.Id
						}
						return "nil"
					}(),
					"hypothesisId": "F",
				})
				// #endregion
				m.lightForm.OnChange = func(field components.FormField) {
					m.handleLightFieldChange(field, actualLightID, callbackBridgeID)
				}
				m.lightForm.OnLinkClick = func(entityType, entityID string) {
					m.navigateToEntity(entityType, entityID, callbackBridgeID)
				}
			} else if !m.lightForm.Editing && m.lightForm.MouseCaptureIdx < 0 && !m.lightForm.DropdownOpen {
				// Light ID unchanged, not editing, not dragging, and dropdown not open - sync form fields from state
				// This prevents overwriting user input during color wheel adjustments and slider dragging
				// Also prevents closing dropdown while user is selecting an option

				// Build control fields
				controlFields := m.buildLightFormFields(light)

				// Add Controls header
				allFields := []components.FormField{
					{
						Type:  components.FormFieldHeader,
						Label: "Controls",
					},
				}
				allFields = append(allFields, controlFields...)

				// Build and append detail fields
				detailFields := m.buildLightDetailFormFields(light, state)
				allFields = append(allFields, detailFields...)

				// Calculate max label width across all fields
				maxLabelWidth := 0
				for _, field := range allFields {
					if len(field.Label) > maxLabelWidth {
						maxLabelWidth = len(field.Label)
					}
				}
				m.lightForm.MaxLabelWidth = maxLabelWidth
				m.lightForm.SectionHeader = "" // No section header since we now use FormFieldHeader

				// Preserve cursor position
				oldCursor := m.lightForm.Cursor
				m.lightForm.SetFields(allFields)
				if oldCursor < len(allFields) {
					m.lightForm.Cursor = oldCursor
				}

				// Get the actual light ID from the light object (use *light.Id, not node.Item.ID)
				actualLightID := ""
				if light.Id != nil {
					actualLightID = *light.Id
				} else {
					// Fallback to node.Item.ID if light.Id is nil (shouldn't happen)
					actualLightID = node.Item.ID
				}
				// Capture bridgeID for callback closure
				callbackBridgeID := bridgeID
				// Ensure callback is set
				m.lightForm.OnChange = func(field components.FormField) {
					m.handleLightFieldChange(field, actualLightID, callbackBridgeID)
				}
				m.lightForm.OnLinkClick = func(entityType, entityID string) {
					m.navigateToEntity(entityType, entityID, callbackBridgeID)
				}
			}

			// Render the unified form with both controls and details
			if len(m.lightForm.Fields) > 0 {
				formContent := m.lightForm.View()
				// Zones are marked in form content - they'll be scanned from final output
				content.WriteString(formContent)
			}
		}
	default:
		// Clear form for non-light entities
		m.lightForm.SetFields([]components.FormField{})
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

// buildLightFormFields builds form fields for controlling a light.
func (m *Model) buildLightFormFields(light hueclient.LightGet) []components.FormField {
	var fields []components.FormField

	// On/Off toggle
	onValue := 0
	if light.On != nil && light.On.On != nil && *light.On.On {
		onValue = 1
	}
	fields = append(fields, components.FormField{
		ID:             "on",
		Label:          "Power",
		Type:           components.FormFieldToggle,
		Value:          onValue,
		ToggleOnLabel:  "On",
		ToggleOffLabel: "Off",
	})

	// Brightness (if dimmable)
	if light.Dimming != nil && light.Dimming.Brightness != nil {
		brightness := int(*light.Dimming.Brightness)
		fields = append(fields, components.FormField{
			ID:    "brightness",
			Label: "Brightness",
			Type:  components.FormFieldBrightness,
			Value: brightness,
			Min:   0,
			Max:   100,
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
		fields = append(fields, components.FormField{
			ID:    "colortemp",
			Label: "Color Temp",
			Type:  components.FormFieldColorTemp,
			Value: mirek,
			Min:   minMirek,
			Max:   maxMirek,
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
		fields = append(fields, components.FormField{
			ID:     "color",
			Label:  "Color",
			Type:   components.FormFieldColor,
			ColorX: x,
			ColorY: y,
		})
	}

	// Effect (if supported)
	if light.Effects != nil && light.Effects.EffectValues != nil {
		var options []components.FormSelectOption
		currentEffect := -1
		if light.Effects.Status != nil {
			for i, effect := range *light.Effects.EffectValues {
				effectStr := string(effect)
				displayName := hue.EffectDisplayName(effectStr)
				options = append(options, components.FormSelectOption{
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
			fields = append(fields, components.FormField{
				ID:      "effect",
				Label:   "Effect",
				Type:    components.FormFieldSelect,
				Value:   currentEffect,
				Options: options,
			})
		}
	}

	return fields
}

// buildLightDetailFormFields builds form fields for all static detail sections.
// These are read-only fields that display light information (not controls).
func (m *Model) buildLightDetailFormFields(light hueclient.LightGet, state *hue.BridgeState) []components.FormField {
	var fields []components.FormField

	// Get owning device info for product details
	var device *hueclient.DeviceGet
	if state != nil && light.Owner != nil && light.Owner.Rid != nil {
		if d, ok := state.GetDevice(*light.Owner.Rid); ok {
			device = &d
		}
	}

	// Product section
	var productFields []components.FormField
	if device != nil && device.ProductData != nil {
		pd := device.ProductData
		if pd.ProductName != nil || pd.ManufacturerName != nil || pd.ModelId != nil || pd.SoftwareVersion != nil || pd.HardwarePlatformType != nil {
			productFields = append(productFields, components.FormField{
				Type:  components.FormFieldHeader,
				Label: "Product",
			})
			if pd.ProductName != nil {
				productFields = append(productFields, components.FormField{
					ID:        "product",
					Label:     "Product",
					Type:      components.FormFieldText,
					TextValue: *pd.ProductName,
					ReadOnly:  true,
				})
			}
			if pd.ManufacturerName != nil {
				productFields = append(productFields, components.FormField{
					ID:        "manufacturer",
					Label:     "Manufacturer",
					Type:      components.FormFieldText,
					TextValue: *pd.ManufacturerName,
					ReadOnly:  true,
				})
			}
			if pd.ModelId != nil {
				productFields = append(productFields, components.FormField{
					ID:        "model",
					Label:     "Model",
					Type:      components.FormFieldText,
					TextValue: *pd.ModelId,
					ReadOnly:  true,
				})
			}
			if pd.SoftwareVersion != nil {
				productFields = append(productFields, components.FormField{
					ID:        "firmware",
					Label:     "Firmware",
					Type:      components.FormFieldText,
					TextValue: *pd.SoftwareVersion,
					ReadOnly:  true,
				})
			}
			if pd.HardwarePlatformType != nil {
				productFields = append(productFields, components.FormField{
					ID:        "hardware",
					Label:     "Hardware",
					Type:      components.FormFieldText,
					TextValue: *pd.HardwarePlatformType,
					ReadOnly:  true,
				})
			}
		}
	}
	fields = append(fields, productFields...)

	// Classification section
	var classFields []components.FormField
	hasClassSection := false
	if device != nil && device.ProductData != nil && device.ProductData.ProductArchetype != nil {
		hasClassSection = true
	}
	if light.Type != nil {
		hasClassSection = true
	}
	if light.Mode != nil {
		hasClassSection = true
	}
	if hasClassSection {
		classFields = append(classFields, components.FormField{
			Type:  components.FormFieldHeader,
			Label: "Classification",
		})
		// Check if device has archetype in metadata (user-changeable) or product data (manufacturer default)
		var currentArchetypePtr *hueclient.ProductArchetype
		if device != nil && device.Metadata != nil && device.Metadata.Archetype != nil {
			// Prefer metadata archetype (user-changeable)
			currentArchetypePtr = device.Metadata.Archetype
		} else if device != nil && device.ProductData != nil && device.ProductData.ProductArchetype != nil {
			// Fall back to product data archetype (manufacturer default)
			currentArchetypePtr = device.ProductData.ProductArchetype
		}

		if currentArchetypePtr != nil {
			// Build archetype options from complete list
			archetypeKeys := getSortedProductArchetypes()
			archetypeOptions := make([]components.FormSelectOption, len(archetypeKeys))
			for i, key := range archetypeKeys {
				archetypeOptions[i] = components.FormSelectOption{
					Label: hue.ProductArchetypeDisplayName(key),
					Value: i,
				}
			}

			// Find current archetype index
			currentArchetype := string(*currentArchetypePtr)
			currentIndex := 0 // default to first option
			for i, key := range archetypeKeys {
				if key == currentArchetype {
					currentIndex = i
					break
				}
			}

			// Store device ID for updates
			deviceID := ""
			if light.Owner != nil && light.Owner.Rid != nil {
				deviceID = *light.Owner.Rid
			}

			classFields = append(classFields, components.FormField{
				ID:       "archetype:" + deviceID,
				Label:    "Archetype",
				Type:     components.FormFieldSelect,
				Value:    currentIndex,
				Options:  archetypeOptions,
				ReadOnly: false,
			})
		}
		if light.Type != nil {
			classFields = append(classFields, components.FormField{
				ID:        "type",
				Label:     "Type",
				Type:      components.FormFieldText,
				TextValue: string(*light.Type),
				ReadOnly:  true,
			})
		}
		if light.Mode != nil {
			classFields = append(classFields, components.FormField{
				ID:        "mode",
				Label:     "Mode",
				Type:      components.FormFieldText,
				TextValue: string(*light.Mode),
				ReadOnly:  true,
			})
		}
	}
	fields = append(fields, classFields...)

	// Name section
	var nameFields []components.FormField
	currentName := ""
	hasNameSection := false
	if device != nil && device.Metadata != nil && device.Metadata.Name != nil {
		currentName = *device.Metadata.Name
		hasNameSection = true
	}
	if light.Metadata != nil && light.Metadata.Name != nil {
		alternateName := *light.Metadata.Name
		if alternateName != currentName {
			hasNameSection = true
		}
	}
	if hasNameSection {
		nameFields = append(nameFields, components.FormField{
			Type:  components.FormFieldHeader,
			Label: "Name",
		})
		if currentName != "" && device != nil {
			// Store device ID in a custom field for onChange handling
			deviceID := ""
			if light.Owner != nil && light.Owner.Rid != nil {
				deviceID = *light.Owner.Rid
			}
			nameFields = append(nameFields, components.FormField{
				ID:        "name:" + deviceID, // Include device ID for updates
				Label:     "Name",
				Type:      components.FormFieldText,
				TextValue: currentName,
				ReadOnly:  false,
			})
		}
		if light.Metadata != nil && light.Metadata.Name != nil {
			alternateName := *light.Metadata.Name
			if alternateName != currentName {
				nameFields = append(nameFields, components.FormField{
					ID:        "alternate-name",
					Label:     "Alternate name",
					Type:      components.FormFieldText,
					TextValue: alternateName,
					ReadOnly:  true,
				})
			}
		}
	}
	fields = append(fields, nameFields...)

	// IDs section
	var idFields []components.FormField
	hasIDSection := false
	if light.Id != nil {
		hasIDSection = true
	}
	if light.Owner != nil && light.Owner.Rid != nil {
		hasIDSection = true
	}
	if light.IdV1 != nil {
		hasIDSection = true
	}
	if hasIDSection {
		idFields = append(idFields, components.FormField{
			Type:  components.FormFieldHeader,
			Label: "IDs",
		})
		if light.Id != nil {
			idFields = append(idFields, components.FormField{
				ID:        "light-id",
				Label:     "Light ID",
				Type:      components.FormFieldText,
				TextValue: *light.Id,
				ReadOnly:  true,
			})
		}
		if light.Owner != nil && light.Owner.Rid != nil {
			deviceID := *light.Owner.Rid
			idFields = append(idFields, components.FormField{
				ID:             "device-id",
				Label:          "Device ID",
				Type:           components.FormFieldText,
				TextValue:      deviceID,
				ReadOnly:       true,
				IsLink:         true,
				LinkEntityType: "device",
				LinkEntityID:   deviceID,
			})
		}
		if light.IdV1 != nil {
			idFields = append(idFields, components.FormField{
				ID:        "v1-id",
				Label:     "V1 ID",
				Type:      components.FormFieldText,
				TextValue: *light.IdV1,
				ReadOnly:  true,
			})
		}
	}
	fields = append(fields, idFields...)

	// Technical info section (capabilities and limits, not current state)
	var techFields []components.FormField
	hasTechInfo := false

	// Min dim level (capability limit)
	if light.Dimming != nil && light.Dimming.MinDimLevel != nil {
		if !hasTechInfo {
			techFields = append(techFields, components.FormField{
				Type:  components.FormFieldHeader,
				Label: "Technical",
			})
			hasTechInfo = true
		}
		techFields = append(techFields, components.FormField{
			ID:        "min-dim-level",
			Label:     "Min dim level",
			Type:      components.FormFieldText,
			TextValue: fmt.Sprintf("%.0f%%", float64(*light.Dimming.MinDimLevel)),
			ReadOnly:  true,
		})
	}

	// Gamut type (color capability info)
	if light.Color != nil && light.Color.GamutType != nil {
		if !hasTechInfo {
			techFields = append(techFields, components.FormField{
				Type:  components.FormFieldHeader,
				Label: "Technical",
			})
			hasTechInfo = true
		}
		techFields = append(techFields, components.FormField{
			ID:        "gamut",
			Label:     "Gamut",
			Type:      components.FormFieldText,
			TextValue: string(*light.Color.GamutType),
			ReadOnly:  true,
		})
	}

	// Color temperature range (capability limits)
	if light.ColorTemperature != nil && light.ColorTemperature.MirekSchema != nil {
		schema := light.ColorTemperature.MirekSchema
		if schema.MirekMinimum != nil && schema.MirekMaximum != nil {
			if !hasTechInfo {
				techFields = append(techFields, components.FormField{
					Type:  components.FormFieldHeader,
					Label: "Technical",
				})
				hasTechInfo = true
			}
			minK := 1000000 / int(*schema.MirekMaximum)
			maxK := 1000000 / int(*schema.MirekMinimum)
			techFields = append(techFields, components.FormField{
				ID:        "ct-range",
				Label:     "CT range",
				Type:      components.FormFieldText,
				TextValue: fmt.Sprintf("%dK - %dK", minK, maxK),
				ReadOnly:  true,
			})
		}
	}
	fields = append(fields, techFields...)

	// Dynamics section (active when dynamic scenes are running)
	var dynamicsFields []components.FormField
	if light.Dynamics != nil && (light.Dynamics.Status != nil || light.Dynamics.Speed != nil) {
		dynamicsFields = append(dynamicsFields, components.FormField{
			Type:  components.FormFieldHeader,
			Label: "Dynamics",
		})
		if light.Dynamics.Status != nil {
			dynamicsFields = append(dynamicsFields, components.FormField{
				ID:        "dynamics-status",
				Label:     "Status",
				Type:      components.FormFieldText,
				TextValue: string(*light.Dynamics.Status),
				ReadOnly:  true,
			})
		}
		if light.Dynamics.Speed != nil {
			dynamicsFields = append(dynamicsFields, components.FormField{
				ID:        "dynamics-speed",
				Label:     "Speed",
				Type:      components.FormFieldText,
				TextValue: fmt.Sprintf("%.2f", *light.Dynamics.Speed),
				ReadOnly:  true,
			})
		}
	}
	fields = append(fields, dynamicsFields...)

	// Capabilities section (multiline bullet list)
	var caps []string
	if light.Dimming != nil {
		caps = append(caps, "• Dimming")
	}
	if light.Color != nil {
		caps = append(caps, "• Color")
	}
	if light.ColorTemperature != nil {
		caps = append(caps, "• Color Temperature")
	}
	if light.Gradient != nil {
		caps = append(caps, "• Gradient")
	}
	if light.Effects != nil {
		caps = append(caps, "• Effects")
	}
	if light.TimedEffects != nil {
		caps = append(caps, "• Timed Effects")
	}
	capText := ""
	if len(caps) > 0 {
		capText = strings.Join(caps, "\n  ")
	} else {
		capText = lipgloss.NewStyle().Foreground(m.styles.Theme.TextMuted).Render("On/Off only")
	}
	fields = append(fields, components.FormField{
		Type:  components.FormFieldHeader,
		Label: "Capabilities",
	})
	// For bullet lists, we add them as separate fields each with indentation
	for _, cap := range caps {
		fields = append(fields, components.FormField{
			ID:        "capabilities",
			Label:     "",
			Type:      components.FormFieldText,
			TextValue: cap,
			ReadOnly:  true,
		})
	}
	if len(caps) == 0 {
		fields = append(fields, components.FormField{
			ID:        "capabilities",
			Label:     "",
			Type:      components.FormFieldText,
			TextValue: capText,
			ReadOnly:  true,
		})
	}

	// Available Effects section (multiline bullet list)
	if light.Effects != nil && light.Effects.EffectValues != nil && len(*light.Effects.EffectValues) > 0 {
		fields = append(fields, components.FormField{
			Type:  components.FormFieldHeader,
			Label: "Available Effects",
		})
		for _, effect := range *light.Effects.EffectValues {
			fields = append(fields, components.FormField{
				ID:        "available-effects",
				Label:     "",
				Type:      components.FormFieldText,
				TextValue: fmt.Sprintf("• %s", hue.EffectDisplayName(string(effect))),
				ReadOnly:  true,
			})
		}
	}

	// Gradient section
	if light.Gradient != nil {
		hasGradientFields := false
		if light.Gradient.Mode != nil || light.Gradient.PixelCount != nil || light.Gradient.Points != nil {
			hasGradientFields = true
		}
		if hasGradientFields {
			fields = append(fields, components.FormField{
				Type:  components.FormFieldHeader,
				Label: "Gradient",
			})
			if light.Gradient.Mode != nil {
				// Mode field shown as muted in original - use TextMuted style
				modeText := lipgloss.NewStyle().Foreground(m.styles.Theme.TextMuted).Render(string(*light.Gradient.Mode))
				fields = append(fields, components.FormField{
					ID:        "gradient-mode",
					Label:     "Mode",
					Type:      components.FormFieldText,
					TextValue: modeText,
					ReadOnly:  true,
				})
			}
			if light.Gradient.PixelCount != nil {
				fields = append(fields, components.FormField{
					ID:        "gradient-pixels",
					Label:     "Pixels",
					Type:      components.FormFieldText,
					TextValue: fmt.Sprintf("%d", *light.Gradient.PixelCount),
					ReadOnly:  true,
				})
			}
			if light.Gradient.Points != nil && len(*light.Gradient.Points) > 0 {
				fields = append(fields, components.FormField{
					ID:        "gradient-points",
					Label:     "Points",
					Type:      components.FormFieldText,
					TextValue: fmt.Sprintf("%d", len(*light.Gradient.Points)),
					ReadOnly:  true,
				})
			}
		}
	}

	// Signaling Modes section (multiline bullet list)
	if light.Signaling != nil && light.Signaling.SignalValues != nil && len(*light.Signaling.SignalValues) > 0 {
		fields = append(fields, components.FormField{
			Type:  components.FormFieldHeader,
			Label: "Signaling Modes",
		})
		for _, sig := range *light.Signaling.SignalValues {
			fields = append(fields, components.FormField{
				ID:        "signaling-modes",
				Label:     "",
				Type:      components.FormFieldText,
				TextValue: fmt.Sprintf("• %s", hue.SignalingModeDisplayName(string(sig))),
				ReadOnly:  true,
			})
		}
	}

	// Power-on Behavior section
	if light.Powerup != nil && light.Powerup.Preset != nil {
		fields = append(fields, components.FormField{
			Type:  components.FormFieldHeader,
			Label: "Power-on Behavior",
		})

		// Build preset options from complete list
		presetKeys := getSortedPowerupPresets()
		presetOptions := make([]components.FormSelectOption, len(presetKeys))
		for i, key := range presetKeys {
			presetOptions[i] = components.FormSelectOption{
				Label: hue.PowerupPresetDisplayName(key),
				Value: i,
			}
		}

		// Find current preset index
		currentPreset := string(*light.Powerup.Preset)
		currentIndex := 0
		for i, key := range presetKeys {
			if key == currentPreset {
				currentIndex = i
				break
			}
		}

		// Get light ID for updates
		lightID := ""
		if light.Id != nil {
			lightID = *light.Id
		}

		fields = append(fields, components.FormField{
			ID:       "powerup-preset:" + lightID,
			Label:    "Preset",
			Type:     components.FormFieldSelect,
			Value:    currentIndex,
			Options:  presetOptions,
			ReadOnly: false,
		})
	}

	// Device Services section (multiline bullet list)
	if device != nil && device.Services != nil && len(*device.Services) > 0 {
		fields = append(fields, components.FormField{
			Type:  components.FormFieldHeader,
			Label: "Device Services",
		})
		for _, svc := range *device.Services {
			rtype := "unknown"
			if svc.Rtype != nil {
				rtype = hue.DeviceServiceDisplayName(string(*svc.Rtype))
			}
			var svcText string
			if svc.Rid != nil && light.Id != nil && *svc.Rid == *light.Id {
				svcText = fmt.Sprintf("• %s %s", rtype, lipgloss.NewStyle().Foreground(m.styles.Theme.TextMuted).Render("(this)"))
			} else {
				svcText = fmt.Sprintf("• %s", rtype)
			}
			fields = append(fields, components.FormField{
				ID:        "device-services",
				Label:     "",
				Type:      components.FormFieldText,
				TextValue: svcText,
				ReadOnly:  true,
			})
		}
	}

	return fields
}


// handleLightFieldChange handles changes to light form fields and updates the light via bridge actions.
func (m *Model) handleLightFieldChange(field components.FormField, lightID string, bridgeID string) {
	// #region agent log
	debugLog("rendering.go:793", "handleLightFieldChange entry", map[string]interface{}{
		"fieldID": field.ID, "fieldValue": field.Value, "lightID": lightID,
		"bridgeID":     bridgeID,
		"hypothesisId": "F",
	})
	// #endregion

	bridge := m.manager.GetBridge(bridgeID)
	if bridge == nil {
		// #region agent log
		debugLog("rendering.go:800", "bridge is nil", map[string]interface{}{
			"bridgeID":     bridgeID,
			"hypothesisId": "F",
		})
		// #endregion
		m.status = fmt.Sprintf("Bridge not found: %s", bridgeID)
		return
	}

	var err error
	// Handle fields with embedded IDs (name:deviceID, archetype:deviceID, powerup-preset:lightID)
	if strings.HasPrefix(field.ID, "name:") {
		deviceID := strings.TrimPrefix(field.ID, "name:")
		err = bridge.SetDeviceName(deviceID, field.TextValue)
	} else if strings.HasPrefix(field.ID, "archetype:") {
		deviceID := strings.TrimPrefix(field.ID, "archetype:")
		// Map index back to archetype string using the same sorted list
		archetypeKeys := getSortedProductArchetypes()
		if field.Value >= 0 && field.Value < len(archetypeKeys) {
			archetype := hueclient.ProductArchetype(archetypeKeys[field.Value])
			err = bridge.SetDeviceArchetype(deviceID, archetype)
		}
	} else if strings.HasPrefix(field.ID, "powerup-preset:") {
		lightIDFromField := strings.TrimPrefix(field.ID, "powerup-preset:")
		// Map index back to preset string using the same sorted list
		presetKeys := getSortedPowerupPresets()
		if field.Value >= 0 && field.Value < len(presetKeys) {
			preset := hueclient.PowerupPreset(presetKeys[field.Value])
			err = bridge.SetLightPowerupPreset(lightIDFromField, preset)
		}
	} else {
		// Handle regular fields
		switch field.ID {
		case "on":
			newOn := field.Value != 0
			err = bridge.SetLightOn(lightID, newOn)
		case "brightness":
			err = bridge.SetLightBrightness(lightID, float64(field.Value))
		case "colortemp":
			err = bridge.SetLightColorTemperature(lightID, field.Value)
		case "color":
			err = bridge.SetLightColor(lightID, field.ColorX, field.ColorY)
		case "effect":
			// Get the effect from the options based on the Value (index)
			if state := bridge.GetState(); state != nil {
				if light, ok := state.GetLight(lightID); ok {
					if light.Effects != nil && light.Effects.EffectValues != nil {
						effects := *light.Effects.EffectValues
						if field.Value >= 0 && field.Value < len(effects) {
							effect := effects[field.Value]
							err = bridge.SetLightEffect(lightID, effect)
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
