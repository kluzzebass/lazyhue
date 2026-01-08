package panels

import (
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/kluzzebass/lazyhue/internal/ui"
)

// TabData represents a single tab's data and scroll state.
type TabData struct {
	ID     string
	Title  string
	Items  []list.Item
	Cursor int
	Offset int
}

// TabbedPanel provides a panel with multiple tabs, each containing a scrollable list.
type TabbedPanel struct {
	tabs       []TabData
	activeTab  int
	styles     ui.Styles
	width      int
	height     int
	panelTitle string
	panelKey   string
}

// NewTabbedPanel creates a new tabbed panel.
func NewTabbedPanel(styles ui.Styles, panelTitle, panelKey string, tabTitles []string) *TabbedPanel {
	tabs := make([]TabData, len(tabTitles))
	for i, title := range tabTitles {
		tabs[i] = TabData{
			ID:    title,
			Title: title,
		}
	}

	return &TabbedPanel{
		tabs:       tabs,
		activeTab:  0,
		styles:     styles,
		panelTitle: panelTitle,
		panelKey:   panelKey,
	}
}

// SetSize updates the panel dimensions.
func (p *TabbedPanel) SetSize(width, height int) {
	p.width = width
	p.height = height
}

// viewHeight returns the number of visible lines.
func (p *TabbedPanel) viewHeight() int {
	return max(1, p.height-2)
}

// SetItems sets items for a specific tab by index.
func (p *TabbedPanel) SetItems(tabIndex int, items []list.Item) {
	if tabIndex >= 0 && tabIndex < len(p.tabs) {
		p.tabs[tabIndex].Items = items
		p.clampTab(tabIndex)
	}
}

// SetItemsByID sets items for a specific tab by ID.
func (p *TabbedPanel) SetItemsByID(tabID string, items []list.Item) {
	for i := range p.tabs {
		if p.tabs[i].ID == tabID {
			p.tabs[i].Items = items
			p.clampTab(i)
			return
		}
	}
}

// clampTab ensures cursor and offset are valid for a tab.
func (p *TabbedPanel) clampTab(tabIndex int) {
	tab := &p.tabs[tabIndex]
	if len(tab.Items) == 0 {
		tab.Cursor = 0
		tab.Offset = 0
	} else if tab.Cursor >= len(tab.Items) {
		tab.Cursor = len(tab.Items) - 1
		p.ensureCursorVisible(tabIndex)
	}
}

// ensureCursorVisible adjusts scroll offset for a tab.
func (p *TabbedPanel) ensureCursorVisible(tabIndex int) {
	tab := &p.tabs[tabIndex]
	viewHeight := p.viewHeight()

	if tab.Cursor < tab.Offset {
		tab.Offset = tab.Cursor
	}
	if tab.Cursor >= tab.Offset+viewHeight {
		tab.Offset = tab.Cursor - viewHeight + 1
	}

	maxOffset := max(0, len(tab.Items)-viewHeight)
	if tab.Offset > maxOffset {
		tab.Offset = maxOffset
	}
	if tab.Offset < 0 {
		tab.Offset = 0
	}
}

// moveCursor moves the cursor in the active tab.
func (p *TabbedPanel) moveCursor(delta int) {
	if p.activeTab < 0 || p.activeTab >= len(p.tabs) {
		return
	}
	tab := &p.tabs[p.activeTab]
	if len(tab.Items) == 0 {
		return
	}

	tab.Cursor += delta
	if tab.Cursor < 0 {
		tab.Cursor = 0
	}
	if tab.Cursor >= len(tab.Items) {
		tab.Cursor = len(tab.Items) - 1
	}
	p.ensureCursorVisible(p.activeTab)
}

// ActiveTabID returns the ID of the currently active tab.
func (p *TabbedPanel) ActiveTabID() string {
	if p.activeTab >= 0 && p.activeTab < len(p.tabs) {
		return p.tabs[p.activeTab].ID
	}
	return ""
}

// NextTab switches to the next tab.
func (p *TabbedPanel) NextTab() {
	if len(p.tabs) > 0 {
		p.activeTab = (p.activeTab + 1) % len(p.tabs)
	}
}

// PrevTab switches to the previous tab.
func (p *TabbedPanel) PrevTab() {
	if len(p.tabs) > 0 {
		p.activeTab = (p.activeTab - 1 + len(p.tabs)) % len(p.tabs)
	}
}

// SetActiveTab sets the active tab by index.
func (p *TabbedPanel) SetActiveTab(index int) {
	if index >= 0 && index < len(p.tabs) {
		p.activeTab = index
	}
}

// SelectedItem returns the currently selected item in the active tab.
func (p *TabbedPanel) SelectedItem() (EntityItem, bool) {
	if p.activeTab < 0 || p.activeTab >= len(p.tabs) {
		return EntityItem{}, false
	}
	tab := &p.tabs[p.activeTab]
	if tab.Cursor >= 0 && tab.Cursor < len(tab.Items) {
		if ei, ok := tab.Items[tab.Cursor].(EntityItem); ok {
			return ei, true
		}
	}
	return EntityItem{}, false
}

// Update handles input for the tabbed panel.
func (p *TabbedPanel) Update(msg tea.Msg) tea.Cmd {
	viewHeight := p.viewHeight()

	switch msg := msg.(type) {
	case tea.MouseMsg:
		switch msg.Button {
		case tea.MouseButtonWheelUp:
			p.moveCursor(-1)
			return nil
		case tea.MouseButtonWheelDown:
			p.moveCursor(1)
			return nil
		}

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, key.NewBinding(key.WithKeys("left", "h"))):
			p.PrevTab()
			return nil
		case key.Matches(msg, key.NewBinding(key.WithKeys("right", "l"))):
			p.NextTab()
			return nil
		case key.Matches(msg, key.NewBinding(key.WithKeys("up", "k"))):
			p.moveCursor(-1)
		case key.Matches(msg, key.NewBinding(key.WithKeys("down", "j"))):
			p.moveCursor(1)
		case key.Matches(msg, key.NewBinding(key.WithKeys("pgup"))):
			p.moveCursor(-viewHeight)
		case key.Matches(msg, key.NewBinding(key.WithKeys("pgdown"))):
			p.moveCursor(viewHeight)
		case key.Matches(msg, key.NewBinding(key.WithKeys("home", "g"))):
			if p.activeTab >= 0 && p.activeTab < len(p.tabs) {
				p.tabs[p.activeTab].Cursor = 0
				p.tabs[p.activeTab].Offset = 0
			}
		case key.Matches(msg, key.NewBinding(key.WithKeys("end", "G"))):
			if p.activeTab >= 0 && p.activeTab < len(p.tabs) {
				tab := &p.tabs[p.activeTab]
				tab.Cursor = max(0, len(tab.Items)-1)
				p.ensureCursorVisible(p.activeTab)
			}
		}
	}

	return nil
}

// Key returns the keyboard shortcut key.
func (p *TabbedPanel) Key() string {
	return p.panelKey
}

// Title returns the panel title.
func (p *TabbedPanel) Title() string {
	return p.panelTitle
}

// SelectedEntity returns the currently selected entity.
func (p *TabbedPanel) SelectedEntity() (*EntityItem, bool) {
	if p.activeTab < 0 || p.activeTab >= len(p.tabs) {
		return nil, false
	}
	tab := &p.tabs[p.activeTab]
	if tab.Cursor >= 0 && tab.Cursor < len(tab.Items) {
		if ei, ok := tab.Items[tab.Cursor].(EntityItem); ok {
			return &ei, true
		}
	}
	return nil, false
}

// View renders the tabbed panel.
func (p *TabbedPanel) View(active bool) string {
	contentHeight := p.viewHeight()
	contentWidth := max(1, p.width-2)

	var lines []string
	itemCount := 0
	itemIndex := 0
	scrollOffset := 0

	if p.activeTab >= 0 && p.activeTab < len(p.tabs) {
		tab := &p.tabs[p.activeTab]
		itemCount = len(tab.Items)
		itemIndex = tab.Cursor
		scrollOffset = tab.Offset

		if itemCount == 0 {
			lines = append(lines, p.styles.Muted.Render("No items"))
		} else {
			endIdx := min(tab.Offset+contentHeight, len(tab.Items))
			for i := tab.Offset; i < endIdx; i++ {
				item := tab.Items[i]
				selected := i == tab.Cursor
				line := p.renderItem(item, selected, active, contentWidth)
				lines = append(lines, line)
			}
		}
	}

	// Pad to fill height
	for len(lines) < contentHeight {
		lines = append(lines, "")
	}

	content := lipgloss.JoinVertical(lipgloss.Left, lines...)

	// Build tab titles for border
	tabTitles := make([]string, len(p.tabs))
	for i, tab := range p.tabs {
		tabTitles[i] = tab.Title
	}

	cfg := ui.BorderConfig{
		PanelKey:    p.panelKey,
		Tabs:        tabTitles,
		ActiveTab:   p.activeTab,
		ItemIndex:   itemIndex,
		ItemCount:   itemCount,
		ScrollPos:   scrollOffset,
		TotalHeight: itemCount,
		ViewHeight:  contentHeight,
	}

	return ui.RenderBorderedPanel(content, p.width, p.height, active, p.styles, cfg)
}

// renderItem renders a single list item.
func (p *TabbedPanel) renderItem(item list.Item, selected, active bool, width int) string {
	ei, ok := item.(EntityItem)
	if !ok {
		return ""
	}

	// Use centralized indicator rendering
	indicator := RenderEntityIndicator(ei, p.styles, selected && active)

	name := ei.Name
	line := indicator + " " + name

	// Truncate if needed
	if lipgloss.Width(line) > width {
		line = line[:width-1] + "…"
	}

	// Style with full-width background
	var style lipgloss.Style
	if selected && active {
		style = p.styles.SelectedItem.Width(width)
	} else {
		style = p.styles.ListItem.Width(width)
	}

	return style.Render(line)
}

// HasItems returns true if the active tab has any items.
func (p *TabbedPanel) HasItems() bool {
	if p.activeTab >= 0 && p.activeTab < len(p.tabs) {
		return len(p.tabs[p.activeTab].Items) > 0
	}
	return false
}

// HandleClick handles a mouse click at relative coordinates within the panel.
func (p *TabbedPanel) HandleClick(relX, relY int) {
	// Check if click is in the top border row (y=0) where tabs are
	if relY == 0 {
		// Calculate tab positions
		// Layout: border + [key] + border + tab1 + sep + tab2 + sep + tab3 + ...
		// The panel key takes about 4 chars "[3]" plus some border
		startX := 5 // Skip "╭[3]─"

		for i, tab := range p.tabs {
			tabWidth := lipgloss.Width(tab.Title)
			endX := startX + tabWidth

			if relX >= startX && relX < endX {
				p.activeTab = i
				return
			}

			// Move past this tab and the separator "─"
			startX = endX + 1
		}
		return
	}

	// Click on content area - select item
	if relY >= 1 && p.activeTab >= 0 && p.activeTab < len(p.tabs) {
		tab := &p.tabs[p.activeTab]
		itemIndex := tab.Offset + (relY - 1)
		if itemIndex >= 0 && itemIndex < len(tab.Items) {
			tab.Cursor = itemIndex
		}
	}
}
