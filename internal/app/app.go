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
	PanelIDLog       = "log"
	PanelIDStatus    = "status"
)

// Model is the main Bubble Tea model for the application.
type Model struct {
	// Configuration
	config      *config.Config
	credentials *config.CredentialStore
	uiState     *config.UIStateStore

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
	popupPanel     *panels.PopupPanel
	logPanel       *panels.LogPanel
	statusBar      *panels.StatusBar
	styles         ui.Styles

	// Keybindings - defined with handlers, single source of truth
	panelBindings  map[string][]ui.Binding // Panel-specific bindings
	globalBindings []ui.Binding            // Global bindings (navigation, quit, etc.)

	// Layout tree
	layoutTree *layout.Tree

	// Current selection
	selectedItem      *panels.EntityItem
	displayedBridgeID string // Bridge currently shown in hierarchy panel

	// Window dimensions
	width  int
	height int

	// State
	ready           bool
	quitting        bool
	pairing         bool
	pairingFor      *hue.BridgeInfo
	pairingCancel   context.CancelFunc
	statusMsg       string
	isError         bool
	statusExpiry    time.Time // When to auto-clear status (zero means no expiry)
	logPanelVisible bool

	// Live editing state
	editingLightID    string // Light ID being edited (for live sync)
	syncLivePopup     func() // Callback to sync popup with live state
	blinkTimerRunning bool   // Whether the color wheel blink timer is active

	// SSE event channel for receiving bridge events from goroutines
	eventChan chan BridgeEventMsg
}

// buildLayoutTree creates the layout tree with or without the log panel.
func buildLayoutTree(showLog bool) *layout.Tree {
	var rightColumn layout.Node
	if showLog {
		// Right column split into detail (⅔) and log (⅓)
		rightColumn = layout.VSplit(
			layout.Child{Size: layout.Flex(0.67), Node: layout.NewLeaf(PanelIDDetail)},
			layout.Child{Size: layout.Flex(0.33), Node: layout.NewLeaf(PanelIDLog)},
		)
	} else {
		// Right column is just detail panel
		rightColumn = layout.NewLeaf(PanelIDDetail)
	}

	return layout.NewTree(
		layout.VSplit(
			// Main content area
			layout.Child{Size: layout.Flex(1), Node: layout.HSplit(
				// Left column (40%)
				layout.Child{Size: layout.Flex(0.4), Node: layout.VSplit(
					layout.Child{Size: layout.Fixed(5), Node: layout.NewLeaf(PanelIDBridges)},
					layout.Child{Size: layout.Flex(1), Node: layout.NewLeaf(PanelIDHierarchy)},
				)},
				// Right column (60%)
				layout.Child{Size: layout.Flex(0.6), Node: rightColumn},
			)},
			// Status bar at bottom
			layout.Child{Size: layout.Fixed(1), Node: layout.NewLeaf(PanelIDStatus)},
		),
	)
}

// New creates a new application model.
func New(cfg *config.Config, creds *config.CredentialStore, uiState *config.UIStateStore) Model {
	styles := ui.DefaultStyles()

	// Create panels indexed by ID
	panelMap := map[string]panels.Panel{
		PanelIDBridges:   panels.NewBridgePanel(styles, "1"),
		PanelIDHierarchy: panels.NewHomeTabbedPanel(styles, "2"),
	}

	// Panel order for keyboard focus cycling
	panelOrder := []string{
		PanelIDBridges,
		PanelIDHierarchy,
	}

	// Build layout tree - this defines visual structure
	// Build initial layout without log panel (starts hidden)
	layoutTree := buildLayoutTree(false)

	m := Model{
		config:       cfg,
		credentials:  creds,
		uiState:      uiState,
		manager:      hue.NewManager(creds),
		styles:       styles,
		panelMap:     panelMap,
		panelOrder:   panelOrder,
		focusIndex:   0,
		layoutTree:   layoutTree,
		detailPanel:  panels.NewDetailsPanel(styles),
		logPanel:     panels.NewLogPanel(styles),
		helpPanel:    panels.NewHelpPanel(styles),
		pairingPanel: panels.NewPairingPanel(styles),
		popupPanel:   panels.NewPopupPanel(styles),
		statusBar:    panels.NewStatusBar(styles),
		eventChan:    make(chan BridgeEventMsg, 100), // Buffered channel for SSE events
	}

	// Initialize keybindings - handlers are defined here, right next to keys
	m.initBindings()

	// Set up detail panel field save callback
	m.detailPanel.SetOnFieldSave(m.handleDetailFieldSave)

	// Restore last selected bridge from saved state (visual selection only)
	// Active bridge will be set after bridges are loaded in connectFromStoredCredentials
	if uiState.LastSelectedBridgeID != "" {
		m.bridgePanel().SetInitialSelection(uiState.LastSelectedBridgeID)
	}

	// Restore focused panel (-1 = detail panel, 0+ = main panels)
	if uiState.FocusedPanelIndex == -1 || (uiState.FocusedPanelIndex >= 0 && uiState.FocusedPanelIndex < len(m.panelOrder)) {
		m.focusIndex = uiState.FocusedPanelIndex
	}

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
		startStateSaveTicker(),             // Periodic state saves for crash recovery
		listenForBridgeEvents(m.eventChan), // Start listening for SSE events
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

	// Now that bridges are added, set the active bridge from saved state
	if m.uiState.LastSelectedBridgeID != "" {
		m.manager.SetActiveBridge(m.uiState.LastSelectedBridgeID)
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
		// Handle popup panel input first (highest priority modal)
		if m.popupPanel.IsVisible() {
			if cmd := m.popupPanel.Update(msg); cmd != nil {
				cmds = append(cmds, cmd)
			}
			// Start blink timer for popup if not already running
			if !m.blinkTimerRunning {
				m.blinkTimerRunning = true
				cmds = append(cmds, tea.Tick(400*time.Millisecond, func(t time.Time) tea.Msg {
					return blinkTickMsg{}
				}))
			}
			return m, tea.Batch(cmds...)
		}

		// Handle pairing panel input
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
		// Pass keys to right column panels for scrolling
		if !m.helpPanel.IsVisible() {
			if m.focusedOnDetail() {
				if panelCmd := m.updateFocusedPanel(msg); panelCmd != nil {
					cmds = append(cmds, panelCmd)
				}
			} else if m.focusedOnLog() {
				m.logPanel.Update(msg)
			}
		}
		return m, tea.Batch(cmds...)

	case tea.MouseMsg:
		// Handle mouse in popup panel first (highest priority modal)
		if m.popupPanel.IsVisible() {
			if cmd := m.popupPanel.Update(msg); cmd != nil {
				cmds = append(cmds, cmd)
			}
			// Start blink timer for popup if not already running
			if !m.blinkTimerRunning {
				m.blinkTimerRunning = true
				cmds = append(cmds, tea.Tick(400*time.Millisecond, func(t time.Time) tea.Msg {
					return blinkTickMsg{}
				}))
			}
			return m, tea.Batch(cmds...)
		}

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
			// Set up SSE event callback to notify the app when events arrive
			eventChan := m.eventChan
			bridge.OnEvent(func(bridgeID, resourceType, resourceID, eventType string) {
				select {
				case eventChan <- BridgeEventMsg{
					BridgeID:     bridgeID,
					ResourceType: resourceType,
					ResourceID:   resourceID,
					EventType:    eventType,
				}:
				default:
					// Channel full, drop event (will catch up on next one)
				}
			})
			m.setStatusTemporary("Connected to "+bridge.Info.Name, false, 3*time.Second)
			m.updateBridgePanel() // This will set active bridge if none is set
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
		m.updateBridgePanel()
		// Check if the user has this bridge selected in the panel
		selectedBridge := m.bridgePanel().SelectedBridge()
		if selectedBridge != nil && selectedBridge.Info.ID == msg.BridgeID {
			// User is looking at this bridge - switch to it if connected
			if selectedBridge.IsConnected() {
				m.manager.SetActiveBridge(msg.BridgeID)
				m.refreshAllPanels()
				m.restoreExpandedState(msg.BridgeID)
			}
		} else if msg.BridgeID == m.displayedBridgeID {
			// Bridge being displayed got updated
			m.refreshAllPanels()
			m.restoreExpandedState(msg.BridgeID)
		}
		m.clearStatus()

	case StateSyncErrorMsg:
		m.setStatus("Sync error: "+msg.Err.Error(), true)

	case LightsSyncedMsg:
		m.bridgePanel().SetPolling("", false)
		// Log the request if action info is provided
		if msg.Action != "" {
			m.logPanel.AddEntry("request", msg.Target+": "+msg.Action)
		}
		// Refresh hierarchy only if this bridge is being displayed
		// (or if BridgeID is empty for batch sync and we're displaying a connected bridge)
		if msg.BridgeID == m.displayedBridgeID || (msg.BridgeID == "" && m.displayedBridgeID != "") {
			m.refreshHierarchyPanel()
			m.updateDetailPanel()
		}

	case BridgeEventMsg:
		// SSE event received - state was already updated in-memory by the bridge
		// Log the event with rich details
		details := m.buildEventDetails(msg)
		m.logPanel.AddEvent(details)
		// Just flash the indicator for THIS bridge and refresh the UI
		m.bridgePanel().SetPolling(msg.BridgeID, true)
		cmds = append(cmds, tea.Tick(300*time.Millisecond, func(t time.Time) tea.Msg {
			return indicatorRefreshMsg{}
		}))
		// Refresh UI only if this bridge is actually being displayed
		// (don't refresh if user is viewing a disconnected bridge)
		if msg.BridgeID == m.displayedBridgeID {
			m.refreshHierarchyPanel()
			m.updateDetailPanel()
			// Sync live popup if editing the light that changed
			if m.editingLightID != "" && msg.ResourceType == "light" && msg.ResourceID == m.editingLightID {
				if m.syncLivePopup != nil {
					m.syncLivePopup()
				}
			}
		}
		// Re-listen for more events
		cmds = append(cmds, listenForBridgeEvents(m.eventChan))

	case SyncTickMsg:
		connectedBridges := m.manager.ConnectedBridges()
		if len(connectedBridges) > 0 {
			m.bridgePanel().SetPolling("", true) // Flash all connected bridges
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

	case StateSaveTickMsg:
		// Periodically save UI state to handle abrupt termination
		if bridgeID := m.manager.GetActiveBridgeID(); bridgeID != "" {
			m.saveExpandedState(bridgeID)
			m.uiState.LastSelectedBridgeID = bridgeID
			_ = m.uiState.Save()
		}
		cmds = append(cmds, startStateSaveTicker())

	case SignalQuitMsg:
		// SIGTERM/SIGINT received - save state and quit
		m.quitting = true
		if bridgeID := m.manager.GetActiveBridgeID(); bridgeID != "" {
			m.saveExpandedState(bridgeID)
			m.uiState.LastSelectedBridgeID = bridgeID
			_ = m.uiState.Save()
		}
		return m, tea.Quit

	case indicatorRefreshMsg:
		// Check if status message should be cleared
		m.checkStatusExpiry()

	case panels.StartBlinkTickMsg:
		// Start the blink ticker (from details panel entering color edit mode)
		if !m.blinkTimerRunning {
			m.blinkTimerRunning = true
			cmds = append(cmds, tea.Tick(400*time.Millisecond, func(t time.Time) tea.Msg {
				return blinkTickMsg{}
			}))
		}

	case blinkTickMsg:
		// Toggle color wheel blink state for popup and details panel
		m.popupPanel.ToggleBlink()
		m.detailPanel.ToggleBlink()

		// Continue ticking while popup is visible OR details panel is editing color
		if m.popupPanel.IsVisible() || m.detailPanel.IsEditingColor() {
			cmds = append(cmds, tea.Tick(400*time.Millisecond, func(t time.Time) tea.Msg {
				return blinkTickMsg{}
			}))
		} else {
			m.blinkTimerRunning = false
		}

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
		m.setStatusTemporary("Pairing cancelled", false, 3*time.Second)
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
		return panel.Update(msg)
	}

	return nil
}
