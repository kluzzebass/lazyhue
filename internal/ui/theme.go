package ui

import "github.com/charmbracelet/lipgloss"

// Theme defines the color scheme and styles for the UI.
type Theme struct {
	// Base colors
	Primary    lipgloss.Color
	Secondary  lipgloss.Color
	Accent     lipgloss.Color
	Background lipgloss.Color
	Foreground lipgloss.Color
	Muted      lipgloss.Color
	Error      lipgloss.Color
	Success    lipgloss.Color
	Warning    lipgloss.Color

	// Status indicators
	OnColor  lipgloss.Color
	OffColor lipgloss.Color
}

// DefaultTheme returns the default color theme.
func DefaultTheme() Theme {
	return Theme{
		Primary:    lipgloss.Color("#7C3AED"), // violet
		Secondary:  lipgloss.Color("#3B82F6"), // blue
		Accent:     lipgloss.Color("#F59E0B"), // amber
		Background: lipgloss.Color("#1F2937"), // gray-800
		Foreground: lipgloss.Color("#F9FAFB"), // gray-50
		Muted:      lipgloss.Color("#6B7280"), // gray-500
		Error:      lipgloss.Color("#EF4444"), // red
		Success:    lipgloss.Color("#10B981"), // emerald
		Warning:    lipgloss.Color("#F59E0B"), // amber

		OnColor:  lipgloss.Color("#FCD34D"), // yellow-300 (light on)
		OffColor: lipgloss.Color("#6B7280"), // gray-500 (light off)
	}
}

// Styles holds pre-built lipgloss styles.
type Styles struct {
	Theme Theme

	// Layout
	App         lipgloss.Style
	Header      lipgloss.Style
	StatusBar   lipgloss.Style
	LeftPanel   lipgloss.Style
	RightPanel  lipgloss.Style
	PanelTitle  lipgloss.Style
	ActivePanel lipgloss.Style

	// List items
	ListItem      lipgloss.Style
	SelectedItem  lipgloss.Style
	ListItemTitle lipgloss.Style
	ListItemDesc  lipgloss.Style

	// Status indicators
	OnIndicator  lipgloss.Style
	OffIndicator lipgloss.Style
	Connected    lipgloss.Style
	Disconnected lipgloss.Style

	// Text
	Title    lipgloss.Style
	Subtitle lipgloss.Style
	Muted    lipgloss.Style
	Error    lipgloss.Style
	Success  lipgloss.Style
	KeyHelp  lipgloss.Style
}

// DefaultStyles returns the default UI styles.
func DefaultStyles() Styles {
	theme := DefaultTheme()

	return Styles{
		Theme: theme,

		App: lipgloss.NewStyle().
			Background(theme.Background),

		Header: lipgloss.NewStyle().
			Bold(true).
			Foreground(theme.Foreground).
			Background(theme.Primary).
			Padding(0, 1),

		StatusBar: lipgloss.NewStyle().
			Foreground(theme.Muted).
			Padding(0, 1),

		LeftPanel: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(theme.Muted).
			Padding(0, 1),

		RightPanel: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(theme.Muted).
			Padding(0, 1),

		ActivePanel: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(theme.Primary).
			Padding(0, 1),

		PanelTitle: lipgloss.NewStyle().
			Bold(true).
			Foreground(theme.Primary),

		ListItem: lipgloss.NewStyle().
			Foreground(theme.Foreground),

		SelectedItem: lipgloss.NewStyle().
			Bold(true).
			Foreground(theme.Foreground).
			Background(theme.Primary),

		ListItemTitle: lipgloss.NewStyle().
			Foreground(theme.Foreground),

		ListItemDesc: lipgloss.NewStyle().
			Foreground(theme.Muted),

		OnIndicator: lipgloss.NewStyle().
			Foreground(theme.OnColor).
			SetString("●"),

		OffIndicator: lipgloss.NewStyle().
			Foreground(theme.OffColor).
			SetString("○"),

		Connected: lipgloss.NewStyle().
			Foreground(theme.Success),

		Disconnected: lipgloss.NewStyle().
			Foreground(theme.Error),

		Title: lipgloss.NewStyle().
			Bold(true).
			Foreground(theme.Foreground),

		Subtitle: lipgloss.NewStyle().
			Foreground(theme.Secondary),

		Muted: lipgloss.NewStyle().
			Foreground(theme.Muted),

		Error: lipgloss.NewStyle().
			Foreground(theme.Error),

		Success: lipgloss.NewStyle().
			Foreground(theme.Success),

		KeyHelp: lipgloss.NewStyle().
			Foreground(theme.Muted),
	}
}

// OnOffIndicator returns the appropriate indicator for a light state.
func (s Styles) OnOffIndicator(on bool) string {
	if on {
		return s.OnIndicator.String()
	}
	return s.OffIndicator.String()
}
