package layout

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/kluzzebass/lazyhue/internal/ui2/component"
)

// Spacer takes up space without rendering anything visible.
// Use it to create consistent indentation or alignment.
type Spacer struct {
	*component.BaseComponent
	Width int
}

// NewSpacer creates a new spacer with the given width.
func NewSpacer(width int) *Spacer {
	return &Spacer{
		BaseComponent: component.NewBaseComponent(),
		Width:         width,
	}
}

// SetWidth sets the spacer width.
func (s *Spacer) SetWidth(width int) {
	s.Width = width
}

// Update is a no-op for spacers.
func (s *Spacer) Update(msg tea.Msg) (component.Component, tea.Cmd) {
	return s, nil
}

// RouteEvent returns false as spacers don't handle events.
func (s *Spacer) RouteEvent(msg tea.Msg) (bool, tea.Cmd) {
	return false, nil
}

// View renders empty space.
func (s *Spacer) View() string {
	if s.Width <= 0 {
		return ""
	}
	return strings.Repeat(" ", s.Width)
}

// CanFocus returns false - spacers cannot be focused.
func (s *Spacer) CanFocus() bool {
	return false
}
