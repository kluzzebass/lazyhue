package panels

import (
	tea "github.com/charmbracelet/bubbletea"
)

// Panel is the interface that all left-side panels implement.
type Panel interface {
	// View renders the panel. active indicates if this panel has focus.
	View(active bool) string

	// Update handles messages for this panel.
	Update(msg tea.Msg) tea.Cmd

	// SetSize sets the panel dimensions.
	SetSize(width, height int)

	// Key returns the keyboard shortcut key for this panel (e.g., "1", "2").
	Key() string

	// Title returns the panel title.
	Title() string

	// SelectedEntity returns the currently selected entity, if any.
	SelectedEntity() (*EntityItem, bool)
}

// PanelConfig defines a panel's position and identity.
type PanelConfig struct {
	Key   string // Keyboard shortcut (e.g., "1", "2")
	Title string
}

