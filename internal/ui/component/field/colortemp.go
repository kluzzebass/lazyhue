package field

import (
	"fmt"
	"math"

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
	slider := NewSliderComponent(id, label, value, min, max, 10, styles, zones)
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

// renderColorTempBar renders a gradient bar from warm (left) to cool (right).
func (c *ColorTempSliderComponent) renderColorTempBar(value, min, max, width int, styles *ui.Styles) string {
	if max <= min {
		return ""
	}

	// Calculate position (inverted: high mirek → left)
	pos := (max - value) * width / (max - min)
	pos = clamp(pos, 0, width-1)

	bar := ""
	for j := 0; j < width; j++ {
		ratio := float64(j) / float64(maxInt(1, width-1))
		mirek := int(math.Round(float64(max) - ratio*float64(max-min)))
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

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
