package field

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/charmbracelet/lipgloss/v2"
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

	// colorsDirty is set when user is actively editing colors (prevents SSE overwrites)
	colorsDirty bool
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
	c.powerToggle.OnLabel = ""
	c.powerToggle.OffLabel = ""

	c.identifyButton = NewButtonComponent(baseID+":identify", "", "ID", styles, zones)

	c.brightnessSlider = NewBrightnessSliderComponent(baseID+":brightness", "Brightness", 100, styles, zones)

	c.colorTempSlider = NewColorTempSliderComponent(baseID+":colortemp", "Color Temp", 350, 153, 500, styles, zones)

	c.colorWheel = NewColorWheelComponent(baseID+":color-wheel", "Color", 0.3127, 0.329, styles, zones)
	// Increase color wheel radii to match the height of all sliders
	c.colorWheel.Wheel.RadiusX = 10
	c.colorWheel.Wheel.RadiusY = 5

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
// If colorsDirty is true, color-related controls are not updated (preserves user edits).
func (c *LightControlComponent) SetState(state LightControlState) {
	// Always update capabilities
	prevHasColor := c.state.HasColor
	prevHasColorTemp := c.state.HasColorTemp
	prevHasEffects := c.state.HasEffects
	prevHasGradient := c.state.HasGradient

	c.state.HasDimming = state.HasDimming
	c.state.HasColor = state.HasColor
	c.state.HasColorTemp = state.HasColorTemp
	c.state.HasEffects = state.HasEffects
	c.state.HasGradient = state.HasGradient

	// Update power toggle
	c.state.On = state.On
	c.powerToggle.SetValue(state.On)

	// Update brightness
	c.state.Brightness = state.Brightness
	c.brightnessSlider.SetValue(state.Brightness)
	c.colorTempSlider.Brightness = state.Brightness
	c.colorWheel.Brightness = state.Brightness

	// Skip color-related updates if user is actively editing
	if !c.colorsDirty {
		// Update color temp
		c.state.ColorTemp = state.ColorTemp
		c.state.MinMirek = state.MinMirek
		c.state.MaxMirek = state.MaxMirek
		c.state.MirekValid = state.MirekValid
		c.colorTempSlider.SetValue(state.ColorTemp)
		c.colorTempSlider.Min = state.MinMirek
		c.colorTempSlider.Max = state.MaxMirek
		c.colorTempSlider.Inactive = !state.MirekValid

		// Update color
		c.state.ColorX = state.ColorX
		c.state.ColorY = state.ColorY
		c.brightnessSlider.ColorX = state.ColorX
		c.brightnessSlider.ColorY = state.ColorY
		if state.MirekValid {
			c.brightnessSlider.ColorTempMirek = state.ColorTemp
		}

		// Sync all color controls to the new XY values
		c.syncColorControlsToPoint(state.ColorX, state.ColorY)

		// Update inactive state
		c.colorWheel.Inactive = state.MirekValid
		c.rgbControl.Inactive = state.MirekValid
		c.hslControl.Inactive = state.MirekValid
		c.hsvControl.Inactive = state.MirekValid
	}

	// Update effect selector
	c.state.EffectIndex = state.EffectIndex
	c.state.Effects = state.Effects
	if len(state.Effects) > 0 {
		options := make([]Option, len(state.Effects))
		for i, effect := range state.Effects {
			options[i] = Option{Label: effect, Value: i}
		}
		c.effectSelect.SetOptions(options)
		c.effectSelect.SetValue(state.EffectIndex)
	}

	// Update gradient editor (only if not editing gradient)
	if !c.gradientEditor.Editing {
		c.state.GradientPoints = state.GradientPoints
		c.state.GradientMode = state.GradientMode
		c.state.GradientModes = state.GradientModes
		c.state.MaxGradientPoints = state.MaxGradientPoints
		if state.HasGradient && len(state.GradientPoints) > 0 {
			c.gradientEditor.Points = state.GradientPoints
			c.gradientEditor.MaxPoints = state.MaxGradientPoints
		}
	}

	// Update focusable fields if capabilities changed
	if prevHasColor != state.HasColor || prevHasColorTemp != state.HasColorTemp ||
		prevHasEffects != state.HasEffects || prevHasGradient != state.HasGradient {
		c.updateFocusableFields()
	}
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

// syncColorControlsToPoint syncs all color controls (wheel, RGB, HSL, HSV) to the given XY point.
func (c *LightControlComponent) syncColorControlsToPoint(x, y float64) {
	// Update color wheel
	c.colorWheel.SetColor(x, y)

	// Convert XY to RGB
	r, g, b := ui.XyToRGB(x, y, 100)
	c.rgbControl.SetRGB(int(r), int(g), int(b))

	// Convert RGB to HSL
	h, s, l := ui.RGBToHSL(r, g, b)
	c.hslControl.SetHSL(h, s, l)

	// Convert RGB to HSV
	hh, ss, v := ui.RGBToHSV(r, g, b)
	c.hsvControl.SetHSV(hh, ss, v)
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
		// Set colorsDirty for color control captures
		if c.focusIndex >= 0 && c.focusIndex < len(c.focusableFields) {
			fieldIdx := c.focusableFields[c.focusIndex]
			if fieldIdx >= 4 && fieldIdx <= 7 {
				c.colorsDirty = true
			}
		}
	}
	if _, ok := msg.(EndCaptureMsg); ok {
		c.captureField = -1
		c.colorsDirty = false // Allow SSE updates again
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
		// Track gradient selection before routing
		prevGradientSelection := c.gradientEditor.SelectedIndex

		if handled, cmd := focusedField.RouteEvent(msg); handled {
			fieldIdx := c.focusableFields[c.focusIndex]

			// For color controls, do immediate sync on key events too
			if fieldIdx >= 4 && fieldIdx <= 7 {
				if _, ok := msg.(tea.KeyMsg); ok {
					c.colorsDirty = true
					c.syncColorFromField(fieldIdx)
				}
			}

			// Sync color controls when gradient swatch selection changes
			if fieldIdx == 9 && c.gradientEditor.SelectedIndex != prevGradientSelection {
				if c.gradientEditor.SelectedIndex >= 0 && c.gradientEditor.SelectedIndex < len(c.gradientEditor.Points) {
					pt := c.gradientEditor.Points[c.gradientEditor.SelectedIndex]
					c.syncColorControlsToPoint(pt.X, pt.Y)
				}
			}

			return true, c.wrapFieldCmd(cmd, fieldIdx)
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

			// Sync color controls when gradient point changes
			if fieldIndex == 9 {
				if gv, ok := fcm.Value.(GradientValue); ok {
					// Update internal state
					c.state.GradientPoints = gv.Points
					// Sync color controls to show the selected gradient point's color
					if c.gradientEditor.SelectedIndex >= 0 && c.gradientEditor.SelectedIndex < len(gv.Points) {
						pt := gv.Points[c.gradientEditor.SelectedIndex]
						c.syncColorControlsToPoint(pt.X, pt.Y)
					}
				}
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

			// For color controls, do immediate sync to prevent lag
			if fieldIdx >= 4 && fieldIdx <= 7 {
				c.colorsDirty = true // Prevent SSE overwrites while editing
				c.syncColorFromField(fieldIdx)
			}

			return true, c.wrapFieldCmd(cmd, fieldIdx)
		}
	}

	return false, nil
}

// syncColorFromField syncs all color controls from the given field's current values.
func (c *LightControlComponent) syncColorFromField(fieldIdx int) {
	if c.updatingColors {
		return
	}
	c.updatingColors = true
	defer func() { c.updatingColors = false }()

	var x, y float64

	switch fieldIdx {
	case 4: // Color wheel
		x = c.colorWheel.ColorX
		y = c.colorWheel.ColorY

	case 5: // RGB
		x, y = ui.RGBIntToXY(c.rgbControl.Red, c.rgbControl.Green, c.rgbControl.Blue)

	case 6: // HSL
		r, g, b := ui.HSLToRGB(c.hslControl.Hue, c.hslControl.Saturation, c.hslControl.Lightness)
		x, y = ui.RGBToXY(r, g, b)

	case 7: // HSV
		r, g, b := ui.HsvToRGB(c.hsvControl.Hue, c.hsvControl.Saturation, c.hsvControl.Value)
		x, y = ui.RGBToXY(r, g, b)

	default:
		return
	}

	// Update internal state
	c.state.ColorX = x
	c.state.ColorY = y
	c.brightnessSlider.ColorX = x
	c.brightnessSlider.ColorY = y

	// Convert XY to RGB for syncing other controls
	r, g, b := ui.XyToRGB(x, y, 100)

	// Sync controls (skip the source)
	if fieldIdx != 4 {
		c.colorWheel.SetColor(x, y)
	}
	if fieldIdx != 5 {
		c.rgbControl.SetRGB(int(r), int(g), int(b))
	}
	if fieldIdx != 6 {
		h, s, l := ui.RGBToHSL(r, g, b)
		c.hslControl.SetHSL(h, s, l)
	}
	if fieldIdx != 7 {
		hh, ss, v := ui.RGBToHSV(r, g, b)
		c.hsvControl.SetHSV(hh, ss, v)
	}
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
	// Calculate left column height (sliders)
	leftHeight := 0

	if c.state.HasDimming {
		leftHeight++ // Brightness
	}
	if c.state.HasColorTemp {
		leftHeight++ // Color temp
	}
	if c.state.HasColor {
		leftHeight += c.rgbControl.FieldHeight() // RGB (3 rows)
		leftHeight += c.hslControl.FieldHeight() // HSL (3 rows)
		leftHeight += c.hsvControl.FieldHeight() // HSV (3 rows)
	}

	// Right column is color wheel
	rightHeight := 0
	if c.state.HasColor {
		rightHeight = c.colorWheel.FieldHeight()
	}

	// Content height is max of left and right
	contentHeight := leftHeight
	if rightHeight > contentHeight {
		contentHeight = rightHeight
	}

	// Add 2 for border (top + bottom)
	height := contentHeight + 2

	// Add rows below border
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
	// Build left column (sliders)
	var leftLines []string

	// Brightness slider
	if c.state.HasDimming {
		leftLines = append(leftLines, c.renderFieldRow(2, c.brightnessSlider))
	}

	// Color temp slider
	if c.state.HasColorTemp {
		leftLines = append(leftLines, c.renderFieldRow(3, c.colorTempSlider))
	}

	// RGB (3 rows)
	if c.state.HasColor {
		rgbLines := strings.Split(c.renderMultiRowField(5, c.rgbControl), "\n")
		leftLines = append(leftLines, rgbLines...)
	}

	// HSL (3 rows)
	if c.state.HasColor {
		hslLines := strings.Split(c.renderMultiRowField(6, c.hslControl), "\n")
		leftLines = append(leftLines, hslLines...)
	}

	// HSV (3 rows)
	if c.state.HasColor {
		hsvLines := strings.Split(c.renderMultiRowField(7, c.hsvControl), "\n")
		leftLines = append(leftLines, hsvLines...)
	}

	// Build right column (color wheel) if we have color capability
	var rightLines []string
	if c.state.HasColor {
		wheelView := c.renderMultiRowField(4, c.colorWheel)
		rightLines = strings.Split(wheelView, "\n")
	}

	// Join left and right columns side by side
	var contentLines []string
	maxLines := len(leftLines)
	if len(rightLines) > maxLines {
		maxLines = len(rightLines)
	}

	// Find max width of left column for padding
	leftWidth := 0
	for _, line := range leftLines {
		w := lipgloss.Width(line)
		if w > leftWidth {
			leftWidth = w
		}
	}

	// Find max width of right column for padding
	rightWidth := 0
	for _, line := range rightLines {
		w := lipgloss.Width(line)
		if w > rightWidth {
			rightWidth = w
		}
	}

	for i := 0; i < maxLines; i++ {
		var left, right string
		if i < len(leftLines) {
			left = leftLines[i]
		}
		if i < len(rightLines) {
			right = rightLines[i]
		}

		// Pad columns to consistent width
		leftPadded := left + strings.Repeat(" ", leftWidth-lipgloss.Width(left))
		rightPadded := right + strings.Repeat(" ", rightWidth-lipgloss.Width(right))
		contentLines = append(contentLines, leftPadded+" "+rightPadded)
	}

	// Calculate total content width
	contentWidth := leftWidth + 1 + rightWidth

	// Build border manually with name in top and ID button in bottom
	borderColor := c.Styles.Theme.Border
	borderStyle := lipgloss.NewStyle().Foreground(borderColor)

	nameStyle := c.Styles.Base
	if !c.state.On {
		nameStyle = c.Styles.Dimmed
	}
	nameStr := nameStyle.Render(c.Name)

	// On/Off toggle in top border (no label)
	toggleStr := c.powerToggle.ViewControl()

	// ID button for bottom border
	idButton := c.identifyButton.ViewControl()
	idWidth := lipgloss.Width(idButton)

	// Top border: ╭─ ○ Name ───────────────╮
	topLeft := borderStyle.Render("╭")
	topRight := borderStyle.Render("╮")
	topContent := toggleStr + " " + nameStr
	topContentWidth := lipgloss.Width(topContent)
	leftDashWidth := 1

	// Bottom border: ╰───────────────── [ID] ─╯
	bottomLeft := borderStyle.Render("╰")
	bottomRight := borderStyle.Render("╯")
	bottomContentWidth := idWidth
	neededTopWidth := leftDashWidth + topContentWidth
	if neededTopWidth > contentWidth {
		contentWidth = neededTopWidth
	}
	if bottomContentWidth > contentWidth {
		contentWidth = bottomContentWidth
	}
	topFillWidth := contentWidth - neededTopWidth
	if topFillWidth < 0 {
		topFillWidth = 0
	}
	topFillLeft := borderStyle.Render(strings.Repeat("─", leftDashWidth))
	topFillRight := borderStyle.Render(strings.Repeat("─", topFillWidth))
	topBorder := topLeft + topFillLeft + topContent + topFillRight + topRight
	bottomFillLeft := contentWidth - bottomContentWidth
	if bottomFillLeft < 0 {
		bottomFillLeft = 0
	}
	bottomFill := borderStyle.Render(strings.Repeat("─", bottomFillLeft))
	bottomBorder := bottomLeft + bottomFill + idButton + bottomRight

	// Content rows with side borders
	var result strings.Builder
	result.WriteString(topBorder)
	for _, line := range contentLines {
		result.WriteString("\n")
		result.WriteString(borderStyle.Render("│"))
		lineWidth := lipgloss.Width(line)
		if lineWidth < contentWidth {
			line = line + strings.Repeat(" ", contentWidth-lineWidth)
		}
		result.WriteString(line)
		result.WriteString(borderStyle.Render("│"))
	}
	result.WriteString("\n")
	result.WriteString(bottomBorder)

	// Add effect selector below border if present
	if c.state.HasEffects {
		result.WriteString("\n")
		result.WriteString(c.renderFieldRow(8, c.effectSelect))
	}

	// Add gradient editor below border if present
	if c.state.HasGradient {
		result.WriteString("\n")
		result.WriteString(c.renderMultiRowField(9, c.gradientEditor))
	}

	return result.String()
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
