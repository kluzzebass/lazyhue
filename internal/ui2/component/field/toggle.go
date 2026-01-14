package field

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/kluzzebass/lazyhue/internal/ui2"
	"github.com/kluzzebass/lazyhue/internal/ui2/component"
	zone "github.com/lrstanley/bubblezone/v2"
)

// ToggleComponent is a boolean on/off toggle field.
type ToggleComponent struct {
	*BaseField

	Value    bool
	OnLabel  string
	OffLabel string
}

// NewToggleComponent creates a new toggle component.
func NewToggleComponent(id, label string, value bool, styles *ui2.Styles, zones *zone.Manager) *ToggleComponent {
	return &ToggleComponent{
		BaseField: NewBaseField(id, label, styles, zones),
		Value:     value,
		OnLabel:   "On",
		OffLabel:  "Off",
	}
}

// SetLabels sets custom on/off labels.
func (t *ToggleComponent) SetLabels(onLabel, offLabel string) {
	if onLabel != "" {
		t.OnLabel = onLabel
	}
	if offLabel != "" {
		t.OffLabel = offLabel
	}
}

// SetValue sets the toggle value.
func (t *ToggleComponent) SetValue(value bool) {
	t.Value = value
}

// Update handles events for the toggle.
func (t *ToggleComponent) Update(msg tea.Msg) (component.Component, tea.Cmd) {
	if t.ReadOnly {
		return t, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter", " ":
			return t.toggle()
		}

	case tea.MouseClickMsg:
		if msg.Button == tea.MouseLeft {
			// Check if click is within our zone
			if t.Zones != nil {
				if z := t.Zones.Get(t.ZoneID()); z != nil && z.InBounds(msg) {
					return t.toggle()
				}
			}
		}
	}

	return t, nil
}

// RouteEvent routes events to this component.
func (t *ToggleComponent) RouteEvent(msg tea.Msg) (bool, tea.Cmd) {
	if t.ReadOnly {
		return false, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter", " ":
			_, cmd := t.toggle()
			return true, cmd
		}

	case tea.MouseClickMsg:
		if msg.Button == tea.MouseLeft {
			if t.Zones != nil {
				if z := t.Zones.Get(t.ZoneID()); z != nil && z.InBounds(msg) {
					_, cmd := t.toggle()
					return true, cmd
				}
			}
		}
	}

	return false, nil
}

// toggle flips the value and returns a change message.
func (t *ToggleComponent) toggle() (component.Component, tea.Cmd) {
	t.Value = !t.Value

	return t, func() tea.Msg {
		return FieldChangedMsg{
			FieldID: t.ID,
			Value:   ToggleValue{On: t.Value},
		}
	}
}

// ViewControl renders only the control portion (no label).
func (t *ToggleComponent) ViewControl() string {
	var valueStr string
	if t.Value {
		valueStr = "[●] " + t.OnLabel
	} else {
		valueStr = "[ ] " + t.OffLabel
	}

	// Wrap with zone for mouse detection
	if t.Zones != nil {
		return t.Zones.Mark(t.ZoneID(), valueStr)
	}

	return valueStr
}

// View renders the toggle field (label + control for backwards compatibility).
func (t *ToggleComponent) View() string {
	labelStr := t.Label
	if t.MaxLabelWidth > 0 {
		labelStr = fmt.Sprintf("%-*s", t.MaxLabelWidth, t.Label)
	}

	// Combine label and control
	return fmt.Sprintf("  %s  %s", labelStr, t.ViewControl())
}
