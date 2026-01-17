package app

import (
	"fmt"
	"sort"
	"strings"

	"image/color"

	"github.com/charmbracelet/lipgloss/v2"
	"github.com/kluzzebass/lazyhue/internal/hue"
	"github.com/kluzzebass/lazyhue/internal/hueclient"
	"github.com/kluzzebass/lazyhue/internal/ui"
	"github.com/kluzzebass/lazyhue/internal/ui/component/field"
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

// updateSceneColorModeFlags updates the Inactive flags on scene color controls
// without rebuilding the entire UI, preserving drag state.
// colorActive: true = color wheel is active, false = color temp is active
func (m *Model) updateSceneColorModeFlags(sceneID, lightID string, colorActive bool) {
	// Update color wheel: inactive when color temp is active
	colorWheelID := FieldIDSceneActionColor(sceneID, lightID)
	var colorX, colorY float64
	if comp := m.lightGrid.GetComponentByID(colorWheelID); comp != nil {
		if cw, ok := comp.(*field.ColorWheelComponent); ok {
			cw.Inactive = !colorActive
			colorX = cw.ColorX
			colorY = cw.ColorY
		}
	}

	// Update color temp slider: inactive when color is active
	colorTempID := FieldIDSceneActionColorTemp(sceneID, lightID)
	var colorTempMirek int
	if comp := m.lightGrid.GetComponentByID(colorTempID); comp != nil {
		if ct, ok := comp.(*field.ColorTempSliderComponent); ok {
			ct.Inactive = colorActive
			colorTempMirek = ct.Value
		}
	}

	// Update brightness slider color to match the new active mode
	brightnessID := FieldIDSceneActionBrightness(sceneID, lightID)
	if comp := m.lightGrid.GetComponentByID(brightnessID); comp != nil {
		if bs, ok := comp.(*field.BrightnessSliderComponent); ok {
			if colorActive {
				bs.ColorX = colorX
				bs.ColorY = colorY
				bs.ColorTempMirek = 0
			} else {
				bs.ColorX = 0
				bs.ColorY = 0
				bs.ColorTempMirek = colorTempMirek
			}
		}
	}
}

// updateSceneBrightness updates the brightness dimming on scene color controls
// without rebuilding the entire UI, allowing real-time dimming during brightness drag.
func (m *Model) updateSceneBrightness(sceneID, lightID string, brightness int) {
	// Update color wheel brightness
	colorWheelID := FieldIDSceneActionColor(sceneID, lightID)
	if comp := m.lightGrid.GetComponentByID(colorWheelID); comp != nil {
		if cw, ok := comp.(*field.ColorWheelComponent); ok {
			cw.Brightness = brightness
		}
	}

	// Update color temp slider brightness
	colorTempID := FieldIDSceneActionColorTemp(sceneID, lightID)
	if comp := m.lightGrid.GetComponentByID(colorTempID); comp != nil {
		if ct, ok := comp.(*field.ColorTempSliderComponent); ok {
			ct.Brightness = brightness
		}
	}
}

// updateSceneBrightnessColor updates the brightness slider's color display
// when the color value changes (during drag operations).
func (m *Model) updateSceneBrightnessColor(sceneID, lightID string, colorX, colorY float64) {
	brightnessID := FieldIDSceneActionBrightness(sceneID, lightID)
	if comp := m.lightGrid.GetComponentByID(brightnessID); comp != nil {
		if bs, ok := comp.(*field.BrightnessSliderComponent); ok {
			bs.ColorX = colorX
			bs.ColorY = colorY
			bs.ColorTempMirek = 0
		}
	}
}

// updateSceneBrightnessColorTemp updates the brightness slider's color temp display
// when the color temp value changes (during drag operations).
func (m *Model) updateSceneBrightnessColorTemp(sceneID, lightID string, mirek int) {
	brightnessID := FieldIDSceneActionBrightness(sceneID, lightID)
	if comp := m.lightGrid.GetComponentByID(brightnessID); comp != nil {
		if bs, ok := comp.(*field.BrightnessSliderComponent); ok {
			bs.ColorX = 0
			bs.ColorY = 0
			bs.ColorTempMirek = mirek
		}
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
			actualLightID := light.Id
			if actualLightID == "" {
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
		var zone hueclient.ZoneGet
		var ok bool
		if node.Item.RawPtr != nil {
			if z, typeOk := node.Item.RawPtr.(hueclient.ZoneGet); typeOk {
				zone = z
				ok = true
			}
		}
		if !ok && state != nil {
			zone, ok = state.GetZone(node.Item.ID)
		}
		if ok {
			// Convert ZoneGet to RoomGet-like structure for rendering
			room := hueclient.RoomGet{
				Children: zone.Children,
				Id:       zone.Id,
				IdV1:     zone.IdV1,
				Metadata: zone.Metadata,
				Services: zone.Services,
			}
			rows := m.buildRoomGridRows(room, true, state)
			m.lightGrid.SetRows(rows)
			if len(m.lightGrid.Children()) > 0 {
				content.WriteString(m.lightGrid.View())
			}
		}

	case panels.EntityScene:
		m.selectedLightID = ""
		var scene hueclient.SceneGet
		var ok bool
		// Always get fresh scene data from state (not RawPtr cache) because
		// optimistic updates modify the state directly for immediate UI feedback
		if state != nil {
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
	if strings.HasPrefix(msg.FieldID, FieldPrefixName) {
		deviceID := strings.TrimPrefix(msg.FieldID, FieldPrefixName)
		if v, ok := msg.Value.(field.TextValue); ok {
			err = bridge.SetDeviceName(deviceID, v.Text)
		}
	} else if strings.HasPrefix(msg.FieldID, FieldPrefixArchetype) {
		deviceID := strings.TrimPrefix(msg.FieldID, FieldPrefixArchetype)
		if v, ok := msg.Value.(field.SelectValue); ok {
			archetypeKeys := getSortedProductArchetypes()
			if v.Index >= 0 && v.Index < len(archetypeKeys) {
				archetype := hueclient.DeviceArchetype(archetypeKeys[v.Index])
				err = bridge.SetDeviceArchetype(deviceID, archetype)
			}
		}
	} else if strings.HasPrefix(msg.FieldID, FieldPrefixPowerupPreset) {
		lightIDFromField := strings.TrimPrefix(msg.FieldID, FieldPrefixPowerupPreset)
		if v, ok := msg.Value.(field.SelectValue); ok {
			presetKeys := getSortedPowerupPresets()
			if v.Index >= 0 && v.Index < len(presetKeys) {
				preset := hueclient.UpdateLightJSONBodyPowerupPreset(presetKeys[v.Index])
				err = bridge.SetLightPowerupPreset(lightIDFromField, preset)
			}
		}
	} else if strings.HasPrefix(msg.FieldID, FieldPrefixRoomArchetype) {
		roomID := strings.TrimPrefix(msg.FieldID, FieldPrefixRoomArchetype)
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
	} else if strings.HasPrefix(msg.FieldID, FieldPrefixZoneArchetype) {
		zoneID := strings.TrimPrefix(msg.FieldID, FieldPrefixZoneArchetype)
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
	} else if strings.HasPrefix(msg.FieldID, FieldPrefixRoomName) {
		roomID := strings.TrimPrefix(msg.FieldID, FieldPrefixRoomName)
		if v, ok := msg.Value.(field.TextValue); ok {
			m.status = fmt.Sprintf("Renaming room to \"%s\"", v.Text)
			err = bridge.RenameRoom(roomID, v.Text)
			// Don't call updateDetailContent - SSE event will refresh UI
			if err != nil {
				m.status = fmt.Sprintf("Error: %v", err)
			}
			return
		}
	} else if strings.HasPrefix(msg.FieldID, FieldPrefixZoneName) {
		zoneID := strings.TrimPrefix(msg.FieldID, FieldPrefixZoneName)
		if v, ok := msg.Value.(field.TextValue); ok {
			m.status = fmt.Sprintf("Renaming zone to \"%s\"", v.Text)
			err = bridge.RenameZone(zoneID, v.Text)
			// Don't call updateDetailContent - SSE event will refresh UI
			if err != nil {
				m.status = fmt.Sprintf("Error: %v", err)
			}
			return
		}
	} else if strings.HasPrefix(msg.FieldID, FieldPrefixDeviceRoom) || strings.HasPrefix(msg.FieldID, FieldPrefixLightRoom) {
		// Handle both device and light room assignment (both use device ID)
		deviceID := strings.TrimPrefix(msg.FieldID, FieldPrefixDeviceRoom)
		if strings.HasPrefix(msg.FieldID, FieldPrefixLightRoom) {
			deviceID = strings.TrimPrefix(msg.FieldID, FieldPrefixLightRoom)
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
					newRoomID = room.Id
					if room.Metadata.Name != "" {
						newRoomName = room.Metadata.Name
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
	} else if strings.HasPrefix(msg.FieldID, FieldPrefixLightZones) {
		// Handle zone membership checkbox - format is "light-zones:{lightID}"
		lightID := strings.TrimPrefix(msg.FieldID, FieldPrefixLightZones)
		if v, ok := msg.Value.(field.CheckboxValue); ok {
			bridgeState := bridge.GetState()
			if bridgeState != nil {
				allZones := bridgeState.AllZones()
				currentZones := bridgeState.GetLightZones(lightID)

				// Build current zone membership set
				currentZoneIDs := make(map[string]bool)
				for _, zone := range currentZones {
					if zone.Id != "" {
						currentZoneIDs[zone.Id] = true
					}
				}

				// Process changes
				var added, removed int
				for i, zone := range allZones {
					if zone.Id == "" {
						continue
					}
					zoneID := zone.Id
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
	} else if strings.HasPrefix(msg.FieldID, FieldPrefixRoomCreateScene) {
		roomID := strings.TrimPrefix(msg.FieldID, FieldPrefixRoomCreateScene)
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
	} else if strings.HasPrefix(msg.FieldID, FieldPrefixZoneCreateScene) {
		zoneID := strings.TrimPrefix(msg.FieldID, FieldPrefixZoneCreateScene)
		if _, ok := msg.Value.(field.ButtonValue); ok {
			// Get zone name for the scene name
			sceneName := "New Scene"
			bridgeState := bridge.GetState()
			if bridgeState != nil {
				if zone, ok := bridgeState.GetZone(zoneID); ok {
					zoneName := zone.ZoneName("Zone")
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
	} else if strings.HasPrefix(msg.FieldID, FieldPrefixSceneName) {
		sceneID := strings.TrimPrefix(msg.FieldID, FieldPrefixSceneName)
		if v, ok := msg.Value.(field.TextValue); ok {
			m.status = fmt.Sprintf("Renaming scene to \"%s\"", v.Text)
			err = bridge.RenameScene(sceneID, v.Text)
			// Don't call updateDetailContent - SSE event will refresh UI
			if err != nil {
				m.status = fmt.Sprintf("Error: %v", err)
			}
			return
		}
	} else if strings.HasPrefix(msg.FieldID, FieldPrefixBridgeName) {
		deviceID := strings.TrimPrefix(msg.FieldID, FieldPrefixBridgeName)
		if v, ok := msg.Value.(field.TextValue); ok {
			m.status = fmt.Sprintf("Renaming bridge to \"%s\"", v.Text)
			err = bridge.RenameDevice(deviceID, v.Text)
			// Don't call updateDetailContent - SSE event will refresh UI
			if err != nil {
				m.status = fmt.Sprintf("Error: %v", err)
			}
			return
		}
	} else if strings.HasPrefix(msg.FieldID, FieldPrefixIdentify) {
		// General identify handler for lights and devices
		deviceID := strings.TrimPrefix(msg.FieldID, FieldPrefixIdentify)
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
	} else if strings.HasPrefix(msg.FieldID, FieldPrefixBridgeIdentify) {
		// Bridge-specific identify handler
		deviceID := strings.TrimPrefix(msg.FieldID, FieldPrefixBridgeIdentify)
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
	} else if strings.HasPrefix(msg.FieldID, FieldPrefixEffectSpeed) {
		lightID := strings.TrimPrefix(msg.FieldID, FieldPrefixEffectSpeed)
		if v, ok := msg.Value.(field.SliderValue); ok {
			speed := float32(v.Value) / 100.0
			m.status = fmt.Sprintf("Effect speed: %d%%", v.Value)
			err = bridge.SetLightEffectSpeed(lightID, speed)
			if err != nil {
				m.status = fmt.Sprintf("Error: %v", err)
			}
			return
		}
	} else if strings.HasPrefix(msg.FieldID, FieldPrefixTimedEffectStop) {
		lightID := strings.TrimPrefix(msg.FieldID, FieldPrefixTimedEffectStop)
		if _, ok := msg.Value.(field.ButtonValue); ok {
			m.status = "Stopping timed effect..."
			err = bridge.SetLightTimedEffect(lightID, hueclient.SupportedTimedEffectsNoEffect, 0)
			if err != nil {
				m.status = fmt.Sprintf("Error: %v", err)
			} else {
				m.status = "Timed effect stopped"
			}
			return
		}
	} else if strings.HasPrefix(msg.FieldID, FieldPrefixTimedEffectDuration) {
		lightID := strings.TrimPrefix(msg.FieldID, FieldPrefixTimedEffectDuration)
		if v, ok := msg.Value.(field.SliderValue); ok {
			m.timedEffectDurations[lightID] = v.Value
			m.status = fmt.Sprintf("Timed effect duration: %d min", v.Value)
			return
		}
	} else if strings.HasPrefix(msg.FieldID, FieldPrefixTimedEffectTrigger) {
		// Format: "timed-effect-trigger:lightID:effect"
		rest := strings.TrimPrefix(msg.FieldID, FieldPrefixTimedEffectTrigger)
		parts := strings.SplitN(rest, ":", 2)
		if len(parts) == 2 {
			lightID := parts[0]
			effectStr := parts[1]
			if _, ok := msg.Value.(field.ButtonValue); ok {
				effect := hueclient.SupportedTimedEffects(effectStr)
				// Use stored duration or default to 30 minutes
				durationMin := 30
				if d, ok := m.timedEffectDurations[lightID]; ok {
					durationMin = d
				}
				durationMs := durationMin * 60 * 1000
				m.status = fmt.Sprintf("Starting %s effect (%d min)...", effectStr, durationMin)
				err = bridge.SetLightTimedEffect(lightID, effect, durationMs)
				if err != nil {
					m.status = fmt.Sprintf("Error: %v", err)
				} else {
					// Capitalize first letter manually
					effectDisplay := effectStr
					if len(effectStr) > 0 {
						effectDisplay = strings.ToUpper(effectStr[:1]) + effectStr[1:]
					}
					m.status = fmt.Sprintf("%s effect started", effectDisplay)
				}
				return
			}
		}
	} else if strings.HasPrefix(msg.FieldID, FieldPrefixSignalStop) {
		lightID := strings.TrimPrefix(msg.FieldID, FieldPrefixSignalStop)
		if _, ok := msg.Value.(field.ButtonValue); ok {
			m.status = "Stopping signal..."
			err = bridge.SetLightSignaling(lightID, hueclient.NoSignal, 0)
			if err != nil {
				m.status = fmt.Sprintf("Error: %v", err)
			} else {
				m.status = "Signal stopped"
			}
			return
		}
	} else if strings.HasPrefix(msg.FieldID, FieldPrefixSignalDuration) {
		lightID := strings.TrimPrefix(msg.FieldID, FieldPrefixSignalDuration)
		if v, ok := msg.Value.(field.SliderValue); ok {
			m.signalDurations[lightID] = v.Value
			m.status = fmt.Sprintf("Signal duration: %d sec", v.Value)
			return
		}
	} else if strings.HasPrefix(msg.FieldID, FieldPrefixSignalTrigger) {
		// Format: "signal-trigger:lightID:signal"
		rest := strings.TrimPrefix(msg.FieldID, FieldPrefixSignalTrigger)
		parts := strings.SplitN(rest, ":", 2)
		if len(parts) == 2 {
			lightID := parts[0]
			signalStr := parts[1]
			if _, ok := msg.Value.(field.ButtonValue); ok {
				signal := hueclient.SupportedSignals(signalStr)
				// Use stored duration or default to 15 seconds
				durationSec := 15
				if d, ok := m.signalDurations[lightID]; ok {
					durationSec = d
				}
				durationMs := durationSec * 1000
				displayName := hue.SignalingModeDisplayName(signalStr)
				m.status = fmt.Sprintf("Starting %s signal (%d sec)...", displayName, durationSec)
				err = bridge.SetLightSignaling(lightID, signal, durationMs)
				if err != nil {
					m.status = fmt.Sprintf("Error: %v", err)
				} else {
					m.status = fmt.Sprintf("%s signal started", displayName)
				}
				return
			}
		}
	} else if strings.HasPrefix(msg.FieldID, FieldPrefixDeviceName) {
		deviceID := strings.TrimPrefix(msg.FieldID, FieldPrefixDeviceName)
		if v, ok := msg.Value.(field.TextValue); ok {
			m.status = fmt.Sprintf("Renaming device to \"%s\"", v.Text)
			err = bridge.RenameDevice(deviceID, v.Text)
			// Don't call updateDetailContent - SSE event will refresh UI
			if err != nil {
				m.status = fmt.Sprintf("Error: %v", err)
			}
			return
		}
	} else if strings.HasPrefix(msg.FieldID, FieldPrefixSceneRecall) {
		sceneID := strings.TrimPrefix(msg.FieldID, FieldPrefixSceneRecall)
		if _, ok := msg.Value.(field.ButtonValue); ok {
			m.status = "Activating scene..."
			err = bridge.RecallScene(sceneID)
			if err == nil {
				m.status = "Scene activated"
			}
		}
	} else if strings.HasPrefix(msg.FieldID, FieldPrefixSmartSceneToggle) {
		sceneID := strings.TrimPrefix(msg.FieldID, FieldPrefixSmartSceneToggle)
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
	} else if strings.HasPrefix(msg.FieldID, FieldPrefixSceneActionOn) {
		// Scene action on/off toggle - format: "scene-action-on:<sceneID>:<lightID>"
		rest := strings.TrimPrefix(msg.FieldID, FieldPrefixSceneActionOn)
		parts := strings.SplitN(rest, ":", 2)
		if len(parts) == 2 {
			sceneID := parts[0]
			lightID := parts[1]
			if v, ok := msg.Value.(field.ToggleValue); ok {
				state := "off"
				if v.On {
					state = "on"
				}
				m.status = fmt.Sprintf("Setting scene light to %s...", state)
				err = bridge.UpdateSceneActionOn(sceneID, lightID, v.On)
				if err != nil {
					m.status = fmt.Sprintf("Error: %v", err)
				} else {
					m.status = "Scene updated"
				}
				return
			}
		}
	} else if strings.HasPrefix(msg.FieldID, FieldPrefixSceneActionBri) {
		// Scene action brightness - format: "scene-action-bri:<sceneID>:<lightID>"
		rest := strings.TrimPrefix(msg.FieldID, FieldPrefixSceneActionBri)
		parts := strings.SplitN(rest, ":", 2)
		if len(parts) == 2 {
			sceneID := parts[0]
			lightID := parts[1]
			if v, ok := msg.Value.(field.SliderValue); ok {
				m.status = fmt.Sprintf("Setting scene brightness to %d%%...", v.Value)
				err = bridge.UpdateSceneActionBrightness(sceneID, lightID, float32(v.Value))
				if err != nil {
					m.status = fmt.Sprintf("Error: %v", err)
				} else {
					m.status = "Scene updated"
					// Update brightness dimming on color controls in real-time
					m.updateSceneBrightness(sceneID, lightID, v.Value)
				}
				return
			}
		}
	} else if strings.HasPrefix(msg.FieldID, FieldPrefixSceneActionColor) {
		// Scene action color - format: "scene-action-color:<sceneID>:<lightID>"
		rest := strings.TrimPrefix(msg.FieldID, FieldPrefixSceneActionColor)
		parts := strings.SplitN(rest, ":", 2)
		if len(parts) == 2 {
			sceneID := parts[0]
			lightID := parts[1]
			if v, ok := msg.Value.(field.ColorValue); ok {
				m.status = "Setting scene color..."
				err, modeSwitched := bridge.UpdateSceneActionColor(sceneID, lightID, float32(v.X), float32(v.Y))
				if err != nil {
					m.status = fmt.Sprintf("Error: %v", err)
				} else {
					m.status = "Scene updated"
					// Update Inactive flags directly instead of rebuilding UI
					// This preserves the component state during drag operations
					if modeSwitched {
						m.updateSceneColorModeFlags(sceneID, lightID, true)
					}
					// Update brightness slider color in real-time
					m.updateSceneBrightnessColor(sceneID, lightID, v.X, v.Y)
				}
				return
			}
		}
	} else if strings.HasPrefix(msg.FieldID, FieldPrefixSceneActionCT) {
		// Scene action color temperature - format: "scene-action-ct:<sceneID>:<lightID>"
		rest := strings.TrimPrefix(msg.FieldID, FieldPrefixSceneActionCT)
		parts := strings.SplitN(rest, ":", 2)
		if len(parts) == 2 {
			sceneID := parts[0]
			lightID := parts[1]
			if v, ok := msg.Value.(field.SliderValue); ok {
				// Convert Kelvin from display to mirek for API
				kelvin := 1000000 / v.Value
				m.status = fmt.Sprintf("Setting scene color temp to %dK...", kelvin)
				err, modeSwitched := bridge.UpdateSceneActionColorTemp(sceneID, lightID, v.Value)
				if err != nil {
					m.status = fmt.Sprintf("Error: %v", err)
				} else {
					m.status = "Scene updated"
					// Update Inactive flags directly instead of rebuilding UI
					// This preserves the component state during drag operations
					if modeSwitched {
						m.updateSceneColorModeFlags(sceneID, lightID, false)
					}
					// Update brightness slider color temp in real-time
					m.updateSceneBrightnessColorTemp(sceneID, lightID, v.Value)
				}
				return
			}
		}
	} else if strings.HasPrefix(msg.FieldID, FieldPrefixMotionEnabled) {
		motionID := strings.TrimPrefix(msg.FieldID, FieldPrefixMotionEnabled)
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
	} else if strings.HasPrefix(msg.FieldID, FieldPrefixMotionSensitivity) {
		motionID := strings.TrimPrefix(msg.FieldID, FieldPrefixMotionSensitivity)
		if v, ok := msg.Value.(field.SliderValue); ok {
			m.status = fmt.Sprintf("Motion sensitivity: %d", v.Value)
			err = bridge.SetMotionSensorSensitivity(motionID, v.Value)
			if err != nil {
				m.status = fmt.Sprintf("Error: %v", err)
			}
			return
		}
	} else if strings.HasPrefix(msg.FieldID, FieldPrefixTempEnabled) {
		tempID := strings.TrimPrefix(msg.FieldID, FieldPrefixTempEnabled)
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
	} else if strings.HasPrefix(msg.FieldID, FieldPrefixLightLevelEnabled) {
		llID := strings.TrimPrefix(msg.FieldID, FieldPrefixLightLevelEnabled)
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
	} else if strings.HasPrefix(msg.FieldID, FieldPrefixRoomPower) || strings.HasPrefix(msg.FieldID, FieldPrefixZonePower) {
		// Room or zone grouped light power control
		var glID string
		if strings.HasPrefix(msg.FieldID, FieldPrefixRoomPower) {
			glID = strings.TrimPrefix(msg.FieldID, FieldPrefixRoomPower)
		} else {
			glID = strings.TrimPrefix(msg.FieldID, FieldPrefixZonePower)
		}
		if v, ok := msg.Value.(field.ToggleValue); ok {
			err = bridge.SetGroupedLightOn(glID, v.On)
		}
	} else if strings.HasPrefix(msg.FieldID, FieldPrefixRoomBrightness) || strings.HasPrefix(msg.FieldID, FieldPrefixZoneBrightness) {
		// Room or zone grouped light brightness control
		var glID string
		if strings.HasPrefix(msg.FieldID, FieldPrefixRoomBrightness) {
			glID = strings.TrimPrefix(msg.FieldID, FieldPrefixRoomBrightness)
		} else {
			glID = strings.TrimPrefix(msg.FieldID, FieldPrefixZoneBrightness)
		}
		if v, ok := msg.Value.(field.SliderValue); ok {
			err = bridge.SetGroupedLightBrightness(glID, float64(v.Value))
		}
	} else if strings.HasPrefix(msg.FieldID, FieldPrefixGradientMode) {
		// Gradient mode selection
		gradientLightID := strings.TrimPrefix(msg.FieldID, FieldPrefixGradientMode)
		if v, ok := msg.Value.(field.SelectValue); ok {
			state := bridge.GetState()
			if state != nil {
				if light, ok := state.GetLight(gradientLightID); ok {
					if light.Gradient != nil && len(light.Gradient.ModeValues) > 0 {
						modes := light.Gradient.ModeValues
						if v.Index >= 0 && v.Index < len(modes) {
							m.status = fmt.Sprintf("Gradient mode: %s", string(modes[v.Index]))
							err = bridge.SetLightGradientMode(gradientLightID, hueclient.LightGetGradientMode(modes[v.Index]))
						}
					}
				}
			}
		}
	} else if strings.HasPrefix(msg.FieldID, FieldPrefixGradientPoints) {
		// Gradient points changed
		gradientLightID := strings.TrimPrefix(msg.FieldID, FieldPrefixGradientPoints)
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
						if light.Effects != nil && len(light.Effects.EffectValues) > 0 {
							effects := light.Effects.EffectValues
							if v.Index >= 0 && v.Index < len(effects) {
								effect := hueclient.Effect(effects[v.Index])
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
