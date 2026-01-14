package field

import (
	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/kluzzebass/lazyhue/internal/ui"
	"github.com/kluzzebass/lazyhue/internal/ui/component"
	zone "github.com/lrstanley/bubblezone/v2"
)

// HeaderComponent is a non-interactive section header.
// It cannot be focused and does not handle any events.
type HeaderComponent struct {
	*BaseField
}

// NewHeaderComponent creates a new header component.
func NewHeaderComponent(id, label string, styles *ui.Styles, zones *zone.Manager) *HeaderComponent {
	base := NewBaseField(id, label, styles, zones)
	// Headers cannot be focused
	base.BaseComponent.SetFocusState(component.FocusNone)

	return &HeaderComponent{
		BaseField: base,
	}
}

// CanFocus returns false - headers cannot be focused.
func (h *HeaderComponent) CanFocus() bool {
	return false
}

// Update is a no-op for headers.
func (h *HeaderComponent) Update(msg tea.Msg) (component.Component, tea.Cmd) {
	return h, nil
}

// RouteEvent is a no-op for headers - they don't handle events.
func (h *HeaderComponent) RouteEvent(msg tea.Msg) (bool, tea.Cmd) {
	return false, nil
}

// ViewControl renders only the control portion (which is empty for headers).
// Headers are label-only, so this returns an empty string.
func (h *HeaderComponent) ViewControl() string {
	return ""
}

// View renders the header as a styled subtitle with a blank line above for spacing.
func (h *HeaderComponent) View() string {
	return "\n" + h.Styles.Subtitle.Render(h.Label)
}
