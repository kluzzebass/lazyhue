package panels

import (
	"fmt"
	"io"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/kluzzebass/lazyhue/internal/hue"
	"github.com/kluzzebass/lazyhue/internal/ui"
)

// BridgeItem wraps a bridge for the list component.
type BridgeItem struct {
	Bridge *hue.Bridge
}

func (i BridgeItem) FilterValue() string {
	if i.Bridge != nil {
		return i.Bridge.Info.Name
	}
	return ""
}

// BridgeDelegate renders bridge list items.
type BridgeDelegate struct {
	Styles       ui.Styles
	ActiveBridge string
}

func (d BridgeDelegate) Height() int                             { return 1 }
func (d BridgeDelegate) Spacing() int                            { return 0 }
func (d BridgeDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }

func (d BridgeDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	i, ok := item.(BridgeItem)
	if !ok || i.Bridge == nil {
		return
	}

	bridge := i.Bridge
	
	// Active indicator
	active := " "
	if bridge.Info.ID == d.ActiveBridge {
		active = ">"
	}

	// Status indicator
	var status string
	switch bridge.Status {
	case hue.StatusConnected:
		status = d.Styles.Connected.Render("●")
	case hue.StatusConnecting:
		status = d.Styles.Muted.Render("○")
	case hue.StatusPairing:
		status = d.Styles.Muted.Render("◐")
	case hue.StatusDisconnected:
		status = d.Styles.Disconnected.Render("○")
	case hue.StatusError:
		status = d.Styles.Error.Render("✕")
	default:
		status = d.Styles.Muted.Render("?")
	}

	name := bridge.Info.Name
	if name == "" {
		name = bridge.Info.IPAddress
	}

	// Truncate if needed
	maxLen := 20
	if len(name) > maxLen {
		name = name[:maxLen-1] + "…"
	}

	if index == m.Index() {
		name = d.Styles.SelectedItem.Render(name)
	}

	fmt.Fprintf(w, "%s %s %s", active, status, name)
}

// BridgePanel shows the list of bridges.
type BridgePanel struct {
	list         list.Model
	styles       ui.Styles
	width        int
	height       int
	activeBridge string
}

// NewBridgePanel creates a new bridge panel.
func NewBridgePanel(styles ui.Styles) *BridgePanel {
	delegate := BridgeDelegate{Styles: styles}
	l := list.New([]list.Item{}, delegate, 0, 0)
	l.SetShowTitle(false)
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	l.SetShowHelp(false)
	l.SetShowPagination(false)
	l.InfiniteScrolling = false

	return &BridgePanel{
		list:   l,
		styles: styles,
	}
}

// SetBridges updates the bridge list.
func (p *BridgePanel) SetBridges(bridges []*hue.Bridge, activeBridgeID string) {
	p.activeBridge = activeBridgeID
	
	// Update delegate with active bridge
	p.list.SetDelegate(BridgeDelegate{
		Styles:       p.styles,
		ActiveBridge: activeBridgeID,
	})

	items := make([]list.Item, len(bridges))
	for i, b := range bridges {
		items[i] = BridgeItem{Bridge: b}
	}
	p.list.SetItems(items)
}

// SetSize updates the panel dimensions.
func (p *BridgePanel) SetSize(width, height int) {
	p.width = width
	p.height = height

	// Content area: width minus borders (2), height minus borders (2)
	contentWidth := max(1, width-2)
	contentHeight := max(1, height-2)

	p.list.SetSize(contentWidth, contentHeight)
}

// SelectedBridge returns the currently selected bridge.
func (p *BridgePanel) SelectedBridge() *hue.Bridge {
	item := p.list.SelectedItem()
	if item == nil {
		return nil
	}
	if bi, ok := item.(BridgeItem); ok {
		return bi.Bridge
	}
	return nil
}

// Update handles input for the bridge panel.
func (p *BridgePanel) Update(msg tea.Msg) (*BridgePanel, tea.Cmd) {
	var cmd tea.Cmd
	p.list, cmd = p.list.Update(msg)
	return p, cmd
}

// View renders the bridge panel.
func (p *BridgePanel) View(active bool) string {
	content := p.list.View()
	if len(p.list.Items()) == 0 {
		content = p.styles.Muted.Render("No bridges\n\nPress P to pair")
	}

	cfg := ui.BorderConfig{
		Title:       "[1] Bridges",
		ItemIndex:   p.list.Index(),
		ItemCount:   len(p.list.Items()),
		ScrollPos:   p.list.Index(),
		TotalHeight: len(p.list.Items()),
		ViewHeight:  p.height - 2,
	}

	return ui.RenderBorderedPanel(content, p.width, p.height, active, p.styles, cfg)
}

// HasBridges returns true if there are any bridges.
func (p *BridgePanel) HasBridges() bool {
	return len(p.list.Items()) > 0
}

