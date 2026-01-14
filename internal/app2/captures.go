package app2

import tea "github.com/charmbracelet/bubbletea/v2"

// RenameCapture handles rename mode input.
// When active, it captures all input and routes it to the rename text field.
type RenameCapture struct {
	m *Model // Reference to main model for state access
}

// NewRenameCapture creates a new rename capture.
func NewRenameCapture(m *Model) *RenameCapture {
	return &RenameCapture{m: m}
}

// Name returns the debug name for this capture.
func (r *RenameCapture) Name() string {
	return "rename"
}

// IsActive returns whether rename mode is currently active.
func (r *RenameCapture) IsActive() bool {
	return r.m.renaming
}

// HandleEvent processes events when rename mode is active.
// NOTE: This is legacy code - rename input is now handled via TextInputModal
// in the component tree. This capture is kept for compatibility but
// should be removed in Phase 5 cleanup.
func (r *RenameCapture) HandleEvent(msg tea.Msg) (bool, tea.Cmd) {
	// Legacy: rename is now handled by renameModal (TextInputModal)
	// This function should not be called if the modal is active
	if !r.m.renaming {
		return false, nil
	}

	// Delegate to the modal's RouteEvent - it will return messages
	// that the main Update() handles
	return r.m.renameModal.RouteEvent(msg)
}
