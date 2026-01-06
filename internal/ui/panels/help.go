// Package panels provides UI panel components.
package panels

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/kluzzebass/lazyhue/internal/ui"
)

// HelpPanel displays keybindings using the generic PopupPanel.
type HelpPanel struct {
	popup          *PopupPanel
	styles         ui.Styles
	panelBindings  []ui.Binding
	globalBindings []ui.Binding
	panelTitle     string
}

// NewHelpPanel creates a new help panel.
func NewHelpPanel(styles ui.Styles) *HelpPanel {
	popup := NewPopupPanel(styles)
	popup.SetRatio(0.5, 0.5)

	return &HelpPanel{
		popup:  popup,
		styles: styles,
	}
}

// SetPanelBindings sets the focused panel's bindings and title for contextual help.
func (h *HelpPanel) SetPanelBindings(title string, bindings []ui.Binding) {
	h.panelTitle = title
	h.panelBindings = bindings
}

// SetGlobalBindings sets the global bindings to display.
func (h *HelpPanel) SetGlobalBindings(bindings []ui.Binding) {
	h.globalBindings = bindings
}

// Toggle toggles the help panel visibility.
func (h *HelpPanel) Toggle() {
	if h.popup.IsVisible() {
		h.popup.Hide()
	} else {
		content := h.buildContent()
		h.popup.ShowDisplay("Keybindings", content, nil)
	}
}

// Hide hides the help panel.
func (h *HelpPanel) Hide() {
	h.popup.Hide()
}

// IsVisible returns whether the help panel is visible.
func (h *HelpPanel) IsVisible() bool {
	return h.popup.IsVisible()
}

// SetSize updates the screen dimensions.
func (h *HelpPanel) SetSize(width, height int) {
	h.popup.SetSize(width, height)
}

// Update handles input for the help panel.
func (h *HelpPanel) Update(msg tea.Msg) {
	h.popup.Update(msg)
}

// View renders the help panel.
func (h *HelpPanel) View() string {
	return h.popup.View()
}

// Width returns the width of the help panel.
func (h *HelpPanel) Width() int {
	return h.popup.Width()
}

// Height returns the height of the help panel.
func (h *HelpPanel) Height() int {
	return h.popup.Height()
}

func (h *HelpPanel) buildContent() string {
	keyStyle := lipgloss.NewStyle().Foreground(h.styles.Theme.Accent)
	sepStyle := lipgloss.NewStyle().Foreground(h.styles.Theme.Muted)

	// Convert panel bindings to display format
	var contextBindings [][2]string
	for _, b := range h.panelBindings {
		contextBindings = append(contextBindings, [2]string{b.Display, b.Desc})
	}

	// Convert global bindings to display format
	var globalBindings [][2]string
	for _, b := range h.globalBindings {
		globalBindings = append(globalBindings, [2]string{b.Display, b.Desc})
	}

	// Add panel focus keys
	globalBindings = append(globalBindings,
		[2]string{"", ""},
		[2]string{"0-5", "Focus panel"},
	)

	// Calculate max key width across ALL bindings
	maxKeyWidth := 0
	for _, b := range contextBindings {
		if w := lipgloss.Width(b[0]); w > maxKeyWidth {
			maxKeyWidth = w
		}
	}
	for _, b := range globalBindings {
		if w := lipgloss.Width(b[0]); w > maxKeyWidth {
			maxKeyWidth = w
		}
	}

	var sb strings.Builder

	// Header padding to align with description column (key width + 1 space)
	headerPad := strings.Repeat(" ", maxKeyWidth+1)

	// Contextual section (from focused panel)
	if len(contextBindings) > 0 && h.panelTitle != "" {
		sb.WriteString(headerPad + sepStyle.Render("─── "+h.panelTitle+" ───"))
		sb.WriteString("\n")
		for _, b := range contextBindings {
			sb.WriteString(h.formatBinding(b[0], b[1], maxKeyWidth, keyStyle))
			sb.WriteString("\n")
		}
		sb.WriteString("\n")
	}

	// Global section
	sb.WriteString(headerPad + sepStyle.Render("─── Global ───"))
	sb.WriteString("\n")

	for _, b := range globalBindings {
		if b[0] == "" {
			sb.WriteString("\n")
		} else {
			sb.WriteString(h.formatBinding(b[0], b[1], maxKeyWidth, keyStyle))
			sb.WriteString("\n")
		}
	}

	return strings.TrimSuffix(sb.String(), "\n")
}

func (h *HelpPanel) formatBinding(key, desc string, keyWidth int, keyStyle lipgloss.Style) string {
	// Right-pad the key to keyWidth, then right-align within that space
	keyLen := lipgloss.Width(key)
	padding := keyWidth - keyLen
	paddedKey := strings.Repeat(" ", padding) + key
	return keyStyle.Render(paddedKey) + " " + desc
}
