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
	bridges      []*hue.Bridge
	styles       ui.Styles
	width        int
	height       int
	activeBridge string
	panelKey     string
	panelTitle   string
	cursor       int
	offset       int
	selectedID   string // Track selection by bridge ID for stability across list changes
	// Polling indicator: brief flash for minIndicatorVisible
	pollingUntil time.Time
	// Discovery indicator: brief flash for minIndicatorVisible
	discoveringUntil time.Time
}

// NewBridgePanel creates a new bridge panel.
func NewBridgePanel(styles ui.Styles, panelKey string) *BridgePanel {
	return &BridgePanel{
		styles:     styles,
		panelKey:   panelKey,
		panelTitle: "Bridges",
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
		p.cursor = 0
		p.offset = 0
		p.selectedID = ""
		return
	}

	// Restore cursor position based on selectedID (sticky selection)
	if p.selectedID != "" {
		for i, bridge := range p.bridges {
			if bridge.Info.ID == p.selectedID {
				p.cursor = i
				p.ensureCursorVisible()
				return
			}
		}
	}

	// selectedID not found or empty - clamp cursor and set selectedID
	if p.cursor >= len(p.bridges) {
		p.cursor = len(p.bridges) - 1
	}
	if p.cursor < 0 {
		p.cursor = 0
	}
	p.selectedID = p.bridges[p.cursor].Info.ID
	p.ensureCursorVisible()
}

// SetPolling triggers a brief polling indicator flash.
func (p *BridgePanel) SetPolling(polling bool) {
	if polling {
		p.pollingUntil = time.Now().Add(minIndicatorVisible)
	}
}

// SetDiscovering triggers a brief discovery indicator flash.
func (p *BridgePanel) SetDiscovering(discovering bool) {
	if discovering {
		p.discoveringUntil = time.Now().Add(minIndicatorVisible)
	}
}

// isPollingVisible returns true if polling indicator should show (brief flash).
func (p *BridgePanel) isPollingVisible() bool {
	return time.Now().Before(p.pollingUntil)
}

// isDiscoveringVisible returns true if discovery indicator should show (brief flash).
func (p *BridgePanel) isDiscoveringVisible() bool {
	return time.Now().Before(p.discoveringUntil)
}

// SetSize updates the panel dimensions.
func (p *BridgePanel) SetSize(width, height int) {
	p.width = width
	p.height = height
}

// viewHeight returns the number of visible lines.
func (p *BridgePanel) viewHeight() int {
	return max(1, p.height-2)
}

// moveCursor moves the cursor by delta and updates selectedID.
func (p *BridgePanel) moveCursor(delta int) {
	if len(p.bridges) == 0 {
		return
	}
	p.cursor += delta
	if p.cursor < 0 {
		p.cursor = 0
	}
	if p.cursor >= len(p.bridges) {
		p.cursor = len(p.bridges) - 1
	}
	// Update selectedID for sticky selection
	p.selectedID = p.bridges[p.cursor].Info.ID
	p.ensureCursorVisible()
}

// ensureCursorVisible adjusts scroll offset.
func (p *BridgePanel) ensureCursorVisible() {
	viewHeight := p.viewHeight()

	if p.cursor < p.offset {
		p.offset = p.cursor
	}
	if p.cursor >= p.offset+viewHeight {
		p.offset = p.cursor - viewHeight + 1
	}

	maxOffset := max(0, len(p.bridges)-viewHeight)
	if p.offset > maxOffset {
		p.offset = maxOffset
	}
	if p.offset < 0 {
		p.offset = 0
	}
}

// SelectedBridge returns the currently selected bridge.
func (p *BridgePanel) SelectedBridge() *hue.Bridge {
	if p.cursor >= 0 && p.cursor < len(p.bridges) {
		return p.bridges[p.cursor]
	}
	return nil
}

// Update handles input for the bridge panel.
func (p *BridgePanel) Update(msg tea.Msg) tea.Cmd {
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
			if len(p.bridges) > 0 {
				p.selectedID = p.bridges[0].Info.ID
			}
		case key.Matches(msg, key.NewBinding(key.WithKeys("end", "G"))):
			p.cursor = max(0, len(p.bridges)-1)
			p.ensureCursorVisible()
			if len(p.bridges) > 0 {
				p.selectedID = p.bridges[p.cursor].Info.ID
			}
		}
	}
	return nil
}

// View renders the bridge panel.
func (p *BridgePanel) View(active bool) string {
	contentHeight := p.viewHeight()
	contentWidth := max(1, p.width-2)

	var lines []string

	if len(p.bridges) == 0 {
		lines = append(lines, p.styles.Muted.Render("No bridges"))
		lines = append(lines, "")
		lines = append(lines, p.styles.Muted.Render("Press P to pair"))
	} else {
		endIdx := min(p.offset+contentHeight, len(p.bridges))
		for i := p.offset; i < endIdx; i++ {
			bridge := p.bridges[i]
			selected := i == p.cursor
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
		ItemIndex:      p.cursor,
		ItemCount:      len(p.bridges),
		ScrollPos:      p.offset,
		TotalHeight:    len(p.bridges),
		ViewHeight:     contentHeight,
	}

	return ui.RenderBorderedPanel(content, p.width, p.height, active, p.styles, cfg)
}

// renderBridge renders a single bridge item.
func (p *BridgePanel) renderBridge(bridge *hue.Bridge, selected, active bool, width int) string {
	// Status indicator
	var indicator string
	var indicatorColor lipgloss.Color

	switch bridge.Status {
	case hue.StatusConnected:
		// Show hollow circle when polling, filled when idle
		if p.isPollingVisible() {
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
	return p.cursor
}

// HandleClick handles a mouse click at relative coordinates within the panel.
func (p *BridgePanel) HandleClick(relX, relY int) {
	// Content starts at y=1 (after top border)
	if relY < 1 {
		return
	}

	// Calculate which bridge was clicked
	itemIndex := p.offset + (relY - 1)
	if itemIndex >= 0 && itemIndex < len(p.bridges) {
		p.cursor = itemIndex
		// Update selectedID for sticky selection
		p.selectedID = p.bridges[p.cursor].Info.ID
	}
}
