package panels

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/kluzzebass/lazyhue/internal/ui"
)

// StatusBar renders the bottom status bar with keybinding hints.
type StatusBar struct {
	styles  ui.Styles
	keys    ui.KeyMap
	width   int
	message string
	isError bool
}

// NewStatusBar creates a new status bar.
func NewStatusBar(styles ui.Styles, keys ui.KeyMap) *StatusBar {
	return &StatusBar{
		styles: styles,
		keys:   keys,
	}
}

// SetWidth updates the status bar width.
func (s *StatusBar) SetWidth(width int) {
	s.width = width
}

// SetMessage sets a temporary status message.
func (s *StatusBar) SetMessage(msg string, isError bool) {
	s.message = msg
	s.isError = isError
}

// ClearMessage clears the status message.
func (s *StatusBar) ClearMessage() {
	s.message = ""
	s.isError = false
}

// View renders the status bar.
func (s *StatusBar) View() string {
	if s.message != "" {
		style := s.styles.StatusBar
		if s.isError {
			style = style.Foreground(s.styles.Theme.Error)
		}
		return style.Width(s.width).Render(s.message)
	}

	// Build keybinding hints
	hints := s.buildHints()
	return s.styles.StatusBar.Width(s.width).Render(hints)
}

func (s *StatusBar) buildHints() string {
	bindings := s.keys.ShortHelp()
	hints := make([]string, 0, len(bindings))

	for _, b := range bindings {
		if !b.Enabled() {
			continue
		}
		hint := fmt.Sprintf("%s %s", keyStr(b), b.Help().Desc)
		hints = append(hints, hint)
	}

	return strings.Join(hints, "  ")
}

func keyStr(k key.Binding) string {
	keys := k.Keys()
	if len(keys) == 0 {
		return ""
	}
	return keys[0]
}
