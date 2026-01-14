// Package components provides reusable UI components for terminal applications.
// These components are designed to be library-ready and have no app-specific dependencies.
package components

import (
	"strings"

	"github.com/charmbracelet/lipgloss/v2"
)

// TabBarStyle defines the styling for a tab bar.
type TabBarStyle struct {
	// Active tab styling
	ActiveTab lipgloss.Style
	// Inactive tab styling
	InactiveTab lipgloss.Style
	// Gap filler styling (the line that fills remaining space)
	Gap lipgloss.Style
}

// DefaultTabBarStyle returns a default tab bar style.
func DefaultTabBarStyle() TabBarStyle {
	highlight := lipgloss.Color("#7D56F4")
	subtle := lipgloss.Color("#383838")

	// Active tab border - open bottom to connect to content
	activeTabBorder := lipgloss.Border{
		Top:         "─",
		Bottom:      " ",
		Left:        "│",
		Right:       "│",
		TopLeft:     "╭",
		TopRight:    "╮",
		BottomLeft:  "┘",
		BottomRight: "└",
	}

	// Inactive tab border - closed with connector
	inactiveTabBorder := lipgloss.Border{
		Top:         "─",
		Bottom:      "─",
		Left:        "│",
		Right:       "│",
		TopLeft:     "╭",
		TopRight:    "╮",
		BottomLeft:  "┴",
		BottomRight: "┴",
	}

	activeTab := lipgloss.NewStyle().
		Border(activeTabBorder, true).
		BorderForeground(highlight).
		Padding(0, 1)

	inactiveTab := lipgloss.NewStyle().
		Border(inactiveTabBorder, true).
		BorderForeground(subtle).
		Padding(0, 1)

	// Gap fills remaining space with just a bottom border
	gap := inactiveTab.
		BorderTop(false).
		BorderLeft(false).
		BorderRight(false)

	return TabBarStyle{
		ActiveTab:   activeTab,
		InactiveTab: inactiveTab,
		Gap:         gap,
	}
}

// Tab represents a single tab.
type Tab struct {
	ID    string // Unique identifier
	Label string // Display label
}

// TabBar renders a horizontal tab bar with one active tab.
type TabBar struct {
	tabs       []Tab
	activeIdx  int
	style      TabBarStyle
	width      int
}

// NewTabBar creates a new tab bar with the given tabs.
func NewTabBar(tabs []Tab) *TabBar {
	return &TabBar{
		tabs:  tabs,
		style: DefaultTabBarStyle(),
	}
}

// SetStyle sets the tab bar style.
func (t *TabBar) SetStyle(style TabBarStyle) {
	t.style = style
}

// SetWidth sets the total width for the tab bar.
// If set, a gap will fill the remaining space.
func (t *TabBar) SetWidth(width int) {
	t.width = width
}

// SetActive sets the active tab by index.
func (t *TabBar) SetActive(index int) {
	if index >= 0 && index < len(t.tabs) {
		t.activeIdx = index
	}
}

// SetActiveByID sets the active tab by ID.
func (t *TabBar) SetActiveByID(id string) {
	for i, tab := range t.tabs {
		if tab.ID == id {
			t.activeIdx = i
			return
		}
	}
}

// ActiveIndex returns the current active tab index.
func (t *TabBar) ActiveIndex() int {
	return t.activeIdx
}

// ActiveTab returns the current active tab.
func (t *TabBar) ActiveTab() *Tab {
	if t.activeIdx >= 0 && t.activeIdx < len(t.tabs) {
		return &t.tabs[t.activeIdx]
	}
	return nil
}

// Next moves to the next tab, wrapping around.
func (t *TabBar) Next() {
	if len(t.tabs) == 0 {
		return
	}
	t.activeIdx = (t.activeIdx + 1) % len(t.tabs)
}

// Prev moves to the previous tab, wrapping around.
func (t *TabBar) Prev() {
	if len(t.tabs) == 0 {
		return
	}
	t.activeIdx = (t.activeIdx - 1 + len(t.tabs)) % len(t.tabs)
}

// Tabs returns all tabs.
func (t *TabBar) Tabs() []Tab {
	return t.tabs
}

// SetTabs replaces all tabs.
func (t *TabBar) SetTabs(tabs []Tab) {
	t.tabs = tabs
	if t.activeIdx >= len(tabs) {
		t.activeIdx = 0
	}
}

// View renders the tab bar.
func (t *TabBar) View() string {
	if len(t.tabs) == 0 {
		return ""
	}

	var renderedTabs []string

	for i, tab := range t.tabs {
		if i == t.activeIdx {
			renderedTabs = append(renderedTabs, t.style.ActiveTab.Render(tab.Label))
		} else {
			renderedTabs = append(renderedTabs, t.style.InactiveTab.Render(tab.Label))
		}
	}

	row := lipgloss.JoinHorizontal(lipgloss.Top, renderedTabs...)

	// Add gap to fill remaining width if width is set
	if t.width > 0 {
		rowWidth := lipgloss.Width(row)
		gapWidth := t.width - rowWidth - 2 // -2 for border edges
		if gapWidth > 0 {
			gap := t.style.Gap.Render(strings.Repeat(" ", gapWidth))
			row = lipgloss.JoinHorizontal(lipgloss.Bottom, row, gap)
		}
	}

	return row
}

// ViewWithZones renders the tab bar with clickable zones.
// The zoneFn should return a zone-wrapped string for each tab.
// Example: zoneFn = func(tabID, content string) string { return zone.Mark("tab-"+tabID, content) }
func (t *TabBar) ViewWithZones(zoneFn func(tabID, content string) string) string {
	if len(t.tabs) == 0 {
		return ""
	}

	var renderedTabs []string

	for i, tab := range t.tabs {
		var rendered string
		if i == t.activeIdx {
			rendered = t.style.ActiveTab.Render(tab.Label)
		} else {
			rendered = t.style.InactiveTab.Render(tab.Label)
		}
		// Wrap in zone if zoneFn is provided
		if zoneFn != nil {
			rendered = zoneFn(tab.ID, rendered)
		}
		renderedTabs = append(renderedTabs, rendered)
	}

	row := lipgloss.JoinHorizontal(lipgloss.Top, renderedTabs...)

	// Add gap to fill remaining width if width is set
	if t.width > 0 {
		rowWidth := lipgloss.Width(row)
		gapWidth := t.width - rowWidth - 2
		if gapWidth > 0 {
			gap := t.style.Gap.Render(strings.Repeat(" ", gapWidth))
			row = lipgloss.JoinHorizontal(lipgloss.Bottom, row, gap)
		}
	}

	return row
}
