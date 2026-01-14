package layout

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/charmbracelet/lipgloss/v2"
	"github.com/kluzzebass/lazyhue/internal/ui/component"
)

// Label is a simple text display component.
// It cannot be focused and just renders aligned text.
type Label struct {
	*component.BaseComponent
	Text    string
	Width   int // Padded width (0 = natural width)
	Height  int // Padded height (0 = natural height)
	Style   lipgloss.Style
	Padding Spacing // Inner spacing
	Margin  Spacing // Outer spacing
	HAlign  HAlign
	VAlign  VAlign
}

// NewLabel creates a new label component.
func NewLabel(text string) *Label {
	return &Label{
		BaseComponent: component.NewBaseComponent(),
		Text:          text,
		HAlign:        HAlignLeft,
		VAlign:        VAlignTop,
	}
}

// NewLabelWithWidth creates a label with a fixed width.
func NewLabelWithWidth(text string, width int) *Label {
	return &Label{
		BaseComponent: component.NewBaseComponent(),
		Text:          text,
		Width:         width,
		HAlign:        HAlignLeft,
		VAlign:        VAlignTop,
	}
}

// SetWidth sets the label width.
func (l *Label) SetWidth(width int) *Label {
	l.Width = width
	return l
}

// SetHeight sets the label height.
func (l *Label) SetHeight(height int) *Label {
	l.Height = height
	return l
}

// SetStyle sets the label style.
func (l *Label) SetStyle(style lipgloss.Style) *Label {
	l.Style = style
	return l
}

// SetPadding sets the inner spacing.
func (l *Label) SetPadding(p Spacing) *Label {
	l.Padding = p
	return l
}

// SetMargin sets the outer spacing.
func (l *Label) SetMargin(m Spacing) *Label {
	l.Margin = m
	return l
}

// SetHAlign sets horizontal alignment.
func (l *Label) SetHAlign(align HAlign) *Label {
	l.HAlign = align
	return l
}

// SetVAlign sets vertical alignment.
func (l *Label) SetVAlign(align VAlign) *Label {
	l.VAlign = align
	return l
}

// Update is a no-op for labels.
func (l *Label) Update(msg tea.Msg) (component.Component, tea.Cmd) {
	return l, nil
}

// RouteEvent returns false as labels don't handle events.
func (l *Label) RouteEvent(msg tea.Msg) (bool, tea.Cmd) {
	return false, nil
}

// View renders the label text with optional padding, margin, and styling.
func (l *Label) View() string {
	text := l.Text

	// Apply fixed width with alignment
	if l.Width > 0 {
		textWidth := len([]rune(text))
		if textWidth < l.Width {
			padding := l.Width - textWidth
			switch l.HAlign {
			case HAlignLeft:
				text = fmt.Sprintf("%-*s", l.Width, text)
			case HAlignCenter:
				left := padding / 2
				right := padding - left
				text = fmt.Sprintf("%*s%s%*s", left, "", l.Text, right, "")
			case HAlignRight:
				text = fmt.Sprintf("%*s", l.Width, text)
			}
		}
	}

	// Apply padding (inner spacing)
	text = ApplySpacing(text, l.Padding)

	// Apply vertical alignment if height is set
	if l.Height > 0 {
		text = AlignVertical(text, l.Height, l.VAlign, l.Width)
	}

	// Apply style
	if l.Style.Value() != "" {
		text = l.Style.Render(text)
	}

	// Apply margin (outer spacing)
	return ApplySpacing(text, l.Margin)
}

// CanFocus returns false - labels cannot be focused.
func (l *Label) CanFocus() bool {
	return false
}
