package field

import (
	"fmt"

	"github.com/charmbracelet/lipgloss/v2"
	"github.com/kluzzebass/lazyhue/internal/ui"
	zone "github.com/lrstanley/bubblezone/v2"
)

// BrightnessSliderComponent is a brightness slider (0-100%).
type BrightnessSliderComponent struct {
	*SliderComponent

	// Color to display (XY color space, 0-1 range)
	// If ColorX and ColorY are both 0, falls back to white or color temp
	ColorX float64
	ColorY float64

	// Color temperature in mirek (if using color temp mode instead of XY)
	// Only used if ColorX and ColorY are both 0
	ColorTempMirek int
}

// NewBrightnessSliderComponent creates a new brightness slider.
func NewBrightnessSliderComponent(id, label string, value int, styles *ui.Styles, zones *zone.Manager) *BrightnessSliderComponent {
	slider := NewSliderComponent(id, label, value, 0, 100, 5, styles, zones)

	b := &BrightnessSliderComponent{
		SliderComponent: slider,
	}

	// Set custom rendering
	slider.RenderBar = b.renderBrightnessBar
	slider.FormatValue = b.formatBrightness

	return b
}

// renderBrightnessBar renders a gradient bar from dim to bright.
func (b *BrightnessSliderComponent) renderBrightnessBar(value, min, max, width int, styles *ui.Styles) string {
	if max <= min {
		return ""
	}

	filled := value * width / max

	// Get the base color (at full brightness)
	var baseR, baseG, baseB uint8
	if b.ColorX != 0 || b.ColorY != 0 {
		// Use XY color
		baseR, baseG, baseB = ui.XyToRGB(b.ColorX, b.ColorY, 100)
	} else if b.ColorTempMirek != 0 {
		// Use color temperature
		baseR, baseG, baseB = ui.MirekToRGB(b.ColorTempMirek)
	} else {
		// Default to white
		baseR, baseG, baseB = 255, 255, 255
	}

	bar := ""
	for j := 0; j < width; j++ {
		if j < filled {
			// Gradient from dim to bright using the base color
			// Scale from ~50% to 100% intensity across the bar
			intensity := 0.5 + (float64(j) * 0.5 / float64(width))
			r := uint8(float64(baseR) * intensity)
			g := uint8(float64(baseG) * intensity)
			bl := uint8(float64(baseB) * intensity)
			bar += lipgloss.NewStyle().Foreground(lipgloss.Color(fmt.Sprintf("#%02x%02x%02x", r, g, bl))).Render("█")
		} else {
			bar += lipgloss.NewStyle().Foreground(styles.Theme.TextMuted).Render("░")
		}
	}

	return bar
}

// formatBrightness formats the brightness as a percentage.
func (b *BrightnessSliderComponent) formatBrightness(value, min, max int) string {
	pct := 0
	if max > 0 {
		pct = value * 100 / max
	}
	return fmt.Sprintf("%3d%%", pct)
}
