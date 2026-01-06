package panels

import (
	"strings"

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
	styles     ui.Styles
	width      int
	height     int
	panelKey   string
	panelTitle string

	// Tree data
	roots    []*TreeNode
	flatList []*TreeNode // Flattened visible nodes for navigation
	cursor   int
	offset   int // Scroll offset (first visible item index)
}

// NewTreePanel creates a new tree panel.
func NewTreePanel(styles ui.Styles, title, panelKey string) *TreePanel {
	return &TreePanel{
		styles:     styles,
		panelTitle: title,
		panelKey:   panelKey,
	}
}

// SetSize updates the panel dimensions.
func (p *TreePanel) SetSize(width, height int) {
	p.width = width
	p.height = height
}

// SetRoots sets the tree data, preserving expanded state from previous tree.
func (p *TreePanel) SetRoots(roots []*TreeNode) {
	// Save current expanded state before replacing
	expandedState := p.getExpandedState()

	p.roots = roots

	// Restore expanded state to new tree
	p.restoreExpandedState(expandedState)

	p.rebuildFlatList()
	// Keep cursor in bounds
	if p.cursor >= len(p.flatList) {
		p.cursor = max(0, len(p.flatList)-1)
	}
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

	// Clamp cursor and offset to new list size
	if len(p.flatList) == 0 {
		p.cursor = 0
		p.offset = 0
	} else {
		if p.cursor >= len(p.flatList) {
			p.cursor = len(p.flatList) - 1
		}
		p.ensureCursorVisible()
	}
}

// Update handles input.
func (p *TreePanel) Update(msg tea.Msg) tea.Cmd {
	viewHeight := p.viewHeight()

	switch msg := msg.(type) {
	case tea.MouseMsg:
		switch msg.Button {
		case tea.MouseButtonWheelUp:
			p.moveCursor(-1)
		case tea.MouseButtonWheelDown:
			p.moveCursor(1)
		}

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, key.NewBinding(key.WithKeys("up", "k"))):
			p.moveCursor(-1)
		case key.Matches(msg, key.NewBinding(key.WithKeys("down", "j"))):
			p.moveCursor(1)
		case key.Matches(msg, key.NewBinding(key.WithKeys("pgup"))):
			p.moveCursor(-viewHeight)
		case key.Matches(msg, key.NewBinding(key.WithKeys("pgdown"))):
			p.moveCursor(viewHeight)
		case key.Matches(msg, key.NewBinding(key.WithKeys("home"))):
			p.cursor = 0
			p.offset = 0
		case key.Matches(msg, key.NewBinding(key.WithKeys("end"))):
			p.cursor = max(0, len(p.flatList)-1)
			p.ensureCursorVisible()
		case key.Matches(msg, key.NewBinding(key.WithKeys("right", "l"))):
			// Expand only (not toggle)
			if p.cursor < len(p.flatList) {
				node := p.flatList[p.cursor]
				if len(node.Children) > 0 && !node.Expanded {
					node.Expanded = true
					p.rebuildFlatList()
				}
			}
		case key.Matches(msg, key.NewBinding(key.WithKeys("left", "h"))):
			// Collapse only
			if p.cursor < len(p.flatList) {
				node := p.flatList[p.cursor]
				if node.Expanded && len(node.Children) > 0 {
					node.Expanded = false
					p.rebuildFlatList()
				}
			}
		}
	}
	return nil
}

// viewHeight returns the number of visible lines.
func (p *TreePanel) viewHeight() int {
	return max(1, p.height-2)
}

// moveCursor moves the cursor by delta, adjusting scroll as needed.
func (p *TreePanel) moveCursor(delta int) {
	if len(p.flatList) == 0 {
		return
	}

	// Move cursor
	p.cursor += delta
	if p.cursor < 0 {
		p.cursor = 0
	}
	if p.cursor >= len(p.flatList) {
		p.cursor = len(p.flatList) - 1
	}

	p.ensureCursorVisible()
}

// ensureCursorVisible adjusts the scroll offset to keep cursor in view.
func (p *TreePanel) ensureCursorVisible() {
	viewHeight := p.viewHeight()

	// Scroll up if cursor is above visible area
	if p.cursor < p.offset {
		p.offset = p.cursor
	}

	// Scroll down if cursor is below visible area
	if p.cursor >= p.offset+viewHeight {
		p.offset = p.cursor - viewHeight + 1
	}

	// Clamp offset
	maxOffset := max(0, len(p.flatList)-viewHeight)
	if p.offset > maxOffset {
		p.offset = maxOffset
	}
	if p.offset < 0 {
		p.offset = 0
	}
}

// IsGroupSelected returns true if the cursor is on a group (not a leaf).
func (p *TreePanel) IsGroupSelected() bool {
	if p.cursor < len(p.flatList) {
		return len(p.flatList[p.cursor].Children) > 0
	}
	return false
}

// ToggleSelected toggles expand/collapse if on a group node.
func (p *TreePanel) ToggleSelected() {
	if p.cursor < len(p.flatList) {
		node := p.flatList[p.cursor]
		if len(node.Children) > 0 {
			node.Expanded = !node.Expanded
			p.rebuildFlatList()
		}
	}
}

// View renders the tree panel.
func (p *TreePanel) View(active bool) string {
	contentHeight := p.viewHeight()
	contentWidth := max(1, p.width-2)

	// Build visible content using scroll offset
	var lines []string
	endIdx := min(p.offset+contentHeight, len(p.flatList))

	for i := p.offset; i < endIdx; i++ {
		node := p.flatList[i]
		line := p.renderNode(node, i == p.cursor, active)
		// Truncate to width
		if lipgloss.Width(line) > contentWidth {
			line = line[:contentWidth]
		}
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
		ItemIndex:   p.cursor,
		ItemCount:   len(p.flatList),
		ScrollPos:   p.offset,
		TotalHeight: len(p.flatList),
		ViewHeight:  contentHeight,
	}

	return ui.RenderBorderedPanel(content, p.width, p.height, active, p.styles, cfg)
}

// renderNode renders a single node line.
func (p *TreePanel) renderNode(node *TreeNode, selected, active bool) string {
	// Find depth
	depth := p.nodeDepth(node)

	// Build prefix
	prefix := ""
	for i := 0; i < depth; i++ {
		prefix += "  "
	}

	// Expand indicator for groups
	expandIndicator := "  "
	if len(node.Children) > 0 {
		if node.Expanded {
			expandIndicator = "▼ "
		} else {
			expandIndicator = "▶ "
		}
	}

	// Build the line content
	label := node.Label
	lineWidth := max(1, p.width-2)

	// If this is a light with a color, we need to render the indicator separately
	var line string
	if node.Item != nil && node.Item.IndicatorColor != "" && len(label) > 0 {
		// Split off the first character (brightness indicator) and color it
		runes := []rune(label)
		indicator := string(runes[0])
		rest := string(runes[1:])

		// Build line with colored indicator
		plainLine := prefix + expandIndicator + indicator + rest
		visibleWidth := lipgloss.Width(plainLine)
		padding := ""
		if visibleWidth < lineWidth {
			padding = strings.Repeat(" ", lineWidth-visibleWidth)
		}

		// Apply selection style to everything except the colored indicator
		var style lipgloss.Style
		if selected && active {
			style = p.styles.SelectedItem
		} else {
			style = p.styles.ListItem
		}

		// For the colored indicator, keep its foreground but inherit background from selection
		indicatorStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color(node.Item.IndicatorColor))
		if selected && active {
			// Copy the background from SelectedItem style
			indicatorStyle = indicatorStyle.Background(p.styles.SelectedItem.GetBackground())
		}
		coloredIndicator := indicatorStyle.Render(indicator)

		// Render: prefix + expandIndicator styled, then colored indicator, then rest + padding styled
		styledPrefix := style.Render(prefix + expandIndicator)
		styledRest := style.Render(rest + padding)
		line = styledPrefix + coloredIndicator + styledRest
	} else {
		// No special coloring needed
		line = prefix + expandIndicator + label
		visibleWidth := lipgloss.Width(line)
		if visibleWidth < lineWidth {
			line = line + strings.Repeat(" ", lineWidth-visibleWidth)
		}

		// Determine if this is a leaf node (light, device, scene, etc.) or a folder
		isLeaf := node.Item != nil && (node.Item.Type == EntityLight ||
			node.Item.Type == EntityDevice ||
			node.Item.Type == EntityScene ||
			node.Item.Type == EntityBridge ||
			node.Item.Type == EntityEntertainment)

		var style lipgloss.Style
		if selected && active {
			style = p.styles.SelectedItem
		} else if !isLeaf {
			// Folders/groups are muted
			style = p.styles.Muted.Bold(true)
		} else {
			style = p.styles.ListItem
		}
		line = style.Render(line)
	}

	return line
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
	if p.cursor < len(p.flatList) {
		node := p.flatList[p.cursor]
		if node.Item != nil {
			return node.Item, true
		}
	}
	return nil, false
}

// SelectedNode returns the currently selected node.
func (p *TreePanel) SelectedNode() *TreeNode {
	if p.cursor < len(p.flatList) {
		return p.flatList[p.cursor]
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
	itemIndex := p.offset + (relY - 1)
	if itemIndex >= 0 && itemIndex < len(p.flatList) {
		node := p.flatList[itemIndex]
		wasAlreadySelected := (itemIndex == p.cursor)
		p.cursor = itemIndex

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
