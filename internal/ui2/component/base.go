package component

import tea "github.com/charmbracelet/bubbletea/v2"

// BaseComponent provides common functionality for all components.
// Embed this in concrete component types to get default implementations.
type BaseComponent struct {
	bounds Rect
	focus  FocusState
	parent Component
}

// NewBaseComponent creates a new base component with default values.
func NewBaseComponent() *BaseComponent {
	return &BaseComponent{
		bounds: Rect{},
		focus:  FocusNone,
		parent: nil,
	}
}

// Layout sets the bounds for this component.
// Override in container components to recursively layout children.
func (b *BaseComponent) Layout(bounds Rect) {
	b.bounds = bounds
}

// Bounds returns the current bounds of this component.
func (b *BaseComponent) Bounds() Rect {
	return b.bounds
}

// Update is a no-op for the base component.
// Override in concrete components to handle events.
func (b *BaseComponent) Update(msg tea.Msg) (Component, tea.Cmd) {
	return b, nil
}

// RouteEvent is a no-op for the base component (leaf nodes).
// Override in container components to route to children.
func (b *BaseComponent) RouteEvent(msg tea.Msg) (bool, tea.Cmd) {
	return false, nil
}

// View returns an empty string for the base component.
// Override in concrete components to render content.
func (b *BaseComponent) View() string {
	return ""
}

// IsFocused returns true if this component has active focus.
func (b *BaseComponent) IsFocused() bool {
	return b.focus == FocusActive
}

// CanFocus returns true if this component can receive focus.
// By default, components can receive passive focus.
// Override to return false for non-focusable components.
func (b *BaseComponent) CanFocus() bool {
	return b.focus != FocusNone
}

// Focus sets this component to active focus.
// Note: Parent containers are responsible for blurring siblings.
func (b *BaseComponent) Focus() {
	b.focus = FocusActive
}

// Blur removes focus from this component.
func (b *BaseComponent) Blur() {
	if b.focus == FocusActive {
		b.focus = FocusPassive
	}
}

// Children returns an empty slice for the base component (leaf node).
// Override in container components to return actual children.
func (b *BaseComponent) Children() []Component {
	return nil
}

// Parent returns the parent component.
func (b *BaseComponent) Parent() Component {
	return b.parent
}

// SetParent sets the parent component.
func (b *BaseComponent) SetParent(parent Component) {
	b.parent = parent
}

// SetFocusState sets the focus state directly.
// Useful for initialization.
func (b *BaseComponent) SetFocusState(state FocusState) {
	b.focus = state
}
