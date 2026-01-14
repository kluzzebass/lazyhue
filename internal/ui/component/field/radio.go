package field

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/kluzzebass/lazyhue/internal/ui"
	"github.com/kluzzebass/lazyhue/internal/ui/component"
	zone "github.com/lrstanley/bubblezone/v2"
)

// RadioComponent is a radio button group field.
type RadioComponent struct {
	*BaseField

	Value    int      // Currently selected option value
	Options  []Option // Available options
	Vertical bool     // If true, render options vertically

	// Edit state
	EditCursor    int // Current selection when editing
	OriginalValue int // Value before editing (for cancel)
}

// NewRadioComponent creates a new radio component.
func NewRadioComponent(id, label string, value int, options []Option, vertical bool, styles *ui.Styles, zones *zone.Manager) *RadioComponent {
	// Find the index of the current value
	cursor := 0
	for i, opt := range options {
		if opt.Value == value {
			cursor = i
			break
		}
	}

	return &RadioComponent{
		BaseField:  NewBaseField(id, label, styles, zones),
		Value:      value,
		Options:    options,
		Vertical:   vertical,
		EditCursor: cursor,
	}
}

// SetValue sets the selected option.
func (r *RadioComponent) SetValue(value int) {
	r.Value = value
	for i, opt := range r.Options {
		if opt.Value == value {
			r.EditCursor = i
			break
		}
	}
}

// SetOptions sets the available options.
func (r *RadioComponent) SetOptions(options []Option) {
	r.Options = options
}

// Update handles events for the radio field.
func (r *RadioComponent) Update(msg tea.Msg) (component.Component, tea.Cmd) {
	if r.ReadOnly {
		return r, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if r.Editing {
			return r.handleEditingKey(msg)
		}
		return r.handleNormalKey(msg)

	case tea.MouseClickMsg:
		return r.handleMouseClick(msg)
	}

	return r, nil
}

// RouteEvent routes events to this component.
func (r *RadioComponent) RouteEvent(msg tea.Msg) (bool, tea.Cmd) {
	if r.ReadOnly {
		return false, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if r.Editing {
			_, cmd := r.handleEditingKey(msg)
			return true, cmd // Editing captures all keys
		}
		switch msg.String() {
		case "enter", " ":
			_, cmd := r.startEditing()
			return true, cmd
		}

	case tea.MouseClickMsg:
		if msg.Button == tea.MouseLeft {
			// Check option zones
			if r.Zones != nil {
				for i := range r.Options {
					optionZoneID := fmt.Sprintf("%s-option-%d", r.ZoneID(), i)
					if z := r.Zones.Get(optionZoneID); z != nil && z.InBounds(msg) {
						_, cmd := r.selectOption(i)
						return true, cmd
					}
				}
			}
		}
	}

	return false, nil
}

func (r *RadioComponent) handleNormalKey(msg tea.KeyMsg) (component.Component, tea.Cmd) {
	switch msg.String() {
	case "enter", " ":
		return r.startEditing()
	}
	return r, nil
}

func (r *RadioComponent) handleEditingKey(msg tea.KeyMsg) (component.Component, tea.Cmd) {
	switch msg.String() {
	case "esc":
		// Cancel and restore original value
		r.Value = r.OriginalValue
		r.Editing = false
		return r, nil

	case "enter":
		// Confirm selection
		r.Editing = false
		return r, func() tea.Msg {
			return FieldChangedMsg{
				FieldID: r.ID,
				Value:   SelectValue{Index: r.Value},
			}
		}

	case "up", "k":
		if r.Vertical && r.EditCursor > 0 {
			r.EditCursor--
			r.Value = r.Options[r.EditCursor].Value
		}

	case "down", "j":
		if r.Vertical && r.EditCursor < len(r.Options)-1 {
			r.EditCursor++
			r.Value = r.Options[r.EditCursor].Value
		}

	case "left", "h":
		if !r.Vertical && r.EditCursor > 0 {
			r.EditCursor--
			r.Value = r.Options[r.EditCursor].Value
		}

	case "right", "l":
		if !r.Vertical && r.EditCursor < len(r.Options)-1 {
			r.EditCursor++
			r.Value = r.Options[r.EditCursor].Value
		}
	}

	return r, nil
}

func (r *RadioComponent) handleMouseClick(msg tea.MouseClickMsg) (component.Component, tea.Cmd) {
	if r.Zones == nil {
		return r, nil
	}

	// Check which option was clicked
	for i := range r.Options {
		optionZoneID := fmt.Sprintf("%s-option-%d", r.ZoneID(), i)
		if z := r.Zones.Get(optionZoneID); z != nil && z.InBounds(msg) {
			return r.selectOption(i)
		}
	}

	return r, nil
}

func (r *RadioComponent) startEditing() (component.Component, tea.Cmd) {
	r.Editing = true
	r.OriginalValue = r.Value
	// Find cursor position for current value
	for i, opt := range r.Options {
		if opt.Value == r.Value {
			r.EditCursor = i
			break
		}
	}
	return r, nil
}

func (r *RadioComponent) selectOption(index int) (component.Component, tea.Cmd) {
	if index < 0 || index >= len(r.Options) {
		return r, nil
	}

	r.Value = r.Options[index].Value
	r.EditCursor = index

	return r, func() tea.Msg {
		return FieldChangedMsg{
			FieldID: r.ID,
			Value:   SelectValue{Index: r.Value},
		}
	}
}

// FieldHeight returns the number of rows this component takes up.
func (r *RadioComponent) FieldHeight() int {
	if r.Vertical {
		return len(r.Options)
	}
	return 1
}

// ViewControl renders only the control portion (no label).
func (r *RadioComponent) ViewControl() string {
	if r.Vertical {
		return r.renderVerticalControl()
	}
	return r.renderHorizontalControl()
}

func (r *RadioComponent) renderVerticalControl() string {
	var out strings.Builder

	for i, opt := range r.Options {
		if i > 0 {
			out.WriteString("\n")
		}

		indicator := "○"
		if opt.Value == r.Value {
			indicator = "●"
		}

		line := fmt.Sprintf("%s %s", indicator, opt.Label)

		// Mark each option with a zone
		if r.Zones != nil {
			optionZoneID := fmt.Sprintf("%s-option-%d", r.ZoneID(), i)
			line = r.Zones.Mark(optionZoneID, line)
		}

		out.WriteString(line)
	}

	return out.String()
}

func (r *RadioComponent) renderHorizontalControl() string {
	var parts []string
	for i, opt := range r.Options {
		indicator := "○"
		if opt.Value == r.Value {
			indicator = "●"
		}
		optStr := fmt.Sprintf("%s %s", indicator, opt.Label)

		// Mark each option with a zone
		if r.Zones != nil {
			optionZoneID := fmt.Sprintf("%s-option-%d", r.ZoneID(), i)
			optStr = r.Zones.Mark(optionZoneID, optStr)
		}

		parts = append(parts, optStr)
	}

	control := strings.Join(parts, "  ")

	// Also mark the whole control
	if r.Zones != nil {
		return r.Zones.Mark(r.ZoneID(), control)
	}

	return control
}

// View renders the radio field.
func (r *RadioComponent) View() string {
	if r.Vertical {
		return r.renderVertical()
	}
	return r.renderHorizontal()
}

func (r *RadioComponent) renderVertical() string {
	var out strings.Builder

	// First row has the label
	labelStr := r.Label
	if r.MaxLabelWidth > 0 {
		labelStr = fmt.Sprintf("%-*s", r.MaxLabelWidth, r.Label)
	}

	for i, opt := range r.Options {
		indicator := "○"
		if opt.Value == r.Value {
			indicator = "●"
		}

		var line string
		if i == 0 {
			line = fmt.Sprintf("  %s  %s %s", labelStr, indicator, opt.Label)
		} else {
			// Indent subsequent rows to align with first option
			indent := strings.Repeat(" ", 2+r.MaxLabelWidth+2)
			line = fmt.Sprintf("%s%s %s", indent, indicator, opt.Label)
		}

		// Mark each option with a zone
		if r.Zones != nil {
			optionZoneID := fmt.Sprintf("%s-option-%d", r.ZoneID(), i)
			line = r.Zones.Mark(optionZoneID, line)
		}

		out.WriteString(line)
		if i < len(r.Options)-1 {
			out.WriteString("\n")
		}
	}

	return out.String()
}

func (r *RadioComponent) renderHorizontal() string {
	labelStr := r.Label
	if r.MaxLabelWidth > 0 {
		labelStr = fmt.Sprintf("%-*s", r.MaxLabelWidth, r.Label)
	}

	var parts []string
	for i, opt := range r.Options {
		indicator := "○"
		if opt.Value == r.Value {
			indicator = "●"
		}
		optStr := fmt.Sprintf("%s %s", indicator, opt.Label)

		// Mark each option with a zone
		if r.Zones != nil {
			optionZoneID := fmt.Sprintf("%s-option-%d", r.ZoneID(), i)
			optStr = r.Zones.Mark(optionZoneID, optStr)
		}

		parts = append(parts, optStr)
	}

	line := fmt.Sprintf("  %s  %s", labelStr, strings.Join(parts, "  "))

	// Also mark the whole field
	if r.Zones != nil {
		return r.Zones.Mark(r.ZoneID(), line)
	}

	return line
}
