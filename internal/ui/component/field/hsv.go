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

// HSVComponent is an HSV color picker with 3 sliders.
type HSVComponent struct {
	*BaseField

	// Current HSV values
	Hue        int // 0-360
	Saturation int // 0-100
	Value      int // 0-100

	// Original values for cancel
	OriginalHue        int
	OriginalSaturation int
	OriginalValue      int

	// Which slider is focused when editing (0=H, 1=S, 2=V)
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

// NewHSVComponent creates a new HSV color picker component.
func NewHSVComponent(id, label string, hue, saturation, value int, styles *ui.Styles, zones *zone.Manager) *HSVComponent {
	return &HSVComponent{
		BaseField:   NewBaseField(id, label, styles, zones),
		Hue:         hue,
		Saturation:  saturation,
		Value:       value,
		sliderWidth: 20,
		ShowSwatch:  true,
	}
}

// SetHSV sets the HSV values.
func (h *HSVComponent) SetHSV(hue, saturation, value int) {
	h.Hue = hue
	h.Saturation = saturation
	h.Value = value
}

// Update handles events for the HSV picker.
func (h *HSVComponent) Update(msg tea.Msg) (component.Component, tea.Cmd) {
	// Events are handled via RouteEvent
	return h, nil
}

// RouteEvent routes events to this component.
func (h *HSVComponent) RouteEvent(msg tea.Msg) (bool, tea.Cmd) {
	if h.ReadOnly || h.Inactive {
		return false, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "left", "h":
			if !h.Editing {
				h.saveOriginal()
			}
			h.adjustSlider(-1)
			return true, h.emitChange()

		case "right", "l":
			if !h.Editing {
				h.saveOriginal()
			}
			h.adjustSlider(1)
			return true, h.emitChange()

		case "up", "k":
			if h.SliderFocus > 0 {
				h.SliderFocus--
				return true, nil
			}
			return false, nil

		case "down", "j":
			if h.SliderFocus < 2 {
				h.SliderFocus++
				return true, nil
			}
			return false, nil

		case "esc":
			if h.Editing {
				h.restoreOriginal()
				h.Editing = false
				return true, nil
			}

		case "enter", " ":
			if h.Editing {
				h.Editing = false
				return true, h.emitChange()
			}
		}

	case tea.MouseClickMsg:
		if msg.Button == tea.MouseLeft {
			for row := range 3 {
				rowZoneID := fmt.Sprintf("%s-row-%d", h.ZoneID(), row)
				if h.Zones != nil {
					if z := h.Zones.Get(rowZoneID); z != nil && z.InBounds(msg) {
						_, cmd := h.handleSliderClick(msg, row, z)
						return true, cmd
					}
				}
			}
		}

	case tea.MouseMotionMsg:
		if h.Dragging && h.DragZoneInfo != nil {
			h.updateFromMouseX(msg.X)
			return true, h.emitChange()
		}

	case tea.MouseReleaseMsg:
		if h.Dragging {
			h.Dragging = false
			h.DragZoneInfo = nil
			return true, tea.Batch(
				func() tea.Msg { return EndCaptureMsg{FieldID: h.ID} },
				h.emitChange(),
			)
		}
	}

	return false, nil
}

func (h *HSVComponent) updateFromMouseX(mouseX int) {
	if h.DragZoneInfo == nil {
		return
	}
	barStartOffset := 3 // "H: " prefix
	clickPos := mouseX - h.DragZoneInfo.StartX - barStartOffset
	if clickPos < 0 {
		clickPos = 0
	}
	if clickPos > h.sliderWidth-1 {
		clickPos = h.sliderWidth - 1
	}
	// Map 0..sliderWidth-1 to full range
	switch h.SliderFocus {
	case 0: // Hue (0-360)
		h.Hue = clickPos * 360 / (h.sliderWidth - 1)
	case 1: // Saturation (0-100)
		h.Saturation = clickPos * 100 / (h.sliderWidth - 1)
	case 2: // Value (0-100)
		h.Value = clickPos * 100 / (h.sliderWidth - 1)
	}
}

func (h *HSVComponent) saveOriginal() {
	h.Editing = true
	h.OriginalHue = h.Hue
	h.OriginalSaturation = h.Saturation
	h.OriginalValue = h.Value
}

func (h *HSVComponent) restoreOriginal() {
	h.Hue = h.OriginalHue
	h.Saturation = h.OriginalSaturation
	h.Value = h.OriginalValue
}

func (h *HSVComponent) emitChange() tea.Cmd {
	return func() tea.Msg {
		return FieldChangedMsg{
			FieldID: h.ID,
			Value:   HSVValue{Hue: h.Hue, Saturation: h.Saturation, Value: h.Value},
		}
	}
}

func (h *HSVComponent) adjustSlider(direction int) {
	step := 5
	switch h.SliderFocus {
	case 0: // Hue
		step = 10 // Hue has larger range (0-360)
		h.Hue = clamp(h.Hue+direction*step, 0, 360)
	case 1: // Saturation
		h.Saturation = clamp(h.Saturation+direction*step, 0, 100)
	case 2: // Value
		h.Value = clamp(h.Value+direction*step, 0, 100)
	}
}

func (h *HSVComponent) handleSliderClick(msg tea.MouseClickMsg, row int, z *zone.ZoneInfo) (component.Component, tea.Cmd) {
	if !h.Editing {
		h.saveOriginal()
	}

	h.SliderFocus = row

	// Calculate position within slider bar
	// The slider bar starts after "H: " (3 chars from zone start)
	barStartOffset := 3
	clickPos := msg.X - z.StartX - barStartOffset
	if clickPos < 0 {
		clickPos = 0
	}
	if clickPos > h.sliderWidth-1 {
		clickPos = h.sliderWidth - 1
	}

	// Convert position to value (map 0..sliderWidth-1 to full range)
	switch row {
	case 0: // Hue (0-360)
		h.Hue = clickPos * 360 / (h.sliderWidth - 1)
	case 1: // Saturation (0-100)
		h.Saturation = clickPos * 100 / (h.sliderWidth - 1)
	case 2: // Value (0-100)
		h.Value = clickPos * 100 / (h.sliderWidth - 1)
	}

	// Start drag mode
	h.Dragging = true
	h.DragZoneInfo = z

	return h, tea.Batch(
		func() tea.Msg { return StartCaptureMsg{FieldID: h.ID} },
		h.emitChange(),
	)
}

// FieldHeight returns the number of rows this component takes up.
func (h *HSVComponent) FieldHeight() int {
	return 3 // Always 3 rows for H, S, V sliders
}

// ViewControl renders only the control portion (no label).
// Returns 3 lines separated by newlines.
func (h *HSVComponent) ViewControl() string {
	var out strings.Builder

	for row := range 3 {
		if row > 0 {
			out.WriteString("\n")
		}

		sliderLine := h.renderSliderRow(row)

		// Highlight focused slider when component is focused
		if h.IsFocused() && row == h.SliderFocus {
			sliderLine = h.Styles.Selected.Render(sliderLine)
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

// View renders the HSV picker (label + control for backwards compatibility).
func (h *HSVComponent) View() string {
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

func (h *HSVComponent) renderSliderRow(row int) string {
	rowLabels := []string{"H", "S", "V"}
	rowValues := []int{h.Hue, h.Saturation, h.Value}
	rowMaxes := []int{360, 100, 100}

	rowVal := rowValues[row]
	rowMax := rowMaxes[row]
	pos := rowVal * h.sliderWidth / max(1, rowMax)
	if pos >= h.sliderWidth {
		pos = h.sliderWidth - 1
	}

	var bar string
	switch row {
	case 0: // Hue - rainbow gradient
		for j := range h.sliderWidth {
			hVal := j * 360 / h.sliderWidth
			r, g, b := ui.HsvToRGB(hVal, 100, 100)
			if h.Inactive {
				gray := (int(r) + int(g) + int(b)) / 3
				r, g, b = uint8(gray), uint8(gray), uint8(gray)
			}
			char := "─"
			if j == pos {
				char = "●"
			}
			bar += lipgloss.NewStyle().Foreground(lipgloss.Color(fmt.Sprintf("#%02x%02x%02x", r, g, b))).Render(char)
		}

	case 1: // Saturation - gray to full color
		for j := range h.sliderWidth {
			s := j * 100 / h.sliderWidth
			r, g, b := ui.HsvToRGB(h.Hue, s, 100) // Full value for saturation preview
			if h.Inactive {
				gray := (int(r) + int(g) + int(b)) / 3
				r, g, b = uint8(gray), uint8(gray), uint8(gray)
			}
			char := "─"
			if j == pos {
				char = "●"
			}
			bar += lipgloss.NewStyle().Foreground(lipgloss.Color(fmt.Sprintf("#%02x%02x%02x", r, g, b))).Render(char)
		}

	case 2: // Value - black to full color
		for j := range h.sliderWidth {
			v := j * 100 / h.sliderWidth
			r, g, b := ui.HsvToRGB(h.Hue, h.Saturation, v)
			if h.Inactive {
				gray := (int(r) + int(g) + int(b)) / 3
				r, g, b = uint8(gray), uint8(gray), uint8(gray)
			}
			char := "─"
			if j == pos {
				char = "●"
			}
			bar += lipgloss.NewStyle().Foreground(lipgloss.Color(fmt.Sprintf("#%02x%02x%02x", r, g, b))).Render(char)
		}
	}

	// Value display
	valueStr := fmt.Sprintf("%s: %s %3d", rowLabels[row], bar, rowVal)

	if !h.ShowSwatch {
		return valueStr
	}

	// Color preview stripe
	pr, pg, pb := ui.HsvToRGB(h.Hue, h.Saturation, h.Value)
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
