package panels

import (
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/kluzzebass/lazyhue/internal/ui"
)

// ListPanel provides a simple panel with a single list.
type ListPanel struct {
	list       list.Model
	styles     ui.Styles
	width      int
	height     int
	panelTitle string
	panelKey   string // The number key for this panel (e.g., "1")
	emptyText  string // Text to show when list is empty
}

// NewListPanel creates a new list panel.
func NewListPanel(styles ui.Styles, panelTitle, panelKey, emptyText string, delegate list.ItemDelegate) *ListPanel {
	l := list.New([]list.Item{}, delegate, 0, 0)
	l.SetShowTitle(false)
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	l.SetShowHelp(false)
	l.SetShowPagination(false)
	l.InfiniteScrolling = false

	return &ListPanel{
		list:       l,
		styles:     styles,
		panelTitle: panelTitle,
		panelKey:   panelKey,
		emptyText:  emptyText,
	}
}

// SetSize updates the panel dimensions.
func (p *ListPanel) SetSize(width, height int) {
	p.width = width
	p.height = height

	// Content area: width minus borders (2), height minus borders (2)
	contentWidth := max(1, width-2)
	contentHeight := max(1, height-2)

	p.list.SetSize(contentWidth, contentHeight)
}

// SetItems sets the list items.
func (p *ListPanel) SetItems(items []list.Item) {
	p.list.SetItems(items)
}

// SelectedItem returns the currently selected item.
func (p *ListPanel) SelectedItem() (list.Item, bool) {
	item := p.list.SelectedItem()
	return item, item != nil
}

// SelectedEntityItem returns the currently selected item as an EntityItem.
func (p *ListPanel) SelectedEntityItem() (EntityItem, bool) {
	item := p.list.SelectedItem()
	if item == nil {
		return EntityItem{}, false
	}
	if ei, ok := item.(EntityItem); ok {
		return ei, true
	}
	return EntityItem{}, false
}

// Update handles input for the list panel.
func (p *ListPanel) Update(msg tea.Msg) (*ListPanel, tea.Cmd) {
	var cmd tea.Cmd
	p.list, cmd = p.list.Update(msg)
	return p, cmd
}

// View renders the list panel.
func (p *ListPanel) View(active bool) string {
	var content string
	if len(p.list.Items()) == 0 {
		content = p.styles.Muted.Render(p.emptyText)
	} else {
		content = p.list.View()
	}

	cfg := ui.BorderConfig{
		Title:       "[" + p.panelKey + "] " + p.panelTitle,
		ItemIndex:   p.list.Index(),
		ItemCount:   len(p.list.Items()),
		ScrollPos:   p.list.Index(),
		TotalHeight: len(p.list.Items()),
		ViewHeight:  p.height - 2,
	}

	return ui.RenderBorderedPanel(content, p.width, p.height, active, p.styles, cfg)
}

// HasItems returns true if the list has any items.
func (p *ListPanel) HasItems() bool {
	return len(p.list.Items()) > 0
}

// SetDelegate updates the list delegate.
func (p *ListPanel) SetDelegate(delegate list.ItemDelegate) {
	p.list.SetDelegate(delegate)
}

// Index returns the currently selected index.
func (p *ListPanel) Index() int {
	return p.list.Index()
}

