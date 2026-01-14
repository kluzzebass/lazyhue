package component

import (
	"log/slog"

	"github.com/charmbracelet/bubbles/v2/textinput"
	tea "github.com/charmbracelet/bubbletea/v2"
)

// TextInputConfirmedMsg is sent when the user confirms the text input.
type TextInputConfirmedMsg struct {
	ModalID string
	Value   string
}

// TextInputCancelledMsg is sent when the user cancels the text input.
type TextInputCancelledMsg struct {
	ModalID string
}

// TextInputModal is a modal component that captures text input.
// It owns its own textinput.Model and communicates via messages.
type TextInputModal struct {
	*BaseComponent
	id     string
	active bool
	input  textinput.Model
}

// NewTextInputModal creates a new text input modal.
func NewTextInputModal(id string) *TextInputModal {
	ti := textinput.New()
	ti.CharLimit = 64

	m := &TextInputModal{
		BaseComponent: NewBaseComponent(),
		id:            id,
		active:        false,
		input:         ti,
	}
	m.BaseComponent.SetFocusState(FocusNone)
	return m
}

// SetActive activates or deactivates the modal.
func (m *TextInputModal) SetActive(active bool) {
	m.active = active
	if active {
		m.BaseComponent.SetFocusState(FocusActive)
		m.input.Focus()
	} else {
		m.BaseComponent.SetFocusState(FocusNone)
		m.input.Blur()
	}
	slog.Debug("TextInputModal.SetActive", "id", m.id, "active", active)
}

// IsActive returns whether the modal is currently active.
func (m *TextInputModal) IsActive() bool {
	return m.active
}

// CanFocus returns true only when the modal is active.
func (m *TextInputModal) CanFocus() bool {
	return m.active
}

// SetValue sets the input value.
func (m *TextInputModal) SetValue(value string) {
	m.input.SetValue(value)
	m.input.CursorEnd()
}

// Value returns the current input value.
func (m *TextInputModal) Value() string {
	return m.input.Value()
}

// SetPlaceholder sets the input placeholder.
func (m *TextInputModal) SetPlaceholder(placeholder string) {
	m.input.Placeholder = placeholder
}

// RouteEvent handles events when the modal is active.
// Returns messages for confirm/cancel, handles input internally.
func (m *TextInputModal) RouteEvent(msg tea.Msg) (bool, tea.Cmd) {
	if !m.active {
		return false, nil
	}

	slog.Debug("TextInputModal.RouteEvent", "id", m.id, "msg_type", slog.Any("%T", msg))

	if key, ok := msg.(tea.KeyMsg); ok {
		switch key.String() {
		case "enter":
			// Return confirmed message with current value
			return true, func() tea.Msg {
				return TextInputConfirmedMsg{
					ModalID: m.id,
					Value:   m.input.Value(),
				}
			}
		case "esc":
			// Return cancelled message
			return true, func() tea.Msg {
				return TextInputCancelledMsg{
					ModalID: m.id,
				}
			}
		}
	}

	// Pass all other messages to the text input
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return true, cmd
}

// Update delegates to RouteEvent.
func (m *TextInputModal) Update(msg tea.Msg) (Component, tea.Cmd) {
	_, cmd := m.RouteEvent(msg)
	return m, cmd
}

// View renders the text input.
func (m *TextInputModal) View() string {
	return m.input.View()
}

// ID returns the modal's identifier.
func (m *TextInputModal) ID() string {
	return m.id
}

// Input returns the underlying text input for styling purposes.
func (m *TextInputModal) Input() *textinput.Model {
	return &m.input
}
