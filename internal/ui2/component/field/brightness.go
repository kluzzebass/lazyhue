package field

import (
	"fmt"

	"github.com/charmbracelet/lipgloss/v2"
	"github.com/kluzzebass/lazyhue/internal/ui2"
	zone "github.com/lrstanley/bubblezone/v2"
)

// BrightnessSliderComponent is a brightness slider (0-100%).
type BrightnessSliderComponent struct {
	*SliderComponent
}

// NewBrightnessSliderComponent creates a new brightness slider.
func NewBrightnessSliderComponent(id, label string, value int, styles *ui2.Styles, zones *zone.Manager) *BrightnessSliderComponent {
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
func (b *BrightnessSliderComponent) renderBrightnessBar(value, min, max, width int, styles *ui2.Styles) string {
	if max <= min {
		return ""
	}

	filled := value * width / max

	bar := ""
	for j := 0; j < width; j++ {
		if j < filled {
			// Gradient from dim to bright
			intensity := 180 + (j * 75 / width)
			bar += lipgloss.NewStyle().Foreground(lipgloss.Color(fmt.Sprintf("#%02x%02x%02x", intensity, intensity, intensity))).Render("█")
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
