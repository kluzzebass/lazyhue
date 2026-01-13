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
	zone "github.com/lrstanley/bubblezone/v2"

	"github.com/kluzzebass/lazyhue/internal/config"
	"github.com/kluzzebass/lazyhue/internal/hue"
	"github.com/kluzzebass/lazyhue/internal/hueclient"
	"github.com/kluzzebass/lazyhue/internal/ui2"
	"github.com/kluzzebass/lazyhue/internal/ui2/components"
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
type Model struct {
	// Service layer
	manager     *hue.Manager
	credentials *config.CredentialStore

	// UI components
	tree           *panels.TreePanel
	detailViewport viewport.Model
	logViewport    viewport.Model
	keys           ui2.KeyMap
	help           help.Model
	styles         ui2.Styles
	zones          *zone.Manager
	layout         *layout.Tree

	// Panel management
	panelOrder []string // Panel IDs in focus-cycle order

	// State
	width            int
	height           int
	focusedPane      string
	previousPane     string // Track previous panel for Escape key
	activeBridgeID   string
	status           string
	activities       []Activity
	quitting         bool
	eventChan        chan bridgeEventMsg
	requestChan      chan requestMsg
	errorChan        chan errorMsg
	eventCancelFuncs map[string]context.CancelFunc
	bridgeBlinkUntil map[string]time.Time // Track when bridge blink indicators should stop
	lightForm        *components.Form     // Form for light controls in detail panel
	selectedLightID   string               // ID of currently selected light (for form updates)
	lastTreeClick     time.Time            // Track last tree item click for double-click detection
	lastTreeClickID   string               // Track which tree item was last clicked

	// Rename mode state
	renaming           bool                 // Whether we're in rename mode
	renameInput        textinput.Model      // Text input for renaming
	renameEntityType   panels.EntityType    // Type of entity being renamed
	renameEntityID     string               // ID of entity being renamed
	renameBridgeID     string               // Bridge ID the entity belongs to
	renameOriginalName string               // Original name for cancel

	// Help display
	showHelp bool // Whether help is displayed in detail panel
}

// New creates a new application model.
func New(creds *config.CredentialStore) Model {
	styles := ui2.DefaultStyles()
	keys := ui2.DefaultKeyMap()
	zones := zone.New()

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

	// Initialize rename text input
	renameInput := textinput.New()
	renameInput.Prompt = "Name: "
	renameInput.CharLimit = 32

	return Model{
		manager:          hue.NewManager(creds),
		credentials:      creds,
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
		bridgeBlinkUntil: make(map[string]time.Time),
		lightForm:        components.NewForm(&styles, zones),
		selectedLightID:  "",
		renameInput:      renameInput,
	}
}

// Init implements tea.Model.
func (m Model) Init() tea.Cmd {
	return m.loadBridgesFromCredentials()
}

// Update implements tea.Model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	// Handle blink tick messages from form first (before switch)
	if _, ok := msg.(components.BlinkTickMsg); ok {
		if m.lightForm != nil && len(m.lightForm.Fields) > 0 {
			var cmd tea.Cmd
			m.lightForm, cmd = m.lightForm.Update(msg)
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
			// Update detail content to reflect blink state change
			m.updateDetailContent()
			return m, tea.Batch(cmds...)
		}
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		// Reserve space for help
		helpHeight := 1 // status line only
		contentHeight := m.height - helpHeight

		// Update layout
		m.layout.Layout(m.width, contentHeight)

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

		m.help.Width = m.width
		return m, nil

	case bridgesDiscoveredMsg:
		// Add discovered bridges to the manager
		for _, info := range msg.bridges {
			// Add if not already in manager
			if m.manager.GetBridge(info.ID) == nil {
				m.manager.AddBridge(info)
			}
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

		// If this is the first connected bridge, make it active
		if m.activeBridgeID == "" {
			m.activeBridgeID = msg.bridgeID
			m.manager.SetActiveBridge(msg.bridgeID)
			m.status = "Syncing state..."
		}

		// Sync state for this bridge and rebuild tree
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

			// If this is the active bridge, rebuild tree
			if msg.bridgeID == m.activeBridgeID {
				m.status = "Ready"
				m.rebuildTreeForActiveTab()
			}
		}

		return m, tea.Batch(cmds...)

	case bridgeEventMsg:
		bridge := m.manager.GetBridge(msg.bridgeID)
		var state *hue.BridgeState
		if bridge != nil {
			state = bridge.GetState()
		}
		event := parseEventFromBridgeCallback(msg.bridgeID, msg.resourceType, msg.resourceID, msg.eventType, state)
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
			// BUT only if the form is not currently being edited (to prevent interrupting user input)
			if msg.resourceType == "light" && msg.resourceID == m.selectedLightID && state != nil && !m.lightForm.Editing {
				if light, ok := state.GetLight(msg.resourceID); ok {
					fields := m.buildLightFormFields(light)
					m.lightForm.SetFields(fields)
				}
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

	case tea.KeyMsg:
		// Check for panel focus shortcuts (number keys)
		keyStr := msg.String()

		// Check panels in panelOrder - keys are assigned automatically based on position
		// Keys are 0-based: first panel gets "0", second gets "1", etc.
		for _, panelID := range m.panelOrder {
			panelKey := m.getPanelKey(panelID)
			if keyStr == panelKey {
				m.previousPane = m.focusedPane
				m.focusedPane = panelID
				return m, nil
			}
		}

		switch {
		case key.Matches(msg, m.keys.Quit):
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

		case key.Matches(msg, m.keys.TestForm):
			m.showTestForm()
			return m, nil

		case key.Matches(msg, m.keys.Rename):
			// Start rename mode for the selected entity
			if cmd := m.startRenameMode(); cmd != nil {
				return m, cmd
			}
			return m, nil

		case key.Matches(msg, m.keys.NextBridge):
			m.tree.NextTab()
			m.rebuildTreeForActiveTab()
			return m, nil

		case key.Matches(msg, m.keys.PrevBridge):
			m.tree.PrevTab()
			m.rebuildTreeForActiveTab()
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

		case key.Matches(msg, key.NewBinding(key.WithKeys("esc"))):
			// Don't handle escape globally if in rename mode - let panel handle it
			if m.renaming {
				break
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
		// Handle Enter key to navigate to details panel
		if keyMsg, ok := msg.(tea.KeyMsg); ok {
			if keyMsg.String() == "enter" {
				node := m.tree.SelectedNode()
				if node != nil && node.Item != nil {
					// Navigate to details panel
					m.previousPane = PanelTree
					m.focusedPane = PanelDetail
					m.updateDetailContent()
					return m, nil
				}
				// If no item, let tree handle it (toggle expand)
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
						// Double-click detected - navigate to details
						m.previousPane = PanelTree
						m.focusedPane = PanelDetail
						m.updateDetailContent()
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
		}

	case PanelDetail:
		// Handle rename mode first - all keys go to rename input
		if m.renaming {
			if keyMsg, ok := msg.(tea.KeyMsg); ok {
				switch keyMsg.String() {
				case "enter":
					// Save and exit rename mode
					cmd := m.confirmRename()
					m.updateDetailContent()
					return m, cmd
				case "esc":
					// Cancel rename mode
					m.cancelRename()
					m.updateDetailContent()
					return m, nil
				default:
					// Pass key to text input
					var cmd tea.Cmd
					m.renameInput, cmd = m.renameInput.Update(msg)
					m.updateDetailContent()
					return m, cmd
				}
			}
			return m, nil
		}

		// Handle Escape key to return to previous panel
		if keyMsg, ok := msg.(tea.KeyMsg); ok {
			if keyMsg.String() == "esc" {
				// Check if form is editing - if so, cancel edit first
				if m.lightForm != nil && m.lightForm.Editing {
					var cmd tea.Cmd
					m.lightForm, cmd = m.lightForm.Update(msg)
					if cmd != nil {
						cmds = append(cmds, cmd)
					}
					m.updateDetailContent()
					return m, tea.Batch(cmds...)
				}
				// Otherwise, return to previous panel
				m.focusedPane = m.previousPane
				if m.focusedPane == PanelDetail || m.focusedPane == PanelLog {
					m.focusedPane = PanelTree
				}
				m.previousPane = PanelTree
				return m, nil
			}
		}

		// Pass mouse messages to form
		// Zones are scanned from FINAL output (after viewport, borders, etc.), so they're in SCREEN coordinates
		// We should pass screen coordinates directly to form
		formHandledMouse := false
		if m.lightForm != nil && len(m.lightForm.Fields) > 0 && m.focusedPane == PanelDetail {
			detailBounds := m.layout.Bounds(PanelDetail)

			// Handle mouse motion for dragging
			if mouseMotion, ok := msg.(tea.MouseMotionMsg); ok {
				// If form has active capture, pass motion events to it
				if m.lightForm.MouseCaptureIdx >= 0 {
					var cmd tea.Cmd
					m.lightForm, cmd = m.lightForm.Update(mouseMotion)
					if cmd != nil {
						cmds = append(cmds, cmd)
					}
					m.updateDetailContent()
					return m, tea.Batch(cmds...)
				}
			}

			// Handle mouse release to end drag
			if mouseRelease, ok := msg.(tea.MouseReleaseMsg); ok {
				if m.lightForm.MouseCaptureIdx >= 0 {
					var cmd tea.Cmd
					m.lightForm, cmd = m.lightForm.Update(mouseRelease)
					if cmd != nil {
						cmds = append(cmds, cmd)
					}
					return m, tea.Batch(cmds...)
				}
			}

			// Handle mouse clicks
			if mouseClick, ok := msg.(tea.MouseClickMsg); ok {
				// Check if click is within detail panel bounds
				if mouseClick.X >= detailBounds.X && mouseClick.X < detailBounds.X+detailBounds.Width &&
					mouseClick.Y >= detailBounds.Y && mouseClick.Y < detailBounds.Y+detailBounds.Height {

					// Pass screen coordinates directly - zones are in screen space after final scan
					// The form's handleMouseClick will check zones using the same zone manager
					var cmd tea.Cmd
					// Store old field values to detect changes (for toggles, etc.)
					oldFieldValues := make(map[int]int)
					for i, field := range m.lightForm.Fields {
						oldFieldValues[i] = field.Value
					}
					oldCursor := m.lightForm.Cursor
					oldCapture := m.lightForm.MouseCaptureIdx
					oldDropdownOpen := m.lightForm.DropdownOpen

					// Update form with mouse click - it will check zones internally
					// If there's an active capture, this will handle dragging
					m.lightForm, cmd = m.lightForm.Update(mouseClick)
					if cmd != nil {
						cmds = append(cmds, cmd)
					}

					// Check if form handled the click by checking:
					// 1. Cursor changed
					// 2. Mouse capture started/changed
					// 3. Field values changed (for toggles, etc.)
					// 4. Dropdown state changed
					// 5. Active capture exists (user is dragging - always update during drag)
					fieldChanged := false
					for i, field := range m.lightForm.Fields {
						if oldVal, ok := oldFieldValues[i]; ok && field.Value != oldVal {
							fieldChanged = true
							break
						}
					}
					dropdownChanged := m.lightForm.DropdownOpen != oldDropdownOpen
					if m.lightForm.Cursor != oldCursor || m.lightForm.MouseCaptureIdx != oldCapture || fieldChanged || dropdownChanged {
						formHandledMouse = true
						m.updateDetailContent()
						return m, tea.Batch(cmds...)
					}
					// If there's an active capture, always update (user is dragging)
					if m.lightForm.MouseCaptureIdx >= 0 {
						formHandledMouse = true
						m.updateDetailContent()
						return m, tea.Batch(cmds...)
					}
				}
			}
		}

		// Update light form if it exists and has fields (but skip if we already handled mouse click above)
		formHandledKey := false
		if m.lightForm != nil && len(m.lightForm.Fields) > 0 && !formHandledMouse {
			// Check if this is a key the form handles
			if keyMsg, ok := msg.(tea.KeyMsg); ok {
				keyStr := keyMsg.String()
				// These are keys the form handles for navigation/editing - always consume them
				formNavigationKeys := map[string]bool{
					"up": true, "down": true, "k": true, "j": true,
					"left": true, "right": true, "h": true, "l": true,
					"enter": true, " ": true,
				}
				if formNavigationKeys[keyStr] {
					// Always let form handle these keys when it has fields
					formHandledKey = true
					var cmd tea.Cmd
					m.lightForm, cmd = m.lightForm.Update(msg)
					if cmd != nil {
						cmds = append(cmds, cmd)
					}
					// Update detail content to reflect form changes
					// updateDetailContent preserves form state when editing, so this is safe
					m.updateDetailContent()
				} else {
					// Other keys, let form try to handle them first
					var cmd tea.Cmd
					m.lightForm, cmd = m.lightForm.Update(msg)
					if cmd != nil {
						cmds = append(cmds, cmd)
					}
				}
			} else if _, ok := msg.(tea.MouseClickMsg); !ok {
				// Non-key, non-mouse messages, let form handle them
				// (Mouse clicks are handled above, so skip them here)
				var cmd tea.Cmd
				m.lightForm, cmd = m.lightForm.Update(msg)
				if cmd != nil {
					cmds = append(cmds, cmd)
				}
			}
		}
		// Only update viewport if form didn't handle the key
		if !formHandledKey {
			var cmd tea.Cmd
			m.detailViewport, cmd = m.detailViewport.Update(msg)
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
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

	// Compose layout
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
	// This is now handled by buildHomeTree which shows all bridges
	if m.tree.ActiveTabID() == "home" {
		m.buildHomeTree(nil) // Pass nil to build for all bridges
	} else {
		// For lights/scenes tabs, still use active bridge for now
		bridge := m.manager.GetBridge(m.activeBridgeID)
		if bridge == nil {
			return
		}
		state := bridge.GetState()
		tabID := m.tree.ActiveTabID()

		switch tabID {
		case "lights":
			m.buildLightsTree(state)
		case "devices":
			m.buildDevicesTree(state)
		case "scenes":
			m.buildScenesTree(state)
		default:
			m.buildHomeTree(nil)
		}
	}
}

// startRenameMode initiates rename mode for the selected entity.
func (m *Model) startRenameMode() tea.Cmd {
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

	// Initialize the text input with current name
	m.renameInput.SetValue(renameName)
	m.renameInput.Focus()
	m.renameInput.CursorEnd()

	// Switch to detail panel
	m.previousPane = m.focusedPane
	m.focusedPane = PanelDetail

	m.status = fmt.Sprintf("Renaming %s...", renameType.String())

	// Update the UI immediately
	m.updateDetailContent()

	return textinput.Blink
}

// findBridgeForEntity finds the bridge ID that owns the given entity.
func (m *Model) findBridgeForEntity(item *panels.EntityItem) string {
	// Use the bridge ID stored in the entity item
	if item.BridgeID != "" {
		return item.BridgeID
	}
	// Fallback to active bridge if no bridge ID is stored (shouldn't happen)
	return m.activeBridgeID
}

// cancelRename cancels rename mode and restores state.
func (m *Model) cancelRename() {
	m.renaming = false
	m.renameEntityID = ""
	m.renameEntityType = 0
	m.renameBridgeID = ""
	m.renameOriginalName = ""
	m.renameInput.Blur()
	m.status = "Rename cancelled"
}

// confirmRename saves the new name and exits rename mode.
func (m *Model) confirmRename() tea.Cmd {
	newName := m.renameInput.Value()
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
	m.renameInput.Blur()

	return nil
}
