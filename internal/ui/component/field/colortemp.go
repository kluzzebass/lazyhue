package field

import (
	"fmt"

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
		// Gradient from warm orange to cool blue
		warmR, warmG, warmB := 255, 180, 100 // Warm (left, high mirek)
		coolR, coolG, coolB := 150, 200, 255 // Cool (right, low mirek)
		r := warmR + (coolR-warmR)*j/width
		g := warmG + (coolG-warmG)*j/width
		b := warmB + (coolB-warmB)*j/width

		// Desaturate if inactive (not the current color mode)
		if c.Inactive {
			gray := (r + g + b) / 3
			// Blend 75% toward gray for a muted look
			r = (r + gray*3) / 4
			g = (g + gray*3) / 4
			b = (b + gray*3) / 4
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
