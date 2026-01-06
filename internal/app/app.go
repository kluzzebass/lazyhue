package app

import (
	"context"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
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
	FocusEntityList FocusRegion = iota
	FocusDetailView
)

// Model is the main Bubble Tea model for the application.
type Model struct {
	// Configuration
	config      *config.Config
	credentials *config.CredentialStore

	// Bridge management
	manager *hue.Manager

	// UI state
	focus       FocusRegion
	entityPanel *panels.EntityPanel
	detailPanel *panels.DetailsPanel
	header      *panels.Header
	statusBar   *panels.StatusBar
	styles      ui.Styles
	keys        ui.KeyMap

	// Current view
	selectedItem *panels.EntityItem
	viewMode     string // "rooms", "zones", "lights"

	// Window dimensions
	width  int
	height int

	// State
	ready     bool
	quitting  bool
	statusMsg string
	isError   bool
}

// New creates a new application model.
func New(cfg *config.Config, creds *config.CredentialStore) Model {
	styles := ui.DefaultStyles()
	keys := ui.DefaultKeyMap()

	return Model{
		config:      cfg,
		credentials: creds,
		manager:     hue.NewManager(creds),
		styles:      styles,
		keys:        keys,
		entityPanel: panels.NewEntityPanel(styles),
		detailPanel: panels.NewDetailsPanel(styles),
		header:      panels.NewHeader(styles),
		statusBar:   panels.NewStatusBar(styles, keys),
		viewMode:    "rooms",
		focus:       FocusEntityList,
	}
}

// Init initializes the application.
func (m Model) Init() tea.Cmd {
	return tea.Batch(
		discoverBridges(),
		tea.SetWindowTitle("lazyhue"),
	)
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
		cmd := m.handleBridgesDiscovered(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}

	case BridgeConnectedMsg:
		m.setStatus("Connected to bridge", false)
		cmds = append(cmds, syncBridgeState(m.manager.GetBridge(msg.BridgeID)))
		cmds = append(cmds, startSyncTicker())

	case StateSyncedMsg:
		m.refreshEntityList()
		m.clearStatus()

	case LightsSyncedMsg:
		m.refreshEntityList()

	case StateSyncErrorMsg:
		m.setStatus("Sync error: "+msg.Err.Error(), true)

	case SyncTickMsg:
		bridge := m.manager.GetActiveBridge()
		if bridge != nil && bridge.IsConnected() {
			// Only sync lights on tick for efficiency
			cmds = append(cmds, syncLights(bridge))
		}
		cmds = append(cmds, startSyncTicker())

	case PairingRequiredMsg:
		m.setStatus("Press the link button on your Hue bridge...", false)
		cmds = append(cmds, startPairing(msg.Bridge))

	case PairingSuccessMsg:
		m.handlePairingSuccess(msg)
		cmds = append(cmds, connectBridge(m.manager.GetBridge(msg.BridgeID), msg.ApiKey))

	case PairingFailedMsg:
		m.setStatus("Pairing failed: "+msg.Err.Error(), true)

	case ActionErrorMsg:
		m.setStatus(msg.Action+": "+msg.Err.Error(), true)

	case StatusMsg:
		m.setStatus(msg.Message, msg.IsError)

	case ClearStatusMsg:
		m.clearStatus()
	}

	// Update focused panel
	switch m.focus {
	case FocusEntityList:
		var cmd tea.Cmd
		m.entityPanel, cmd = m.entityPanel.Update(msg)
		cmds = append(cmds, cmd)

		// Update details when selection changes
		if item, ok := m.entityPanel.SelectedItem(); ok {
			m.selectedItem = &item
			bridge := m.manager.GetActiveBridge()
			if bridge != nil {
				m.detailPanel.SetItem(&item, bridge.GetState())
			}
		}

	case FocusDetailView:
		var cmd tea.Cmd
		m.detailPanel, cmd = m.detailPanel.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m *Model) handleKey(msg tea.KeyMsg) tea.Cmd {
	switch {
	case key.Matches(msg, m.keys.Quit):
		m.quitting = true
		return tea.Quit

	case key.Matches(msg, m.keys.NextPane):
		m.focus = (m.focus + 1) % 2

	case key.Matches(msg, m.keys.PrevPane):
		m.focus = (m.focus + 1) % 2

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
		m.refreshEntityList()

	case key.Matches(msg, m.keys.PrevBridge):
		m.manager.PrevBridge()
		m.refreshEntityList()

	case key.Matches(msg, m.keys.Refresh):
		bridge := m.manager.GetActiveBridge()
		if bridge != nil {
			m.setStatus("Refreshing...", false)
			return syncBridgeState(bridge)
		}
	}

	return nil
}

func (m *Model) handleMouse(msg tea.MouseMsg) tea.Cmd {
	if msg.Action == tea.MouseActionPress {
		leftPanelWidth := m.width / 3
		if msg.X < leftPanelWidth {
			m.focus = FocusEntityList
		} else {
			m.focus = FocusDetailView
		}
	}
	return nil
}

func (m *Model) handleBridgesDiscovered(msg BridgesDiscoveredMsg) tea.Cmd {
	if len(msg.Bridges) == 0 {
		m.setStatus("No bridges found. Press P to pair a new bridge.", true)
		return nil
	}

	var cmds []tea.Cmd
	for _, info := range msg.Bridges {
		bridge := m.manager.AddBridge(info)

		if cred, ok := m.credentials.Get(info.ID); ok {
			cmds = append(cmds, connectBridge(bridge, cred.ApiKey))
		} else {
			return func() tea.Msg {
				return PairingRequiredMsg{Bridge: info}
			}
		}
	}

	return tea.Batch(cmds...)
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
	case panels.EntityRoom:
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
	case panels.EntityRoom:
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
	case panels.EntityRoom:
		if room, ok := panels.GetRoomFromItem(*m.selectedItem); ok {
			if gl, ok := bridge.GetState().RoomGroupedLight(room); ok {
				if gl.Id != nil {
					current := float64(0)
					if gl.Dimming != nil && gl.Dimming.Brightness != nil {
						current = float64(*gl.Dimming.Brightness)
					}
					newBrightness := clamp(current+delta, 0, 100)
					return setGroupedLightBrightness(bridge, *gl.Id, newBrightness)
				}
			}
		}
	case panels.EntityLight:
		if light, ok := panels.GetLightFromItem(*m.selectedItem); ok {
			current := float64(0)
			if light.Dimming != nil && light.Dimming.Brightness != nil {
				current = float64(*light.Dimming.Brightness)
			}
			newBrightness := clamp(current+delta, 0, 100)
			return setLightBrightness(bridge, m.selectedItem.ID, newBrightness)
		}
	}

	return nil
}

func (m *Model) updateLayout() {
	leftWidth := m.width / 3
	rightWidth := m.width - leftWidth - 1
	contentHeight := m.height - 3

	m.entityPanel.SetSize(leftWidth, contentHeight)
	m.detailPanel.SetSize(rightWidth, contentHeight)
	m.header.SetWidth(m.width)
	m.statusBar.SetWidth(m.width)
}

func (m *Model) refreshEntityList() {
	bridge := m.manager.GetActiveBridge()
	if bridge == nil {
		return
	}

	state := bridge.GetState()
	if state == nil {
		return
	}

	var items []list.Item
	switch m.viewMode {
	case "rooms":
		items = panels.BuildRoomItems(state)
		m.entityPanel.SetTitle("Rooms")
	case "lights":
		items = panels.BuildLightItems(state)
		m.entityPanel.SetTitle("Lights")
	}

	m.entityPanel.SetItems(items)
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

	bridge := m.manager.GetActiveBridge()
	header := m.header.View(bridge, m.manager.BridgeCount())

	leftPanel := m.entityPanel.View(m.focus == FocusEntityList)
	rightPanel := m.detailPanel.View(m.focus == FocusDetailView)

	content := lipgloss.JoinHorizontal(lipgloss.Top, leftPanel, rightPanel)

	status := m.statusBar.View()

	return lipgloss.JoinVertical(lipgloss.Left, header, content, status)
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

func clamp(value, min, max float64) float64 {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}
