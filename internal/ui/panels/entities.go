// Package panels provides UI panel components.
package panels

import (
	"fmt"
	"io"
	"math"
	"sort"

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
	EntityDevice
	EntityEntertainment
	EntityBridge
)

// EntityItem wraps an entity for the list component.
type EntityItem struct {
	ID             string
	Name           string
	Type           EntityType
	IsOn           bool
	RawPtr         any    // The underlying openhue type for access to full data
	IndicatorColor string // Hex color for the indicator (e.g., "#ff0000"), empty for default
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

	// Sort by the actual display name (from device, not light metadata)
	sort.Slice(items, func(i, j int) bool {
		return items[i].(EntityItem).Name < items[j].(EntityItem).Name
	})

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

	// Build group ID -> name and group ID -> scenes
	// Groups can be rooms or zones
	groupNames := make(map[string]string)
	groupScenes := make(map[string][]openhue.SceneGet)
	ungroupedScenes := []openhue.SceneGet{}
	groupIsZone := make(map[string]bool)

	for _, room := range state.AllRooms() {
		if room.Id != nil && room.Metadata != nil && room.Metadata.Name != nil {
			groupNames[*room.Id] = *room.Metadata.Name
			groupIsZone[*room.Id] = false
		}
	}

	for _, zone := range state.AllZones() {
		if zone.Id != nil && zone.Metadata != nil && zone.Metadata.Name != nil {
			groupNames[*zone.Id] = *zone.Metadata.Name
			groupIsZone[*zone.Id] = true
		}
	}

	// Group scenes by room/zone
	for _, scene := range scenes {
		if scene.Group != nil && scene.Group.Rid != nil {
			rid := *scene.Group.Rid
			groupScenes[rid] = append(groupScenes[rid], scene)
		} else {
			ungroupedScenes = append(ungroupedScenes, scene)
		}
	}

	// Build tree nodes
	var roots []*TreeNode

	// Sort group IDs by name for stable ordering
	type groupEntry struct {
		id     string
		name   string
		isZone bool
	}
	var groupList []groupEntry
	for gid, name := range groupNames {
		if len(groupScenes[gid]) > 0 {
			groupList = append(groupList, groupEntry{gid, name, groupIsZone[gid]})
		}
	}
	// Sort by name
	for i := 0; i < len(groupList)-1; i++ {
		for j := i + 1; j < len(groupList); j++ {
			if groupList[i].name > groupList[j].name {
				groupList[i], groupList[j] = groupList[j], groupList[i]
			}
		}
	}

	for _, entry := range groupList {
		label := entry.name
		if entry.isZone {
			label = entry.name + " (zone)"
		}
		groupNode := &TreeNode{
			Label:    label,
			Item:     nil, // Group header, not selectable as entity
			Expanded: true,
		}

		for _, scene := range groupScenes[entry.id] {
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
			groupNode.Children = append(groupNode.Children, sceneNode)
		}

		roots = append(roots, groupNode)
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

// BuildZoneItems converts zones to list items.
func BuildZoneItems(state *hue.BridgeState) []list.Item {
	zones := state.AllZones()
	items := make([]list.Item, 0, len(zones))

	for _, zone := range zones {
		id := ""
		if zone.Id != nil {
			id = *zone.Id
		}

		name := ""
		if zone.Metadata != nil && zone.Metadata.Name != nil {
			name = *zone.Metadata.Name
		}

		isOn := state.IsZoneOn(zone)

		items = append(items, EntityItem{
			ID:     id,
			Name:   name,
			Type:   EntityZone,
			IsOn:   isOn,
			RawPtr: zone,
		})
	}
	return items
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
// Works for both rooms and zones since they use the same type.
func GetRoomFromItem(item EntityItem) (openhue.RoomGet, bool) {
	if item.Type == EntityRoom || item.Type == EntityZone {
		room, ok := item.RawPtr.(openhue.RoomGet)
		return room, ok
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

// BridgeData holds combined bridge information for display.
type BridgeData struct {
	Bridge *hue.Bridge
}

// GetBridgeFromItem extracts the BridgeData from an EntityItem.
func GetBridgeFromItem(item EntityItem) (BridgeData, bool) {
	if item.Type != EntityBridge {
		return BridgeData{}, false
	}
	data, ok := item.RawPtr.(BridgeData)
	return data, ok
}

// BuildHierarchyTree creates a unified tree with rooms/zones as top-level,
// and lights, devices, scenes as sub-categories under each.
// Ungrouped items appear in a special section at the bottom.
func BuildHierarchyTree(state *hue.BridgeState) []*TreeNode {
	if state == nil {
		return nil
	}

	var roots []*TreeNode

	// Build lookup maps for grouping
	rooms := state.AllRooms()
	zones := state.AllZones()
	allLights := state.AllLights()
	allScenes := state.AllScenes()
	allDevices := state.AllDevices()

	// Track which lights and devices are grouped
	groupedLightIDs := make(map[string]bool)
	groupedDeviceIDs := make(map[string]bool)

	// Build device ID -> device map for quick lookup
	deviceMap := make(map[string]openhue.DeviceGet)
	for _, d := range allDevices {
		if d.Id != nil {
			deviceMap[*d.Id] = d
		}
	}

	// Build device -> lights mapping (lights are owned by devices)
	deviceLights := make(map[string][]openhue.LightGet)
	for _, light := range allLights {
		if light.Owner != nil && light.Owner.Rid != nil {
			deviceLights[*light.Owner.Rid] = append(deviceLights[*light.Owner.Rid], light)
		}
	}

	// Build group ID -> scenes mapping
	groupScenes := make(map[string][]openhue.SceneGet)
	for _, scene := range allScenes {
		if scene.Group != nil && scene.Group.Rid != nil {
			groupScenes[*scene.Group.Rid] = append(groupScenes[*scene.Group.Rid], scene)
		}
	}

	// Helper to create a room/zone node with its children
	buildGroupNode := func(group openhue.RoomGet, isZone bool) *TreeNode {
		groupID := ""
		groupName := ""
		if group.Id != nil {
			groupID = *group.Id
		}
		if group.Metadata != nil && group.Metadata.Name != nil {
			groupName = *group.Metadata.Name
		}

		label := groupName
		if isZone {
			label = groupName + " (zone)"
		}

		groupNode := &TreeNode{
			Label:    label,
			Item:     nil, // Group header
			Expanded: true,
		}

		// Collect lights in this group (through device children)
		var groupLights []openhue.LightGet
		var groupDevices []openhue.DeviceGet

		if group.Children != nil {
			for _, child := range *group.Children {
				if child.Rid == nil {
					continue
				}
				deviceID := *child.Rid

				// Get lights from this device
				if lights, ok := deviceLights[deviceID]; ok {
					groupLights = append(groupLights, lights...)
					for _, l := range lights {
						if l.Id != nil {
							groupedLightIDs[*l.Id] = true
						}
					}
				}

				// Check if this device is not a light-owner (e.g., sensor, switch)
				if device, ok := deviceMap[deviceID]; ok {
					// If device has no lights under it, it's a standalone device
					if _, hasLights := deviceLights[deviceID]; !hasLights {
						groupDevices = append(groupDevices, device)
						groupedDeviceIDs[deviceID] = true
					} else {
						// Mark the device as grouped even if it owns lights
						groupedDeviceIDs[deviceID] = true
					}
				}
			}
		}

		// Sort lights by name
		sort.Slice(groupLights, func(i, j int) bool {
			nameI, nameJ := "", ""
			if groupLights[i].Metadata != nil && groupLights[i].Metadata.Name != nil {
				nameI = *groupLights[i].Metadata.Name
			}
			if groupLights[j].Metadata != nil && groupLights[j].Metadata.Name != nil {
				nameJ = *groupLights[j].Metadata.Name
			}
			return nameI < nameJ
		})

		// Sort devices by name
		sort.Slice(groupDevices, func(i, j int) bool {
			nameI, nameJ := "", ""
			if groupDevices[i].Metadata != nil && groupDevices[i].Metadata.Name != nil {
				nameI = *groupDevices[i].Metadata.Name
			}
			if groupDevices[j].Metadata != nil && groupDevices[j].Metadata.Name != nil {
				nameJ = *groupDevices[j].Metadata.Name
			}
			return nameI < nameJ
		})

		// Lights category
		if len(groupLights) > 0 {
			lightsNode := &TreeNode{
				Label:    fmt.Sprintf("Lights (%d)", len(groupLights)),
				Item:     nil,
				Expanded: true,
			}
			for _, light := range groupLights {
				lightsNode.Children = append(lightsNode.Children, buildLightNode(light, state))
			}
			groupNode.Children = append(groupNode.Children, lightsNode)
		}

		// Devices category (non-light devices)
		if len(groupDevices) > 0 {
			devicesNode := &TreeNode{
				Label:    fmt.Sprintf("Devices (%d)", len(groupDevices)),
				Item:     nil,
				Expanded: false, // Start collapsed
			}
			for _, device := range groupDevices {
				devicesNode.Children = append(devicesNode.Children, buildDeviceNode(device, state))
			}
			groupNode.Children = append(groupNode.Children, devicesNode)
		}

		// Scenes category
		scenes := groupScenes[groupID]
		if len(scenes) > 0 {
			// Sort scenes by name
			sort.Slice(scenes, func(i, j int) bool {
				nameI, nameJ := "", ""
				if scenes[i].Metadata != nil && scenes[i].Metadata.Name != nil {
					nameI = *scenes[i].Metadata.Name
				}
				if scenes[j].Metadata != nil && scenes[j].Metadata.Name != nil {
					nameJ = *scenes[j].Metadata.Name
				}
				return nameI < nameJ
			})

			scenesNode := &TreeNode{
				Label:    fmt.Sprintf("Scenes (%d)", len(scenes)),
				Item:     nil,
				Expanded: false, // Start collapsed
			}
			for _, scene := range scenes {
				scenesNode.Children = append(scenesNode.Children, buildSceneNode(scene))
			}
			groupNode.Children = append(groupNode.Children, scenesNode)
		}

		return groupNode
	}

	// Add rooms
	for _, room := range rooms {
		roots = append(roots, buildGroupNode(room, false))
	}

	// Add zones
	for _, zone := range zones {
		roots = append(roots, buildGroupNode(zone, true))
	}

	// Build ungrouped section
	var ungroupedLights []openhue.LightGet
	var ungroupedDevices []openhue.DeviceGet

	// Filter for devices that own lights but are not in any room
	lightOwnerIDs := make(map[string]bool)
	for _, light := range allLights {
		if light.Owner != nil && light.Owner.Rid != nil {
			lightOwnerIDs[*light.Owner.Rid] = true
		}
	}

	for _, light := range allLights {
		if light.Id != nil && !groupedLightIDs[*light.Id] {
			ungroupedLights = append(ungroupedLights, light)
		}
	}

	for _, device := range allDevices {
		if device.Id == nil {
			continue
		}
		deviceID := *device.Id

		// Skip if already grouped
		if groupedDeviceIDs[deviceID] {
			continue
		}

		// Skip light-owning devices (they'll be covered by ungrouped lights)
		if lightOwnerIDs[deviceID] {
			continue
		}

		// Skip bridge devices
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

		ungroupedDevices = append(ungroupedDevices, device)
	}

	// Add ungrouped section if there are any ungrouped items
	if len(ungroupedLights) > 0 || len(ungroupedDevices) > 0 {
		ungroupedNode := &TreeNode{
			Label:    "─── Ungrouped ───",
			Item:     nil,
			Expanded: true,
		}

		if len(ungroupedLights) > 0 {
			// Sort by name
			sort.Slice(ungroupedLights, func(i, j int) bool {
				nameI, nameJ := "", ""
				if ungroupedLights[i].Metadata != nil && ungroupedLights[i].Metadata.Name != nil {
					nameI = *ungroupedLights[i].Metadata.Name
				}
				if ungroupedLights[j].Metadata != nil && ungroupedLights[j].Metadata.Name != nil {
					nameJ = *ungroupedLights[j].Metadata.Name
				}
				return nameI < nameJ
			})

			lightsNode := &TreeNode{
				Label:    fmt.Sprintf("Lights (%d)", len(ungroupedLights)),
				Item:     nil,
				Expanded: true,
			}
			for _, light := range ungroupedLights {
				lightsNode.Children = append(lightsNode.Children, buildLightNode(light, state))
			}
			ungroupedNode.Children = append(ungroupedNode.Children, lightsNode)
		}

		if len(ungroupedDevices) > 0 {
			// Sort by name
			sort.Slice(ungroupedDevices, func(i, j int) bool {
				nameI, nameJ := "", ""
				if ungroupedDevices[i].Metadata != nil && ungroupedDevices[i].Metadata.Name != nil {
					nameI = *ungroupedDevices[i].Metadata.Name
				}
				if ungroupedDevices[j].Metadata != nil && ungroupedDevices[j].Metadata.Name != nil {
					nameJ = *ungroupedDevices[j].Metadata.Name
				}
				return nameI < nameJ
			})

			devicesNode := &TreeNode{
				Label:    fmt.Sprintf("Devices (%d)", len(ungroupedDevices)),
				Item:     nil,
				Expanded: false,
			}
			for _, device := range ungroupedDevices {
				devicesNode.Children = append(devicesNode.Children, buildDeviceNode(device, state))
			}
			ungroupedNode.Children = append(ungroupedNode.Children, devicesNode)
		}

		roots = append(roots, ungroupedNode)
	}

	return roots
}

// brightnessIndicator returns a character representing the brightness level.
// Uses circle fill characters: ○ ◔ ◑ ◕ ●
func brightnessIndicator(brightness float64) string {
	switch {
	case brightness <= 0:
		return "○" // off/empty
	case brightness <= 25:
		return "◔" // quarter
	case brightness <= 50:
		return "◑" // half
	case brightness <= 75:
		return "◕" // three-quarters
	default:
		return "●" // full
	}
}

// xyToRGB converts CIE XY color coordinates to RGB.
// Based on the standard conversion formula for Hue lights.
func xyToRGB(x, y, brightness float64) (r, g, b uint8) {
	// Avoid division by zero
	if y == 0 {
		y = 0.00001
	}

	// Calculate XYZ
	Y := brightness / 100.0
	X := (Y / y) * x
	Z := (Y / y) * (1.0 - x - y)

	// Convert to RGB using Wide RGB D65 matrix
	rFloat := X*1.656492 - Y*0.354851 - Z*0.255038
	gFloat := -X*0.707196 + Y*1.655397 + Z*0.036152
	bFloat := X*0.051713 - Y*0.121364 + Z*1.011530

	// Apply reverse gamma correction
	applyGamma := func(v float64) float64 {
		if v <= 0.0031308 {
			return 12.92 * v
		}
		return 1.055*math.Pow(v, 1.0/2.4) - 0.055
	}

	rFloat = applyGamma(rFloat)
	gFloat = applyGamma(gFloat)
	bFloat = applyGamma(bFloat)

	// Clamp and convert to 0-255
	clamp := func(v float64) uint8 {
		if v < 0 {
			return 0
		}
		if v > 1 {
			return 255
		}
		return uint8(v * 255)
	}

	return clamp(rFloat), clamp(gFloat), clamp(bFloat)
}

// mirekToRGB converts color temperature in mirek to RGB.
// Mirek range is typically 153 (cool/6500K) to 500 (warm/2000K).
func mirekToRGB(mirek int) (r, g, b uint8) {
	// Convert mirek to Kelvin: K = 1,000,000 / mirek
	kelvin := 1000000.0 / float64(mirek)

	// Approximate RGB from color temperature
	var rFloat, gFloat, bFloat float64

	// Red
	if kelvin <= 6600 {
		rFloat = 255
	} else {
		rFloat = 329.698727446 * math.Pow(kelvin/100-60, -0.1332047592)
	}

	// Green
	if kelvin <= 6600 {
		gFloat = 99.4708025861*math.Log(kelvin/100) - 161.1195681661
	} else {
		gFloat = 288.1221695283 * math.Pow(kelvin/100-60, -0.0755148492)
	}

	// Blue
	if kelvin >= 6600 {
		bFloat = 255
	} else if kelvin <= 1900 {
		bFloat = 0
	} else {
		bFloat = 138.5177312231*math.Log(kelvin/100-10) - 305.0447927307
	}

	// Clamp to 0-255
	clamp := func(v float64) uint8 {
		if v < 0 {
			return 0
		}
		if v > 255 {
			return 255
		}
		return uint8(v)
	}

	return clamp(rFloat), clamp(gFloat), clamp(bFloat)
}

// getLightColor extracts the RGB color from a light.
func getLightColor(light openhue.LightGet) lipgloss.Color {
	brightness := 100.0
	if light.Dimming != nil && light.Dimming.Brightness != nil {
		brightness = float64(*light.Dimming.Brightness)
	}

	// Try XY color first (color lights)
	if light.Color != nil && light.Color.Xy != nil {
		if light.Color.Xy.X != nil && light.Color.Xy.Y != nil {
			x := float64(*light.Color.Xy.X)
			y := float64(*light.Color.Xy.Y)
			r, g, b := xyToRGB(x, y, brightness)
			return lipgloss.Color(fmt.Sprintf("#%02x%02x%02x", r, g, b))
		}
	}

	// Try color temperature (white ambiance lights)
	if light.ColorTemperature != nil && light.ColorTemperature.Mirek != nil {
		if light.ColorTemperature.MirekValid == nil || *light.ColorTemperature.MirekValid {
			r, g, b := mirekToRGB(*light.ColorTemperature.Mirek)
			return lipgloss.Color(fmt.Sprintf("#%02x%02x%02x", r, g, b))
		}
	}

	// Default to warm white for non-color lights
	return lipgloss.Color("#ffcc66")
}

// buildLightNode creates a tree node for a light.
func buildLightNode(light openhue.LightGet, state *hue.BridgeState) *TreeNode {
	// Get the user-assigned name from the owning device
	name := ""
	if light.Owner != nil && light.Owner.Rid != nil {
		if device, ok := state.GetDevice(*light.Owner.Rid); ok {
			if device.Metadata != nil && device.Metadata.Name != nil {
				name = *device.Metadata.Name
			}
		}
	}
	// Fallback to light's own metadata name if device lookup fails
	if name == "" && light.Metadata != nil && light.Metadata.Name != nil {
		name = *light.Metadata.Name
	}

	id := ""
	if light.Id != nil {
		id = *light.Id
	}

	// Build label with brightness indicator (plain text, color stored separately)
	var label string
	var brightness float64
	var indicatorColor string
	if light.IsOn() {
		if light.Dimming != nil && light.Dimming.Brightness != nil {
			brightness = float64(*light.Dimming.Brightness)
		} else {
			brightness = 100 // Assume full if no dimming info
		}
		indicator := brightnessIndicator(brightness)
		// Store color separately for rendering
		indicatorColor = string(getLightColor(light))
		label = indicator + " " + name
	} else {
		label = "○ " + name
	}

	return &TreeNode{
		Label: label,
		Item: &EntityItem{
			ID:             id,
			Name:           name,
			Type:           EntityLight,
			IsOn:           light.IsOn(),
			RawPtr:         light,
			IndicatorColor: indicatorColor,
		},
	}
}

// buildDeviceNode creates a tree node for a device.
func buildDeviceNode(device openhue.DeviceGet, state *hue.BridgeState) *TreeNode {
	name := ""
	if device.Metadata != nil && device.Metadata.Name != nil {
		name = *device.Metadata.Name
	}

	id := ""
	if device.Id != nil {
		id = *device.Id
	}

	// Check if device has motion sensor and its state
	hasMotion, isDetecting := state.GetDeviceMotionState(device)

	var label string
	isOn := false
	if hasMotion {
		if isDetecting {
			label = "● " + name // Motion detected
			isOn = true
		} else {
			label = "○ " + name // No motion
		}
	} else {
		label = "◦ " + name // Non-motion device (neutral indicator)
	}

	return &TreeNode{
		Label: label,
		Item: &EntityItem{
			ID:     id,
			Name:   name,
			Type:   EntityDevice,
			IsOn:   isOn,
			RawPtr: device,
		},
	}
}

// buildSceneNode creates a tree node for a scene.
func buildSceneNode(scene openhue.SceneGet) *TreeNode {
	name := ""
	if scene.Metadata != nil && scene.Metadata.Name != nil {
		name = *scene.Metadata.Name
	}

	id := ""
	if scene.Id != nil {
		id = *scene.Id
	}

	// Check if scene is active
	isActive := false
	if scene.Status != nil && scene.Status.Active != nil {
		// Scene is active if status is "static" or "dynamic_palette" (not "inactive")
		isActive = *scene.Status.Active != openhue.SceneGetStatusActiveInactive
	}

	indicator := "○"
	if isActive {
		indicator = "●"
	}
	label := indicator + " " + name

	return &TreeNode{
		Label: label,
		Item: &EntityItem{
			ID:     id,
			Name:   name,
			Type:   EntityScene,
			IsOn:   isActive,
			RawPtr: scene,
		},
	}
}
