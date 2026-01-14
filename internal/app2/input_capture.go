package app2

import tea "github.com/charmbracelet/bubbletea/v2"

// InputCapture represents anything that can capture user input (modal, form, etc.)
// Captures are organized in a priority stack - higher priority captures receive
// events before lower priority ones.
type InputCapture interface {
	// HandleEvent processes an event. Returns true if event was handled and consumed.
	// If handled=true, the event will not be passed to lower-priority captures or
	// the normal event routing system.
	HandleEvent(msg tea.Msg) (handled bool, cmd tea.Cmd)

	// IsActive returns whether this capture should receive events.
	// Inactive captures are skipped during event routing.
	IsActive() bool

	// Name returns a debug name for this capture (for logging/debugging).
	Name() string
}

// InputStack manages priority-ordered input captures.
// Captures are stored in priority order - top of stack has highest priority.
type InputStack struct {
	captures []InputCapture
}

// NewInputStack creates a new empty input stack.
func NewInputStack() *InputStack {
	return &InputStack{captures: []InputCapture{}}
}

// Push adds a capture to the stack.
// Captures are pushed in priority order: first pushed = lowest priority,
// last pushed = highest priority.
func (s *InputStack) Push(c InputCapture) {
	s.captures = append(s.captures, c)
}

// Pop removes the top capture from the stack.
// This is rarely needed as captures manage their own active state.
func (s *InputStack) Pop() {
	if len(s.captures) > 0 {
		s.captures = s.captures[:len(s.captures)-1]
	}
}

// Route attempts to route an event through the capture stack.
// Returns true if the event was handled by any active capture.
// Events are routed from top to bottom (highest to lowest priority).
func (s *InputStack) Route(msg tea.Msg) (handled bool, cmd tea.Cmd) {
	// Try from top to bottom (highest priority first)
	for i := len(s.captures) - 1; i >= 0; i-- {
		capture := s.captures[i]
		if !capture.IsActive() {
			continue // Skip inactive captures
		}
		if handled, cmd := capture.HandleEvent(msg); handled {
			return true, cmd // Event consumed
		}
	}
	return false, nil // No capture handled the event
}

// HasActive returns true if any capture in the stack is active.
// This is used to determine if global key processing should be skipped.
func (s *InputStack) HasActive() bool {
	for _, c := range s.captures {
		if c.IsActive() {
			return true
		}
	}
	return false
}

// ActiveCaptures returns the names of all currently active captures.
// Useful for debugging to see what's capturing input.
func (s *InputStack) ActiveCaptures() []string {
	var names []string
	for _, c := range s.captures {
		if c.IsActive() {
			names = append(names, c.Name())
		}
	}
	return names
}
