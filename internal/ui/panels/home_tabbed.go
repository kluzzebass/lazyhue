package panels

import (
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/kluzzebass/lazyhue/internal/ui"
)

// HomeTabbedPanel is the second column panel with Home (tree), Lights, Devices, and Scenes tabs.
type HomeTabbedPanel struct {
	styles   ui.Styles
	width    int
	height   int
	panelKey string

	// Tabs
	activeTab int // 0=Home, 1=Lights, 2=Devices, 3=Scenes
	tabTitles []string

	// Home tab uses the embedded tree panel
	treePanel *TreePanel

	// Lights, Devices and Scenes tabs use flat lists
	lights        []list.Item
	devices       []list.Item
	scenes        []list.Item
	lightsCursor  int
	lightsOffset  int
	devicesCursor int
	devicesOffset int
	scenesCursor  int
	scenesOffset  int
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
	p.width = width
	p.height = height
	// Pass size to tree panel (subtract 1 for tabs header)
	p.treePanel.SetSize(width, height)
}

// viewHeight returns the number of visible lines for content.
func (p *HomeTabbedPanel) viewHeight() int {
	return max(1, p.height-2)
}

// SetRoots sets the tree data (for Home tab).
func (p *HomeTabbedPanel) SetRoots(roots []*TreeNode) {
	p.treePanel.SetRoots(roots)
}

// SetLights sets the lights list.
func (p *HomeTabbedPanel) SetLights(items []list.Item) {
	p.lights = items
	p.clampLights()
}

// SetDevices sets the devices list.
func (p *HomeTabbedPanel) SetDevices(items []list.Item) {
	p.devices = items
	p.clampDevices()
}

// SetScenes sets the scenes list.
func (p *HomeTabbedPanel) SetScenes(items []list.Item) {
	p.scenes = items
	p.clampScenes()
}

func (p *HomeTabbedPanel) clampLights() {
	if len(p.lights) == 0 {
		p.lightsCursor = 0
		p.lightsOffset = 0
	} else if p.lightsCursor >= len(p.lights) {
		p.lightsCursor = len(p.lights) - 1
	}
}

func (p *HomeTabbedPanel) clampDevices() {
	if len(p.devices) == 0 {
		p.devicesCursor = 0
		p.devicesOffset = 0
	} else if p.devicesCursor >= len(p.devices) {
		p.devicesCursor = len(p.devices) - 1
	}
}

func (p *HomeTabbedPanel) clampScenes() {
	if len(p.scenes) == 0 {
		p.scenesCursor = 0
		p.scenesOffset = 0
	} else if p.scenesCursor >= len(p.scenes) {
		p.scenesCursor = len(p.scenes) - 1
	}
}

func (p *HomeTabbedPanel) ensureLightsVisible() {
	viewHeight := p.viewHeight()
	if p.lightsCursor < p.lightsOffset {
		p.lightsOffset = p.lightsCursor
	}
	if p.lightsCursor >= p.lightsOffset+viewHeight {
		p.lightsOffset = p.lightsCursor - viewHeight + 1
	}
	maxOffset := max(0, len(p.lights)-viewHeight)
	if p.lightsOffset > maxOffset {
		p.lightsOffset = maxOffset
	}
}

func (p *HomeTabbedPanel) ensureDevicesVisible() {
	viewHeight := p.viewHeight()
	if p.devicesCursor < p.devicesOffset {
		p.devicesOffset = p.devicesCursor
	}
	if p.devicesCursor >= p.devicesOffset+viewHeight {
		p.devicesOffset = p.devicesCursor - viewHeight + 1
	}
	maxOffset := max(0, len(p.devices)-viewHeight)
	if p.devicesOffset > maxOffset {
		p.devicesOffset = maxOffset
	}
}

func (p *HomeTabbedPanel) ensureScenesVisible() {
	viewHeight := p.viewHeight()
	if p.scenesCursor < p.scenesOffset {
		p.scenesOffset = p.scenesCursor
	}
	if p.scenesCursor >= p.scenesOffset+viewHeight {
		p.scenesOffset = p.scenesCursor - viewHeight + 1
	}
	maxOffset := max(0, len(p.scenes)-viewHeight)
	if p.scenesOffset > maxOffset {
		p.scenesOffset = maxOffset
	}
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
	viewHeight := p.viewHeight()

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
		p.treePanel.moveCursor(delta)
	case 1: // Lights
		if len(p.lights) == 0 {
			return
		}
		p.lightsCursor += delta
		if p.lightsCursor < 0 {
			p.lightsCursor = 0
		}
		if p.lightsCursor >= len(p.lights) {
			p.lightsCursor = len(p.lights) - 1
		}
		p.ensureLightsVisible()
	case 2: // Devices
		if len(p.devices) == 0 {
			return
		}
		p.devicesCursor += delta
		if p.devicesCursor < 0 {
			p.devicesCursor = 0
		}
		if p.devicesCursor >= len(p.devices) {
			p.devicesCursor = len(p.devices) - 1
		}
		p.ensureDevicesVisible()
	case 3: // Scenes
		if len(p.scenes) == 0 {
			return
		}
		p.scenesCursor += delta
		if p.scenesCursor < 0 {
			p.scenesCursor = 0
		}
		if p.scenesCursor >= len(p.scenes) {
			p.scenesCursor = len(p.scenes) - 1
		}
		p.ensureScenesVisible()
	}
}

func (p *HomeTabbedPanel) moveCursorToStart() {
	switch p.activeTab {
	case 0:
		p.treePanel.cursor = 0
		p.treePanel.offset = 0
	case 1:
		p.lightsCursor = 0
		p.lightsOffset = 0
	case 2:
		p.devicesCursor = 0
		p.devicesOffset = 0
	case 3:
		p.scenesCursor = 0
		p.scenesOffset = 0
	}
}

func (p *HomeTabbedPanel) moveCursorToEnd() {
	switch p.activeTab {
	case 0:
		if len(p.treePanel.flatList) > 0 {
			p.treePanel.cursor = len(p.treePanel.flatList) - 1
			p.treePanel.ensureCursorVisible()
		}
	case 1:
		if len(p.lights) > 0 {
			p.lightsCursor = len(p.lights) - 1
			p.ensureLightsVisible()
		}
	case 2:
		if len(p.devices) > 0 {
			p.devicesCursor = len(p.devices) - 1
			p.ensureDevicesVisible()
		}
	case 3:
		if len(p.scenes) > 0 {
			p.scenesCursor = len(p.scenes) - 1
			p.ensureScenesVisible()
		}
	}
}

// SelectedEntity returns the currently selected entity.
func (p *HomeTabbedPanel) SelectedEntity() (*EntityItem, bool) {
	switch p.activeTab {
	case 0: // Home (tree)
		return p.treePanel.SelectedEntity()
	case 1: // Lights
		if p.lightsCursor >= 0 && p.lightsCursor < len(p.lights) {
			if ei, ok := p.lights[p.lightsCursor].(EntityItem); ok {
				return &ei, true
			}
		}
	case 2: // Devices
		if p.devicesCursor >= 0 && p.devicesCursor < len(p.devices) {
			if ei, ok := p.devices[p.devicesCursor].(EntityItem); ok {
				return &ei, true
			}
		}
	case 3: // Scenes
		if p.scenesCursor >= 0 && p.scenesCursor < len(p.scenes) {
			if ei, ok := p.scenes[p.scenesCursor].(EntityItem); ok {
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
				p.lightsCursor = i
				p.ensureLightsVisible()
				return true
			}
		}
	case 2: // Devices
		for i, item := range p.devices {
			if ei, ok := item.(EntityItem); ok && ei.ID == id {
				p.devicesCursor = i
				p.ensureDevicesVisible()
				return true
			}
		}
	case 3: // Scenes
		for i, item := range p.scenes {
			if ei, ok := item.(EntityItem); ok && ei.ID == id {
				p.scenesCursor = i
				p.ensureScenesVisible()
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
	contentHeight := p.viewHeight()
	contentWidth := max(1, p.width-2)

	var lines []string
	var itemCount, itemIndex, scrollOffset int

	switch p.activeTab {
	case 0: // Home (tree)
		// Use tree panel's rendering but extract content
		itemCount = len(p.treePanel.flatList)
		itemIndex = p.treePanel.cursor
		scrollOffset = p.treePanel.offset

		if itemCount == 0 {
			lines = append(lines, p.styles.Muted.Render("No items"))
		} else {
			endIdx := min(scrollOffset+contentHeight, itemCount)
			for i := scrollOffset; i < endIdx; i++ {
				node := p.treePanel.flatList[i]
				selected := i == p.treePanel.cursor
				line := p.treePanel.renderNode(node, selected, active)
				lines = append(lines, line)
			}
		}

	case 1: // Lights
		itemCount = len(p.lights)
		itemIndex = p.lightsCursor
		scrollOffset = p.lightsOffset

		if itemCount == 0 {
			lines = append(lines, p.styles.Muted.Render("No lights"))
		} else {
			endIdx := min(scrollOffset+contentHeight, itemCount)
			for i := scrollOffset; i < endIdx; i++ {
				selected := i == p.lightsCursor
				line := p.renderListItem(p.lights[i], selected, active, contentWidth)
				lines = append(lines, line)
			}
		}

	case 2: // Devices
		itemCount = len(p.devices)
		itemIndex = p.devicesCursor
		scrollOffset = p.devicesOffset

		if itemCount == 0 {
			lines = append(lines, p.styles.Muted.Render("No devices"))
		} else {
			endIdx := min(scrollOffset+contentHeight, itemCount)
			for i := scrollOffset; i < endIdx; i++ {
				selected := i == p.devicesCursor
				line := p.renderListItem(p.devices[i], selected, active, contentWidth)
				lines = append(lines, line)
			}
		}

	case 3: // Scenes
		itemCount = len(p.scenes)
		itemIndex = p.scenesCursor
		scrollOffset = p.scenesOffset

		if itemCount == 0 {
			lines = append(lines, p.styles.Muted.Render("No scenes"))
		} else {
			endIdx := min(scrollOffset+contentHeight, itemCount)
			for i := scrollOffset; i < endIdx; i++ {
				selected := i == p.scenesCursor
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

	return ui.RenderBorderedPanel(content, p.width, p.height, active, p.styles, cfg)
}

// renderListItem renders a flat list item with type-appropriate indicators.
func (p *HomeTabbedPanel) renderListItem(item list.Item, selected, active bool, width int) string {
	ei, ok := item.(EntityItem)
	if !ok {
		return ""
	}

	var style lipgloss.Style
	if selected && active {
		style = p.styles.SelectedItem.Width(width)
	} else {
		style = p.styles.ListItem.Width(width)
	}

	// Use centralized indicator rendering
	indicatorStr := RenderEntityIndicator(ei, p.styles, selected && active)

	name := ei.Name
	plainLine := indicatorStr + " " + name

	// Calculate visible width for truncation
	visibleWidth := lipgloss.Width(plainLine)
	if visibleWidth > width {
		// Need to truncate the name
		availableForName := width - lipgloss.Width(indicatorStr) - 2 // space + ellipsis
		if availableForName > 0 && len(name) > availableForName {
			name = name[:availableForName] + "…"
		}
	}

	// Build the styled line
	styledRest := style.Render(" " + name)

	// Pad to full width
	lineWidth := lipgloss.Width(indicatorStr + styledRest)
	padding := ""
	if lineWidth < width {
		padding = style.Render(strings.Repeat(" ", width-lineWidth))
	}

	return indicatorStr + styledRest + padding
}

// HasItems returns true if the active tab has any items.
func (p *HomeTabbedPanel) HasItems() bool {
	switch p.activeTab {
	case 0:
		return len(p.treePanel.flatList) > 0
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
			itemIndex := p.lightsOffset + (relY - 1)
			if itemIndex >= 0 && itemIndex < len(p.lights) {
				p.lightsCursor = itemIndex
			}
		case 2:
			itemIndex := p.devicesOffset + (relY - 1)
			if itemIndex >= 0 && itemIndex < len(p.devices) {
				p.devicesCursor = itemIndex
			}
		case 3:
			itemIndex := p.scenesOffset + (relY - 1)
			if itemIndex >= 0 && itemIndex < len(p.scenes) {
				p.scenesCursor = itemIndex
			}
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
	p.lightsCursor = 0
	p.lightsOffset = 0
	p.devicesCursor = 0
	p.devicesOffset = 0
	p.scenesCursor = 0
	p.scenesOffset = 0
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
