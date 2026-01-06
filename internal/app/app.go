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
	"github.com/kluzzebass/lazyhue/internal/ui/panels"
)

// FocusRegion identifies which panel has focus.
type FocusRegion int

const (
	FocusDetail  FocusRegion = iota // 0
	FocusBridges                    // 1
	FocusGroups                     // 2
	FocusLights                     // 3
	FocusDevices                    // 4
	FocusScenes                     // 5
	focusCount                      // sentinel for cycling
)

// Model is the main Bubble Tea model for the application.
type Model struct {
	// Configuration
	config      *config.Config
	credentials *config.CredentialStore

	// Bridge management
	manager *hue.Manager

	// UI panels
	focus        FocusRegion
	bridgePanel  *panels.BridgePanel
	groupsPanel  *panels.TabbedPanel // Rooms | Zones | Entertainment
	lightsPanel  *panels.ListPanel
	devicesPanel *panels.ListPanel
	scenesPanel  *panels.ListPanel
	detailPanel  *panels.DetailsPanel
	statusBar    *panels.StatusBar
	styles       ui.Styles
	keys         ui.KeyMap

	// Current selection
	selectedItem *panels.EntityItem

	// Window dimensions
	width  int
	height int

	// Layout (calculated in updateLayout, used by mouse handling)
	layout struct {
		leftWidth     int
		panelTops     [5]int // Y positions where each left panel starts
		contentHeight int
	}

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

	return Model{
		config:       cfg,
		credentials:  creds,
		manager:      hue.NewManager(creds),
		styles:       styles,
		keys:         keys,
		bridgePanel:  panels.NewBridgePanel(styles),
		groupsPanel:  panels.NewTabbedPanel(styles, "Groups", "2", []string{"Rooms", "Zones", "Entertainment"}),
		lightsPanel:  panels.NewListPanel(styles, "Lights", "3", "No lights", panels.EntityDelegate{Styles: styles}),
		devicesPanel: panels.NewListPanel(styles, "Devices", "4", "No devices", panels.EntityDelegate{Styles: styles}),
		scenesPanel:  panels.NewListPanel(styles, "Scenes", "5", "No scenes", panels.EntityDelegate{Styles: styles}),
		detailPanel:  panels.NewDetailsPanel(styles),
		statusBar:    panels.NewStatusBar(styles, keys),
		focus:        FocusBridges,
	}
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

	// Also run discovery in background to find new/unknown bridges
	cmds = append(cmds, discoverBridges())

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
		cmd := m.handleMouse(msg)
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
		m.refreshAllPanels()
		m.syncSelectionFromFocusedPanel()
		m.clearStatus()

	case LightsSyncedMsg:
		m.refreshLightsPanel()

	case StateSyncErrorMsg:
		m.setStatus("Sync error: "+msg.Err.Error(), true)

	case SyncTickMsg:
		bridge := m.manager.GetActiveBridge()
		if bridge != nil && bridge.IsConnected() {
			cmds = append(cmds, syncLights(bridge))
		}
		cmds = append(cmds, startSyncTicker())

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
	var cmd tea.Cmd

	switch m.focus {
	case FocusBridges:
		m.bridgePanel, cmd = m.bridgePanel.Update(msg)

	case FocusGroups:
		m.groupsPanel, cmd = m.groupsPanel.Update(msg)
		if item, ok := m.groupsPanel.SelectedItem(); ok {
			m.selectedItem = &item
			m.updateDetailPanel()
		}

	case FocusLights:
		m.lightsPanel, cmd = m.lightsPanel.Update(msg)
		if item, ok := m.lightsPanel.SelectedEntityItem(); ok {
			m.selectedItem = &item
			m.updateDetailPanel()
		}

	case FocusDevices:
		m.devicesPanel, cmd = m.devicesPanel.Update(msg)
		if item, ok := m.devicesPanel.SelectedEntityItem(); ok {
			m.selectedItem = &item
			m.updateDetailPanel()
		}

	case FocusScenes:
		m.scenesPanel, cmd = m.scenesPanel.Update(msg)
		if item, ok := m.scenesPanel.SelectedEntityItem(); ok {
			m.selectedItem = &item
			m.updateDetailPanel()
		}

	case FocusDetail:
		m.detailPanel, cmd = m.detailPanel.Update(msg)
	}

	return cmd
}

func (m *Model) handleKey(msg tea.KeyMsg) tea.Cmd {
	switch {
	case key.Matches(msg, m.keys.Quit):
		m.quitting = true
		return tea.Quit

	// Panel focus with number keys
	case key.Matches(msg, m.keys.FocusDetail):
		m.focus = FocusDetail
	case key.Matches(msg, m.keys.FocusBridges):
		m.focus = FocusBridges
	case key.Matches(msg, m.keys.FocusGroups):
		m.focus = FocusGroups
		m.syncSelectionFromFocusedPanel()
	case key.Matches(msg, m.keys.FocusLights):
		m.focus = FocusLights
		m.syncSelectionFromFocusedPanel()
	case key.Matches(msg, m.keys.FocusDevices):
		m.focus = FocusDevices
		m.syncSelectionFromFocusedPanel()
	case key.Matches(msg, m.keys.FocusScenes):
		m.focus = FocusScenes
		m.syncSelectionFromFocusedPanel()

	case key.Matches(msg, m.keys.NextPane):
		m.focus = (m.focus + 1) % focusCount
		m.syncSelectionFromFocusedPanel()
	case key.Matches(msg, m.keys.PrevPane):
		m.focus = (m.focus + focusCount - 1) % focusCount
		m.syncSelectionFromFocusedPanel()

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
	switch m.focus {
	case FocusBridges:
		if bridge := m.bridgePanel.SelectedBridge(); bridge != nil {
			m.manager.SetActiveBridge(bridge.Info.ID)
			m.updateBridgePanel()
			m.refreshAllPanels()
			m.focus = FocusGroups
		}
	case FocusScenes:
		// Activate the selected scene
		if m.selectedItem != nil && m.selectedItem.Type == panels.EntityScene {
			bridge := m.manager.GetActiveBridge()
			if bridge != nil {
				return recallScene(bridge, m.selectedItem.ID)
			}
		}
	}
	return nil
}

func (m *Model) handleMouse(msg tea.MouseMsg) tea.Cmd {
	if msg.Action != tea.MouseActionPress {
		return nil
	}

	if msg.X < m.layout.leftWidth {
		// Left column - determine which panel based on Y position
		y := msg.Y
		switch {
		case y < m.layout.panelTops[1]:
			m.focus = FocusBridges
		case y < m.layout.panelTops[2]:
			m.focus = FocusGroups
		case y < m.layout.panelTops[3]:
			m.focus = FocusLights
		case y < m.layout.panelTops[4]:
			m.focus = FocusDevices
		default:
			m.focus = FocusScenes
		}
		m.syncSelectionFromFocusedPanel()
	} else {
		m.focus = FocusDetail
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
	// Left column gets 40% of width, right column gets the rest
	leftWidth := m.width * 4 / 10
	if leftWidth < 25 {
		leftWidth = 25
	}
	rightWidth := m.width - leftWidth
	contentHeight := m.height - 1 // status bar

	// Bridge panel: fixed height for ~3 items (5 rows including border)
	bridgeHeight := 5
	remainingHeight := contentHeight - bridgeHeight

	// Divide remaining height among 4 panels
	panelHeight := remainingHeight / 4
	remainder := remainingHeight % 4

	// Distribute remainder to make heights more even
	heights := [4]int{}
	for i := range heights {
		heights[i] = panelHeight
		if i < remainder {
			heights[i]++
		}
	}

	// Store layout for mouse handling
	m.layout.leftWidth = leftWidth
	m.layout.contentHeight = contentHeight
	m.layout.panelTops[0] = 0                                                         // Bridges
	m.layout.panelTops[1] = bridgeHeight                                              // Groups
	m.layout.panelTops[2] = bridgeHeight + heights[0]                                 // Lights
	m.layout.panelTops[3] = bridgeHeight + heights[0] + heights[1]                    // Devices
	m.layout.panelTops[4] = bridgeHeight + heights[0] + heights[1] + heights[2]       // Scenes

	m.bridgePanel.SetSize(leftWidth, bridgeHeight)
	m.groupsPanel.SetSize(leftWidth, heights[0])
	m.lightsPanel.SetSize(leftWidth, heights[1])
	m.devicesPanel.SetSize(leftWidth, heights[2])
	m.scenesPanel.SetSize(leftWidth, heights[3])
	m.detailPanel.SetSize(rightWidth, contentHeight)
	m.statusBar.SetWidth(m.width)
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
		m.groupsPanel.SetItemsByID("Rooms", nil)
		m.groupsPanel.SetItemsByID("Zones", nil)
		m.groupsPanel.SetItemsByID("Entertainment", nil)
		return
	}

	state := bridge.GetState()
	if state == nil {
		return
	}

	// Rooms tab
	m.groupsPanel.SetItemsByID("Rooms", panels.BuildRoomItems(state))

	// Zones tab
	m.groupsPanel.SetItemsByID("Zones", panels.BuildZoneItems(state))

	// Entertainment tab
	m.groupsPanel.SetItemsByID("Entertainment", panels.BuildEntertainmentItems(state))
}

func (m *Model) refreshLightsPanel() {
	bridge := m.manager.GetActiveBridge()
	if bridge == nil {
		m.lightsPanel.SetItems(nil)
		return
	}
	state := bridge.GetState()
	if state == nil {
		return
	}
	m.lightsPanel.SetItems(panels.BuildLightItems(state))
}

func (m *Model) refreshDevicesPanel() {
	bridge := m.manager.GetActiveBridge()
	if bridge == nil {
		m.devicesPanel.SetItems(nil)
		return
	}
	state := bridge.GetState()
	if state == nil {
		return
	}
	m.devicesPanel.SetItems(panels.BuildDeviceItems(state))
}

func (m *Model) refreshScenesPanel() {
	bridge := m.manager.GetActiveBridge()
	if bridge == nil {
		m.scenesPanel.SetItems(nil)
		return
	}
	state := bridge.GetState()
	if state == nil {
		return
	}
	m.scenesPanel.SetItems(panels.BuildSceneItems(state))
}

func (m *Model) updateBridgePanel() {
	bridges := m.manager.AllBridges()
	sort.Slice(bridges, func(i, j int) bool {
		return bridges[i].Info.Name < bridges[j].Info.Name
	})
	m.bridgePanel.SetBridges(bridges, m.manager.GetActiveBridgeID())
}

func (m *Model) updateDetailPanel() {
	bridge := m.manager.GetActiveBridge()
	if bridge != nil && m.selectedItem != nil {
		m.detailPanel.SetItem(m.selectedItem, bridge.GetState())
	}
}

func (m *Model) syncSelectionFromFocusedPanel() {
	switch m.focus {
	case FocusGroups:
		if item, ok := m.groupsPanel.SelectedItem(); ok {
			m.selectedItem = &item
			m.updateDetailPanel()
		}
	case FocusLights:
		if item, ok := m.lightsPanel.SelectedEntityItem(); ok {
			m.selectedItem = &item
			m.updateDetailPanel()
		}
	case FocusDevices:
		if item, ok := m.devicesPanel.SelectedEntityItem(); ok {
			m.selectedItem = &item
			m.updateDetailPanel()
		}
	case FocusScenes:
		if item, ok := m.scenesPanel.SelectedEntityItem(); ok {
			m.selectedItem = &item
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
	if m.focus == FocusBridges {
		if selected := m.bridgePanel.SelectedBridge(); selected != nil {
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

	// Left column: stacked panels
	leftColumn := lipgloss.JoinVertical(lipgloss.Left,
		m.bridgePanel.View(m.focus == FocusBridges),
		m.groupsPanel.View(m.focus == FocusGroups),
		m.lightsPanel.View(m.focus == FocusLights),
		m.devicesPanel.View(m.focus == FocusDevices),
		m.scenesPanel.View(m.focus == FocusScenes),
	)

	// Right column: detail panel
	rightColumn := m.detailPanel.View(m.focus == FocusDetail)

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

func startSyncTicker() tea.Cmd {
	return tea.Tick(2*time.Second, func(t time.Time) tea.Msg {
		return SyncTickMsg{}
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
		return nil
	}
}

func toggleGroupedLight(bridge *hue.Bridge, groupID string) tea.Cmd {
	return func() tea.Msg {
		if err := bridge.ToggleGroupedLight(groupID); err != nil {
			return ActionErrorMsg{Action: "toggle group", Err: err}
		}
		return nil
	}
}

func setLightOn(bridge *hue.Bridge, lightID string, on bool) tea.Cmd {
	return func() tea.Msg {
		if err := bridge.SetLightOn(lightID, on); err != nil {
			return ActionErrorMsg{Action: "set light", Err: err}
		}
		return nil
	}
}

func setGroupedLightOn(bridge *hue.Bridge, groupID string, on bool) tea.Cmd {
	return func() tea.Msg {
		if err := bridge.SetGroupedLightOn(groupID, on); err != nil {
			return ActionErrorMsg{Action: "set group", Err: err}
		}
		return nil
	}
}

func setLightBrightness(bridge *hue.Bridge, lightID string, brightness float64) tea.Cmd {
	return func() tea.Msg {
		if err := bridge.SetLightBrightness(lightID, brightness); err != nil {
			return ActionErrorMsg{Action: "set brightness", Err: err}
		}
		return nil
	}
}

func setGroupedLightBrightness(bridge *hue.Bridge, groupID string, brightness float64) tea.Cmd {
	return func() tea.Msg {
		if err := bridge.SetGroupedLightBrightness(groupID, brightness); err != nil {
			return ActionErrorMsg{Action: "set brightness", Err: err}
		}
		return nil
	}
}

func recallScene(bridge *hue.Bridge, sceneID string) tea.Cmd {
	return func() tea.Msg {
		if err := bridge.RecallScene(sceneID); err != nil {
			return ActionErrorMsg{Action: "recall scene", Err: err}
		}
		m := StatusMsg{Message: "Scene activated", IsError: false}
		return m
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
