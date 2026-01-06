// Package panels provides UI panel components.
package panels

import (
	"fmt"
	"io"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/kluzzebass/lazyhue/internal/hue"
	"github.com/kluzzebass/lazyhue/internal/ui"
	"github.com/openhue/openhue-go"
)

// EntityType identifies what kind of entity an item represents.
type EntityType int

const (
	EntityRoom EntityType = iota
	EntityZone
	EntityLight
	EntityScene
)

// EntityItem wraps an entity for the list component.
type EntityItem struct {
	ID     string
	Name   string
	Type   EntityType
	IsOn   bool
	RawPtr any // The underlying openhue type for access to full data
}

func (i EntityItem) FilterValue() string { return i.Name }

// EntityDelegate renders entity list items.
type EntityDelegate struct {
	Styles ui.Styles
}

func (d EntityDelegate) Height() int                             { return 1 }
func (d EntityDelegate) Spacing() int                            { return 0 }
func (d EntityDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }

func (d EntityDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	i, ok := item.(EntityItem)
	if !ok {
		return
	}

	cursor := "  "
	if index == m.Index() {
		cursor = "> "
	}

	indicator := d.Styles.OnOffIndicator(i.IsOn)
	name := i.Name

	if index == m.Index() {
		name = d.Styles.SelectedItem.Render(name)
	}

	fmt.Fprintf(w, "%s%s %s", cursor, indicator, name)
}

// EntityPanel is the left panel showing rooms/zones/lights.
type EntityPanel struct {
	list   list.Model
	styles ui.Styles
	title  string
	width  int
	height int
}

// NewEntityPanel creates a new entity list panel.
func NewEntityPanel(styles ui.Styles) *EntityPanel {
	delegate := EntityDelegate{Styles: styles}
	l := list.New([]list.Item{}, delegate, 0, 0)
	l.SetShowTitle(false)
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(true)
	l.SetShowHelp(false)

	return &EntityPanel{
		list:   l,
		styles: styles,
		title:  "Entities",
	}
}

// SetItems updates the list items.
func (p *EntityPanel) SetItems(items []list.Item) {
	p.list.SetItems(items)
}

// SetSize updates the panel dimensions.
func (p *EntityPanel) SetSize(width, height int) {
	p.width = width
	p.height = height
	p.list.SetSize(width-4, height-4)
}

// SetTitle sets the panel title.
func (p *EntityPanel) SetTitle(title string) {
	p.title = title
}

// SelectedItem returns the currently selected item.
func (p *EntityPanel) SelectedItem() (EntityItem, bool) {
	item := p.list.SelectedItem()
	if item == nil {
		return EntityItem{}, false
	}
	ei, ok := item.(EntityItem)
	return ei, ok
}

// Update handles input for the entity list.
func (p *EntityPanel) Update(msg tea.Msg) (*EntityPanel, tea.Cmd) {
	var cmd tea.Cmd
	p.list, cmd = p.list.Update(msg)
	return p, cmd
}

// View renders the entity panel.
func (p *EntityPanel) View(active bool) string {
	titleStyle := p.styles.PanelTitle
	title := titleStyle.Render(p.title)

	var panelStyle lipgloss.Style
	if active {
		panelStyle = p.styles.ActivePanel
	} else {
		panelStyle = p.styles.LeftPanel
	}

	content := lipgloss.JoinVertical(lipgloss.Left,
		title,
		p.list.View(),
	)

	return panelStyle.Width(p.width).Height(p.height).Render(content)
}

// BuildRoomItems converts rooms to list items.
func BuildRoomItems(state *hue.BridgeState) []list.Item {
	rooms := state.AllRooms()
	items := make([]list.Item, 0, len(rooms))
	for _, room := range rooms {
		name := ""
		if room.Metadata != nil && room.Metadata.Name != nil {
			name = *room.Metadata.Name
		}
		id := ""
		if room.Id != nil {
			id = *room.Id
		}
		items = append(items, EntityItem{
			ID:     id,
			Name:   name,
			Type:   EntityRoom,
			IsOn:   state.IsRoomOn(room),
			RawPtr: room,
		})
	}
	return items
}

// BuildLightItems converts lights to list items.
func BuildLightItems(state *hue.BridgeState) []list.Item {
	lights := state.AllLights()
	items := make([]list.Item, 0, len(lights))
	for _, light := range lights {
		name := ""
		if light.Metadata != nil && light.Metadata.Name != nil {
			name = *light.Metadata.Name
		}
		id := ""
		if light.Id != nil {
			id = *light.Id
		}
		items = append(items, EntityItem{
			ID:     id,
			Name:   name,
			Type:   EntityLight,
			IsOn:   light.IsOn(),
			RawPtr: light,
		})
	}
	return items
}

// BuildSceneItems converts scenes to list items for a specific group.
func BuildSceneItems(state *hue.BridgeState, groupID string) []list.Item {
	scenes := state.RoomScenes(groupID)
	items := make([]list.Item, 0, len(scenes))
	for _, scene := range scenes {
		name := ""
		if scene.Metadata != nil && scene.Metadata.Name != nil {
			name = *scene.Metadata.Name
		}
		id := ""
		if scene.Id != nil {
			id = *scene.Id
		}
		items = append(items, EntityItem{
			ID:     id,
			Name:   name,
			Type:   EntityScene,
			IsOn:   false,
			RawPtr: scene,
		})
	}
	return items
}

// GetRoomFromItem extracts the openhue.RoomGet from an EntityItem.
func GetRoomFromItem(item EntityItem) (openhue.RoomGet, bool) {
	if item.Type != EntityRoom {
		return openhue.RoomGet{}, false
	}
	room, ok := item.RawPtr.(openhue.RoomGet)
	return room, ok
}

// GetLightFromItem extracts the openhue.LightGet from an EntityItem.
func GetLightFromItem(item EntityItem) (openhue.LightGet, bool) {
	if item.Type != EntityLight {
		return openhue.LightGet{}, false
	}
	light, ok := item.RawPtr.(openhue.LightGet)
	return light, ok
}
