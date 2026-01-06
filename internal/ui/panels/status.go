package panels

import (
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/kluzzebass/lazyhue/internal/ui"
)

const minIndicatorDuration = 250 * time.Millisecond

// StatusBar renders the bottom status bar with keybinding hints.
type StatusBar struct {
	styles  ui.Styles
	width   int
	message string
	isError bool

	// Panel bindings - set by app when focus changes
	panelBindings []ui.Binding

	// Popup mode overrides normal hints
	popupHints string

	// Activity tracking with minimum visibility
	pollingUntil     time.Time
	discoveringUntil time.Time
}

// NewStatusBar creates a new status bar.
func NewStatusBar(styles ui.Styles) *StatusBar {
	return &StatusBar{
		styles: styles,
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

// SetPopupHints sets hints to show when a popup is visible.
func (s *StatusBar) SetPopupHints(hints string) {
	s.popupHints = hints
}

// ClearPopupHints clears popup hints, returning to normal hints.
func (s *StatusBar) ClearPopupHints() {
	s.popupHints = ""
}

// SetBindings sets the current panel's bindings for display.
func (s *StatusBar) SetBindings(bindings []ui.Binding) {
	s.panelBindings = bindings
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
	// Account for StatusBar style padding (1 on each side = 2 total)
	const stylePadding = 2
	contentWidth := s.width - stylePadding
	if contentWidth < 10 {
		contentWidth = 10
	}

	// Build activity indicators first to know their width
	indicators := s.buildIndicators()
	indicatorWidth := lipgloss.Width(indicators)

	// Calculate available width for hints
	// Reserve: indicatorWidth + 1 space before indicators
	availableForHints := contentWidth - indicatorWidth - 1
	if availableForHints < 10 {
		availableForHints = 10
	}

	// Build left content, constrained to available width
	var leftContent string
	if s.message != "" {
		msg := s.message
		if s.isError {
			msg = s.styles.Error.Render(msg)
		}
		if lipgloss.Width(msg) > availableForHints {
			leftContent = truncateToWidth(s.message, availableForHints)
			if s.isError {
				leftContent = s.styles.Error.Render(leftContent)
			}
		} else {
			leftContent = msg
		}
	} else {
		leftContent = s.buildHints(availableForHints)
	}

	// Build the full line: hints + padding + indicators = exactly contentWidth
	leftWidth := lipgloss.Width(leftContent)
	padding := contentWidth - leftWidth - indicatorWidth
	if padding < 1 {
		padding = 1
	}

	full := leftContent + strings.Repeat(" ", padding) + indicators
	return s.styles.StatusBar.Render(full)
}

// truncateToWidth truncates a string to fit within maxWidth visual columns.
func truncateToWidth(s string, maxWidth int) string {
	if maxWidth <= 3 {
		return "..."
	}
	if lipgloss.Width(s) <= maxWidth {
		return s
	}
	// Simple truncation - remove characters until it fits (reserve 3 for "...")
	runes := []rune(s)
	for len(runes) > 0 && lipgloss.Width(string(runes))+3 > maxWidth {
		runes = runes[:len(runes)-1]
	}
	return string(runes) + "..."
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

func (s *StatusBar) buildHints(maxWidth int) string {
	// Show popup hints if set
	if s.popupHints != "" {
		if lipgloss.Width(s.popupHints) > maxWidth {
			return truncateToWidth(s.popupHints, maxWidth)
		}
		return s.popupHints
	}

	// Build hints from panel bindings (sorted by priority)
	var allHints []string

	// Copy and sort by priority (highest first)
	bindings := make([]ui.Binding, len(s.panelBindings))
	copy(bindings, s.panelBindings)
	sort.Slice(bindings, func(i, j int) bool {
		return bindings[i].Priority > bindings[j].Priority
	})

	// Add panel-specific hints (only those with priority > 0)
	for _, b := range bindings {
		if b.Priority > 0 {
			allHints = append(allHints, b.Desc+": "+b.Display)
		}
	}

	// Add global hints
	allHints = append(allHints, "Navigate: ↑/↓", "Help: ?", "Quit: q")

	// Join hints, but truncate if too long
	separator := " | "
	var result strings.Builder
	for i, hint := range allHints {
		if i > 0 {
			// Check if adding separator + next hint would exceed width
			nextPart := separator + hint
			if lipgloss.Width(result.String()+nextPart) > maxWidth {
				break // Stop adding hints
			}
			result.WriteString(separator)
		} else {
			if lipgloss.Width(hint) > maxWidth {
				return truncateToWidth(hint, maxWidth)
			}
		}
		result.WriteString(hint)
	}

	return result.String()
}
