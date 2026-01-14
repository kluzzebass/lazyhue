package layout

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/kluzzebass/lazyhue/internal/ui2/component"
)

// RowChild represents a child component in a row with its width specification.
type RowChild struct {
	Width     int                 // Fixed width (0 = flex/remaining space)
	Component component.Component
}

// Row is a horizontal layout container that renders children side-by-side.
// It routes key events to the focused child and mouse events by position.
type Row struct {
	*component.BaseComponent
	children []RowChild
	focused  int // Index of focused child (-1 = none)
}

// NewRow creates a new row with the given children.
func NewRow(children ...RowChild) *Row {
	r := &Row{
		BaseComponent: component.NewBaseComponent(),
		children:      children,
		focused:       -1,
	}

	// Set parent references and find first focusable child
	for i := range r.children {
		r.children[i].Component.SetParent(r)
		if r.focused < 0 && r.children[i].Component.CanFocus() {
			r.focused = i
		}
	}

	return r
}

// Children returns the child components.
func (r *Row) Children() []component.Component {
	children := make([]component.Component, len(r.children))
	for i, child := range r.children {
		children[i] = child.Component
	}
	return children
}

// Update handles events by delegating to the focused child.
func (r *Row) Update(msg tea.Msg) (component.Component, tea.Cmd) {
	if r.focused >= 0 && r.focused < len(r.children) {
		child := r.children[r.focused].Component
		updated, cmd := child.Update(msg)
		r.children[r.focused].Component = updated
		return r, cmd
	}
	return r, nil
}

// RouteEvent routes events to focusable children.
func (r *Row) RouteEvent(msg tea.Msg) (bool, tea.Cmd) {
	// Try the focused child first
	if r.focused >= 0 && r.focused < len(r.children) {
		child := r.children[r.focused].Component
		if handled, cmd := child.RouteEvent(msg); handled {
			return true, cmd
		}
	}

	// Try other focusable children
	for i, child := range r.children {
		if i == r.focused {
			continue
		}
		if !child.Component.CanFocus() {
			continue
		}
		if handled, cmd := child.Component.RouteEvent(msg); handled {
			r.focused = i
			return true, cmd
		}
	}

	return false, nil
}

// View renders children side by side.
func (r *Row) View() string {
	if len(r.children) == 0 {
		return ""
	}

	var out strings.Builder
	for _, child := range r.children {
		view := child.Component.View()

		// If fixed width, pad or truncate
		if child.Width > 0 {
			viewWidth := runeWidth(view)
			if viewWidth < child.Width {
				view += strings.Repeat(" ", child.Width-viewWidth)
			} else if viewWidth > child.Width {
				view = truncateToWidth(view, child.Width)
			}
		}

		out.WriteString(view)
	}

	return out.String()
}

// CanFocus returns true if any child can focus.
func (r *Row) CanFocus() bool {
	for _, child := range r.children {
		if child.Component.CanFocus() {
			return true
		}
	}
	return false
}

// Focus focuses this row and its first focusable child.
func (r *Row) Focus() {
	r.BaseComponent.Focus()

	if r.focused < 0 {
		// Find first focusable child
		for i, child := range r.children {
			if child.Component.CanFocus() {
				r.focused = i
				child.Component.Focus()
				break
			}
		}
	} else if r.focused < len(r.children) {
		r.children[r.focused].Component.Focus()
	}
}

// Blur blurs this row and all children.
func (r *Row) Blur() {
	r.BaseComponent.Blur()

	for _, child := range r.children {
		child.Component.Blur()
	}
}

// GetFocusedChild returns the currently focused child, or nil if none.
func (r *Row) GetFocusedChild() component.Component {
	if r.focused >= 0 && r.focused < len(r.children) {
		return r.children[r.focused].Component
	}
	return nil
}

// SetFocusedChild focuses the child at the given index.
func (r *Row) SetFocusedChild(index int) {
	if index < 0 || index >= len(r.children) {
		return
	}
	if !r.children[index].Component.CanFocus() {
		return
	}

	// Blur current
	if r.focused >= 0 && r.focused < len(r.children) {
		r.children[r.focused].Component.Blur()
	}

	r.focused = index
	r.children[index].Component.Focus()
}

// Helper functions

// runeWidth returns the display width of a string (simple version).
func runeWidth(s string) int {
	return len([]rune(s))
}

// truncateToWidth truncates a string to fit within width.
func truncateToWidth(s string, width int) string {
	runes := []rune(s)
	if len(runes) <= width {
		return s
	}
	return string(runes[:width])
}
