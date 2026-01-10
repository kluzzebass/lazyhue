package panels

import (
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/kluzzebass/lazyhue/internal/hue"
	"github.com/kluzzebass/lazyhue/internal/ui"
)

const minIndicatorVisible = 250 * time.Millisecond

// BridgePanel shows the list of bridges with custom scrolling.
type BridgePanel struct {
	ScrollState          // Embedded scroll state for cursor/offset management
	bridges      []*hue.Bridge
	styles       ui.Styles
	activeBridge string
	panelKey     string
	panelTitle   string
	selectedID   string // Track selection by bridge ID for stability across list changes
	// Polling indicator per bridge: brief flash for minIndicatorVisible
	pollingUntil map[string]time.Time
	// Discovery indicator: brief flash for minIndicatorVisible
	discoveringUntil time.Time
}

// NewBridgePanel creates a new bridge panel.
func NewBridgePanel(styles ui.Styles, panelKey string) *BridgePanel {
	return &BridgePanel{
		styles:       styles,
		panelKey:     panelKey,
		panelTitle:   "Bridges",
		pollingUntil: make(map[string]time.Time),
	}
}

// SetInitialSelection sets the initial selected bridge ID (used to restore selection on startup).
func (p *BridgePanel) SetInitialSelection(bridgeID string) {
	p.selectedID = bridgeID
}

// SetBridges updates the bridge list.
func (p *BridgePanel) SetBridges(bridges []*hue.Bridge, activeBridgeID string) {
	p.bridges = bridges
	p.activeBridge = activeBridgeID

	if len(p.bridges) == 0 {
		p.SetCursor(0)
		p.SetOffset(0)
		p.selectedID = ""
		return
	}

	// Restore cursor position based on selectedID (sticky selection)
	if p.selectedID != "" {
		for i, bridge := range p.bridges {
			if bridge.Info.ID == p.selectedID {
				p.SetCursor(i)
				p.EnsureCursorVisible(len(p.bridges))
				return
			}
		}
	}

	// selectedID not found or empty - clamp cursor and set selectedID
	p.ClampCursor(len(p.bridges))
	cursor := p.Cursor()
	if cursor >= 0 && cursor < len(p.bridges) {
		p.selectedID = p.bridges[cursor].Info.ID
	}
}

// moveCursor moves the cursor by delta and updates selectedID.
func (p *BridgePanel) moveCursor(delta int) {
	p.MoveCursor(delta, len(p.bridges))
	// Update selectedID for sticky selection
	cursor := p.Cursor()
	if cursor >= 0 && cursor < len(p.bridges) {
		p.selectedID = p.bridges[cursor].Info.ID
	}
}

// SetPolling triggers a brief polling indicator flash for a specific bridge.
// If bridgeID is empty, flashes all connected bridges.
func (p *BridgePanel) SetPolling(bridgeID string, polling bool) {
	if polling {
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
	}
}

// SetDiscovering triggers a brief discovery indicator flash.
func (p *BridgePanel) SetDiscovering(discovering bool) {
	if discovering {
		p.discoveringUntil = time.Now().Add(minIndicatorVisible)
	}
}

// isPollingVisible returns true if a bridge's polling indicator should still be shown.
func (p *BridgePanel) isPollingVisible(bridgeID string) bool {
	until, ok := p.pollingUntil[bridgeID]
	return ok && time.Now().Before(until)
}

// isDiscoveringVisible returns true if the discovery indicator should still be shown.
func (p *BridgePanel) isDiscoveringVisible() bool {
	return time.Now().Before(p.discoveringUntil)
}

// SelectedBridge returns the currently selected bridge.
func (p *BridgePanel) SelectedBridge() *hue.Bridge {
	cursor := p.Cursor()
	if cursor >= 0 && cursor < len(p.bridges) {
		return p.bridges[cursor]
	}
	return nil
}

// Update handles input for the bridge panel.
func (p *BridgePanel) Update(msg tea.Msg) tea.Cmd {
	itemCount := len(p.bridges)

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
			p.moveCursor(-p.ViewHeight())
		case key.Matches(msg, key.NewBinding(key.WithKeys("pgdown"))):
			p.moveCursor(p.ViewHeight())
		case key.Matches(msg, key.NewBinding(key.WithKeys("home", "g"))):
			p.MoveToStart()
			if itemCount > 0 {
				p.selectedID = p.bridges[0].Info.ID
			}
		case key.Matches(msg, key.NewBinding(key.WithKeys("end", "G"))):
			p.MoveToEnd(itemCount)
			if itemCount > 0 {
				p.selectedID = p.bridges[p.Cursor()].Info.ID
			}
		}
	}
	return nil
}

// View renders the bridge panel.
func (p *BridgePanel) View(active bool) string {
	contentHeight := p.ViewHeight()
	contentWidth := p.ContentWidth()

	var lines []string

	if len(p.bridges) == 0 {
		lines = append(lines, p.styles.Muted.Render("No bridges"))
		lines = append(lines, "")
		lines = append(lines, p.styles.Muted.Render("Press P to pair"))
	} else {
		start, end := p.VisibleRange(len(p.bridges))
		for i := start; i < end; i++ {
			bridge := p.bridges[i]
			selected := i == p.Cursor()
			line := p.renderBridge(bridge, selected, active, contentWidth)
			lines = append(lines, line)
		}
	}

	// Pad to fill height
	for len(lines) < contentHeight {
		lines = append(lines, "")
	}

	content := lipgloss.JoinVertical(lipgloss.Left, lines...)

	// Discovery indicator (always visible)
	var discoverIndicator string
	if p.isDiscoveringVisible() {
		discoverIndicator = IndicatorOn // Active scanning
	} else {
		discoverIndicator = IndicatorOff // Idle
	}

	cfg := ui.BorderConfig{
		PanelKey:       p.panelKey,
		Title:          p.panelTitle,
		TitleIndicator: discoverIndicator,
		ItemIndex:      p.Cursor(),
		ItemCount:      len(p.bridges),
		ScrollPos:      p.Offset(),
		TotalHeight:    len(p.bridges),
		ViewHeight:     contentHeight,
	}

	return ui.RenderBorderedPanel(content, p.Width(), p.Height(), active, p.styles, cfg)
}

// renderBridge renders a single bridge item.
func (p *BridgePanel) renderBridge(bridge *hue.Bridge, selected, active bool, width int) string {
	// Status indicator
	var indicator string
	var indicatorColor lipgloss.Color

	switch bridge.Status {
	case hue.StatusConnected:
		// Show hollow circle when polling, filled when idle
		if p.isPollingVisible(bridge.Info.ID) {
			indicator = IndicatorOff
		} else {
			indicator = IndicatorOn
		}
		indicatorColor = p.styles.Theme.Foreground
	case hue.StatusConnecting:
		indicator = IndicatorOff
		indicatorColor = p.styles.Theme.Warning
	case hue.StatusPairing:
		indicator = "◐" // Half-filled for pairing in progress
		indicatorColor = p.styles.Theme.Warning
	case hue.StatusDisconnected:
		indicator = IndicatorOff
		indicatorColor = p.styles.Theme.Muted
	case hue.StatusError:
		indicator = "✕" // X for error
		indicatorColor = p.styles.Theme.Error
	default:
		indicator = "?"
		indicatorColor = p.styles.Theme.Muted
	}

	// Use the bridge name, falling back to IP if empty
	name := bridge.Info.Name
	if name == "" {
		name = bridge.Info.IPAddress
	}

	// Check for WiFi connectivity (Bridge Pro)
	wifiMarker := ""
	if wifi := bridge.GetState().GetWifiConnectivity(); len(wifi) > 0 {
		for _, w := range wifi {
			if w.Status == "connected" {
				wifiMarker = " [WiFi]"
				break
			}
		}
	}

	// Add check mark for active bridge
	activeMarker := ""
	if bridge.Info.ID == p.activeBridge {
		activeMarker = " ✓"
	}

	// Calculate padding for full-width background
	plainLine := indicator + " " + name + wifiMarker + activeMarker
	visibleWidth := lipgloss.Width(plainLine)
	padding := ""
	if visibleWidth < width {
		padding = strings.Repeat(" ", width-visibleWidth)
	}

	// Style the line
	var style lipgloss.Style
	if selected && active {
		style = p.styles.SelectedItem
	} else {
		style = p.styles.ListItem
	}

	// Render indicator with color, rest with line style
	indicatorStyle := lipgloss.NewStyle().Foreground(indicatorColor)
	if selected && active {
		indicatorStyle = indicatorStyle.Background(p.styles.SelectedItem.GetBackground())
	}

	coloredIndicator := indicatorStyle.Render(indicator)
	styledRest := style.Render(" " + name + wifiMarker + activeMarker + padding)

	return coloredIndicator + styledRest
}

// Key returns the keyboard shortcut key.
func (p *BridgePanel) Key() string {
	return p.panelKey
}

// Title returns the panel title.
func (p *BridgePanel) Title() string {
	return p.panelTitle
}

// SelectedEntity returns an EntityItem for the selected bridge.
func (p *BridgePanel) SelectedEntity() (*EntityItem, bool) {
	bridge := p.SelectedBridge()
	if bridge == nil {
		return nil, false
	}

	name := bridge.Info.Name
	if name == "" {
		name = bridge.Info.IPAddress
	}

	return &EntityItem{
		ID:     bridge.Info.ID,
		Name:   name,
		Type:   EntityBridge,
		IsOn:   bridge.IsConnected(),
		RawPtr: BridgeData{Bridge: bridge},
	}, true
}

// HasBridges returns true if there are any bridges.
func (p *BridgePanel) HasBridges() bool {
	return len(p.bridges) > 0
}

// Index returns the current cursor position.
func (p *BridgePanel) Index() int {
	return p.Cursor()
}

// HandleClick handles a mouse click at relative coordinates within the panel.
func (p *BridgePanel) HandleClick(relX, relY int) {
	idx := p.ScrollState.HandleClick(relY, len(p.bridges))
	if idx >= 0 {
		// Update selectedID for sticky selection
		p.selectedID = p.bridges[idx].Info.ID
	}
}
