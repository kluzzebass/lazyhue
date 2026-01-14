package field

import (
	"fmt"

	"github.com/charmbracelet/bubbles/v2/textinput"
	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/kluzzebass/lazyhue/internal/ui2"
	"github.com/kluzzebass/lazyhue/internal/ui2/component"
	zone "github.com/lrstanley/bubblezone/v2"
)

// TextComponent is a text input field.
type TextComponent struct {
	*BaseField

	Value         string
	OriginalValue string // Value before editing (for cancel)

	// The textinput model
	Input textinput.Model
}

// NewTextComponent creates a new text component.
func NewTextComponent(id, label, value string, styles *ui2.Styles, zones *zone.Manager) *TextComponent {
	input := textinput.New()
	input.SetValue(value)
	input.Prompt = ""
	input.CharLimit = 64

	return &TextComponent{
		BaseField: NewBaseField(id, label, styles, zones),
		Value:     value,
		Input:     input,
	}
}

// SetValue sets the text value.
func (t *TextComponent) SetValue(value string) {
	t.Value = value
	t.Input.SetValue(value)
}

// Update handles events for the text field.
func (t *TextComponent) Update(msg tea.Msg) (component.Component, tea.Cmd) {
	if t.ReadOnly {
		return t, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if t.Editing {
			return t.handleEditingKey(msg)
		}
		return t.handleNormalKey(msg)

	case tea.MouseClickMsg:
		if msg.Button == tea.MouseLeft {
			if t.Zones != nil {
				if z := t.Zones.Get(t.ZoneID()); z != nil && z.InBounds(msg) {
					return t.startEditing()
				}
			}
		}
	}

	return t, nil
}

// RouteEvent routes events to this component.
func (t *TextComponent) RouteEvent(msg tea.Msg) (bool, tea.Cmd) {
	if t.ReadOnly {
		return false, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if t.Editing {
			_, cmd := t.handleEditingKey(msg)
			return true, cmd // Editing captures all keys
		}
		switch msg.String() {
		case "enter", " ":
			_, cmd := t.startEditing()
			return true, cmd
		}

	case tea.MouseClickMsg:
		if msg.Button == tea.MouseLeft {
			if t.Zones != nil {
				if z := t.Zones.Get(t.ZoneID()); z != nil && z.InBounds(msg) {
					_, cmd := t.startEditing()
					return true, cmd
				}
			}
		}
	}

	return false, nil
}

func (t *TextComponent) handleNormalKey(msg tea.KeyMsg) (component.Component, tea.Cmd) {
	switch msg.String() {
	case "enter", " ":
		return t.startEditing()
	}
	return t, nil
}

func (t *TextComponent) handleEditingKey(msg tea.KeyMsg) (component.Component, tea.Cmd) {
	switch msg.String() {
	case "esc":
		// Cancel and restore original value
		t.Input.SetValue(t.OriginalValue)
		t.Value = t.OriginalValue
		t.Editing = false
		t.Input.Blur()
		return t, nil

	case "enter":
		// Confirm edit
		t.Value = t.Input.Value()
		t.Editing = false
		t.Input.Blur()
		return t, func() tea.Msg {
			return FieldChangedMsg{
				FieldID: t.ID,
				Value:   TextValue{Text: t.Value},
			}
		}
	}

	// Pass other keys to textinput
	var cmd tea.Cmd
	t.Input, cmd = t.Input.Update(msg)
	return t, cmd
}

func (t *TextComponent) startEditing() (component.Component, tea.Cmd) {
	t.Editing = true
	t.OriginalValue = t.Value
	t.Input.SetValue(t.Value)
	t.Input.Focus()
	return t, nil
}

// View renders the text field.
func (t *TextComponent) View() string {
	labelStr := t.Label
	if t.MaxLabelWidth > 0 {
		labelStr = fmt.Sprintf("%-*s", t.MaxLabelWidth, t.Label)
	}

	var valueStr string
	if t.Editing {
		// Show the textinput with cursor
		valueStr = t.Input.View()
	} else {
		valueStr = t.Value
	}

	line := fmt.Sprintf("  %s  %s", labelStr, valueStr)

	if t.Zones != nil {
		return t.Zones.Mark(t.ZoneID(), line)
	}

	return line
}
