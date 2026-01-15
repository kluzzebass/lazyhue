package field

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/kluzzebass/lazyhue/internal/ui"
	"github.com/kluzzebass/lazyhue/internal/ui/component"
	zone "github.com/lrstanley/bubblezone/v2"
)

// CheckboxValue represents the value of a checkbox group.
type CheckboxValue struct {
	Selected map[int]bool // Map of selected option values
}

// CheckboxComponent is a checkbox group field (multiple selections allowed).
type CheckboxComponent struct {
	*BaseField

	Selected map[int]bool // Currently selected option values
	Options  []Option     // Available options
	Vertical bool         // If true, render options vertically

	// Edit state
	EditCursor int              // Current cursor position when editing
	Original   map[int]bool     // Values before editing (for cancel)
}

// NewCheckboxComponent creates a new checkbox component.
func NewCheckboxComponent(id, label string, selected map[int]bool, options []Option, vertical bool, styles *ui.Styles, zones *zone.Manager) *CheckboxComponent {
	if selected == nil {
		selected = make(map[int]bool)
	}
	return &CheckboxComponent{
		BaseField:  NewBaseField(id, label, styles, zones),
		Selected:   selected,
		Options:    options,
		Vertical:   vertical,
		EditCursor: 0,
	}
}

// SetSelected sets the selected options.
func (c *CheckboxComponent) SetSelected(selected map[int]bool) {
	if selected == nil {
		selected = make(map[int]bool)
	}
	c.Selected = selected
}

// SetOptions sets the available options.
func (c *CheckboxComponent) SetOptions(options []Option) {
	c.Options = options
}

// IsSelected returns true if the option value is selected.
func (c *CheckboxComponent) IsSelected(value int) bool {
	return c.Selected[value]
}

// Update handles events for the checkbox field.
func (c *CheckboxComponent) Update(msg tea.Msg) (component.Component, tea.Cmd) {
	if c.ReadOnly {
		return c, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if c.Editing {
			return c.handleEditingKey(msg)
		}
		return c.handleNormalKey(msg)

	case tea.MouseClickMsg:
		return c.handleMouseClick(msg)
	}

	return c, nil
}

// RouteEvent routes events to this component.
func (c *CheckboxComponent) RouteEvent(msg tea.Msg) (bool, tea.Cmd) {
	if c.ReadOnly {
		return false, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if c.Editing {
			_, cmd := c.handleEditingKey(msg)
			return true, cmd // Editing captures all keys
		}
		switch msg.String() {
		case "enter", " ":
			_, cmd := c.startEditing()
			return true, cmd
		}

	case tea.MouseClickMsg:
		if msg.Button == tea.MouseLeft {
			// Check option zones
			if c.Zones != nil {
				for i, opt := range c.Options {
					optionZoneID := fmt.Sprintf("%s-option-%d", c.ZoneID(), i)
					if z := c.Zones.Get(optionZoneID); z != nil && z.InBounds(msg) {
						_, cmd := c.toggleOption(opt.Value)
						return true, cmd
					}
				}
			}
		}
	}

	return false, nil
}

func (c *CheckboxComponent) handleNormalKey(msg tea.KeyMsg) (component.Component, tea.Cmd) {
	switch msg.String() {
	case "enter", " ":
		return c.startEditing()
	}
	return c, nil
}

func (c *CheckboxComponent) handleEditingKey(msg tea.KeyMsg) (component.Component, tea.Cmd) {
	switch msg.String() {
	case "esc":
		// Cancel and restore original values
		c.Selected = c.Original
		c.Editing = false
		return c, nil

	case "enter":
		// Confirm selection
		c.Editing = false
		return c, func() tea.Msg {
			return FieldChangedMsg{
				FieldID: c.ID,
				Value:   CheckboxValue{Selected: c.copySelected()},
			}
		}

	case " ", "x":
		// Toggle current option
		if c.EditCursor >= 0 && c.EditCursor < len(c.Options) {
			value := c.Options[c.EditCursor].Value
			c.Selected[value] = !c.Selected[value]
		}

	case "up", "k":
		if c.Vertical && c.EditCursor > 0 {
			c.EditCursor--
		}

	case "down", "j":
		if c.Vertical && c.EditCursor < len(c.Options)-1 {
			c.EditCursor++
		}

	case "left", "h":
		if !c.Vertical && c.EditCursor > 0 {
			c.EditCursor--
		}

	case "right", "l":
		if !c.Vertical && c.EditCursor < len(c.Options)-1 {
			c.EditCursor++
		}
	}

	return c, nil
}

func (c *CheckboxComponent) handleMouseClick(msg tea.MouseClickMsg) (component.Component, tea.Cmd) {
	if c.Zones == nil {
		return c, nil
	}

	// Check which option was clicked
	for i, opt := range c.Options {
		optionZoneID := fmt.Sprintf("%s-option-%d", c.ZoneID(), i)
		if z := c.Zones.Get(optionZoneID); z != nil && z.InBounds(msg) {
			return c.toggleOption(opt.Value)
		}
	}

	return c, nil
}

func (c *CheckboxComponent) startEditing() (component.Component, tea.Cmd) {
	c.Editing = true
	c.Original = c.copySelected()
	c.EditCursor = 0
	return c, nil
}

func (c *CheckboxComponent) toggleOption(value int) (component.Component, tea.Cmd) {
	c.Selected[value] = !c.Selected[value]

	return c, func() tea.Msg {
		return FieldChangedMsg{
			FieldID: c.ID,
			Value:   CheckboxValue{Selected: c.copySelected()},
		}
	}
}

func (c *CheckboxComponent) copySelected() map[int]bool {
	copy := make(map[int]bool)
	for k, v := range c.Selected {
		copy[k] = v
	}
	return copy
}

// FieldHeight returns the number of rows this component takes up.
func (c *CheckboxComponent) FieldHeight() int {
	if c.Vertical {
		return len(c.Options)
	}
	return 1
}

// ViewControl renders only the control portion (no label).
func (c *CheckboxComponent) ViewControl() string {
	if c.Vertical {
		return c.renderVerticalControl()
	}
	return c.renderHorizontalControl()
}

func (c *CheckboxComponent) renderVerticalControl() string {
	var out strings.Builder

	for i, opt := range c.Options {
		if i > 0 {
			out.WriteString("\n")
		}

		indicator := "[ ]"
		if c.Selected[opt.Value] {
			indicator = "[x]"
		}

		line := fmt.Sprintf("%s %s", indicator, opt.Label)

		// Mark each option with a zone
		if c.Zones != nil {
			optionZoneID := fmt.Sprintf("%s-option-%d", c.ZoneID(), i)
			line = c.Zones.Mark(optionZoneID, line)
		}

		out.WriteString(line)
	}

	return out.String()
}

func (c *CheckboxComponent) renderHorizontalControl() string {
	var parts []string
	for i, opt := range c.Options {
		indicator := "[ ]"
		if c.Selected[opt.Value] {
			indicator = "[x]"
		}
		optStr := fmt.Sprintf("%s %s", indicator, opt.Label)

		// Mark each option with a zone
		if c.Zones != nil {
			optionZoneID := fmt.Sprintf("%s-option-%d", c.ZoneID(), i)
			optStr = c.Zones.Mark(optionZoneID, optStr)
		}

		parts = append(parts, optStr)
	}

	control := strings.Join(parts, "  ")

	// Also mark the whole control
	if c.Zones != nil {
		return c.Zones.Mark(c.ZoneID(), control)
	}

	return control
}

// View renders the checkbox field.
func (c *CheckboxComponent) View() string {
	if c.Vertical {
		return c.renderVertical()
	}
	return c.renderHorizontal()
}

func (c *CheckboxComponent) renderVertical() string {
	var out strings.Builder

	// First row has the label
	labelStr := c.Label
	if c.MaxLabelWidth > 0 {
		labelStr = fmt.Sprintf("%-*s", c.MaxLabelWidth, c.Label)
	}

	for i, opt := range c.Options {
		indicator := "[ ]"
		if c.Selected[opt.Value] {
			indicator = "[x]"
		}

		var line string
		if i == 0 {
			line = fmt.Sprintf("  %s  %s %s", labelStr, indicator, opt.Label)
		} else {
			// Indent subsequent rows to align with first option
			indent := strings.Repeat(" ", 2+c.MaxLabelWidth+2)
			line = fmt.Sprintf("%s%s %s", indent, indicator, opt.Label)
		}

		// Mark each option with a zone
		if c.Zones != nil {
			optionZoneID := fmt.Sprintf("%s-option-%d", c.ZoneID(), i)
			line = c.Zones.Mark(optionZoneID, line)
		}

		out.WriteString(line)
		if i < len(c.Options)-1 {
			out.WriteString("\n")
		}
	}

	return out.String()
}

func (c *CheckboxComponent) renderHorizontal() string {
	labelStr := c.Label
	if c.MaxLabelWidth > 0 {
		labelStr = fmt.Sprintf("%-*s", c.MaxLabelWidth, c.Label)
	}

	var parts []string
	for i, opt := range c.Options {
		indicator := "[ ]"
		if c.Selected[opt.Value] {
			indicator = "[x]"
		}
		optStr := fmt.Sprintf("%s %s", indicator, opt.Label)

		// Mark each option with a zone
		if c.Zones != nil {
			optionZoneID := fmt.Sprintf("%s-option-%d", c.ZoneID(), i)
			optStr = c.Zones.Mark(optionZoneID, optStr)
		}

		parts = append(parts, optStr)
	}

	line := fmt.Sprintf("  %s  %s", labelStr, strings.Join(parts, "  "))

	// Also mark the whole field
	if c.Zones != nil {
		return c.Zones.Mark(c.ZoneID(), line)
	}

	return line
}
