// Package app is the application using bubbletea, bubbles, and lipgloss.
package app

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/v2/key"
	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/charmbracelet/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/kluzzebass/lazyhue/internal/config"
	"github.com/kluzzebass/lazyhue/internal/hue"
	"github.com/kluzzebass/lazyhue/internal/ui/component"
	"github.com/kluzzebass/lazyhue/internal/ui/component/field"
	"github.com/kluzzebass/lazyhue/internal/ui/panels"
)

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
			// Re-render grid view WITHOUT rebuilding rows (preserves edit state)
			if m.lightGrid.RowCount() > 0 {
				m.detailViewport.SetContent(m.lightGrid.View())
			}
			return m, tea.Batch(cmds...)
		}
	}

	// Handle field changed messages from new form component
	if fieldMsg, ok := msg.(field.FieldChangedMsg); ok {
		m.handleNewFieldChange(fieldMsg)
		return m, nil
	}

	switch msg := msg.(type) {
	case SignalQuitMsg:
		// Handle signal-triggered quit (SIGINT, SIGTERM, SIGHUP)
		m.saveState()
		m.quitting = true
		for _, cancel := range m.eventCancelFuncs {
			cancel()
		}
		return m, tea.Quit

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

		// Only update status during initial discovery (no bridges yet) or when pairing
		// Don't overwrite status during periodic discovery when bridges are connected
		if m.manager.BridgeCount() == 0 {
			if len(msg.bridges) == 0 {
				m.status = "No bridges found. Press 'p' to pair."
			} else {
				m.status = fmt.Sprintf("Found %d bridge(s)", len(msg.bridges))
			}
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
			// Restore UI state (only once, after first bridge state is synced)
			if !m.stateRestored {
				m.stateRestored = true
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
		event := parseEventFromUpdate(msg.bridgeID, bridgeName, msg.update, msg.eventType, msg.receivedAt, state)
		m.activities = append(m.activities, event)
		m.updateLogContent()

		// Trigger blink for the bridge that received the event
		m.bridgeBlinkUntil[msg.bridgeID] = time.Now().Add(300 * time.Millisecond)
		cmds = append(cmds, tea.Tick(300*time.Millisecond, func(t time.Time) tea.Msg {
			return bridgeBlinkTickMsg{bridgeID: msg.bridgeID}
		}))

		if bridge := m.manager.GetBridge(msg.bridgeID); bridge != nil {
			m.rebuildTreeForActiveTab()
			// Update detail panel if the event is for the currently selected entity
			// Skip if grid is in editing mode to preserve edit state
			if !m.lightGrid.IsEditing() {
				if node := m.tree.SelectedNode(); node != nil && node.Item != nil {
					selectedID := node.Item.ID
					// Check if event matches selected entity
					shouldUpdate := false
					switch msg.update.Type {
					case "light":
						shouldUpdate = msg.update.ID == m.selectedLightID || msg.update.ID == selectedID
					case "room", "zone", "scene", "device", "grouped_light":
						shouldUpdate = msg.update.ID == selectedID
					}
					if shouldUpdate {
						m.updateDetailContent()
					}
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

		// 2. Inline editing (e.g., rename in grid) takes priority over global keys
		if m.lightGrid.IsEditing() {
			if handled, cmd := m.lightGrid.RouteEvent(msg); handled {
				if cmd != nil {
					cmds = append(cmds, cmd)
				}
				// Re-render the grid view WITHOUT rebuilding rows (preserves edit state)
				if m.lightGrid.RowCount() > 0 {
					m.detailViewport.SetContent(m.lightGrid.View())
				}
				return m, tea.Batch(cmds...)
			}
		}

		// 3. Global keys - always available (except when modal/editing is active, handled above)
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

		case keyStr == "a":
			// Toggle activity log panel
			m.showActivity = !m.showActivity
			m.rebuildLayout()
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
			m.previousPane = m.focusedPane
			switch m.focusedPane {
			case PanelTree:
				m.focusedPane = PanelDetail
			case PanelDetail:
				if m.showActivity {
					m.focusedPane = PanelLog
				} else {
					m.focusedPane = PanelTree
				}
			default:
				m.focusedPane = PanelTree
			}
			return m, nil

		case key.Matches(msg, m.keys.PrevPanel):
			m.previousPane = m.focusedPane
			switch m.focusedPane {
			case PanelTree:
				if m.showActivity {
					m.focusedPane = PanelLog
				} else {
					m.focusedPane = PanelDetail
				}
			case PanelLog:
				m.focusedPane = PanelDetail
			default:
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
					// Update detail content to reflect scroll position
					// Note: Selection doesn't change on scroll (TreePanel preserves selection)
					m.updateDetailContent()
					return m, tea.Batch(cmds...)

				case PanelDetail:
					// Route to grid component first (for dropdowns, sliders, etc.)
					if handled, cmd := m.lightGrid.RouteEvent(msg); handled {
						// Don't rebuild grid - just re-render to preserve dropdown state
						var content strings.Builder
						if m.lightGrid.RowCount() > 0 {
							content.WriteString(m.lightGrid.View())
						}
						m.detailViewport.SetContent(content.String())
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
				// Delete selected item - show confirmation
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
				case panels.EntityRoom, panels.EntityZone, panels.EntityScene, panels.EntitySmartScene, panels.EntityDevice, panels.EntityLight:
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
			// Don't call updateDetailContent here - it would rebuild the grid and reset focus
			// Instead, just re-render the existing grid to show focus changes
			var content strings.Builder
			if m.lightGrid.RowCount() > 0 {
				content.WriteString(m.lightGrid.View())
			}
			m.detailViewport.SetContent(content.String())
			// Scroll viewport to keep focused row visible
			m.scrollDetailViewportToFocusedRow()
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

	// Render left column: tree
	treeContent := m.tree.View(m.focusedPane == PanelTree)
	leftColumn := treeContent

	// Get panel keys from panelOrder
	detailKey := m.getPanelKey(PanelDetail)

	// Render right column: detail (+ log if visible)
	detailContent := m.renderDetailPanel(detailBounds.Width, detailBounds.Height, m.focusedPane == PanelDetail, detailKey)
	var rightColumn string
	if m.showActivity {
		logBounds := m.layout.Bounds(PanelLog)
		logKey := m.getPanelKey(PanelLog)
		logContent := m.renderLogPanel(logBounds.Width, logBounds.Height, m.focusedPane == PanelLog, logKey)
		rightColumn = lipgloss.JoinVertical(lipgloss.Left, detailContent, logContent)
	} else {
		rightColumn = detailContent
	}

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