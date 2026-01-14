// Package app2 is the v2 application using bubbletea v2, bubbles v2, and lipgloss v2.
package app2

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/v2/help"
	"github.com/charmbracelet/bubbles/v2/key"
	"github.com/charmbracelet/bubbles/v2/textinput"
	"github.com/charmbracelet/bubbles/v2/viewport"
	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/charmbracelet/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	zone "github.com/lrstanley/bubblezone/v2"

	"github.com/kluzzebass/lazyhue/internal/config"
	"github.com/kluzzebass/lazyhue/internal/hue"
	"github.com/kluzzebass/lazyhue/internal/hueclient"
	"github.com/kluzzebass/lazyhue/internal/ui2"
	"github.com/kluzzebass/lazyhue/internal/ui2/component"
	"github.com/kluzzebass/lazyhue/internal/ui2/component/field"
	gridlayout "github.com/kluzzebass/lazyhue/internal/ui2/component/layout"
	"github.com/kluzzebass/lazyhue/internal/ui2/layout"
	"github.com/kluzzebass/lazyhue/internal/ui2/panels"
)

// Panel IDs
const (
	PanelDetail = "detail"
	PanelTree   = "tree"
	PanelLog    = "log"
)

// Model is the main application model.
// NavigationEntry represents a point in navigation history.
type NavigationEntry struct {
	EntityType string    // Type of entity ("light", "device", "scene", etc.)
	EntityID   string    // ID of the entity
	BridgeID   string    // Bridge the entity belongs to
	Timestamp  time.Time // When this navigation occurred
}

type Model struct {
	// Service layer
	manager     *hue.Manager
	credentials *config.CredentialStore
	uiState     *config.UIStateStore

	// UI components
	tree           *panels.TreePanel
	detailViewport viewport.Model
	logViewport    viewport.Model
	keys           ui2.KeyMap
	help           help.Model
	styles         ui2.Styles
	zones          *zone.Manager
	layout         *layout.Tree

	// Component tree for event routing
	componentRoot component.Component

	// Panel management
	panelOrder []string // Panel IDs in focus-cycle order

	// State
	width        int
	height       int
	focusedPane  string
	previousPane string // Track previous panel for Escape key
	status       string
	activities       []Activity
	quitting         bool
	eventChan        chan bridgeEventMsg
	requestChan      chan requestMsg
	errorChan        chan errorMsg
	eventCancelFuncs map[string]context.CancelFunc
	bridgeBlinkUntil map[string]time.Time // Track when bridge blink indicators should stop
	lightGrid *gridlayout.Grid // Grid layout for light controls
	selectedLightID   string               // ID of currently selected light (for form updates)
	lastTreeClick     time.Time            // Track last tree item click for double-click detection
	lastTreeClickID   string               // Track which tree item was last clicked

	// Rename mode state
	renaming           bool                 // Whether we're in rename mode
	// renameInput removed - now owned by renameModal (TextInputModal)
	renameEntityType   panels.EntityType    // Type of entity being renamed
	renameEntityID     string               // ID of entity being renamed
	renameBridgeID     string               // Bridge ID the entity belongs to
	renameOriginalName string               // Original name for cancel

	// Delete confirmation state
	confirmingDelete     bool              // Whether we're showing delete confirmation
	deleteBridgeID       string            // ID of bridge to delete
	deleteBridgeName     string            // Name of bridge to delete (for display)
	confirmingDeleteEntity bool            // Whether we're showing entity delete confirmation
	deleteEntityID       string            // ID of entity to delete
	deleteEntityName     string            // Name of entity to delete (for display)
	deleteEntityType     panels.EntityType // Type of entity to delete
	deleteEntityBridgeID string            // Bridge ID the entity belongs to

	// Pairing state
	pairing           bool                    // Whether we're in pairing mode
	pairingFor        *hue.BridgeInfo         // Bridge being paired
	pairingCancel     context.CancelFunc      // Function to cancel pairing
	pairingStartTime  time.Time               // When pairing started
	discoveredBridges []hue.BridgeInfo        // Discovered bridges available for pairing
	pairingRemaining  int                     // Seconds remaining in pairing countdown
	pairingRequested  bool                    // Whether user explicitly requested pairing via 'p' key

	// Create mode state
	creatingRoom   bool   // Whether we're creating a room
	creatingZone   bool   // Whether we're creating a zone
	createBridgeID string // Bridge to create room/zone in

	// Input capture stack (for modal input handling) - legacy, being replaced
	inputStack *InputStack

	// Modal components for the component tree
	renameModal *component.TextInputModal
	createModal *component.TextInputModal // For creating rooms/zones
	detailStack *component.StackedContainer

	// Help display
	showHelp bool // Whether help is displayed in detail panel

	// Navigation history
	navigationHistory []NavigationEntry
	historyIndex      int // Current position in history (-1 if no history or at latest)

	// Keybindings for help system
	panelBindings  map[string][]Binding
	globalBindings []Binding
}

// New creates a new application model.
func New(creds *config.CredentialStore) Model {
	styles := ui2.DefaultStyles()
	keys := ui2.DefaultKeyMap()
	zones := zone.New()

	// Load UI state (or create empty if doesn't exist)
	uiState, err := config.LoadUIState()
	if err != nil {
		// If we can't load state, just start with empty state
		uiState = config.NewUIStateStore()
	}

	// Define panel order for automatic key assignment
	// Detail (0), Tree (1), Log (2)
	panelOrder := []string{
		PanelDetail,
		PanelTree,
		PanelLog,
	}

	// Create tree panel (key will be assigned automatically)
	tree := panels.NewTreePanel(styles, zones, "Home", "")

	// Assign keys automatically based on position in panelOrder
	// Keys are 0-based: first panel gets "0", second gets "1", etc.
	for i, panelID := range panelOrder {
		key := fmt.Sprintf("%d", i)
		switch panelID {
		case PanelTree:
			tree.SetKey(key)
		}
	}

	// Create help model
	h := help.New()

	// Build layout tree
	// Left column: Tree
	// Right column: Detail + Log
	layoutRoot := layout.HSplit(
		layout.Child{Size: layout.Flex(0.4), Node: layout.NewLeaf(PanelTree)},
		layout.Child{Size: layout.Flex(0.6), Node: layout.VSplit(
			layout.Child{Size: layout.Flex(0.67), Node: layout.NewLeaf(PanelDetail)},
			layout.Child{Size: layout.Flex(0.33), Node: layout.NewLeaf(PanelLog)},
		)},
	)
	layoutTree := layout.NewTree(layoutRoot)

	m := Model{
		manager:          hue.NewManager(creds),
		credentials:      creds,
		uiState:          uiState,
		tree:             tree,
		panelOrder:       panelOrder,
		detailViewport:   viewport.New(),
		logViewport:      viewport.New(),
		keys:             keys,
		help:             h,
		styles:           styles,
		zones:            zones,
		layout:           layoutTree,
		focusedPane:      PanelTree,
		previousPane:     PanelTree,
		status:           "Loading bridges...",
		activities:       []Activity{},
		eventChan:        make(chan bridgeEventMsg, 100),
		requestChan:      make(chan requestMsg, 100),
		errorChan:        make(chan errorMsg, 100),
		eventCancelFuncs: make(map[string]context.CancelFunc),
		bridgeBlinkUntil:   make(map[string]time.Time),
		lightGrid: gridlayout.NewGrid().SetGaps(2, 0).SetFocusIndicator(true, "> ", "  "),
		selectedLightID:    "",
		historyIndex:     -1, // No history initially
		inputStack:       NewInputStack(),
	}

	// Initialize keybindings
	m.initBindings()

	// Initialize input capture stack (legacy - keeping for now)
	m.inputStack.Push(NewRenameCapture(&m))

	// Build component tree for event routing
	// Mirrors the layout tree structure:
	// HSplit(Tree, VSplit(DetailStack, Log))
	treeComponent := component.NewTreePanelComponent(PanelTree, m.tree)
	detailViewport := component.NewViewportComponent(PanelDetail, &m.detailViewport)
	logComponent := component.NewViewportComponent(PanelLog, &m.logViewport)

	// Create rename modal - a TextInputModal that owns its own text input
	// and communicates via messages (TextInputConfirmedMsg, TextInputCancelledMsg)
	m.renameModal = component.NewTextInputModal("rename-modal")
	m.renameModal.Input().Prompt = "Name: "
	m.renameModal.Input().CharLimit = 32
	m.renameModal.Input().Styles = textinput.DefaultStyles(true)

	// Create modal for creating rooms/zones - same pattern as rename
	m.createModal = component.NewTextInputModal("create-modal")
	m.createModal.Input().Prompt = "Name: "
	m.createModal.Input().CharLimit = 32
	m.createModal.Input().Styles = textinput.DefaultStyles(true)

	// Detail area uses a StackedContainer so modals can overlay the viewport
	m.detailStack = component.NewStackedContainer(detailViewport, m.renameModal, m.createModal)

	m.componentRoot = component.NewHSplit(
		component.ComponentChild{Size: component.Flex(0.4), Component: treeComponent},
		component.ComponentChild{Size: component.Flex(0.6), Component: component.NewVSplit(
			component.ComponentChild{Size: component.Flex(0.67), Component: m.detailStack},
			component.ComponentChild{Size: component.Flex(0.33), Component: logComponent},
		)},
	)

	// Set initial focus on tree panel
	treeComponent.Focus()

	return m
}

// Init implements tea.Model.
func (m Model) Init() tea.Cmd {
	return tea.Batch(
		m.loadBridgesFromCredentials(),
		discoverBridges(),        // Initial discovery
		m.startStateSaveTicker(), // Periodic state saves
		startDiscoveryTicker(),   // Periodic bridge discovery
	)
}

// Update implements tea.Model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	// Handle blink tick messages from grid component
	if blinkMsg, ok := msg.(field.BlinkTickMsg); ok {
		if m.lightGrid != nil {
			_, cmd := m.lightGrid.Update(blinkMsg)
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
			m.updateDetailContent()
			return m, tea.Batch(cmds...)
		}
	}

	// Handle field changed messages from new form component
	if fieldMsg, ok := msg.(field.FieldChangedMsg); ok {
		m.handleNewFieldChange(fieldMsg)
		return m, nil
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		// Check for minimum usable width
		const minWidth = 80
		if m.width < minWidth {
			m.status = fmt.Sprintf("Terminal too narrow (min %d cols, got %d)", minWidth, m.width)
		}

		// Reserve space for help
		helpHeight := 1 // status line only
		contentHeight := m.height - helpHeight

		// Update old layout (for rendering)
		m.layout.Layout(m.width, contentHeight)

		// Update component tree layout (for event routing)
		m.componentRoot.Layout(component.Rect{
			X:      0,
			Y:      0,
			Width:  m.width,
			Height: contentHeight,
		})

		// Update panel sizes
		treeBounds := m.layout.Bounds(PanelTree)
		m.tree.SetSize(treeBounds.Width, treeBounds.Height)

		detailBounds := m.layout.Bounds(PanelDetail)
		detailWidth := detailBounds.Width - 2
		if detailWidth < 1 {
			detailWidth = 1
		}
		detailHeight := detailBounds.Height - 2
		if detailHeight < 1 {
			detailHeight = 1
		}
		m.detailViewport.SetWidth(detailWidth)
		m.detailViewport.SetHeight(detailHeight)

		// Re-render detail content with new width
		m.updateDetailContent()

		logBounds := m.layout.Bounds(PanelLog)
		logWidth := logBounds.Width - 2
		if logWidth < 1 {
			logWidth = 1
		}
		logHeight := logBounds.Height - 2
		if logHeight < 1 {
			logHeight = 1
		}
		m.logViewport.SetWidth(logWidth)
		m.logViewport.SetHeight(logHeight)

		// Re-render log content with new width to handle truncation
		m.updateLogContent()

		m.help.Width = m.width
		return m, nil

	case bridgesDiscoveredMsg:
		// Store discovered bridges
		m.discoveredBridges = msg.bridges

		// Update status
		if len(msg.bridges) == 0 {
			if m.manager.BridgeCount() == 0 {
				m.status = "No bridges found. Press 'p' to pair."
			}
		} else {
			m.status = fmt.Sprintf("Found %d bridge(s)", len(msg.bridges))
		}

		// Add discovered bridges to manager (but don't connect - that happens separately)
		for _, info := range msg.bridges {
			// Check if we already have this bridge by ID
			existing := m.manager.GetBridge(info.ID)
			if existing != nil {
				// Update IP if changed
				if existing.Info.IPAddress != info.IPAddress {
					existing.Info.IPAddress = info.IPAddress
					// Update credentials
					if cred, ok := m.credentials.Get(info.ID); ok {
						cred.IPAddress = info.IPAddress
						m.credentials.Set(cred)
						_ = m.credentials.Save()
					}
				}
				continue
			}

			// Check by IP to avoid duplicates
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

			// Add new bridge to manager (but don't connect)
			m.manager.AddBridge(info)
		}

		// Rebuild tree to show newly discovered bridges
		m.rebuildTreeForActiveTab()

		// If user explicitly requested pairing (via 'p' key), start pairing with first unpaired bridge
		if m.pairingRequested {
			m.pairingRequested = false // Clear flag

			// Find first unpaired bridge
			var unpairedBridge *hue.BridgeInfo
			for i := range msg.bridges {
				if _, ok := m.credentials.Get(msg.bridges[i].ID); !ok {
					unpairedBridge = &msg.bridges[i]
					break
				}
			}

			if unpairedBridge != nil {
				// Start pairing mode
				m.pairing = true
				m.pairingFor = unpairedBridge
				m.pairingStartTime = time.Now()
				m.pairingRemaining = 60

				// Create context for pairing with 60s timeout
				ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
				m.pairingCancel = cancel

				// Switch to detail panel
				m.previousPane = m.focusedPane
				m.focusedPane = PanelDetail

				m.status = fmt.Sprintf("Pairing with %s...", unpairedBridge.Name)
				m.updateDetailContent()

				// Start pairing command and tick
				cmds = append(cmds, startPairing(ctx, *unpairedBridge))
				cmds = append(cmds, pairingTick())
			} else {
				// No unpaired bridges found
				m.status = "No unpaired bridges found"
			}
		}

		return m, tea.Batch(cmds...)

	case pairingTickMsg:
		if m.pairing {
			// Update countdown
			elapsed := time.Since(m.pairingStartTime)
			m.pairingRemaining = 60 - int(elapsed.Seconds())
			if m.pairingRemaining < 0 {
				m.pairingRemaining = 0
			}

			// Update UI
			m.updateDetailContent()

			// Continue ticking if still pairing
			if m.pairingRemaining > 0 {
				return m, pairingTick()
			} else {
				// Timeout - cancel pairing
				if m.pairingCancel != nil {
					m.pairingCancel()
				}
				m.pairing = false
				m.pairingFor = nil
				m.pairingCancel = nil
				m.status = "Pairing timed out"
				m.updateDetailContent()
			}
		}
		return m, nil

	case pairingSuccessMsg:
		if m.pairing {
			// Stop pairing mode
			m.pairing = false
			if m.pairingCancel != nil {
				m.pairingCancel()
			}
			pairingFor := m.pairingFor
			m.pairingFor = nil
			m.pairingCancel = nil

			// Save credentials
			m.credentials.Set(config.BridgeCredential{
				BridgeID:  msg.BridgeID,
				Name:      pairingFor.Name,
				IPAddress: pairingFor.IPAddress,
				ApiKey:    msg.ApiKey,
			})
			if err := m.credentials.Save(); err != nil {
				m.status = fmt.Sprintf("Pairing succeeded but failed to save credentials: %v", err)
			} else {
				m.status = fmt.Sprintf("Successfully paired with %s", pairingFor.Name)
			}

			// Add bridge to manager and connect
			m.manager.AddBridge(*pairingFor)
			if bridge := m.manager.GetBridge(msg.BridgeID); bridge != nil {
				cmds = append(cmds, connectBridge(bridge, msg.ApiKey))
			}

			// Update UI
			m.updateDetailContent()
		}
		return m, tea.Batch(cmds...)

	case pairingFailedMsg:
		if m.pairing {
			// Stop pairing mode
			m.pairing = false
			if m.pairingCancel != nil {
				m.pairingCancel()
			}
			pairingFor := m.pairingFor
			m.pairingFor = nil
			m.pairingCancel = nil

			m.status = fmt.Sprintf("Pairing failed with %s: %v", pairingFor.Name, msg.Err)
			m.updateDetailContent()
		}
		return m, nil

	case bridgeConnectedMsg:
		// Set up request and error logging callbacks
		bridge := m.manager.GetBridge(msg.bridgeID)
		if bridge != nil {
			ctx := context.Background()
			m.startRequestListener(ctx, bridge)
			m.startErrorListener(ctx, bridge)
			cmds = append(cmds, m.listenForRequests())
			cmds = append(cmds, m.listenForErrors())
		}

		// Sync state for this bridge and rebuild tree
		m.status = "Syncing state..."
		cmds = append(cmds, m.syncBridgeState(msg.bridgeID))
		m.rebuildTreeForActiveTab()
		return m, tea.Batch(cmds...)

	case stateSyncedMsg:
		m.rebuildTreeForActiveTab()

		bridge := m.manager.GetBridge(msg.bridgeID)
		if bridge != nil {
			// Start SSE listener for this bridge
			ctx, cancel := context.WithCancel(context.Background())
			// Cancel existing listener for this bridge if any
			if existingCancel, ok := m.eventCancelFuncs[msg.bridgeID]; ok {
				existingCancel()
			}
			m.eventCancelFuncs[msg.bridgeID] = cancel
			go m.startSSEListener(ctx, bridge)
			cmds = append(cmds, m.listenForEvents())

			// Rebuild tree and restore state after bridge state has loaded
			m.status = "Ready"
			m.rebuildTreeForActiveTab()
			// Restore UI state (only once, when first bridge connects)
			if len(m.manager.ConnectedBridges()) == 1 {
				m.restoreState()
			}
		}

		return m, tea.Batch(cmds...)

	case bridgeEventMsg:
		bridge := m.manager.GetBridge(msg.bridgeID)
		var state *hue.BridgeState
		bridgeName := ""
		if bridge != nil {
			state = bridge.GetState()
			bridgeName = bridge.Info.Name
		}
		event := parseEventFromBridgeCallback(msg.bridgeID, bridgeName, msg.resourceType, msg.resourceID, msg.eventType, state)
		m.activities = append(m.activities, event)
		m.updateLogContent()

		// Trigger blink for the bridge that received the event
		m.bridgeBlinkUntil[msg.bridgeID] = time.Now().Add(300 * time.Millisecond)
		cmds = append(cmds, tea.Tick(300*time.Millisecond, func(t time.Time) tea.Msg {
			return bridgeBlinkTickMsg{bridgeID: msg.bridgeID}
		}))

		if bridge := m.manager.GetBridge(msg.bridgeID); bridge != nil {
			m.rebuildTreeForActiveTab()
			// Update form if the event is for the currently selected light
			if msg.resourceType == "light" && msg.resourceID == m.selectedLightID {
				m.updateDetailContent()
			}
		}

		cmds = append(cmds, m.listenForEvents())
		return m, tea.Batch(cmds...)

	case requestMsg:
		bridgeName := ""
		if msg.bridgeID != "" {
			if bridge := m.manager.GetBridge(msg.bridgeID); bridge != nil {
				bridgeName = bridge.Info.Name
			}
		}
		m.activities = append(m.activities, &RequestActivity{
			timestamp:  time.Now(),
			bridgeID:   msg.bridgeID,
			bridgeName: bridgeName,
			message:    msg.message,
		})
		m.updateLogContent()
		// Keep listening for more requests
		cmds = append(cmds, m.listenForRequests())
		return m, tea.Batch(cmds...)

	case errorMsg:
		bridgeName := ""
		if msg.bridgeID != "" {
			if bridge := m.manager.GetBridge(msg.bridgeID); bridge != nil {
				bridgeName = bridge.Info.Name
			}
		}
		m.activities = append(m.activities, &ErrorActivity{
			timestamp:  time.Now(),
			bridgeID:   msg.bridgeID,
			bridgeName: bridgeName,
			message:    msg.message,
		})
		m.updateLogContent()
		m.status = fmt.Sprintf("Error: %s", msg.message)
		// Keep listening for more errors
		cmds = append(cmds, m.listenForErrors())
		return m, tea.Batch(cmds...)

	case stateSaveTickMsg:
		// Periodic state save to handle abrupt termination
		m.saveState()
		// Restart the ticker
		return m, m.startStateSaveTicker()

	case discoveryTickMsg:
		// Periodic bridge discovery
		cmds = append(cmds, discoverBridges())
		// Restart the ticker
		cmds = append(cmds, startDiscoveryTicker())
		return m, tea.Batch(cmds...)

	case bridgeBlinkTickMsg:
		// Clear blink state for this bridge
		if _, ok := m.bridgeBlinkUntil[msg.bridgeID]; ok {
			delete(m.bridgeBlinkUntil, msg.bridgeID)
			m.rebuildTreeForActiveTab()
		}
		return m, nil

	case errMsg:
		m.status = fmt.Sprintf("Error: %v", msg.err)
		m.activities = append(m.activities, &ErrorActivity{
			timestamp: time.Now(),
			message:   m.status,
		})
		m.updateLogContent()
		return m, nil

	case component.TextInputConfirmedMsg:
		// Handle confirmed input from text input modals
		switch msg.ModalID {
		case "rename-modal":
			cmd := m.confirmRenameWithValue(msg.Value)
			m.updateDetailContent()
			return m, cmd
		case "create-modal":
			cmd := m.confirmCreateWithValue(msg.Value)
			m.updateDetailContent()
			return m, cmd
		}
		return m, nil

	case component.TextInputCancelledMsg:
		// Handle cancelled input from text input modals
		switch msg.ModalID {
		case "rename-modal":
			m.cancelRename()
			m.updateDetailContent()
		case "create-modal":
			m.cancelCreate()
			m.updateDetailContent()
		}
		return m, nil

	case tea.KeyMsg:
		// 1. Modal input takes highest priority - route through component tree
		//    when any modal is active to capture all input
		if m.renameModal.IsActive() || m.createModal.IsActive() {
			if handled, cmd := m.componentRoot.RouteEvent(msg); handled {
				if cmd != nil {
					cmds = append(cmds, cmd)
				}
				// Refresh the detail panel to show updated input
				m.updateDetailContent()
				return m, tea.Batch(cmds...)
			}
		}

		// 2. Global keys - always available (except when modal is active, handled above)
		keyStr := msg.String()

		switch {
		case key.Matches(msg, m.keys.Quit):
			m.saveState() // Save state before quitting
			m.quitting = true
			for _, cancel := range m.eventCancelFuncs {
				cancel()
			}
			return m, tea.Quit

		case key.Matches(msg, m.keys.Help):
			// Toggle help display in detail panel
			m.showHelp = !m.showHelp
			if m.showHelp {
				m.previousPane = m.focusedPane
				m.focusedPane = PanelDetail
			}
			m.updateDetailContent()
			return m, nil

		case key.Matches(msg, m.keys.Rename):
			// Start rename mode for the selected entity
			if cmd := m.startRenameMode(); cmd != nil {
				return m, cmd
			}
			return m, nil

		case keyStr == "p":
			// Start bridge pairing
			if cmd := m.startBridgePairing(); cmd != nil {
				return m, cmd
			}
			return m, nil

		case key.Matches(msg, m.keys.NextBridge):
			m.tree.NextTab()
			m.rebuildTreeForActiveTab()
			m.saveState() // Persist tab change
			return m, nil

		case key.Matches(msg, m.keys.PrevBridge):
			m.tree.PrevTab()
			m.rebuildTreeForActiveTab()
			m.saveState() // Persist tab change
			return m, nil

		case key.Matches(msg, m.keys.NextPanel):
			switch m.focusedPane {
			case PanelTree:
				m.previousPane = m.focusedPane
				m.focusedPane = PanelDetail
			case PanelDetail:
				m.previousPane = m.focusedPane
				m.focusedPane = PanelLog
			default:
				m.previousPane = m.focusedPane
				m.focusedPane = PanelTree
			}
			return m, nil

		case key.Matches(msg, m.keys.PrevPanel):
			switch m.focusedPane {
			case PanelTree:
				m.previousPane = m.focusedPane
				m.focusedPane = PanelLog
			case PanelLog:
				m.previousPane = m.focusedPane
				m.focusedPane = PanelDetail
			default:
				m.previousPane = m.focusedPane
				m.focusedPane = PanelTree
			}
			return m, nil

		case key.Matches(msg, key.NewBinding(key.WithKeys("[", "alt+left"))):
			// Navigate back in history
			m.navigateBack()
			return m, nil

		case key.Matches(msg, key.NewBinding(key.WithKeys("]", "alt+right"))):
			// Navigate forward in history
			m.navigateForward()
			return m, nil

		case key.Matches(msg, key.NewBinding(key.WithKeys("esc"))):
			// Don't handle escape globally if in rename mode - let panel handle it
			if m.renaming {
				break
			}
			// Route to focused component first - it may want to handle escape
			if m.focusedPane == PanelDetail {
				if handled, cmd := m.lightGrid.RouteEvent(msg); handled {
					return m, cmd
				}
			}
			// Close help if showing
			if m.showHelp {
				m.showHelp = false
				m.updateDetailContent()
				return m, nil
			}
			// Escape returns to previous panel (or tree if already there)
			if m.focusedPane == PanelDetail || m.focusedPane == PanelLog {
				m.focusedPane = m.previousPane
				// If previous was also detail/log, default to tree
				if m.focusedPane == PanelDetail || m.focusedPane == PanelLog {
					m.focusedPane = PanelTree
				}
				m.previousPane = PanelTree
				return m, nil
			}
			// If in tree panel, let it handle escape (might collapse or something)
		}

		// 3. Panel focus shortcuts (number keys)
		for _, panelID := range m.panelOrder {
			panelKey := m.getPanelKey(panelID)
			if keyStr == panelKey {
				m.previousPane = m.focusedPane
				m.focusedPane = panelID
				return m, nil
			}
		}

	case tea.MouseWheelMsg:
		// Handle scroll wheel - route to panel under mouse cursor
		helpHeight := 1
		mouseY := msg.Y
		mouseX := msg.X

		// Check which panel the mouse is over
		if mouseY < m.height-helpHeight {
			if leaf := m.layout.At(mouseX, mouseY); leaf != nil {
				// Route scroll event to the panel under the mouse
				switch leaf.ID {
				case PanelTree:
					var cmd tea.Cmd
					m.tree, cmd = m.tree.Update(msg)
					if cmd != nil {
						cmds = append(cmds, cmd)
					}
					// Update detail content if selection changed
					if node := m.tree.SelectedNode(); node != nil {
						m.updateDetailContent()
						// Auto-focus detail panel when selecting a light (for easier mouse interaction)
						if node.Item != nil && node.Item.Type == panels.EntityLight {
							m.previousPane = m.focusedPane
							m.focusedPane = PanelDetail
						}
					}
					return m, tea.Batch(cmds...)

				case PanelDetail:
					// Route to grid component first
					if handled, cmd := m.lightGrid.RouteEvent(msg); handled {
						m.updateDetailContent() // Re-render to show cursor changes
						return m, cmd
					}
					// Fall back to viewport scroll
					var cmd tea.Cmd
					m.detailViewport, cmd = m.detailViewport.Update(msg)
					if cmd != nil {
						cmds = append(cmds, cmd)
					}
					return m, tea.Batch(cmds...)

				case PanelLog:
					// Pass to log viewport
					var cmd tea.Cmd
					m.logViewport, cmd = m.logViewport.Update(msg)
					if cmd != nil {
						cmds = append(cmds, cmd)
					}
					return m, tea.Batch(cmds...)
				}
			}
		}

	case tea.MouseClickMsg:
		// Handle panel focus switching on click
		// Don't consume the message - let panel-specific handlers process it too
		if msg.Button == tea.MouseLeft {
			// Account for help bar at bottom
			helpHeight := 1 // status line only
			clickY := msg.Y
			clickX := msg.X

			// Check if click is in a panel (excluding help area)
			if clickY < m.height-helpHeight {
				// Layout uses contentHeight (height - helpHeight), so coordinates are already correct
				if leaf := m.layout.At(clickX, clickY); leaf != nil {
					if leaf.ID != m.focusedPane {
						m.previousPane = m.focusedPane
						m.focusedPane = leaf.ID
						// If switching to detail panel, update content
						if m.focusedPane == PanelDetail {
							m.updateDetailContent()
						}
						// Don't return - fall through so the click also performs the action
					}
				}
			}
		}
		// Fall through to panel-specific handlers
	}

	// Pass events to focused panel
	switch m.focusedPane {
	case PanelTree:
		// Handle tree-specific keys
		if keyMsg, ok := msg.(tea.KeyMsg); ok {
			switch keyMsg.String() {
			case "enter":
				node := m.tree.SelectedNode()
				if node != nil && node.Item != nil {
					// Rooms should expand/collapse like other grouping items
					// User can use Tab to navigate to detail panel if needed
					if node.Item.Type == panels.EntityRoom {
						// Let tree handle it (toggle expand)
					} else {
						// Navigate to details panel for other entity types
						m.previousPane = PanelTree
						m.focusedPane = PanelDetail
						m.updateDetailContent()
						return m, nil
					}
				}
				// If no item or room, let tree handle it (toggle expand)

			case "x":
				// Delete selected item (bridge, room, or zone) - show confirmation
				item := m.tree.SelectedItem()
				if item == nil {
					m.status = "No item selected"
					return m, nil
				}

				// Route to appropriate delete confirmation
				switch item.Type {
				case panels.EntityBridge:
					if cmd := m.startBridgeDeleteConfirmation(); cmd != nil {
						return m, cmd
					}
				case panels.EntityRoom, panels.EntityZone:
					if cmd := m.startEntityDeleteConfirmation(); cmd != nil {
						return m, cmd
					}
				default:
					m.status = "Cannot delete this item"
				}
				return m, nil

			case "n":
				// Create new room
				if cmd := m.startCreateRoom(); cmd != nil {
					return m, cmd
				}
				return m, nil

			case "N":
				// Create new zone
				if cmd := m.startCreateZone(); cmd != nil {
					return m, cmd
				}
				return m, nil
			}
		}

		// Check for mouse click on tree item
		if mouseClick, ok := msg.(tea.MouseClickMsg); ok {
			if mouseClick.Button == tea.MouseLeft {
				// Check if this is a double-click (click on same item within 500ms)
				node := m.tree.SelectedNode()
				if node != nil && node.Item != nil && node.ID == m.lastTreeClickID {
					elapsed := time.Since(m.lastTreeClick)
					if elapsed < 500*time.Millisecond {
						// Double-click detected
						// Rooms should expand/collapse, not navigate to details
						if node.Item.Type != panels.EntityRoom {
							// Navigate to details for non-room entities
							m.previousPane = PanelTree
							m.focusedPane = PanelDetail
							m.updateDetailContent()
						}
						m.lastTreeClickID = "" // Reset to prevent triple-click navigation
						return m, nil
					}
				}
				// Update last click tracking
				if node != nil {
					m.lastTreeClick = time.Now()
					m.lastTreeClickID = node.ID
				}
			}
		}

		oldTabID := m.tree.ActiveTabID()
		var cmd tea.Cmd
		m.tree, cmd = m.tree.Update(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}

		// If tab changed, rebuild tree
		newTabID := m.tree.ActiveTabID()
		if newTabID != oldTabID {
			m.rebuildTreeForActiveTab()
		}

		// Update detail content if selection changed
		if node := m.tree.SelectedNode(); node != nil {
			m.updateDetailContent()
			// Don't auto-focus detail panel - let user explicitly navigate with Enter
		}

	case PanelDetail:
		// Handle pairing mode first
		if m.pairing {
			if keyMsg, ok := msg.(tea.KeyMsg); ok {
				switch keyMsg.String() {
				case "esc":
					// Cancel pairing
					m.cancelPairing()
					m.updateDetailContent()
					return m, nil
				}
			}
			return m, nil
		}

		// Handle delete confirmation mode
		if m.confirmingDelete {
			if keyMsg, ok := msg.(tea.KeyMsg); ok {
				switch keyMsg.String() {
				case "y", "Y":
					// Confirm deletion
					cmd := m.confirmBridgeDelete()
					m.updateDetailContent()
					return m, cmd
				case "n", "N", "esc":
					// Cancel deletion
					m.cancelBridgeDelete()
					m.updateDetailContent()
					return m, nil
				}
			}
			return m, nil
		}

		// Handle entity delete confirmation mode
		if m.confirmingDeleteEntity {
			if keyMsg, ok := msg.(tea.KeyMsg); ok {
				switch keyMsg.String() {
				case "y", "Y":
					// Confirm deletion
					cmd := m.confirmEntityDelete()
					m.updateDetailContent()
					return m, cmd
				case "n", "N", "esc":
					// Cancel deletion
					m.cancelEntityDelete()
					m.updateDetailContent()
					return m, nil
				}
			}
			return m, nil
		}

		// NOTE: Create room/zone mode is now handled early in Update() via handleCreateFormInput()
		// to prevent global keys from triggering while editing the name.

		// NOTE: Rename mode is now handled via TextInputModal in the component tree.
		// When renaming, events are routed through componentRoot.RouteEvent() which
		// sends TextInputConfirmedMsg/TextInputCancelledMsg that we handle above.

		// Route all events through the grid component first
		// The grid handles its own keyboard navigation, mouse clicks, and editing
		if handled, cmd := m.lightGrid.RouteEvent(msg); handled {
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
			m.updateDetailContent()
			return m, tea.Batch(cmds...)
		}

		// Handle Escape key to return to previous panel (form didn't consume it)
		if keyMsg, ok := msg.(tea.KeyMsg); ok {
			if keyMsg.String() == "esc" {
				m.focusedPane = m.previousPane
				if m.focusedPane == PanelDetail || m.focusedPane == PanelLog {
					m.focusedPane = PanelTree
				}
				m.previousPane = PanelTree
				return m, nil
			}
		}

		// Form didn't handle it - pass to viewport for scrolling
		var cmd tea.Cmd
		m.detailViewport, cmd = m.detailViewport.Update(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}

	case PanelLog:
		var cmd tea.Cmd
		m.logViewport, cmd = m.logViewport.Update(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	}

	return m, tea.Batch(cmds...)
}

// rebuildTreeForActiveTab rebuilds the tree based on the active tab.

// View implements tea.Model.
func (m Model) View() string {
	if m.quitting {
		return "Goodbye!\n"
	}

	var output strings.Builder

	// Calculate panel bounds
	detailBounds := m.layout.Bounds(PanelDetail)
	logBounds := m.layout.Bounds(PanelLog)

	// Render left column: tree
	treeContent := m.tree.View(m.focusedPane == PanelTree)
	leftColumn := treeContent

	// Get panel keys from panelOrder
	detailKey := m.getPanelKey(PanelDetail)
	logKey := m.getPanelKey(PanelLog)

	// Render right column: detail + log
	detailContent := m.renderDetailPanel(detailBounds.Width, detailBounds.Height, m.focusedPane == PanelDetail, detailKey)
	logContent := m.renderLogPanel(logBounds.Width, logBounds.Height, m.focusedPane == PanelLog, logKey)
	rightColumn := lipgloss.JoinVertical(lipgloss.Left, detailContent, logContent)

	// Compose layout - constrain each column to prevent terminal overflow
	treeBounds := m.layout.Bounds(PanelTree)
	if lipgloss.Width(leftColumn) > treeBounds.Width {
		leftColumn = ansi.Truncate(leftColumn, treeBounds.Width, "")
	}

	rightMaxWidth := m.width - treeBounds.Width
	if lipgloss.Width(rightColumn) > rightMaxWidth {
		rightColumn = ansi.Truncate(rightColumn, rightMaxWidth, "")
	}

	mainContent := lipgloss.JoinHorizontal(lipgloss.Top, leftColumn, rightColumn)

	// Render status line with minimal help hint
	statusStyle := lipgloss.NewStyle().
		Foreground(m.styles.Theme.TextMuted).
		PaddingLeft(1)
	helpHint := lipgloss.NewStyle().
		Foreground(m.styles.Theme.TextMuted).
		Render("  ?:help  q:quit")
	statusLine := statusStyle.Render(m.status) + helpHint

	// Combine with status (no full help bar at bottom anymore)
	full := lipgloss.JoinVertical(lipgloss.Left, mainContent, statusLine)

	// Constrain to terminal size
	result := lipgloss.Place(m.width, m.height, lipgloss.Left, lipgloss.Top, full)
	output.WriteString(result)

	return m.zones.Scan(output.String())
}

// getPanelKey returns the hotkey for a panel based on its position in panelOrder.
func (m *Model) getPanelKey(panelID string) string {
	for i, id := range m.panelOrder {
		if id == panelID {
			return fmt.Sprintf("%d", i)
		}
	}
	return ""
}

// rebuildTreeForActiveTab rebuilds the tree for the active tab, showing all bridges.
func (m *Model) rebuildTreeForActiveTab() {
	// All tree builders now show entities from all bridges
	tabID := m.tree.ActiveTabID()

	switch tabID {
	case "home":
		m.buildHomeTree(nil)
	case "lights":
		m.buildLightsTree(nil)
	case "devices":
		m.buildDevicesTree(nil)
	case "scenes":
		m.buildScenesTree(nil)
	default:
		m.buildHomeTree(nil)
	}
}

// startRenameMode initiates rename mode for the selected entity.
func (m *Model) startRenameMode() tea.Cmd {
	// Close help if showing
	m.showHelp = false

	// Get the selected entity from the tree
	item := m.tree.SelectedItem()
	if item == nil {
		m.status = "No entity selected"
		return nil
	}

	// Find which bridge this entity belongs to
	bridgeID := m.findBridgeForEntity(item)
	if bridgeID == "" {
		m.status = "Could not find bridge for entity"
		return nil
	}

	// Determine what to rename based on entity type
	renameType := item.Type
	renameID := item.ID
	renameName := item.Name

	switch item.Type {
	case panels.EntityDevice, panels.EntityRoom, panels.EntityZone, panels.EntityScene:
		// These are directly renameable
	case panels.EntityLight:
		// Lights are renamed via their owning device - find the device
		// Use RawPtr if available since it has the light data
		var light hueclient.LightGet
		var hasLight bool
		if item.RawPtr != nil {
			if l, ok := item.RawPtr.(hueclient.LightGet); ok {
				light = l
				hasLight = true
			}
		}
		if !hasLight {
			// Fallback to state lookup
			bridge := m.manager.GetBridge(bridgeID)
			if bridge != nil {
				if state := bridge.GetState(); state != nil {
					light, hasLight = state.GetLight(item.ID)
				}
			}
		}
		if !hasLight {
			m.status = "Light not found"
			return nil
		}
		if light.Owner == nil || light.Owner.Rid == nil {
			m.status = "Light has no owning device"
			return nil
		}
		deviceID := *light.Owner.Rid
		bridge := m.manager.GetBridge(bridgeID)
		if bridge == nil {
			m.status = "Bridge not found"
			return nil
		}
		state := bridge.GetState()
		if state == nil {
			m.status = "Bridge state not available"
			return nil
		}
		device, ok := state.GetDevice(deviceID)
		if !ok {
			m.status = "Owning device not found"
			return nil
		}
		// Rename the device instead
		renameType = panels.EntityDevice
		renameID = deviceID
		if device.Metadata != nil && device.Metadata.Name != nil {
			renameName = *device.Metadata.Name
		}
	default:
		m.status = fmt.Sprintf("Cannot rename %s", item.Type.String())
		return nil
	}

	// Set up rename state
	m.renaming = true
	m.renameEntityType = renameType
	m.renameEntityID = renameID
	m.renameBridgeID = bridgeID
	m.renameOriginalName = renameName

	// Blur all components so the modal can capture input exclusively
	component.BlurAll(m.componentRoot)

	// Set up the rename modal with the current name
	m.renameModal.SetValue(renameName)
	m.renameModal.SetActive(true)

	// Switch to detail panel
	m.previousPane = m.focusedPane
	m.focusedPane = PanelDetail

	m.status = fmt.Sprintf("Renaming %s...", renameType.String())

	// Update the UI immediately
	m.updateDetailContent()

	return nil
}

// findBridgeForEntity finds the bridge ID that owns the given entity.
func (m *Model) findBridgeForEntity(item *panels.EntityItem) string {
	// Return the bridge ID stored in the entity item
	return item.BridgeID
}

// cancelRename cancels rename mode and restores state.
func (m *Model) cancelRename() {
	m.renaming = false
	m.renameEntityID = ""
	m.renameEntityType = 0
	m.renameBridgeID = ""
	m.renameOriginalName = ""
	m.renameModal.SetActive(false) // Deactivate modal to release input capture
	m.status = "Rename cancelled"
}

// confirmRenameWithValue saves the new name and exits rename mode.
// This is called via TextInputConfirmedMsg from the rename modal.
func (m *Model) confirmRenameWithValue(newName string) tea.Cmd {
	if newName == "" {
		m.status = "Name cannot be empty"
		return nil
	}

	if newName == m.renameOriginalName {
		// No change
		m.cancelRename()
		m.status = "No change"
		return nil
	}

	// Get the bridge
	bridge := m.manager.GetBridge(m.renameBridgeID)
	if bridge == nil {
		m.status = "Bridge not found"
		m.cancelRename()
		return nil
	}

	// Call the appropriate rename function
	var err error
	switch m.renameEntityType {
	case panels.EntityDevice:
		err = bridge.RenameDevice(m.renameEntityID, newName)
	case panels.EntityRoom:
		err = bridge.RenameRoom(m.renameEntityID, newName)
	case panels.EntityZone:
		err = bridge.RenameZone(m.renameEntityID, newName)
	case panels.EntityScene:
		err = bridge.RenameScene(m.renameEntityID, newName)
	default:
		m.status = fmt.Sprintf("Cannot rename %s", m.renameEntityType.String())
		m.cancelRename()
		return nil
	}

	if err != nil {
		m.status = fmt.Sprintf("Rename failed: %v", err)
	} else {
		m.status = fmt.Sprintf("Renamed to \"%s\"", newName)
	}

	// Exit rename mode
	m.renaming = false
	m.renameEntityID = ""
	m.renameEntityType = 0
	m.renameBridgeID = ""
	m.renameOriginalName = ""
	m.renameModal.SetActive(false) // Deactivate modal to release input capture

	return nil
}

// startBridgeDeleteConfirmation initiates bridge deletion confirmation.
func (m *Model) startBridgeDeleteConfirmation() tea.Cmd {
	// Close help if showing
	m.showHelp = false

	// Get the selected item from tree - must be a bridge
	item := m.tree.SelectedItem()
	if item == nil || item.Type != panels.EntityBridge {
		m.status = "No bridge selected"
		return nil
	}

	bridgeID := item.ID
	bridge := m.manager.GetBridge(bridgeID)
	if bridge == nil {
		m.status = "Bridge not found"
		return nil
	}

	// Set confirmation state
	m.confirmingDelete = true
	m.deleteBridgeID = bridgeID
	m.deleteBridgeName = bridge.Info.Name

	// Switch to detail panel to show confirmation
	m.previousPane = m.focusedPane
	m.focusedPane = PanelDetail

	m.status = "Confirm bridge deletion..."

	// Update UI to show confirmation dialog
	m.updateDetailContent()

	return nil
}

// cancelBridgeDelete cancels the bridge deletion.
func (m *Model) cancelBridgeDelete() {
	m.confirmingDelete = false
	m.deleteBridgeID = ""
	m.deleteBridgeName = ""
	m.status = "Bridge deletion cancelled"
}

// confirmBridgeDelete performs the actual bridge deletion.
func (m *Model) confirmBridgeDelete() tea.Cmd {
	if m.deleteBridgeID == "" {
		m.status = "No bridge to delete"
		m.confirmingDelete = false
		return nil
	}

	// Remove the bridge from the manager (also removes credentials)
	if !m.manager.RemoveBridge(m.deleteBridgeID) {
		m.status = "Failed to remove bridge"
		m.confirmingDelete = false
		return nil
	}

	// Save credentials after deletion
	if err := m.credentials.Save(); err != nil {
		m.status = fmt.Sprintf("Bridge removed but failed to save credentials: %v", err)
	} else {
		m.status = fmt.Sprintf("Bridge \"%s\" deleted", m.deleteBridgeName)
	}

	// Clear confirmation state
	bridgeName := m.deleteBridgeName
	m.confirmingDelete = false
	m.deleteBridgeID = ""
	m.deleteBridgeName = ""

	// Rebuild tree to reflect deletion
	m.rebuildTreeForActiveTab()

	// Log the deletion
	m.activities = append(m.activities, &RequestActivity{
		timestamp: time.Now(),
		message:   fmt.Sprintf("Bridge \"%s\" deleted", bridgeName),
	})
	m.updateLogContent()

	return nil
}

// startEntityDeleteConfirmation initiates entity deletion confirmation.
func (m *Model) startEntityDeleteConfirmation() tea.Cmd {
	// Close help if showing
	m.showHelp = false

	// Get the selected item from tree
	item := m.tree.SelectedItem()
	if item == nil {
		m.status = "No entity selected"
		return nil
	}

	// Only rooms and zones can be deleted
	if item.Type != panels.EntityRoom && item.Type != panels.EntityZone {
		m.status = "Only rooms and zones can be deleted"
		return nil
	}

	// Find which bridge this entity belongs to
	bridgeID := m.findBridgeForEntity(item)
	if bridgeID == "" {
		m.status = "Could not find bridge for entity"
		return nil
	}

	// Set confirmation state
	m.confirmingDeleteEntity = true
	m.deleteEntityID = item.ID
	m.deleteEntityName = item.Name
	m.deleteEntityType = item.Type
	m.deleteEntityBridgeID = bridgeID

	// Switch to detail panel to show confirmation
	m.previousPane = m.focusedPane
	m.focusedPane = PanelDetail

	m.status = fmt.Sprintf("Confirm %s deletion...", item.Type.String())

	// Update UI to show confirmation dialog
	m.updateDetailContent()

	return nil
}

// cancelEntityDelete cancels the entity deletion.
func (m *Model) cancelEntityDelete() {
	m.confirmingDeleteEntity = false
	m.deleteEntityID = ""
	m.deleteEntityName = ""
	m.deleteEntityType = 0
	m.deleteEntityBridgeID = ""
	m.status = "Deletion cancelled"
}

// confirmEntityDelete performs the actual entity deletion.
func (m *Model) confirmEntityDelete() tea.Cmd {
	if m.deleteEntityID == "" {
		m.status = "No entity to delete"
		m.confirmingDeleteEntity = false
		return nil
	}

	// Get the bridge
	bridge := m.manager.GetBridge(m.deleteEntityBridgeID)
	if bridge == nil {
		m.status = "Bridge not found"
		m.confirmingDeleteEntity = false
		return nil
	}

	// Call the appropriate delete function
	var err error
	switch m.deleteEntityType {
	case panels.EntityRoom:
		err = bridge.DeleteRoom(m.deleteEntityID)
	case panels.EntityZone:
		err = bridge.DeleteZone(m.deleteEntityID)
	default:
		m.status = fmt.Sprintf("Cannot delete %s", m.deleteEntityType.String())
		m.confirmingDeleteEntity = false
		return nil
	}

	if err != nil {
		m.status = fmt.Sprintf("Delete failed: %v", err)
	} else {
		m.status = fmt.Sprintf("%s \"%s\" deleted", m.deleteEntityType.String(), m.deleteEntityName)
	}

	// Clear confirmation state
	entityName := m.deleteEntityName
	entityType := m.deleteEntityType
	m.confirmingDeleteEntity = false
	m.deleteEntityID = ""
	m.deleteEntityName = ""
	m.deleteEntityType = 0
	m.deleteEntityBridgeID = ""

	// Rebuild tree to reflect deletion
	m.rebuildTreeForActiveTab()

	// Log the deletion
	m.activities = append(m.activities, &RequestActivity{
		timestamp: time.Now(),
		message:   fmt.Sprintf("%s \"%s\" deleted", entityType.String(), entityName),
	})

	return nil
}

// startCreateRoom initiates room creation using the modal.
func (m *Model) startCreateRoom() tea.Cmd {
	// Close help if showing
	m.showHelp = false

	// Find the bridge to create the room in
	bridgeID := m.findBridgeForCurrentContext()
	if bridgeID == "" {
		m.status = "No bridge available"
		return nil
	}

	// Set creation state
	m.creatingRoom = true
	m.creatingZone = false
	m.createBridgeID = bridgeID

	// Activate the create modal
	component.BlurAll(m.componentRoot)
	m.createModal.SetActive(true)
	m.createModal.Input().SetValue("")
	m.createModal.Input().Focus()

	// Switch to detail panel to show the modal
	m.previousPane = m.focusedPane
	m.focusedPane = PanelDetail

	m.status = "Enter room name..."
	m.updateDetailContent()

	return nil
}

// startCreateZone initiates zone creation using the modal.
func (m *Model) startCreateZone() tea.Cmd {
	// Close help if showing
	m.showHelp = false

	// Find the bridge to create the zone in
	bridgeID := m.findBridgeForCurrentContext()
	if bridgeID == "" {
		m.status = "No bridge available"
		return nil
	}

	// Set creation state
	m.creatingRoom = false
	m.creatingZone = true
	m.createBridgeID = bridgeID

	// Activate the create modal
	component.BlurAll(m.componentRoot)
	m.createModal.SetActive(true)
	m.createModal.Input().SetValue("")
	m.createModal.Input().Focus()

	// Switch to detail panel to show the modal
	m.previousPane = m.focusedPane
	m.focusedPane = PanelDetail

	m.status = "Enter zone name..."
	m.updateDetailContent()

	return nil
}

// cancelCreate cancels room/zone creation.
func (m *Model) cancelCreate() {
	m.creatingRoom = false
	m.creatingZone = false
	m.createBridgeID = ""
	m.createModal.SetActive(false)
	m.status = "Creation cancelled"
}

// confirmCreateWithValue performs the actual room/zone creation with the given name.
func (m *Model) confirmCreateWithValue(name string) tea.Cmd {
	if name == "" {
		m.status = "Name is required"
		return nil
	}

	// Get the bridge
	bridge := m.manager.GetBridge(m.createBridgeID)
	if bridge == nil {
		m.status = "Bridge not found"
		m.cancelCreate()
		return nil
	}

	// Use "other" as default archetype
	archetype := hueclient.RoomArchetypeOther

	// Call the appropriate create function
	var err error
	var entityType string
	if m.creatingRoom {
		entityType = "Room"
		err = bridge.CreateRoom(name, archetype, nil)
	} else if m.creatingZone {
		entityType = "Zone"
		err = bridge.CreateZone(name, archetype, nil)
	}

	if err != nil {
		m.status = fmt.Sprintf("Create failed: %v", err)
	} else {
		m.status = fmt.Sprintf("%s \"%s\" created", entityType, name)
	}

	// Clear creation state
	m.creatingRoom = false
	m.creatingZone = false
	m.createBridgeID = ""
	m.createModal.SetActive(false)

	// Rebuild tree to reflect creation
	m.rebuildTreeForActiveTab()

	// Log the creation
	m.activities = append(m.activities, &RequestActivity{
		timestamp: time.Now(),
		message:   fmt.Sprintf("%s \"%s\" created", entityType, name),
	})
	m.updateLogContent()

	return nil
}

// findBridgeForCurrentContext returns the bridge ID for the current selection context.
func (m *Model) findBridgeForCurrentContext() string {
	// First try to find from selected item
	if item := m.tree.SelectedItem(); item != nil {
		if bridgeID := m.findBridgeForEntity(item); bridgeID != "" {
			return bridgeID
		}
	}

	// Fall back to first connected bridge
	bridges := m.manager.AllBridges()
	for _, bridge := range bridges {
		if bridge.IsConnected() {
			return bridge.Info.ID
		}
	}

	return ""
}

// startBridgePairing starts the bridge pairing process.
func (m *Model) startBridgePairing() tea.Cmd {
	// Close help if showing
	m.showHelp = false

	// Set flag that user requested pairing
	m.pairingRequested = true

	// Start discovery - when it completes, it will check pairingRequested flag
	m.status = "Discovering bridges..."

	return discoverBridges()
}

// cancelPairing cancels the current pairing operation.
func (m *Model) cancelPairing() {
	if m.pairingCancel != nil {
		m.pairingCancel()
	}
	m.pairing = false
	m.pairingFor = nil
	m.pairingCancel = nil
	m.status = "Pairing cancelled"
}

// navigateToEntity navigates to a specific entity by type and ID.
func (m *Model) navigateToEntity(entityType, entityID, bridgeID string) {
	// Find the entity in the tree
	node := m.tree.FindEntity(entityType, entityID)
	if node == nil {
		m.status = fmt.Sprintf("Entity not found: %s %s", entityType, entityID)
		return
	}

	// Add current position to history before navigating (if we have a current selection)
	if currentNode := m.tree.SelectedNode(); currentNode != nil && currentNode.Item != nil {
		// Only add to history if we're not already in the middle of history navigation
		if m.historyIndex == -1 || m.historyIndex == len(m.navigationHistory)-1 {
			// At the end of history or no history - add new entry
			entry := NavigationEntry{
				EntityType: currentNode.Item.Type.String(),
				EntityID:   currentNode.Item.ID,
				BridgeID:   currentNode.Item.BridgeID,
				Timestamp:  time.Now(),
			}
			m.navigationHistory = append(m.navigationHistory, entry)
			m.historyIndex = len(m.navigationHistory) - 1
		} else {
			// In the middle of history - truncate forward history and add new entry
			m.navigationHistory = m.navigationHistory[:m.historyIndex+1]
			entry := NavigationEntry{
				EntityType: currentNode.Item.Type.String(),
				EntityID:   currentNode.Item.ID,
				BridgeID:   currentNode.Item.BridgeID,
				Timestamp:  time.Now(),
			}
			m.navigationHistory = append(m.navigationHistory, entry)
			m.historyIndex = len(m.navigationHistory) - 1
		}
	}

	// Select the node in the tree
	m.tree.SelectNode(node)
	m.saveState() // Persist selection

	// Update detail panel
	m.updateDetailContent()

	// Auto-focus detail panel for easier interaction
	m.previousPane = m.focusedPane
	m.focusedPane = PanelDetail

	m.status = fmt.Sprintf("Navigated to %s", node.Item.Name)
}

// navigateBack navigates backward in history.
func (m *Model) navigateBack() {
	if len(m.navigationHistory) == 0 || m.historyIndex <= 0 {
		m.status = "No previous navigation"
		return
	}

	// Move back in history
	m.historyIndex--
	entry := m.navigationHistory[m.historyIndex]

	// Find and select the entity
	node := m.tree.FindEntity(entry.EntityType, entry.EntityID)
	if node == nil {
		m.status = fmt.Sprintf("Entity no longer exists: %s", entry.EntityID)
		// Remove invalid entry and try again
		m.navigationHistory = append(m.navigationHistory[:m.historyIndex], m.navigationHistory[m.historyIndex+1:]...)
		if m.historyIndex >= len(m.navigationHistory) {
			m.historyIndex = len(m.navigationHistory) - 1
		}
		return
	}

	m.tree.SelectNode(node)
	m.saveState() // Persist selection
	m.updateDetailContent()
	m.previousPane = m.focusedPane
	m.focusedPane = PanelDetail

	m.status = fmt.Sprintf("Back: %s (%d/%d)", node.Item.Name, m.historyIndex+1, len(m.navigationHistory))
}

// navigateForward navigates forward in history.
func (m *Model) navigateForward() {
	if len(m.navigationHistory) == 0 || m.historyIndex >= len(m.navigationHistory)-1 {
		m.status = "No forward navigation"
		return
	}

	// Move forward in history
	m.historyIndex++
	entry := m.navigationHistory[m.historyIndex]

	// Find and select the entity
	node := m.tree.FindEntity(entry.EntityType, entry.EntityID)
	if node == nil {
		m.status = fmt.Sprintf("Entity no longer exists: %s", entry.EntityID)
		// Remove invalid entry and try again
		m.navigationHistory = append(m.navigationHistory[:m.historyIndex], m.navigationHistory[m.historyIndex+1:]...)
		if m.historyIndex >= len(m.navigationHistory) {
			m.historyIndex = len(m.navigationHistory) - 1
		}
		return
	}

	m.tree.SelectNode(node)
	m.saveState() // Persist selection
	m.updateDetailContent()
	m.previousPane = m.focusedPane
	m.focusedPane = PanelDetail

	m.status = fmt.Sprintf("Forward: %s (%d/%d)", node.Item.Name, m.historyIndex+1, len(m.navigationHistory))
}

// scrollDetailViewportToCursor scrolls the detail viewport to ensure the current form field is visible.
// With the new component-based form, each field tracks its own height.
func (m *Model) scrollDetailViewportToCursor() {
	// The new FormComponent handles its own cursor tracking
	// For now, we'll let the viewport handle scrolling naturally
	// TODO: Add Height() method to field components for precise scrolling
}
