package component

import tea "github.com/charmbracelet/bubbletea/v2"

// Component represents a unified UI element that handles layout, event routing, and rendering.
// Every UI element (containers, panels, forms, modals) implements this interface.
type Component interface {
	// Layout calculates and stores bounds for this component and its children.
	// Called when window size changes or layout needs recalculation.
	Layout(bounds Rect)

	// Bounds returns the current bounds of this component.
	Bounds() Rect

	// Update handles an event and returns the updated component and any command.
	// This is where component-specific logic lives.
	Update(msg tea.Msg) (Component, tea.Cmd)

	// RouteEvent attempts to route an event through this component's hierarchy.
	// Returns true if the event was handled and consumed.
	// Containers route to children, leaf components handle directly.
	RouteEvent(msg tea.Msg) (handled bool, cmd tea.Cmd)

	// View renders this component to a string.
	View() string

	// Focus management
	IsFocused() bool
	CanFocus() bool
	Focus()
	Blur()

	// Hierarchy navigation
	Children() []Component
	Parent() Component
	SetParent(parent Component)
}

// Rect represents a rectangular area with position and size.
type Rect struct {
	X      int
	Y      int
	Width  int
	Height int
}

// Contains returns true if the point (x, y) is within this rectangle.
func (r Rect) Contains(x, y int) bool {
	return x >= r.X && x < r.X+r.Width &&
		y >= r.Y && y < r.Y+r.Height
}

// Size represents a component's size specification.
type Size interface {
	// Calculate returns the actual size in pixels/cells given the total available space.
	Calculate(total int) int
	IsFlex() bool
}

// FixedSize represents a fixed size in pixels/cells.
type FixedSize struct {
	Value int
}

// Calculate returns the fixed value.
func (f FixedSize) Calculate(total int) int {
	return f.Value
}

// IsFlex returns false for fixed sizes.
func (f FixedSize) IsFlex() bool {
	return false
}

// Fixed creates a fixed size specification.
func Fixed(value int) Size {
	return FixedSize{Value: value}
}

// FlexSize represents a flexible size as a proportion of available space.
type FlexSize struct {
	Weight float64
}

// Calculate returns the proportional size based on the weight.
func (f FlexSize) Calculate(total int) int {
	return int(float64(total) * f.Weight)
}

// IsFlex returns true for flex sizes.
func (f FlexSize) IsFlex() bool {
	return true
}

// Flex creates a flex size specification with the given weight (0.0 to 1.0).
func Flex(weight float64) Size {
	return FlexSize{Weight: weight}
}

// FocusState represents the focus state of a component.
type FocusState int

const (
	// FocusNone means the component cannot receive focus.
	FocusNone FocusState = iota
	// FocusPassive means the component can receive focus but doesn't show a visual indicator.
	FocusPassive
	// FocusActive means the component has focus and shows a visual indicator.
	FocusActive
)
