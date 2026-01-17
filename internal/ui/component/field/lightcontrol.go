package field

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/kluzzebass/lazyhue/internal/ui"
	"github.com/kluzzebass/lazyhue/internal/ui/component"
	"github.com/kluzzebass/lazyhue/internal/ui/components"
	zone "github.com/lrstanley/bubblezone/v2"
)

// LightControlState holds all state for a light control component.
type LightControlState struct {
	// Power
	On bool

	// Brightness (0-100)
	Brightness int

	// Color temperature
	ColorTemp int
	MinMirek  int
	MaxMirek  int

	// Color (XY)
	ColorX, ColorY float64

	// Mode (determines which color control is active)
	MirekValid bool // true = color temp active, false = XY color active

	// Effect
	EffectIndex int
	Effects     []string

	// Gradient
	GradientPoints    []GradientPoint
	GradientMode      int
	GradientModes     []string
	MaxGradientPoints int

	// Capabilities (determines which controls exist)
	HasDimming   bool
	HasColor     bool
	HasColorTemp bool
	HasEffects   bool
	HasGradient  bool
}

// LightControlComponent combines all light controls into a single, reusable component.
type LightControlComponent struct {
	*BaseField

	// Identity
	LightID  string
	BridgeID string
	Name     string

	// Child components (composed)
	powerToggle      *ToggleComponent
	identifyButton   *ButtonComponent
	brightnessSlider *BrightnessSliderComponent
	colorTempSlider  *ColorTempSliderComponent
	colorWheel       *ColorWheelComponent
	rgbControl       *RGBComponent
	hslControl       *HSLComponent
	hsvControl       *HSVComponent
	effectSelect     *SelectComponent
	gradientEditor   *GradientEditorComponent

	// State
	state LightControlState

	// Focus management
	focusIndex      int   // Which child has focus
	focusableFields []int // Indices of focusable children based on capabilities

	// Mouse capture state
	captureField int // Index of field capturing mouse (-1 if none)

	// Internal state for reactive color updates
	updatingColors bool // Prevents infinite loops when syncing colors
}

// NewLightControlComponent creates a new unified light control component.
func NewLightControlComponent(lightID, bridgeID, name string, styles *ui.Styles, zones *zone.Manager) *LightControlComponent {
	baseID := fmt.Sprintf("lc:%s", lightID)

	c := &LightControlComponent{
		BaseField:    NewBaseField(baseID, "", styles, zones),
		LightID:      lightID,
		BridgeID:     bridgeID,
		Name:         name,
		captureField: -1,
	}

	// Create child components with prefixed IDs
	c.powerToggle = NewToggleComponent(baseID+":on", "", true, styles, zones)
	c.powerToggle.SetLabels("", "") // No labels, just checkbox

	c.identifyButton = NewButtonComponent(baseID+":identify", "", "ID", styles, zones)

	c.brightnessSlider = NewBrightnessSliderComponent(baseID+":brightness", "Brightness", 100, styles, zones)

	c.colorTempSlider = NewColorTempSliderComponent(baseID+":colortemp", "Color Temp", 350, 153, 500, styles, zones)

	c.colorWheel = NewColorWheelComponent(baseID+":color-wheel", "Color", 0.3127, 0.329, styles, zones)

	c.rgbControl = NewRGBComponent(baseID+":rgb", "RGB", 255, 255, 255, styles, zones)
	c.rgbControl.ShowSwatch = false

	c.hslControl = NewHSLComponent(baseID+":hsl", "HSL", 0, 0, 50, styles, zones)
	c.hslControl.ShowSwatch = false

	c.hsvControl = NewHSVComponent(baseID+":hsv", "HSV", 0, 0, 100, styles, zones)
	c.hsvControl.ShowSwatch = false

	c.effectSelect = NewSelectComponent(baseID+":effect", "Effect", 0, nil, styles, zones)

	c.gradientEditor = NewGradientEditorComponent(baseID+":gradient", "Gradient", nil, 5, styles, zones)

	// Set parent for all children
	c.powerToggle.SetParent(c)
	c.identifyButton.SetParent(c)
	c.brightnessSlider.SetParent(c)
	c.colorTempSlider.SetParent(c)
	c.colorWheel.SetParent(c)
	c.rgbControl.SetParent(c)
	c.hslControl.SetParent(c)
	c.hsvControl.SetParent(c)
	c.effectSelect.SetParent(c)
	c.gradientEditor.SetParent(c)

	// Initialize focusable fields (will be updated by SetState)
	c.updateFocusableFields()

	return c
}

// SetState updates all internal state and child components.
func (c *LightControlComponent) SetState(state LightControlState) {
	c.state = state

	// Update power toggle
	c.powerToggle.SetValue(state.On)

	// Update brightness
	c.brightnessSlider.SetValue(state.Brightness)
	c.brightnessSlider.ColorX = state.ColorX
	c.brightnessSlider.ColorY = state.ColorY
	if state.MirekValid {
		c.brightnessSlider.ColorTempMirek = state.ColorTemp
	}

	// Update color temp slider
	c.colorTempSlider.SetValue(state.ColorTemp)
	c.colorTempSlider.Min = state.MinMirek
	c.colorTempSlider.Max = state.MaxMirek
	c.colorTempSlider.Inactive = !state.MirekValid
	c.colorTempSlider.Brightness = state.Brightness

	// Update color controls
	c.colorWheel.SetColor(state.ColorX, state.ColorY)
	c.colorWheel.Inactive = state.MirekValid
	c.colorWheel.Brightness = state.Brightness

	// Convert XY to RGB, HSL, and HSV for the other controls
	r, g, b := ui.XyToRGB(state.ColorX, state.ColorY, 100)
	c.rgbControl.SetRGB(int(r), int(g), int(b))
	c.rgbControl.Inactive = state.MirekValid

	h, s, l := ui.RGBToHSL(r, g, b)
	c.hslControl.SetHSL(h, s, l)
	c.hslControl.Inactive = state.MirekValid

	hh, ss, v := ui.RGBToHSV(r, g, b)
	c.hsvControl.SetHSV(hh, ss, v)
	c.hsvControl.Inactive = state.MirekValid

	// Update effect selector
	if len(state.Effects) > 0 {
		options := make([]Option, len(state.Effects))
		for i, effect := range state.Effects {
			options[i] = Option{Label: effect, Value: i}
		}
		c.effectSelect.SetOptions(options)
		c.effectSelect.SetValue(state.EffectIndex)
	}

	// Update gradient editor
	if state.HasGradient && len(state.GradientPoints) > 0 {
		c.gradientEditor.Points = state.GradientPoints
		c.gradientEditor.MaxPoints = state.MaxGradientPoints
	}

	// Update focusable fields based on capabilities
	c.updateFocusableFields()
}

// updateFocusableFields rebuilds the list of focusable field indices.
func (c *LightControlComponent) updateFocusableFields() {
	c.focusableFields = []int{}

	// Order: Power, Identify, Brightness, ColorTemp, ColorWheel, RGB, HSL, HSV, Effect, Gradient
	c.focusableFields = append(c.focusableFields, 0) // Power toggle
	c.focusableFields = append(c.focusableFields, 1) // Identify button

	if c.state.HasDimming {
		c.focusableFields = append(c.focusableFields, 2) // Brightness
	}
	if c.state.HasColorTemp {
		c.focusableFields = append(c.focusableFields, 3) // Color temp
	}
	if c.state.HasColor {
		c.focusableFields = append(c.focusableFields, 4) // Color wheel
		c.focusableFields = append(c.focusableFields, 5) // RGB
		c.focusableFields = append(c.focusableFields, 6) // HSL
		c.focusableFields = append(c.focusableFields, 7) // HSV
	}
	if c.state.HasEffects {
		c.focusableFields = append(c.focusableFields, 8) // Effect
	}
	if c.state.HasGradient {
		c.focusableFields = append(c.focusableFields, 9) // Gradient
	}

	// Focus first field if current focus is invalid
	if len(c.focusableFields) > 0 && c.focusIndex >= len(c.focusableFields) {
		c.focusIndex = 0
	}

	// Update focus state on children
	c.updateChildFocus()
}

// getFieldByIndex returns the component at the given logical index.
func (c *LightControlComponent) getFieldByIndex(index int) component.Component {
	switch index {
	case 0:
		return c.powerToggle
	case 1:
		return c.identifyButton
	case 2:
		return c.brightnessSlider
	case 3:
		return c.colorTempSlider
	case 4:
		return c.colorWheel
	case 5:
		return c.rgbControl
	case 6:
		return c.hslControl
	case 7:
		return c.hsvControl
	case 8:
		return c.effectSelect
	case 9:
		return c.gradientEditor
	}
	return nil
}

// getFocusedField returns the currently focused field component.
func (c *LightControlComponent) getFocusedField() component.Component {
	if c.focusIndex >= 0 && c.focusIndex < len(c.focusableFields) {
		return c.getFieldByIndex(c.focusableFields[c.focusIndex])
	}
	return nil
}

// updateChildFocus updates focus state on all children.
func (c *LightControlComponent) updateChildFocus() {
	// Blur all children
	c.powerToggle.Blur()
	c.identifyButton.Blur()
	c.brightnessSlider.Blur()
	c.colorTempSlider.Blur()
	c.colorWheel.Blur()
	c.rgbControl.Blur()
	c.hslControl.Blur()
	c.hsvControl.Blur()
	c.effectSelect.Blur()
	c.gradientEditor.Blur()

	// Focus the current field
	if field := c.getFocusedField(); field != nil {
		field.Focus()
	}
}

// Update handles events for the light control component.
func (c *LightControlComponent) Update(msg tea.Msg) (component.Component, tea.Cmd) {
	var cmds []tea.Cmd

	// Handle blink tick messages - route to appropriate field
	if blinkMsg, ok := msg.(BlinkTickMsg); ok {
		if strings.HasPrefix(blinkMsg.FieldID, c.ID) {
			if strings.Contains(blinkMsg.FieldID, ":color-wheel") {
				_, cmd := c.colorWheel.Update(msg)
				if cmd != nil {
					cmds = append(cmds, cmd)
				}
			} else if strings.Contains(blinkMsg.FieldID, ":gradient") {
				_, cmd := c.gradientEditor.Update(msg)
				if cmd != nil {
					cmds = append(cmds, cmd)
				}
			}
		}
		return c, tea.Batch(cmds...)
	}

	// Handle button reset messages
	if resetMsg, ok := msg.(buttonResetMsg); ok {
		if strings.HasPrefix(resetMsg.fieldID, c.ID) {
			c.identifyButton.Update(msg)
		}
		return c, nil
	}

	// Handle capture field messages
	if _, ok := msg.(StartCaptureMsg); ok {
		c.captureField = c.focusIndex
	}
	if _, ok := msg.(EndCaptureMsg); ok {
		c.captureField = -1
	}

	return c, tea.Batch(cmds...)
}

// RouteEvent routes events to the appropriate child.
func (c *LightControlComponent) RouteEvent(msg tea.Msg) (bool, tea.Cmd) {
	// If capturing mouse, route all mouse events to capture field
	if c.captureField >= 0 && c.captureField < len(c.focusableFields) {
		switch msg.(type) {
		case tea.MouseMsg, tea.MouseClickMsg, tea.MouseReleaseMsg, tea.MouseMotionMsg:
			field := c.getFieldByIndex(c.focusableFields[c.captureField])
			if field != nil {
				if handled, cmd := field.RouteEvent(msg); handled {
					return true, c.wrapFieldCmd(cmd, c.focusableFields[c.captureField])
				}
			}
		}
	}

	// Route to focused field first
	if focusedField := c.getFocusedField(); focusedField != nil {
		if handled, cmd := focusedField.RouteEvent(msg); handled {
			return true, c.wrapFieldCmd(cmd, c.focusableFields[c.focusIndex])
		}
	}

	// Handle form-level navigation
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return c.handleKey(msg)

	case tea.MouseClickMsg:
		return c.handleMouseClick(msg)
	}

	return false, nil
}

// wrapFieldCmd wraps a command from a child field to handle reactive updates.
func (c *LightControlComponent) wrapFieldCmd(cmd tea.Cmd, fieldIndex int) tea.Cmd {
	if cmd == nil {
		return nil
	}

	// Execute the command and intercept the message to handle reactive updates
	return func() tea.Msg {
		msg := cmd()

		// Check if it's a field changed message from a color control
		if fcm, ok := msg.(FieldChangedMsg); ok {
			// Handle reactive color updates
			c.handleReactiveColorUpdate(fcm, fieldIndex)

			// Rewrite the field ID to use the standard color field ID
			// so the app handler can route it correctly
			if fieldIndex == 4 || fieldIndex == 5 || fieldIndex == 6 || fieldIndex == 7 {
				// Color wheel, RGB, HSL, or HSV -> emit as color change
				fcm.FieldID = c.ID + ":color"
				fcm.Value = ColorValue{X: c.state.ColorX, Y: c.state.ColorY}
			}

			return fcm
		}

		return msg
	}
}

// handleReactiveColorUpdate syncs all color controls when one changes.
func (c *LightControlComponent) handleReactiveColorUpdate(msg FieldChangedMsg, fieldIndex int) {
	if c.updatingColors {
		return // Prevent infinite loop
	}
	c.updatingColors = true
	defer func() { c.updatingColors = false }()

	switch fieldIndex {
	case 4: // Color wheel
		if cv, ok := msg.Value.(ColorValue); ok {
			c.state.ColorX = cv.X
			c.state.ColorY = cv.Y

			// Update RGB, HSL, and HSV
			r, g, b := ui.XyToRGB(cv.X, cv.Y, 100)
			c.rgbControl.SetRGB(int(r), int(g), int(b))

			h, s, l := ui.RGBToHSL(r, g, b)
			c.hslControl.SetHSL(h, s, l)

			hh, ss, v := ui.RGBToHSV(r, g, b)
			c.hsvControl.SetHSV(hh, ss, v)

			// Update brightness slider color
			c.brightnessSlider.ColorX = cv.X
			c.brightnessSlider.ColorY = cv.Y
		}

	case 5: // RGB
		if rv, ok := msg.Value.(RGBValue); ok {
			// Convert RGB to XY
			x, y := ui.RGBIntToXY(rv.Red, rv.Green, rv.Blue)
			c.state.ColorX = x
			c.state.ColorY = y

			// Update color wheel
			c.colorWheel.SetColor(x, y)

			// Update HSL
			h, s, l := ui.RGBToHSL(uint8(rv.Red), uint8(rv.Green), uint8(rv.Blue))
			c.hslControl.SetHSL(h, s, l)

			// Update HSV
			hh, ss, v := ui.RGBToHSV(uint8(rv.Red), uint8(rv.Green), uint8(rv.Blue))
			c.hsvControl.SetHSV(hh, ss, v)

			// Update brightness slider color
			c.brightnessSlider.ColorX = x
			c.brightnessSlider.ColorY = y
		}

	case 6: // HSL
		if hv, ok := msg.Value.(HSLValue); ok {
			// Convert HSL to RGB then XY
			r, g, b := ui.HSLToRGB(hv.Hue, hv.Saturation, hv.Lightness)
			x, y := ui.RGBToXY(r, g, b)
			c.state.ColorX = x
			c.state.ColorY = y

			// Update color wheel
			c.colorWheel.SetColor(x, y)

			// Update RGB
			c.rgbControl.SetRGB(int(r), int(g), int(b))

			// Update HSV
			hh, ss, v := ui.RGBToHSV(r, g, b)
			c.hsvControl.SetHSV(hh, ss, v)

			// Update brightness slider color
			c.brightnessSlider.ColorX = x
			c.brightnessSlider.ColorY = y
		}

	case 7: // HSV
		if hv, ok := msg.Value.(HSVValue); ok {
			// Convert HSV to RGB then XY
			r, g, b := ui.HsvToRGB(hv.Hue, hv.Saturation, hv.Value)
			x, y := ui.RGBToXY(r, g, b)
			c.state.ColorX = x
			c.state.ColorY = y

			// Update color wheel
			c.colorWheel.SetColor(x, y)

			// Update RGB
			c.rgbControl.SetRGB(int(r), int(g), int(b))

			// Update HSL
			h, s, l := ui.RGBToHSL(r, g, b)
			c.hslControl.SetHSL(h, s, l)

			// Update brightness slider color
			c.brightnessSlider.ColorX = x
			c.brightnessSlider.ColorY = y
		}
	}
}

func (c *LightControlComponent) handleKey(msg tea.KeyMsg) (bool, tea.Cmd) {
	// Child components return false when they want parent to handle navigation
	// (e.g., at edge of multi-slider control), so we handle it here
	switch msg.String() {
	case "up", "k":
		c.moveFocus(-1)
		return true, nil

	case "down", "j":
		c.moveFocus(1)
		return true, nil
	}

	return false, nil
}

func (c *LightControlComponent) handleMouseClick(msg tea.MouseClickMsg) (bool, tea.Cmd) {
	// Try routing the click to each field - the field's RouteEvent
	// will check its own zones (including row zones for multi-row fields)
	for i, fieldIdx := range c.focusableFields {
		field := c.getFieldByIndex(fieldIdx)
		if field == nil {
			continue
		}

		// Let the field handle the click - it knows its own zone structure
		if handled, cmd := field.RouteEvent(msg); handled {
			// Focus this field
			c.focusIndex = i
			c.updateChildFocus()
			return true, c.wrapFieldCmd(cmd, fieldIdx)
		}
	}

	return false, nil
}

// moveFocus moves focus by delta, staying within focusable fields.
func (c *LightControlComponent) moveFocus(delta int) {
	if len(c.focusableFields) == 0 {
		return
	}

	newIndex := c.focusIndex + delta
	if newIndex < 0 {
		newIndex = 0
	}
	if newIndex >= len(c.focusableFields) {
		newIndex = len(c.focusableFields) - 1
	}

	if newIndex != c.focusIndex {
		c.focusIndex = newIndex
		c.updateChildFocus()
	}
}

// FieldHeight returns the number of rows this component takes up.
func (c *LightControlComponent) FieldHeight() int {
	height := 1 // Header row (name + power + identify)

	if c.state.HasDimming {
		height++ // Brightness
	}
	if c.state.HasColorTemp {
		height++ // Color temp
	}
	if c.state.HasColor {
		height += c.colorWheel.FieldHeight() // Color wheel (multi-row)
		height += c.rgbControl.FieldHeight() // RGB (3 rows)
		height += c.hslControl.FieldHeight() // HSL (3 rows)
		height += c.hsvControl.FieldHeight() // HSV (3 rows)
	}
	if c.state.HasEffects {
		height += c.effectSelect.FieldHeight()
	}
	if c.state.HasGradient {
		height += c.gradientEditor.FieldHeight()
	}

	return height
}

// ViewControl renders the light control component.
func (c *LightControlComponent) ViewControl() string {
	var out strings.Builder

	// Header row: [●] Light Name [ID]
	out.WriteString(c.renderHeaderRow())

	// Brightness slider
	if c.state.HasDimming {
		out.WriteString("\n")
		out.WriteString(c.renderFieldRow(2, c.brightnessSlider))
	}

	// Color temp slider
	if c.state.HasColorTemp {
		out.WriteString("\n")
		out.WriteString(c.renderFieldRow(3, c.colorTempSlider))
	}

	// Color controls
	if c.state.HasColor {
		// Color wheel
		out.WriteString("\n")
		out.WriteString(c.renderMultiRowField(4, c.colorWheel))

		// RGB
		out.WriteString("\n")
		out.WriteString(c.renderMultiRowField(5, c.rgbControl))

		// HSL
		out.WriteString("\n")
		out.WriteString(c.renderMultiRowField(6, c.hslControl))

		// HSV
		out.WriteString("\n")
		out.WriteString(c.renderMultiRowField(7, c.hsvControl))
	}

	// Effect selector
	if c.state.HasEffects {
		out.WriteString("\n")
		out.WriteString(c.renderFieldRow(8, c.effectSelect))
	}

	// Gradient editor
	if c.state.HasGradient {
		out.WriteString("\n")
		out.WriteString(c.renderMultiRowField(9, c.gradientEditor))
	}

	return out.String()
}

// renderHeaderRow renders the header with power toggle, name, and identify button.
func (c *LightControlComponent) renderHeaderRow() string {
	// Focus indicator
	focusPrefix := "  "
	if c.isFocusedField(0) || c.isFocusedField(1) {
		focusPrefix = "> "
	}

	// Power toggle
	powerView := c.powerToggle.ViewControl()

	// Name with styling based on power state
	nameStyle := c.Styles.Base
	if !c.state.On {
		nameStyle = c.Styles.Dimmed
	}
	nameView := nameStyle.Render(c.Name)

	// Identify button
	identifyView := c.identifyButton.ViewControl()

	return fmt.Sprintf("%s%s %s %s", focusPrefix, powerView, nameView, identifyView)
}

// renderFieldRow renders a single-row field with focus indicator (no label - minimal labeling).
func (c *LightControlComponent) renderFieldRow(fieldIndex int, cr ControlRenderer) string {
	focusPrefix := "  "
	if c.isFocusedField(fieldIndex) {
		focusPrefix = "> "
	}

	return focusPrefix + cr.ViewControl()
}

// renderMultiRowField renders a multi-row field with focus indicator on first row (no label - minimal labeling).
func (c *LightControlComponent) renderMultiRowField(fieldIndex int, cr ControlRenderer) string {
	focusPrefix := "  "
	if c.isFocusedField(fieldIndex) {
		focusPrefix = "> "
	}

	control := cr.ViewControl()
	controlLines := strings.Split(control, "\n")

	var out strings.Builder
	for i, line := range controlLines {
		if i > 0 {
			out.WriteString("\n")
		}

		if i == 0 {
			out.WriteString(focusPrefix + line)
		} else {
			// Indent continuation lines to align with first line
			out.WriteString("  " + line)
		}
	}

	return out.String()
}

// isFocusedField checks if the given field index is currently focused.
func (c *LightControlComponent) isFocusedField(fieldIndex int) bool {
	if c.focusIndex >= 0 && c.focusIndex < len(c.focusableFields) {
		return c.focusableFields[c.focusIndex] == fieldIndex
	}
	return false
}

// View renders the complete light control component.
func (c *LightControlComponent) View() string {
	return c.ViewControl()
}

// Children returns all child components.
func (c *LightControlComponent) Children() []component.Component {
	return []component.Component{
		c.powerToggle,
		c.identifyButton,
		c.brightnessSlider,
		c.colorTempSlider,
		c.colorWheel,
		c.rgbControl,
		c.hslControl,
		c.effectSelect,
		c.gradientEditor,
	}
}

// CanFocus returns true since this component is always focusable.
func (c *LightControlComponent) CanFocus() bool {
	return true
}

// GetLightID returns the light ID this component controls.
func (c *LightControlComponent) GetLightID() string {
	return c.LightID
}

// GetBridgeID returns the bridge ID this component's light belongs to.
func (c *LightControlComponent) GetBridgeID() string {
	return c.BridgeID
}

// GetState returns the current state.
func (c *LightControlComponent) GetState() LightControlState {
	return c.state
}

// SetColorMode sets whether the light is in color temp mode (mirek valid) or XY color mode.
func (c *LightControlComponent) SetColorMode(mirekValid bool) {
	c.state.MirekValid = mirekValid
	c.colorTempSlider.Inactive = !mirekValid
	c.colorWheel.Inactive = mirekValid
}

// SwitchToColorMode switches to XY color mode.
func (c *LightControlComponent) SwitchToColorMode() {
	c.SetColorMode(false)
}

// SwitchToColorTempMode switches to color temperature mode.
func (c *LightControlComponent) SwitchToColorTempMode() {
	c.SetColorMode(true)
}

// WheelRenderer returns the color wheel for external access if needed.
func (c *LightControlComponent) WheelRenderer() *components.ColorWheel {
	return c.colorWheel.Wheel
}

// FocusFirst focuses the first focusable field.
func (c *LightControlComponent) FocusFirst() {
	if len(c.focusableFields) > 0 {
		c.focusIndex = 0
		c.updateChildFocus()
	}
}

// FocusLast focuses the last focusable field.
func (c *LightControlComponent) FocusLast() {
	if len(c.focusableFields) > 0 {
		c.focusIndex = len(c.focusableFields) - 1
		c.updateChildFocus()
	}
}

// LightControlValue is the value type for light control change messages.
// It provides context about which sub-control triggered the change.
type LightControlValue struct {
	ControlType string // "on", "identify", "brightness", "colortemp", "color", "effect", "gradient-mode", "gradient-points"
	Value       any    // The actual value (same as the sub-component's value type)
}
