package field

import (
	"fmt"
	"math"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/kluzzebass/lazyhue/internal/ui"
	"github.com/kluzzebass/lazyhue/internal/ui/component"
	zone "github.com/lrstanley/bubblezone/v2"
)

// ColorWheelComponent is an XY color picker with an elliptical wheel.
type ColorWheelComponent struct {
	*BaseField

	// Current color in XY space
	ColorX float64
	ColorY float64

	// Original values for cancel
	OriginalX float64
	OriginalY float64

	// The color wheel renderer
	Wheel *ColorWheel

	// Blink timer state
	blinkTimerActive    bool
	blinkTimerScheduled bool

	// Mouse drag state
	Dragging     bool
	DragZoneInfo map[int]*zone.ZoneInfo // Cached zone info for each row during drag

	// Last emitted color value (to avoid duplicate release events)
	lastSentX     float64
	lastSentY     float64
	lastSentValid bool
	lastSentAt    time.Time

	// Inactive renders the wheel in grayscale (not the active color mode)
	Inactive bool

	// Brightness dims the wheel based on light brightness (0-100)
	Brightness int
}

// NewColorWheelComponent creates a new color wheel component.
func NewColorWheelComponent(id, label string, colorX, colorY float64, styles *ui.Styles, zones *zone.Manager) *ColorWheelComponent {
	wheel := NewColorWheel()
	wheel.SetColor(colorX, colorY)

	return &ColorWheelComponent{
		BaseField:  NewBaseField(id, label, styles, zones),
		ColorX:     colorX,
		ColorY:     colorY,
		Wheel:      wheel,
		Brightness: 100, // Full brightness by default
	}
}

// SetColor sets the current color.
func (c *ColorWheelComponent) SetColor(x, y float64) {
	c.ColorX = x
	c.ColorY = y
	c.Wheel.SetColor(x, y)
}

// SetHueSatPosition updates the cursor position from HSV hue/saturation.
func (c *ColorWheelComponent) SetHueSatPosition(hue, sat int) {
	h := ((hue % 360) + 360) % 360
	if sat < 0 {
		sat = 0
	}
	if sat > 100 {
		sat = 100
	}

	angleRad := float64(90-h) * math.Pi / 180
	dist := float64(sat) / 100.0
	xNorm := dist * math.Cos(angleRad)
	yNorm := -dist * math.Sin(angleRad)

	selCol := c.Wheel.RadiusX + int(math.Round(xNorm*float64(c.Wheel.RadiusX)))
	selRow := c.Wheel.RadiusY + int(math.Round(yNorm*float64(c.Wheel.RadiusY)))

	// Clamp to valid range
	if selRow < 0 {
		selRow = 0
	}
	if selRow > c.Wheel.RadiusY*2 {
		selRow = c.Wheel.RadiusY * 2
	}
	if selCol < 0 {
		selCol = 0
	}
	if selCol > c.Wheel.RadiusX*2 {
		selCol = c.Wheel.RadiusX * 2
	}

	c.Wheel.SelRow = selRow
	c.Wheel.SelCol = selCol
	c.Wheel.PosValid = true
}

// Update handles events for the color wheel.
func (c *ColorWheelComponent) Update(msg tea.Msg) (component.Component, tea.Cmd) {
	if c.ReadOnly {
		return c, nil
	}

	switch msg := msg.(type) {
	case BlinkTickMsg:
		if msg.FieldID == c.ID && c.blinkTimerActive && c.Editing {
			c.Wheel.BlinkOn = !c.Wheel.BlinkOn
			c.blinkTimerScheduled = false
			// Schedule next tick
			cmd := c.scheduleBlinkTick()
			return c, cmd
		}

	case tea.KeyMsg:
		if c.Editing {
			return c.handleEditingKey(msg)
		}
		return c.handleNormalKey(msg)

	case tea.MouseClickMsg:
		return c.handleMouseClick(msg)

	case tea.MouseMotionMsg:
		if c.Dragging {
			return c.handleMouseDrag(msg)
		}

	case tea.MouseReleaseMsg:
		if c.Dragging {
			c.Dragging = false
			c.DragZoneInfo = nil
			cmds := []tea.Cmd{func() tea.Msg { return EndCaptureMsg{FieldID: c.ID} }}
			if c.needsFinalChange() {
				c.markSent()
				cmds = append(cmds, func() tea.Msg {
					return FieldChangedMsg{
						FieldID: c.ID,
						Value:   ColorValue{X: c.ColorX, Y: c.ColorY},
					}
				})
			}
			return c, tea.Batch(cmds...)
		}
	}

	return c, nil
}

// RouteEvent routes events to this component.
func (c *ColorWheelComponent) RouteEvent(msg tea.Msg) (bool, tea.Cmd) {
	if c.ReadOnly {
		return false, nil
	}

	switch msg := msg.(type) {
	case BlinkTickMsg:
		if msg.FieldID == c.ID {
			_, cmd := c.Update(msg)
			return true, cmd
		}

	case tea.KeyMsg:
		if c.Editing {
			_, cmd := c.handleEditingKey(msg)
			return true, cmd // Editing captures all keys
		}
		switch msg.String() {
		case "enter", " ":
			_, cmd := c.startEditing()
			return true, cmd
		}

	case tea.MouseClickMsg:
		if msg.Button == tea.MouseLeft {
			// Check wheel row zones - always respond to clicks
			for row := 0; row < c.Wheel.Height(); row++ {
				rowZoneID := fmt.Sprintf("%s-row-%d", c.ZoneID(), row)
				if c.Zones != nil {
					if z := c.Zones.Get(rowZoneID); z != nil && z.InBounds(msg) {
						_, cmd := c.handleWheelClick(msg, row, z)
						return true, cmd
					}
				}
			}
		}

	case tea.MouseMotionMsg:
		if c.Dragging {
			_, cmd := c.handleMouseDrag(msg)
			return true, cmd
		}

	case tea.MouseReleaseMsg:
		if c.Dragging {
			c.Dragging = false
			c.DragZoneInfo = nil
			cmds := []tea.Cmd{func() tea.Msg { return EndCaptureMsg{FieldID: c.ID} }}
			if c.needsFinalChange() {
				c.markSent()
				cmds = append(cmds, func() tea.Msg {
					return FieldChangedMsg{
						FieldID: c.ID,
						Value:   ColorValue{X: c.ColorX, Y: c.ColorY},
					}
				})
			}
			return true, tea.Batch(cmds...)
		}
	}

	return false, nil
}

func (c *ColorWheelComponent) handleNormalKey(msg tea.KeyMsg) (component.Component, tea.Cmd) {
	switch msg.String() {
	case "enter", " ":
		return c.startEditing()
	}
	return c, nil
}

func (c *ColorWheelComponent) handleEditingKey(msg tea.KeyMsg) (component.Component, tea.Cmd) {
	switch msg.String() {
	case "esc":
		// Cancel and restore original color
		c.Wheel.RestoreOriginal()
		c.ColorX = c.Wheel.ColorX
		c.ColorY = c.Wheel.ColorY
		c.Editing = false
		c.blinkTimerActive = false
		c.blinkTimerScheduled = false
		return c, nil

	case "enter":
		// Confirm edit
		c.ColorX = c.Wheel.ColorX
		c.ColorY = c.Wheel.ColorY
		c.Editing = false
		c.blinkTimerActive = false
		c.blinkTimerScheduled = false
		return c, func() tea.Msg {
			return FieldChangedMsg{
				FieldID: c.ID,
				Value:   ColorValue{X: c.ColorX, Y: c.ColorY},
			}
		}

	case "left", "h":
		c.Wheel.MoveLeft()
		c.ColorX = c.Wheel.ColorX
		c.ColorY = c.Wheel.ColorY

	case "right", "l":
		c.Wheel.MoveRight()
		c.ColorX = c.Wheel.ColorX
		c.ColorY = c.Wheel.ColorY

	case "up", "k":
		c.Wheel.MoveUp()
		c.ColorX = c.Wheel.ColorX
		c.ColorY = c.Wheel.ColorY

	case "down", "j":
		c.Wheel.MoveDown()
		c.ColorX = c.Wheel.ColorX
		c.ColorY = c.Wheel.ColorY
	}

	return c, nil
}

func (c *ColorWheelComponent) handleMouseClick(msg tea.MouseClickMsg) (component.Component, tea.Cmd) {
	if !c.Editing {
		// Click on the preview to start editing
		if c.Zones != nil {
			if z := c.Zones.Get(c.ZoneID()); z != nil && z.InBounds(msg) {
				return c.startEditing()
			}
		}
		return c, nil
	}

	// When editing, clicks are handled by RouteEvent checking row zones
	return c, nil
}

func (c *ColorWheelComponent) handleWheelClick(msg tea.MouseClickMsg, row int, z *zone.ZoneInfo) (component.Component, tea.Cmd) {
	// Calculate column from mouse X relative to zone
	col := msg.X - z.StartX
	if c.Wheel.HandleClick(row, col) {
		c.ColorX = c.Wheel.ColorX
		c.ColorY = c.Wheel.ColorY
	}

	// Activate editing mode so cursor is visible
	if !c.Editing {
		c.Editing = true
		c.OriginalX = c.ColorX
		c.OriginalY = c.ColorY
		c.Wheel.SetOriginal(c.ColorX, c.ColorY)
		c.Wheel.BlinkOn = true
		c.blinkTimerActive = true
	}

	// Start drag mode and cache zone info
	c.Dragging = true
	c.DragZoneInfo = make(map[int]*zone.ZoneInfo)
	for r := 0; r < c.Wheel.Height(); r++ {
		rowZoneID := fmt.Sprintf("%s-row-%d", c.ZoneID(), r)
		if c.Zones != nil {
			if zInfo := c.Zones.Get(rowZoneID); zInfo != nil {
				c.DragZoneInfo[r] = zInfo
			}
		}
	}

	// Start capture, blink timer, and send initial change
	c.markSent()
	return c, tea.Batch(
		func() tea.Msg { return StartCaptureMsg{FieldID: c.ID} },
		func() tea.Msg {
			return FieldChangedMsg{
				FieldID: c.ID,
				Value:   ColorValue{X: c.ColorX, Y: c.ColorY},
			}
		},
		c.scheduleBlinkTick(),
	)
}

func (c *ColorWheelComponent) handleMouseDrag(msg tea.MouseMotionMsg) (component.Component, tea.Cmd) {
	if c.DragZoneInfo == nil {
		return c, nil
	}

	// Find which row the mouse is in based on Y coordinate
	for row, zInfo := range c.DragZoneInfo {
		if msg.Y >= zInfo.StartY && msg.Y <= zInfo.EndY {
			// Calculate column from mouse X relative to zone
			col := msg.X - zInfo.StartX
			if c.Wheel.HandleClick(row, col) {
				c.ColorX = c.Wheel.ColorX
				c.ColorY = c.Wheel.ColorY
				// Throttle drag updates to avoid flooding the bridge.
				if c.readyToSend() {
					c.markSent()
					return c, func() tea.Msg {
						return FieldChangedMsg{
							FieldID: c.ID,
							Value:   ColorValue{X: c.ColorX, Y: c.ColorY},
						}
					}
				}
			}
			return c, nil
		}
	}

	return c, nil
}

func (c *ColorWheelComponent) startEditing() (component.Component, tea.Cmd) {
	c.Editing = true
	c.OriginalX = c.ColorX
	c.OriginalY = c.ColorY
	c.Wheel.SetOriginal(c.ColorX, c.ColorY)
	c.Wheel.SetColor(c.ColorX, c.ColorY)
	c.Wheel.BlinkOn = true
	c.blinkTimerActive = true

	// Schedule first blink tick
	cmd := c.scheduleBlinkTick()
	return c, cmd
}

func (c *ColorWheelComponent) scheduleBlinkTick() tea.Cmd {
	if c.blinkTimerScheduled {
		return nil
	}
	c.blinkTimerScheduled = true
	fieldID := c.ID
	return tea.Tick(500*time.Millisecond, func(t time.Time) tea.Msg {
		return BlinkTickMsg{FieldID: fieldID}
	})
}

// FieldHeight returns the number of rows this component takes up.
func (c *ColorWheelComponent) FieldHeight() int {
	return c.Wheel.Height() // Always show the full wheel
}

// ViewControl renders only the control portion (no label).
func (c *ColorWheelComponent) ViewControl() string {
	return c.renderWheelControl() // Always show the wheel
}

func (c *ColorWheelComponent) markSent() {
	c.lastSentX = c.ColorX
	c.lastSentY = c.ColorY
	c.lastSentValid = true
	c.lastSentAt = time.Now()
}

func (c *ColorWheelComponent) needsFinalChange() bool {
	if !c.lastSentValid {
		return true
	}
	const epsilon = 0.00001
	return math.Abs(c.ColorX-c.lastSentX) > epsilon || math.Abs(c.ColorY-c.lastSentY) > epsilon
}

func (c *ColorWheelComponent) readyToSend() bool {
	if c.lastSentAt.IsZero() {
		return true
	}
	return time.Since(c.lastSentAt) >= 50*time.Millisecond
}

func (c *ColorWheelComponent) renderWheelControl() string {
	var out strings.Builder

	// Set grayscale mode based on inactive state
	c.Wheel.Grayscale = c.Inactive
	// Set brightness for dimming effect
	c.Wheel.Brightness = c.Brightness

	wheelLines := strings.Split(c.Wheel.Render(), "\n")
	for row, line := range wheelLines {
		if line == "" {
			continue
		}
		rowLine := line
		if c.Zones != nil {
			rowZoneID := fmt.Sprintf("%s-row-%d", c.ZoneID(), row)
			rowLine = c.Zones.Mark(rowZoneID, rowLine)
		}
		if row > 0 {
			out.WriteString("\n")
		}
		out.WriteString(rowLine)
	}

	result := out.String()

	// Also wrap entire wheel with main zone so FormComponent can detect clicks
	if c.Zones != nil {
		result = c.Zones.Mark(c.ZoneID(), result)
	}

	return result
}

// View renders the color wheel field (label + control for backwards compatibility).
func (c *ColorWheelComponent) View() string {
	labelStr := c.Label
	if c.MaxLabelWidth > 0 {
		labelStr = fmt.Sprintf("%-*s", c.MaxLabelWidth, c.Label)
	}

	control := c.ViewControl()

	if c.Editing {
		// Multi-row: label on first line, control rows indented
		var out strings.Builder
		out.WriteString(fmt.Sprintf("  %s\n", labelStr))
		controlLines := strings.Split(control, "\n")
		for i, line := range controlLines {
			out.WriteString("    " + line)
			if i < len(controlLines)-1 {
				out.WriteString("\n")
			}
		}
		return out.String()
	}

	// Single row: label + control
	return fmt.Sprintf("  %s  %s", labelStr, control)
}
