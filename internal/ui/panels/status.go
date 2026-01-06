package panels

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/lipgloss"
	"github.com/kluzzebass/lazyhue/internal/ui"
)

const minIndicatorDuration = 250 * time.Millisecond

// StatusBar renders the bottom status bar with keybinding hints.
type StatusBar struct {
	styles  ui.Styles
	keys    ui.KeyMap
	width   int
	message string
	isError bool

	// Activity tracking with minimum visibility
	pollingUntil     time.Time
	discoveringUntil time.Time
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

// SetPolling sets the polling activity indicator.
func (s *StatusBar) SetPolling(active bool) {
	if active {
		s.pollingUntil = time.Now().Add(minIndicatorDuration)
	}
}

// SetDiscovering sets the discovery activity indicator.
func (s *StatusBar) SetDiscovering(active bool) {
	if active {
		s.discoveringUntil = time.Now().Add(minIndicatorDuration)
	}
}

// isPollingVisible returns true if polling indicator should be shown.
func (s *StatusBar) isPollingVisible() bool {
	return time.Now().Before(s.pollingUntil)
}

// isDiscoveringVisible returns true if discovery indicator should be shown.
func (s *StatusBar) isDiscoveringVisible() bool {
	return time.Now().Before(s.discoveringUntil)
}

// View renders the status bar.
func (s *StatusBar) View() string {
	// Build left content
	var leftContent string
	if s.message != "" {
		if s.isError {
			leftContent = s.styles.StatusBar.Foreground(s.styles.Theme.Error).Render(s.message)
		} else {
			leftContent = s.message
		}
	} else {
		leftContent = s.buildHints()
	}

	// Build activity indicators
	indicators := s.buildIndicators()

	// Use lipgloss.Width to account for ANSI escape codes
	leftWidth := lipgloss.Width(leftContent)
	indicatorWidth := lipgloss.Width(indicators)
	availableWidth := s.width - 2 // Leave margin

	padding := availableWidth - leftWidth - indicatorWidth
	if padding < 1 {
		padding = 1
	}

	full := leftContent + strings.Repeat(" ", padding) + indicators
	return s.styles.StatusBar.Width(s.width).Render(full)
}

func (s *StatusBar) buildIndicators() string {
	var indicators []string

	// Polling indicator (green when active)
	if s.isPollingVisible() {
		indicators = append(indicators, s.styles.Success.Render("●"))
	} else {
		indicators = append(indicators, s.styles.Muted.Render("○"))
	}

	// Discovery indicator (blue when active)
	if s.isDiscoveringVisible() {
		indicators = append(indicators, s.styles.Subtitle.Render("●"))
	} else {
		indicators = append(indicators, s.styles.Muted.Render("○"))
	}

	return strings.Join(indicators, " ")
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
