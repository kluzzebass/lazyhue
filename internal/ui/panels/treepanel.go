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

// SetRoots sets the tree data.
func (p *TreePanel) SetRoots(roots []*TreeNode) {
	p.roots = roots
	p.rebuildFlatList()
	// Keep cursor in bounds
	if p.cursor >= len(p.flatList) {
		p.cursor = max(0, len(p.flatList)-1)
	}
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
		Title:       "[" + p.panelKey + "] " + p.panelTitle,
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
	// Selection indicator
	selector := "  "
	if selected {
		selector = "> "
	}

	// Find depth
	depth := p.nodeDepth(node)

	// Build prefix
	prefix := ""
	for i := 0; i < depth; i++ {
		prefix += "  "
	}

	// Expand indicator
	indicator := "  "
	if len(node.Children) > 0 {
		if node.Expanded {
			indicator = "▼ "
		} else {
			indicator = "▶ "
		}
	}

	label := node.Label

	// Style based on selection and whether it's a group
	var style lipgloss.Style
	if selected && active {
		style = p.styles.SelectedItem
	} else if node.Item == nil {
		// Group header
		style = p.styles.Muted.Bold(true)
	} else {
		style = p.styles.ListItem
	}

	return style.Render(selector + prefix + indicator + label)
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
