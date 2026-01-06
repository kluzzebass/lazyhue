package app

import (
	"context"
	"sort"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/kluzzebass/lazyhue/internal/config"
	"github.com/kluzzebass/lazyhue/internal/hue"
	"github.com/kluzzebass/lazyhue/internal/ui"
	"github.com/kluzzebass/lazyhue/internal/ui/layout"
	"github.com/kluzzebass/lazyhue/internal/ui/panels"
)

// Panel IDs for layout tree
const (
	PanelIDBridges = "bridges"
	PanelIDScenes  = "scenes"
	PanelIDGroups  = "groups"
	PanelIDLights  = "lights"
	PanelIDDevices = "devices"
	PanelIDDetail  = "detail"
	PanelIDStatus  = "status"
)

// Model is the main Bubble Tea model for the application.
type Model struct {
	// Configuration
	config      *config.Config
	credentials *config.CredentialStore

	// Bridge management
	manager *hue.Manager

	// UI panels indexed by ID
	panelMap       map[string]panels.Panel
	panelOrder     []string // Panel IDs in focus-cycle order
	focusIndex     int      // Index into panelOrder, -1 for detail panel
	lastFocusIndex int      // Previous focus index (for returning from detail)
	detailPanel    *panels.DetailsPanel
	statusBar      *panels.StatusBar
	styles         ui.Styles
	keys           ui.KeyMap

	// Layout tree
	layoutTree *layout.Tree

	// Current selection
	selectedItem *panels.EntityItem

	// Window dimensions
	width  int
	height int

	// State
	ready      bool
	quitting   bool
	pairing    bool
	pairingFor *hue.BridgeInfo
	statusMsg  string
	isError    bool
}

// New creates a new application model.
func New(cfg *config.Config, creds *config.CredentialStore) Model {
	styles := ui.DefaultStyles()
	keys := ui.DefaultKeyMap()

	// Create panels indexed by ID
	panelMap := map[string]panels.Panel{
		PanelIDBridges: panels.NewBridgePanel(styles, "1"),
		PanelIDScenes:  panels.NewTreePanel(styles, "Scenes", "2"),
		PanelIDGroups:  panels.NewTabbedPanel(styles, "Groups", "3", []string{"Rooms", "Zones", "Entertainment"}),
		PanelIDLights:  panels.NewListPanel(styles, "Lights", "4", "No lights", panels.EntityDelegate{Styles: styles}),
		PanelIDDevices: panels.NewListPanel(styles, "Devices", "5", "No devices", panels.EntityDelegate{Styles: styles}),
	}

	// Panel order for keyboard focus cycling (reorder here!)
	panelOrder := []string{
		PanelIDBridges,
		PanelIDScenes,
		PanelIDGroups,
		PanelIDLights,
		PanelIDDevices,
	}

	// Build layout tree - this defines visual structure
	// To reorder visually, change the tree structure here
	layoutTree := layout.NewTree(
		layout.VSplit(
			// Main content area
			layout.Child{Size: layout.Flex(1), Node: layout.HSplit(
				// Left column (40%)
				layout.Child{Size: layout.Flex(0.4), Node: layout.VSplit(
					layout.Child{Size: layout.Fixed(5), Node: layout.NewLeaf(PanelIDBridges)},
					layout.Child{Size: layout.Flex(1), Node: layout.NewLeaf(PanelIDScenes)},
					layout.Child{Size: layout.Flex(1), Node: layout.NewLeaf(PanelIDGroups)},
					layout.Child{Size: layout.Flex(1), Node: layout.NewLeaf(PanelIDLights)},
					layout.Child{Size: layout.Flex(1), Node: layout.NewLeaf(PanelIDDevices)},
				)},
				// Right column (60%)
				layout.Child{Size: layout.Flex(0.6), Node: layout.NewLeaf(PanelIDDetail)},
			)},
			// Status bar at bottom
			layout.Child{Size: layout.Fixed(1), Node: layout.NewLeaf(PanelIDStatus)},
		),
	)

	return Model{
		config:      cfg,
		credentials: creds,
		manager:     hue.NewManager(creds),
		styles:      styles,
		keys:        keys,
		panelMap:    panelMap,
		panelOrder:  panelOrder,
		focusIndex:  0,
		layoutTree:  layoutTree,
		detailPanel: panels.NewDetailsPanel(styles),
		statusBar:   panels.NewStatusBar(styles, keys),
	}
}

// Panel accessors for type-specific operations
func (m *Model) bridgePanel() *panels.BridgePanel {
	return m.panelMap[PanelIDBridges].(*panels.BridgePanel)
}

func (m *Model) scenesPanel() *panels.TreePanel {
	return m.panelMap[PanelIDScenes].(*panels.TreePanel)
}

func (m *Model) groupsPanel() *panels.TabbedPanel {
	return m.panelMap[PanelIDGroups].(*panels.TabbedPanel)
}

func (m *Model) lightsPanel() *panels.ListPanel {
	return m.panelMap[PanelIDLights].(*panels.ListPanel)
}

func (m *Model) devicesPanel() *panels.ListPanel {
	return m.panelMap[PanelIDDevices].(*panels.ListPanel)
}

func (m *Model) focusedOnDetail() bool {
	return m.focusIndex < 0
}

func (m *Model) focusedPanelID() string {
	if m.focusIndex >= 0 && m.focusIndex < len(m.panelOrder) {
		return m.panelOrder[m.focusIndex]
	}
	return PanelIDDetail
}

func (m *Model) focusedPanel() panels.Panel {
	if m.focusIndex >= 0 && m.focusIndex < len(m.panelOrder) {
		return m.panelMap[m.panelOrder[m.focusIndex]]
	}
	return nil
}

// Init initializes the application.
func (m Model) Init() tea.Cmd {
	cmds := []tea.Cmd{
		tea.SetWindowTitle("lazyhue"),
	}

	// Try stored credentials first for fast startup
	if len(m.credentials.Bridges) > 0 {
		cmds = append(cmds, m.connectFromStoredCredentials())
	}

	// Run discovery immediately (via tick with 0 delay)
	cmds = append(cmds, tea.Tick(0, func(t time.Time) tea.Msg {
		return DiscoveryTickMsg{}
	}))

	// Start sync ticker (will poll when a bridge is connected)
	cmds = append(cmds, startSyncTicker())

	return tea.Batch(cmds...)
}

// connectFromStoredCredentials attempts to connect using saved credentials.
func (m *Model) connectFromStoredCredentials() tea.Cmd {
	return func() tea.Msg {
		for _, cred := range m.credentials.Bridges {
			info := hue.BridgeInfo{
				ID:        cred.BridgeID,
				Name:      cred.BridgeName,
				IPAddress: cred.IPAddress,
				Host:      cred.IPAddress,
			}
			bridge := m.manager.AddBridge(info)
			if err := bridge.Connect(cred.ApiKey); err == nil {
				return BridgeConnectedMsg{BridgeID: cred.BridgeID}
			}
		}
		return nil
	}
}

// Update handles messages.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.updateLayout()
		m.ready = true

	case tea.KeyMsg:
		if m.quitting {
			return m, nil
		}
		cmd := m.handleKey(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}

	case tea.MouseMsg:
		// Handle click for focus switching
		cmd := m.handleMouse(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
		// Also pass to focused panel for scroll wheel support
		cmd = m.updateFocusedPanel(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}

	case BridgesDiscoveredMsg:
		m.handleBridgesDiscovered(msg)
		m.updateBridgePanel()

	case BridgeConnectedMsg:
		m.setStatus("Connected to bridge", false)
		m.updateBridgePanel()
		cmds = append(cmds, syncBridgeState(m.manager.GetBridge(msg.BridgeID)))
		cmds = append(cmds, startSyncTicker())

	case BridgeDisconnectedMsg:
		m.updateBridgePanel()
		if msg.Err != nil {
			m.setStatus("Bridge disconnected: "+msg.Err.Error(), true)
		}

	case StateSyncedMsg:
		// Update stored bridge name if it changed
		if bridge := m.manager.GetBridge(msg.BridgeID); bridge != nil {
			if cred, ok := m.credentials.Get(msg.BridgeID); ok {
				if cred.BridgeName != bridge.Info.Name && bridge.Info.Name != "" {
					cred.BridgeName = bridge.Info.Name
					m.credentials.Set(cred)
					_ = m.credentials.Save()
				}
			}
		}
		m.refreshAllPanels()
		m.syncSelectionFromFocusedPanel()
		m.clearStatus()

	case LightsSyncedMsg:
		m.refreshLightsPanel()
		m.refreshGroupsPanel() // Room on/off status depends on lights
		m.updateDetailPanel()  // Detail view might show light info

	case StateSyncErrorMsg:
		m.setStatus("Sync error: "+msg.Err.Error(), true)

	case SyncTickMsg:
		bridge := m.manager.GetActiveBridge()
		if bridge != nil && bridge.IsConnected() {
			m.statusBar.SetPolling(true)
			cmds = append(cmds, syncLightsAndGroups(bridge))
			// Schedule a redraw after indicator duration to turn it off
			cmds = append(cmds, tea.Tick(300*time.Millisecond, func(t time.Time) tea.Msg {
				return indicatorRefreshMsg{}
			}))
		}
		cmds = append(cmds, startSyncTicker())

	case DiscoveryTickMsg:
		m.statusBar.SetDiscovering(true)
		cmds = append(cmds, discoverBridges(), startDiscoveryTicker())
		// Schedule a redraw after indicator duration to turn it off
		cmds = append(cmds, tea.Tick(300*time.Millisecond, func(t time.Time) tea.Msg {
			return indicatorRefreshMsg{}
		}))

	case indicatorRefreshMsg:
		// Just triggers a redraw to update indicator visibility

	case PairingSuccessMsg:
		m.pairing = false
		m.pairingFor = nil
		m.handlePairingSuccess(msg)
		m.updateBridgePanel()
		cmds = append(cmds, connectBridge(m.manager.GetBridge(msg.BridgeID), msg.ApiKey))

	case PairingFailedMsg:
		m.pairing = false
		m.pairingFor = nil
		m.setStatus("Pairing failed: "+msg.Err.Error(), true)
		m.updateBridgePanel()

	case ActionErrorMsg:
		m.setStatus(msg.Action+": "+msg.Err.Error(), true)

	case StatusMsg:
		m.setStatus(msg.Message, msg.IsError)

	case ClearStatusMsg:
		m.clearStatus()
	}

	// Update focused panel
	cmd := m.updateFocusedPanel(msg)
	if cmd != nil {
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m *Model) updateFocusedPanel(msg tea.Msg) tea.Cmd {
	if m.focusedOnDetail() {
		var cmd tea.Cmd
		m.detailPanel, cmd = m.detailPanel.Update(msg)
		return cmd
	}

	if panel := m.focusedPanel(); panel != nil {
		cmd := panel.Update(msg)
		// Update selection for detail panel
		if item, ok := panel.SelectedEntity(); ok {
			m.selectedItem = item
			m.updateDetailPanel()
		}
		return cmd
	}

	return nil
}

func (m *Model) handleKey(msg tea.KeyMsg) tea.Cmd {
	switch {
	case key.Matches(msg, m.keys.Quit):
		m.quitting = true
		return tea.Quit

	// Panel focus with number keys - check each panel's key
	case key.Matches(msg, m.keys.FocusDetail):
		if m.focusIndex >= 0 {
			m.lastFocusIndex = m.focusIndex
		}
		m.focusIndex = -1 // Detail panel
	default:
		// Check if key matches any panel's shortcut
		for i, id := range m.panelOrder {
			if panel := m.panelMap[id]; panel != nil {
				if msg.String() == panel.Key() {
					m.focusIndex = i
					m.syncSelectionFromFocusedPanel()
					return nil
				}
			}
		}
	}

	switch {
	case key.Matches(msg, m.keys.NextPane):
		// Cycle only through left panels (skip detail)
		if m.focusIndex < 0 {
			m.focusIndex = 0
		} else {
			m.focusIndex++
			if m.focusIndex >= len(m.panelOrder) {
				m.focusIndex = 0
			}
		}
		m.syncSelectionFromFocusedPanel()

	case key.Matches(msg, m.keys.PrevPane):
		// Cycle only through left panels (skip detail)
		if m.focusIndex < 0 {
			m.focusIndex = len(m.panelOrder) - 1
		} else {
			m.focusIndex--
			if m.focusIndex < 0 {
				m.focusIndex = len(m.panelOrder) - 1
			}
		}
		m.syncSelectionFromFocusedPanel()

	case key.Matches(msg, key.NewBinding(key.WithKeys("esc"))):
		// Escape from detail panel goes back to previous panel
		if m.focusIndex < 0 {
			m.focusIndex = m.lastFocusIndex
			if m.focusIndex < 0 || m.focusIndex >= len(m.panelOrder) {
				m.focusIndex = 0
			}
			m.syncSelectionFromFocusedPanel()
		}

	case key.Matches(msg, m.keys.Select):
		return m.handleSelect()

	case key.Matches(msg, m.keys.PairBridge):
		return m.startBridgePairing()

	case key.Matches(msg, m.keys.Toggle):
		return m.toggleSelected()

	case key.Matches(msg, m.keys.TurnOn):
		return m.setSelectedOn(true)

	case key.Matches(msg, m.keys.TurnOff):
		return m.setSelectedOn(false)

	case key.Matches(msg, m.keys.BrightnessUp):
		return m.adjustBrightness(10)

	case key.Matches(msg, m.keys.BrightnessDown):
		return m.adjustBrightness(-10)

	case key.Matches(msg, m.keys.NextBridge):
		m.manager.NextBridge()
		m.updateBridgePanel()
		m.refreshAllPanels()

	case key.Matches(msg, m.keys.PrevBridge):
		m.manager.PrevBridge()
		m.updateBridgePanel()
		m.refreshAllPanels()

	case key.Matches(msg, m.keys.Refresh):
		bridge := m.manager.GetActiveBridge()
		if bridge != nil {
			m.setStatus("Refreshing...", false)
			return syncBridgeState(bridge)
		}
	}

	return nil
}

func (m *Model) handleSelect() tea.Cmd {
	// Bridge panel: select bridge
	if m.focusedPanelID() == PanelIDBridges {
		if bridge := m.bridgePanel().SelectedBridge(); bridge != nil {
			m.manager.SetActiveBridge(bridge.Info.ID)
			m.updateBridgePanel()
			m.refreshAllPanels()
			// Move to next panel in order
			if m.focusIndex < len(m.panelOrder)-1 {
				m.focusIndex++
			}
		}
		return nil
	}

	// Scene panel: toggle group or activate scene
	if m.focusedPanelID() == PanelIDScenes {
		sp := m.scenesPanel()
		if sp.IsGroupSelected() {
			sp.ToggleSelected()
			return nil
		}
	}

	// Activate scene if one is selected
	if m.selectedItem != nil && m.selectedItem.Type == panels.EntityScene {
		bridge := m.manager.GetActiveBridge()
		if bridge != nil {
			return recallScene(bridge, m.selectedItem.ID)
		}
	}
	return nil
}

func (m *Model) handleMouse(msg tea.MouseMsg) tea.Cmd {
	if msg.Action != tea.MouseActionPress {
		return nil
	}

	// Use layout tree to find clicked panel
	if leaf := m.layoutTree.At(msg.X, msg.Y); leaf != nil {
		if leaf.ID == PanelIDDetail {
			if m.focusIndex >= 0 {
				m.lastFocusIndex = m.focusIndex
			}
			m.focusIndex = -1
		} else {
			// Find index in panelOrder
			for i, id := range m.panelOrder {
				if id == leaf.ID {
					m.focusIndex = i
					break
				}
			}
		}
		m.syncSelectionFromFocusedPanel()
	}
	return nil
}

func (m *Model) handleBridgesDiscovered(msg BridgesDiscoveredMsg) {
	if len(msg.Bridges) == 0 {
		if m.manager.BridgeCount() == 0 {
			m.setStatus("No bridges found. Press P to pair.", false)
		}
		return
	}

	for _, info := range msg.Bridges {
		// Check if we already have this bridge by ID
		existing := m.manager.GetBridge(info.ID)
		if existing != nil {
			// Update IP if changed
			if existing.Info.IPAddress != info.IPAddress {
				existing.Info.IPAddress = info.IPAddress
				existing.Info.Host = info.IPAddress
				if cred, ok := m.credentials.Get(info.ID); ok {
					cred.IPAddress = info.IPAddress
					m.credentials.Set(cred)
					_ = m.credentials.Save()
				}
			}
			// Update name if discovery provides one
			if info.Name != "" && existing.Info.Name != info.Name {
				existing.Info.Name = info.Name
			}
			continue
		}

		// Also check by IP to avoid duplicates from discovery vs stored credentials
		existsByIP := false
		for _, bridge := range m.manager.AllBridges() {
			if bridge.Info.IPAddress == info.IPAddress {
				existsByIP = true
				break
			}
		}
		if existsByIP {
			continue
		}

		m.manager.AddBridge(info)
	}
}

func (m *Model) handlePairingSuccess(msg PairingSuccessMsg) {
	bridge := m.manager.GetBridge(msg.BridgeID)
	if bridge == nil {
		return
	}

	m.credentials.Set(config.BridgeCredential{
		BridgeID:   msg.BridgeID,
		BridgeName: bridge.Info.Name,
		IPAddress:  bridge.Info.IPAddress,
		ApiKey:     msg.ApiKey,
	})
	_ = m.credentials.Save()
	m.setStatus("Pairing successful!", false)
}

func (m *Model) toggleSelected() tea.Cmd {
	bridge := m.manager.GetActiveBridge()
	if bridge == nil || m.selectedItem == nil {
		return nil
	}

	switch m.selectedItem.Type {
	case panels.EntityRoom, panels.EntityZone:
		if room, ok := panels.GetRoomFromItem(*m.selectedItem); ok {
			if gl, ok := bridge.GetState().RoomGroupedLight(room); ok {
				if gl.Id != nil {
					return toggleGroupedLight(bridge, *gl.Id)
				}
			}
		}
	case panels.EntityLight:
		return toggleLight(bridge, m.selectedItem.ID)
	}
	return nil
}

func (m *Model) setSelectedOn(on bool) tea.Cmd {
	bridge := m.manager.GetActiveBridge()
	if bridge == nil || m.selectedItem == nil {
		return nil
	}

	switch m.selectedItem.Type {
	case panels.EntityRoom, panels.EntityZone:
		if room, ok := panels.GetRoomFromItem(*m.selectedItem); ok {
			if gl, ok := bridge.GetState().RoomGroupedLight(room); ok {
				if gl.Id != nil {
					return setGroupedLightOn(bridge, *gl.Id, on)
				}
			}
		}
	case panels.EntityLight:
		return setLightOn(bridge, m.selectedItem.ID, on)
	}
	return nil
}

func (m *Model) adjustBrightness(delta float64) tea.Cmd {
	bridge := m.manager.GetActiveBridge()
	if bridge == nil || m.selectedItem == nil {
		return nil
	}

	switch m.selectedItem.Type {
	case panels.EntityRoom, panels.EntityZone:
		if room, ok := panels.GetRoomFromItem(*m.selectedItem); ok {
			if gl, ok := bridge.GetState().RoomGroupedLight(room); ok {
				if gl.Id != nil {
					current := float64(0)
					if gl.Dimming != nil && gl.Dimming.Brightness != nil {
						current = float64(*gl.Dimming.Brightness)
					}
					return setGroupedLightBrightness(bridge, *gl.Id, clamp(current+delta, 0, 100))
				}
			}
		}
	case panels.EntityLight:
		if light, ok := panels.GetLightFromItem(*m.selectedItem); ok {
			current := float64(0)
			if light.Dimming != nil && light.Dimming.Brightness != nil {
				current = float64(*light.Dimming.Brightness)
			}
			return setLightBrightness(bridge, m.selectedItem.ID, clamp(current+delta, 0, 100))
		}
	}
	return nil
}

func (m *Model) updateLayout() {
	// Calculate layout tree
	m.layoutTree.Layout(m.width, m.height)

	// Apply sizes to all panels from the tree
	for id, panel := range m.panelMap {
		bounds := m.layoutTree.Bounds(id)
		panel.SetSize(bounds.Width, bounds.Height)
	}

	// Size detail panel
	bounds := m.layoutTree.Bounds(PanelIDDetail)
	m.detailPanel.SetSize(bounds.Width, bounds.Height)

	// Size status bar
	bounds = m.layoutTree.Bounds(PanelIDStatus)
	m.statusBar.SetWidth(bounds.Width)
}

func (m *Model) refreshAllPanels() {
	m.refreshGroupsPanel()
	m.refreshLightsPanel()
	m.refreshDevicesPanel()
	m.refreshScenesPanel()
	m.updateDetailPanel() // Always refresh details with latest state
}

func (m *Model) refreshGroupsPanel() {
	bridge := m.manager.GetActiveBridge()
	if bridge == nil {
		m.groupsPanel().SetItemsByID("Rooms", nil)
		m.groupsPanel().SetItemsByID("Zones", nil)
		m.groupsPanel().SetItemsByID("Entertainment", nil)
		return
	}

	state := bridge.GetState()
	if state == nil {
		return
	}

	// Rooms tab
	m.groupsPanel().SetItemsByID("Rooms", panels.BuildRoomItems(state))

	// Zones tab
	m.groupsPanel().SetItemsByID("Zones", panels.BuildZoneItems(state))

	// Entertainment tab
	m.groupsPanel().SetItemsByID("Entertainment", panels.BuildEntertainmentItems(state))
}

func (m *Model) refreshLightsPanel() {
	bridge := m.manager.GetActiveBridge()
	if bridge == nil {
		m.lightsPanel().SetItems(nil)
		return
	}
	state := bridge.GetState()
	if state == nil {
		return
	}
	m.lightsPanel().SetItems(panels.BuildLightItems(state))
}

func (m *Model) refreshDevicesPanel() {
	bridge := m.manager.GetActiveBridge()
	if bridge == nil {
		m.devicesPanel().SetItems(nil)
		return
	}
	state := bridge.GetState()
	if state == nil {
		return
	}
	m.devicesPanel().SetItems(panels.BuildDeviceItems(state))
}

func (m *Model) refreshScenesPanel() {
	bridge := m.manager.GetActiveBridge()
	if bridge == nil {
		m.scenesPanel().SetRoots(nil)
		return
	}
	state := bridge.GetState()
	if state == nil {
		return
	}
	m.scenesPanel().SetRoots(panels.BuildSceneTree(state))
}

func (m *Model) updateBridgePanel() {
	bridges := m.manager.AllBridges()
	sort.Slice(bridges, func(i, j int) bool {
		return bridges[i].Info.Name < bridges[j].Info.Name
	})
	m.bridgePanel().SetBridges(bridges, m.manager.GetActiveBridgeID())
}

func (m *Model) updateDetailPanel() {
	bridge := m.manager.GetActiveBridge()
	if bridge != nil && m.selectedItem != nil {
		m.detailPanel.SetItem(m.selectedItem, bridge.GetState())
	}
}

func (m *Model) syncSelectionFromFocusedPanel() {
	if panel := m.focusedPanel(); panel != nil {
		if item, ok := panel.SelectedEntity(); ok {
			m.selectedItem = item
			m.updateDetailPanel()
		}
	}
}

func (m *Model) startBridgePairing() tea.Cmd {
	if m.pairing {
		m.setStatus("Already pairing...", false)
		return nil
	}

	var bridgeInfo hue.BridgeInfo
	if m.focusedPanelID() == PanelIDBridges {
		if selected := m.bridgePanel().SelectedBridge(); selected != nil {
			if selected.IsConnected() {
				m.setStatus("Bridge already connected", false)
				return nil
			}
			bridgeInfo = selected.Info
		}
	}

	if bridgeInfo.ID == "" {
		for _, bridge := range m.manager.AllBridges() {
			if !bridge.IsConnected() {
				bridgeInfo = bridge.Info
				break
			}
		}
	}

	if bridgeInfo.ID == "" {
		m.setStatus("No unpaired bridge found. Discovering...", false)
		return discoverBridges()
	}

	m.pairing = true
	m.pairingFor = &bridgeInfo
	bridge := m.manager.GetBridge(bridgeInfo.ID)
	if bridge != nil {
		bridge.Status = hue.StatusPairing
		m.updateBridgePanel()
	}
	m.setStatus("Press the link button on "+bridgeInfo.Name+"...", false)
	return startPairing(bridgeInfo)
}

func (m *Model) setStatus(msg string, isError bool) {
	m.statusMsg = msg
	m.isError = isError
	m.statusBar.SetMessage(msg, isError)
}

func (m *Model) clearStatus() {
	m.statusMsg = ""
	m.isError = false
	m.statusBar.ClearMessage()
}

// View renders the UI.
func (m Model) View() string {
	if !m.ready {
		return "Initializing..."
	}

	if m.quitting {
		return "Goodbye!\n"
	}

	// Render left column panels in layout order
	// The layout tree defines the visual order via the VSplit children
	leftPanelIDs := []string{PanelIDBridges, PanelIDScenes, PanelIDGroups, PanelIDLights, PanelIDDevices}
	panelViews := make([]string, 0, len(leftPanelIDs))
	for _, id := range leftPanelIDs {
		if panel := m.panelMap[id]; panel != nil {
			isActive := m.focusedPanelID() == id
			panelViews = append(panelViews, panel.View(isActive))
		}
	}
	leftColumn := lipgloss.JoinVertical(lipgloss.Left, panelViews...)

	// Right column: detail panel
	rightColumn := m.detailPanel.View(m.focusedOnDetail())

	// Combine columns - constrain to actual dimensions
	content := lipgloss.JoinHorizontal(lipgloss.Top, leftColumn, rightColumn)

	// Status bar at bottom
	status := m.statusBar.View()

	// Combine and constrain to terminal size
	full := lipgloss.JoinVertical(lipgloss.Left, content, status)

	// Use Place to ensure we don't exceed terminal dimensions
	return lipgloss.Place(m.width, m.height, lipgloss.Left, lipgloss.Top, full)
}

// Commands

func discoverBridges() tea.Cmd {
	return func() tea.Msg {
		discovery := hue.NewDiscoveryService(2 * time.Second)
		bridges := discovery.DiscoverAll()
		return BridgesDiscoveredMsg{Bridges: bridges}
	}
}

func connectBridge(bridge *hue.Bridge, apiKey string) tea.Cmd {
	return func() tea.Msg {
		if err := bridge.Connect(apiKey); err != nil {
			return BridgeDisconnectedMsg{BridgeID: bridge.Info.ID, Err: err}
		}
		return BridgeConnectedMsg{BridgeID: bridge.Info.ID}
	}
}

func syncBridgeState(bridge *hue.Bridge) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		if err := bridge.SyncAll(ctx); err != nil {
			return StateSyncErrorMsg{BridgeID: bridge.Info.ID, Err: err}
		}
		return StateSyncedMsg{BridgeID: bridge.Info.ID}
	}
}

func syncLights(bridge *hue.Bridge) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		if err := bridge.SyncLights(ctx); err != nil {
			return StateSyncErrorMsg{BridgeID: bridge.Info.ID, Err: err}
		}
		return LightsSyncedMsg{BridgeID: bridge.Info.ID}
	}
}

func syncLightsAndGroups(bridge *hue.Bridge) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		if err := bridge.SyncLights(ctx); err != nil {
			return StateSyncErrorMsg{BridgeID: bridge.Info.ID, Err: err}
		}
		if err := bridge.SyncGroupedLights(ctx); err != nil {
			return StateSyncErrorMsg{BridgeID: bridge.Info.ID, Err: err}
		}
		return LightsSyncedMsg{BridgeID: bridge.Info.ID}
	}
}

func startSyncTicker() tea.Cmd {
	return tea.Tick(2*time.Second, func(t time.Time) tea.Msg {
		return SyncTickMsg{}
	})
}

func startDiscoveryTicker() tea.Cmd {
	return tea.Tick(30*time.Second, func(t time.Time) tea.Msg {
		return DiscoveryTickMsg{}
	})
}

func startPairing(info hue.BridgeInfo) tea.Cmd {
	return func() tea.Msg {
		auth, err := hue.NewAuthenticator(info.IPAddress)
		if err != nil {
			return PairingFailedMsg{BridgeID: info.ID, Err: err}
		}

		apiKey, err := auth.AuthenticateWithPolling(60*time.Second, 500*time.Millisecond)
		if err != nil {
			return PairingFailedMsg{BridgeID: info.ID, Err: err}
		}

		return PairingSuccessMsg{BridgeID: info.ID, ApiKey: apiKey}
	}
}

func toggleLight(bridge *hue.Bridge, lightID string) tea.Cmd {
	return func() tea.Msg {
		if err := bridge.ToggleLight(lightID); err != nil {
			return ActionErrorMsg{Action: "toggle light", Err: err}
		}
		return LightsSyncedMsg{BridgeID: bridge.Info.ID}
	}
}

func toggleGroupedLight(bridge *hue.Bridge, groupID string) tea.Cmd {
	return func() tea.Msg {
		if err := bridge.ToggleGroupedLight(groupID); err != nil {
			return ActionErrorMsg{Action: "toggle group", Err: err}
		}
		return LightsSyncedMsg{BridgeID: bridge.Info.ID}
	}
}

func setLightOn(bridge *hue.Bridge, lightID string, on bool) tea.Cmd {
	return func() tea.Msg {
		if err := bridge.SetLightOn(lightID, on); err != nil {
			return ActionErrorMsg{Action: "set light", Err: err}
		}
		return LightsSyncedMsg{BridgeID: bridge.Info.ID}
	}
}

func setGroupedLightOn(bridge *hue.Bridge, groupID string, on bool) tea.Cmd {
	return func() tea.Msg {
		if err := bridge.SetGroupedLightOn(groupID, on); err != nil {
			return ActionErrorMsg{Action: "set group", Err: err}
		}
		return LightsSyncedMsg{BridgeID: bridge.Info.ID}
	}
}

func setLightBrightness(bridge *hue.Bridge, lightID string, brightness float64) tea.Cmd {
	return func() tea.Msg {
		if err := bridge.SetLightBrightness(lightID, brightness); err != nil {
			return ActionErrorMsg{Action: "set brightness", Err: err}
		}
		return LightsSyncedMsg{BridgeID: bridge.Info.ID}
	}
}

func setGroupedLightBrightness(bridge *hue.Bridge, groupID string, brightness float64) tea.Cmd {
	return func() tea.Msg {
		if err := bridge.SetGroupedLightBrightness(groupID, brightness); err != nil {
			return ActionErrorMsg{Action: "set brightness", Err: err}
		}
		return LightsSyncedMsg{BridgeID: bridge.Info.ID}
	}
}

func recallScene(bridge *hue.Bridge, sceneID string) tea.Cmd {
	return func() tea.Msg {
		if err := bridge.RecallScene(sceneID); err != nil {
			return ActionErrorMsg{Action: "recall scene", Err: err}
		}
		return LightsSyncedMsg{BridgeID: bridge.Info.ID}
	}
}

func clamp(value, min, max float64) float64 {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}
