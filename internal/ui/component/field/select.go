package field

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/kluzzebass/lazyhue/internal/ui"
	"github.com/kluzzebass/lazyhue/internal/ui/component"
	zone "github.com/lrstanley/bubblezone/v2"
)

// SelectComponent is a dropdown select field.
type SelectComponent struct {
	*BaseField

	Value   int      // Currently selected option index
	Options []Option // Available options

	// Dropdown state
	Open   bool
	Cursor int // Current highlight position in dropdown
	Scroll int // Scroll offset for long option lists
}

const maxVisibleOptions = 5

// NewSelectComponent creates a new select component.
func NewSelectComponent(id, label string, value int, options []Option, styles *ui.Styles, zones *zone.Manager) *SelectComponent {
	return &SelectComponent{
		BaseField: NewBaseField(id, label, styles, zones),
		Value:     value,
		Options:   options,
		Cursor:    value, // Start with current value highlighted
	}
}

// SetValue sets the selected option.
func (s *SelectComponent) SetValue(value int) {
	s.Value = value
	s.Cursor = value
}

// SetOptions sets the available options.
func (s *SelectComponent) SetOptions(options []Option) {
	s.Options = options
}

// IsOpen returns whether the dropdown is open.
func (s *SelectComponent) IsOpen() bool {
	return s.Open
}

// Update handles events for the select field.
func (s *SelectComponent) Update(msg tea.Msg) (component.Component, tea.Cmd) {
	if s.ReadOnly {
		return s, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if s.Open {
			return s.handleDropdownKey(msg)
		}
		return s.handleClosedKey(msg)

	case tea.MouseClickMsg:
		return s.handleMouseClick(msg)

	case tea.MouseWheelMsg:
		if s.Open {
			return s.handleMouseWheel(msg)
		}
	}

	return s, nil
}

// RouteEvent routes events to this component.
func (s *SelectComponent) RouteEvent(msg tea.Msg) (bool, tea.Cmd) {
	if s.ReadOnly {
		return false, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if s.Open {
			_, cmd := s.handleDropdownKey(msg)
			return true, cmd // Dropdown captures all keys when open
		}
		switch msg.String() {
		case "enter", " ":
			_, cmd := s.openDropdown()
			return true, cmd
		}

	case tea.MouseClickMsg:
		if msg.Button == tea.MouseLeft {
			// Check main zone
			if s.Zones != nil {
				if z := s.Zones.Get(s.ZoneID()); z != nil && z.InBounds(msg) {
					_, cmd := s.handleMouseClick(msg)
					return true, cmd
				}
			}
			// Check dropdown option zones
			if s.Open {
				for i := range s.Options {
					optionZoneID := fmt.Sprintf("%s-option-%d", s.ZoneID(), i)
					if z := s.Zones.Get(optionZoneID); z != nil && z.InBounds(msg) {
						_, cmd := s.selectOption(i)
						return true, cmd
					}
				}
			}
		}

	case tea.MouseWheelMsg:
		if s.Open {
			_, cmd := s.handleMouseWheel(msg)
			return true, cmd
		}
	}

	return false, nil
}

func (s *SelectComponent) handleClosedKey(msg tea.KeyMsg) (component.Component, tea.Cmd) {
	switch msg.String() {
	case "enter", " ":
		return s.openDropdown()
	}
	return s, nil
}

func (s *SelectComponent) handleDropdownKey(msg tea.KeyMsg) (component.Component, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if s.Cursor > 0 {
			s.Cursor--
			s.ensureCursorVisible()
		}
	case "down", "j":
		if s.Cursor < len(s.Options)-1 {
			s.Cursor++
			s.ensureCursorVisible()
		}
	case "enter", " ":
		return s.selectOption(s.Cursor)
	case "esc":
		s.Open = false
		s.Cursor = s.Value // Reset to current value
		s.Scroll = 0
	}
	return s, nil
}

func (s *SelectComponent) handleMouseClick(msg tea.MouseClickMsg) (component.Component, tea.Cmd) {
	if !s.Open {
		return s.openDropdown()
	}

	// Check if click is on an option
	if s.Zones != nil {
		for i := range s.Options {
			optionZoneID := fmt.Sprintf("%s-option-%d", s.ZoneID(), i)
			if z := s.Zones.Get(optionZoneID); z != nil && z.InBounds(msg) {
				return s.selectOption(i)
			}
		}
	}

	// Click outside options closes dropdown
	s.Open = false
	s.Cursor = s.Value
	s.Scroll = 0
	return s, nil
}

func (s *SelectComponent) handleMouseWheel(msg tea.MouseWheelMsg) (component.Component, tea.Cmd) {
	switch msg.Button {
	case tea.MouseWheelUp:
		if s.Cursor > 0 {
			s.Cursor--
			s.ensureCursorVisible()
		}
	case tea.MouseWheelDown:
		if s.Cursor < len(s.Options)-1 {
			s.Cursor++
			s.ensureCursorVisible()
		}
	}
	return s, nil
}

func (s *SelectComponent) openDropdown() (component.Component, tea.Cmd) {
	s.Open = true
	s.Cursor = s.Value
	s.ensureCursorVisible()
	return s, nil
}

func (s *SelectComponent) selectOption(index int) (component.Component, tea.Cmd) {
	if index < 0 || index >= len(s.Options) {
		return s, nil
	}

	s.Value = s.Options[index].Value
	s.Open = false
	s.Cursor = index
	s.Scroll = 0

	return s, func() tea.Msg {
		return FieldChangedMsg{
			FieldID: s.ID,
			Value:   SelectValue{Index: s.Value},
		}
	}
}

func (s *SelectComponent) ensureCursorVisible() {
	if s.Cursor < s.Scroll {
		s.Scroll = s.Cursor
	} else if s.Cursor >= s.Scroll+maxVisibleOptions {
		s.Scroll = s.Cursor - maxVisibleOptions + 1
	}
}

// FieldHeight returns the number of rows this component takes up.
func (s *SelectComponent) FieldHeight() int {
	if s.Open {
		// Header line + top border + visible options + bottom border
		visibleOpts := len(s.Options)
		if visibleOpts > maxVisibleOptions {
			visibleOpts = maxVisibleOptions
		}
		return 1 + 1 + visibleOpts + 1 // dropdown header + border + options + border
	}
	return 1
}

// ViewControl renders only the control portion (no label).
func (s *SelectComponent) ViewControl() string {
	if s.Open {
		return s.renderDropdownControl()
	}
	return s.renderClosedControl()
}

func (s *SelectComponent) renderClosedControl() string {
	// Get current option label
	currentLabel := "---"
	for _, opt := range s.Options {
		if opt.Value == s.Value {
			currentLabel = opt.Label
			break
		}
	}

	control := fmt.Sprintf("[%s] ▼", currentLabel)

	if s.Zones != nil {
		return s.Zones.Mark(s.ZoneID(), control)
	}

	return control
}

func (s *SelectComponent) renderDropdownControl() string {
	var out strings.Builder

	// First line shows dropdown indicator
	out.WriteString("[▼]\n")

	// Calculate visible range
	start := s.Scroll
	end := start + maxVisibleOptions
	if end > len(s.Options) {
		end = len(s.Options)
	}

	// Find max option label width
	maxLen := 0
	for _, opt := range s.Options {
		if len(opt.Label) > maxLen {
			maxLen = len(opt.Label)
		}
	}

	// Render dropdown box with scroll indicators
	hasMore := end < len(s.Options)
	hasLess := start > 0
	boxWidth := maxLen + 2

	// Top border with optional scroll-up indicator (on right side)
	if hasLess {
		out.WriteString("┌" + strings.Repeat("─", boxWidth-1) + "▲┐\n")
	} else {
		out.WriteString("┌" + strings.Repeat("─", boxWidth) + "┐\n")
	}

	for i := start; i < end; i++ {
		opt := s.Options[i]
		prefix := "  "
		if i == s.Cursor {
			prefix = "> "
		}
		label := opt.Label + strings.Repeat(" ", maxLen-len(opt.Label))
		optionLine := fmt.Sprintf("│%s%s│", prefix, label)

		// Mark each option with a zone
		if s.Zones != nil {
			optionZoneID := fmt.Sprintf("%s-option-%d", s.ZoneID(), i)
			optionLine = s.Zones.Mark(optionZoneID, optionLine)
		}

		out.WriteString(optionLine + "\n")
	}

	// Bottom border with optional scroll-down indicator (on right side)
	if hasMore {
		out.WriteString("└" + strings.Repeat("─", boxWidth-1) + "▼┘")
	} else {
		out.WriteString("└" + strings.Repeat("─", boxWidth) + "┘")
	}

	// Wrap entire dropdown area in main zone
	result := out.String()
	if s.Zones != nil {
		result = s.Zones.Mark(s.ZoneID(), result)
	}

	return result
}

// View renders the select field.
func (s *SelectComponent) View() string {
	// Build the label
	labelStr := s.Label
	if s.MaxLabelWidth > 0 {
		labelStr = fmt.Sprintf("%-*s", s.MaxLabelWidth, s.Label)
	}

	if s.Open {
		return s.renderDropdown(labelStr)
	}

	return s.renderClosed(labelStr)
}

func (s *SelectComponent) renderClosed(labelStr string) string {
	// Get current option label
	currentLabel := "---"
	for _, opt := range s.Options {
		if opt.Value == s.Value {
			currentLabel = opt.Label
			break
		}
	}

	line := fmt.Sprintf("  %s  [%s] ▼", labelStr, currentLabel)

	if s.Zones != nil {
		return s.Zones.Mark(s.ZoneID(), line)
	}

	return line
}

func (s *SelectComponent) renderDropdown(labelStr string) string {
	var out strings.Builder

	// First line shows label and dropdown indicator
	out.WriteString(fmt.Sprintf("  %s  [▼]\n", labelStr))

	// Calculate visible range
	start := s.Scroll
	end := start + maxVisibleOptions
	if end > len(s.Options) {
		end = len(s.Options)
	}

	// Find max option label width
	maxLen := 0
	for _, opt := range s.Options {
		if len(opt.Label) > maxLen {
			maxLen = len(opt.Label)
		}
	}

	// Render dropdown box with scroll indicators
	hasMore := end < len(s.Options)
	hasLess := start > 0
	boxWidth := maxLen + 2
	boxIndent := "    "

	// Top border with optional scroll-up indicator (on right side)
	if hasLess {
		out.WriteString(boxIndent + "┌" + strings.Repeat("─", boxWidth-1) + "▲┐\n")
	} else {
		out.WriteString(boxIndent + "┌" + strings.Repeat("─", boxWidth) + "┐\n")
	}

	for i := start; i < end; i++ {
		opt := s.Options[i]
		prefix := "  "
		if i == s.Cursor {
			prefix = "> "
		}
		label := opt.Label + strings.Repeat(" ", maxLen-len(opt.Label))
		optionLine := fmt.Sprintf("%s│%s%s│", boxIndent, prefix, label)

		// Mark each option with a zone
		if s.Zones != nil {
			optionZoneID := fmt.Sprintf("%s-option-%d", s.ZoneID(), i)
			optionLine = s.Zones.Mark(optionZoneID, optionLine)
		}

		out.WriteString(optionLine + "\n")
	}

	// Bottom border with optional scroll-down indicator (on right side)
	if hasMore {
		out.WriteString(boxIndent + "└" + strings.Repeat("─", boxWidth-1) + "▼┘")
	} else {
		out.WriteString(boxIndent + "└" + strings.Repeat("─", boxWidth) + "┘")
	}

	// Wrap entire dropdown area in main zone
	result := out.String()
	if s.Zones != nil {
		result = s.Zones.Mark(s.ZoneID(), result)
	}

	return result
}
