package panels

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/kluzzebass/lazyhue/internal/hue"
	"github.com/kluzzebass/lazyhue/internal/hueclient"
	"github.com/kluzzebass/lazyhue/internal/ui"
	"github.com/kluzzebass/lazyhue/internal/ui/panels/details"
)

// DetailsPanel shows details for the selected entity using the new view system.
type DetailsPanel struct {
	viewport viewport.Model
	styles   ui.Styles
	item     *EntityItem
	state    *hue.BridgeState
	width    int
	height   int

	// Shared form component for editable fields
	form *Form

	// Callback when a field is saved
	onFieldSave func(entityType EntityType, entityID string, fieldID string, value interface{})
}

// NewDetailsPanel creates a new details panel.
func NewDetailsPanel(styles ui.Styles) *DetailsPanel {
	vp := viewport.New(0, 0)
	form := NewForm(styles)

	p := &DetailsPanel{
		viewport: vp,
		styles:   styles,
		form:     form,
	}

	// Set up form callbacks
	form.OnChange = func(field FormField) {
		p.handleFieldChange(field)
	}

	return p
}

// SetOnFieldSave sets the callback for when an editable field is saved.
func (p *DetailsPanel) SetOnFieldSave(fn func(entityType EntityType, entityID string, fieldID string, value interface{})) {
	p.onFieldSave = fn
}

// handleFieldChange is called when a form field value changes.
func (p *DetailsPanel) handleFieldChange(field FormField) {
	if p.onFieldSave == nil || p.item == nil {
		return
	}

	var value interface{}
	switch field.Type {
	case FormFieldToggle, FormFieldBrightness, FormFieldColorTemp, FormFieldSelect:
		value = field.Value
	case FormFieldText:
		value = field.TextValue
	case FormFieldColor:
		value = [2]float64{field.ColorX, field.ColorY}
	default:
		value = field.Value
	}

	p.onFieldSave(p.item.Type, p.item.ID, field.ID, value)
	p.updateContent()
}

// SetItem updates the displayed entity.
func (p *DetailsPanel) SetItem(item *EntityItem, state *hue.BridgeState) {
	p.item = item
	p.state = state
	p.buildFormFields()
	p.updateContent()
}

// buildFormFields creates the form fields for the current entity.
func (p *DetailsPanel) buildFormFields() {
	var fields []FormField

	if p.item == nil {
		p.form.SetFields(fields)
		return
	}

	switch p.item.Type {
	case EntityLight:
		if light, ok := GetLightFromItem(*p.item); ok {
			fields = p.buildLightFields(light)
		}
	case EntityRoom:
		if room, ok := GetRoomFromItem(*p.item); ok {
			fields = p.buildRoomFields(room, false)
		}
	case EntityZone:
		if zone, ok := GetRoomFromItem(*p.item); ok {
			fields = p.buildRoomFields(zone, true)
		}
	}

	p.form.SetFields(fields)
}

// buildLightFields creates form fields for a light.
func (p *DetailsPanel) buildLightFields(light hueclient.LightGet) []FormField {
	var fields []FormField

	// On/Off toggle
	onValue := 0
	if light.On != nil && light.On.On != nil && *light.On.On {
		onValue = 1
	}
	fields = append(fields, FormField{
		ID:             "on",
		Label:          "Power",
		Type:           FormFieldToggle,
		Value:          onValue,
		ToggleOnLabel:  "On",
		ToggleOffLabel: "Off",
	})

	// Brightness (if dimmable)
	if light.Dimming != nil && light.Dimming.Brightness != nil {
		brightness := int(*light.Dimming.Brightness)
		fields = append(fields, FormField{
			ID:    "brightness",
			Label: "Brightness",
			Type:  FormFieldBrightness,
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
		fields = append(fields, FormField{
			ID:    "colortemp",
			Label: "Color Temp",
			Type:  FormFieldColorTemp,
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
		fields = append(fields, FormField{
			ID:     "color",
			Label:  "Color",
			Type:   FormFieldColor,
			ColorX: x,
			ColorY: y,
		})
	}

	// Effect (if supported)
	if light.Effects != nil && light.Effects.EffectValues != nil && len(*light.Effects.EffectValues) > 0 {
		currentEffect := "no_effect"
		if light.Effects.Effect != nil {
			currentEffect = string(*light.Effects.Effect)
		}

		options := []FormSelectOption{}
		currentIdx := 0
		for i, effect := range *light.Effects.EffectValues {
			options = append(options, FormSelectOption{
				Label: hue.EffectDisplayName(string(effect)),
				Value: i,
			})
			if string(effect) == currentEffect {
				currentIdx = i
			}
		}

		fields = append(fields, FormField{
			ID:      "effect",
			Label:   "Effect",
			Type:    FormFieldSelect,
			Value:   currentIdx,
			Options: options,
		})
	}

	return fields
}

// buildRoomFields creates form fields for a room/zone.
func (p *DetailsPanel) buildRoomFields(room hueclient.RoomGet, isZone bool) []FormField {
	var fields []FormField

	// Archetype selector
	archetypes := hue.RoomArchetypeList()
	currentArchetype := ""
	if room.Metadata != nil && room.Metadata.Archetype != nil {
		currentArchetype = string(*room.Metadata.Archetype)
	}

	options := []FormSelectOption{}
	currentIdx := 0
	for i, arch := range archetypes {
		options = append(options, FormSelectOption{
			Label: hue.RoomArchetypeDisplayName(arch),
			Value: i,
		})
		if arch == currentArchetype {
			currentIdx = i
		}
	}

	fields = append(fields, FormField{
		ID:      "archetype",
		Label:   "Archetype",
		Type:    FormFieldSelect,
		Value:   currentIdx,
		Options: options,
	})

	return fields
}

// SetSize updates the panel dimensions.
func (p *DetailsPanel) SetSize(width, height int) {
	p.width = width
	p.height = height
	p.viewport.Width = max(1, width-2)
	p.viewport.Height = max(1, height-2)
	p.updateContent()
}

func (p *DetailsPanel) updateContent() {
	if p.item == nil {
		p.viewport.SetContent("No selection")
		return
	}

	// Build header
	var content strings.Builder
	title := p.styles.Title.Render(p.item.Name)
	content.WriteString(title)
	content.WriteString("\n")
	content.WriteString(strings.Repeat("─", max(0, min(30, p.width-6))))
	content.WriteString("\n\n")

	// Render editable fields using the shared form
	if len(p.form.Fields) > 0 {
		content.WriteString(p.form.Render(p.width - 4))
		content.WriteString("\n")
	}

	// Build view based on entity type
	var view *details.View
	switch p.item.Type {
	case EntityRoom:
		if room, ok := GetRoomFromItem(*p.item); ok {
			view = p.buildRoomView(room, false)
		}
	case EntityZone:
		if zone, ok := GetRoomFromItem(*p.item); ok {
			view = p.buildRoomView(zone, true)
		}
	case EntityLight:
		if light, ok := GetLightFromItem(*p.item); ok {
			view = p.buildLightView(light)
		}
	case EntityScene:
		if scene, ok := GetSceneFromItem(*p.item); ok {
			view = p.buildSceneView(scene)
		}
	case EntityDevice:
		if device, ok := GetDeviceFromItem(*p.item); ok {
			view = p.buildDeviceView(device)
		}
	case EntityEntertainment:
		if cfg, ok := GetEntertainmentFromItem(*p.item); ok {
			view = p.buildEntertainmentView(cfg)
		}
	case EntityLightsCategory:
		if data, ok := p.item.RawPtr.(LightsCategoryData); ok {
			view = p.buildLightsCategoryView(data)
		}
	case EntityDevicesCategory:
		if data, ok := p.item.RawPtr.(DevicesCategoryData); ok {
			view = p.buildDevicesCategoryView(data)
		}
	case EntityScenesCategory:
		if data, ok := p.item.RawPtr.(ScenesCategoryData); ok {
			view = p.buildScenesCategoryView(data)
		}
	case EntityBridge:
		if bridgeData, ok := GetBridgeFromItem(*p.item); ok {
			view = p.buildBridgeView(bridgeData)
		}
	default:
		view = details.NewView(p.styles)
		fields := details.NewFields()
		fields.AddMuted("Type", fmt.Sprintf("%d", p.item.Type))
		view.Add(fields)
	}

	if view != nil {
		content.WriteString(view.Render())
	}

	p.viewport.SetContent(content.String())
}

// StartBlinkTickMsg is sent to start the blink ticker.
type StartBlinkTickMsg struct{}

// Update handles input for the details panel.
func (p *DetailsPanel) Update(msg tea.Msg) (*DetailsPanel, tea.Cmd) {
	// Handle form input first
	if len(p.form.Fields) > 0 {
		wasEditingColor := p.form.IsEditingColor()

		switch m := msg.(type) {
		case tea.KeyMsg:
			if p.form.HandleKey(m.String()) {
				p.updateContent()
				// Start blink ticker if we just started editing a color
				if !wasEditingColor && p.form.IsEditingColor() {
					return p, func() tea.Msg { return StartBlinkTickMsg{} }
				}
				return p, nil
			}
		case tea.MouseMsg:
			// Calculate form field start position
			// Border (row 0), Title (row 1), separator (row 2), blank line (row 3), form starts at row 4
			fieldStartY := 4
			labelWidth := 0
			for _, f := range p.form.Fields {
				if len(f.Label) > labelWidth {
					labelWidth = len(f.Label)
				}
			}
			if p.form.HandleMouse(m, fieldStartY, labelWidth) {
				p.updateContent()
				// Start blink ticker if we just started editing a color
				if !wasEditingColor && p.form.IsEditingColor() {
					return p, func() tea.Msg { return StartBlinkTickMsg{} }
				}
				return p, nil
			}
		}
	}

	// Handle viewport navigation
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		switch keyMsg.String() {
		case "g":
			p.viewport.GotoTop()
			return p, nil
		case "G":
			p.viewport.GotoBottom()
			return p, nil
		}
	}

	var cmd tea.Cmd
	p.viewport, cmd = p.viewport.Update(msg)
	return p, cmd
}

// View renders the details panel.
func (p *DetailsPanel) View(active bool) string {
	content := p.viewport.View()

	contentLines := strings.Count(p.viewport.View(), "\n") + 1
	totalLines := p.viewport.TotalLineCount()

	lastVisibleLine := p.viewport.YOffset + contentLines
	if lastVisibleLine > totalLines {
		lastVisibleLine = totalLines
	}

	cfg := ui.BorderConfig{
		Title:       "[0] Details",
		ItemIndex:   lastVisibleLine - 1,
		ItemCount:   totalLines,
		ScrollPos:   p.viewport.YOffset,
		TotalHeight: totalLines,
		ViewHeight:  contentLines,
	}

	return ui.RenderBorderedPanel(content, p.width, p.height, active, p.styles, cfg)
}

// Title returns the panel title.
func (p *DetailsPanel) Title() string {
	return "Details"
}

// HasItems returns true if there is content to display.
func (p *DetailsPanel) HasItems() bool {
	return p.item != nil
}

// HandleScroll handles scroll events.
func (p *DetailsPanel) HandleScroll(lines int) {
	if lines > 0 {
		p.viewport.LineDown(lines)
	} else {
		p.viewport.LineUp(-lines)
	}
}

// ToggleBlink toggles the blink state for the color wheel indicator.
func (p *DetailsPanel) ToggleBlink() {
	p.form.ToggleBlink()
	// Re-render to show the blink change
	if p.form.IsEditingColor() {
		p.updateContent()
	}
}

// IsEditingColor returns true if we're currently editing a color field.
func (p *DetailsPanel) IsEditingColor() bool {
	return p.form.IsEditingColor()
}

// Builder methods are implemented in separate files:
// - details_device.go: buildDeviceView
// - details_room.go: buildRoomView
// - details_light.go: buildLightView
// - details_scene.go: buildSceneView
// - details_entertainment.go: buildEntertainmentView
// - details_category.go: buildLightsCategoryView, buildDevicesCategoryView, buildScenesCategoryView
// - details_bridge.go: buildBridgeView
