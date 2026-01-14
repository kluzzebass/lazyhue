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

// HSLComponent is an HSL color picker with 3 sliders.
type HSLComponent struct {
	*BaseField

	// Current HSL values
	Hue        int // 0-360
	Saturation int // 0-100
	Lightness  int // 0-100

	// Original values for cancel
	OriginalHue        int
	OriginalSaturation int
	OriginalLightness  int

	// Which slider is focused when editing (0=H, 1=S, 2=L)
	SliderFocus int

	// Slider bar width
	sliderWidth int
}

// NewHSLComponent creates a new HSL color picker component.
func NewHSLComponent(id, label string, hue, saturation, lightness int, styles *ui2.Styles, zones *zone.Manager) *HSLComponent {
	return &HSLComponent{
		BaseField:   NewBaseField(id, label, styles, zones),
		Hue:         hue,
		Saturation:  saturation,
		Lightness:   lightness,
		sliderWidth: 20,
	}
}

// SetHSL sets the HSL values.
func (h *HSLComponent) SetHSL(hue, saturation, lightness int) {
	h.Hue = hue
	h.Saturation = saturation
	h.Lightness = lightness
}

// Update handles events for the HSL picker.
func (h *HSLComponent) Update(msg tea.Msg) (component.Component, tea.Cmd) {
	if h.ReadOnly {
		return h, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if h.Editing {
			return h.handleEditingKey(msg)
		}
		return h.handleNormalKey(msg)

	case tea.MouseClickMsg:
		return h.handleMouseClick(msg)
	}

	return h, nil
}

// RouteEvent routes events to this component.
func (h *HSLComponent) RouteEvent(msg tea.Msg) (bool, tea.Cmd) {
	if h.ReadOnly {
		return false, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if h.Editing {
			_, cmd := h.handleEditingKey(msg)
			return true, cmd // Editing captures all keys
		}
		switch msg.String() {
		case "enter", " ":
			_, cmd := h.startEditing()
			return true, cmd
		}

	case tea.MouseClickMsg:
		if msg.Button == tea.MouseLeft {
			// Check slider row zones
			for row := 0; row < 3; row++ {
				rowZoneID := fmt.Sprintf("%s-row-%d", h.ZoneID(), row)
				if h.Zones != nil {
					if z := h.Zones.Get(rowZoneID); z != nil && z.InBounds(msg) {
						_, cmd := h.handleSliderClick(msg, row, z)
						return true, cmd
					}
				}
			}
		}
	}

	return false, nil
}

func (h *HSLComponent) handleNormalKey(msg tea.KeyMsg) (component.Component, tea.Cmd) {
	switch msg.String() {
	case "enter", " ":
		return h.startEditing()
	}
	return h, nil
}

func (h *HSLComponent) handleEditingKey(msg tea.KeyMsg) (component.Component, tea.Cmd) {
	switch msg.String() {
	case "esc":
		// Cancel and restore original values
		h.Hue = h.OriginalHue
		h.Saturation = h.OriginalSaturation
		h.Lightness = h.OriginalLightness
		h.Editing = false
		return h, nil

	case "enter":
		// Confirm edit
		h.Editing = false
		return h, func() tea.Msg {
			return FieldChangedMsg{
				FieldID: h.ID,
				Value:   HSLValue{Hue: h.Hue, Saturation: h.Saturation, Lightness: h.Lightness},
			}
		}

	case "up", "k":
		// Move to previous slider
		if h.SliderFocus > 0 {
			h.SliderFocus--
		}

	case "down", "j":
		// Move to next slider
		if h.SliderFocus < 2 {
			h.SliderFocus++
		}

	case "left", "h":
		h.adjustSlider(-1)

	case "right", "l":
		h.adjustSlider(1)
	}

	return h, nil
}

func (h *HSLComponent) adjustSlider(direction int) {
	step := 5
	switch h.SliderFocus {
	case 0: // Hue
		step = 10 // Hue has larger range (0-360)
		h.Hue = clamp(h.Hue+direction*step, 0, 360)
	case 1: // Saturation
		h.Saturation = clamp(h.Saturation+direction*step, 0, 100)
	case 2: // Lightness
		h.Lightness = clamp(h.Lightness+direction*step, 0, 100)
	}
}

func (h *HSLComponent) handleMouseClick(msg tea.MouseClickMsg) (component.Component, tea.Cmd) {
	if !h.Editing {
		// Click on preview to start editing
		if h.Zones != nil {
			if z := h.Zones.Get(h.ZoneID()); z != nil && z.InBounds(msg) {
				return h.startEditing()
			}
		}
	}
	return h, nil
}

func (h *HSLComponent) handleSliderClick(msg tea.MouseClickMsg, row int, z *zone.ZoneInfo) (component.Component, tea.Cmd) {
	if !h.Editing {
		h.startEditing()
	}

	h.SliderFocus = row

	// Calculate position within slider bar
	// The slider bar starts after "  H: " (5 chars from zone start)
	barStartOffset := 5
	clickPos := msg.X - z.StartX - barStartOffset
	if clickPos < 0 {
		clickPos = 0
	}
	if clickPos >= h.sliderWidth {
		clickPos = h.sliderWidth - 1
	}

	// Convert position to value
	switch row {
	case 0: // Hue (0-360)
		h.Hue = clickPos * 360 / h.sliderWidth
	case 1: // Saturation (0-100)
		h.Saturation = clickPos * 100 / h.sliderWidth
	case 2: // Lightness (0-100)
		h.Lightness = clickPos * 100 / h.sliderWidth
	}

	return h, nil
}

func (h *HSLComponent) startEditing() (component.Component, tea.Cmd) {
	h.Editing = true
	h.OriginalHue = h.Hue
	h.OriginalSaturation = h.Saturation
	h.OriginalLightness = h.Lightness
	h.SliderFocus = 0
	return h, nil
}

// FieldHeight returns the number of rows this component takes up.
func (h *HSLComponent) FieldHeight() int {
	return 3 // Always 3 rows for H, S, L sliders
}

// ViewControl renders only the control portion (no label).
// Returns 3 lines separated by newlines.
func (h *HSLComponent) ViewControl() string {
	var out strings.Builder

	for row := 0; row < 3; row++ {
		if row > 0 {
			out.WriteString("\n")
		}

		sliderLine := h.renderSliderRow(row)

		// Highlight focused slider when editing
		if h.Editing && row == h.SliderFocus {
			sliderLine = h.Styles.Focused.Render(sliderLine)
		}

		// Mark row with zone
		if h.Zones != nil {
			rowZoneID := fmt.Sprintf("%s-row-%d", h.ZoneID(), row)
			sliderLine = h.Zones.Mark(rowZoneID, sliderLine)
		}

		out.WriteString(sliderLine)
	}

	return out.String()
}

// View renders the HSL picker (label + control for backwards compatibility).
func (h *HSLComponent) View() string {
	labelStr := h.Label
	if h.MaxLabelWidth > 0 {
		labelStr = fmt.Sprintf("%-*s", h.MaxLabelWidth, h.Label)
	}

	control := h.ViewControl()
	controlLines := strings.Split(control, "\n")

	var out strings.Builder

	for i, controlLine := range controlLines {
		if i > 0 {
			out.WriteString("\n")
		}

		var rowLabel string
		if i == 0 {
			rowLabel = fmt.Sprintf("  %s  ", labelStr)
		} else {
			// Indent subsequent rows
			rowLabel = strings.Repeat(" ", 2+h.MaxLabelWidth+2)
		}

		out.WriteString(rowLabel + controlLine)
	}

	return out.String()
}

func (h *HSLComponent) renderSliderRow(row int) string {
	rowLabels := []string{"H", "S", "L"}
	rowValues := []int{h.Hue, h.Saturation, h.Lightness}
	rowMaxes := []int{360, 100, 100}

	rowVal := rowValues[row]
	rowMax := rowMaxes[row]
	pos := rowVal * h.sliderWidth / maxInt(1, rowMax)
	if pos >= h.sliderWidth {
		pos = h.sliderWidth - 1
	}

	var bar string
	switch row {
	case 0: // Hue - rainbow gradient
		for j := 0; j < h.sliderWidth; j++ {
			hVal := j * 360 / h.sliderWidth
			r, g, b := ui2.HsvToRGB(hVal, 100, 100)
			char := "─"
			if j == pos {
				char = "●"
			}
			bar += lipgloss.NewStyle().Foreground(lipgloss.Color(fmt.Sprintf("#%02x%02x%02x", r, g, b))).Render(char)
		}

	case 1: // Saturation - gray to full color
		for j := 0; j < h.sliderWidth; j++ {
			s := j * 100 / h.sliderWidth
			r, g, b := ui2.HsvToRGB(h.Hue, s, 100)
			char := "─"
			if j == pos {
				char = "●"
			}
			bar += lipgloss.NewStyle().Foreground(lipgloss.Color(fmt.Sprintf("#%02x%02x%02x", r, g, b))).Render(char)
		}

	case 2: // Lightness - black to white through color
		for j := 0; j < h.sliderWidth; j++ {
			l := j * 100 / h.sliderWidth
			r, g, b := ui2.HsvToRGB(h.Hue, h.Saturation, l)
			char := "─"
			if j == pos {
				char = "●"
			}
			bar += lipgloss.NewStyle().Foreground(lipgloss.Color(fmt.Sprintf("#%02x%02x%02x", r, g, b))).Render(char)
		}
	}

	// Value display
	valueStr := fmt.Sprintf("%s: %s %3d", rowLabels[row], bar, rowVal)

	// Color preview stripe
	pr, pg, pb := ui2.HsvToRGB(h.Hue, h.Saturation, h.Lightness)
	colorHex := fmt.Sprintf("#%02x%02x%02x", pr, pg, pb)
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
