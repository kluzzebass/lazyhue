package app

import (
	"context"
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/v2/help"
	"github.com/charmbracelet/bubbles/v2/textinput"
	"github.com/charmbracelet/bubbles/v2/viewport"
	zone "github.com/lrstanley/bubblezone/v2"

	"github.com/kluzzebass/lazyhue/internal/config"
	"github.com/kluzzebass/lazyhue/internal/hue"
	"github.com/kluzzebass/lazyhue/internal/ui"
	"github.com/kluzzebass/lazyhue/internal/ui/component"
	gridlayout "github.com/kluzzebass/lazyhue/internal/ui/component/layout"
	"github.com/kluzzebass/lazyhue/internal/ui/layout"
	"github.com/kluzzebass/lazyhue/internal/ui/panels"
)

// Panel IDs
const (
	PanelDetail = "detail"
	PanelTree   = "tree"
	PanelLog    = "log"
)

// NavigationEntry represents a point in navigation history.
type NavigationEntry struct {
	EntityType string    // Type of entity ("light", "device", "scene", etc.)
	EntityID   string    // ID of the entity
	BridgeID   string    // Bridge the entity belongs to
	Timestamp  time.Time // When this navigation occurred
}

// Model is the main application model.
type Model struct {
	// Service layer
	manager     *hue.Manager
	credentials *config.CredentialStore
	uiState     *config.UIStateStore

	// UI components
	tree           *panels.TreePanel
	detailViewport viewport.Model
	logViewport    viewport.Model
	keys           ui.KeyMap
	help           help.Model
	styles         ui.Styles
	zones          *zone.Manager
	layout         *layout.Tree

	// Component tree for event routing
	componentRoot component.Component

	// Panel management
	panelOrder []string // Panel IDs in focus-cycle order

	// State
	width            int
	height           int
	focusedPane      string
	previousPane     string // Track previous panel for Escape key
	status           string
	activities       []Activity
	quitting         bool
	eventChan        chan bridgeEventMsg
	requestChan      chan requestMsg
	errorChan        chan errorMsg
	eventCancelFuncs map[string]context.CancelFunc
	bridgeBlinkUntil map[string]time.Time // Track when bridge blink indicators should stop
	lightGrid        *gridlayout.Grid     // Grid layout for light controls
	selectedLightID  string               // ID of currently selected light (for form updates)
	lastTreeClick    time.Time            // Track last tree item click for double-click detection
	lastTreeClickID  string               // Track which tree item was last clicked

	// Duration settings for controls (stored per light)
	signalDurations      map[string]int // Signal duration in seconds per light ID
	timedEffectDurations map[string]int // Timed effect duration in minutes per light ID

	// Rename mode state
	renaming bool // Whether we're in rename mode
	// renameInput removed - now owned by renameModal (TextInputModal)
	renameEntityType   panels.EntityType // Type of entity being renamed
	renameEntityID     string            // ID of entity being renamed
	renameBridgeID     string            // Bridge ID the entity belongs to
	renameOriginalName string            // Original name for cancel

	// Delete confirmation state
	confirmingDelete       bool              // Whether we're showing delete confirmation
	deleteBridgeID         string            // ID of bridge to delete
	deleteBridgeName       string            // Name of bridge to delete (for display)
	confirmingDeleteEntity bool              // Whether we're showing entity delete confirmation
	deleteEntityID         string            // ID of entity to delete
	deleteEntityName       string            // Name of entity to delete (for display)
	deleteEntityType       panels.EntityType // Type of entity to delete
	deleteEntityBridgeID   string            // Bridge ID the entity belongs to

	// Pairing state
	pairing           bool               // Whether we're in pairing mode
	pairingFor        *hue.BridgeInfo    // Bridge being paired
	pairingCancel     context.CancelFunc // Function to cancel pairing
	pairingStartTime  time.Time          // When pairing started
	discoveredBridges []hue.BridgeInfo   // Discovered bridges available for pairing
	pairingRemaining  int                // Seconds remaining in pairing countdown
	pairingRequested  bool               // Whether user explicitly requested pairing via 'p' key

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
	showHelp     bool // Whether help is displayed in detail panel
	showActivity bool // Whether activity log panel is visible

	// State restoration
	stateRestored      bool   // Whether UI state has been restored from disk
	pendingSelectionID string // Entity ID to select once it appears in the tree

	// Navigation history
	navigationHistory []NavigationEntry
	historyIndex      int // Current position in history (-1 if no history or at latest)

	// Keybindings for help system
	panelBindings  map[string][]Binding
	globalBindings []Binding
}

// New creates a new application model.
func New(creds *config.CredentialStore) Model {
	styles := ui.DefaultStyles()
	keys := ui.DefaultKeyMap()
	zones := zone.New()

	// Load UI state (or create empty if doesn't exist)
	uiState, err := config.LoadUIState()
	if err != nil {
		// If we can't load state, just start with empty state
		uiState = config.NewUIStateStore()
	}

	// Define panel order for automatic key assignment
	// Tree (1), Detail (2), Log (3)
	panelOrder := []string{
		PanelTree,
		PanelDetail,
		PanelLog,
	}

	// Create tree panel (key will be assigned automatically)
	tree := panels.NewTreePanel(styles, zones, "Home", "")

	// Assign keys automatically based on position in panelOrder
	// Keys are 1-based: first panel gets "1", second gets "2", etc.
	for i, panelID := range panelOrder {
		key := fmt.Sprintf("%d", i+1)
		switch panelID {
		case PanelTree:
			tree.SetKey(key)
		}
	}

	// Create help model
	h := help.New()

	// Build layout tree
	// Left column: Tree (40% of width, capped to content width)
	// Right column: Detail + Log (remaining width)
	// Note: rebuildLayout() recalculates this dynamically as tree content changes
	maxTreeWidth := max(tree.MaxContentWidth(), 30) // 30 = minimum usable width
	treeSizeSpec := layout.FlexWithConstraints(0.4, 0, maxTreeWidth)
	layoutRoot := layout.HSplit(
		layout.Child{Size: treeSizeSpec, Node: layout.NewLeaf(PanelTree)},
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
		bridgeBlinkUntil:     make(map[string]time.Time),
		signalDurations:      make(map[string]int),
		timedEffectDurations: make(map[string]int),
		lightGrid:            gridlayout.NewGrid().SetGaps(2, 0).SetFocusIndicator(true, "> ", "  "),
		selectedLightID:      "",
		historyIndex:     -1, // No history initially
		inputStack:       NewInputStack(),
		showActivity:     true, // Activity log visible by default
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
