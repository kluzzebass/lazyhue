package field

import (
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/kluzzebass/lazyhue/internal/ui"
	"github.com/kluzzebass/lazyhue/internal/ui/component"
	zone "github.com/lrstanley/bubblezone/v2"
)

// buttonResetMsg is sent to reset the button's pressed state.
type buttonResetMsg struct {
	fieldID string
}

// ButtonComponent is a clickable button that triggers an action.
type ButtonComponent struct {
	*BaseField

	ButtonLabel string
	pressed     bool // Visual feedback state
}

// NewButtonComponent creates a new button component.
func NewButtonComponent(id, label, buttonLabel string, styles *ui.Styles, zones *zone.Manager) *ButtonComponent {
	return &ButtonComponent{
		BaseField:   NewBaseField(id, label, styles, zones),
		ButtonLabel: buttonLabel,
	}
}

// Update handles events for the button.
func (b *ButtonComponent) Update(msg tea.Msg) (component.Component, tea.Cmd) {
	// Handle reset message
	if resetMsg, ok := msg.(buttonResetMsg); ok {
		if resetMsg.fieldID == b.ID {
			b.pressed = false
		}
		return b, nil
	}

	if b.ReadOnly {
		return b, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter", " ":
			return b.press()
		}

	case tea.MouseClickMsg:
		if msg.Button == tea.MouseLeft {
			if b.Zones != nil {
				if z := b.Zones.Get(b.ZoneID()); z != nil && z.InBounds(msg) {
					return b.press()
				}
			}
		}
	}

	return b, nil
}

// RouteEvent routes events to this component.
func (b *ButtonComponent) RouteEvent(msg tea.Msg) (bool, tea.Cmd) {
	// Handle reset message
	if resetMsg, ok := msg.(buttonResetMsg); ok {
		if resetMsg.fieldID == b.ID {
			b.pressed = false
			return true, nil
		}
		return false, nil
	}

	if b.ReadOnly {
		return false, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter", " ":
			_, cmd := b.press()
			return true, cmd
		}

	case tea.MouseClickMsg:
		if msg.Button == tea.MouseLeft {
			if b.Zones != nil {
				if z := b.Zones.Get(b.ZoneID()); z != nil && z.InBounds(msg) {
					_, cmd := b.press()
					return true, cmd
				}
			}
		}
	}

	return false, nil
}

// press triggers the button action and visual feedback.
func (b *ButtonComponent) press() (component.Component, tea.Cmd) {
	b.pressed = true

	// Return both the action message and a timer to reset the pressed state
	return b, tea.Batch(
		func() tea.Msg {
			return FieldChangedMsg{
				FieldID: b.ID,
				Value:   ButtonValue{Pressed: true},
			}
		},
		tea.Tick(250*time.Millisecond, func(t time.Time) tea.Msg {
			return buttonResetMsg{fieldID: b.ID}
		}),
	)
}

// ViewControl renders only the control portion (no label).
func (b *ButtonComponent) ViewControl() string {
	// Render as a button-like element with visual feedback when pressed
	var style = b.Styles.Base
	if b.pressed {
		style = b.Styles.Accent
	} else if b.IsFocused() {
		style = b.Styles.Focused
	}

	buttonStr := style.Render(fmt.Sprintf("[ %s ]", b.ButtonLabel))

	// Wrap with zone for mouse detection
	if b.Zones != nil {
		return b.Zones.Mark(b.ZoneID(), buttonStr)
	}

	return buttonStr
}

// View renders the button field (label + control for backwards compatibility).
func (b *ButtonComponent) View() string {
	labelStr := b.Label
	if b.MaxLabelWidth > 0 {
		labelStr = fmt.Sprintf("%-*s", b.MaxLabelWidth, b.Label)
	}

	return fmt.Sprintf("  %s  %s", labelStr, b.ViewControl())
}
