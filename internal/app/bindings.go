package app

import (
	"github.com/kluzzebass/lazyhue/internal/ui"
)

// initBindings sets up all keybindings.
// Bindings use Action identifiers - handlers are in dispatch.go
func (m *Model) initBindings() {
	// Panel-specific bindings
	m.panelBindings = map[string][]ui.Binding{
		PanelIDBridges: {
			{Keys: []string{"enter"}, Display: "<enter>", Desc: "Connect", Action: ui.ActionSelect, Priority: 100},
		},
		PanelIDHierarchy: {
			{Keys: []string{"enter"}, Display: "<enter>", Desc: "Select/Activate", Action: ui.ActionSelect, Priority: 100},
			{Keys: []string{" "}, Display: "<space>", Desc: "Toggle light", Action: ui.ActionToggle, Priority: 95},
			{Keys: []string{"left", "h"}, Display: "←/h", Desc: "Collapse", Action: ui.ActionCollapse, Priority: 90},
			{Keys: []string{"right", "l"}, Display: "→/l", Desc: "Expand", Action: ui.ActionExpand, Priority: 90},
			{Keys: []string{"o"}, Display: "o", Desc: "Turn on", Action: ui.ActionTurnOn, Priority: 80},
			{Keys: []string{"O"}, Display: "O", Desc: "Turn off", Action: ui.ActionTurnOff, Priority: 80},
			{Keys: []string{"+", "="}, Display: "+", Desc: "Brighter", Action: ui.ActionBrightnessUp, Priority: 70},
			{Keys: []string{"-"}, Display: "-", Desc: "Dimmer", Action: ui.ActionBrightnessDown, Priority: 70},
		},
		PanelIDDetail: {
			{Keys: []string{"esc"}, Display: "<esc>", Desc: "Back", Action: ui.ActionBack, Priority: 100},
		},
	}

	// Global bindings - work in any context
	m.globalBindings = []ui.Binding{
		{Keys: []string{"up", "k"}, Display: "↑/k", Desc: "Up", Action: ui.ActionUp},
		{Keys: []string{"down", "j"}, Display: "↓/j", Desc: "Down", Action: ui.ActionDown},
		{Keys: []string{"g", "home"}, Display: "g/Home", Desc: "Top", Action: ui.ActionTop},
		{Keys: []string{"G", "end"}, Display: "G/End", Desc: "Bottom", Action: ui.ActionBottom},
		{Keys: []string{"pgup"}, Display: "PgUp", Desc: "Page up", Action: ui.ActionPageUp},
		{Keys: []string{"pgdown"}, Display: "PgDn", Desc: "Page down", Action: ui.ActionPageDown},
		{Keys: []string{"tab"}, Display: "Tab", Desc: "Next panel", Action: ui.ActionNextPanel},
		{Keys: []string{"shift+tab"}, Display: "S-Tab", Desc: "Prev panel", Action: ui.ActionPrevPanel},
		{Keys: []string{"]"}, Display: "]", Desc: "Next bridge", Action: ui.ActionNextBridge},
		{Keys: []string{"["}, Display: "[", Desc: "Prev bridge", Action: ui.ActionPrevBridge},
		{Keys: []string{"?"}, Display: "?", Desc: "Help", Action: ui.ActionHelp, Priority: 10},
		{Keys: []string{"q", "ctrl+c"}, Display: "q", Desc: "Quit", Action: ui.ActionQuit, Priority: 5},
	}
}
