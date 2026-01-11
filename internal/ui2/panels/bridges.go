package panels

import (
	"fmt"
	"io"
	"time"

	"github.com/charmbracelet/bubbles/v2/key"
	"github.com/charmbracelet/bubbles/v2/list"
	"github.com/charmbracelet/bubbles/v2/spinner"
	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/charmbracelet/lipgloss/v2"
	zone "github.com/lrstanley/bubblezone/v2"

	"github.com/kluzzebass/lazyhue/internal/hue"
	"github.com/kluzzebass/lazyhue/internal/ui2"
)

const minIndicatorVisible = 250 * time.Millisecond

// BridgeItem represents a bridge in the list.
type BridgeItem struct {
	Bridge       *hue.Bridge
	IsActive     bool
	IsPolling    bool
	IsDiscovered bool
}

// FilterValue implements list.Item.
func (i BridgeItem) FilterValue() string {
	if i.Bridge != nil && i.Bridge.Info.Name != "" {
		return i.Bridge.Info.Name
	}
	return ""
}

// Title returns the bridge name.
func (i BridgeItem) Title() string {
	if i.Bridge == nil {
		return "Unknown"
	}
	if i.Bridge.Info.Name != "" {
		return i.Bridge.Info.Name
	}
	return i.Bridge.Info.ID
}

// Description returns status info.
func (i BridgeItem) Description() string {
	if i.Bridge == nil {
		return ""
	}
	return i.Bridge.Info.IPAddress
}

// BridgeDelegate handles rendering of bridge items.
type BridgeDelegate struct {
	Styles  ui2.Styles
	Zones   *zone.Manager
	Spinner spinner.Model
}

// Height returns the height of a bridge item.
func (d BridgeDelegate) Height() int { return 1 }

// Spacing returns the spacing between items.
func (d BridgeDelegate) Spacing() int { return 0 }

// Update handles item-level updates.
func (d BridgeDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }

// Render renders a single bridge item.
func (d BridgeDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	bi, ok := item.(BridgeItem)
	if !ok {
		return
	}

	isSelected := index == m.Index()
	bridge := bi.Bridge
	if bridge == nil {
		return
	}

	// Status indicator
	var statusIndicator string
	switch bridge.Status {
	case hue.StatusConnected:
		if bi.IsPolling {
			statusIndicator = d.Styles.Warning.Render("◉") // Polling indicator
		} else {
			statusIndicator = d.Styles.Success.Render("●") // Connected
		}
	case hue.StatusConnecting:
		statusIndicator = d.Styles.Warning.Render("○") // Connecting
	case hue.StatusDisconnected:
		statusIndicator = d.Styles.Error.Render("○") // Disconnected
	case hue.StatusPairing:
		statusIndicator = d.Styles.Warning.Render("⏳") // Pairing
	default:
		statusIndicator = d.Styles.Dimmed.Render("?")
	}

	// Active bridge indicator
	activeIndicator := "  "
	if bi.IsActive {
		activeIndicator = d.Styles.Accent.Render("▶ ")
	}

	// Name
	name := bridge.Info.Name
	if name == "" {
		name = bridge.Info.ID
	}
	if isSelected {
		name = d.Styles.Selected.Render(name)
	}

	// Compose the line
	line := fmt.Sprintf("%s%s %s", activeIndicator, statusIndicator, name)

	// Wrap in a clickable zone
	if d.Zones != nil {
		zoneID := ui2.BridgeItemZone(bridge.Info.ID)
		line = d.Zones.Mark(zoneID, line)
	}

	fmt.Fprint(w, line)
}

// BridgePanel displays the list of bridges.
type BridgePanel struct {
	list    list.Model
	styles  ui2.Styles
	zones   *zone.Manager
	spinner spinner.Model
	title   string

	// Bridge data
	bridges      []*hue.Bridge
	activeBridge string
	pollingUntil map[string]time.Time

	// Discovery state
	discovering      bool
	discoveringUntil time.Time

	// Dimensions
	width  int
	height int
}

// NewBridgePanel creates a new bridge panel.
func NewBridgePanel(styles ui2.Styles, zones *zone.Manager) *BridgePanel {
	s := spinner.New()
	s.Spinner = spinner.Dot

	delegate := BridgeDelegate{
		Styles:  styles,
		Zones:   zones,
		Spinner: s,
	}

	l := list.New([]list.Item{}, delegate, 0, 0)
	l.SetShowTitle(false)
	l.SetShowStatusBar(false)
	l.SetShowFilter(false)
	l.SetShowHelp(false)
	l.SetShowPagination(false)
	l.DisableQuitKeybindings()

	return &BridgePanel{
		list:         l,
		styles:       styles,
		zones:        zones,
		spinner:      s,
		title:        "Bridges",
		pollingUntil: make(map[string]time.Time),
	}
}

// SetBridges updates the bridge list.
func (p *BridgePanel) SetBridges(bridges []*hue.Bridge, activeBridgeID string) {
	p.bridges = bridges
	p.activeBridge = activeBridgeID

	// Convert to list items
	items := make([]list.Item, len(bridges))
	for i, b := range bridges {
		items[i] = BridgeItem{
			Bridge:    b,
			IsActive:  b.Info.ID == activeBridgeID,
			IsPolling: p.isPollingVisible(b.Info.ID),
		}
	}
	p.list.SetItems(items)
}

// isPollingVisible returns true if the polling indicator should be shown.
func (p *BridgePanel) isPollingVisible(bridgeID string) bool {
	if until, ok := p.pollingUntil[bridgeID]; ok {
		return time.Now().Before(until)
	}
	return false
}

// SetPolling triggers a brief polling indicator for a bridge.
func (p *BridgePanel) SetPolling(bridgeID string) {
	if bridgeID == "" {
		// Flash all connected bridges
		for _, b := range p.bridges {
			if b.Status == hue.StatusConnected {
				p.pollingUntil[b.Info.ID] = time.Now().Add(minIndicatorVisible)
			}
		}
	} else {
		p.pollingUntil[bridgeID] = time.Now().Add(minIndicatorVisible)
	}
	// Refresh items to show indicator
	p.SetBridges(p.bridges, p.activeBridge)
}

// SetDiscovering sets the discovering state.
func (p *BridgePanel) SetDiscovering(discovering bool) {
	p.discovering = discovering
	if discovering {
		p.discoveringUntil = time.Now().Add(minIndicatorVisible)
	}
}

// IsDiscovering returns true if discovery is in progress or indicator is visible.
func (p *BridgePanel) IsDiscovering() bool {
	return p.discovering || time.Now().Before(p.discoveringUntil)
}

// SelectedBridge returns the currently selected bridge.
func (p *BridgePanel) SelectedBridge() *hue.Bridge {
	if p.list.Index() < 0 || p.list.Index() >= len(p.bridges) {
		return nil
	}
	return p.bridges[p.list.Index()]
}

// SetSize sets the panel dimensions.
func (p *BridgePanel) SetSize(width, height int) {
	p.width = width
	p.height = height
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
func (p *BridgePanel) Update(msg tea.Msg) (*BridgePanel, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, key.NewBinding(key.WithKeys("enter", " "))):
			// Select bridge
			if bridge := p.SelectedBridge(); bridge != nil {
				// Return a command to activate this bridge
				return p, nil // App will handle Enter key
			}
		}

	case tea.MouseClickMsg:
		// Check for zone clicks
		if p.zones != nil {
			for i, b := range p.bridges {
				zoneID := ui2.BridgeItemZone(b.Info.ID)
				if p.zones.Get(zoneID).InBounds(msg) {
					p.list.Select(i)
					return p, nil
				}
			}
		}

	case spinner.TickMsg:
		if p.IsDiscovering() {
			var cmd tea.Cmd
			p.spinner, cmd = p.spinner.Update(msg)
			cmds = append(cmds, cmd)
		}
	}

	// Pass to list
	var cmd tea.Cmd
	p.list, cmd = p.list.Update(msg)
	if cmd != nil {
		cmds = append(cmds, cmd)
	}

	return p, tea.Batch(cmds...)
}

// View renders the panel.
func (p *BridgePanel) View(focused bool) string {
	borderStyle := p.styles.Panel
	if focused {
		borderStyle = p.styles.PanelActive
	}

	// Title with optional discovery indicator
	title := p.title
	if p.IsDiscovering() {
		title += " " + p.spinner.View()
	}

	titleStyle := lipgloss.NewStyle().
		Foreground(p.styles.Theme.Primary).
		Bold(true)

	content := titleStyle.Render(title) + "\n" + p.list.View()

	return borderStyle.
		Width(p.width).
		Height(p.height).
		Render(content)
}

// Index returns the current selection index.
func (p *BridgePanel) Index() int {
	return p.list.Index()
}

// Select sets the selection index.
func (p *BridgePanel) Select(index int) {
	p.list.Select(index)
}

// Len returns the number of bridges.
func (p *BridgePanel) Len() int {
	return len(p.bridges)
}
