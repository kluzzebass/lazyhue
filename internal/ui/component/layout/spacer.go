package layout

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/kluzzebass/lazyhue/internal/ui/component"
)

// Spacer takes up space without rendering anything visible.
// Use it to create consistent indentation or alignment.
type Spacer struct {
	*component.BaseComponent
	Width   int
	Height  int
	Padding Spacing // Inner spacing
	Margin  Spacing // Outer spacing
}

// NewSpacer creates a new spacer with the given width.
func NewSpacer(width int) *Spacer {
	return &Spacer{
		BaseComponent: component.NewBaseComponent(),
		Width:         width,
		Height:        1,
	}
}

// NewSpacerWH creates a new spacer with width and height.
func NewSpacerWH(width, height int) *Spacer {
	return &Spacer{
		BaseComponent: component.NewBaseComponent(),
		Width:         width,
		Height:        height,
	}
}

// SetWidth sets the spacer width.
func (s *Spacer) SetWidth(width int) *Spacer {
	s.Width = width
	return s
}

// SetHeight sets the spacer height.
func (s *Spacer) SetHeight(height int) *Spacer {
	s.Height = height
	return s
}

// SetPadding sets the inner spacing.
func (s *Spacer) SetPadding(p Spacing) *Spacer {
	s.Padding = p
	return s
}

// SetMargin sets the outer spacing.
func (s *Spacer) SetMargin(m Spacing) *Spacer {
	s.Margin = m
	return s
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
	width := s.Width
	height := s.Height
	if width <= 0 && height <= 0 {
		return ""
	}
	if width <= 0 {
		width = 1
	}
	if height <= 0 {
		height = 1
	}

	line := strings.Repeat(" ", width)
	lines := make([]string, height)
	for i := range lines {
		lines[i] = line
	}
	content := strings.Join(lines, "\n")

	// Apply padding (inner spacing)
	content = ApplySpacing(content, s.Padding)

	// Apply margin (outer spacing)
	return ApplySpacing(content, s.Margin)
}

// CanFocus returns false - spacers cannot be focused.
func (s *Spacer) CanFocus() bool {
	return false
}
