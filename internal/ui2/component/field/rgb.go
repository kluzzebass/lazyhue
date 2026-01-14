package field

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/charmbracelet/lipgloss/v2"
	"github.com/kluzzebass/lazyhue/internal/ui2"
	"github.com/kluzzebass/lazyhue/internal/ui2/component"
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
}

// NewRGBComponent creates a new RGB color picker component.
func NewRGBComponent(id, label string, red, green, blue int, styles *ui2.Styles, zones *zone.Manager) *RGBComponent {
	return &RGBComponent{
		BaseField:   NewBaseField(id, label, styles, zones),
		Red:         red,
		Green:       green,
		Blue:        blue,
		sliderWidth: 20,
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
	if c.ReadOnly {
		return c, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if c.Editing {
			return c.handleEditingKey(msg)
		}
		return c.handleNormalKey(msg)

	case tea.MouseClickMsg:
		return c.handleMouseClick(msg)
	}

	return c, nil
}

// RouteEvent routes events to this component.
func (c *RGBComponent) RouteEvent(msg tea.Msg) (bool, tea.Cmd) {
	if c.ReadOnly {
		return false, nil
	}

	switch msg := msg.(type) {
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
	}

	return false, nil
}

func (c *RGBComponent) handleNormalKey(msg tea.KeyMsg) (component.Component, tea.Cmd) {
	switch msg.String() {
	case "enter", " ":
		return c.startEditing()
	}
	return c, nil
}

func (c *RGBComponent) handleEditingKey(msg tea.KeyMsg) (component.Component, tea.Cmd) {
	switch msg.String() {
	case "esc":
		// Cancel and restore original values
		c.Red = c.OriginalRed
		c.Green = c.OriginalGreen
		c.Blue = c.OriginalBlue
		c.Editing = false
		return c, nil

	case "enter":
		// Confirm edit
		c.Editing = false
		return c, func() tea.Msg {
			return FieldChangedMsg{
				FieldID: c.ID,
				Value:   RGBValue{Red: c.Red, Green: c.Green, Blue: c.Blue},
			}
		}

	case "up", "k":
		// Move to previous slider
		if c.SliderFocus > 0 {
			c.SliderFocus--
		}

	case "down", "j":
		// Move to next slider
		if c.SliderFocus < 2 {
			c.SliderFocus++
		}

	case "left", "h":
		c.adjustSlider(-1)

	case "right", "l":
		c.adjustSlider(1)
	}

	return c, nil
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

func (c *RGBComponent) handleMouseClick(msg tea.MouseClickMsg) (component.Component, tea.Cmd) {
	if !c.Editing {
		// Click on preview to start editing
		if c.Zones != nil {
			if z := c.Zones.Get(c.ZoneID()); z != nil && z.InBounds(msg) {
				return c.startEditing()
			}
		}
	}
	return c, nil
}

func (c *RGBComponent) handleSliderClick(msg tea.MouseClickMsg, row int, z *zone.ZoneInfo) (component.Component, tea.Cmd) {
	if !c.Editing {
		c.startEditing()
	}

	c.SliderFocus = row

	// Calculate position within slider bar
	// The slider bar starts after "  R: " (5 chars from zone start)
	barStartOffset := 5
	clickPos := msg.X - z.StartX - barStartOffset
	if clickPos < 0 {
		clickPos = 0
	}
	if clickPos >= c.sliderWidth {
		clickPos = c.sliderWidth - 1
	}

	// Convert position to value (0-255)
	value := clickPos * 255 / c.sliderWidth

	switch row {
	case 0: // Red
		c.Red = value
	case 1: // Green
		c.Green = value
	case 2: // Blue
		c.Blue = value
	}

	return c, nil
}

func (c *RGBComponent) startEditing() (component.Component, tea.Cmd) {
	c.Editing = true
	c.OriginalRed = c.Red
	c.OriginalGreen = c.Green
	c.OriginalBlue = c.Blue
	c.SliderFocus = 0
	return c, nil
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

		// Highlight focused slider when editing
		if c.Editing && row == c.SliderFocus {
			sliderLine = c.Styles.Focused.Render(sliderLine)
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

		// Highlight focused slider when editing
		if c.Editing && row == c.SliderFocus {
			sliderLine = c.Styles.Focused.Render(sliderLine)
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
		char := "─"
		if j == pos {
			char = "●"
		}
		bar += lipgloss.NewStyle().Foreground(lipgloss.Color(fmt.Sprintf("#%02x%02x%02x", r, g, b))).Render(char)
	}

	// Value display
	valueStr := fmt.Sprintf("%s: %s %3d", rowLabels[row], bar, rowVal)

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
