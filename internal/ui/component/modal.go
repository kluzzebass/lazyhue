package component

import (
	"log/slog"

	tea "github.com/charmbracelet/bubbletea/v2"
)

// ModalComponent is a component that captures all input when active.
// It's used for overlays like rename dialogs, confirmation dialogs, etc.
type ModalComponent struct {
	*BaseComponent
	id       string
	active   bool
	onUpdate func(msg tea.Msg) (handled bool, cmd tea.Cmd) // Custom update handler
	onView   func() string                                  // Custom view handler
}

// NewModalComponent creates a new modal component.
// onUpdate is called for each message when the modal is active.
// onView is called to render the modal content.
func NewModalComponent(id string, onUpdate func(tea.Msg) (bool, tea.Cmd), onView func() string) *ModalComponent {
	m := &ModalComponent{
		BaseComponent: NewBaseComponent(),
		id:            id,
		active:        false,
		onUpdate:      onUpdate,
		onView:        onView,
	}
	// Modals start inactive (FocusNone means CanFocus returns false)
	m.BaseComponent.SetFocusState(FocusNone)
	return m
}

// SetActive activates or deactivates the modal.
// When active, the modal captures all input.
func (m *ModalComponent) SetActive(active bool) {
	m.active = active
	if active {
		m.BaseComponent.SetFocusState(FocusActive)
	} else {
		m.BaseComponent.SetFocusState(FocusNone)
	}
	slog.Debug("ModalComponent.SetActive", "id", m.id, "active", active)
}

// IsActive returns whether the modal is currently active.
func (m *ModalComponent) IsActive() bool {
	return m.active
}

// CanFocus returns true only when the modal is active.
func (m *ModalComponent) CanFocus() bool {
	return m.active
}

// RouteEvent handles events when the modal is active.
func (m *ModalComponent) RouteEvent(msg tea.Msg) (bool, tea.Cmd) {
	if !m.active {
		return false, nil
	}

	slog.Debug("ModalComponent.RouteEvent", "id", m.id, "msg_type", slog.Any("%T", msg))

	if m.onUpdate != nil {
		return m.onUpdate(msg)
	}

	return true, nil // Consume all events when active even without handler
}

// Update delegates to RouteEvent.
func (m *ModalComponent) Update(msg tea.Msg) (Component, tea.Cmd) {
	_, cmd := m.RouteEvent(msg)
	return m, cmd
}

// View renders the modal content.
func (m *ModalComponent) View() string {
	if m.onView != nil {
		return m.onView()
	}
	return ""
}

// ID returns the modal's identifier.
func (m *ModalComponent) ID() string {
	return m.id
}
