package component

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea/v2"
)

// MockComponent is a test component that tracks what messages it received.
type MockComponent struct {
	*BaseComponent
	id              string
	receivedMsgs    []tea.Msg
	handleEvent     bool // If true, HandleEvent returns true
	canFocus        bool
	updateCallCount int
}

func NewMockComponent(id string, handleEvent bool, canFocus bool) *MockComponent {
	m := &MockComponent{
		BaseComponent:   NewBaseComponent(),
		id:              id,
		receivedMsgs:    []tea.Msg{},
		handleEvent:     handleEvent,
		canFocus:        canFocus,
		updateCallCount: 0,
	}
	if canFocus {
		m.BaseComponent.SetFocusState(FocusPassive)
	}
	return m
}

func (m *MockComponent) RouteEvent(msg tea.Msg) (bool, tea.Cmd) {
	m.receivedMsgs = append(m.receivedMsgs, msg)
	return m.handleEvent, nil
}

func (m *MockComponent) Update(msg tea.Msg) (Component, tea.Cmd) {
	m.updateCallCount++
	m.receivedMsgs = append(m.receivedMsgs, msg)
	return m, nil
}

func (m *MockComponent) CanFocus() bool {
	return m.canFocus
}

func (m *MockComponent) View() string {
	return m.id
}

// mockKeyMsg is a simple message type for testing.
type mockKeyMsg string

// TestContainerRouting tests that containers route events to focused children.
func TestContainerRouting(t *testing.T) {
	// Create a container with two mock children
	child1 := NewMockComponent("child1", true, true)
	child2 := NewMockComponent("child2", false, true)

	container := NewHSplit(
		ComponentChild{Size: Flex(0.5), Component: child1},
		ComponentChild{Size: Flex(0.5), Component: child2},
	)

	// Layout the container
	container.Layout(Rect{X: 0, Y: 0, Width: 100, Height: 50})

	// Focus child1
	container.focused = 0
	child1.Focus()

	// Send a key event
	msg := mockKeyMsg("a")
	handled, _ := container.RouteEvent(msg)

	// Verify child1 received the event
	if !handled {
		t.Errorf("Expected event to be handled")
	}
	if len(child1.receivedMsgs) != 1 {
		t.Errorf("Expected child1 to receive 1 message, got %d", len(child1.receivedMsgs))
	}
	if len(child2.receivedMsgs) != 0 {
		t.Errorf("Expected child2 to receive 0 messages, got %d", len(child2.receivedMsgs))
	}
}

// TestMouseHitTesting tests that mouse clicks route to the correct child based on bounds.
func TestMouseHitTesting(t *testing.T) {
	// Create a horizontal split with two children
	child1 := NewMockComponent("child1", true, true)
	child2 := NewMockComponent("child2", true, true)

	container := NewHSplit(
		ComponentChild{Size: Flex(0.5), Component: child1},
		ComponentChild{Size: Flex(0.5), Component: child2},
	)

	// Layout: child1 = [0-50], child2 = [50-100]
	container.Layout(Rect{X: 0, Y: 0, Width: 100, Height: 50})

	// Click in child1's area (x=25)
	mouse1 := tea.MouseClickMsg{X: 25, Y: 25}
	handled1, _ := container.RouteEvent(mouse1)

	if !handled1 {
		t.Errorf("Expected mouse click in child1 to be handled")
	}
	if container.focused != 0 {
		t.Errorf("Expected child1 to be focused, got focused=%d", container.focused)
	}
	if !child1.IsFocused() {
		t.Errorf("Expected child1 to have active focus")
	}
	if len(child1.receivedMsgs) != 1 {
		t.Errorf("Expected child1 to receive click, got %d messages", len(child1.receivedMsgs))
	}

	// Click in child2's area (x=75)
	mouse2 := tea.MouseClickMsg{X: 75, Y: 25}
	handled2, _ := container.RouteEvent(mouse2)

	if !handled2 {
		t.Errorf("Expected mouse click in child2 to be handled")
	}
	if container.focused != 1 {
		t.Errorf("Expected child2 to be focused, got focused=%d", container.focused)
	}
	if !child2.IsFocused() {
		t.Errorf("Expected child2 to have active focus")
	}
	if child1.IsFocused() {
		t.Errorf("Expected child1 to be blurred after child2 focus")
	}
}

// TestStackedContainerPriority tests that stacked containers route events top to bottom.
func TestStackedContainerPriority(t *testing.T) {
	// Create a stack with a modal on top
	basePanel := NewMockComponent("base", false, true)
	modal := NewMockComponent("modal", true, true)

	stack := NewStackedContainer(basePanel, modal)
	stack.Layout(Rect{X: 0, Y: 0, Width: 100, Height: 50})

	// Send event - modal should handle it
	msg := mockKeyMsg("a")
	handled, _ := stack.RouteEvent(msg)

	if !handled {
		t.Errorf("Expected event to be handled by modal")
	}
	if len(modal.receivedMsgs) != 1 {
		t.Errorf("Expected modal to receive event, got %d messages", len(modal.receivedMsgs))
	}
	if len(basePanel.receivedMsgs) != 0 {
		t.Errorf("Expected base panel to not receive event, got %d messages", len(basePanel.receivedMsgs))
	}
}

// TestLayoutCalculation tests that containers correctly calculate child bounds.
func TestLayoutCalculation(t *testing.T) {
	child1 := NewMockComponent("child1", false, true)
	child2 := NewMockComponent("child2", false, true)

	// Horizontal split: 40% / 60%
	container := NewHSplit(
		ComponentChild{Size: Flex(0.4), Component: child1},
		ComponentChild{Size: Flex(0.6), Component: child2},
	)

	container.Layout(Rect{X: 0, Y: 0, Width: 100, Height: 50})

	// Check child1 bounds
	bounds1 := child1.Bounds()
	if bounds1.Width != 40 {
		t.Errorf("Expected child1 width=40, got %d", bounds1.Width)
	}
	if bounds1.Height != 50 {
		t.Errorf("Expected child1 height=50, got %d", bounds1.Height)
	}

	// Check child2 bounds
	bounds2 := child2.Bounds()
	if bounds2.Width != 60 {
		t.Errorf("Expected child2 width=60, got %d", bounds2.Width)
	}
	if bounds2.X != 40 {
		t.Errorf("Expected child2 x=40, got %d", bounds2.X)
	}
}

// TestVSplitLayout tests vertical split layout.
func TestVSplitLayout(t *testing.T) {
	child1 := NewMockComponent("child1", false, true)
	child2 := NewMockComponent("child2", false, true)

	// Vertical split: 67% / 33%
	container := NewVSplit(
		ComponentChild{Size: Flex(0.67), Component: child1},
		ComponentChild{Size: Flex(0.33), Component: child2},
	)

	container.Layout(Rect{X: 0, Y: 0, Width: 100, Height: 100})

	// Check child1 bounds
	bounds1 := child1.Bounds()
	if bounds1.Height != 67 {
		t.Errorf("Expected child1 height=67, got %d", bounds1.Height)
	}
	if bounds1.Width != 100 {
		t.Errorf("Expected child1 width=100, got %d", bounds1.Width)
	}

	// Check child2 bounds
	bounds2 := child2.Bounds()
	if bounds2.Height != 33 {
		t.Errorf("Expected child2 height=33, got %d", bounds2.Height)
	}
	if bounds2.Y != 67 {
		t.Errorf("Expected child2 y=67, got %d", bounds2.Y)
	}
}

// TestFocusManagement tests that focusing a component blurs siblings.
func TestFocusManagement(t *testing.T) {
	child1 := NewMockComponent("child1", false, true)
	child2 := NewMockComponent("child2", false, true)

	container := NewHSplit(
		ComponentChild{Size: Flex(0.5), Component: child1},
		ComponentChild{Size: Flex(0.5), Component: child2},
	)

	container.Layout(Rect{X: 0, Y: 0, Width: 100, Height: 50})

	// Focus child1 using container
	container.FocusChild(0)
	if !child1.IsFocused() {
		t.Errorf("Expected child1 to be focused")
	}

	// Focus child2 - should blur child1
	container.FocusChild(1)
	if !child2.IsFocused() {
		t.Errorf("Expected child2 to be focused")
	}
	if child1.IsFocused() {
		t.Errorf("Expected child1 to be blurred when child2 is focused")
	}
}

// TestFixedSize tests layout with fixed sizes.
func TestFixedSize(t *testing.T) {
	child1 := NewMockComponent("child1", false, true)
	child2 := NewMockComponent("child2", false, true)

	// Horizontal: 30 fixed, rest flex
	container := NewHSplit(
		ComponentChild{Size: Fixed(30), Component: child1},
		ComponentChild{Size: Flex(1.0), Component: child2},
	)

	container.Layout(Rect{X: 0, Y: 0, Width: 100, Height: 50})

	bounds1 := child1.Bounds()
	if bounds1.Width != 30 {
		t.Errorf("Expected child1 width=30 (fixed), got %d", bounds1.Width)
	}

	bounds2 := child2.Bounds()
	if bounds2.Width != 70 {
		t.Errorf("Expected child2 width=70 (100-30), got %d", bounds2.Width)
	}
}

// TestRectContains tests the Rect.Contains method.
func TestRectContains(t *testing.T) {
	rect := Rect{X: 10, Y: 20, Width: 50, Height: 30}

	tests := []struct {
		x, y     int
		expected bool
	}{
		{10, 20, true},   // top-left corner
		{59, 49, true},   // bottom-right corner (exclusive)
		{30, 35, true},   // inside
		{5, 25, false},   // left of rect
		{65, 35, false},  // right of rect
		{30, 15, false},  // above rect
		{30, 55, false},  // below rect
		{60, 30, false},  // exactly at right edge (exclusive)
		{30, 50, false},  // exactly at bottom edge (exclusive)
	}

	for _, test := range tests {
		result := rect.Contains(test.x, test.y)
		if result != test.expected {
			t.Errorf("Rect.Contains(%d, %d) = %v, expected %v", test.x, test.y, result, test.expected)
		}
	}
}

// TestStackedContainerPushPop tests push/pop operations on stacked containers.
func TestStackedContainerPushPop(t *testing.T) {
	base := NewMockComponent("base", false, true)
	stack := NewStackedContainer(base)
	stack.Layout(Rect{X: 0, Y: 0, Width: 100, Height: 50})

	if len(stack.Children()) != 1 {
		t.Errorf("Expected 1 child, got %d", len(stack.Children()))
	}

	// Push a modal
	modal := NewMockComponent("modal", true, true)
	stack.Push(modal)

	if len(stack.Children()) != 2 {
		t.Errorf("Expected 2 children after push, got %d", len(stack.Children()))
	}

	// Modal should receive events
	msg := mockKeyMsg("a")
	handled, _ := stack.RouteEvent(msg)

	if !handled {
		t.Errorf("Expected modal to handle event")
	}
	if len(modal.receivedMsgs) != 1 {
		t.Errorf("Expected modal to receive event")
	}

	// Pop the modal
	popped := stack.Pop()
	if popped != modal {
		t.Errorf("Expected to pop modal")
	}
	if len(stack.Children()) != 1 {
		t.Errorf("Expected 1 child after pop, got %d", len(stack.Children()))
	}
}
