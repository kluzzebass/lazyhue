package panels

import (
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/kluzzebass/lazyhue/internal/ui"
)

// HomeTabbedPanel is the second column panel with Home (tree), Lights, Devices, and Scenes tabs.
type HomeTabbedPanel struct {
	ScrollState       // Embedded for width/height and helper methods
	styles   ui.Styles
	panelKey string

	// Tabs
	activeTab int // 0=Home, 1=Lights, 2=Devices, 3=Scenes
	tabTitles []string

	// Home tab uses the embedded tree panel
	treePanel *TreePanel

	// Lights, Devices and Scenes tabs use flat lists with their own scroll state
	lights   []list.Item
	devices  []list.Item
	scenes   []list.Item
	lightsScroll  ScrollState
	devicesScroll ScrollState
	scenesScroll  ScrollState
}

// NewHomeTabbedPanel creates a new home tabbed panel.
func NewHomeTabbedPanel(styles ui.Styles, panelKey string) *HomeTabbedPanel {
	return &HomeTabbedPanel{
		styles:    styles,
		panelKey:  panelKey,
		tabTitles: []string{"Home", "Lights", "Devices", "Scenes"},
		treePanel: NewTreePanel(styles, "Home", panelKey),
	}
}

// SetSize updates the panel dimensions.
func (p *HomeTabbedPanel) SetSize(width, height int) {
	p.ScrollState.SetSize(width, height)
	// Pass size to tree panel
	p.treePanel.SetSize(width, height)
	// Update scroll states for other tabs
	p.lightsScroll.SetSize(width, height)
	p.devicesScroll.SetSize(width, height)
	p.scenesScroll.SetSize(width, height)
}

// SetRoots sets the tree data (for Home tab).
func (p *HomeTabbedPanel) SetRoots(roots []*TreeNode) {
	p.treePanel.SetRoots(roots)
}

// SetLights sets the lights list.
func (p *HomeTabbedPanel) SetLights(items []list.Item) {
	p.lights = items
	p.lightsScroll.ClampCursor(len(p.lights))
}

// SetDevices sets the devices list.
func (p *HomeTabbedPanel) SetDevices(items []list.Item) {
	p.devices = items
	p.devicesScroll.ClampCursor(len(p.devices))
}

// SetScenes sets the scenes list.
func (p *HomeTabbedPanel) SetScenes(items []list.Item) {
	p.scenes = items
	p.scenesScroll.ClampCursor(len(p.scenes))
}

// ActiveTabIndex returns the active tab index.
func (p *HomeTabbedPanel) ActiveTabIndex() int {
	return p.activeTab
}

// SetActiveTab sets the active tab index.
func (p *HomeTabbedPanel) SetActiveTab(tab int) {
	if tab >= 0 && tab < len(p.tabTitles) {
		p.activeTab = tab
	}
}

// NextTab switches to the next tab.
func (p *HomeTabbedPanel) NextTab() {
	p.activeTab = (p.activeTab + 1) % len(p.tabTitles)
}

// PrevTab switches to the previous tab.
func (p *HomeTabbedPanel) PrevTab() {
	p.activeTab = (p.activeTab - 1 + len(p.tabTitles)) % len(p.tabTitles)
}

// Update handles input for the panel.
func (p *HomeTabbedPanel) Update(msg tea.Msg) tea.Cmd {
	viewHeight := p.ViewHeight()

	switch msg := msg.(type) {
	case tea.MouseMsg:
		switch msg.Button {
		case tea.MouseButtonWheelUp:
			p.moveCursor(-1)
			return nil
		case tea.MouseButtonWheelDown:
			p.moveCursor(1)
			return nil
		}

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, key.NewBinding(key.WithKeys("left", "h"))):
			// In Home tab, left collapses node; otherwise switch tabs
			if p.activeTab == 0 {
				return p.treePanel.Update(msg)
			}
			p.PrevTab()
			return nil
		case key.Matches(msg, key.NewBinding(key.WithKeys("right", "l"))):
			// In Home tab, right expands node; otherwise switch tabs
			if p.activeTab == 0 {
				return p.treePanel.Update(msg)
			}
			p.NextTab()
			return nil
		case key.Matches(msg, key.NewBinding(key.WithKeys("tab"))):
			p.NextTab()
			return nil
		case key.Matches(msg, key.NewBinding(key.WithKeys("shift+tab"))):
			p.PrevTab()
			return nil
		case key.Matches(msg, key.NewBinding(key.WithKeys("up", "k"))):
			p.moveCursor(-1)
		case key.Matches(msg, key.NewBinding(key.WithKeys("down", "j"))):
			p.moveCursor(1)
		case key.Matches(msg, key.NewBinding(key.WithKeys("pgup"))):
			p.moveCursor(-viewHeight)
		case key.Matches(msg, key.NewBinding(key.WithKeys("pgdown"))):
			p.moveCursor(viewHeight)
		case key.Matches(msg, key.NewBinding(key.WithKeys("home", "g"))):
			p.moveCursorToStart()
		case key.Matches(msg, key.NewBinding(key.WithKeys("end", "G"))):
			p.moveCursorToEnd()
		}
	}

	return nil
}

func (p *HomeTabbedPanel) moveCursor(delta int) {
	switch p.activeTab {
	case 0: // Home (tree)
		p.treePanel.MoveCursor(delta, p.treePanel.FlatListLen())
	case 1: // Lights
		p.lightsScroll.MoveCursor(delta, len(p.lights))
	case 2: // Devices
		p.devicesScroll.MoveCursor(delta, len(p.devices))
	case 3: // Scenes
		p.scenesScroll.MoveCursor(delta, len(p.scenes))
	}
}

func (p *HomeTabbedPanel) moveCursorToStart() {
	switch p.activeTab {
	case 0:
		p.treePanel.MoveToStart()
	case 1:
		p.lightsScroll.MoveToStart()
	case 2:
		p.devicesScroll.MoveToStart()
	case 3:
		p.scenesScroll.MoveToStart()
	}
}

func (p *HomeTabbedPanel) moveCursorToEnd() {
	switch p.activeTab {
	case 0:
		p.treePanel.MoveToEnd(p.treePanel.FlatListLen())
	case 1:
		p.lightsScroll.MoveToEnd(len(p.lights))
	case 2:
		p.devicesScroll.MoveToEnd(len(p.devices))
	case 3:
		p.scenesScroll.MoveToEnd(len(p.scenes))
	}
}

// SelectedEntity returns the currently selected entity.
func (p *HomeTabbedPanel) SelectedEntity() (*EntityItem, bool) {
	switch p.activeTab {
	case 0: // Home (tree)
		return p.treePanel.SelectedEntity()
	case 1: // Lights
		cursor := p.lightsScroll.Cursor()
		if cursor >= 0 && cursor < len(p.lights) {
			if ei, ok := p.lights[cursor].(EntityItem); ok {
				return &ei, true
			}
		}
	case 2: // Devices
		cursor := p.devicesScroll.Cursor()
		if cursor >= 0 && cursor < len(p.devices) {
			if ei, ok := p.devices[cursor].(EntityItem); ok {
				return &ei, true
			}
		}
	case 3: // Scenes
		cursor := p.scenesScroll.Cursor()
		if cursor >= 0 && cursor < len(p.scenes) {
			if ei, ok := p.scenes[cursor].(EntityItem); ok {
				return &ei, true
			}
		}
	}
	return nil, false
}

// SelectByID finds and selects an entity by its ID.
// Returns true if the entity was found and selected.
func (p *HomeTabbedPanel) SelectByID(id string) bool {
	if id == "" {
		return false
	}

	switch p.activeTab {
	case 0: // Home (tree)
		return p.treePanel.SelectByID(id)
	case 1: // Lights
		for i, item := range p.lights {
			if ei, ok := item.(EntityItem); ok && ei.ID == id {
				p.lightsScroll.SetCursor(i)
				p.lightsScroll.EnsureCursorVisible(len(p.lights))
				return true
			}
		}
	case 2: // Devices
		for i, item := range p.devices {
			if ei, ok := item.(EntityItem); ok && ei.ID == id {
				p.devicesScroll.SetCursor(i)
				p.devicesScroll.EnsureCursorVisible(len(p.devices))
				return true
			}
		}
	case 3: // Scenes
		for i, item := range p.scenes {
			if ei, ok := item.(EntityItem); ok && ei.ID == id {
				p.scenesScroll.SetCursor(i)
				p.scenesScroll.EnsureCursorVisible(len(p.scenes))
				return true
			}
		}
	}
	return false
}

// Key returns the keyboard shortcut key.
func (p *HomeTabbedPanel) Key() string {
	return p.panelKey
}

// Title returns the panel title.
func (p *HomeTabbedPanel) Title() string {
	return p.tabTitles[p.activeTab]
}

// View renders the panel.
func (p *HomeTabbedPanel) View(active bool) string {
	contentHeight := p.ViewHeight()
	contentWidth := p.ContentWidth()

	var lines []string
	var itemCount, itemIndex, scrollOffset int

	switch p.activeTab {
	case 0: // Home (tree)
		// Use tree panel's rendering but extract content
		itemCount = p.treePanel.FlatListLen()
		scrollOffset = p.treePanel.Offset()
		// Show last visible line, not cursor position
		lastVisible := scrollOffset + contentHeight
		if lastVisible > itemCount {
			lastVisible = itemCount
		}
		itemIndex = lastVisible - 1
		if itemIndex < 0 {
			itemIndex = 0
		}

		if itemCount == 0 {
			lines = append(lines, p.styles.Muted.Render("No items"))
		} else {
			endIdx := min(scrollOffset+contentHeight, itemCount)
			for i := scrollOffset; i < endIdx; i++ {
				node := p.treePanel.FlatListItem(i)
				selected := i == p.treePanel.Cursor()
				line := p.treePanel.renderNode(node, selected, active)
				lines = append(lines, line)
			}
		}

	case 1: // Lights
		itemCount = len(p.lights)
		scrollOffset = p.lightsScroll.Offset()
		// Show last visible line
		lastVisible := scrollOffset + contentHeight
		if lastVisible > itemCount {
			lastVisible = itemCount
		}
		itemIndex = lastVisible - 1
		if itemIndex < 0 {
			itemIndex = 0
		}

		if itemCount == 0 {
			lines = append(lines, p.styles.Muted.Render("No lights"))
		} else {
			endIdx := min(scrollOffset+contentHeight, itemCount)
			for i := scrollOffset; i < endIdx; i++ {
				selected := i == p.lightsScroll.Cursor()
				line := p.renderListItem(p.lights[i], selected, active, contentWidth)
				lines = append(lines, line)
			}
		}

	case 2: // Devices
		itemCount = len(p.devices)
		scrollOffset = p.devicesScroll.Offset()
		// Show last visible line
		lastVisible := scrollOffset + contentHeight
		if lastVisible > itemCount {
			lastVisible = itemCount
		}
		itemIndex = lastVisible - 1
		if itemIndex < 0 {
			itemIndex = 0
		}

		if itemCount == 0 {
			lines = append(lines, p.styles.Muted.Render("No devices"))
		} else {
			endIdx := min(scrollOffset+contentHeight, itemCount)
			for i := scrollOffset; i < endIdx; i++ {
				selected := i == p.devicesScroll.Cursor()
				line := p.renderListItem(p.devices[i], selected, active, contentWidth)
				lines = append(lines, line)
			}
		}

	case 3: // Scenes
		itemCount = len(p.scenes)
		scrollOffset = p.scenesScroll.Offset()
		// Show last visible line
		lastVisible := scrollOffset + contentHeight
		if lastVisible > itemCount {
			lastVisible = itemCount
		}
		itemIndex = lastVisible - 1
		if itemIndex < 0 {
			itemIndex = 0
		}

		if itemCount == 0 {
			lines = append(lines, p.styles.Muted.Render("No scenes"))
		} else {
			endIdx := min(scrollOffset+contentHeight, itemCount)
			for i := scrollOffset; i < endIdx; i++ {
				selected := i == p.scenesScroll.Cursor()
				line := p.renderListItem(p.scenes[i], selected, active, contentWidth)
				lines = append(lines, line)
			}
		}
	}

	// Pad to fill height
	for len(lines) < contentHeight {
		lines = append(lines, "")
	}

	content := lipgloss.JoinVertical(lipgloss.Left, lines...)

	cfg := ui.BorderConfig{
		PanelKey:    p.panelKey,
		Tabs:        p.tabTitles,
		ActiveTab:   p.activeTab,
		ItemIndex:   itemIndex,
		ItemCount:   itemCount,
		ScrollPos:   scrollOffset,
		TotalHeight: itemCount,
		ViewHeight:  contentHeight,
	}

	return ui.RenderBorderedPanel(content, p.Width(), p.Height(), active, p.styles, cfg)
}

// renderListItem renders a flat list item with type-appropriate indicators.
func (p *HomeTabbedPanel) renderListItem(item list.Item, selected, active bool, width int) string {
	ei, ok := item.(EntityItem)
	if !ok {
		return ""
	}
	return RenderEntityLine(ei, width, selected, active, p.styles)
}

// HasItems returns true if the active tab has any items.
func (p *HomeTabbedPanel) HasItems() bool {
	switch p.activeTab {
	case 0:
		return p.treePanel.FlatListLen() > 0
	case 1:
		return len(p.lights) > 0
	case 2:
		return len(p.devices) > 0
	case 3:
		return len(p.scenes) > 0
	}
	return false
}

// HandleClick handles a mouse click at relative coordinates.
func (p *HomeTabbedPanel) HandleClick(relX, relY int) {
	// Check if click is in the top border row (y=0) where tabs are
	if relY == 0 {
		startX := 5 // Skip "╭[2]─"
		for i, title := range p.tabTitles {
			tabWidth := lipgloss.Width(title)
			endX := startX + tabWidth
			if relX >= startX && relX < endX {
				p.activeTab = i
				return
			}
			startX = endX + 1
		}
		return
	}

	// Click on content area - select item
	if relY >= 1 {
		switch p.activeTab {
		case 0:
			p.treePanel.HandleClick(relX, relY)
		case 1:
			p.lightsScroll.HandleClick(relY, len(p.lights))
		case 2:
			p.devicesScroll.HandleClick(relY, len(p.devices))
		case 3:
			p.scenesScroll.HandleClick(relY, len(p.scenes))
		}
	}
}

// GetNodeStates returns the current expanded state of tree nodes.
func (p *HomeTabbedPanel) GetNodeStates() map[string]bool {
	return p.treePanel.GetNodeStates()
}

// SetNodeStates restores the expanded state of tree nodes.
func (p *HomeTabbedPanel) SetNodeStates(states map[string]bool) {
	p.treePanel.SetNodeStates(states)
}

// Clear clears all content.
func (p *HomeTabbedPanel) Clear() {
	p.treePanel.SetRoots(nil)
	p.lights = nil
	p.devices = nil
	p.scenes = nil
	p.lightsScroll = ScrollState{}
	p.devicesScroll = ScrollState{}
	p.scenesScroll = ScrollState{}
}

// IsGroupSelected returns true if a group node is selected (Home tab only).
func (p *HomeTabbedPanel) IsGroupSelected() bool {
	if p.activeTab != 0 {
		return false
	}
	return p.treePanel.IsGroupSelected()
}

// ToggleSelected toggles the selected node (Home tab only).
func (p *HomeTabbedPanel) ToggleSelected() {
	if p.activeTab == 0 {
		p.treePanel.ToggleSelected()
	}
}
