package component

import (
	"fmt"
	"log/slog"
	"strings"

	tea "github.com/charmbracelet/bubbletea/v2"
)

// ComponentChild represents a child component with its size specification.
type ComponentChild struct {
	Size      Size
	Component Component
}

// Container is a component that contains and manages child components.
// It handles layout calculation and event routing to children.
type Container struct {
	*BaseComponent
	children   []ComponentChild
	focused    int  // Index of focused child (-1 = none)
	horizontal bool // true = HSplit, false = VSplit
}

// NewHSplit creates a horizontal split container (side by side).
func NewHSplit(children ...ComponentChild) *Container {
	c := &Container{
		BaseComponent: NewBaseComponent(),
		children:      children,
		focused:       -1,
		horizontal:    true,
	}
	c.BaseComponent.SetFocusState(FocusPassive)

	// Set parent references
	for i := range c.children {
		c.children[i].Component.SetParent(c)
	}

	return c
}

// NewVSplit creates a vertical split container (top to bottom).
func NewVSplit(children ...ComponentChild) *Container {
	c := &Container{
		BaseComponent: NewBaseComponent(),
		children:      children,
		focused:       -1,
		horizontal:    false,
	}
	c.BaseComponent.SetFocusState(FocusPassive)

	// Set parent references
	for i := range c.children {
		c.children[i].Component.SetParent(c)
	}

	return c
}

// Layout calculates bounds for this container and recursively layouts children.
func (c *Container) Layout(bounds Rect) {
	c.bounds = bounds

	if len(c.children) == 0 {
		return
	}

	// Calculate total flex weight and fixed space
	var totalFlexWeight float64
	var totalFixedSpace int

	for _, child := range c.children {
		if child.Size.IsFlex() {
			totalFlexWeight += child.Size.(FlexSize).Weight
		} else {
			totalFixedSpace += child.Size.Calculate(0)
		}
	}

	// Available space for flex children
	var availableSpace int
	if c.horizontal {
		availableSpace = bounds.Width - totalFixedSpace
	} else {
		availableSpace = bounds.Height - totalFixedSpace
	}

	// Layout children
	currentX := bounds.X
	currentY := bounds.Y

	for i, child := range c.children {
		var childBounds Rect

		if c.horizontal {
			// Horizontal split - allocate width
			var width int
			if child.Size.IsFlex() {
				flexSize := child.Size.(FlexSize)
				if totalFlexWeight > 0 {
					width = int(float64(availableSpace) * (flexSize.Weight / totalFlexWeight))
				}
			} else {
				width = child.Size.Calculate(bounds.Width)
			}

			childBounds = Rect{
				X:      currentX,
				Y:      currentY,
				Width:  width,
				Height: bounds.Height,
			}
			currentX += width
		} else {
			// Vertical split - allocate height
			var height int
			if child.Size.IsFlex() {
				flexSize := child.Size.(FlexSize)
				if totalFlexWeight > 0 {
					height = int(float64(availableSpace) * (flexSize.Weight / totalFlexWeight))
				}
			} else {
				height = child.Size.Calculate(bounds.Height)
			}

			childBounds = Rect{
				X:      currentX,
				Y:      currentY,
				Width:  bounds.Width,
				Height: height,
			}
			currentY += height
		}

		c.children[i].Component.Layout(childBounds)
	}
}

// RouteEvent routes events to children based on focus and hit-testing.
func (c *Container) RouteEvent(msg tea.Msg) (bool, tea.Cmd) {
	if len(c.children) == 0 {
		slog.Debug("Container.RouteEvent: no children, returning unhandled")
		return false, nil
	}

	msgType := fmt.Sprintf("%T", msg)
	slog.Debug("Container.RouteEvent", "msg_type", msgType, "focused", c.focused)

	// 1. For mouse events, do hit-testing first (takes priority over focus)
	if mouse, ok := msg.(tea.MouseClickMsg); ok {
		slog.Debug("Container.RouteEvent: mouse click, testing children", "x", mouse.X, "y", mouse.Y)
		return c.routeMouseEvent(mouse)
	}

	// 2. For non-mouse events, try ALL children that can focus
	// This allows modals deep in the tree to capture events when active
	for i, child := range c.children {
		if !child.Component.CanFocus() {
			continue
		}
		slog.Debug("Container.RouteEvent: trying focusable child", "child", i)
		if handled, cmd := child.Component.RouteEvent(msg); handled {
			slog.Debug("Container.RouteEvent: child handled event", "child", i)
			return true, cmd
		}
	}

	slog.Debug("Container.RouteEvent: no child handled event")
	return false, nil
}

// routeMouseEvent routes mouse clicks to children based on bounds.
func (c *Container) routeMouseEvent(mouse tea.MouseClickMsg) (bool, tea.Cmd) {
	x, y := mouse.X, mouse.Y

	// Find child containing the click point
	for i, child := range c.children {
		bounds := child.Component.Bounds()
		slog.Debug("Container.routeMouseEvent: testing child bounds",
			"child", i, "bounds_x", bounds.X, "bounds_y", bounds.Y,
			"bounds_w", bounds.Width, "bounds_h", bounds.Height)

		if bounds.Contains(x, y) {
			slog.Debug("Container.routeMouseEvent: child contains click, focusing and routing", "child", i)
			// Focus this child (and blur siblings)
			c.FocusChild(i)

			// Route event to child
			if handled, cmd := child.Component.RouteEvent(mouse); handled {
				slog.Debug("Container.routeMouseEvent: child handled event", "child", i)
				return true, cmd
			}

			// Even if not handled, consume the click (focused the component)
			slog.Debug("Container.routeMouseEvent: child focused but did not handle event", "child", i)
			return true, nil
		}
	}

	slog.Debug("Container.routeMouseEvent: no child contains click", "x", x, "y", y)
	return false, nil
}

// Update updates this container. For containers, this typically just routes to children.
func (c *Container) Update(msg tea.Msg) (Component, tea.Cmd) {
	// Containers delegate to RouteEvent
	if handled, cmd := c.RouteEvent(msg); handled {
		return c, cmd
	}
	return c, nil
}

// View renders this container by rendering all children.
func (c *Container) View() string {
	if len(c.children) == 0 {
		return ""
	}

	if c.horizontal {
		return c.viewHorizontal()
	}
	return c.viewVertical()
}

// viewHorizontal renders children side by side.
func (c *Container) viewHorizontal() string {
	// Render all children
	childViews := make([]string, len(c.children))
	for i, child := range c.children {
		childViews[i] = child.Component.View()
	}

	// Split into lines
	childLines := make([][]string, len(childViews))
	maxLines := 0
	for i, view := range childViews {
		lines := strings.Split(view, "\n")
		childLines[i] = lines
		if len(lines) > maxLines {
			maxLines = len(lines)
		}
	}

	// Combine horizontally
	var result strings.Builder
	for lineIdx := 0; lineIdx < maxLines; lineIdx++ {
		for childIdx, lines := range childLines {
			if lineIdx < len(lines) {
				result.WriteString(lines[lineIdx])
			} else {
				// Pad with spaces if this child doesn't have this line
				width := c.children[childIdx].Component.Bounds().Width
				result.WriteString(strings.Repeat(" ", width))
			}
		}
		if lineIdx < maxLines-1 {
			result.WriteString("\n")
		}
	}

	return result.String()
}

// viewVertical renders children top to bottom.
func (c *Container) viewVertical() string {
	var result strings.Builder

	for i, child := range c.children {
		result.WriteString(child.Component.View())
		if i < len(c.children)-1 {
			result.WriteString("\n")
		}
	}

	return result.String()
}

// Children returns the child components.
func (c *Container) Children() []Component {
	children := make([]Component, len(c.children))
	for i, child := range c.children {
		children[i] = child.Component
	}
	return children
}

// Focus focuses this container and the first focusable child.
func (c *Container) Focus() {
	c.BaseComponent.Focus()

	// Focus first focusable child if none focused
	if c.focused < 0 {
		for i, child := range c.children {
			if child.Component.CanFocus() {
				c.FocusChild(i)
				break
			}
		}
	}
}

// FocusChild focuses the child at the given index and blurs all other children.
func (c *Container) FocusChild(index int) {
	if index < 0 || index >= len(c.children) {
		return
	}

	// Blur all children first
	for _, child := range c.children {
		child.Component.Blur()
	}

	// Focus the specified child
	c.focused = index
	c.children[index].Component.Focus()
}

// StackedContainer is a container that renders children on top of each other.
// The last child is rendered on top and gets priority for event routing.
// This is used for modals and overlays.
type StackedContainer struct {
	*BaseComponent
	children []Component
}

// NewStackedContainer creates a new stacked container.
func NewStackedContainer(children ...Component) *StackedContainer {
	s := &StackedContainer{
		BaseComponent: NewBaseComponent(),
		children:      children,
	}
	s.BaseComponent.SetFocusState(FocusPassive)

	// Set parent references
	for i := range s.children {
		s.children[i].SetParent(s)
	}

	return s
}

// Layout passes the full bounds to all children (they're stacked).
func (s *StackedContainer) Layout(bounds Rect) {
	s.bounds = bounds

	// All children get the full bounds since they stack
	for _, child := range s.children {
		child.Layout(bounds)
	}
}

// RouteEvent routes to children from top to bottom (highest priority first).
func (s *StackedContainer) RouteEvent(msg tea.Msg) (bool, tea.Cmd) {
	msgType := fmt.Sprintf("%T", msg)
	slog.Debug("StackedContainer.RouteEvent", "msg_type", msgType, "children", len(s.children))

	// Try from top to bottom (last added = highest priority)
	for i := len(s.children) - 1; i >= 0; i-- {
		child := s.children[i]
		if !child.CanFocus() {
			slog.Debug("StackedContainer.RouteEvent: child cannot focus, skipping", "child", i)
			continue // Skip non-focusable children
		}
		slog.Debug("StackedContainer.RouteEvent: trying child", "child", i)
		if handled, cmd := child.RouteEvent(msg); handled {
			slog.Debug("StackedContainer.RouteEvent: child handled event", "child", i)
			return true, cmd
		}
		slog.Debug("StackedContainer.RouteEvent: child did not handle event", "child", i)
	}
	slog.Debug("StackedContainer.RouteEvent: no child handled event")
	return false, nil
}

// Update updates this container.
func (s *StackedContainer) Update(msg tea.Msg) (Component, tea.Cmd) {
	if handled, cmd := s.RouteEvent(msg); handled {
		return s, cmd
	}
	return s, nil
}

// View renders all children, with later children drawn on top.
func (s *StackedContainer) View() string {
	if len(s.children) == 0 {
		return ""
	}

	// For simplicity, just render the topmost active child
	// (Proper overlay rendering would require more sophisticated compositing)
	for i := len(s.children) - 1; i >= 0; i-- {
		if s.children[i].CanFocus() {
			return s.children[i].View()
		}
	}

	// Fallback to bottom child
	return s.children[0].View()
}

// Children returns the child components.
func (s *StackedContainer) Children() []Component {
	return s.children
}

// Push adds a child to the top of the stack.
func (s *StackedContainer) Push(child Component) {
	child.SetParent(s)
	s.children = append(s.children, child)
	child.Layout(s.bounds) // Layout new child
}

// Pop removes the top child from the stack.
func (s *StackedContainer) Pop() Component {
	if len(s.children) == 0 {
		return nil
	}
	child := s.children[len(s.children)-1]
	s.children = s.children[:len(s.children)-1]
	return child
}

// BlurAll recursively blurs all components in a tree.
// Call this on the root before activating a modal to ensure
// the modal is the only component with FocusActive.
func BlurAll(c Component) {
	if c == nil {
		return
	}
	c.Blur()
	for _, child := range c.Children() {
		BlurAll(child)
	}
}
