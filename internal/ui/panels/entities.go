// Package panels provides UI panel components.
package panels

import (
	"fmt"
	"io"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
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
	EntityDevice
	EntityEntertainment
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
	l.SetShowPagination(false)
	l.InfiniteScrolling = false

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

	// Content area: width minus borders (2), height minus borders (2)
	contentWidth := max(1, width-2)
	contentHeight := max(1, height-2)

	p.list.SetSize(contentWidth, contentHeight)
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
	content := p.list.View()

	cfg := ui.BorderConfig{
		Title:       p.title,
		ItemIndex:   p.list.Index(),
		ItemCount:   len(p.list.Items()),
		ScrollPos:   p.list.Index(),
		TotalHeight: len(p.list.Items()),
		ViewHeight:  p.height - 2,
	}

	return ui.RenderBorderedPanel(content, p.width, p.height, active, p.styles, cfg)
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
// Note: Light names come from the owning device, not the light's deprecated Metadata.Name
func BuildLightItems(state *hue.BridgeState) []list.Item {
	lights := state.AllLights()
	items := make([]list.Item, 0, len(lights))

	// Build device ID -> name lookup for proper light names
	deviceNames := make(map[string]string)
	for _, device := range state.AllDevices() {
		if device.Id != nil && device.Metadata != nil && device.Metadata.Name != nil {
			deviceNames[*device.Id] = *device.Metadata.Name
		}
	}

	for _, light := range lights {
		id := ""
		if light.Id != nil {
			id = *light.Id
		}

		// Get name from owning device (preferred) or fall back to light metadata
		name := ""
		if light.Owner != nil && light.Owner.Rid != nil {
			if deviceName, ok := deviceNames[*light.Owner.Rid]; ok {
				name = deviceName
			}
		}
		if name == "" && light.Metadata != nil && light.Metadata.Name != nil {
			name = *light.Metadata.Name
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

// BuildSceneItems converts all scenes to list items, including room/zone context.
func BuildSceneItems(state *hue.BridgeState) []list.Item {
	scenes := state.AllScenes()
	items := make([]list.Item, 0, len(scenes))

	// Build room ID -> name lookup
	roomNames := make(map[string]string)
	for _, room := range state.AllRooms() {
		if room.Id != nil && room.Metadata != nil && room.Metadata.Name != nil {
			roomNames[*room.Id] = *room.Metadata.Name
		}
	}

	for _, scene := range scenes {
		name := ""
		if scene.Metadata != nil && scene.Metadata.Name != nil {
			name = *scene.Metadata.Name
		}

		// Add room/zone context
		if scene.Group != nil && scene.Group.Rid != nil {
			if roomName, ok := roomNames[*scene.Group.Rid]; ok {
				name = name + " (" + roomName + ")"
			}
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

// BuildSceneTree creates a tree structure of scenes grouped by room/zone.
func BuildSceneTree(state *hue.BridgeState) []*TreeNode {
	scenes := state.AllScenes()

	// Build room ID -> name and room ID -> scenes
	roomNames := make(map[string]string)
	roomScenes := make(map[string][]openhue.SceneGet)
	ungroupedScenes := []openhue.SceneGet{}

	for _, room := range state.AllRooms() {
		if room.Id != nil && room.Metadata != nil && room.Metadata.Name != nil {
			roomNames[*room.Id] = *room.Metadata.Name
		}
	}

	// Group scenes by room
	for _, scene := range scenes {
		if scene.Group != nil && scene.Group.Rid != nil {
			rid := *scene.Group.Rid
			roomScenes[rid] = append(roomScenes[rid], scene)
		} else {
			ungroupedScenes = append(ungroupedScenes, scene)
		}
	}

	// Build tree nodes
	var roots []*TreeNode

	// Sort room IDs by name for stable ordering
	type roomEntry struct {
		id   string
		name string
	}
	var roomList []roomEntry
	for rid, name := range roomNames {
		if len(roomScenes[rid]) > 0 {
			roomList = append(roomList, roomEntry{rid, name})
		}
	}
	// Sort by name
	for i := 0; i < len(roomList)-1; i++ {
		for j := i + 1; j < len(roomList); j++ {
			if roomList[i].name > roomList[j].name {
				roomList[i], roomList[j] = roomList[j], roomList[i]
			}
		}
	}

	for _, entry := range roomList {
		roomNode := &TreeNode{
			Label:    entry.name,
			Item:     nil, // Group header, not selectable as entity
			Expanded: true,
		}

		for _, scene := range roomScenes[entry.id] {
			name := ""
			if scene.Metadata != nil && scene.Metadata.Name != nil {
				name = *scene.Metadata.Name
			}
			id := ""
			if scene.Id != nil {
				id = *scene.Id
			}
			sceneNode := &TreeNode{
				Label: name,
				Item: &EntityItem{
					ID:     id,
					Name:   name,
					Type:   EntityScene,
					IsOn:   false,
					RawPtr: scene,
				},
			}
			roomNode.Children = append(roomNode.Children, sceneNode)
		}

		roots = append(roots, roomNode)
	}

	// Add ungrouped scenes at root level
	for _, scene := range ungroupedScenes {
		name := ""
		if scene.Metadata != nil && scene.Metadata.Name != nil {
			name = *scene.Metadata.Name
		}
		id := ""
		if scene.Id != nil {
			id = *scene.Id
		}
		roots = append(roots, &TreeNode{
			Label: name,
			Item: &EntityItem{
				ID:     id,
				Name:   name,
				Type:   EntityScene,
				IsOn:   false,
				RawPtr: scene,
			},
		})
	}

	return roots
}

// BuildZoneItems returns an empty list (zones not yet supported by openhue-go Home).
func BuildZoneItems(state *hue.BridgeState) []list.Item {
	// Zones are not yet exposed via openhue-go Home interface
	return nil
}

// BuildDeviceItems converts devices to list items, filtering out lights and bridges.
func BuildDeviceItems(state *hue.BridgeState) []list.Item {
	devices := state.AllDevices()
	items := make([]list.Item, 0, len(devices))

	// Build a set of device IDs that own lights (to exclude them)
	lightOwnerIDs := make(map[string]bool)
	for _, light := range state.AllLights() {
		if light.Owner != nil && light.Owner.Rid != nil {
			lightOwnerIDs[*light.Owner.Rid] = true
		}
	}

	for _, device := range devices {
		id := ""
		if device.Id != nil {
			id = *device.Id
		}

		// Skip devices that own lights (these are light fixtures)
		if lightOwnerIDs[id] {
			continue
		}

		// Skip bridge devices (check if device provides a "bridge" service)
		isBridge := false
		if device.Services != nil {
			for _, svc := range *device.Services {
				if svc.Rtype != nil && *svc.Rtype == openhue.ResourceIdentifierRtypeBridge {
					isBridge = true
					break
				}
			}
		}
		if isBridge {
			continue
		}

		name := ""
		if device.Metadata != nil && device.Metadata.Name != nil {
			name = *device.Metadata.Name
		}

		items = append(items, EntityItem{
			ID:     id,
			Name:   name,
			Type:   EntityDevice,
			IsOn:   false, // Devices don't have a direct on/off
			RawPtr: device,
		})
	}
	return items
}

// BuildEntertainmentItems returns an empty list (entertainment areas not yet supported by openhue-go Home).
func BuildEntertainmentItems(state *hue.BridgeState) []list.Item {
	// Entertainment areas are not yet exposed via openhue-go Home interface
	return nil
}

// GetRoomFromItem extracts the openhue.RoomGet from an EntityItem.
func GetRoomFromItem(item EntityItem) (openhue.RoomGet, bool) {
	if item.Type == EntityRoom {
		room, ok := item.RawPtr.(openhue.RoomGet)
		return room, ok
	}
	// Also support zones since they have similar structure
	if item.Type == EntityZone {
		// Zones can be treated similarly for grouped light purposes
		return openhue.RoomGet{}, false
	}
	return openhue.RoomGet{}, false
}

// GetLightFromItem extracts the openhue.LightGet from an EntityItem.
func GetLightFromItem(item EntityItem) (openhue.LightGet, bool) {
	if item.Type != EntityLight {
		return openhue.LightGet{}, false
	}
	light, ok := item.RawPtr.(openhue.LightGet)
	return light, ok
}


// GetDeviceFromItem extracts the openhue.DeviceGet from an EntityItem.
func GetDeviceFromItem(item EntityItem) (openhue.DeviceGet, bool) {
	if item.Type != EntityDevice {
		return openhue.DeviceGet{}, false
	}
	device, ok := item.RawPtr.(openhue.DeviceGet)
	return device, ok
}

// GetSceneFromItem extracts the openhue.SceneGet from an EntityItem.
func GetSceneFromItem(item EntityItem) (openhue.SceneGet, bool) {
	if item.Type != EntityScene {
		return openhue.SceneGet{}, false
	}
	scene, ok := item.RawPtr.(openhue.SceneGet)
	return scene, ok
}
