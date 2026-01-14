package panels

import (
	"fmt"
	"image/color"
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
		// Special handling for bridges - show blink when Brightness > 0
		if node.Item.Type == EntityBridge {
			if node.Item.Brightness > 0 {
				// Blinking state - use bright accent color
				indicator = d.Styles.Accent.Render("◉") + " "
			} else if node.Item.IsOn {
				// Connected but not blinking
				indicator = d.Styles.Success.Render("●") + " "
			} else {
				// Disconnected
				indicator = d.Styles.Dimmed.Render("○") + " "
			}
		} else if node.Item.IsOn {
			// Use RenderBrightnessIndicator for lights with brightness and color
			if node.Item.Brightness > 0 {
				if node.Item.IndicatorColor != "" {
					indicator = ui2.RenderBrightnessIndicatorFromHex(node.Item.Brightness, node.Item.IndicatorColor) + " "
				} else {
					// Default to theme success color if no indicator color
					indicator = ui2.RenderBrightnessIndicator(node.Item.Brightness, d.Styles.Theme.Success) + " "
				}
			} else {
				// Fallback for on items without brightness
				if node.Item.IndicatorColor != "" {
					indicator = lipgloss.NewStyle().
						Foreground(lipgloss.Color(node.Item.IndicatorColor)).
						Render("●") + " "
				} else {
					indicator = d.Styles.Success.Render("●") + " "
				}
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

	// Tabs
	tabs      []string
	activeTab int

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
		tabs:   []string{"Home", "Lights", "Devices", "Scenes"},
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
		case key.Matches(msg, key.NewBinding(key.WithKeys("tab"))):
			p.NextTab()
			return p, nil
		case key.Matches(msg, key.NewBinding(key.WithKeys("shift+tab"))):
			p.PrevTab()
			return p, nil
		case key.Matches(msg, key.NewBinding(key.WithKeys("left", "h"))):
			// Only collapse in Home tab, otherwise switch tabs
			if p.activeTab == 0 {
				p.Collapse()
			} else {
				p.PrevTab()
			}
			return p, nil
		case key.Matches(msg, key.NewBinding(key.WithKeys("right", "l"))):
			// Only expand in Home tab, otherwise switch tabs
			if p.activeTab == 0 {
				p.Expand()
			} else {
				p.NextTab()
			}
			return p, nil
		case key.Matches(msg, key.NewBinding(key.WithKeys("enter", " "))):
			// Enter on a node with children toggles expanded
			// Enter on a node with an item should be handled by app layer (navigate to details)
			node := p.SelectedNode()
			if node != nil && len(node.Children) > 0 {
				p.ToggleExpanded()
				return p, nil
			}
			// If node has an item, let app layer handle it (will navigate to details)
			// Pass through to list so app can catch it
		}

	case tea.MouseClickMsg:
		// Check for tab clicks first
		if p.zones != nil {
			for i := range p.tabs {
				zoneID := ui2.TabZone(i)
				if p.zones.Get(zoneID).InBounds(msg) {
					if msg.Button == tea.MouseLeft {
						p.SetActiveTab(i)
						// Return a special message that app layer can handle to rebuild tree
						return p, nil
					}
				}
			}
		}

		// Check for tree item zone clicks
		if p.zones != nil {
			for i, f := range p.flatList {
				zoneID := ui2.TreeItemZone(f.Node.ID)
				if p.zones.Get(zoneID).InBounds(msg) {
					wasSelected := p.list.Index() == i
					p.list.Select(i)
					if msg.Button == tea.MouseLeft {
						// Click on expand icon toggles
						if f.HasChildren {
							p.ToggleExpanded()
						} else if f.Node.Item != nil && wasSelected {
							// Double-click (click on already selected item) or special handling
							// For now, just select - app layer can handle navigation on Enter
							// We could add a special message here, but let's keep it simple
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
	// Determine border color
	borderColor := p.styles.Theme.Border
	if focused {
		borderColor = p.styles.Theme.Accent
	}

	// Build top border with tabs
	topBorder := p.renderTabbedBorder(focused, borderColor)

	// Render the list content
	content := p.list.View()

	// Calculate dimensions
	innerWidth := p.width - 2   // Account for left+right border
	innerHeight := p.height - 3 // Account for top border (with tabs), bottom border, and content
	if innerWidth < 1 {
		innerWidth = 1
	}
	if innerHeight < 1 {
		innerHeight = 1
	}

	// Constrain content height
	contentLines := strings.Split(content, "\n")
	// Remove trailing empty line if present (list may add trailing newline)
	if len(contentLines) > 0 && contentLines[len(contentLines)-1] == "" {
		contentLines = contentLines[:len(contentLines)-1]
	}
	if len(contentLines) > innerHeight {
		contentLines = contentLines[:innerHeight]
	}

	// Get border characters
	border := lipgloss.RoundedBorder()
	borderStyleColor := lipgloss.NewStyle().Foreground(borderColor)

	// Build sides and bottom
	leftBorder := borderStyleColor.Render(border.Left)
	rightBorder := borderStyleColor.Render(border.Right)
	bottomBorder := borderStyleColor.Render(border.BottomLeft) +
		borderStyleColor.Render(strings.Repeat(border.Bottom, innerWidth)) +
		borderStyleColor.Render(border.BottomRight)

	// Build panel
	var lines []string
	lines = append(lines, topBorder)

	// Content lines with side borders
	for _, line := range contentLines {
		// Truncate line if too long
		if lipgloss.Width(line) > innerWidth {
			line = lipgloss.Place(innerWidth, 1, lipgloss.Left, lipgloss.Top, line)
		}
		paddedLine := lipgloss.Place(innerWidth, 1, lipgloss.Left, lipgloss.Top, line)
		lines = append(lines, leftBorder+paddedLine+rightBorder)
	}

	// Fill to exact height (1 for top border + content + 1 for bottom border)
	targetHeight := p.height
	for len(lines) < targetHeight-1 {
		paddedLine := strings.Repeat(" ", innerWidth)
		lines = append(lines, leftBorder+paddedLine+rightBorder)
	}

	lines = append(lines, bottomBorder)

	return strings.Join(lines, "\n")
}

// renderTabbedBorder renders the top border with tabs embedded.
func (p *TreePanel) renderTabbedBorder(focused bool, borderColor color.Color) string {
	border := lipgloss.RoundedBorder()

	// Build key prefix (e.g., "[2]")
	keyRendered := ""
	keyWidth := 0
	if p.key != "" {
		keyStyle := lipgloss.NewStyle().
			Foreground(p.styles.Theme.Primary).
			Bold(true)
		keyRendered = keyStyle.Render("[" + p.key + "]")
		keyWidth = lipgloss.Width(keyRendered)
	}

	// Build tabs with clickable zones
	var tabParts []string
	for i, tab := range p.tabs {
		var tabStyle lipgloss.Style
		if i == p.activeTab {
			tabStyle = lipgloss.NewStyle().
				Foreground(p.styles.Theme.Primary).
				Bold(true)
		} else {
			tabStyle = lipgloss.NewStyle().
				Foreground(p.styles.Theme.TextMuted)
		}
		tabRendered := tabStyle.Render(tab)
		// Mark tab as clickable zone
		if p.zones != nil {
			zoneID := ui2.TabZone(i)
			tabRendered = p.zones.Mark(zoneID, tabRendered)
		}
		tabParts = append(tabParts, tabRendered)
		if i < len(p.tabs)-1 {
			sepStyle := lipgloss.NewStyle().Foreground(borderColor)
			tabParts = append(tabParts, sepStyle.Render(border.Top))
		}
	}
	tabString := strings.Join(tabParts, "")
	tabWidth := lipgloss.Width(tabString)

	// Calculate border segments
	leftPadding := 1                                                                  // after TopLeft
	middlePadding := 1                                                                // between key and tabs
	remainingWidth := p.width - keyWidth - tabWidth - leftPadding - middlePadding - 2 // -2 for corners

	if remainingWidth < 0 {
		remainingWidth = 0
	}

	borderStyle := lipgloss.NewStyle().Foreground(borderColor)

	if keyRendered != "" {
		return borderStyle.Render(border.TopLeft) +
			borderStyle.Render(strings.Repeat(border.Top, leftPadding)) +
			keyRendered +
			borderStyle.Render(strings.Repeat(border.Top, middlePadding)) +
			tabString +
			borderStyle.Render(strings.Repeat(border.Top, remainingWidth)) +
			borderStyle.Render(border.TopRight)
	}

	return borderStyle.Render(border.TopLeft) +
		borderStyle.Render(strings.Repeat(border.Top, leftPadding)) +
		tabString +
		borderStyle.Render(strings.Repeat(border.Top, remainingWidth+middlePadding)) +
		borderStyle.Render(border.TopRight)
}

// Title returns the panel title.
func (p *TreePanel) Title() string { return p.title }

// SetTitle sets the panel title.
func (p *TreePanel) SetTitle(title string) { p.title = title }

// Key returns the panel hotkey.
func (p *TreePanel) Key() string { return p.key }

// SetKey sets the panel hotkey.
func (p *TreePanel) SetKey(key string) { p.key = key }

// ActiveTabIndex returns the active tab index.
func (p *TreePanel) ActiveTabIndex() int {
	return p.activeTab
}

// SetActiveTab sets the active tab index.
func (p *TreePanel) SetActiveTab(tab int) {
	if tab >= 0 && tab < len(p.tabs) {
		p.activeTab = tab
		p.title = p.tabs[tab]
	}
}

// NextTab switches to the next tab.
func (p *TreePanel) NextTab() {
	if len(p.tabs) == 0 {
		return
	}
	p.activeTab = (p.activeTab + 1) % len(p.tabs)
	p.title = p.tabs[p.activeTab]
}

// PrevTab switches to the previous tab.
func (p *TreePanel) PrevTab() {
	if len(p.tabs) == 0 {
		return
	}
	p.activeTab = (p.activeTab - 1 + len(p.tabs)) % len(p.tabs)
	p.title = p.tabs[p.activeTab]
}

// ActiveTabID returns the active tab ID (for compatibility with app layer).
func (p *TreePanel) ActiveTabID() string {
	if p.activeTab >= 0 && p.activeTab < len(p.tabs) {
		switch p.tabs[p.activeTab] {
		case "Home":
			return "home"
		case "Lights":
			return "lights"
		case "Devices":
			return "devices"
		case "Scenes":
			return "scenes"
		}
	}
	return "home"
}

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

// FindEntity searches the tree for a node with the given entity type and ID.
// Returns nil if not found.
func (p *TreePanel) FindEntity(entityType, entityID string) *TreeNode {
	// Search all roots
	for _, root := range p.roots {
		if node := findEntityInNode(root, entityType, entityID); node != nil {
			return node
		}
	}
	return nil
}

// findEntityInNode recursively searches a node and its children for an entity.
func findEntityInNode(node *TreeNode, entityType, entityID string) *TreeNode {
	// Check this node
	if node.Item != nil && node.Item.Type.String() == entityType && node.Item.ID == entityID {
		return node
	}

	// Search children
	for _, child := range node.Children {
		if found := findEntityInNode(child, entityType, entityID); found != nil {
			return found
		}
	}

	return nil
}

// SelectNode selects a specific node in the tree.
// If the node is not currently visible (parent collapsed), expands ancestors to make it visible.
func (p *TreePanel) SelectNode(node *TreeNode) {
	// First, expand all ancestors to make this node visible
	expandAncestors(node, p.roots)

	// Rebuild flat list with ancestors expanded
	p.rebuildFlatList()

	// Find the node in the flat list and select it
	for i, flat := range p.flatList {
		if flat.Node == node {
			p.list.Select(i)
			return
		}
	}
}

// expandAncestors recursively expands all ancestors of the target node.
func expandAncestors(target *TreeNode, roots []*TreeNode) bool {
	for _, root := range roots {
		if expandAncestorsInNode(target, root) {
			return true
		}
	}
	return false
}

// expandAncestorsInNode checks if target is a descendant of node, and if so, expands the path to it.
func expandAncestorsInNode(target *TreeNode, node *TreeNode) bool {
	// Check if this node is the target
	if node == target {
		return true
	}

	// Check children
	for _, child := range node.Children {
		if expandAncestorsInNode(target, child) {
			// Found in this subtree - expand this node
			node.Expanded = true
			return true
		}
	}

	return false
}

// GetExpandedStates returns the current expanded state map for persistence.
func (p *TreePanel) GetExpandedStates() map[string]bool {
	return p.getExpandedState()
}

// SetExpandedStates restores the expanded state from a map.
func (p *TreePanel) SetExpandedStates(state map[string]bool) {
	p.restoreExpandedState(state)
	p.rebuildFlatList()
}

// SelectByID selects a node by entity ID. Returns true if found and selected.
// If the node is in a collapsed branch, expands ancestors to make it visible.
func (p *TreePanel) SelectByID(entityID string) bool {
	// Search through full tree (not just flat list) to find the node
	node := p.findNodeByItemID(entityID)
	if node == nil {
		return false
	}

	// Use SelectNode which handles expanding ancestors
	p.SelectNode(node)
	return true
}

// findNodeByItemID searches the full tree for a node with matching Item.ID.
func (p *TreePanel) findNodeByItemID(entityID string) *TreeNode {
	var search func(nodes []*TreeNode) *TreeNode
	search = func(nodes []*TreeNode) *TreeNode {
		for _, node := range nodes {
			if node.Item != nil && node.Item.ID == entityID {
				return node
			}
			if len(node.Children) > 0 {
				if found := search(node.Children); found != nil {
					return found
				}
			}
		}
		return nil
	}
	return search(p.roots)
}
