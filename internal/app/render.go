package app

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// View renders the UI.
func (m Model) View() string {
	if !m.ready {
		return "Initializing..."
	}

	if m.quitting {
		return "Goodbye!\n"
	}

	// Render left column panels in layout order
	leftPanelIDs := []string{PanelIDBridges, PanelIDHierarchy}
	panelViews := make([]string, 0, len(leftPanelIDs))
	for _, id := range leftPanelIDs {
		if panel := m.panelMap[id]; panel != nil {
			isActive := m.focusedPanelID() == id
			panelViews = append(panelViews, panel.View(isActive))
		}
	}
	leftColumn := lipgloss.JoinVertical(lipgloss.Left, panelViews...)

	// Right column: detail panel + log panel (if visible)
	detailView := m.detailPanel.View(m.focusedOnDetail())
	var rightColumn string
	if m.logPanelVisible {
		logView := m.logPanel.View(m.focusedOnLog())
		rightColumn = lipgloss.JoinVertical(lipgloss.Left, detailView, logView)
	} else {
		rightColumn = detailView
	}

	// Combine columns - constrain to actual dimensions
	content := lipgloss.JoinHorizontal(lipgloss.Top, leftColumn, rightColumn)

	// Status bar at bottom
	status := m.statusBar.View()

	// Combine and constrain to terminal size
	full := lipgloss.JoinVertical(lipgloss.Left, content, status)

	// Place base UI
	result := lipgloss.Place(m.width, m.height, lipgloss.Left, lipgloss.Top, full)

	// Overlay help panel if visible
	if m.helpPanel.IsVisible() {
		result = m.overlayHelp(result)
	}

	// Overlay pairing panel if visible
	if m.pairingPanel.IsVisible() {
		result = m.overlayPairing(result)
	}

	// Overlay popup panel if visible (highest priority)
	if m.popupPanel.IsVisible() {
		result = m.overlayPopup(result)
	}

	return result
}

// overlayPairing composites pairing panel on top of base UI.
func (m *Model) overlayPairing(base string) string {
	pairingView := m.pairingPanel.View()
	pairingWidth := m.pairingPanel.Width()
	pairingHeight := m.pairingPanel.Height()

	// Calculate centered position
	startX := (m.width - pairingWidth) / 2
	startY := (m.height - pairingHeight) / 2
	if startX < 0 {
		startX = 0
	}
	if startY < 0 {
		startY = 0
	}

	baseLines := strings.Split(base, "\n")
	pairingLines := strings.Split(pairingView, "\n")

	// Ensure we have enough base lines
	for len(baseLines) < m.height {
		baseLines = append(baseLines, "")
	}

	// Overlay each pairing line onto the corresponding base line
	for i, pairingLine := range pairingLines {
		baseY := startY + i
		if baseY >= len(baseLines) {
			break
		}

		baseLine := baseLines[baseY]
		baseVisualWidth := lipgloss.Width(baseLine)

		var result strings.Builder

		// Left portion
		if startX > 0 {
			if baseVisualWidth > 0 {
				left := ansiTruncate(baseLine, startX)
				result.WriteString(left)
				leftWidth := lipgloss.Width(left)
				for j := leftWidth; j < startX; j++ {
					result.WriteByte(' ')
				}
			} else {
				for j := 0; j < startX; j++ {
					result.WriteByte(' ')
				}
			}
		}

		// Pairing line
		result.WriteString(pairingLine)

		// Right portion
		pairingEnd := startX + lipgloss.Width(pairingLine)
		if pairingEnd < baseVisualWidth {
			right := ansiSubstring(baseLine, pairingEnd)
			result.WriteString(right)
		}

		baseLines[baseY] = result.String()
	}

	return strings.Join(baseLines, "\n")
}

// overlayHelp composites help panel on top of base UI.
func (m *Model) overlayHelp(base string) string {
	helpView := m.helpPanel.View()
	helpWidth := m.helpPanel.Width()
	helpHeight := m.helpPanel.Height()

	// Calculate centered position
	startX := (m.width - helpWidth) / 2
	startY := (m.height - helpHeight) / 2
	if startX < 0 {
		startX = 0
	}
	if startY < 0 {
		startY = 0
	}

	baseLines := strings.Split(base, "\n")
	helpLines := strings.Split(helpView, "\n")

	// Ensure we have enough base lines
	for len(baseLines) < m.height {
		baseLines = append(baseLines, "")
	}

	// Overlay each help line onto the corresponding base line
	for i, helpLine := range helpLines {
		baseY := startY + i
		if baseY >= len(baseLines) {
			break
		}

		// Get base line, pad if needed
		baseLine := baseLines[baseY]
		baseVisualWidth := lipgloss.Width(baseLine)

		// Build: left of base + help line + right of base
		var result strings.Builder

		// Left portion: take visual columns 0 to startX from base
		if startX > 0 {
			if baseVisualWidth > 0 {
				left := ansiTruncate(baseLine, startX)
				result.WriteString(left)
				// Pad if left is shorter than startX
				leftWidth := lipgloss.Width(left)
				for j := leftWidth; j < startX; j++ {
					result.WriteByte(' ')
				}
			} else {
				for j := 0; j < startX; j++ {
					result.WriteByte(' ')
				}
			}
		}

		// Help line (already has its own styling)
		result.WriteString(helpLine)

		// Right portion: take visual columns after help ends
		helpEnd := startX + lipgloss.Width(helpLine)
		if helpEnd < baseVisualWidth {
			right := ansiSubstring(baseLine, helpEnd)
			result.WriteString(right)
		}

		baseLines[baseY] = result.String()
	}

	return strings.Join(baseLines, "\n")
}

// ansiTruncate returns the first n visual columns of s, preserving ANSI codes.
func ansiTruncate(s string, n int) string {
	if n <= 0 {
		return ""
	}

	var result strings.Builder
	col := 0
	i := 0
	runes := []rune(s)

	for i < len(runes) && col < n {
		if runes[i] == '\x1b' {
			// Consume entire escape sequence
			result.WriteRune(runes[i])
			i++
			for i < len(runes) && !isAnsiTerminator(runes[i]) {
				result.WriteRune(runes[i])
				i++
			}
			if i < len(runes) {
				result.WriteRune(runes[i])
				i++
			}
		} else {
			result.WriteRune(runes[i])
			col++
			i++
		}
	}

	return result.String()
}

// ansiSubstring returns s starting from visual column n.
func ansiSubstring(s string, n int) string {
	if n <= 0 {
		return s
	}

	col := 0
	i := 0
	runes := []rune(s)

	// Skip to column n
	for i < len(runes) && col < n {
		if runes[i] == '\x1b' {
			// Skip entire escape sequence
			i++
			for i < len(runes) && !isAnsiTerminator(runes[i]) {
				i++
			}
			if i < len(runes) {
				i++
			}
		} else {
			col++
			i++
		}
	}

	if i >= len(runes) {
		return ""
	}

	return string(runes[i:])
}

func isAnsiTerminator(r rune) bool {
	return (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z')
}

// overlayPopup composites popup panel on top of base UI.
func (m *Model) overlayPopup(base string) string {
	popupView := m.popupPanel.View()
	popupWidth := m.popupPanel.Width()
	popupHeight := m.popupPanel.Height()

	// Calculate centered position
	startX := (m.width - popupWidth) / 2
	startY := (m.height - popupHeight) / 2
	if startX < 0 {
		startX = 0
	}
	if startY < 0 {
		startY = 0
	}

	baseLines := strings.Split(base, "\n")
	popupLines := strings.Split(popupView, "\n")

	// Ensure we have enough base lines
	for len(baseLines) < m.height {
		baseLines = append(baseLines, "")
	}

	// Overlay each popup line onto the corresponding base line
	for i, popupLine := range popupLines {
		baseY := startY + i
		if baseY >= len(baseLines) {
			break
		}

		baseLine := baseLines[baseY]
		baseVisualWidth := lipgloss.Width(baseLine)

		var result strings.Builder

		// Left portion
		if startX > 0 {
			if baseVisualWidth > 0 {
				left := ansiTruncate(baseLine, startX)
				result.WriteString(left)
				leftWidth := lipgloss.Width(left)
				for j := leftWidth; j < startX; j++ {
					result.WriteByte(' ')
				}
			} else {
				for j := 0; j < startX; j++ {
					result.WriteByte(' ')
				}
			}
		}

		// Popup line
		result.WriteString(popupLine)

		// Right portion
		popupEnd := startX + lipgloss.Width(popupLine)
		if popupEnd < baseVisualWidth {
			right := ansiSubstring(baseLine, popupEnd)
			result.WriteString(right)
		}

		baseLines[baseY] = result.String()
	}

	return strings.Join(baseLines, "\n")
}


