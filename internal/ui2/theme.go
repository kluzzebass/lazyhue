// Package ui2 provides the v2 UI implementation using bubbletea v2, bubbles v2, and lipgloss v2.
package ui2

import (
	"image/color"

	"github.com/charmbracelet/lipgloss/v2"
)

// Theme defines the color palette for the application.
type Theme struct {
	// Primary colors
	Primary   color.Color
	Secondary color.Color
	Accent    color.Color

	// Text colors
	Text       color.Color
	TextMuted  color.Color
	TextBright color.Color

	// Background colors
	Background      color.Color
	BackgroundLight color.Color

	// Status colors
	Success color.Color
	Warning color.Color
	Error   color.Color

	// UI element colors
	Border       color.Color
	BorderActive color.Color
	Selection    color.Color
	Highlight    color.Color
}

// DefaultTheme returns the default dark theme.
func DefaultTheme() Theme {
	return Theme{
		// Primary colors - Philips Hue orange/amber
		Primary:   lipgloss.Color("#FF9500"),
		Secondary: lipgloss.Color("#FF6B00"),
		Accent:    lipgloss.Color("#FFB800"),

		// Text colors
		Text:       lipgloss.Color("#E0E0E0"),
		TextMuted:  lipgloss.Color("#808080"),
		TextBright: lipgloss.Color("#FFFFFF"),

		// Background colors
		Background:      lipgloss.Color("#1A1A1A"),
		BackgroundLight: lipgloss.Color("#2A2A2A"),

		// Status colors
		Success: lipgloss.Color("#00C853"),
		Warning: lipgloss.Color("#FFD600"),
		Error:   lipgloss.Color("#FF5252"),

		// UI element colors
		Border:       lipgloss.Color("#404040"),
		BorderActive: lipgloss.Color("#FF9500"),
		Selection:    lipgloss.Color("#3A3A3A"),
		Highlight:    lipgloss.Color("#FF9500"),
	}
}

// Styles contains pre-built lipgloss styles for common UI elements.
type Styles struct {
	Theme Theme

	// Base styles
	Base      lipgloss.Style
	Focused   lipgloss.Style
	Blurred   lipgloss.Style
	Selected  lipgloss.Style
	Dimmed    lipgloss.Style
	Highlight lipgloss.Style

	// Text styles
	Title       lipgloss.Style
	Subtitle    lipgloss.Style
	Label       lipgloss.Style
	Value       lipgloss.Style
	Placeholder lipgloss.Style

	// Panel styles
	Panel       lipgloss.Style
	PanelActive lipgloss.Style
	Header      lipgloss.Style
	StatusBar   lipgloss.Style

	// List styles
	ListItem         lipgloss.Style
	ListItemSelected lipgloss.Style
	ListItemDimmed   lipgloss.Style

	// Help styles
	HelpKey  lipgloss.Style
	HelpDesc lipgloss.Style

	// Status styles
	Success lipgloss.Style
	Warning lipgloss.Style
	Error   lipgloss.Style
}

// NewStyles creates a new Styles instance from the given theme.
func NewStyles(theme Theme) Styles {
	return Styles{
		Theme: theme,

		// Base styles
		Base:      lipgloss.NewStyle().Foreground(theme.Text),
		Focused:   lipgloss.NewStyle().Foreground(theme.TextBright),
		Blurred:   lipgloss.NewStyle().Foreground(theme.TextMuted),
		Selected:  lipgloss.NewStyle().Foreground(theme.Primary).Bold(true),
		Dimmed:    lipgloss.NewStyle().Foreground(theme.TextMuted),
		Highlight: lipgloss.NewStyle().Foreground(theme.Highlight).Bold(true),

		// Text styles
		Title:       lipgloss.NewStyle().Foreground(theme.TextBright).Bold(true),
		Subtitle:    lipgloss.NewStyle().Foreground(theme.TextMuted),
		Label:       lipgloss.NewStyle().Foreground(theme.Text),
		Value:       lipgloss.NewStyle().Foreground(theme.TextBright),
		Placeholder: lipgloss.NewStyle().Foreground(theme.TextMuted).Italic(true),

		// Panel styles
		Panel: lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(theme.Border),
		PanelActive: lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(theme.BorderActive),
		Header: lipgloss.NewStyle().
			Foreground(theme.Primary).
			Bold(true).
			Padding(0, 1),
		StatusBar: lipgloss.NewStyle().
			Foreground(theme.TextMuted).
			Padding(0, 1),

		// List styles
		ListItem: lipgloss.NewStyle().
			Foreground(theme.Text).
			PaddingLeft(2),
		ListItemSelected: lipgloss.NewStyle().
			Foreground(theme.Primary).
			Bold(true).
			PaddingLeft(1),
		ListItemDimmed: lipgloss.NewStyle().
			Foreground(theme.TextMuted).
			PaddingLeft(2),

		// Help styles
		HelpKey: lipgloss.NewStyle().
			Foreground(theme.Accent).
			Bold(true),
		HelpDesc: lipgloss.NewStyle().
			Foreground(theme.TextMuted),

		// Status styles
		Success: lipgloss.NewStyle().Foreground(theme.Success),
		Warning: lipgloss.NewStyle().Foreground(theme.Warning),
		Error:   lipgloss.NewStyle().Foreground(theme.Error),
	}
}

// DefaultStyles returns styles using the default theme.
func DefaultStyles() Styles {
	return NewStyles(DefaultTheme())
}
