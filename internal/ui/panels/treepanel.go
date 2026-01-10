package panels

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/kluzzebass/lazyhue/internal/ui"
)

// TreeNode represents a node in the tree.
type TreeNode struct {
	Label    string
	Item     *EntityItem // nil for group headers
	Children []*TreeNode
	Expanded bool
}

// TreePanel displays a hierarchical tree of items.
type TreePanel struct {
	ScrollState        // Embedded scroll state for cursor/offset management
	styles     ui.Styles
	panelKey   string
	panelTitle string

	// Tree data
	roots    []*TreeNode
	flatList []*TreeNode // Flattened visible nodes for navigation
}

// NewTreePanel creates a new tree panel.
func NewTreePanel(styles ui.Styles, title, panelKey string) *TreePanel {
	return &TreePanel{
		styles:     styles,
		panelTitle: title,
		panelKey:   panelKey,
	}
}

// SetRoots sets the tree data, preserving expanded state from previous tree.
func (p *TreePanel) SetRoots(roots []*TreeNode) {
	// Save current expanded state before replacing
	expandedState := p.getExpandedState()

	p.roots = roots

	// Restore expanded state to new tree
	p.restoreExpandedState(expandedState)

	p.rebuildFlatList()
}

// nodeKey returns a stable key for a node (entity ID if available, otherwise label).
func nodeKey(node *TreeNode) string {
	if node.Item != nil && node.Item.ID != "" {
		return node.Item.ID
	}
	return node.Label
}

// getExpandedState captures the expanded state of all nodes, keyed by path.
func (p *TreePanel) getExpandedState() map[string]bool {
	state := make(map[string]bool)
	var walk func(nodes []*TreeNode, path string)
	walk = func(nodes []*TreeNode, path string) {
		for _, node := range nodes {
			nodePath := path + "/" + nodeKey(node)
			state[nodePath] = node.Expanded
			if len(node.Children) > 0 {
				walk(node.Children, nodePath)
			}
		}
	}
	walk(p.roots, "")
	return state
}

// restoreExpandedState applies saved expanded state to the new tree.
func (p *TreePanel) restoreExpandedState(state map[string]bool) {
	if len(state) == 0 {
		return // First load, keep defaults
	}
	var walk func(nodes []*TreeNode, path string)
	walk = func(nodes []*TreeNode, path string) {
		for _, node := range nodes {
			nodePath := path + "/" + nodeKey(node)
			if expanded, ok := state[nodePath]; ok {
				node.Expanded = expanded
			}
			if len(node.Children) > 0 {
				walk(node.Children, nodePath)
			}
		}
	}
	walk(p.roots, "")
}

// GetNodeStates returns a map of all node paths to their expanded state.
func (p *TreePanel) GetNodeStates() map[string]bool {
	states := make(map[string]bool)
	var walk func(nodes []*TreeNode, path string)
	walk = func(nodes []*TreeNode, path string) {
		for _, node := range nodes {
			nodePath := path + "/" + nodeKey(node)
			// Only save state for nodes that have children (can be expanded)
			if len(node.Children) > 0 {
				states[nodePath] = node.Expanded
				walk(node.Children, nodePath)
			}
		}
	}
	walk(p.roots, "")
	return states
}

// SetNodeStates applies saved expanded/collapsed states to the tree.
func (p *TreePanel) SetNodeStates(states map[string]bool) {
	if len(states) == 0 {
		return
	}
	var walk func(nodes []*TreeNode, path string)
	walk = func(nodes []*TreeNode, path string) {
		for _, node := range nodes {
			nodePath := path + "/" + nodeKey(node)
			// Apply saved state if it exists, otherwise keep default
			if expanded, ok := states[nodePath]; ok {
				node.Expanded = expanded
			}
			if len(node.Children) > 0 {
				walk(node.Children, nodePath)
			}
		}
	}
	walk(p.roots, "")
	p.rebuildFlatList()
}

// rebuildFlatList creates a flat list of visible nodes for navigation.
func (p *TreePanel) rebuildFlatList() {
	p.flatList = nil
	var walk func(nodes []*TreeNode)
	walk = func(nodes []*TreeNode) {
		for _, node := range nodes {
			p.flatList = append(p.flatList, node)
			if node.Expanded && len(node.Children) > 0 {
				walk(node.Children)
			}
		}
	}
	walk(p.roots)

	// Clamp cursor to new list size
	p.ClampCursor(len(p.flatList))
}

// itemCount returns the number of items in the flat list.
func (p *TreePanel) itemCount() int {
	return len(p.flatList)
}

// Update handles input.
func (p *TreePanel) Update(msg tea.Msg) tea.Cmd {
	itemCount := p.itemCount()

	switch msg := msg.(type) {
	case tea.MouseMsg:
		switch msg.Button {
		case tea.MouseButtonWheelUp:
			p.MoveCursor(-1, itemCount)
		case tea.MouseButtonWheelDown:
			p.MoveCursor(1, itemCount)
		}

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, key.NewBinding(key.WithKeys("up", "k"))):
			p.MoveCursor(-1, itemCount)
		case key.Matches(msg, key.NewBinding(key.WithKeys("down", "j"))):
			p.MoveCursor(1, itemCount)
		case key.Matches(msg, key.NewBinding(key.WithKeys("pgup"))):
			p.PageUp(itemCount)
		case key.Matches(msg, key.NewBinding(key.WithKeys("pgdown"))):
			p.PageDown(itemCount)
		case key.Matches(msg, key.NewBinding(key.WithKeys("home"))):
			p.MoveToStart()
		case key.Matches(msg, key.NewBinding(key.WithKeys("end"))):
			p.MoveToEnd(itemCount)
		case key.Matches(msg, key.NewBinding(key.WithKeys("right", "l"))):
			// Expand only (not toggle)
			cursor := p.Cursor()
			if cursor < len(p.flatList) {
				node := p.flatList[cursor]
				if len(node.Children) > 0 && !node.Expanded {
					node.Expanded = true
					p.rebuildFlatList()
				}
			}
		case key.Matches(msg, key.NewBinding(key.WithKeys("left", "h"))):
			// Collapse only
			cursor := p.Cursor()
			if cursor < len(p.flatList) {
				node := p.flatList[cursor]
				if node.Expanded && len(node.Children) > 0 {
					node.Expanded = false
					p.rebuildFlatList()
				}
			}
		}
	}
	return nil
}

// IsGroupSelected returns true if the cursor is on a group (not a leaf).
func (p *TreePanel) IsGroupSelected() bool {
	cursor := p.Cursor()
	if cursor < len(p.flatList) {
		return len(p.flatList[cursor].Children) > 0
	}
	return false
}

// ToggleSelected toggles expand/collapse if on a group node.
func (p *TreePanel) ToggleSelected() {
	cursor := p.Cursor()
	if cursor < len(p.flatList) {
		node := p.flatList[cursor]
		if len(node.Children) > 0 {
			node.Expanded = !node.Expanded
			p.rebuildFlatList()
		}
	}
}

// View renders the tree panel.
func (p *TreePanel) View(active bool) string {
	contentHeight := p.ViewHeight()
	contentWidth := p.ContentWidth()

	// Build visible content using scroll offset
	var lines []string
	start, end := p.VisibleRange(len(p.flatList))

	for i := start; i < end; i++ {
		node := p.flatList[i]
		line := p.renderNode(node, i == p.Cursor(), active)
		// Truncate to width using proper visual width handling
		line = ui.TruncateString(line, contentWidth)
		lines = append(lines, line)
	}

	// Pad to fill height
	for len(lines) < contentHeight {
		lines = append(lines, "")
	}

	content := lipgloss.JoinVertical(lipgloss.Left, lines...)

	cfg := ui.BorderConfig{
		PanelKey:    p.panelKey,
		Title:       p.panelTitle,
		ItemIndex:   p.Cursor(),
		ItemCount:   len(p.flatList),
		ScrollPos:   p.Offset(),
		TotalHeight: len(p.flatList),
		ViewHeight:  contentHeight,
	}

	return ui.RenderBorderedPanel(content, p.Width(), p.Height(), active, p.styles, cfg)
}

// renderNode renders a single node line.
func (p *TreePanel) renderNode(node *TreeNode, selected, active bool) string {
	depth := p.nodeDepth(node)
	lineWidth := p.ContentWidth()
	return RenderTreeLine(node, depth, lineWidth, selected, active, p.styles)
}

// nodeDepth finds the depth of a node.
func (p *TreePanel) nodeDepth(target *TreeNode) int {
	var findDepth func(nodes []*TreeNode, depth int) int
	findDepth = func(nodes []*TreeNode, depth int) int {
		for _, node := range nodes {
			if node == target {
				return depth
			}
			if len(node.Children) > 0 {
				if d := findDepth(node.Children, depth+1); d >= 0 {
					return d
				}
			}
		}
		return -1
	}
	d := findDepth(p.roots, 0)
	if d < 0 {
		return 0
	}
	return d
}

// Key returns the keyboard shortcut key.
func (p *TreePanel) Key() string {
	return p.panelKey
}

// Title returns the panel title.
func (p *TreePanel) Title() string {
	return p.panelTitle
}

// SelectedEntity returns the currently selected entity (skips group headers).
func (p *TreePanel) SelectedEntity() (*EntityItem, bool) {
	cursor := p.Cursor()
	if cursor < len(p.flatList) {
		node := p.flatList[cursor]
		if node.Item != nil {
			return node.Item, true
		}
	}
	return nil, false
}

// SelectedNode returns the currently selected node.
func (p *TreePanel) SelectedNode() *TreeNode {
	cursor := p.Cursor()
	if cursor < len(p.flatList) {
		return p.flatList[cursor]
	}
	return nil
}

// SelectByID finds and selects a node by its entity ID.
// Returns true if the entity was found and selected.
func (p *TreePanel) SelectByID(id string) bool {
	if id == "" {
		return false
	}
	for i, node := range p.flatList {
		if node.Item != nil && node.Item.ID == id {
			p.SetCursor(i)
			p.EnsureCursorVisible(len(p.flatList))
			return true
		}
	}
	return false
}

// FlatListLen returns the number of items in the flattened tree.
func (p *TreePanel) FlatListLen() int {
	return len(p.flatList)
}

// FlatListItem returns the node at the given index in the flat list.
func (p *TreePanel) FlatListItem(index int) *TreeNode {
	if index >= 0 && index < len(p.flatList) {
		return p.flatList[index]
	}
	return nil
}

// HandleClick handles a mouse click at relative coordinates within the panel.
func (p *TreePanel) HandleClick(relX, relY int) {
	// Content starts at y=1 (after top border), x=1 (after left border)
	if relY < 1 {
		return
	}

	// Calculate which item was clicked
	itemIndex := p.Offset() + (relY - 1)
	if itemIndex >= 0 && itemIndex < len(p.flatList) {
		node := p.flatList[itemIndex]
		wasAlreadySelected := (itemIndex == p.Cursor())
		p.SetCursor(itemIndex)

		// Toggle expand/collapse if:
		// 1. Click is on the expand/collapse indicator, OR
		// 2. Clicking on an already-selected group node
		if len(node.Children) > 0 {
			depth := p.nodeDepth(node)
			indicatorStart := 1 + (depth * 2) // 1 for border, then indentation
			indicatorEnd := indicatorStart + 2 // indicator is 2 chars wide (▶ or ▼ + space)

			clickedOnIndicator := relX >= indicatorStart && relX < indicatorEnd

			if clickedOnIndicator || wasAlreadySelected {
				node.Expanded = !node.Expanded
				p.rebuildFlatList()
			}
		}
	}
}
