package panels

import (
	"fmt"
	"image/color"
	"io"
	"regexp"
	"strings"

	"github.com/charmbracelet/bubbles/v2/key"
	"github.com/charmbracelet/bubbles/v2/list"
	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/charmbracelet/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	zone "github.com/lrstanley/bubblezone/v2"

	"github.com/kluzzebass/lazyhue/internal/ui"
)

// countSuffixPattern matches " (n)" at the end of labels where n is a number
var countSuffixPattern = regexp.MustCompile(`^(.+?) \((\d+)\)$`)

// splitLabelCount separates a label with a count suffix like "Rooms (3)"
// into the base label "Rooms" and the count "3". Returns empty count if no match.
func splitLabelCount(label string) (base, count string) {
	matches := countSuffixPattern.FindStringSubmatch(label)
	if matches == nil {
		return label, ""
	}
	return matches[1], matches[2]
}

// TreeNode represents a node in the hierarchical tree.
type TreeNode struct {
	ID           string      // Unique identifier for the node
	Label        string      // Display label
	GroupSuffix  string      // Optional group suffix like "(Room)" - rendered in secondary color
	GroupType    EntityType  // Type of group (EntityRoom or EntityZone) for suffix coloring
	BridgeSuffix string      // Optional bridge suffix like "[Bridge]" - rendered dimmed
	Item         *EntityItem // nil for group headers/folders
	Children     []*TreeNode // Child nodes
	Expanded     bool        // Whether this node's children are visible
	Depth        int         // Nesting depth (0 = root)
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
	Styles ui.Styles
	Zones  *zone.Manager
	Width  int // Available width for truncation
}

// Height returns the height of a single item.
func (d TreeDelegate) Height() int { return 1 }

// Spacing returns the spacing between items.
func (d TreeDelegate) Spacing() int { return 0 }

// Update handles item-level updates.
func (d TreeDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }

// getEntityColor returns the theme color for a given entity type.
func (d TreeDelegate) getEntityColor(entityType EntityType) color.Color {
	switch entityType {
	case EntityLight:
		return d.Styles.Theme.EntityLight
	case EntityBridge:
		return d.Styles.Theme.EntityBridge
	case EntityRoom:
		return d.Styles.Theme.EntityRoom
	case EntityZone:
		return d.Styles.Theme.EntityZone
	case EntityDevice:
		return d.Styles.Theme.EntityDevice
	case EntityScene, EntitySmartScene:
		return d.Styles.Theme.EntityScene
	case EntityZoneScene, EntityZoneSmartScene:
		return d.Styles.Theme.EntityZoneScene
	case EntityEntertainment:
		return d.Styles.Theme.EntityEntertainment
	// Category folders use neutral white
	case EntityLightsCategory, EntityDevicesCategory, EntityScenesCategory, EntitySmartScenesCategory,
		EntityRoomsCategory, EntityZonesCategory, EntityEntertainmentCategory:
		return d.Styles.Theme.TextBright
	default:
		return d.Styles.Theme.TextBright
	}
}

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
		// Get the entity type color
		entityColor := d.getEntityColor(node.Item.Type)

		// Special handling for bridges
		if node.Item.Type == EntityBridge {
			if node.Item.IndicatorColor != "" {
				// Blinking state (message received) - use indicator color (warning)
				indicator = lipgloss.NewStyle().Foreground(lipgloss.Color(node.Item.IndicatorColor)).Render("●") + " "
			} else if node.Item.IsOn {
				// Connected - use green/success color
				indicator = d.Styles.Success.Render("●") + " "
			} else {
				// Disconnected
				indicator = d.Styles.Dimmed.Render("○") + " "
			}
		} else if node.Item.IsOn {
			// Use RenderBrightnessIndicator for lights with brightness and color
			if node.Item.Brightness > 0 {
				if node.Item.IndicatorColor != "" {
					// Lights use their actual color
					indicator = ui.RenderBrightnessIndicatorFromHex(node.Item.Brightness, node.Item.IndicatorColor) + " "
				} else {
					// Use entity type color
					indicator = ui.RenderBrightnessIndicator(node.Item.Brightness, entityColor) + " "
				}
			} else {
				// Fallback for on items without brightness
				if node.Item.IndicatorColor != "" {
					indicator = lipgloss.NewStyle().
						Foreground(lipgloss.Color(node.Item.IndicatorColor)).
						Render("●") + " "
				} else {
					indicator = lipgloss.NewStyle().Foreground(entityColor).Render("●") + " "
				}
			}
		} else {
			indicator = d.Styles.Dimmed.Render("○") + " "
		}
	}

	// Build the label - split off count suffix if present
	baseLabel, countStr := splitLabelCount(node.Label)
	label := baseLabel
	countSuffix := ""
	if countStr != "" {
		// Format count with dimmed brackets: [n]
		countSuffix = " " + d.Styles.Dimmed.Render("[") + countStr + d.Styles.Dimmed.Render("]")
	}

	suffix := ""
	if node.GroupSuffix != "" {
		// Group (room/zone) suffix - dimmed brackets, entity color content
		// GroupSuffix is "(RoomName)" so strip the parens and re-add as brackets with styling
		content := strings.TrimPrefix(strings.TrimSuffix(node.GroupSuffix, ")"), "(")
		// Use GroupType to determine color (default to room if not set)
		groupColor := d.Styles.Theme.EntityRoom
		if node.GroupType == EntityZone {
			groupColor = d.Styles.Theme.EntityZone
		}
		groupStyle := lipgloss.NewStyle().Foreground(groupColor)
		suffix += " " + d.Styles.Dimmed.Render("[") + groupStyle.Render(content) + d.Styles.Dimmed.Render("]")
	}
	if node.BridgeSuffix != "" {
		// Bridge suffix - dimmed brackets, entity color content
		// BridgeSuffix is "[BridgeName]" so strip the brackets and re-add with styling
		content := strings.TrimPrefix(strings.TrimSuffix(node.BridgeSuffix, "]"), "[")
		bridgeStyle := lipgloss.NewStyle().Foreground(d.Styles.Theme.EntityBridge)
		suffix += " " + d.Styles.Dimmed.Render("[") + bridgeStyle.Render(content) + d.Styles.Dimmed.Render("]")
	}
	if isSelected {
		label = d.Styles.Selected.Render(label) + countSuffix
	} else if node.Item == nil {
		// Group headers get a different style
		label = d.Styles.Title.Render(label) + countSuffix
	} else {
		// Apply entity type color to label
		entityColor := d.getEntityColor(node.Item.Type)
		label = lipgloss.NewStyle().Foreground(entityColor).Render(label) + countSuffix
	}

	// Compose the line
	line := indent + prefix + indicator + label + suffix

	// Truncate if line exceeds available width
	if d.Width > 0 && lipgloss.Width(line) > d.Width {
		line = ansi.Truncate(line, d.Width, "…")
	}

	// Wrap in a clickable zone if we have a zone manager
	if d.Zones != nil {
		zoneID := ui.TreeItemZone(node.ID)
		line = d.Zones.Mark(zoneID, line)
	}

	fmt.Fprint(w, line)
}

// TreePanel displays a hierarchical tree of items using list.Model.
type TreePanel struct {
	list     list.Model
	delegate TreeDelegate
	styles   ui.Styles
	zones    *zone.Manager
	title    string
	key      string

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
func NewTreePanel(styles ui.Styles, zones *zone.Manager, title, panelKey string) *TreePanel {
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
		list:     l,
		delegate: delegate,
		styles:   styles,
		zones:    zones,
		title:    title,
		key:      panelKey,
		tabs:     []string{"Home", "Lights", "Devices", "Scenes"},
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

	// Update delegate width for truncation
	p.delegate.Width = innerWidth
	p.list.SetDelegate(p.delegate)
}

// MinContentWidth returns the minimum width needed to display the tab bar,
// including borders. This is the absolute minimum usable width for the panel.
func (p *TreePanel) MinContentWidth() int {
	// Calculate tab bar width: "[n]-Tab1-Tab2-Tab3-..."
	tabBarWidth := 0
	if p.key != "" {
		tabBarWidth = 3 + 1 // "[n]-"
	}
	for i, tab := range p.tabs {
		tabBarWidth += len(tab)
		if i < len(p.tabs)-1 {
			tabBarWidth += 1 // "-" separator
		}
	}
	// Add corners (2) + left padding (1)
	tabBarWidth += 3
	return tabBarWidth
}

// MaxContentWidth returns the maximum width needed to display the tree content
// when fully expanded, including borders. This is used to cap the panel width.
func (p *TreePanel) MaxContentWidth() int {
	maxWidth := 0

	// Calculate tab bar width: "[n]-Tab1-Tab2-Tab3-..."
	// Key "[n]" = 3 chars, then "-" separator, then tabs joined by "-"
	tabBarWidth := 0
	if p.key != "" {
		tabBarWidth = 3 + 1 // "[n]-"
	}
	for i, tab := range p.tabs {
		tabBarWidth += len(tab)
		if i < len(p.tabs)-1 {
			tabBarWidth += 1 // "-" separator
		}
	}
	// Add corners (2) + left padding (1)
	tabBarWidth += 3

	// Calculate max width recursively for all nodes
	var calcNodeWidth func(node *TreeNode, depth int)
	calcNodeWidth = func(node *TreeNode, depth int) {
		// Width components:
		// - indent: 2 chars per depth level
		// - prefix: 2 chars (expand/collapse indicator)
		// - indicator: 2 chars (on/off dot + space)
		// - label: text length
		// - suffixes: group suffix + bridge suffix
		indent := depth * 2
		prefix := 2
		indicator := 2

		// Calculate label width (strip any count suffix styling)
		baseLabel, countStr := splitLabelCount(node.Label)
		labelWidth := len(baseLabel)
		if countStr != "" {
			labelWidth += 2 + len(countStr) + 1 // " [n]"
		}

		suffixWidth := 0
		if node.GroupSuffix != "" {
			// " [GroupName]" - strip parens, add brackets
			content := strings.TrimPrefix(strings.TrimSuffix(node.GroupSuffix, ")"), "(")
			suffixWidth += 2 + len(content) + 1 // " []" + content
		}
		if node.BridgeSuffix != "" {
			// " [BridgeName]" - strip brackets, re-add
			content := strings.TrimPrefix(strings.TrimSuffix(node.BridgeSuffix, "]"), "[")
			suffixWidth += 2 + len(content) + 1 // " []" + content
		}

		nodeWidth := indent + prefix + indicator + labelWidth + suffixWidth
		if nodeWidth > maxWidth {
			maxWidth = nodeWidth
		}

		// Recurse to children
		for _, child := range node.Children {
			calcNodeWidth(child, depth+1)
		}
	}

	// Process all root nodes
	for _, root := range p.roots {
		calcNodeWidth(root, 0)
	}

	// Return the maximum of tab bar width or content width, plus borders
	contentWidth := maxWidth + 2
	if tabBarWidth > contentWidth {
		return tabBarWidth
	}
	return contentWidth
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

	case tea.MouseWheelMsg:
		// Bubbles v2 list doesn't handle mouse wheel, so we convert to cursor movement
		switch msg.Button {
		case tea.MouseWheelUp:
			p.list.CursorUp()
		case tea.MouseWheelDown:
			p.list.CursorDown()
		}
		return p, nil

	case tea.MouseClickMsg:
		// Check for tab clicks first
		if p.zones != nil {
			for i := range p.tabs {
				zoneID := ui.TabZone(i)
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
				zoneID := ui.TreeItemZone(f.Node.ID)
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
	innerWidth := p.width - 2  // Account for left+right border
	innerHeight := p.height - 2 // Account for top border (with tabs) + bottom border
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
	bottomBorder := borderStyleColor.Render(border.BottomLeft) +
		borderStyleColor.Render(strings.Repeat(border.Bottom, innerWidth)) +
		borderStyleColor.Render(border.BottomRight)

	// Calculate scrollbar - estimate scroll position from cursor
	totalItems := len(p.flatList)
	visibleItems := innerHeight
	scrollPos := 0
	if totalItems > visibleItems {
		// Estimate scroll position: cursor tends to be in the middle of the view
		// The list keeps selection visible, so scroll offset is roughly cursor - half visible
		cursorPos := p.list.Index()
		halfVisible := visibleItems / 2
		scrollPos = cursorPos - halfVisible
		if scrollPos < 0 {
			scrollPos = 0
		}
		maxScroll := totalItems - visibleItems
		if scrollPos > maxScroll {
			scrollPos = maxScroll
		}
	}
	rightBorders := ui.BuildRightBorderWithScrollbar(border, innerHeight, borderColor, scrollPos, totalItems, visibleItems)

	// Build panel
	var lines []string
	lines = append(lines, topBorder)

	// Content lines with side borders
	for i, line := range contentLines {
		// Truncate line if too long using ansi.Truncate which handles ANSI codes properly
		lineWidth := lipgloss.Width(line)
		if lineWidth > innerWidth {
			line = ansi.Truncate(line, innerWidth, "…")
			lineWidth = lipgloss.Width(line)
		}
		// Pad to fill width
		if lineWidth < innerWidth {
			line = line + strings.Repeat(" ", innerWidth-lineWidth)
		}
		rightBorder := borderStyleColor.Render(border.Right)
		if i < len(rightBorders) {
			rightBorder = rightBorders[i]
		}
		lines = append(lines, leftBorder+line+rightBorder)
	}

	// Fill to exact height (1 for top border + content + 1 for bottom border)
	targetHeight := p.height
	contentIdx := len(contentLines)
	for len(lines) < targetHeight-1 {
		paddedLine := strings.Repeat(" ", innerWidth)
		rightBorder := borderStyleColor.Render(border.Right)
		if contentIdx < len(rightBorders) {
			rightBorder = rightBorders[contentIdx]
		}
		lines = append(lines, leftBorder+paddedLine+rightBorder)
		contentIdx++
	}

	lines = append(lines, bottomBorder)

	return strings.Join(lines, "\n")
}

// renderTabbedBorder renders the top border with tabs embedded.
func (p *TreePanel) renderTabbedBorder(focused bool, borderColor color.Color) string {
	border := lipgloss.RoundedBorder()
	borderStyle := lipgloss.NewStyle().Foreground(borderColor)

	// Target inner width (between corners)
	innerWidth := p.width - 2
	if innerWidth < 1 {
		innerWidth = 1
	}

	// Build key prefix (e.g., "[2]")
	keyRendered := ""
	if p.key != "" {
		keyStyle := lipgloss.NewStyle().
			Foreground(p.styles.Theme.Primary).
			Bold(true)
		keyRendered = keyStyle.Render("[" + p.key + "]")
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
			zoneID := ui.TabZone(i)
			tabRendered = p.zones.Mark(zoneID, tabRendered)
		}
		tabParts = append(tabParts, tabRendered)
		if i < len(p.tabs)-1 {
			sepStyle := lipgloss.NewStyle().Foreground(borderColor)
			tabParts = append(tabParts, sepStyle.Render(border.Top))
		}
	}
	tabString := strings.Join(tabParts, "")

	// Calculate border segments
	leftPadding := 1  // after TopLeft
	middlePadding := 1 // between key and tabs

	// Build the content part (key + tabs with padding)
	var content string
	if keyRendered != "" {
		content = borderStyle.Render(strings.Repeat(border.Top, leftPadding)) +
			keyRendered +
			borderStyle.Render(strings.Repeat(border.Top, middlePadding)) +
			tabString
	} else {
		content = borderStyle.Render(strings.Repeat(border.Top, leftPadding)) +
			tabString
	}
	contentWidth := lipgloss.Width(content)

	// If content is wider than available space, truncate it
	if contentWidth > innerWidth {
		content = ansi.Truncate(content, innerWidth, "")
		contentWidth = lipgloss.Width(content)
	}

	// Fill remaining width with border
	remainingWidth := innerWidth - contentWidth
	if remainingWidth < 0 {
		remainingWidth = 0
	}

	return borderStyle.Render(border.TopLeft) +
		content +
		borderStyle.Render(strings.Repeat(border.Top, remainingWidth)) +
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
