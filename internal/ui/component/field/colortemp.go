package field

import (
	"fmt"
	"math"

	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/charmbracelet/lipgloss/v2"
	"github.com/kluzzebass/lazyhue/internal/ui"
	zone "github.com/lrstanley/bubblezone/v2"
)

// ColorTempSliderComponent is a color temperature slider.
// Warm (high mirek) = left, Cool (low mirek) = right.
type ColorTempSliderComponent struct {
	*SliderComponent
}

// NewColorTempSliderComponent creates a new color temperature slider.
func NewColorTempSliderComponent(id, label string, value, min, max int, styles *ui.Styles, zones *zone.Manager) *ColorTempSliderComponent {
	slider := NewSliderComponent(id, label, value, min, max, 100, styles, zones)
	slider.Inverted = true // Left = warm (high mirek), right = cool (low mirek)
	slider.BarWidth = 21

	c := &ColorTempSliderComponent{
		SliderComponent: slider,
	}

	// Set custom rendering
	slider.RenderBar = c.renderColorTempBar
	slider.FormatValue = c.formatColorTemp

	return c
}

// RouteEvent routes events to this component using a Kelvin-linear scale.
func (c *ColorTempSliderComponent) RouteEvent(msg tea.Msg) (bool, tea.Cmd) {
	if c.ReadOnly {
		return false, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "left", "h":
			return true, c.adjustKelvin(-c.Step)
		case "right", "l":
			return true, c.adjustKelvin(c.Step)
		}

	case tea.MouseClickMsg:
		if msg.Button == tea.MouseLeft {
			if c.Zones != nil {
				if z := c.Zones.Get(c.ZoneID()); z != nil && z.InBounds(msg) {
					cmd := c.handleKelvinClick(msg)
					return true, cmd
				}
			}
		}

	case tea.MouseMotionMsg:
		if c.Dragging {
			cmd := c.handleKelvinDrag(msg.X)
			return true, cmd
		}

	case tea.MouseReleaseMsg:
		if c.Dragging {
			c.Dragging = false
			c.DragZoneStartX = -1
			return true, func() tea.Msg { return EndCaptureMsg{FieldID: c.ID} }
		}
	}

	return false, nil
}

// renderColorTempBar renders a gradient bar from warm (left) to cool (right).
func (c *ColorTempSliderComponent) renderColorTempBar(value, min, max, width int, styles *ui.Styles) string {
	if max <= min {
		return ""
	}

	minK, maxK := c.kelvinRange()
	currentK := c.valueKelvin()
	if maxK == minK {
		maxK = minK + 1
	}
	pos := (currentK - minK) * (width - 1) / (maxK - minK)
	pos = clamp(pos, 0, width-1)

	bar := ""
	for j := 0; j < width; j++ {
		ratio := float64(j) / float64(maxInt(1, width-1))
		kelvin := float64(minK) + ratio*float64(maxK-minK)
		mirek := int(math.Round(1000000.0 / math.Max(1.0, kelvin)))
		r, g, b := ui.MirekToRGB(mirek)

		// Desaturate if inactive (not the current color mode)
		if c.Inactive {
			// Calculate luminance (perceived brightness)
			lum := (299*int(r) + 587*int(g) + 114*int(b)) / 1000
			r, g, b = uint8(lum), uint8(lum), uint8(lum)
		}

		// Apply brightness dimming (0% brightness → 50% luminance, 100% → 100%)
		if c.Brightness < 100 && c.Brightness >= 0 {
			factor := 0.5 + float64(c.Brightness)/200.0 // 0→0.5, 100→1.0
			r = uint8(float64(r) * factor)
			g = uint8(float64(g) * factor)
			b = uint8(float64(b) * factor)
		}

		char := "─"
		if j == pos {
			char = "●"
		}
		bar += lipgloss.NewStyle().Foreground(lipgloss.Color(fmt.Sprintf("#%02x%02x%02x", r, g, b))).Render(char)
	}

	return bar
}

// formatColorTemp formats the color temperature as Kelvin.
func (c *ColorTempSliderComponent) formatColorTemp(value, min, max int) string {
	// Convert mirek to Kelvin (mirek = 1,000,000 / K)
	kelvin := 1000000 / maxInt(1, value)
	return fmt.Sprintf("%dK", kelvin)
}

func (c *ColorTempSliderComponent) adjustKelvin(delta int) tea.Cmd {
	kelvin := c.valueKelvin()
	minK, maxK := c.kelvinRange()
	newKelvin := clamp(kelvin+delta, minK, maxK)
	newValue := c.kelvinToMirek(newKelvin)
	if newValue == c.Value {
		return nil
	}

	c.Value = newValue
	return func() tea.Msg {
		return FieldChangedMsg{
			FieldID: c.ID,
			Value:   SliderValue{Value: c.Value},
		}
	}
}

func (c *ColorTempSliderComponent) handleKelvinClick(msg tea.MouseClickMsg) tea.Cmd {
	if c.Zones == nil {
		return nil
	}

	zoneInfo := c.Zones.Get(c.ZoneID())
	if zoneInfo == nil {
		return nil
	}

	zoneStartX := c.findZoneStartX(zoneInfo, msg.X, msg.Y)
	if zoneStartX < 0 {
		return nil
	}

	c.DragZoneStartX = zoneStartX
	c.Dragging = true

	newValue := c.valueFromKelvinPosition(msg.X, zoneStartX)
	if newValue != c.Value {
		c.Value = newValue
		return tea.Batch(
			func() tea.Msg { return StartCaptureMsg{FieldID: c.ID} },
			func() tea.Msg { return FieldChangedMsg{FieldID: c.ID, Value: SliderValue{Value: c.Value}} },
		)
	}

	return func() tea.Msg { return StartCaptureMsg{FieldID: c.ID} }
}

func (c *ColorTempSliderComponent) handleKelvinDrag(mouseX int) tea.Cmd {
	if c.DragZoneStartX < 0 {
		return nil
	}

	newValue := c.valueFromKelvinPosition(mouseX, c.DragZoneStartX)
	if newValue != c.Value {
		c.Value = newValue
		return func() tea.Msg {
			return FieldChangedMsg{FieldID: c.ID, Value: SliderValue{Value: c.Value}}
		}
	}

	return nil
}

func (c *ColorTempSliderComponent) valueFromKelvinPosition(mouseX, zoneStartX int) int {
	barWidth := c.barWidth()
	relativeX := clamp(mouseX-zoneStartX, 0, barWidth-1)
	minK, maxK := c.kelvinRange()
	newKelvin := minK + (relativeX * (maxK - minK) / maxInt(1, barWidth-1))
	return c.kelvinToMirek(newKelvin)
}

func (c *ColorTempSliderComponent) kelvinRange() (minK, maxK int) {
	minK = 1000000 / maxInt(1, c.Max)
	maxK = 1000000 / maxInt(1, c.Min)
	if minK > maxK {
		minK, maxK = maxK, minK
	}
	return minK, maxK
}

func (c *ColorTempSliderComponent) valueKelvin() int {
	return 1000000 / maxInt(1, c.Value)
}

func (c *ColorTempSliderComponent) kelvinToMirek(kelvin int) int {
	mirek := 1000000 / maxInt(1, kelvin)
	return clamp(mirek, c.Min, c.Max)
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
