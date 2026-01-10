package panels

import (
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/kluzzebass/lazyhue/internal/ui"
)

// ListPanel provides a simple panel with a scrollable list.
// Uses custom scrolling logic instead of bubbles/list pagination.
type ListPanel struct {
	ScrollState        // Embedded scroll state for cursor/offset management
	items      []list.Item
	delegate   list.ItemDelegate
	styles     ui.Styles
	panelTitle string
	panelKey   string
	emptyText  string
}

// NewListPanel creates a new list panel.
func NewListPanel(styles ui.Styles, panelTitle, panelKey, emptyText string, delegate list.ItemDelegate) *ListPanel {
	return &ListPanel{
		styles:     styles,
		panelTitle: panelTitle,
		panelKey:   panelKey,
		emptyText:  emptyText,
		delegate:   delegate,
	}
}

// SetItems sets the list items.
func (p *ListPanel) SetItems(items []list.Item) {
	p.items = items
	p.ClampCursor(len(p.items))
}

// Update handles input for the list panel.
func (p *ListPanel) Update(msg tea.Msg) tea.Cmd {
	itemCount := len(p.items)

	switch msg := msg.(type) {
	case tea.MouseMsg:
		switch msg.Button {
		case tea.MouseButtonWheelUp:
			p.MoveCursor(-1, itemCount)
			return nil
		case tea.MouseButtonWheelDown:
			p.MoveCursor(1, itemCount)
			return nil
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
		case key.Matches(msg, key.NewBinding(key.WithKeys("home", "g"))):
			p.MoveToStart()
		case key.Matches(msg, key.NewBinding(key.WithKeys("end", "G"))):
			p.MoveToEnd(itemCount)
		}
	}
	return nil
}

// Key returns the keyboard shortcut key.
func (p *ListPanel) Key() string {
	return p.panelKey
}

// Title returns the panel title.
func (p *ListPanel) Title() string {
	return p.panelTitle
}

// SelectedEntity returns the currently selected entity.
func (p *ListPanel) SelectedEntity() (*EntityItem, bool) {
	cursor := p.Cursor()
	if cursor >= 0 && cursor < len(p.items) {
		if ei, ok := p.items[cursor].(EntityItem); ok {
			return &ei, true
		}
	}
	return nil, false
}

// View renders the list panel.
func (p *ListPanel) View(active bool) string {
	contentHeight := p.ViewHeight()
	contentWidth := p.ContentWidth()

	var lines []string

	if len(p.items) == 0 {
		lines = append(lines, p.styles.Muted.Render(p.emptyText))
	} else {
		start, end := p.VisibleRange(len(p.items))
		for i := start; i < end; i++ {
			item := p.items[i]
			selected := i == p.Cursor()

			// Render item
			line := p.renderItem(item, selected, active, contentWidth)
			lines = append(lines, line)
		}
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
		ItemCount:   len(p.items),
		ScrollPos:   p.Offset(),
		TotalHeight: len(p.items),
		ViewHeight:  contentHeight,
	}

	return ui.RenderBorderedPanel(content, p.Width(), p.Height(), active, p.styles, cfg)
}

// renderItem renders a single list item.
func (p *ListPanel) renderItem(item list.Item, selected, active bool, width int) string {
	ei, ok := item.(EntityItem)
	if !ok {
		return ""
	}
	return RenderEntityLine(ei, width, selected, active, p.styles)
}

// HasItems returns true if the list has any items.
func (p *ListPanel) HasItems() bool {
	return len(p.items) > 0
}

// Index returns the currently selected index.
func (p *ListPanel) Index() int {
	return p.Cursor()
}

// SetDelegate updates the item delegate (kept for API compatibility).
func (p *ListPanel) SetDelegate(delegate list.ItemDelegate) {
	p.delegate = delegate
}

// HandleClick handles a mouse click at relative coordinates within the panel.
func (p *ListPanel) HandleClick(relX, relY int) {
	p.ScrollState.HandleClick(relY, len(p.items))
}
