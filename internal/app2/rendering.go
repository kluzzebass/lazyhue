package app2

import (
	"fmt"
	"strings"

	"image/color"

	"github.com/charmbracelet/lipgloss/v2"
	"github.com/kluzzebass/lazyhue/internal/hue"
	"github.com/kluzzebass/lazyhue/internal/hueclient"
	"github.com/kluzzebass/lazyhue/internal/ui2"
	"github.com/kluzzebass/lazyhue/internal/ui2/panels"
)

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
	node := m.tree.SelectedNode()
	if node == nil || node.Item == nil {
		m.detailViewport.SetContent("No selection")
		return
	}

	// Get bridge state if we have an active bridge
	var state *hue.BridgeState
	if m.activeBridgeID != "" {
		bridge := m.manager.GetBridge(m.activeBridgeID)
		if bridge != nil {
			state = bridge.GetState()
		}
	}

	// Build content based on entity type
	var content strings.Builder
	switch node.Item.Type {
	case panels.EntityLight:
		content.WriteString(m.buildLightDetails(node.Item, state))
	default:
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

	// Get the selected item's name for the panel title
	title := "Details"
	node := m.tree.SelectedNode()
	if node != nil && node.Item != nil && node.Item.Name != "" {
		title = node.Item.Name
	}

	topBorder := m.renderPanelHeader(width, key, title, focused, borderColor)

	content := m.detailViewport.View()

	innerWidth := width - 2
	if innerWidth < 1 {
		innerWidth = 1
	}
	innerHeight := height - 2
	if innerHeight < 1 {
		innerHeight = 1
	}

	contentLines := strings.Split(content, "\n")
	// Remove trailing empty line if present (viewport may add trailing newline)
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
	for i := 0; i < len(contentLines) && i < innerHeight; i++ {
		line := contentLines[i]
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

// buildLightDetails builds the detail content for a light entity.
// Formatting matches v1 UI: aligned fields, proper headers, consistent spacing.
func (m *Model) buildLightDetails(item *panels.EntityItem, state *hue.BridgeState) string {
	// Get light from RawPtr first (most reliable), then fall back to state lookup
	var light hueclient.LightGet
	var ok bool

	if item.RawPtr != nil {
		if l, typeOk := item.RawPtr.(hueclient.LightGet); typeOk {
			light = l
			ok = true
		}
	}

	// Fall back to state lookup if RawPtr didn't work
	if !ok && state != nil {
		light, ok = state.GetLight(item.ID)
	}

	// Last resort: search through all lights
	if !ok && state != nil {
		allLights := state.AllLights()
		for _, l := range allLights {
			if l.Id != nil && *l.Id == item.ID {
				light = l
				ok = true
				break
			}
		}
	}

	if !ok {
		return fmt.Sprintf("Light %s not found (RawPtr: %v, State: %v)", item.ID, item.RawPtr != nil, state != nil)
	}

	// Get owning device info for product details
	var device *hueclient.DeviceGet
	if state != nil && light.Owner != nil && light.Owner.Rid != nil {
		if d, ok := state.GetDevice(*light.Owner.Rid); ok {
			device = &d
		}
	}

	// Collect all fields to calculate alignment
	type fieldInfo struct {
		label string
		value string
	}
	var allFields []fieldInfo

	// Product info from device
	var productFields []fieldInfo
	if device != nil && device.ProductData != nil {
		pd := device.ProductData
		if pd.ProductName != nil {
			productFields = append(productFields, fieldInfo{"Product", *pd.ProductName})
		}
		if pd.ManufacturerName != nil {
			productFields = append(productFields, fieldInfo{"Manufacturer", *pd.ManufacturerName})
		}
		if pd.ModelId != nil {
			productFields = append(productFields, fieldInfo{"Model", *pd.ModelId})
		}
		if pd.SoftwareVersion != nil {
			productFields = append(productFields, fieldInfo{"Firmware", *pd.SoftwareVersion})
		}
		if pd.HardwarePlatformType != nil {
			productFields = append(productFields, fieldInfo{"Hardware", *pd.HardwarePlatformType})
		}
		allFields = append(allFields, productFields...)
	}

	// Classification fields
	var classFields []fieldInfo
	if device != nil && device.ProductData != nil && device.ProductData.ProductArchetype != nil {
		classFields = append(classFields, fieldInfo{"Archetype", string(*device.ProductData.ProductArchetype)})
	}
	if light.Type != nil {
		classFields = append(classFields, fieldInfo{"Type", string(*light.Type)})
	}
	if light.Mode != nil {
		classFields = append(classFields, fieldInfo{"Mode", string(*light.Mode)})
	}
	allFields = append(allFields, classFields...)

	// Name fields
	var nameFields []fieldInfo
	var hasNameSection bool
	currentName := ""
	if device != nil && device.Metadata != nil && device.Metadata.Name != nil {
		currentName = *device.Metadata.Name
		nameFields = append(nameFields, fieldInfo{"Name", currentName})
		hasNameSection = true
	}
	// Show light name (deprecated - names are now stored on device) if available and different from current name
	if light.Metadata != nil && light.Metadata.Name != nil {
		lightName := *light.Metadata.Name
		if lightName != currentName {
			nameFields = append(nameFields, fieldInfo{"Light name", lightName})
			hasNameSection = true
		}
	}
	allFields = append(allFields, nameFields...)

	// ID fields
	var idFields []fieldInfo
	if light.Id != nil {
		idFields = append(idFields, fieldInfo{"Light ID", *light.Id})
	}
	if light.Owner != nil && light.Owner.Rid != nil {
		idFields = append(idFields, fieldInfo{"Device ID", *light.Owner.Rid})
	}
	if light.IdV1 != nil {
		idFields = append(idFields, fieldInfo{"V1 ID", *light.IdV1})
	}
	allFields = append(allFields, idFields...)

	// State fields
	var stateFields []fieldInfo
	if light.Dimming != nil {
		if light.Dimming.Brightness != nil {
			stateFields = append(stateFields, fieldInfo{"Brightness", fmt.Sprintf("%.0f%%", float64(*light.Dimming.Brightness))})
		}
		if light.Dimming.MinDimLevel != nil {
			stateFields = append(stateFields, fieldInfo{"Min dim level", fmt.Sprintf("%.0f%%", float64(*light.Dimming.MinDimLevel))})
		}
	}
	if light.Color != nil && light.Color.Xy != nil {
		xy := light.Color.Xy
		if xy.X != nil && xy.Y != nil {
			x, y := *xy.X, *xy.Y
			stateFields = append(stateFields, fieldInfo{"Color XY", fmt.Sprintf("(%.4f, %.4f)", x, y)})
		}
		if light.Color.GamutType != nil {
			stateFields = append(stateFields, fieldInfo{"Gamut", string(*light.Color.GamutType)})
		}
	}
	if light.ColorTemperature != nil {
		if light.ColorTemperature.Mirek != nil {
			mirek := *light.ColorTemperature.Mirek
			kelvin := 1000000 / int(mirek)
			stateFields = append(stateFields, fieldInfo{"Color temp", fmt.Sprintf("%d mirek (~%dK)", mirek, kelvin)})
		}
		if light.ColorTemperature.MirekSchema != nil {
			schema := light.ColorTemperature.MirekSchema
			if schema.MirekMinimum != nil && schema.MirekMaximum != nil {
				minK := 1000000 / int(*schema.MirekMaximum)
				maxK := 1000000 / int(*schema.MirekMinimum)
				stateFields = append(stateFields, fieldInfo{"CT range", fmt.Sprintf("%dK - %dK", minK, maxK)})
			}
		}
	}
	allFields = append(allFields, stateFields...)

	// Dynamics fields
	var dynamicsFields []fieldInfo
	if light.Dynamics != nil {
		if light.Dynamics.Status != nil {
			dynamicsFields = append(dynamicsFields, fieldInfo{"Status", string(*light.Dynamics.Status)})
		}
		if light.Dynamics.Speed != nil {
			dynamicsFields = append(dynamicsFields, fieldInfo{"Speed", fmt.Sprintf("%.2f", *light.Dynamics.Speed)})
		}
		allFields = append(allFields, dynamicsFields...)
	}

	// Gradient fields
	var gradientFields []fieldInfo
	if light.Gradient != nil {
		if light.Gradient.PixelCount != nil {
			gradientFields = append(gradientFields, fieldInfo{"Pixels", fmt.Sprintf("%d", *light.Gradient.PixelCount)})
		}
		if light.Gradient.Points != nil && len(*light.Gradient.Points) > 0 {
			gradientFields = append(gradientFields, fieldInfo{"Points", fmt.Sprintf("%d", len(*light.Gradient.Points))})
		}
		allFields = append(allFields, gradientFields...)
	}

	// Powerup fields
	var powerupFields []fieldInfo
	if light.Powerup != nil && light.Powerup.Preset != nil {
		powerupFields = append(powerupFields, fieldInfo{"Preset", string(*light.Powerup.Preset)})
		allFields = append(allFields, powerupFields...)
	}

	// Calculate max label width for alignment
	maxLabelWidth := 0
	for _, f := range allFields {
		if len(f.label) > maxLabelWidth {
			maxLabelWidth = len(f.label)
		}
	}

	var content strings.Builder

	// Helper to render header (newline before, colon after, newline after)
	renderHeader := func(title string) {
		content.WriteString("\n")
		content.WriteString(m.styles.Subtitle.Render(title + ":"))
		content.WriteString("\n")
	}

	// Helper to render aligned field
	renderField := func(label, value string) {
		labelWithColon := label + ":"
		padding := maxLabelWidth - len(label)
		if padding < 0 {
			padding = 0
		}
		labelStr := labelWithColon + strings.Repeat(" ", padding) + " "
		content.WriteString(fmt.Sprintf("  %s%s\n", labelStr, value))
	}

	// Helper to render muted field
	renderMutedField := func(label, value string) {
		labelWithColon := label + ":"
		padding := maxLabelWidth - len(label)
		if padding < 0 {
			padding = 0
		}
		mutedLabel := lipgloss.NewStyle().Foreground(m.styles.Theme.TextMuted).Render(labelWithColon)
		labelStr := mutedLabel + strings.Repeat(" ", padding) + " "
		content.WriteString(fmt.Sprintf("  %s%s\n", labelStr, value))
	}

	// Product info
	if len(productFields) > 0 {
		renderHeader("Product")
		for _, f := range productFields {
			renderField(f.label, f.value)
		}
		content.WriteString("\n")
	}

	// Classification
	if len(classFields) > 0 {
		renderHeader("Classification")
		for _, f := range classFields {
			renderField(f.label, f.value)
		}
		content.WriteString("\n")
	}

	// Name section
	if hasNameSection {
		renderHeader("Name")
		for _, f := range nameFields {
			if f.label == "Light name" {
				renderMutedField(f.label, f.value)
			} else {
				renderField(f.label, f.value)
			}
		}
		content.WriteString("\n")
	}

	// IDs
	if len(idFields) > 0 {
		renderHeader("IDs")
		for _, f := range idFields {
			renderField(f.label, f.value)
		}
		content.WriteString("\n")
	}

	// Current State
	renderHeader("State")
	isOn := panels.IsLightOn(light)
	status := "off"
	if isOn {
		status = "on"
	}
	brightness := 0.0
	hexColor := ui2.GetLightColor(light)
	if isOn {
		if light.Dimming != nil && light.Dimming.Brightness != nil {
			brightness = float64(*light.Dimming.Brightness)
		} else {
			brightness = 100.0
		}
	}
	indicator := ui2.RenderBrightnessIndicatorFromHex(brightness, hexColor)
	content.WriteString(fmt.Sprintf("  %s %s\n", indicator, status))
	if len(stateFields) > 0 {
		for _, f := range stateFields {
			renderField(f.label, f.value)
		}
	}
	content.WriteString("\n")

	// Dynamics
	if len(dynamicsFields) > 0 {
		renderHeader("Dynamics")
		for _, f := range dynamicsFields {
			renderField(f.label, f.value)
		}
		content.WriteString("\n")
	}

	// Capabilities
	renderHeader("Capabilities")
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
	if len(caps) > 0 {
		for _, cap := range caps {
			content.WriteString(fmt.Sprintf("  • %s\n", cap))
		}
	} else {
		content.WriteString(fmt.Sprintf("  %s\n", lipgloss.NewStyle().Foreground(m.styles.Theme.TextMuted).Render("On/Off only")))
	}
	content.WriteString("\n")

	// Effects
	if light.Effects != nil && light.Effects.EffectValues != nil && len(*light.Effects.EffectValues) > 0 {
		renderHeader("Available Effects")
		for _, effect := range *light.Effects.EffectValues {
			content.WriteString(fmt.Sprintf("  • %s\n", hue.EffectDisplayName(string(effect))))
		}
		content.WriteString("\n")
	}

	// Gradient
	if light.Gradient != nil {
		renderHeader("Gradient")
		if light.Gradient.Mode != nil {
			renderMutedField("Mode", string(*light.Gradient.Mode))
		}
		for _, f := range gradientFields {
			renderField(f.label, f.value)
		}
		content.WriteString("\n")
	}

	// Signaling
	if light.Signaling != nil && light.Signaling.SignalValues != nil && len(*light.Signaling.SignalValues) > 0 {
		renderHeader("Signaling Modes")
		for _, sig := range *light.Signaling.SignalValues {
			content.WriteString(fmt.Sprintf("  • %s\n", string(sig)))
		}
		content.WriteString("\n")
	}

	// Powerup behavior
	if len(powerupFields) > 0 {
		renderHeader("Power-on Behavior")
		for _, f := range powerupFields {
			renderField(f.label, f.value)
		}
		content.WriteString("\n")
	}

	// Device services
	if device != nil && device.Services != nil && len(*device.Services) > 0 {
		renderHeader("Device Services")
		for _, svc := range *device.Services {
			rtype := "unknown"
			if svc.Rtype != nil {
				rtype = string(*svc.Rtype)
			}
			if svc.Rid != nil && light.Id != nil && *svc.Rid == *light.Id {
				content.WriteString(fmt.Sprintf("  • %s %s\n", rtype, lipgloss.NewStyle().Foreground(m.styles.Theme.TextMuted).Render("(this)")))
			} else {
				content.WriteString(fmt.Sprintf("  • %s\n", rtype))
			}
		}
		content.WriteString("\n")
	}

	return content.String()
}
