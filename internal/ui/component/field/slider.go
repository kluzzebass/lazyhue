package field

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/kluzzebass/lazyhue/internal/ui"
	"github.com/kluzzebass/lazyhue/internal/ui/component"
	zone "github.com/lrstanley/bubblezone/v2"
)

// SliderComponent is a base slider field.
// Use BrightnessSliderComponent or ColorTempSliderComponent for specific slider types.
type SliderComponent struct {
	*BaseField

	Value int
	Min   int
	Max   int
	Step  int // Amount to change on arrow key press

	// Mouse capture state
	Dragging       bool
	DragZoneStartX int // Zone start X when drag started

	// Rendering customization (set by subclasses)
	RenderBar   func(value, min, max, width int, styles *ui.Styles) string
	FormatValue func(value, min, max int) string
	Inverted    bool // If true, left=max, right=min (for color temp)
	Inactive    bool // If true, render in grayscale (not the active color mode)
}

// NewSliderComponent creates a new slider component.
func NewSliderComponent(id, label string, value, min, max, step int, styles *ui.Styles, zones *zone.Manager) *SliderComponent {
	return &SliderComponent{
		BaseField:      NewBaseField(id, label, styles, zones),
		Value:          value,
		Min:            min,
		Max:            max,
		Step:           step,
		DragZoneStartX: -1,
	}
}

// SetValue sets the slider value.
func (s *SliderComponent) SetValue(value int) {
	s.Value = clamp(value, s.Min, s.Max)
}

// Update handles events for the slider.
func (s *SliderComponent) Update(msg tea.Msg) (component.Component, tea.Cmd) {
	if s.ReadOnly {
		return s, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		return s.handleKey(msg)

	case tea.MouseClickMsg:
		return s.handleMouseClick(msg)

	case tea.MouseMotionMsg:
		if s.Dragging {
			return s.handleMouseDrag(msg.X)
		}

	case tea.MouseReleaseMsg:
		if s.Dragging {
			s.Dragging = false
			s.DragZoneStartX = -1
			return s, func() tea.Msg { return EndCaptureMsg{FieldID: s.ID} }
		}
	}

	return s, nil
}

// RouteEvent routes events to this component.
func (s *SliderComponent) RouteEvent(msg tea.Msg) (bool, tea.Cmd) {
	if s.ReadOnly {
		return false, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "left", "h":
			_, cmd := s.adjustValue(-s.Step)
			return true, cmd
		case "right", "l":
			_, cmd := s.adjustValue(s.Step)
			return true, cmd
		}

	case tea.MouseClickMsg:
		if msg.Button == tea.MouseLeft {
			if s.Zones != nil {
				if z := s.Zones.Get(s.ZoneID()); z != nil && z.InBounds(msg) {
					_, cmd := s.handleMouseClick(msg)
					return true, cmd
				}
			}
		}

	case tea.MouseMotionMsg:
		if s.Dragging {
			_, cmd := s.handleMouseDrag(msg.X)
			return true, cmd
		}

	case tea.MouseReleaseMsg:
		if s.Dragging {
			s.Dragging = false
			s.DragZoneStartX = -1
			return true, func() tea.Msg { return EndCaptureMsg{FieldID: s.ID} }
		}
	}

	return false, nil
}

func (s *SliderComponent) handleKey(msg tea.KeyMsg) (component.Component, tea.Cmd) {
	switch msg.String() {
	case "left", "h":
		return s.adjustValue(-s.Step)
	case "right", "l":
		return s.adjustValue(s.Step)
	}
	return s, nil
}

func (s *SliderComponent) adjustValue(delta int) (component.Component, tea.Cmd) {
	if s.Inverted {
		delta = -delta
	}

	newValue := clamp(s.Value+delta, s.Min, s.Max)
	if newValue == s.Value {
		return s, nil
	}

	s.Value = newValue
	return s, func() tea.Msg {
		return FieldChangedMsg{
			FieldID: s.ID,
			Value:   SliderValue{Value: s.Value},
		}
	}
}

func (s *SliderComponent) handleMouseClick(msg tea.MouseClickMsg) (component.Component, tea.Cmd) {
	if s.Zones == nil {
		return s, nil
	}

	zoneInfo := s.Zones.Get(s.ZoneID())
	if zoneInfo == nil {
		return s, nil
	}

	// Find zone start X by binary search
	zoneStartX := s.findZoneStartX(zoneInfo, msg.X, msg.Y)
	if zoneStartX < 0 {
		return s, nil
	}

	// Calculate value from position
	s.DragZoneStartX = zoneStartX
	s.Dragging = true

	// Calculate new value
	newValue := s.calculateValueFromX(msg.X, zoneStartX)
	if newValue != s.Value {
		s.Value = newValue
		return s, tea.Batch(
			func() tea.Msg { return StartCaptureMsg{FieldID: s.ID} },
			func() tea.Msg { return FieldChangedMsg{FieldID: s.ID, Value: SliderValue{Value: s.Value}} },
		)
	}

	return s, func() tea.Msg { return StartCaptureMsg{FieldID: s.ID} }
}

func (s *SliderComponent) handleMouseDrag(mouseX int) (component.Component, tea.Cmd) {
	if s.DragZoneStartX < 0 {
		return s, nil
	}

	newValue := s.calculateValueFromX(mouseX, s.DragZoneStartX)
	if newValue != s.Value {
		s.Value = newValue
		return s, func() tea.Msg {
			return FieldChangedMsg{FieldID: s.ID, Value: SliderValue{Value: s.Value}}
		}
	}

	return s, nil
}

func (s *SliderComponent) findZoneStartX(zoneInfo *zone.ZoneInfo, mouseX, mouseY int) int {
	// Binary search to find zone's left edge
	leftBound := 0
	rightBound := mouseX

	for leftBound < rightBound {
		mid := (leftBound + rightBound) / 2
		if zoneInfo.InBounds(tea.MouseClickMsg{X: mid, Y: mouseY}) {
			rightBound = mid
		} else {
			leftBound = mid + 1
		}
	}

	return leftBound
}

func (s *SliderComponent) calculateValueFromX(mouseX, zoneStartX int) int {
	barWidth := 20 // Standard slider bar width

	// Calculate relative position within the bar (0 to barWidth-1)
	relativeX := mouseX - zoneStartX
	relativeX = clamp(relativeX, 0, barWidth-1)

	// Map position to value range
	// Position 0 = Min, Position barWidth-1 = Max
	var newValue int
	if s.Inverted {
		// Inverted: left = max, right = min
		newValue = s.Max - (relativeX * (s.Max - s.Min) / (barWidth - 1))
	} else {
		// Normal: left = min, right = max
		newValue = s.Min + (relativeX * (s.Max - s.Min) / (barWidth - 1))
	}

	return clamp(newValue, s.Min, s.Max)
}

// ViewControl renders only the control portion (no label).
func (s *SliderComponent) ViewControl() string {
	barWidth := 20

	// Build the bar
	var barStr string
	if s.RenderBar != nil {
		barStr = s.RenderBar(s.Value, s.Min, s.Max, barWidth, s.Styles)
	} else {
		barStr = s.defaultRenderBar(barWidth)
	}

	// Build the value display
	var valueStr string
	if s.FormatValue != nil {
		valueStr = s.FormatValue(s.Value, s.Min, s.Max)
	} else {
		valueStr = fmt.Sprintf("%d", s.Value)
	}

	control := barStr + " " + valueStr

	// Wrap with zone for mouse detection
	if s.Zones != nil {
		return s.Zones.Mark(s.ZoneID(), control)
	}

	return control
}

// View renders the slider field (label + control for backwards compatibility).
func (s *SliderComponent) View() string {
	labelStr := s.Label
	if s.MaxLabelWidth > 0 {
		labelStr = fmt.Sprintf("%-*s", s.MaxLabelWidth, s.Label)
	}

	return fmt.Sprintf("  %s  %s", labelStr, s.ViewControl())
}

func (s *SliderComponent) defaultRenderBar(width int) string {
	if s.Max <= s.Min {
		return ""
	}

	filled := (s.Value - s.Min) * width / (s.Max - s.Min)
	filled = clamp(filled, 0, width)

	bar := ""
	for i := 0; i < width; i++ {
		if i < filled {
			bar += "█"
		} else {
			bar += "░"
		}
	}

	return bar
}

// Helper function
func clamp(v, min, max int) int {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}
