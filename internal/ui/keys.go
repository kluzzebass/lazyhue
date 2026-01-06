// Package ui provides terminal UI components and styling.
package ui

import "github.com/charmbracelet/bubbles/key"

// KeyMap defines all keybindings for the application.
type KeyMap struct {
	// Navigation
	Up       key.Binding
	Down     key.Binding
	Left     key.Binding
	Right    key.Binding
	Top      key.Binding
	Bottom   key.Binding
	NextPane key.Binding
	PrevPane key.Binding

	// Panel focus (number keys)
	FocusDetail  key.Binding
	FocusBridges key.Binding
	FocusGroups  key.Binding
	FocusLights  key.Binding
	FocusDevices key.Binding
	FocusScenes  key.Binding

	// Selection
	Select key.Binding
	Back   key.Binding

	// Light/Group actions
	Toggle         key.Binding
	TurnOn         key.Binding
	TurnOff        key.Binding
	BrightnessUp   key.Binding
	BrightnessDown key.Binding
	ColorPicker    key.Binding
	TempPicker     key.Binding

	// Scene actions
	ScenePicker key.Binding

	// Bulk operations
	VisualMode key.Binding
	SelectAll  key.Binding

	// Bridge management
	BridgePicker key.Binding
	NextBridge   key.Binding
	PrevBridge   key.Binding
	Refresh      key.Binding
	PairBridge   key.Binding

	// System
	Help key.Binding
	Quit key.Binding
}

// DefaultKeyMap returns the default keybindings.
func DefaultKeyMap() KeyMap {
	return KeyMap{
		Up: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("↑/k", "up"),
		),
		Down: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("↓/j", "down"),
		),
		Left: key.NewBinding(
			key.WithKeys("left", "h"),
			key.WithHelp("←/h", "left/prev tab"),
		),
		Right: key.NewBinding(
			key.WithKeys("right", "l"),
			key.WithHelp("→/l", "right/next tab"),
		),
		Top: key.NewBinding(
			key.WithKeys("g"),
			key.WithHelp("g", "top"),
		),
		Bottom: key.NewBinding(
			key.WithKeys("G"),
			key.WithHelp("G", "bottom"),
		),
		NextPane: key.NewBinding(
			key.WithKeys("tab"),
			key.WithHelp("tab", "next pane"),
		),
		PrevPane: key.NewBinding(
			key.WithKeys("shift+tab"),
			key.WithHelp("shift+tab", "prev pane"),
		),
		FocusDetail: key.NewBinding(
			key.WithKeys("0"),
			key.WithHelp("0", "detail"),
		),
		FocusBridges: key.NewBinding(
			key.WithKeys("1"),
			key.WithHelp("1", "bridges"),
		),
		FocusScenes: key.NewBinding(
			key.WithKeys("2"),
			key.WithHelp("2", "scenes"),
		),
		FocusGroups: key.NewBinding(
			key.WithKeys("3"),
			key.WithHelp("3", "groups"),
		),
		FocusLights: key.NewBinding(
			key.WithKeys("4"),
			key.WithHelp("4", "lights"),
		),
		FocusDevices: key.NewBinding(
			key.WithKeys("5"),
			key.WithHelp("5", "devices"),
		),
		Select: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "select"),
		),
		Back: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "back"),
		),
		Toggle: key.NewBinding(
			key.WithKeys(" "),
			key.WithHelp("space", "toggle"),
		),
		TurnOn: key.NewBinding(
			key.WithKeys("o"),
			key.WithHelp("o", "turn on"),
		),
		TurnOff: key.NewBinding(
			key.WithKeys("O"),
			key.WithHelp("O", "turn off"),
		),
		BrightnessUp: key.NewBinding(
			key.WithKeys("+", "="),
			key.WithHelp("+", "brightness +10%"),
		),
		BrightnessDown: key.NewBinding(
			key.WithKeys("-"),
			key.WithHelp("-", "brightness -10%"),
		),
		ColorPicker: key.NewBinding(
			key.WithKeys("c"),
			key.WithHelp("c", "color"),
		),
		TempPicker: key.NewBinding(
			key.WithKeys("t"),
			key.WithHelp("t", "temperature"),
		),
		ScenePicker: key.NewBinding(
			key.WithKeys("s"),
			key.WithHelp("s", "scenes"),
		),
		VisualMode: key.NewBinding(
			key.WithKeys("v"),
			key.WithHelp("v", "visual mode"),
		),
		SelectAll: key.NewBinding(
			key.WithKeys("a"),
			key.WithHelp("a", "select all"),
		),
		BridgePicker: key.NewBinding(
			key.WithKeys("B"),
			key.WithHelp("B", "bridges"),
		),
		NextBridge: key.NewBinding(
			key.WithKeys("]"),
			key.WithHelp("]", "next bridge"),
		),
		PrevBridge: key.NewBinding(
			key.WithKeys("["),
			key.WithHelp("[", "prev bridge"),
		),
		Refresh: key.NewBinding(
			key.WithKeys("R"),
			key.WithHelp("R", "refresh"),
		),
		PairBridge: key.NewBinding(
			key.WithKeys("P"),
			key.WithHelp("P", "pair bridge"),
		),
		Help: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "help"),
		),
		Quit: key.NewBinding(
			key.WithKeys("q", "ctrl+c"),
			key.WithHelp("q", "quit"),
		),
	}
}

// ShortHelp returns keybindings for the mini help view.
func (k KeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Up, k.Down, k.Toggle, k.ScenePicker, k.Help, k.Quit}
}

// FullHelp returns keybindings for the expanded help view.
func (k KeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down, k.Left, k.Right, k.Top, k.Bottom},
		{k.NextPane, k.PrevPane, k.Select, k.Back},
		{k.FocusDetail, k.FocusBridges, k.FocusGroups, k.FocusLights, k.FocusDevices, k.FocusScenes},
		{k.Toggle, k.TurnOn, k.TurnOff, k.BrightnessUp, k.BrightnessDown},
		{k.ColorPicker, k.TempPicker, k.ScenePicker},
		{k.VisualMode, k.SelectAll},
		{k.NextBridge, k.PrevBridge, k.Refresh, k.PairBridge},
		{k.Help, k.Quit},
	}
}
