package field

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/charmbracelet/lipgloss/v2"
	"github.com/kluzzebass/lazyhue/internal/ui"
	"github.com/kluzzebass/lazyhue/internal/ui/component"
	zone "github.com/lrstanley/bubblezone/v2"
)

// RGBComponent is an RGB color picker with 3 sliders.
type RGBComponent struct {
	*BaseField

	// Current RGB values (0-255)
	Red   int
	Green int
	Blue  int

	// Original values for cancel
	OriginalRed   int
	OriginalGreen int
	OriginalBlue  int

	// Which slider is focused when editing (0=R, 1=G, 2=B)
	SliderFocus int

	// Slider bar width
	sliderWidth int

	// ShowSwatch controls whether the color preview swatch is displayed
	ShowSwatch bool

	// Inactive grays out the control (e.g., when using color temp mode)
	Inactive bool

	// Drag state
	Dragging     bool
	DragZoneInfo *zone.ZoneInfo
}

// NewRGBComponent creates a new RGB color picker component.
func NewRGBComponent(id, label string, red, green, blue int, styles *ui.Styles, zones *zone.Manager) *RGBComponent {
	return &RGBComponent{
		BaseField:   NewBaseField(id, label, styles, zones),
		Red:         red,
		Green:       green,
		Blue:        blue,
		sliderWidth: 20,
		ShowSwatch:  true,
	}
}

// SetRGB sets the RGB values.
func (c *RGBComponent) SetRGB(red, green, blue int) {
	c.Red = red
	c.Green = green
	c.Blue = blue
}

// Update handles events for the RGB picker.
func (c *RGBComponent) Update(msg tea.Msg) (component.Component, tea.Cmd) {
	// Events are handled via RouteEvent
	return c, nil
}

// RouteEvent routes events to this component.
func (c *RGBComponent) RouteEvent(msg tea.Msg) (bool, tea.Cmd) {
	if c.ReadOnly || c.Inactive {
		return false, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "left", "h":
			if !c.Editing {
				c.saveOriginal()
			}
			c.adjustSlider(-1)
			return true, c.emitChange()

		case "right", "l":
			if !c.Editing {
				c.saveOriginal()
			}
			c.adjustSlider(1)
			return true, c.emitChange()

		case "up", "k":
			// Move to previous slider, or let parent handle if at top
			if c.SliderFocus > 0 {
				c.SliderFocus--
				return true, nil
			}
			return false, nil // Let parent navigate

		case "down", "j":
			// Move to next slider, or let parent handle if at bottom
			if c.SliderFocus < 2 {
				c.SliderFocus++
				return true, nil
			}
			return false, nil // Let parent navigate

		case "esc":
			if c.Editing {
				c.restoreOriginal()
				c.Editing = false
				return true, nil
			}

		case "enter", " ":
			if c.Editing {
				c.Editing = false
				return true, c.emitChange()
			}
		}

	case tea.MouseClickMsg:
		if msg.Button == tea.MouseLeft {
			// Check slider row zones
			for row := 0; row < 3; row++ {
				rowZoneID := fmt.Sprintf("%s-row-%d", c.ZoneID(), row)
				if c.Zones != nil {
					if z := c.Zones.Get(rowZoneID); z != nil && z.InBounds(msg) {
						_, cmd := c.handleSliderClick(msg, row, z)
						return true, cmd
					}
				}
			}
		}

	case tea.MouseMotionMsg:
		if c.Dragging && c.DragZoneInfo != nil {
			c.updateFromMouseX(msg.X)
			return true, c.emitChange()
		}

	case tea.MouseReleaseMsg:
		if c.Dragging {
			c.Dragging = false
			c.DragZoneInfo = nil
			return true, tea.Batch(
				func() tea.Msg { return EndCaptureMsg{FieldID: c.ID} },
				c.emitChange(),
			)
		}
	}

	return false, nil
}

func (c *RGBComponent) updateFromMouseX(mouseX int) {
	if c.DragZoneInfo == nil {
		return
	}
	barStartOffset := 3 // "R: " prefix
	clickPos := mouseX - c.DragZoneInfo.StartX - barStartOffset
	if clickPos < 0 {
		clickPos = 0
	}
	if clickPos > c.sliderWidth-1 {
		clickPos = c.sliderWidth - 1
	}
	// Map 0..sliderWidth-1 to 0..255
	value := clickPos * 255 / (c.sliderWidth - 1)
	switch c.SliderFocus {
	case 0:
		c.Red = value
	case 1:
		c.Green = value
	case 2:
		c.Blue = value
	}
}

func (c *RGBComponent) saveOriginal() {
	c.Editing = true
	c.OriginalRed = c.Red
	c.OriginalGreen = c.Green
	c.OriginalBlue = c.Blue
}

func (c *RGBComponent) restoreOriginal() {
	c.Red = c.OriginalRed
	c.Green = c.OriginalGreen
	c.Blue = c.OriginalBlue
}

func (c *RGBComponent) emitChange() tea.Cmd {
	return func() tea.Msg {
		return FieldChangedMsg{
			FieldID: c.ID,
			Value:   RGBValue{Red: c.Red, Green: c.Green, Blue: c.Blue},
		}
	}
}

func (c *RGBComponent) adjustSlider(direction int) {
	step := 10 // RGB values go 0-255
	switch c.SliderFocus {
	case 0: // Red
		c.Red = clamp(c.Red+direction*step, 0, 255)
	case 1: // Green
		c.Green = clamp(c.Green+direction*step, 0, 255)
	case 2: // Blue
		c.Blue = clamp(c.Blue+direction*step, 0, 255)
	}
}

func (c *RGBComponent) handleSliderClick(msg tea.MouseClickMsg, row int, z *zone.ZoneInfo) (component.Component, tea.Cmd) {
	if !c.Editing {
		c.saveOriginal()
	}

	c.SliderFocus = row

	// Calculate position within slider bar
	// The slider bar starts after "R: " (3 chars from zone start)
	barStartOffset := 3
	clickPos := msg.X - z.StartX - barStartOffset
	if clickPos < 0 {
		clickPos = 0
	}
	if clickPos > c.sliderWidth-1 {
		clickPos = c.sliderWidth - 1
	}

	// Convert position to value (0-255)
	// Map 0..sliderWidth-1 to 0..255
	value := clickPos * 255 / (c.sliderWidth - 1)

	switch row {
	case 0: // Red
		c.Red = value
	case 1: // Green
		c.Green = value
	case 2: // Blue
		c.Blue = value
	}

	// Start drag mode
	c.Dragging = true
	c.DragZoneInfo = z

	return c, tea.Batch(
		func() tea.Msg { return StartCaptureMsg{FieldID: c.ID} },
		c.emitChange(),
	)
}

// FieldHeight returns the number of rows this component takes up.
func (c *RGBComponent) FieldHeight() int {
	return 3 // Always 3 rows for R, G, B sliders
}

// ViewControl renders only the control portion (no label).
// Returns 3 lines separated by newlines.
func (c *RGBComponent) ViewControl() string {
	var out strings.Builder

	for row := 0; row < 3; row++ {
		if row > 0 {
			out.WriteString("\n")
		}

		sliderLine := c.renderSliderRow(row)

		// Highlight focused slider when component is focused
		if c.IsFocused() && row == c.SliderFocus {
			sliderLine = c.Styles.Selected.Render(sliderLine)
		}

		// Mark row with zone
		if c.Zones != nil {
			rowZoneID := fmt.Sprintf("%s-row-%d", c.ZoneID(), row)
			sliderLine = c.Zones.Mark(rowZoneID, sliderLine)
		}

		out.WriteString(sliderLine)
	}

	return out.String()
}

// View renders the RGB picker.
func (c *RGBComponent) View() string {
	labelStr := c.Label
	if c.MaxLabelWidth > 0 {
		labelStr = fmt.Sprintf("%-*s", c.MaxLabelWidth, c.Label)
	}

	var out strings.Builder

	// Render 3 rows
	for row := 0; row < 3; row++ {
		if row > 0 {
			out.WriteString("\n")
		}

		var rowLabel string
		if row == 0 {
			rowLabel = fmt.Sprintf("  %s  ", labelStr)
		} else {
			// Indent subsequent rows
			rowLabel = strings.Repeat(" ", 2+c.MaxLabelWidth+2)
		}

		sliderLine := c.renderSliderRow(row)

		// Highlight focused slider when component is focused
		if c.IsFocused() && row == c.SliderFocus {
			sliderLine = c.Styles.Selected.Render(sliderLine)
		}

		line := rowLabel + sliderLine

		// Mark row with zone
		if c.Zones != nil {
			rowZoneID := fmt.Sprintf("%s-row-%d", c.ZoneID(), row)
			line = c.Zones.Mark(rowZoneID, line)
		}

		out.WriteString(line)
	}

	return out.String()
}

func (c *RGBComponent) renderSliderRow(row int) string {
	rowLabels := []string{"R", "G", "B"}
	rowValues := []int{c.Red, c.Green, c.Blue}

	rowVal := rowValues[row]
	pos := rowVal * c.sliderWidth / 255
	if pos >= c.sliderWidth {
		pos = c.sliderWidth - 1
	}

	var bar string
	for j := 0; j < c.sliderWidth; j++ {
		intensity := j * 255 / c.sliderWidth
		var r, g, b int
		switch row {
		case 0: // Red slider
			r, g, b = intensity, 0, 0
		case 1: // Green slider
			r, g, b = 0, intensity, 0
		case 2: // Blue slider
			r, g, b = 0, 0, intensity
		}
		// Convert to grayscale when inactive
		if c.Inactive {
			gray := (r + g + b) / 3
			r, g, b = gray, gray, gray
		}
		char := "─"
		if j == pos {
			char = "●"
		}
		bar += lipgloss.NewStyle().Foreground(lipgloss.Color(fmt.Sprintf("#%02x%02x%02x", r, g, b))).Render(char)
	}

	// Value display
	valueStr := fmt.Sprintf("%s: %s %3d", rowLabels[row], bar, rowVal)

	if !c.ShowSwatch {
		return valueStr
	}

	// Color preview stripe
	colorHex := fmt.Sprintf("#%02x%02x%02x", c.Red, c.Green, c.Blue)
	colorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(colorHex))
	bgStyle := lipgloss.NewStyle().Background(lipgloss.Color(colorHex))

	var preview string
	switch row {
	case 0: // Top row - rounded corners
		preview = " " + colorStyle.Render("▄▄") + " "
	case 2: // Bottom row - rounded corners
		preview = " " + colorStyle.Render("▀▀") + " "
	default: // Middle row - full block
		preview = bgStyle.Render("    ")
	}

	return valueStr + " " + preview
}
