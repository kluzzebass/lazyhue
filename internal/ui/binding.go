// Package ui provides terminal UI components and styling.
package ui

// Action identifies a keybinding action.
type Action string

const (
	ActionNone Action = ""

	// Navigation
	ActionUp       Action = "up"
	ActionDown     Action = "down"
	ActionTop      Action = "top"
	ActionBottom   Action = "bottom"
	ActionPageUp   Action = "pageup"
	ActionPageDown Action = "pagedown"

	// Panel focus
	ActionNextPanel   Action = "next_panel"
	ActionPrevPanel   Action = "prev_panel"
	ActionFocusDetail Action = "focus_detail"

	// Bridge
	ActionNextBridge   Action = "next_bridge"
	ActionPrevBridge   Action = "prev_bridge"
	ActionForgetBridge Action = "forget_bridge"

	// Entity actions
	ActionSelect         Action = "select"
	ActionToggle         Action = "toggle"
	ActionTurnOn         Action = "turn_on"
	ActionTurnOff        Action = "turn_off"
	ActionBrightnessUp   Action = "brightness_up"
	ActionBrightnessDown Action = "brightness_down"
	ActionExpandCollapse Action = "expand_collapse"
	ActionExpand         Action = "expand"
	ActionCollapse       Action = "collapse"

	// Tabs
	ActionNextTab Action = "next_tab"
	ActionPrevTab Action = "prev_tab"

	// Global
	ActionHelp      Action = "help"
	ActionQuit      Action = "quit"
	ActionBack      Action = "back"
	ActionToggleLog Action = "toggle_log"
)

// Binding represents a keybinding with its associated action.
// Action is dispatched by the app - no closures needed.
type Binding struct {
	Keys     []string // Keys that trigger this binding (e.g., []string{" "})
	Display  string   // Display string for help (e.g., "<space>")
	Desc     string   // Description (e.g., "Toggle light")
	Action   Action   // The action to dispatch
	Priority int      // Higher = shown first in status bar (0 = don't show)
}

// Matches returns true if the given key string matches this binding.
func (b Binding) Matches(key string) bool {
	for _, k := range b.Keys {
		if k == key {
			return true
		}
	}
	return false
}
