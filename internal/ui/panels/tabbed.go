package panels

import (
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/kluzzebass/lazyhue/internal/ui"
)

// Tab represents a single tab in a tabbed panel.
type Tab struct {
	ID    string
	Title string
	List  list.Model
}

// TabbedPanel provides a panel with multiple tabs, each containing a list.
type TabbedPanel struct {
	tabs       []Tab
	activeTab  int
	styles     ui.Styles
	width      int
	height     int
	panelTitle string
	panelKey   string // The number key for this panel (e.g., "2")
}

// NewTabbedPanel creates a new tabbed panel.
func NewTabbedPanel(styles ui.Styles, panelTitle, panelKey string, tabTitles []string) *TabbedPanel {
	tabs := make([]Tab, len(tabTitles))
	for i, title := range tabTitles {
		l := list.New([]list.Item{}, EntityDelegate{Styles: styles}, 0, 0)
		l.SetShowTitle(false)
		l.SetShowStatusBar(false)
		l.SetFilteringEnabled(false)
		l.SetShowHelp(false)
		l.SetShowPagination(false)
		l.InfiniteScrolling = false
		tabs[i] = Tab{
			ID:    title,
			Title: title,
			List:  l,
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

	// Content area: width minus borders (2), height minus borders (2)
	contentWidth := max(1, width-2)
	contentHeight := max(1, height-2)

	for i := range p.tabs {
		p.tabs[i].List.SetSize(contentWidth, contentHeight)
	}
}

// SetItems sets items for a specific tab by index.
func (p *TabbedPanel) SetItems(tabIndex int, items []list.Item) {
	if tabIndex >= 0 && tabIndex < len(p.tabs) {
		p.tabs[tabIndex].List.SetItems(items)
	}
}

// SetItemsByID sets items for a specific tab by ID.
func (p *TabbedPanel) SetItemsByID(tabID string, items []list.Item) {
	for i := range p.tabs {
		if p.tabs[i].ID == tabID {
			p.tabs[i].List.SetItems(items)
			return
		}
	}
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
	item := p.tabs[p.activeTab].List.SelectedItem()
	if item == nil {
		return EntityItem{}, false
	}
	if ei, ok := item.(EntityItem); ok {
		return ei, true
	}
	return EntityItem{}, false
}

// Update handles input for the tabbed panel.
func (p *TabbedPanel) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, key.NewBinding(key.WithKeys("left", "h"))):
			p.PrevTab()
			return nil
		case key.Matches(msg, key.NewBinding(key.WithKeys("right", "l"))):
			p.NextTab()
			return nil
		}
	}

	// Pass to active tab's list
	if p.activeTab >= 0 && p.activeTab < len(p.tabs) {
		var cmd tea.Cmd
		p.tabs[p.activeTab].List, cmd = p.tabs[p.activeTab].List.Update(msg)
		return cmd
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
	item := p.tabs[p.activeTab].List.SelectedItem()
	if item == nil {
		return nil, false
	}
	if ei, ok := item.(EntityItem); ok {
		return &ei, true
	}
	return nil, false
}

// View renders the tabbed panel.
func (p *TabbedPanel) View(active bool) string {
	var content string
	itemCount := 0
	itemIndex := 0

	if p.activeTab >= 0 && p.activeTab < len(p.tabs) {
		content = p.tabs[p.activeTab].List.View()
		itemCount = len(p.tabs[p.activeTab].List.Items())
		itemIndex = p.tabs[p.activeTab].List.Index()
	}

	// Build tab titles for border
	tabTitles := make([]string, len(p.tabs))
	for i, tab := range p.tabs {
		tabTitles[i] = tab.Title
	}

	cfg := ui.BorderConfig{
		TabPrefix:   "[" + p.panelKey + "]",
		Tabs:        tabTitles,
		ActiveTab:   p.activeTab,
		ItemIndex:   itemIndex,
		ItemCount:   itemCount,
		ScrollPos:   itemIndex,
		TotalHeight: itemCount,
		ViewHeight:  p.height - 2,
	}

	return ui.RenderBorderedPanel(content, p.width, p.height, active, p.styles, cfg)
}

// HasItems returns true if the active tab has any items.
func (p *TabbedPanel) HasItems() bool {
	if p.activeTab >= 0 && p.activeTab < len(p.tabs) {
		return len(p.tabs[p.activeTab].List.Items()) > 0
	}
	return false
}
