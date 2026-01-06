package app

import (
	"context"
	"time"

	"github.com/charmbracelet/bubbles/progress"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/kluzzebass/lazyhue/internal/config"
	"github.com/kluzzebass/lazyhue/internal/hue"
	"github.com/kluzzebass/lazyhue/internal/ui"
	"github.com/kluzzebass/lazyhue/internal/ui/layout"
	"github.com/kluzzebass/lazyhue/internal/ui/panels"
)

// Panel IDs for layout tree
const (
	PanelIDBridges   = "bridges"
	PanelIDHierarchy = "hierarchy"
	PanelIDDetail    = "detail"
	PanelIDStatus    = "status"
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
	helpPanel      *panels.HelpPanel
	pairingPanel   *panels.PairingPanel
	statusBar      *panels.StatusBar
	styles         ui.Styles

	// Keybindings - defined with handlers, single source of truth
	panelBindings  map[string][]ui.Binding // Panel-specific bindings
	globalBindings []ui.Binding            // Global bindings (navigation, quit, etc.)

	// Layout tree
	layoutTree *layout.Tree

	// Current selection
	selectedItem *panels.EntityItem

	// Window dimensions
	width  int
	height int

	// State
	ready            bool
	quitting         bool
	pairing          bool
	pairingFor       *hue.BridgeInfo
	pairingCancel    context.CancelFunc
	statusMsg  string
	isError    bool
}

// New creates a new application model.
func New(cfg *config.Config, creds *config.CredentialStore) Model {
	styles := ui.DefaultStyles()

	// Create panels indexed by ID
	panelMap := map[string]panels.Panel{
		PanelIDBridges:   panels.NewBridgePanel(styles, "1"),
		PanelIDHierarchy: panels.NewTreePanel(styles, "Home", "2"),
	}

	// Panel order for keyboard focus cycling
	panelOrder := []string{
		PanelIDBridges,
		PanelIDHierarchy,
	}

	// Build layout tree - this defines visual structure
	layoutTree := layout.NewTree(
		layout.VSplit(
			// Main content area
			layout.Child{Size: layout.Flex(1), Node: layout.HSplit(
				// Left column (40%)
				layout.Child{Size: layout.Flex(0.4), Node: layout.VSplit(
					layout.Child{Size: layout.Fixed(5), Node: layout.NewLeaf(PanelIDBridges)},
					layout.Child{Size: layout.Flex(1), Node: layout.NewLeaf(PanelIDHierarchy)},
				)},
				// Right column (60%)
				layout.Child{Size: layout.Flex(0.6), Node: layout.NewLeaf(PanelIDDetail)},
			)},
			// Status bar at bottom
			layout.Child{Size: layout.Fixed(1), Node: layout.NewLeaf(PanelIDStatus)},
		),
	)

	m := Model{
		config:       cfg,
		credentials:  creds,
		manager:      hue.NewManager(creds),
		styles:       styles,
		panelMap:     panelMap,
		panelOrder:   panelOrder,
		focusIndex:   0,
		layoutTree:   layoutTree,
		detailPanel:  panels.NewDetailsPanel(styles),
		helpPanel:    panels.NewHelpPanel(styles),
		pairingPanel: panels.NewPairingPanel(styles),
		statusBar:    panels.NewStatusBar(styles),
	}

	// Initialize keybindings - handlers are defined here, right next to keys
	m.initBindings()

	return m
}

// Init implements tea.Model.
func (m Model) Init() tea.Cmd {
	// Try to connect with stored credentials
	cmds := []tea.Cmd{
		m.connectFromStoredCredentials(),
		discoverBridges(),
		startSyncTicker(),
		startDiscoveryTicker(),
	}
	return tea.Batch(cmds...)
}

func (m *Model) connectFromStoredCredentials() tea.Cmd {
	var cmds []tea.Cmd

	for _, cred := range m.credentials.Bridges {
		name := cred.Name
		if name == "" {
			name = cred.BridgeID // Fallback for old credentials without name
		}
		info := hue.BridgeInfo{
			ID:        cred.BridgeID,
			Name:      name,
			IPAddress: cred.IPAddress,
		}
		m.manager.AddBridge(info)
		if bridge := m.manager.GetBridge(cred.BridgeID); bridge != nil {
			cmds = append(cmds, connectBridge(bridge, cred.ApiKey))
		}
	}

	return tea.Batch(cmds...)
}

// Update implements tea.Model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.updateLayout()
		m.ready = true

	case tea.KeyMsg:
		// Handle pairing panel input first
		if m.pairingPanel.IsVisible() {
			if cmd := m.pairingPanel.Update(msg); cmd != nil {
				cmds = append(cmds, cmd)
			}
			return m, tea.Batch(cmds...)
		}

		// Clear status message on user interaction
		m.clearStatus()

		cmd := m.handleKey(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
		// Pass keys to detail panel for viewport scrolling (other panels use dispatch)
		if !m.helpPanel.IsVisible() && m.focusedOnDetail() {
			if panelCmd := m.updateFocusedPanel(msg); panelCmd != nil {
				cmds = append(cmds, panelCmd)
			}
		}
		return m, tea.Batch(cmds...)

	case tea.MouseMsg:
		// Handle mouse in help panel when visible
		if m.helpPanel.IsVisible() {
			cmd := m.helpPanel.Update(msg)
			return m, cmd
		}

		// Clear status message on user interaction
		m.clearStatus()

		cmd := m.handleMouse(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
		return m, tea.Batch(cmds...)

	case BridgesDiscoveredMsg:
		m.bridgePanel().SetDiscovering(false)
		m.handleBridgesDiscovered(msg)
		m.updateBridgePanel()

	case BridgeConnectedMsg:
		bridge := m.manager.GetBridge(msg.BridgeID)
		if bridge != nil {
			bridge.Status = hue.StatusConnected
			m.manager.SetActiveBridge(msg.BridgeID)
			m.setStatus("Connected to "+bridge.Info.Name, false)
			m.updateBridgePanel()
			cmds = append(cmds, syncBridgeState(bridge))
		}

	case BridgeDisconnectedMsg:
		bridge := m.manager.GetBridge(msg.BridgeID)
		if bridge != nil {
			bridge.Status = hue.StatusDisconnected
			m.updateBridgePanel()
		}
		m.setStatus("Connection failed: "+msg.Err.Error(), true)

	case StateSyncedMsg:
		m.refreshAllPanels()
		m.updateBridgePanel()
		m.clearStatus()

	case StateSyncErrorMsg:
		m.setStatus("Sync error: "+msg.Err.Error(), true)

	case LightsSyncedMsg:
		m.bridgePanel().SetPolling(false)
		m.refreshHierarchyPanel()
		m.updateDetailPanel()

	case SyncTickMsg:
		connectedBridges := m.manager.ConnectedBridges()
		if len(connectedBridges) > 0 {
			m.bridgePanel().SetPolling(true)
			cmds = append(cmds, syncAllConnectedBridges(connectedBridges))
			cmds = append(cmds, tea.Tick(300*time.Millisecond, func(t time.Time) tea.Msg {
				return indicatorRefreshMsg{}
			}))
		}
		cmds = append(cmds, startSyncTicker())

	case DiscoveryTickMsg:
		m.bridgePanel().SetDiscovering(true)
		cmds = append(cmds, discoverBridges(), startDiscoveryTicker())
		// Schedule a redraw after indicator duration to turn it off
		cmds = append(cmds, tea.Tick(300*time.Millisecond, func(t time.Time) tea.Msg {
			return indicatorRefreshMsg{}
		}))

	case indicatorRefreshMsg:
		// Just triggers a redraw to update indicator visibility

	case PairingTickMsg:
		// Update the pairing panel and continue ticking while pairing is active
		if m.pairing {
			if cmd := m.pairingPanel.Update(msg); cmd != nil {
				cmds = append(cmds, cmd)
			}
			cmds = append(cmds, pairingTick())
		}

	case progress.FrameMsg:
		// Handle progress bar animation frames
		if m.pairingPanel.IsVisible() {
			if cmd := m.pairingPanel.Update(msg); cmd != nil {
				cmds = append(cmds, cmd)
			}
		}

	case panels.PairingCancelledMsg:
		// Cancel the background pairing goroutine
		if m.pairingCancel != nil {
			m.pairingCancel()
			m.pairingCancel = nil
		}
		m.pairing = false
		m.pairingFor = nil
		m.pairingPanel.Hide()
		bridge := m.manager.GetBridge(msg.BridgeID)
		if bridge != nil {
			bridge.Status = hue.StatusDisconnected
		}
		m.setStatus("Pairing cancelled", false)
		m.updateBridgePanel()

	case PairingSuccessMsg:
		if m.pairingCancel != nil {
			m.pairingCancel = nil
		}
		m.pairing = false
		m.pairingFor = nil
		m.pairingPanel.Hide()
		m.handlePairingSuccess(msg)
		m.updateBridgePanel()
		cmds = append(cmds, connectBridge(m.manager.GetBridge(msg.BridgeID), msg.ApiKey))

	case PairingFailedMsg:
		if m.pairingCancel != nil {
			m.pairingCancel = nil
		}
		m.pairing = false
		m.pairingFor = nil
		m.pairingPanel.Hide()
		// Reset bridge status to disconnected
		if bridge := m.manager.GetBridge(msg.BridgeID); bridge != nil {
			bridge.Status = hue.StatusDisconnected
		}
		m.setStatus("Pairing failed: "+msg.Err.Error(), true)
		m.updateBridgePanel()

	case ActionErrorMsg:
		m.setStatus(msg.Action+": "+msg.Err.Error(), true)

	case StatusMsg:
		m.setStatus(msg.Message, msg.IsError)

	case ClearStatusMsg:
		m.clearStatus()

	case panels.HelpExecuteMsg:
		// Execute action selected from help panel
		m.statusBar.ClearPopupHints()
		if cmd := m.dispatch(msg.Action); cmd != nil {
			cmds = append(cmds, cmd)
		}
	}

	// Update focused panel (but not when help is visible)
	if !m.helpPanel.IsVisible() {
		cmd := m.updateFocusedPanel(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
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
		m.syncSelectionFromFocusedPanel()
		return cmd
	}

	return nil
}
