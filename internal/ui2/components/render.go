// Package components provides reusable UI components for the v2 UI.
package components

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss/v2"
	"github.com/kluzzebass/lazyhue/internal/ui2"
)

// Helper functions
func clamp(v, min, max int) int {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// RenderFieldValue renders the value portion of a form field.
// This is a reusable function that can be used by form systems.
// Note: For multi-row fields (HSL, RGB), this only renders the first row.
// The caller should call this for each sub-row with appropriate field data.
func RenderFieldValue(field *FormField, isFocused bool, styles *ui2.Styles, subRow int) string {
	switch field.Type {
	case FormFieldToggle:
		return renderToggle(field)

	case FormFieldSlider:
		return renderSlider(field)

	case FormFieldBrightness:
		return renderBrightness(field, styles)

	case FormFieldColorTemp:
		return renderColorTemp(field, styles)

	case FormFieldColor:
		return renderColor(field, styles)

	case FormFieldSelect:
		return renderSelect(field, false, 0) // dropdownOpen, dropdownScroll

	case FormFieldRadio:
		return renderRadio(field, isFocused, styles, subRow)

	case FormFieldHSL:
		return renderHSL(field, isFocused, styles, subRow)

	case FormFieldRGB:
		return renderRGB(field, isFocused, styles, subRow)

	case FormFieldText:
		return field.TextValue

	default:
		return fmt.Sprintf("%d", field.Value)
	}
}

// renderToggle renders a toggle field.
func renderToggle(field *FormField) string {
	onLabel := field.ToggleOnLabel
	offLabel := field.ToggleOffLabel
	if onLabel == "" {
		onLabel = "On"
	}
	if offLabel == "" {
		offLabel = "Off"
	}
	if field.Value != 0 {
		return "[●] " + onLabel
	}
	return "[ ] " + offLabel
}

// renderSlider renders a generic slider field: [====●----] 3/5
func renderSlider(field *FormField) string {
	sliderWidth := 10
	if field.Max > 0 {
		filled := field.Value * sliderWidth / field.Max
		empty := sliderWidth - filled
		if filled > sliderWidth {
			filled = sliderWidth
			empty = 0
		}
		return "[" + strings.Repeat("=", filled) + "●" + strings.Repeat("-", empty) + "] " + fmt.Sprintf("%d/%d", field.Value, field.Max)
	}
	return fmt.Sprintf("%d", field.Value)
}

// renderBrightness renders a brightness slider with visual gradient bar.
func renderBrightness(field *FormField, styles *ui2.Styles) string {
	barWidth := 20
	pct := 0
	if field.Max > 0 {
		pct = field.Value * 100 / field.Max
	}
	filled := field.Value * barWidth / max(1, field.Max)

	// Build gradient bar with brightness indication
	bar := ""
	for j := 0; j < barWidth; j++ {
		if j < filled {
			// Gradient from dim to bright
			intensity := 180 + (j * 75 / barWidth)
			bar += lipgloss.NewStyle().Foreground(lipgloss.Color(fmt.Sprintf("#%02x%02x%02x", intensity, intensity, intensity))).Render("█")
		} else {
			bar += lipgloss.NewStyle().Foreground(styles.Theme.TextMuted).Render("░")
		}
	}
	return bar + fmt.Sprintf(" %3d%%", pct)
}

// renderColorTemp renders a color temperature slider.
// Warm (high mirek) = left, Cool (low mirek) = right.
func renderColorTemp(field *FormField, styles *ui2.Styles) string {
	barWidth := 20
	if field.Max <= field.Min {
		return fmt.Sprintf("%d", field.Value)
	}

	// Invert position: high mirek → left, low mirek → right
	pos := (field.Max - field.Value) * barWidth / (field.Max - field.Min)
	pos = clamp(pos, 0, barWidth-1)

	// Build gradient bar from warm (left) to cool (right)
	bar := ""
	for j := 0; j < barWidth; j++ {
		// Gradient from warm orange to cool blue
		warmR, warmG, warmB := 255, 180, 100 // Warm (left, high mirek)
		coolR, coolG, coolB := 150, 200, 255 // Cool (right, low mirek)
		r := warmR + (coolR-warmR)*j/barWidth
		g := warmG + (coolG-warmG)*j/barWidth
		b := warmB + (coolB-warmB)*j/barWidth

		char := "─"
		if j == pos {
			char = "●"
		}
		bar += lipgloss.NewStyle().Foreground(lipgloss.Color(fmt.Sprintf("#%02x%02x%02x", r, g, b))).Render(char)
	}
	// Show Kelvin approximation
	kelvin := 1000000 / max(1, field.Value)
	return bar + fmt.Sprintf(" %dK", kelvin)
}

// renderColor renders a color field.
// When not editing, shows a preview. When editing, the caller should render the wheel separately.
func renderColor(field *FormField, styles *ui2.Styles) string {
	r, g, b := ui2.XyToRGB(field.ColorX, field.ColorY, 1.0)
	style := lipgloss.NewStyle().Foreground(lipgloss.Color(fmt.Sprintf("#%02x%02x%02x", r, g, b)))
	return style.Render("███") + " (Enter to edit)"
}

// RenderColorWheel renders the full color wheel for a color field.
// This should be called when the field is being edited.
func RenderColorWheel(field *FormField) string {
	wheel := NewColorWheel()
	wheel.SetColor(field.ColorX, field.ColorY)
	wheel.BlinkOn = true

	var out strings.Builder
	out.WriteString("\n")
	for _, line := range strings.Split(wheel.Render(), "\n") {
		if line != "" {
			out.WriteString("    " + line + "\n")
		}
	}
	return out.String()
}

// renderSelect renders a select dropdown field.
func renderSelect(field *FormField, dropdownOpen bool, dropdownScroll int) string {
	if dropdownOpen {
		return renderDropdown(field, dropdownScroll)
	}
	currentLabel := "---"
	for _, opt := range field.Options {
		if opt.Value == field.Value {
			currentLabel = opt.Label
			break
		}
	}
	return fmt.Sprintf("[%s] ▼", currentLabel)
}

// renderDropdown renders the dropdown menu for a select field.
func renderDropdown(field *FormField, scroll int) string {
	maxVisible := 5
	start := scroll
	end := min(start+maxVisible, len(field.Options))
	if start < 0 {
		start = 0
	}

	maxLen := 0
	for _, opt := range field.Options {
		if len(opt.Label) > maxLen {
			maxLen = len(opt.Label)
		}
	}

	var out strings.Builder
	out.WriteString("┌" + strings.Repeat("─", maxLen+2) + "┐\n")
	for i := start; i < end; i++ {
		opt := field.Options[i]
		prefix := "  "
		if i == field.Value {
			prefix = "> "
		}
		label := opt.Label + strings.Repeat(" ", maxLen-len(opt.Label))
		out.WriteString(fmt.Sprintf("    │%s%s│\n", prefix, label))
	}
	out.WriteString("    └" + strings.Repeat("─", maxLen+2) + "┘")
	return out.String()
}

// renderRadio renders a radio button group.
// For vertical: subRow indicates which option to render.
// For horizontal: subRow should be 0, renders all options.
func renderRadio(field *FormField, isFocused bool, styles *ui2.Styles, subRow int) string {
	if field.Vertical {
		if subRow < len(field.Options) {
			opt := field.Options[subRow]
			indicator := "○"
			if opt.Value == field.Value {
				indicator = "●"
			}
			return fmt.Sprintf("%s %s", indicator, opt.Label)
		}
		return ""
	}

	// Horizontal: render all options
	var parts []string
	for _, opt := range field.Options {
		indicator := "○"
		if opt.Value == field.Value {
			indicator = "●"
		}
		parts = append(parts, fmt.Sprintf("%s %s", indicator, opt.Label))
	}
	return strings.Join(parts, "  ")
}

// renderHSL renders an HSL color picker (3 rows: Hue, Saturation, Lightness).
// subRow: 0=Hue, 1=Saturation, 2=Lightness
func renderHSL(field *FormField, isFocused bool, styles *ui2.Styles, subRow int) string {
	sliderWidth := 20
	rowLabels := []string{"H", "S", "L"}
	rowValues := []int{field.Hue, field.Saturation, field.Lightness}
	rowMaxes := []int{360, 100, 100}

	if subRow >= 3 {
		return ""
	}

	rowVal := rowValues[subRow]
	rowMax := rowMaxes[subRow]
	pos := rowVal * sliderWidth / max(1, rowMax)
	if pos >= sliderWidth {
		pos = sliderWidth - 1
	}

	var bar string
	if subRow == 0 {
		// Hue - rainbow gradient
		for j := 0; j < sliderWidth; j++ {
			h := j * 360 / sliderWidth
			r, g, b := ui2.HsvToRGB(h, 100, 100)
			char := "─"
			if j == pos {
				char = "●"
			}
			bar += lipgloss.NewStyle().Foreground(lipgloss.Color(fmt.Sprintf("#%02x%02x%02x", r, g, b))).Render(char)
		}
	} else if subRow == 1 {
		// Saturation - gray to full color
		for j := 0; j < sliderWidth; j++ {
			s := j * 100 / sliderWidth
			r, g, b := ui2.HsvToRGB(field.Hue, s, 100)
			char := "─"
			if j == pos {
				char = "●"
			}
			bar += lipgloss.NewStyle().Foreground(lipgloss.Color(fmt.Sprintf("#%02x%02x%02x", r, g, b))).Render(char)
		}
	} else {
		// Lightness - black to white through color
		for j := 0; j < sliderWidth; j++ {
			l := j * 100 / sliderWidth
			r, g, b := ui2.HsvToRGB(field.Hue, field.Saturation, l)
			char := "─"
			if j == pos {
				char = "●"
			}
			bar += lipgloss.NewStyle().Foreground(lipgloss.Color(fmt.Sprintf("#%02x%02x%02x", r, g, b))).Render(char)
		}
	}

	// Row highlight when editing (caller should handle this)
	rowLabel := rowLabels[subRow]
	valueStr := fmt.Sprintf("%s: %s %3d", rowLabel, bar, rowVal)

	// Show color preview as vertical stripe with rounded corners
	pr, pg, pb := ui2.HsvToRGB(field.Hue, field.Saturation, field.Lightness)
	colorHex := fmt.Sprintf("#%02x%02x%02x", pr, pg, pb)
	colorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(colorHex))
	bgStyle := lipgloss.NewStyle().Background(lipgloss.Color(colorHex))
	var preview string
	switch subRow {
	case 0: // Top row - rounded corners
		preview = " " + colorStyle.Render("▄▄") + " "
	case 2: // Bottom row - rounded corners
		preview = " " + colorStyle.Render("▀▀") + " "
	default: // Middle row - full block
		preview = bgStyle.Render("    ")
	}
	return valueStr + " " + preview
}

// renderRGB renders an RGB color picker (3 rows: Red, Green, Blue).
// subRow: 0=Red, 1=Green, 2=Blue
func renderRGB(field *FormField, isFocused bool, styles *ui2.Styles, subRow int) string {
	sliderWidth := 20
	rowLabels := []string{"R", "G", "B"}
	rowValues := []int{field.Red, field.Green, field.Blue}

	if subRow >= 3 {
		return ""
	}

	rowVal := rowValues[subRow]
	pos := rowVal * sliderWidth / 255
	if pos >= sliderWidth {
		pos = sliderWidth - 1
	}

	var bar string
	for j := 0; j < sliderWidth; j++ {
		intensity := j * 255 / sliderWidth
		var r, g, b int
		switch subRow {
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

	// Row highlight when editing (caller should handle this)
	rowLabel := rowLabels[subRow]
	valueStr := fmt.Sprintf("%s: %s %3d", rowLabel, bar, rowVal)

	// Show color preview as vertical stripe with rounded corners
	colorHex := fmt.Sprintf("#%02x%02x%02x", field.Red, field.Green, field.Blue)
	colorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(colorHex))
	bgStyle := lipgloss.NewStyle().Background(lipgloss.Color(colorHex))
	var preview string
	switch subRow {
	case 0: // Top row - rounded corners
		preview = " " + colorStyle.Render("▄▄") + " "
	case 2: // Bottom row - rounded corners
		preview = " " + colorStyle.Render("▀▀") + " "
	default: // Middle row - full block
		preview = bgStyle.Render("    ")
	}
	return valueStr + " " + preview
}
