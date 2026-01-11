// lazyhue2 is a test app for the v2 UI implementation.
// This is used to verify bubbles v2, bubbletea v2, lipgloss v2, and bubblezone work correctly.
package main

import (
	"fmt"
	"io"
	"os"

	"github.com/charmbracelet/bubbles/v2/help"
	"github.com/charmbracelet/bubbles/v2/key"
	"github.com/charmbracelet/bubbles/v2/list"
	tea "github.com/charmbracelet/bubbletea/v2"
	zone "github.com/lrstanley/bubblezone/v2"

	"github.com/kluzzebass/lazyhue/internal/ui2"
)

// Item represents a list item.
type item struct {
	title       string
	description string
	id          string
}

func (i item) Title() string       { return i.title }
func (i item) Description() string { return i.description }
func (i item) FilterValue() string { return i.title }

// itemDelegate handles rendering of list items with bubblezone support.
type itemDelegate struct {
	styles ui2.Styles
	zones  *zone.Manager
}

func (d itemDelegate) Height() int                             { return 1 }
func (d itemDelegate) Spacing() int                            { return 0 }
func (d itemDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }

func (d itemDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	i, ok := listItem.(item)
	if !ok {
		return
	}

	var str string
	if index == m.Index() {
		str = d.styles.ListItemSelected.Render("▶ " + i.title)
	} else {
		str = d.styles.ListItem.Render("  " + i.title)
	}

	// Wrap in a clickable zone
	zoneID := ui2.TreeItemZone(i.id)
	str = d.zones.Mark(zoneID, str)

	fmt.Fprint(w, str)
}

// Model is the main application model.
type model struct {
	list    list.Model
	keys    ui2.KeyMap
	help    help.Model
	styles  ui2.Styles
	zones   *zone.Manager
	width   int
	height  int
	status  string
	quiting bool
}

func initialModel() model {
	styles := ui2.DefaultStyles()
	keys := ui2.DefaultKeyMap()
	zones := zone.New()

	// Create sample items
	items := []list.Item{
		item{title: "Living Room", description: "3 lights", id: "room-1"},
		item{title: "Bedroom", description: "2 lights", id: "room-2"},
		item{title: "Kitchen", description: "4 lights", id: "room-3"},
		item{title: "Bathroom", description: "1 light", id: "room-4"},
		item{title: "Office", description: "2 lights", id: "room-5"},
	}

	// Create list with custom delegate
	delegate := itemDelegate{styles: styles, zones: zones}
	l := list.New(items, delegate, 30, 10)
	l.Title = "Rooms"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(true)
	l.Styles.Title = styles.Title

	// Create help model
	h := help.New()

	return model{
		list:   l,
		keys:   keys,
		help:   h,
		styles: styles,
		zones:  zones,
		status: "Click on items or use arrow keys to navigate",
	}
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.list.SetSize(m.width-4, m.height-8)
		m.help.Width = m.width
		return m, nil

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keys.Quit):
			m.quiting = true
			return m, tea.Quit

		case key.Matches(msg, m.keys.Help):
			m.help.ShowAll = !m.help.ShowAll
			return m, nil

		case key.Matches(msg, m.keys.Enter):
			if i, ok := m.list.SelectedItem().(item); ok {
				m.status = fmt.Sprintf("Selected: %s", i.title)
			}
			return m, nil
		}

	case tea.MouseMsg:
		// Check for zone clicks
		for _, i := range m.list.Items() {
			if it, ok := i.(item); ok {
				zoneID := ui2.TreeItemZone(it.id)
				if m.zones.Get(zoneID).InBounds(msg) {
					m.status = fmt.Sprintf("Clicked: %s", it.title)
					// Find and select this item
					for idx, li := range m.list.Items() {
						if lit, ok := li.(item); ok && lit.id == it.id {
							m.list.Select(idx)
							break
						}
					}
					return m, nil
				}
			}
		}
	}

	// Pass to list
	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	if cmd != nil {
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m model) View() string {
	if m.quiting {
		return "Goodbye!\n"
	}

	// Build the view
	var output string

	// Header
	header := m.styles.Header.Render("lazyhue v2 Test App")
	output += header + "\n\n"

	// List panel
	listView := m.styles.Panel.Width(m.width - 2).Render(m.list.View())
	output += listView + "\n"

	// Status bar
	status := m.styles.StatusBar.Render(m.status)
	output += status + "\n"

	// Help
	helpView := m.help.View(m.keys)
	output += "\n" + helpView

	// Scan for zones
	return m.zones.Scan(output)
}

func main() {
	// Create program with mouse support
	p := tea.NewProgram(
		initialModel(),
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
