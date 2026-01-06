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
	items      []list.Item
	delegate   list.ItemDelegate
	styles     ui.Styles
	width      int
	height     int
	panelTitle string
	panelKey   string
	emptyText  string
	cursor     int
	offset     int
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

// SetSize updates the panel dimensions.
func (p *ListPanel) SetSize(width, height int) {
	p.width = width
	p.height = height
}

// SetItems sets the list items.
func (p *ListPanel) SetItems(items []list.Item) {
	p.items = items
	// Clamp cursor
	if len(p.items) == 0 {
		p.cursor = 0
		p.offset = 0
	} else if p.cursor >= len(p.items) {
		p.cursor = len(p.items) - 1
		p.ensureCursorVisible()
	}
}

// viewHeight returns the number of visible lines.
func (p *ListPanel) viewHeight() int {
	return max(1, p.height-2)
}

// moveCursor moves the cursor by delta.
func (p *ListPanel) moveCursor(delta int) {
	if len(p.items) == 0 {
		return
	}
	p.cursor += delta
	if p.cursor < 0 {
		p.cursor = 0
	}
	if p.cursor >= len(p.items) {
		p.cursor = len(p.items) - 1
	}
	p.ensureCursorVisible()
}

// ensureCursorVisible adjusts scroll offset.
func (p *ListPanel) ensureCursorVisible() {
	viewHeight := p.viewHeight()

	if p.cursor < p.offset {
		p.offset = p.cursor
	}
	if p.cursor >= p.offset+viewHeight {
		p.offset = p.cursor - viewHeight + 1
	}

	maxOffset := max(0, len(p.items)-viewHeight)
	if p.offset > maxOffset {
		p.offset = maxOffset
	}
	if p.offset < 0 {
		p.offset = 0
	}
}

// Update handles input for the list panel.
func (p *ListPanel) Update(msg tea.Msg) tea.Cmd {
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
		case key.Matches(msg, key.NewBinding(key.WithKeys("up", "k"))):
			p.moveCursor(-1)
		case key.Matches(msg, key.NewBinding(key.WithKeys("down", "j"))):
			p.moveCursor(1)
		case key.Matches(msg, key.NewBinding(key.WithKeys("pgup"))):
			p.moveCursor(-viewHeight)
		case key.Matches(msg, key.NewBinding(key.WithKeys("pgdown"))):
			p.moveCursor(viewHeight)
		case key.Matches(msg, key.NewBinding(key.WithKeys("home", "g"))):
			p.cursor = 0
			p.offset = 0
		case key.Matches(msg, key.NewBinding(key.WithKeys("end", "G"))):
			p.cursor = max(0, len(p.items)-1)
			p.ensureCursorVisible()
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
	if p.cursor >= 0 && p.cursor < len(p.items) {
		if ei, ok := p.items[p.cursor].(EntityItem); ok {
			return &ei, true
		}
	}
	return nil, false
}

// View renders the list panel.
func (p *ListPanel) View(active bool) string {
	contentHeight := p.viewHeight()
	contentWidth := max(1, p.width-2)

	var lines []string

	if len(p.items) == 0 {
		lines = append(lines, p.styles.Muted.Render(p.emptyText))
	} else {
		endIdx := min(p.offset+contentHeight, len(p.items))
		for i := p.offset; i < endIdx; i++ {
			item := p.items[i]
			selected := i == p.cursor

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
		ItemIndex:   p.cursor,
		ItemCount:   len(p.items),
		ScrollPos:   p.offset,
		TotalHeight: len(p.items),
		ViewHeight:  contentHeight,
	}

	return ui.RenderBorderedPanel(content, p.width, p.height, active, p.styles, cfg)
}

// renderItem renders a single list item.
func (p *ListPanel) renderItem(item list.Item, selected, active bool, width int) string {
	ei, ok := item.(EntityItem)
	if !ok {
		return ""
	}

	// Status indicator
	indicator := "○"
	if ei.IsOn {
		indicator = "●"
	}

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

// HasItems returns true if the list has any items.
func (p *ListPanel) HasItems() bool {
	return len(p.items) > 0
}

// Index returns the currently selected index.
func (p *ListPanel) Index() int {
	return p.cursor
}

// SetDelegate updates the item delegate (kept for API compatibility).
func (p *ListPanel) SetDelegate(delegate list.ItemDelegate) {
	p.delegate = delegate
}
