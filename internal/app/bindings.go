package app

import (
	"github.com/kluzzebass/lazyhue/internal/ui/panels"
)

// Binding represents a keybinding with display and help information.
type Binding struct {
	Keys     []string // Key strings that trigger this binding
	Display  string   // How to display in help (e.g., "↑/k")
	Desc     string   // Description for help
	Priority int      // For sorting in help (higher = shown first)
}

// initBindings sets up all keybindings with their help text.
func (m *Model) initBindings() {
	// Panel-specific bindings
	m.panelBindings = map[string][]Binding{
		PanelTree: {
			{Keys: []string{"enter"}, Display: "Enter", Desc: "Expand/collapse or select", Priority: 100},
			{Keys: []string{" "}, Display: "Space", Desc: "Toggle on/off", Priority: 95},
			{Keys: []string{"r"}, Display: "r", Desc: "Rename selected", Priority: 92},
			{Keys: []string{"n"}, Display: "n", Desc: "New room", Priority: 91},
			{Keys: []string{"N"}, Display: "N", Desc: "New zone", Priority: 91},
			{Keys: []string{"x"}, Display: "x", Desc: "Delete selected", Priority: 90},
			{Keys: []string{"left", "h"}, Display: "←/h", Desc: "Collapse / prev tab", Priority: 85},
			{Keys: []string{"right", "l"}, Display: "→/l", Desc: "Expand / next tab", Priority: 85},
			{Keys: []string{"o"}, Display: "o", Desc: "Turn on", Priority: 80},
			{Keys: []string{"O"}, Display: "O", Desc: "Turn off", Priority: 80},
			{Keys: []string{"+", "="}, Display: "+", Desc: "Brighter", Priority: 70},
			{Keys: []string{"-"}, Display: "-", Desc: "Dimmer", Priority: 70},
		},
		PanelDetail: {
			{Keys: []string{"esc"}, Display: "Esc", Desc: "Back / cancel", Priority: 100},
		},
		PanelLog: {},
	}

	// Global bindings - work in any panel
	m.globalBindings = []Binding{
		{Keys: []string{"up", "k"}, Display: "↑/k", Desc: "Move up", Priority: 100},
		{Keys: []string{"down", "j"}, Display: "↓/j", Desc: "Move down", Priority: 99},
		{Keys: []string{"g", "home"}, Display: "g/Home", Desc: "Go to top", Priority: 95},
		{Keys: []string{"G", "end"}, Display: "G/End", Desc: "Go to bottom", Priority: 94},
		{Keys: []string{"pgup"}, Display: "PgUp", Desc: "Page up", Priority: 90},
		{Keys: []string{"pgdown"}, Display: "PgDn", Desc: "Page down", Priority: 89},
		{Keys: []string{"tab"}, Display: "Tab", Desc: "Next panel", Priority: 50},
		{Keys: []string{"shift+tab"}, Display: "S-Tab", Desc: "Prev panel", Priority: 49},
		{Keys: []string{"0", "1", "2"}, Display: "0/1/2", Desc: "Jump to panel", Priority: 48},
		{Keys: []string{"]"}, Display: "]", Desc: "Next tab", Priority: 40},
		{Keys: []string{"["}, Display: "[", Desc: "Prev tab", Priority: 39},
		{Keys: []string{"p"}, Display: "p", Desc: "Pair new bridge", Priority: 30},
		{Keys: []string{"?"}, Display: "?", Desc: "Toggle help", Priority: 20},
		{Keys: []string{"t"}, Display: "t", Desc: "Test form (debug)", Priority: 10},
		{Keys: []string{"q", "ctrl+c"}, Display: "q", Desc: "Quit", Priority: 5},
	}
}

// getContextBindings returns the bindings for the current context.
func (m *Model) getContextBindings() (panelBindings []Binding, globalBindings []Binding, panelTitle string) {
	// Get panel-specific bindings based on current focus
	var entityType panels.EntityType
	if node := m.tree.SelectedNode(); node != nil && node.Item != nil {
		entityType = node.Item.Type
	}

	// Set panel title based on focused panel
	switch m.focusedPane {
	case PanelTree:
		panelTitle = "Tree"
		// Start with base tree bindings
		panelBindings = make([]Binding, 0, len(m.panelBindings[PanelTree]))

		// Add context-specific hints and filter bindings based on selected entity
		switch entityType {
		case panels.EntityBridge:
			panelTitle = "Tree (Bridge selected)"
			// Include all bindings - 'x' deletes bridges
			panelBindings = append(panelBindings, m.panelBindings[PanelTree]...)
		case panels.EntityRoom:
			panelTitle = "Tree (Room selected)"
			// Include all bindings - 'x' deletes rooms
			panelBindings = append(panelBindings, m.panelBindings[PanelTree]...)
		case panels.EntityZone:
			panelTitle = "Tree (Zone selected)"
			// Include all bindings - 'x' deletes zones
			panelBindings = append(panelBindings, m.panelBindings[PanelTree]...)
		default:
			// For other entities (lights, scenes, devices), exclude 'x' deletion binding
			for _, b := range m.panelBindings[PanelTree] {
				isDeleteBinding := false
				for _, key := range b.Keys {
					if key == "x" {
						isDeleteBinding = true
						break
					}
				}
				if !isDeleteBinding {
					panelBindings = append(panelBindings, b)
				}
			}

			// Set appropriate panel title
			switch entityType {
			case panels.EntityLight:
				panelTitle = "Tree (Light selected)"
			case panels.EntityScene:
				panelTitle = "Tree (Scene selected)"
			case panels.EntityDevice:
				panelTitle = "Tree (Device selected)"
			}
		}
	case PanelDetail:
		if m.confirmingDelete {
			panelTitle = "Delete Confirmation"
			panelBindings = []Binding{
				{Display: "Y", Desc: "Confirm deletion"},
				{Display: "N/Esc", Desc: "Cancel"},
			}
		} else if m.confirmingDeleteEntity {
			panelTitle = "Delete Confirmation"
			panelBindings = []Binding{
				{Display: "Y", Desc: "Confirm deletion"},
				{Display: "N/Esc", Desc: "Cancel"},
			}
		} else if m.renaming {
			panelTitle = "Rename"
			panelBindings = []Binding{
				{Display: "Enter", Desc: "Save"},
				{Display: "Esc", Desc: "Cancel"},
			}
		} else if m.creatingRoom {
			panelTitle = "Create Room"
			panelBindings = []Binding{
				{Display: "↑/↓", Desc: "Navigate fields"},
				{Display: "Enter", Desc: "Submit"},
				{Display: "Esc", Desc: "Cancel"},
			}
		} else if m.creatingZone {
			panelTitle = "Create Zone"
			panelBindings = []Binding{
				{Display: "↑/↓", Desc: "Navigate fields"},
				{Display: "Enter", Desc: "Submit"},
				{Display: "Esc", Desc: "Cancel"},
			}
		} else {
			panelTitle = "Detail"
			panelBindings = m.panelBindings[PanelDetail]
		}
	case PanelLog:
		panelTitle = "Activity Log"
		panelBindings = m.panelBindings[PanelLog]
	}

	globalBindings = m.globalBindings
	return
}
