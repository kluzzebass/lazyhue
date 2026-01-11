package ui2

import (
	"github.com/charmbracelet/bubbles/v2/key"
)

// KeyMap defines all key bindings for the application.
// Using key.Binding provides structured bindings with automatic help text generation.
type KeyMap struct {
	// Navigation
	Up       key.Binding
	Down     key.Binding
	Left     key.Binding
	Right    key.Binding
	PageUp   key.Binding
	PageDown key.Binding
	Home     key.Binding
	End      key.Binding

	// Selection
	Enter  key.Binding
	Space  key.Binding
	Escape key.Binding

	// Actions
	Edit      key.Binding
	Delete    key.Binding
	Create    key.Binding
	Refresh   key.Binding
	Toggle    key.Binding
	BrightUp  key.Binding
	BrightDn  key.Binding
	TurnOn    key.Binding
	TurnOff   key.Binding
	NextPanel key.Binding
	PrevPanel key.Binding

	// Bridge navigation
	NextBridge key.Binding
	PrevBridge key.Binding

	// UI toggles
	Help      key.Binding
	ToggleLog key.Binding

	// Quit
	Quit key.Binding
}

// DefaultKeyMap returns the default key bindings.
func DefaultKeyMap() KeyMap {
	return KeyMap{
		// Navigation
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
			key.WithHelp("←/h", "left"),
		),
		Right: key.NewBinding(
			key.WithKeys("right", "l"),
			key.WithHelp("→/l", "right"),
		),
		PageUp: key.NewBinding(
			key.WithKeys("pgup"),
			key.WithHelp("PgUp", "page up"),
		),
		PageDown: key.NewBinding(
			key.WithKeys("pgdown"),
			key.WithHelp("PgDn", "page down"),
		),
		Home: key.NewBinding(
			key.WithKeys("home", "g"),
			key.WithHelp("g/Home", "top"),
		),
		End: key.NewBinding(
			key.WithKeys("end", "G"),
			key.WithHelp("G/End", "bottom"),
		),

		// Selection
		Enter: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("Enter", "select"),
		),
		Space: key.NewBinding(
			key.WithKeys(" "),
			key.WithHelp("Space", "toggle"),
		),
		Escape: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("Esc", "back"),
		),

		// Actions
		Edit: key.NewBinding(
			key.WithKeys("e"),
			key.WithHelp("e", "edit"),
		),
		Delete: key.NewBinding(
			key.WithKeys("d", "delete"),
			key.WithHelp("d", "delete"),
		),
		Create: key.NewBinding(
			key.WithKeys("n"),
			key.WithHelp("n", "new"),
		),
		Refresh: key.NewBinding(
			key.WithKeys("r"),
			key.WithHelp("r", "refresh"),
		),
		Toggle: key.NewBinding(
			key.WithKeys(" "),
			key.WithHelp("Space", "toggle"),
		),
		BrightUp: key.NewBinding(
			key.WithKeys("+", "="),
			key.WithHelp("+", "brighter"),
		),
		BrightDn: key.NewBinding(
			key.WithKeys("-"),
			key.WithHelp("-", "dimmer"),
		),
		TurnOn: key.NewBinding(
			key.WithKeys("o"),
			key.WithHelp("o", "turn on"),
		),
		TurnOff: key.NewBinding(
			key.WithKeys("O"),
			key.WithHelp("O", "turn off"),
		),
		NextPanel: key.NewBinding(
			key.WithKeys("tab"),
			key.WithHelp("Tab", "next panel"),
		),
		PrevPanel: key.NewBinding(
			key.WithKeys("shift+tab"),
			key.WithHelp("S-Tab", "prev panel"),
		),

		// Bridge navigation
		NextBridge: key.NewBinding(
			key.WithKeys("]"),
			key.WithHelp("]", "next bridge"),
		),
		PrevBridge: key.NewBinding(
			key.WithKeys("["),
			key.WithHelp("[", "prev bridge"),
		),

		// UI toggles
		Help: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "help"),
		),
		ToggleLog: key.NewBinding(
			key.WithKeys("a"),
			key.WithHelp("a", "activity log"),
		),

		// Quit
		Quit: key.NewBinding(
			key.WithKeys("q", "ctrl+c"),
			key.WithHelp("q", "quit"),
		),
	}
}

// ShortHelp returns the key bindings for the short help view.
func (k KeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Help, k.Quit}
}

// FullHelp returns the key bindings for the full help view.
func (k KeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down, k.Left, k.Right},
		{k.Enter, k.Space, k.Escape},
		{k.Edit, k.Delete, k.Create},
		{k.NextPanel, k.PrevPanel},
		{k.Help, k.Quit},
	}
}
