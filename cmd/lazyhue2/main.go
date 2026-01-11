// lazyhue2 is a test app for the v2 UI implementation.
// This is used to verify bubbles v2, bubbletea v2, lipgloss v2, and bubblezone work correctly.
package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/v2/help"
	"github.com/charmbracelet/bubbles/v2/key"
	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/charmbracelet/lipgloss/v2"
	zone "github.com/lrstanley/bubblezone/v2"

	"github.com/kluzzebass/lazyhue/internal/ui2"
	"github.com/kluzzebass/lazyhue/internal/ui2/components"
	"github.com/kluzzebass/lazyhue/internal/ui2/layout"
	"github.com/kluzzebass/lazyhue/internal/ui2/panels"
)

// Panel IDs
const (
	PanelTree   = "tree"
	PanelDetail = "detail"
	PanelLog    = "log"
)

// Model is the main application model.
type model struct {
	tabBar      *components.TabBar
	tree        *panels.TreePanel
	keys        ui2.KeyMap
	help        help.Model
	styles      ui2.Styles
	zones       *zone.Manager
	layout      *layout.Tree
	width       int
	height      int
	focusedPane string // Which pane is focused
	status      string
	logLines    []string
	quiting     bool
}

func initialModel() model {
	styles := ui2.DefaultStyles()
	keys := ui2.DefaultKeyMap()
	zones := zone.New()

	// Create tab bar
	tabBar := components.NewTabBar([]components.Tab{
		{ID: "home", Label: "Home"},
		{ID: "bridges", Label: "Bridges"},
		{ID: "settings", Label: "Settings"},
	})

	// Create tree panel
	tree := panels.NewTreePanel(styles, zones, "Home", "1")

	// Create sample tree data
	roots := createSampleTree()
	tree.SetRoots(roots)

	// Create help model
	h := help.New()

	// Build layout tree
	layoutRoot := layout.HSplit(
		layout.Child{Size: layout.Flex(0.3), Node: layout.NewLeaf(PanelTree)},
		layout.Child{Size: layout.Flex(0.7), Node: layout.VSplit(
			layout.Child{Size: layout.Flex(1), Node: layout.NewLeaf(PanelDetail)},
			layout.Child{Size: layout.Fixed(6), Node: layout.NewLeaf(PanelLog)},
		)},
	)
	layoutTree := layout.NewTree(layoutRoot)

	return model{
		tabBar:      tabBar,
		tree:        tree,
		keys:        keys,
		help:        h,
		styles:      styles,
		zones:       zones,
		layout:      layoutTree,
		focusedPane: PanelTree,
		status:      "Use Tab to switch panels, [/] to switch tabs, arrows to navigate",
		logLines:    []string{"Application started", "Connected to sample bridge"},
	}
}

// createSampleTree creates sample tree data mimicking the Hue hierarchy.
func createSampleTree() []*panels.TreeNode {
	return []*panels.TreeNode{
		{
			ID:       "room-living",
			Label:    "Living Room",
			Expanded: true,
			Children: []*panels.TreeNode{
				{
					ID:       "lights-living",
					Label:    "Lights",
					Expanded: true,
					Children: []*panels.TreeNode{
						{
							ID:    "light-1",
							Label: "Ceiling Light",
							Item: &panels.EntityItem{
								ID:             "light-1",
								Name:           "Ceiling Light",
								Type:           panels.EntityLight,
								IsOn:           true,
								IndicatorColor: "#FFB800",
								Brightness:     80,
							},
						},
						{
							ID:    "light-2",
							Label: "Floor Lamp",
							Item: &panels.EntityItem{
								ID:             "light-2",
								Name:           "Floor Lamp",
								Type:           panels.EntityLight,
								IsOn:           true,
								IndicatorColor: "#FF6B00",
								Brightness:     50,
							},
						},
						{
							ID:    "light-3",
							Label: "TV Backlight",
							Item: &panels.EntityItem{
								ID:         "light-3",
								Name:       "TV Backlight",
								Type:       panels.EntityLight,
								IsOn:       false,
								Brightness: 0,
							},
						},
					},
				},
				{
					ID:    "scenes-living",
					Label: "Scenes",
					Children: []*panels.TreeNode{
						{
							ID:    "scene-1",
							Label: "Relax",
							Item: &panels.EntityItem{
								ID:   "scene-1",
								Name: "Relax",
								Type: panels.EntityScene,
							},
						},
						{
							ID:    "scene-2",
							Label: "Movie Time",
							Item: &panels.EntityItem{
								ID:   "scene-2",
								Name: "Movie Time",
								Type: panels.EntityScene,
							},
						},
					},
				},
			},
		},
		{
			ID:       "room-bedroom",
			Label:    "Bedroom",
			Expanded: false,
			Children: []*panels.TreeNode{
				{
					ID:    "lights-bedroom",
					Label: "Lights",
					Children: []*panels.TreeNode{
						{
							ID:    "light-4",
							Label: "Bedside Lamp",
							Item: &panels.EntityItem{
								ID:         "light-4",
								Name:       "Bedside Lamp",
								Type:       panels.EntityLight,
								IsOn:       false,
								Brightness: 0,
							},
						},
					},
				},
			},
		},
		{
			ID:       "room-kitchen",
			Label:    "Kitchen",
			Expanded: false,
			Children: []*panels.TreeNode{
				{
					ID:    "lights-kitchen",
					Label: "Lights",
					Children: []*panels.TreeNode{
						{
							ID:    "light-5",
							Label: "Under Cabinet",
							Item: &panels.EntityItem{
								ID:             "light-5",
								Name:           "Under Cabinet",
								Type:           panels.EntityLight,
								IsOn:           true,
								IndicatorColor: "#FFFFFF",
								Brightness:     100,
							},
						},
						{
							ID:    "light-6",
							Label: "Pendant Light",
							Item: &panels.EntityItem{
								ID:             "light-6",
								Name:           "Pendant Light",
								Type:           panels.EntityLight,
								IsOn:           true,
								IndicatorColor: "#FFD600",
								Brightness:     70,
							},
						},
					},
				},
			},
		},
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

		// Reserve space for tab bar and help
		tabBarHeight := 3 // Border top + content + border bottom
		helpHeight := 3
		contentHeight := m.height - tabBarHeight - helpHeight

		// Update layout
		m.layout.Layout(m.width, contentHeight)

		// Update panel sizes
		treeBounds := m.layout.Bounds(PanelTree)
		m.tree.SetSize(treeBounds.Width, treeBounds.Height)

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

		case key.Matches(msg, m.keys.NextBridge): // ] - next tab
			m.tabBar.Next()
			if tab := m.tabBar.ActiveTab(); tab != nil {
				m.logLines = append(m.logLines, fmt.Sprintf("Tab: %s", tab.Label))
			}
			return m, nil

		case key.Matches(msg, m.keys.PrevBridge): // [ - prev tab
			m.tabBar.Prev()
			if tab := m.tabBar.ActiveTab(); tab != nil {
				m.logLines = append(m.logLines, fmt.Sprintf("Tab: %s", tab.Label))
			}
			return m, nil

		case key.Matches(msg, m.keys.NextPanel):
			// Cycle focus
			switch m.focusedPane {
			case PanelTree:
				m.focusedPane = PanelDetail
			case PanelDetail:
				m.focusedPane = PanelLog
			default:
				m.focusedPane = PanelTree
			}
			m.logLines = append(m.logLines, fmt.Sprintf("Focused: %s", m.focusedPane))
			return m, nil
		}

	case tea.MouseClickMsg:
		// Check which panel was clicked (offset by tab bar height)
		if leaf := m.layout.At(msg.X, msg.Y-3); leaf != nil { // -3 for tab bar
			if leaf.ID != m.focusedPane {
				m.focusedPane = leaf.ID
				m.logLines = append(m.logLines, fmt.Sprintf("Clicked: %s", leaf.ID))
			}
		}
	}

	// Pass events to focused panel
	if m.focusedPane == PanelTree {
		var cmd tea.Cmd
		m.tree, cmd = m.tree.Update(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}

		// Update status based on selection
		if node := m.tree.SelectedNode(); node != nil {
			if node.Item != nil {
				status := fmt.Sprintf("Selected: %s (%s)", node.Item.Name, node.Item.Type)
				if node.Item.IsOn {
					status += " [ON]"
				} else {
					status += " [OFF]"
				}
				m.status = status
			} else {
				m.status = fmt.Sprintf("Selected: %s (folder)", node.Label)
			}
		}
	}

	return m, tea.Batch(cmds...)
}

func (m model) View() string {
	if m.quiting {
		return "Goodbye!\n"
	}

	var output strings.Builder

	// Tab bar
	m.tabBar.SetWidth(m.width)
	output.WriteString(m.tabBar.View() + "\n")

	// Calculate panel bounds
	detailBounds := m.layout.Bounds(PanelDetail)
	logBounds := m.layout.Bounds(PanelLog)

	// Render tree panel
	treeContent := m.tree.View(m.focusedPane == PanelTree)

	// Render detail panel (placeholder for now)
	detailContent := m.renderDetailPanel(detailBounds.Width, detailBounds.Height, m.focusedPane == PanelDetail)

	// Render log panel
	logContent := m.renderLogPanel(logBounds.Width, logBounds.Height, m.focusedPane == PanelLog)

	// Compose right side (detail + log stacked)
	rightSide := lipgloss.JoinVertical(lipgloss.Left, detailContent, logContent)

	// Compose main content (tree | right side)
	mainContent := lipgloss.JoinHorizontal(lipgloss.Top, treeContent, rightSide)
	output.WriteString(mainContent + "\n")

	// Status bar
	statusStyle := lipgloss.NewStyle().
		Foreground(m.styles.Theme.TextMuted).
		Width(m.width)
	output.WriteString(statusStyle.Render(m.status) + "\n")

	// Help
	helpView := m.help.View(m.keys)
	output.WriteString(helpView)

	// Scan for zones
	return m.zones.Scan(output.String())
}

func (m model) renderDetailPanel(width, height int, focused bool) string {
	borderStyle := m.styles.Panel
	if focused {
		borderStyle = m.styles.PanelActive
	}

	// Calculate inner dimensions (content area inside border)
	innerWidth := width - 2
	innerHeight := height - 2
	if innerWidth < 1 {
		innerWidth = 1
	}
	if innerHeight < 1 {
		innerHeight = 1
	}

	// Get selected item details
	var content string
	if node := m.tree.SelectedNode(); node != nil && node.Item != nil {
		item := node.Item
		lines := []string{
			fmt.Sprintf("Name: %s", item.Name),
			fmt.Sprintf("Type: %s", item.Type),
			fmt.Sprintf("ID:   %s", item.ID),
		}
		if item.Type == panels.EntityLight {
			if item.IsOn {
				lines = append(lines, "Power: ON")
				lines = append(lines, fmt.Sprintf("Brightness: %.0f%%", item.Brightness))
				if item.IndicatorColor != "" {
					lines = append(lines, fmt.Sprintf("Color: %s", item.IndicatorColor))
				}
			} else {
				lines = append(lines, "Power: OFF")
			}
		}
		content = strings.Join(lines, "\n")
	} else if node := m.tree.SelectedNode(); node != nil {
		content = fmt.Sprintf("Folder: %s\n\nExpand to see contents", node.Label)
	} else {
		content = "No selection"
	}

	// Title
	title := "Details"
	titleStyle := lipgloss.NewStyle().
		Foreground(m.styles.Theme.Primary).
		Bold(true)

	// Pad content to fill space
	contentLines := strings.Split(content, "\n")
	maxLines := innerHeight - 1 // -1 for title
	if maxLines < 1 {
		maxLines = 1
	}
	for len(contentLines) < maxLines {
		contentLines = append(contentLines, "")
	}
	if len(contentLines) > maxLines {
		contentLines = contentLines[:maxLines]
	}

	fullContent := titleStyle.Render(title) + "\n" + strings.Join(contentLines, "\n")

	return borderStyle.
		Width(innerWidth).
		Height(innerHeight).
		Render(fullContent)
}

func (m model) renderLogPanel(width, height int, focused bool) string {
	borderStyle := m.styles.Panel
	if focused {
		borderStyle = m.styles.PanelActive
	}

	// Calculate inner dimensions (content area inside border)
	innerWidth := width - 2
	innerHeight := height - 2
	if innerWidth < 1 {
		innerWidth = 1
	}
	if innerHeight < 1 {
		innerHeight = 1
	}

	title := "Activity Log"
	titleStyle := lipgloss.NewStyle().
		Foreground(m.styles.Theme.Primary).
		Bold(true)

	// Show last N log lines (leaving room for title)
	maxLines := innerHeight - 1
	if maxLines < 1 {
		maxLines = 1
	}
	startIdx := len(m.logLines) - maxLines
	if startIdx < 0 {
		startIdx = 0
	}
	visibleLogs := m.logLines[startIdx:]

	logContent := strings.Join(visibleLogs, "\n")

	fullContent := titleStyle.Render(title) + "\n" + logContent

	return borderStyle.
		Width(innerWidth).
		Height(innerHeight).
		Render(fullContent)
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
