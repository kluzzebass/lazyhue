package panels

import (
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/bubbles/v2/key"
	"github.com/charmbracelet/bubbles/v2/list"
	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/charmbracelet/lipgloss/v2"
	zone "github.com/lrstanley/bubblezone/v2"

	"github.com/kluzzebass/lazyhue/internal/ui2"
)

// TreeNode represents a node in the hierarchical tree.
type TreeNode struct {
	ID       string      // Unique identifier for the node
	Label    string      // Display label
	Item     *EntityItem // nil for group headers/folders
	Children []*TreeNode // Child nodes
	Expanded bool        // Whether this node's children are visible
	Depth    int         // Nesting depth (0 = root)
}

// FilterValue implements list.Item for TreeNode.
func (n TreeNode) FilterValue() string { return n.Label }

// FlatNode represents a flattened tree node for display in a list.
type FlatNode struct {
	Node        *TreeNode
	Depth       int
	HasChildren bool
	IsExpanded  bool
}

// FilterValue implements list.Item for FlatNode.
func (f FlatNode) FilterValue() string { return f.Node.Label }

// TreeDelegate handles rendering of tree nodes in a list.
type TreeDelegate struct {
	Styles ui2.Styles
	Zones  *zone.Manager
}

// Height returns the height of a single item.
func (d TreeDelegate) Height() int { return 1 }

// Spacing returns the spacing between items.
func (d TreeDelegate) Spacing() int { return 0 }

// Update handles item-level updates.
func (d TreeDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }

// Render renders a single tree item.
func (d TreeDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	f, ok := item.(FlatNode)
	if !ok {
		return
	}

	node := f.Node
	isSelected := index == m.Index()

	// Build the tree prefix
	indent := strings.Repeat("  ", f.Depth)

	// Expand/collapse indicator
	prefix := "  "
	if f.HasChildren {
		if f.IsExpanded {
			prefix = "▼ "
		} else {
			prefix = "▶ "
		}
	}

	// On/off indicator for entities
	indicator := ""
	if node.Item != nil {
		if node.Item.IsOn {
			// Use custom color if available
			if node.Item.IndicatorColor != "" {
				indicator = lipgloss.NewStyle().
					Foreground(lipgloss.Color(node.Item.IndicatorColor)).
					Render("●") + " "
			} else {
				indicator = d.Styles.Success.Render("●") + " "
			}
		} else {
			indicator = d.Styles.Dimmed.Render("○") + " "
		}
	}

	// Build the label
	label := node.Label
	if isSelected {
		label = d.Styles.Selected.Render(label)
	} else if node.Item == nil {
		// Group headers get a different style
		label = d.Styles.Title.Render(label)
	}

	// Compose the line
	line := indent + prefix + indicator + label

	// Wrap in a clickable zone if we have a zone manager
	if d.Zones != nil {
		zoneID := ui2.TreeItemZone(node.ID)
		line = d.Zones.Mark(zoneID, line)
	}

	fmt.Fprint(w, line)
}

// TreePanel displays a hierarchical tree of items using list.Model.
type TreePanel struct {
	list   list.Model
	styles ui2.Styles
	zones  *zone.Manager
	title  string
	key    string

	// Tree data
	roots    []*TreeNode
	flatList []FlatNode

	// Dimensions
	width  int
	height int
}

// NewTreePanel creates a new tree panel.
func NewTreePanel(styles ui2.Styles, zones *zone.Manager, title, panelKey string) *TreePanel {
	delegate := TreeDelegate{
		Styles: styles,
		Zones:  zones,
	}

	l := list.New([]list.Item{}, delegate, 0, 0)
	l.SetShowTitle(false)
	l.SetShowStatusBar(false)
	l.SetShowFilter(false)
	l.SetShowHelp(false)
	l.SetShowPagination(false)
	l.DisableQuitKeybindings()

	return &TreePanel{
		list:   l,
		styles: styles,
		zones:  zones,
		title:  title,
		key:    panelKey,
	}
}

// SetRoots sets the tree data, preserving expanded state from previous tree.
func (p *TreePanel) SetRoots(roots []*TreeNode) {
	// Save current expanded state before replacing
	expandedState := p.getExpandedState()

	p.roots = roots

	// Restore expanded state to new tree
	p.restoreExpandedState(expandedState)

	// Rebuild the flat list for the list.Model
	p.rebuildFlatList()
}

// getExpandedState captures the expanded state of all nodes.
func (p *TreePanel) getExpandedState() map[string]bool {
	state := make(map[string]bool)
	var walk func(nodes []*TreeNode)
	walk = func(nodes []*TreeNode) {
		for _, node := range nodes {
			if node.ID != "" {
				state[node.ID] = node.Expanded
			}
			if len(node.Children) > 0 {
				walk(node.Children)
			}
		}
	}
	walk(p.roots)
	return state
}

// restoreExpandedState applies saved expanded state to the new tree.
func (p *TreePanel) restoreExpandedState(state map[string]bool) {
	if len(state) == 0 {
		return
	}
	var walk func(nodes []*TreeNode)
	walk = func(nodes []*TreeNode) {
		for _, node := range nodes {
			if node.ID != "" {
				if expanded, ok := state[node.ID]; ok {
					node.Expanded = expanded
				}
			}
			if len(node.Children) > 0 {
				walk(node.Children)
			}
		}
	}
	walk(p.roots)
}

// rebuildFlatList flattens the tree for display.
func (p *TreePanel) rebuildFlatList() {
	p.flatList = nil

	var walk func(nodes []*TreeNode, depth int)
	walk = func(nodes []*TreeNode, depth int) {
		for _, node := range nodes {
			hasChildren := len(node.Children) > 0
			p.flatList = append(p.flatList, FlatNode{
				Node:        node,
				Depth:       depth,
				HasChildren: hasChildren,
				IsExpanded:  node.Expanded,
			})

			if hasChildren && node.Expanded {
				walk(node.Children, depth+1)
			}
		}
	}
	walk(p.roots, 0)

	// Update the list items
	items := make([]list.Item, len(p.flatList))
	for i, f := range p.flatList {
		items[i] = f
	}
	p.list.SetItems(items)
}

// SelectedNode returns the currently selected tree node.
func (p *TreePanel) SelectedNode() *TreeNode {
	if p.list.Index() < 0 || p.list.Index() >= len(p.flatList) {
		return nil
	}
	return p.flatList[p.list.Index()].Node
}

// SelectedItem returns the entity item of the selected node (may be nil for group headers).
func (p *TreePanel) SelectedItem() *EntityItem {
	node := p.SelectedNode()
	if node == nil {
		return nil
	}
	return node.Item
}

// ToggleExpanded toggles the expanded state of the selected node.
func (p *TreePanel) ToggleExpanded() {
	node := p.SelectedNode()
	if node == nil || len(node.Children) == 0 {
		return
	}
	node.Expanded = !node.Expanded
	p.rebuildFlatList()
}

// Expand expands the selected node.
func (p *TreePanel) Expand() {
	node := p.SelectedNode()
	if node == nil || len(node.Children) == 0 || node.Expanded {
		return
	}
	node.Expanded = true
	p.rebuildFlatList()
}

// Collapse collapses the selected node.
func (p *TreePanel) Collapse() {
	node := p.SelectedNode()
	if node == nil {
		return
	}

	// If this node is expanded and has children, collapse it
	if len(node.Children) > 0 && node.Expanded {
		node.Expanded = false
		p.rebuildFlatList()
		return
	}

	// Otherwise, move to parent and collapse it
	// Find the parent by looking backwards in the flat list for a lower depth
	idx := p.list.Index()
	currentDepth := p.flatList[idx].Depth
	for i := idx - 1; i >= 0; i-- {
		if p.flatList[i].Depth < currentDepth {
			// Found parent
			p.flatList[i].Node.Expanded = false
			p.list.Select(i)
			p.rebuildFlatList()
			return
		}
	}
}

// SetSize sets the dimensions of the panel (total dimensions including border).
func (p *TreePanel) SetSize(width, height int) {
	p.width = width
	p.height = height
	// The list renders inside the border, so subtract 2 for each dimension
	innerWidth := width - 2
	innerHeight := height - 2
	if innerWidth < 1 {
		innerWidth = 1
	}
	if innerHeight < 1 {
		innerHeight = 1
	}
	p.list.SetSize(innerWidth, innerHeight)
}

// Update handles messages.
func (p *TreePanel) Update(msg tea.Msg) (*TreePanel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, key.NewBinding(key.WithKeys("left", "h"))):
			p.Collapse()
			return p, nil
		case key.Matches(msg, key.NewBinding(key.WithKeys("right", "l"))):
			p.Expand()
			return p, nil
		case key.Matches(msg, key.NewBinding(key.WithKeys("enter", " "))):
			p.ToggleExpanded()
			return p, nil
		}

	case tea.MouseClickMsg:
		// Check for zone clicks
		if p.zones != nil {
			for i, f := range p.flatList {
				zoneID := ui2.TreeItemZone(f.Node.ID)
				if p.zones.Get(zoneID).InBounds(msg) {
					p.list.Select(i)
					if msg.Button == tea.MouseLeft {
						// Click on expand icon toggles
						if f.HasChildren {
							p.ToggleExpanded()
						}
					}
					return p, nil
				}
			}
		}
	}

	// Pass to underlying list
	var cmd tea.Cmd
	p.list, cmd = p.list.Update(msg)
	return p, cmd
}

// View renders the panel.
func (p *TreePanel) View(focused bool) string {
	// Choose border style based on focus
	borderStyle := p.styles.Panel
	if focused {
		borderStyle = p.styles.PanelActive
	}

	// Render the list content
	content := p.list.View()

	// Wrap in a panel with title
	// Width/Height here are for the content area inside the border
	innerWidth := p.width - 2   // Account for left+right border
	innerHeight := p.height - 2 // Account for top+bottom border
	if innerWidth < 1 {
		innerWidth = 1
	}
	if innerHeight < 1 {
		innerHeight = 1
	}

	panel := borderStyle.
		Width(innerWidth).
		Height(innerHeight).
		Render(content)

	return panel
}

// Title returns the panel title.
func (p *TreePanel) Title() string { return p.title }

// Key returns the panel hotkey.
func (p *TreePanel) Key() string { return p.key }

// Clear clears the tree data.
func (p *TreePanel) Clear() {
	p.roots = nil
	p.flatList = nil
	p.list.SetItems(nil)
}

// Index returns the current cursor position.
func (p *TreePanel) Index() int {
	return p.list.Index()
}

// Select sets the cursor position.
func (p *TreePanel) Select(index int) {
	p.list.Select(index)
}

// Len returns the number of visible items.
func (p *TreePanel) Len() int {
	return len(p.flatList)
}
