package layout

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/charmbracelet/lipgloss/v2"
	"github.com/kluzzebass/lazyhue/internal/ui2/component"
)

// Label is a simple text display component.
// It cannot be focused and just renders aligned text.
type Label struct {
	*component.BaseComponent
	Text  string
	Width int // Padded width (0 = natural width)
	Style lipgloss.Style
}

// NewLabel creates a new label component.
func NewLabel(text string) *Label {
	return &Label{
		BaseComponent: component.NewBaseComponent(),
		Text:          text,
	}
}

// NewLabelWithWidth creates a label with a fixed width.
func NewLabelWithWidth(text string, width int) *Label {
	return &Label{
		BaseComponent: component.NewBaseComponent(),
		Text:          text,
		Width:         width,
	}
}

// SetWidth sets the label width.
func (l *Label) SetWidth(width int) {
	l.Width = width
}

// SetStyle sets the label style.
func (l *Label) SetStyle(style lipgloss.Style) {
	l.Style = style
}

// Update is a no-op for labels.
func (l *Label) Update(msg tea.Msg) (component.Component, tea.Cmd) {
	return l, nil
}

// RouteEvent returns false as labels don't handle events.
func (l *Label) RouteEvent(msg tea.Msg) (bool, tea.Cmd) {
	return false, nil
}

// View renders the label text with optional padding and styling.
func (l *Label) View() string {
	text := l.Text
	if l.Width > 0 {
		text = fmt.Sprintf("%-*s", l.Width, l.Text)
	}
	if l.Style.Value() != "" {
		return l.Style.Render(text)
	}
	return text
}

// CanFocus returns false - labels cannot be focused.
func (l *Label) CanFocus() bool {
	return false
}
