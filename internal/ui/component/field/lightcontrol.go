package field

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/charmbracelet/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/kluzzebass/lazyhue/internal/hue"
	"github.com/kluzzebass/lazyhue/internal/ui"
	"github.com/kluzzebass/lazyhue/internal/ui/component"
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
	gradientMode     *SelectComponent
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

	// showIdentify controls whether the Identify button is rendered/focusable.
	showIdentify bool
}

// NewLightControlComponent creates a new unified light control component.
func NewLightControlComponent(lightID, bridgeID, name string, styles *ui.Styles, zones *zone.Manager) *LightControlComponent {
	baseID := fmt.Sprintf("lc:%s", lightID)
	return NewLightControlComponentWithID(baseID, lightID, bridgeID, name, styles, zones)
}

// NewLightControlComponentWithID creates a new unified light control component with a custom base ID.
func NewLightControlComponentWithID(baseID, lightID, bridgeID, name string, styles *ui.Styles, zones *zone.Manager) *LightControlComponent {
	c := &LightControlComponent{
		BaseField:    NewBaseField(baseID, "", styles, zones),
		LightID:      lightID,
		BridgeID:     bridgeID,
		Name:         name,
		captureField: -1,
		showIdentify: true,
	}

	// Create child components with prefixed IDs
	c.powerToggle = NewToggleComponent(baseID+":on", "", true, styles, zones)
	c.powerToggle.OnLabel = ""
	c.powerToggle.OffLabel = ""
	c.powerToggle.BracketFocusOnly = true

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
	c.effectSelect.CycleOnly = true

	c.gradientMode = NewSelectComponent(baseID+":gradient-mode", "Gradient", 0, nil, styles, zones)
	c.gradientMode.CycleOnly = true

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
	c.gradientMode.SetParent(c)
	c.gradientEditor.SetParent(c)

	// Initialize focusable fields (will be updated by SetState)
	c.updateFocusableFields()

	return c
}

// SetState updates all internal state and child components.
// If colorsDirty is true, color-related controls are not updated (preserves user edits).
func (c *LightControlComponent) SetState(state LightControlState) {
	// Always update capabilities
	prevHasDimming := c.state.HasDimming
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
	// Keep the wheel readable, but still reflect brightness changes.
	if state.Brightness < 35 {
		c.colorWheel.Brightness = 35
	} else {
		c.colorWheel.Brightness = state.Brightness
	}

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
		if state.HasGradient && len(state.GradientPoints) > 0 {
			idx := c.gradientEditor.SelectedIndex
			if idx < 0 || idx >= len(state.GradientPoints) {
				idx = 0
				c.gradientEditor.SelectedIndex = 0
			}
			pt := state.GradientPoints[idx]
			c.state.ColorX = pt.X
			c.state.ColorY = pt.Y
		} else {
			c.state.ColorX = state.ColorX
			c.state.ColorY = state.ColorY
		}
		c.brightnessSlider.ColorX = c.state.ColorX
		c.brightnessSlider.ColorY = c.state.ColorY
		if state.MirekValid {
			c.brightnessSlider.ColorTempMirek = state.ColorTemp
		}

		// Sync all color controls to the active XY values
		c.syncColorControlsToPoint(c.state.ColorX, c.state.ColorY)

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
			options[i] = Option{Label: hue.EffectDisplayName(effect), Value: i}
		}
		c.effectSelect.SetOptions(options)
		c.effectSelect.SetValue(state.EffectIndex)
	}

	// Update gradient mode selector
	c.state.GradientMode = state.GradientMode
	c.state.GradientModes = state.GradientModes
	if len(state.GradientModes) > 0 {
		options := make([]Option, len(state.GradientModes))
		for i, mode := range state.GradientModes {
			options[i] = Option{Label: gradientModeDisplayName(mode), Value: i}
		}
		c.gradientMode.SetOptions(options)
		c.gradientMode.SetValue(state.GradientMode)
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
			c.gradientEditor.MinPoints = 2
		} else if state.HasColor {
			c.gradientEditor.Points = []GradientPoint{{X: state.ColorX, Y: state.ColorY}}
			c.gradientEditor.MaxPoints = 1
			c.gradientEditor.MinPoints = 1
		}

		if len(c.gradientEditor.Points) > 0 &&
			(c.gradientEditor.SelectedIndex < 0 || c.gradientEditor.SelectedIndex >= len(c.gradientEditor.Points)) {
			c.gradientEditor.SelectedIndex = 0
		}
	}

	// Update focusable fields if capabilities changed
	if prevHasDimming != state.HasDimming || prevHasColor != state.HasColor ||
		prevHasColorTemp != state.HasColorTemp || prevHasEffects != state.HasEffects ||
		prevHasGradient != state.HasGradient {
		c.updateFocusableFields()
	}
}

// updateFocusableFields rebuilds the list of focusable field indices.
func (c *LightControlComponent) updateFocusableFields() {
	c.focusableFields = []int{}

	// Order: Power, Brightness, ColorTemp, RGB, HSL, HSV, Effect, GradientMode, ColorWheel, Gradient, Identify
	c.focusableFields = append(c.focusableFields, 0) // Power toggle

	if c.state.HasDimming {
		c.focusableFields = append(c.focusableFields, 2) // Brightness
	}
	if c.state.HasColorTemp {
		c.focusableFields = append(c.focusableFields, 3) // Color temp
	}
	if c.state.HasColor {
		c.focusableFields = append(c.focusableFields, 5) // RGB
		c.focusableFields = append(c.focusableFields, 6) // HSL
		c.focusableFields = append(c.focusableFields, 7) // HSV
	}
	if c.state.HasEffects {
		c.focusableFields = append(c.focusableFields, 8) // Effect
	}
	if c.state.HasGradient && len(c.state.GradientModes) > 0 {
		c.focusableFields = append(c.focusableFields, 10) // Gradient mode
	}
	if c.state.HasColor {
		c.focusableFields = append(c.focusableFields, 4) // Color wheel
	}
	if c.state.HasGradient {
		c.focusableFields = append(c.focusableFields, 9) // Gradient
	}
	if c.showIdentify {
		c.focusableFields = append(c.focusableFields, 1) // Identify button
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

	// Update brightness slider color display (XY mode)
	if c.state.MirekValid {
		c.brightnessSlider.ColorX = 0
		c.brightnessSlider.ColorY = 0
		c.brightnessSlider.ColorTempMirek = c.state.ColorTemp
	} else {
		c.brightnessSlider.ColorX = x
		c.brightnessSlider.ColorY = y
		c.brightnessSlider.ColorTempMirek = 0
	}

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
	case 10:
		return c.gradientMode
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
	c.gradientMode.Blur()
	c.gradientEditor.Blur()

	// Focus the current field
	if field := c.getFocusedField(); field != nil {
		field.Focus()
		switch focused := field.(type) {
		case *RGBComponent:
			if focused.SliderFocus < 0 || focused.SliderFocus > 2 {
				focused.SliderFocus = 0
			}
		case *HSLComponent:
			if focused.SliderFocus < 0 || focused.SliderFocus > 2 {
				focused.SliderFocus = 0
			}
		case *HSVComponent:
			if focused.SliderFocus < 0 || focused.SliderFocus > 2 {
				focused.SliderFocus = 0
			}
		}
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
	// Handle capture field messages
	if captureMsg, ok := msg.(StartCaptureMsg); ok {
		if fieldIdx := c.logicalFieldIndexForID(captureMsg.FieldID); fieldIdx >= 0 {
			c.captureField = c.indexForField(fieldIdx)
			if fieldIdx >= 4 && fieldIdx <= 7 {
				c.colorsDirty = true
			}
		}
	}
	if captureMsg, ok := msg.(EndCaptureMsg); ok {
		if fieldIdx := c.logicalFieldIndexForID(captureMsg.FieldID); fieldIdx >= 0 {
			c.captureField = -1
			c.colorsDirty = false // Allow SSE updates again
		}
	}

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
		if msg, ok := msg.(tea.KeyMsg); ok {
			fieldIdx := c.focusableFields[c.focusIndex]
			if c.state.MirekValid && fieldIdx >= 4 && fieldIdx <= 7 {
				switch msg.String() {
				case "left", "right", "h", "l":
					c.SwitchToColorMode()
				}
			}
		}

		if handled, cmd := focusedField.RouteEvent(msg); handled {
			fieldIdx := c.focusableFields[c.focusIndex]

			// For color controls, do immediate sync on value-changing keys only
			if fieldIdx >= 4 && fieldIdx <= 7 {
				if keyMsg, ok := msg.(tea.KeyMsg); ok {
					switch keyMsg.String() {
					case "left", "right", "h", "l":
						c.colorsDirty = true
						c.syncColorFromField(fieldIdx)
					}
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
		switch msg.(type) {
		case StartCaptureMsg:
			c.captureField = c.indexForField(fieldIndex)
			if fieldIndex >= 4 && fieldIndex <= 7 {
				c.colorsDirty = true
			}
			return msg
		case EndCaptureMsg:
			c.captureField = -1
			c.colorsDirty = false
			return msg
		}
		switch typed := msg.(type) {
		case FieldChangedMsg:
			return c.processFieldChanged(typed, fieldIndex)
		case tea.BatchMsg:
			if len(typed) == 0 {
				return msg
			}
			updated := make(tea.BatchMsg, 0, len(typed))
			for _, batchCmd := range typed {
				if batchCmd == nil {
					continue
				}
				updated = append(updated, c.wrapFieldCmd(batchCmd, fieldIndex))
			}
			return updated
		default:
			return msg
		}
	}
}

func (c *LightControlComponent) indexForField(fieldIndex int) int {
	for i, field := range c.focusableFields {
		if field == fieldIndex {
			return i
		}
	}
	return -1
}

func (c *LightControlComponent) logicalFieldIndexForID(fieldID string) int {
	switch fieldID {
	case c.powerToggle.ID:
		return 0
	case c.identifyButton.ID:
		return 1
	case c.brightnessSlider.ID:
		return 2
	case c.colorTempSlider.ID:
		return 3
	case c.colorWheel.ID:
		return 4
	case c.rgbControl.ID:
		return 5
	case c.hslControl.ID:
		return 6
	case c.hsvControl.ID:
		return 7
	case c.effectSelect.ID:
		return 8
	case c.gradientEditor.ID:
		return 9
	case c.gradientMode.ID:
		return 10
	default:
		return -1
	}
}

func (c *LightControlComponent) processFieldChanged(fcm FieldChangedMsg, fieldIndex int) tea.Msg {
	// Handle reactive color updates
	c.handleReactiveColorUpdate(fcm, fieldIndex)

	// Rewrite the field ID so the app handler can route it correctly
	if fieldIndex == 4 || fieldIndex == 5 || fieldIndex == 6 || fieldIndex == 7 {
		// Color wheel, RGB, HSL, or HSV -> emit as color or gradient point change
		if updated, points := c.updateSelectedGradientPoint(c.state.ColorX, c.state.ColorY); updated {
			fcm.FieldID = c.ID + ":gradient-points"
			fcm.Value = GradientValue{Points: points}
		} else {
			fcm.FieldID = c.ID + ":color"
			fcm.Value = ColorValue{X: c.state.ColorX, Y: c.state.ColorY}
		}
	}
	if fieldIndex == 9 {
		if gv, ok := fcm.Value.(GradientValue); ok {
			fcm.FieldID = c.ID + ":gradient-points"
			fcm.Value = gv
		}
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

// handleReactiveColorUpdate syncs all color controls when one changes.
func (c *LightControlComponent) handleReactiveColorUpdate(msg FieldChangedMsg, fieldIndex int) {
	if c.updatingColors {
		return // Prevent infinite loop
	}
	c.updatingColors = true
	defer func() { c.updatingColors = false }()

	switch fieldIndex {
	case 2: // Brightness slider
		if sv, ok := msg.Value.(SliderValue); ok {
			c.state.Brightness = sv.Value
			c.colorTempSlider.Brightness = sv.Value
			if sv.Value < 35 {
				c.colorWheel.Brightness = 35
			} else {
				c.colorWheel.Brightness = sv.Value
			}
		}

	case 3: // Color temp slider
		if sv, ok := msg.Value.(SliderValue); ok {
			c.SwitchToColorTempMode()
			c.state.ColorTemp = sv.Value
			c.colorTempSlider.SetValue(sv.Value)
			c.brightnessSlider.ColorX = 0
			c.brightnessSlider.ColorY = 0
			c.brightnessSlider.ColorTempMirek = sv.Value
		}

	case 4: // Color wheel
		if cv, ok := msg.Value.(ColorValue); ok {
			c.SwitchToColorMode()
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
			c.SwitchToColorMode()
			// Convert RGB to XY
			x, y := ui.RGBIntToXY(rv.Red, rv.Green, rv.Blue)
			c.state.ColorX = x
			c.state.ColorY = y

			// Update color wheel
			c.colorWheel.SetColor(x, y)
			hh, ss, _ := ui.RGBToHSV(uint8(rv.Red), uint8(rv.Green), uint8(rv.Blue))
			c.colorWheel.SetHueSatPosition(hh, ss)

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
			c.SwitchToColorMode()
			// Convert HSL to RGB, then to XY
			r, g, b := ui.HSLToRGB(hv.Hue, hv.Saturation, hv.Lightness)
			x, y := ui.RGBToXY(r, g, b)
			c.state.ColorX = x
			c.state.ColorY = y

			// Update color wheel
			c.colorWheel.SetColor(x, y)
			hh, ss, _ := ui.RGBToHSV(r, g, b)
			c.colorWheel.SetHueSatPosition(hh, ss)

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
			c.SwitchToColorMode()
			// Convert HSV to RGB, then to XY
			r, g, b := ui.HsvToRGB(hv.Hue, hv.Saturation, hv.Value)
			x, y := ui.RGBToXY(r, g, b)
			c.state.ColorX = x
			c.state.ColorY = y

			// Update color wheel
			c.colorWheel.SetColor(x, y)
			c.colorWheel.SetHueSatPosition(hv.Hue, hv.Saturation)

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

func (c *LightControlComponent) updateSelectedGradientPoint(x, y float64) (bool, []GradientPoint) {
	idx := c.gradientEditor.SelectedIndex
	if idx < 0 || idx >= len(c.gradientEditor.Points) {
		return false, nil
	}

	c.gradientEditor.Points[idx] = GradientPoint{X: x, Y: y}
	c.state.GradientPoints = make([]GradientPoint, len(c.gradientEditor.Points))
	copy(c.state.GradientPoints, c.gradientEditor.Points)

	if !c.state.HasGradient {
		return false, c.state.GradientPoints
	}

	return true, c.state.GradientPoints
}

func (c *LightControlComponent) handleKey(msg tea.KeyMsg) (bool, tea.Cmd) {
	// Child components return false when they want parent to handle navigation
	// (e.g., at edge of multi-slider control), so we handle it here
	switch msg.String() {
	case "up", "k":
		if c.getFocusedField() == nil {
			c.FocusFirst()
			return true, nil
		}
		c.moveFocus(-1)
		return true, nil

	case "down", "j":
		if c.getFocusedField() == nil {
			c.FocusFirst()
			return true, nil
		}
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

		// Switch back to color mode when interacting with color controls
		if c.state.MirekValid && fieldIdx >= 4 && fieldIdx <= 7 {
			c.SwitchToColorMode()
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

			// Sync color controls when gradient swatch selection changes
			if fieldIdx == 9 {
				if c.gradientEditor.SelectedIndex >= 0 && c.gradientEditor.SelectedIndex < len(c.gradientEditor.Points) {
					pt := c.gradientEditor.Points[c.gradientEditor.SelectedIndex]
					c.syncColorControlsToPoint(pt.X, pt.Y)
				}
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
	var hue, sat int
	useHueSat := false

	switch fieldIdx {
	case 4: // Color wheel
		c.SwitchToColorMode()
		x = c.colorWheel.ColorX
		y = c.colorWheel.ColorY

	case 5: // RGB
		c.SwitchToColorMode()
		x, y = ui.RGBIntToXY(c.rgbControl.Red, c.rgbControl.Green, c.rgbControl.Blue)
		hue, sat, _ = ui.RGBToHSV(uint8(c.rgbControl.Red), uint8(c.rgbControl.Green), uint8(c.rgbControl.Blue))
		useHueSat = true

	case 6: // HSL
		c.SwitchToColorMode()
		x, y = ui.HSLToXY(c.hslControl.Hue, c.hslControl.Saturation, c.hslControl.Lightness)
		r, g, b := ui.HSLToRGB(c.hslControl.Hue, c.hslControl.Saturation, c.hslControl.Lightness)
		hue, sat, _ = ui.RGBToHSV(r, g, b)
		useHueSat = true

	case 7: // HSV
		c.SwitchToColorMode()
		x, y = ui.HSVToXY(c.hsvControl.Hue, c.hsvControl.Saturation, c.hsvControl.Value)
		hue = c.hsvControl.Hue
		sat = c.hsvControl.Saturation
		useHueSat = true

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
		if useHueSat {
			c.colorWheel.SetHueSatPosition(hue, sat)
		}
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

	for newIndex >= 0 && newIndex < len(c.focusableFields) && !c.isFieldVisible(c.focusableFields[newIndex]) {
		newIndex += delta
		if newIndex < 0 {
			newIndex = 0
			break
		}
		if newIndex >= len(c.focusableFields) {
			newIndex = len(c.focusableFields) - 1
			break
		}
	}

	if newIndex != c.focusIndex {
		c.focusIndex = newIndex
		c.updateChildFocus()
	}
}

func (c *LightControlComponent) isFieldVisible(fieldIndex int) bool {
	switch fieldIndex {
	case 2:
		return c.state.HasDimming
	case 3:
		return c.state.HasColorTemp
	case 4, 5, 6, 7:
		return c.state.HasColor
	case 8:
		return c.state.HasEffects
	case 9:
		return c.state.HasGradient
	case 10:
		return c.state.HasGradient && len(c.state.GradientModes) > 0
	default:
		return true
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
	rightPadding := 0
	for _, line := range rightLines {
		w := lipgloss.Width(line)
		if w > rightWidth {
			rightWidth = w
		}
	}
	if rightWidth > 0 {
		rightPadding = 2
	}
	rightWidthWithPadding := rightWidth + rightPadding

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
		rightPadded := right + strings.Repeat(" ", rightWidthWithPadding-lipgloss.Width(right))
		contentLines = append(contentLines, leftPadded+" "+rightPadded)
	}

	// Calculate total content width
	contentWidth := leftWidth + 1 + rightWidthWithPadding

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

	// Top right content: swatches
	idButton := ""
	idWidth := 0
	if c.showIdentify {
		idButton = c.identifyButton.ViewControl()
		idWidth = lipgloss.Width(idButton)
	}
	effectStr := ""
	effectWidth := 0
	if c.state.HasEffects {
		effectStr = c.effectSelect.ViewControl()
		effectWidth = lipgloss.Width(effectStr)
	}
	gradientModeStr := ""
	gradientModeWidth := 0
	if c.state.HasGradient && len(c.state.GradientModes) > 0 {
		gradientModeStr = c.gradientMode.ViewControl()
		gradientModeWidth = lipgloss.Width(gradientModeStr)
	}
	swatchStr := ""
	swatchWidth := 0
	if c.state.HasGradient {
		swatchStr = c.gradientEditor.ViewControl()
		swatchWidth = lipgloss.Width(swatchStr)
	}

	// Top border: ╭─ ○ Name ───────────────╮
	topLeft := borderStyle.Render("╭")
	topRight := borderStyle.Render("╮")
	leftDashWidth := 1
	rightTopWidth := 0
	if swatchWidth > 0 {
		rightTopWidth = swatchWidth + 1
	}

	availableNameWidth := contentWidth - leftDashWidth - lipgloss.Width(toggleStr) - 1 - rightTopWidth
	if availableNameWidth < 0 {
		availableNameWidth = 0
	}
	if lipgloss.Width(nameStr) > availableNameWidth {
		nameStr = ansi.Truncate(nameStr, availableNameWidth, "…")
	}

	topContent := toggleStr + "─" + nameStr
	topContentWidth := lipgloss.Width(topContent) + rightTopWidth

	// Bottom border: ╰─[Effect][Gradient]──────────[ID]─╯
	bottomLeft := borderStyle.Render("╰")
	bottomRight := borderStyle.Render("╯")
	bottomSideDash := 1
	leftGroup := ""
	if effectWidth > 0 {
		leftGroup = effectStr
	}
	if gradientModeWidth > 0 {
		if leftGroup != "" {
			leftGroup += borderStyle.Render("─")
		}
		leftGroup += gradientModeStr
	}
	leftGroupWidth := lipgloss.Width(leftGroup)
	bottomContentWidth := leftGroupWidth + idWidth + (bottomSideDash * 2)
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
	topBorder := topLeft + topFillLeft + topContent + topFillRight
	if swatchWidth > 0 {
		topBorder += borderStyle.Render("─") + swatchStr
	}
	topBorder += topRight
	bottomFillLeft := contentWidth - bottomContentWidth
	if bottomFillLeft < 0 {
		bottomFillLeft = 0
	}
	bottomFill := borderStyle.Render(strings.Repeat("─", bottomFillLeft))
	bottomSideDashStr := borderStyle.Render(strings.Repeat("─", bottomSideDash))
	var bottomBorder string
	if leftGroupWidth > 0 {
		bottomBorder = bottomLeft + bottomSideDashStr + leftGroup + bottomFill
	} else {
		bottomBorder = bottomLeft + bottomSideDashStr + bottomFill
	}
	bottomBorder += idButton + bottomSideDashStr + bottomRight

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

	return result.String()
}

// renderFieldRow renders a single-row field with focus indicator (no label - minimal labeling).
func (c *LightControlComponent) renderFieldRow(fieldIndex int, cr ControlRenderer) string {
	focusPrefix := "  "
	if c.isFocusedField(fieldIndex) {
		focusPrefix = "> "
	}

	row := focusPrefix + cr.ViewControl()
	if fieldIndex == 2 && c.isMonoControl() {
		row += " "
	}
	return row
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

func (c *LightControlComponent) isMonoControl() bool {
	return c.state.HasDimming && !c.state.HasColor && !c.state.HasColorTemp && !c.state.HasEffects && !c.state.HasGradient
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

// Blur clears focus from the component and all child controls.
func (c *LightControlComponent) Blur() {
	c.BaseField.Blur()
	c.focusIndex = -1
	c.updateChildFocus()
}

// Focus sets focus on the first available child control.
func (c *LightControlComponent) Focus() {
	c.BaseField.Focus()
	if len(c.focusableFields) == 0 {
		return
	}
	if c.focusIndex < 0 || c.focusIndex >= len(c.focusableFields) {
		c.focusIndex = 0
	}
	c.updateChildFocus()
}

// IsAtFirstFocusable returns true when focus is at the first field.
func (c *LightControlComponent) IsAtFirstFocusable() bool {
	if len(c.focusableFields) == 0 {
		return true
	}
	return c.focusIndex <= 0
}

// IsAtLastFocusable returns true when focus is at the last field.
func (c *LightControlComponent) IsAtLastFocusable() bool {
	if len(c.focusableFields) == 0 {
		return true
	}
	return c.focusIndex >= len(c.focusableFields)-1
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
		c.hsvControl,
		c.effectSelect,
		c.gradientMode,
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
	c.rgbControl.Inactive = mirekValid
	c.hslControl.Inactive = mirekValid
	c.hsvControl.Inactive = mirekValid
	if !mirekValid {
		c.brightnessSlider.ColorTempMirek = 0
	}
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
func (c *LightControlComponent) WheelRenderer() *ColorWheel {
	return c.colorWheel.Wheel
}

func gradientModeDisplayName(mode string) string {
	switch mode {
	case "interpolated_palette":
		return "Interpolated"
	case "interpolated_palette_mirrored":
		return "Mirrored"
	case "random_pixelated":
		return "Pixelated"
	case "segmented_palette":
		return "Segmented"
	default:
		return strings.ReplaceAll(mode, "_", " ")
	}
}


// SetIdentifyVisible controls whether the identify button is shown/focusable.
func (c *LightControlComponent) SetIdentifyVisible(visible bool) {
	c.showIdentify = visible
	c.updateFocusableFields()
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
	ControlType string // "on", "identify", "brightness", "colortemp", "color", "effect", "gradient-points"
	Value       any    // The actual value (same as the sub-component's value type)
}
