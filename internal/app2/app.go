// Package app2 is the v2 application using bubbletea v2, bubbles v2, and lipgloss v2.
package app2

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/v2/help"
	"github.com/charmbracelet/bubbles/v2/key"
	"github.com/charmbracelet/bubbles/v2/viewport"
	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/charmbracelet/lipgloss/v2"
	zone "github.com/lrstanley/bubblezone/v2"

	"github.com/kluzzebass/lazyhue/internal/config"
	"github.com/kluzzebass/lazyhue/internal/hue"
	"github.com/kluzzebass/lazyhue/internal/ui2"
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
	activeBridgeID   string
	status           string
	logEntries       []LogEntry
	quitting         bool
	eventChan        chan bridgeEventMsg
	eventCancelFuncs map[string]context.CancelFunc
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
		status:           "Loading bridges...",
		logEntries:       []LogEntry{{Time: time.Now(), Type: "event", Message: "lazyhue v2 started"}},
		eventChan:        make(chan bridgeEventMsg, 100),
		eventCancelFuncs: make(map[string]context.CancelFunc),
	}
}

// Init implements tea.Model.
func (m Model) Init() tea.Cmd {
	return m.loadBridgesFromCredentials()
}

// Update implements tea.Model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		// Reserve space for help
		helpHeight := 3
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

	case bridgeConnectedMsg:
		m.logEntries = append(m.logEntries, LogEntry{
			Time:    time.Now(),
			Type:    "event",
			Message: fmt.Sprintf("Connected to bridge %s", msg.bridgeID),
		})
		m.updateLogContent()

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
		m.logEntries = append(m.logEntries, LogEntry{
			Time:    time.Now(),
			Type:    "event",
			Message: fmt.Sprintf("State synced for bridge %s", msg.bridgeID),
		})
		m.updateLogContent()
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
		details := m.buildEventDetails(msg)
		m.logEntries = append(m.logEntries, LogEntry{
			Time:           time.Now(),
			Type:           "event",
			ResourceType:   details.ResourceType,
			ResourceName:   details.ResourceName,
			Details:        details.Details,
			IndicatorColor: details.IndicatorColor,
			Brightness:     details.Brightness,
			IsOn:           details.IsOn,
		})
		m.updateLogContent()

		if bridge := m.manager.GetBridge(msg.bridgeID); bridge != nil {
			m.rebuildTreeForActiveTab()
		}

		cmds = append(cmds, m.listenForEvents())
		return m, tea.Batch(cmds...)

	case errMsg:
		m.status = fmt.Sprintf("Error: %v", msg.err)
		m.logEntries = append(m.logEntries, LogEntry{
			Time:    time.Now(),
			Type:    "error",
			Message: m.status,
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
			m.help.ShowAll = !m.help.ShowAll
			return m, nil

		case key.Matches(msg, m.keys.NextBridge):
			m.tree.NextTab()
			m.logEntries = append(m.logEntries, LogEntry{
				Time:    time.Now(),
				Type:    "event",
				Message: fmt.Sprintf("Tab: %s", m.tree.Title()),
			})
			m.updateLogContent()
			m.rebuildTreeForActiveTab()
			return m, nil

		case key.Matches(msg, m.keys.PrevBridge):
			m.tree.PrevTab()
			m.logEntries = append(m.logEntries, LogEntry{
				Time:    time.Now(),
				Type:    "event",
				Message: fmt.Sprintf("Tab: %s", m.tree.Title()),
			})
			m.updateLogContent()
			m.rebuildTreeForActiveTab()
			return m, nil

		case key.Matches(msg, m.keys.NextPanel):
			switch m.focusedPane {
			case PanelTree:
				m.focusedPane = PanelDetail
			case PanelDetail:
				m.focusedPane = PanelLog
			default:
				m.focusedPane = PanelTree
			}
			return m, nil
		}

	case tea.MouseClickMsg:
		if leaf := m.layout.At(msg.X, msg.Y-3); leaf != nil {
			if leaf.ID != m.focusedPane {
				m.focusedPane = leaf.ID
			}
		}
	}

	// Pass events to focused panel
	switch m.focusedPane {
	case PanelTree:
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

		if node := m.tree.SelectedNode(); node != nil {
			if node.Item != nil {
				m.updateDetailContent()
			} else {
				m.updateDetailContent()
			}
		}

	case PanelDetail:
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

	// Compose layout
	mainContent := lipgloss.JoinHorizontal(lipgloss.Top, leftColumn, rightColumn)

	// Combine with help
	full := lipgloss.JoinVertical(lipgloss.Left, mainContent, m.help.View(m.keys))

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
