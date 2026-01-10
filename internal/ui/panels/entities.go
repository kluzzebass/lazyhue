// Package panels provides UI panel components.
package panels

import (
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/kluzzebass/lazyhue/internal/hue"
	"github.com/kluzzebass/lazyhue/internal/hueclient"
	"github.com/kluzzebass/lazyhue/internal/ui"
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
	EntityLightsCategory   // Aggregate for "Lights" folder
	EntityDevicesCategory  // Aggregate for "Devices" folder
	EntityScenesCategory   // Aggregate for "Scenes" folder
)

// LightsCategoryData holds aggregate data for a lights category folder.
type LightsCategoryData struct {
	ParentName string
	Lights     []hueclient.LightGet
}

// DevicesCategoryData holds aggregate data for a devices category folder.
type DevicesCategoryData struct {
	ParentName string
	Devices    []hueclient.DeviceGet
}

// ScenesCategoryData holds aggregate data for a scenes category folder.
type ScenesCategoryData struct {
	ParentName string
	Scenes     []hueclient.SceneGet
}

// IsLightOn checks if a light is on (nil-safe).
func IsLightOn(light hueclient.LightGet) bool {
	return light.On != nil && light.On.On != nil && *light.On.On
}

// IsGroupedLightOn checks if a grouped light is on (nil-safe).
func IsGroupedLightOn(gl hueclient.GroupedLightGet) bool {
	return gl.On != nil && gl.On.On != nil && *gl.On.On
}

// EntityItem wraps an entity for the list component.
type EntityItem struct {
	ID             string
	Name           string
	Type           EntityType
	IsOn           bool
	RawPtr         any     // The underlying hueclient type for access to full data
	IndicatorColor string  // Hex color for the indicator (e.g., "#ff0000"), empty for default
	Brightness     float64 // Brightness level 0-100 for brightness indicator (lights, scenes)
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

		// Get brightness and color for indicator
		var brightness float64
		var indicatorColor string
		if IsLightOn(light) {
			brightness = 100.0
			if light.Dimming != nil && light.Dimming.Brightness != nil {
				brightness = float64(*light.Dimming.Brightness)
			}
			indicatorColor = string(GetLightColor(light))
		}

		items = append(items, EntityItem{
			ID:             id,
			Name:           name,
			Type:           EntityLight,
			IsOn:           IsLightOn(light),
			RawPtr:         light,
			IndicatorColor: indicatorColor,
			Brightness:     brightness,
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

		// Check if scene is active
		isActive := false
		if scene.Status != nil && scene.Status.Active != nil {
			isActive = *scene.Status.Active != hueclient.SceneGetStatusActiveInactive
		}

		// Calculate indicator color and brightness for active scenes (same logic as buildSceneNode)
		var indicatorColor string
		var brightness float64
		if isActive && scene.Group != nil && scene.Group.Rid != nil {
			var lights []hueclient.LightGet
			if room, ok := state.GetRoom(*scene.Group.Rid); ok {
				lights = state.RoomLights(room)
			} else if zone, ok := state.GetZone(*scene.Group.Rid); ok {
				lights = state.RoomLights(zone)
			}

			var totalBrightness, totalR, totalG, totalB float64
			var onCount int
			for _, light := range lights {
				if IsLightOn(light) {
					onCount++
					if light.Dimming != nil && light.Dimming.Brightness != nil {
						totalBrightness += float64(*light.Dimming.Brightness)
					} else {
						totalBrightness += 100
					}
					color := GetLightColor(light)
					colorStr := string(color)
					if len(colorStr) == 7 && colorStr[0] == '#' {
						var r, g, b int
						fmt.Sscanf(colorStr, "#%02x%02x%02x", &r, &g, &b)
						totalR += float64(r)
						totalG += float64(g)
						totalB += float64(b)
					}
				}
			}
			if onCount > 0 {
				brightness = totalBrightness / float64(onCount)
				avgR := int(totalR / float64(onCount))
				avgG := int(totalG / float64(onCount))
				avgB := int(totalB / float64(onCount))
				indicatorColor = fmt.Sprintf("#%02x%02x%02x", avgR, avgG, avgB)
			}
		} else if isActive {
			brightness = 100
		}

		items = append(items, EntityItem{
			ID:             id,
			Name:           name,
			Type:           EntityScene,
			IsOn:           isActive,
			RawPtr:         scene,
			IndicatorColor: indicatorColor,
			Brightness:     brightness,
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
	groupScenes := make(map[string][]hueclient.SceneGet)
	ungroupedScenes := []hueclient.SceneGet{}
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
			Expanded: false, // Start collapsed
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
				if svc.Rtype != nil && *svc.Rtype == hueclient.ResourceIdentifierRtypeBridge {
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

		// Check if device has motion sensor and its state
		hasMotion, isDetecting := state.GetDeviceMotionState(device)
		isOn := hasMotion && isDetecting

		items = append(items, EntityItem{
			ID:     id,
			Name:   name,
			Type:   EntityDevice,
			IsOn:   isOn,
			RawPtr: device,
		})
	}
	return items
}

// BuildEntertainmentItems converts entertainment configurations to list items.
func BuildEntertainmentItems(state *hue.BridgeState) []list.Item {
	configs := state.AllEntertainmentConfigurations()
	items := make([]list.Item, 0, len(configs))

	for _, cfg := range configs {
		name := cfg.ID
		if cfg.Metadata != nil && cfg.Metadata.Name != "" {
			name = cfg.Metadata.Name
		}

		items = append(items, EntityItem{
			ID:     cfg.ID,
			Name:   name,
			Type:   EntityEntertainment,
			IsOn:   cfg.Status == "streaming",
			RawPtr: cfg,
		})
	}
	return items
}

// GetRoomFromItem extracts the hueclient.RoomGet from an EntityItem.
// Works for both rooms and zones since they use the same type.
func GetRoomFromItem(item EntityItem) (hueclient.RoomGet, bool) {
	if item.Type == EntityRoom || item.Type == EntityZone {
		room, ok := item.RawPtr.(hueclient.RoomGet)
		return room, ok
	}
	return hueclient.RoomGet{}, false
}

// GetLightFromItem extracts the hueclient.LightGet from an EntityItem.
func GetLightFromItem(item EntityItem) (hueclient.LightGet, bool) {
	if item.Type != EntityLight {
		return hueclient.LightGet{}, false
	}
	light, ok := item.RawPtr.(hueclient.LightGet)
	return light, ok
}


// GetDeviceFromItem extracts the hueclient.DeviceGet from an EntityItem.
func GetDeviceFromItem(item EntityItem) (hueclient.DeviceGet, bool) {
	if item.Type != EntityDevice {
		return hueclient.DeviceGet{}, false
	}
	device, ok := item.RawPtr.(hueclient.DeviceGet)
	return device, ok
}

// GetSceneFromItem extracts the hueclient.SceneGet from an EntityItem.
func GetSceneFromItem(item EntityItem) (hueclient.SceneGet, bool) {
	if item.Type != EntityScene {
		return hueclient.SceneGet{}, false
	}
	scene, ok := item.RawPtr.(hueclient.SceneGet)
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

// GetEntertainmentFromItem extracts the EntertainmentConfiguration from an EntityItem.
func GetEntertainmentFromItem(item EntityItem) (hue.EntertainmentConfiguration, bool) {
	if item.Type != EntityEntertainment {
		return hue.EntertainmentConfiguration{}, false
	}
	cfg, ok := item.RawPtr.(hue.EntertainmentConfiguration)
	return cfg, ok
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
	deviceMap := make(map[string]hueclient.DeviceGet)
	for _, d := range allDevices {
		if d.Id != nil {
			deviceMap[*d.Id] = d
		}
	}

	// Build device -> lights mapping (lights are owned by devices)
	deviceLights := make(map[string][]hueclient.LightGet)
	for _, light := range allLights {
		if light.Owner != nil && light.Owner.Rid != nil {
			deviceLights[*light.Owner.Rid] = append(deviceLights[*light.Owner.Rid], light)
		}
	}

	// Build group ID -> scenes mapping
	groupScenes := make(map[string][]hueclient.SceneGet)
	for _, scene := range allScenes {
		if scene.Group != nil && scene.Group.Rid != nil {
			groupScenes[*scene.Group.Rid] = append(groupScenes[*scene.Group.Rid], scene)
		}
	}

	// Helper to create a room/zone node with its children
	buildGroupNode := func(group hueclient.RoomGet, isZone bool) *TreeNode {
		groupID := ""
		groupName := ""
		if group.Id != nil {
			groupID = *group.Id
		}
		if group.Metadata != nil && group.Metadata.Name != nil {
			groupName = *group.Metadata.Name
		}

		label := groupName
		entityType := EntityRoom
		if isZone {
			label = groupName + " (zone)"
			entityType = EntityZone
		}

		groupNode := &TreeNode{
			Label: label,
			Item: &EntityItem{
				ID:     groupID,
				Name:   groupName,
				Type:   entityType,
				RawPtr: group, // Store the actual RoomGet for details panel
			},
			Expanded: false, // Start collapsed
		}

		// Collect lights in this group (through device children)
		var groupLights []hueclient.LightGet
		var groupDevices []hueclient.DeviceGet

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

		// Sort lights by name (from owning device, not deprecated light metadata)
		sort.Slice(groupLights, func(i, j int) bool {
			nameI, nameJ := "", ""
			// Get name from owning device
			if groupLights[i].Owner != nil && groupLights[i].Owner.Rid != nil {
				if device, ok := deviceMap[*groupLights[i].Owner.Rid]; ok {
					if device.Metadata != nil && device.Metadata.Name != nil {
						nameI = *device.Metadata.Name
					}
				}
			}
			if groupLights[j].Owner != nil && groupLights[j].Owner.Rid != nil {
				if device, ok := deviceMap[*groupLights[j].Owner.Rid]; ok {
					if device.Metadata != nil && device.Metadata.Name != nil {
						nameJ = *device.Metadata.Name
					}
				}
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
				Label: fmt.Sprintf("Lights (%d)", len(groupLights)),
				Item: &EntityItem{
					ID:   groupID + ":lights",
					Name: "Lights",
					Type: EntityLightsCategory,
					RawPtr: LightsCategoryData{
						ParentName: groupName,
						Lights:     groupLights,
					},
				},
				Expanded: false, // Start collapsed
			}
			for _, light := range groupLights {
				lightsNode.Children = append(lightsNode.Children, buildLightNode(light, state))
			}
			groupNode.Children = append(groupNode.Children, lightsNode)
		}

		// Devices category (non-light devices)
		if len(groupDevices) > 0 {
			devicesNode := &TreeNode{
				Label: fmt.Sprintf("Devices (%d)", len(groupDevices)),
				Item: &EntityItem{
					ID:   groupID + ":devices",
					Name: "Devices",
					Type: EntityDevicesCategory,
					RawPtr: DevicesCategoryData{
						ParentName: groupName,
						Devices:    groupDevices,
					},
				},
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
				Label: fmt.Sprintf("Scenes (%d)", len(scenes)),
				Item: &EntityItem{
					ID:   groupID + ":scenes",
					Name: "Scenes",
					Type: EntityScenesCategory,
					RawPtr: ScenesCategoryData{
						ParentName: groupName,
						Scenes:     scenes,
					},
				},
				Expanded: false, // Start collapsed
			}
			for _, scene := range scenes {
				scenesNode.Children = append(scenesNode.Children, buildSceneNode(scene, state))
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

	// Add entertainment areas
	entertainmentConfigs := state.AllEntertainmentConfigurations()
	for _, cfg := range entertainmentConfigs {
		name := cfg.ID
		if cfg.Metadata != nil && cfg.Metadata.Name != "" {
			name = cfg.Metadata.Name
		}

		entNode := &TreeNode{
			Label: name + " (entertainment)",
			Item: &EntityItem{
				ID:     cfg.ID,
				Name:   name,
				Type:   EntityEntertainment,
				IsOn:   cfg.Status == "streaming",
				RawPtr: cfg,
			},
			Expanded: false,
		}

		// Add lights in this entertainment area as children
		for _, lightEntry := range cfg.Lights {
			if lightEntry.Service != nil && lightEntry.Service.RID != "" {
				if light, ok := state.GetLight(lightEntry.Service.RID); ok {
					entNode.Children = append(entNode.Children, buildLightNode(light, state))
				}
			}
		}

		roots = append(roots, entNode)
	}

	// Build ungrouped section
	var ungroupedLights []hueclient.LightGet
	var ungroupedDevices []hueclient.DeviceGet

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
				if svc.Rtype != nil && *svc.Rtype == hueclient.ResourceIdentifierRtypeBridge {
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
			Label: "─── Ungrouped ───",
			Item: &EntityItem{
				ID:   "_ungrouped",
				Name: "Ungrouped",
			},
			Expanded: false, // Start collapsed
		}

		if len(ungroupedLights) > 0 {
			// Sort by name (from owning device, not deprecated light metadata)
			sort.Slice(ungroupedLights, func(i, j int) bool {
				nameI, nameJ := "", ""
				if ungroupedLights[i].Owner != nil && ungroupedLights[i].Owner.Rid != nil {
					if device, ok := deviceMap[*ungroupedLights[i].Owner.Rid]; ok {
						if device.Metadata != nil && device.Metadata.Name != nil {
							nameI = *device.Metadata.Name
						}
					}
				}
				if ungroupedLights[j].Owner != nil && ungroupedLights[j].Owner.Rid != nil {
					if device, ok := deviceMap[*ungroupedLights[j].Owner.Rid]; ok {
						if device.Metadata != nil && device.Metadata.Name != nil {
							nameJ = *device.Metadata.Name
						}
					}
				}
				return nameI < nameJ
			})

			lightsNode := &TreeNode{
				Label: fmt.Sprintf("Lights (%d)", len(ungroupedLights)),
				Item: &EntityItem{
					ID:   "_ungrouped:lights",
					Name: "Lights",
					Type: EntityLightsCategory,
					RawPtr: LightsCategoryData{
						ParentName: "Ungrouped",
						Lights:     ungroupedLights,
					},
				},
				Expanded: false, // Start collapsed
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
				Label: fmt.Sprintf("Devices (%d)", len(ungroupedDevices)),
				Item: &EntityItem{
					ID:   "_ungrouped:devices",
					Name: "Devices",
					Type: EntityDevicesCategory,
					RawPtr: DevicesCategoryData{
						ParentName: "Ungrouped",
						Devices:    ungroupedDevices,
					},
				},
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
// Thresholds are centered around the visual representation:
// ○=0%, ◔=25%, ◑=50%, ◕=75%, ●=100%
func brightnessIndicator(brightness float64) string {
	switch {
	case brightness <= 0:
		return "○" // off/empty
	case brightness < 37.5:
		return "◔" // quarter (1-37%)
	case brightness < 62.5:
		return "◑" // half (38-62%)
	case brightness < 87.5:
		return "◕" // three-quarters (63-87%)
	default:
		return "●" // full (88-100%)
	}
}

// RenderEntityIndicator renders the appropriate indicator for an entity.
// This is the single source of truth for all entity indicators in the app.
// Parameters:
//   - item: the entity to render an indicator for
//   - styles: UI styles for colors
//   - selected: whether the item is currently selected (for background color)
//
// Returns the styled indicator string.
func RenderEntityIndicator(item EntityItem, styles ui.Styles, selected bool) string {
	// Helper to apply selection background if needed
	withSelectionBg := func(style lipgloss.Style) lipgloss.Style {
		if selected {
			return style.Background(styles.SelectedItem.GetBackground())
		}
		return style
	}

	switch item.Type {
	case EntityLight:
		return renderLightIndicator(item, styles, selected)
	case EntityDevice:
		// Devices with motion show on/off, others show neutral bullet
		if item.IsOn {
			style := withSelectionBg(lipgloss.NewStyle().Foreground(styles.Theme.OnColor))
			return style.Render(IndicatorOn)
		}
		// Check if it's a motion sensor
		if device, ok := item.RawPtr.(hueclient.DeviceGet); ok {
			if device.Services != nil {
				for _, svc := range *device.Services {
					if svc.Rtype != nil && *svc.Rtype == "motion" {
						style := withSelectionBg(lipgloss.NewStyle().Foreground(styles.Theme.Muted))
						return style.Render(IndicatorOff) // Motion sensor, not detecting
					}
				}
			}
		}
		style := withSelectionBg(lipgloss.NewStyle().Foreground(styles.Theme.Foreground))
		return style.Render(IndicatorNeutral) // Non-motion device
	case EntityScene:
		if item.IsOn {
			// Use colored brightness indicator if available (from scene's room average)
			indicatorChar := brightnessIndicator(item.Brightness)
			if item.IndicatorColor != "" {
				style := withSelectionBg(lipgloss.NewStyle().Foreground(lipgloss.Color(item.IndicatorColor)))
				return style.Render(indicatorChar)
			}
			style := withSelectionBg(lipgloss.NewStyle().Foreground(styles.Theme.OnColor))
			return style.Render(indicatorChar)
		}
		style := withSelectionBg(lipgloss.NewStyle().Foreground(styles.Theme.Muted))
		return style.Render(IndicatorOff)
	case EntityRoom, EntityZone:
		if item.IsOn {
			style := withSelectionBg(lipgloss.NewStyle().Foreground(styles.Theme.OnColor))
			return style.Render(IndicatorOn)
		}
		style := withSelectionBg(lipgloss.NewStyle().Foreground(styles.Theme.OffColor))
		return style.Render(IndicatorOff)
	case EntityEntertainment:
		if item.IsOn {
			style := withSelectionBg(lipgloss.NewStyle().Foreground(styles.Theme.Success))
			return style.Render(IndicatorOn)
		}
		style := withSelectionBg(lipgloss.NewStyle().Foreground(styles.Theme.Muted))
		return style.Render(IndicatorOff)
	default:
		if item.IsOn {
			style := withSelectionBg(lipgloss.NewStyle().Foreground(styles.Theme.OnColor))
			return style.Render(IndicatorOn)
		}
		style := withSelectionBg(lipgloss.NewStyle().Foreground(styles.Theme.Muted))
		return style.Render(IndicatorOff)
	}
}

// RenderEntityLine renders a complete entity line with indicator, name, and proper width handling.
// This is the centralized function for rendering list items across all panels.
// Parameters:
//   - item: the entity to render
//   - width: total available width for the line
//   - selected: whether this item is currently selected
//   - active: whether the panel is active (focused)
//   - styles: UI styles for colors
//
// Returns the fully styled line string.
func RenderEntityLine(item EntityItem, width int, selected, active bool, styles ui.Styles) string {
	isSelectedActive := selected && active

	// Get the indicator with proper selection background
	indicator := RenderEntityIndicator(item, styles, isSelectedActive)
	indicatorWidth := lipgloss.Width(indicator)

	// Calculate available width for the rest of the line (space + name + padding)
	restWidth := width - indicatorWidth
	if restWidth < 1 {
		restWidth = 1
	}

	// Build name portion
	name := item.Name
	nameWithSpace := " " + name

	// Truncate name if needed
	if lipgloss.Width(nameWithSpace) > restWidth {
		availableForName := restWidth - 2 // space + ellipsis
		if availableForName > 0 {
			name = ui.TruncateString(name, availableForName) + "…"
			nameWithSpace = " " + name
		}
	}

	// Pad to fill remaining width
	currentWidth := lipgloss.Width(nameWithSpace)
	if currentWidth < restWidth {
		nameWithSpace = nameWithSpace + strings.Repeat(" ", restWidth-currentWidth)
	}

	// Apply style to the rest (indicator already has its own styling)
	var style lipgloss.Style
	if isSelectedActive {
		style = styles.SelectedItem
	} else {
		style = styles.ListItem
	}
	styledRest := style.Render(nameWithSpace)

	return indicator + styledRest
}

// RenderTreeLine renders a tree node line with indentation, expand indicator, entity indicator, and name.
// This handles the extra complexity of tree views (depth, expand/collapse).
func RenderTreeLine(node *TreeNode, depth int, width int, selected, active bool, styles ui.Styles) string {
	isSelectedActive := selected && active
	lineWidth := max(1, width)

	// Build indentation prefix
	prefix := strings.Repeat("  ", depth)

	// Expand indicator for groups
	expandIndicator := "  "
	if len(node.Children) > 0 {
		if node.Expanded {
			expandIndicator = "▼ "
		} else {
			expandIndicator = "▶ "
		}
	}

	// If this is an entity item, use centralized indicator
	if node.Item != nil {
		indicator := RenderEntityIndicator(*node.Item, styles, isSelectedActive)
		indicatorWidth := lipgloss.Width(indicator)

		// Calculate available width for prefix + expand + indicator + space + name
		prefixPart := prefix + expandIndicator
		prefixWidth := lipgloss.Width(prefixPart)
		restWidth := lineWidth - prefixWidth - indicatorWidth
		if restWidth < 1 {
			restWidth = 1
		}

		name := node.Item.Name
		nameWithSpace := " " + name

		// Truncate name if needed
		if lipgloss.Width(nameWithSpace) > restWidth {
			availableForName := restWidth - 2 // space + ellipsis
			if availableForName > 0 {
				name = ui.TruncateString(name, availableForName) + "…"
				nameWithSpace = " " + name
			}
		}

		// Pad to fill remaining width
		currentWidth := lipgloss.Width(nameWithSpace)
		if currentWidth < restWidth {
			nameWithSpace = nameWithSpace + strings.Repeat(" ", restWidth-currentWidth)
		}

		// Apply style
		var style lipgloss.Style
		if isSelectedActive {
			style = styles.SelectedItem
		} else {
			style = styles.ListItem
		}

		styledPrefix := style.Render(prefixPart)
		styledRest := style.Render(nameWithSpace)

		return styledPrefix + indicator + styledRest
	}

	// Non-entity node (folder/group header)
	line := prefix + expandIndicator + node.Label
	visibleWidth := lipgloss.Width(line)
	if visibleWidth < lineWidth {
		line = line + strings.Repeat(" ", lineWidth-visibleWidth)
	}

	// Style for folders
	var style lipgloss.Style
	if isSelectedActive {
		style = styles.SelectedItem
	} else {
		style = styles.Muted.Bold(true)
	}

	return style.Render(line)
}

// renderLightIndicator renders a colored brightness indicator for a light.
func renderLightIndicator(item EntityItem, styles ui.Styles, selected bool) string {
	if !item.IsOn {
		if selected {
			// Add selection background to off indicator
			return lipgloss.NewStyle().
				Foreground(styles.Theme.OffColor).
				Background(styles.SelectedItem.GetBackground()).
				Render(IndicatorOff)
		}
		return styles.OffIndicator.String()
	}

	// Use brightness from EntityItem, fallback to RawPtr for backwards compatibility
	brightness := item.Brightness
	if brightness == 0 {
		if light, ok := item.RawPtr.(hueclient.LightGet); ok {
			if light.Dimming != nil && light.Dimming.Brightness != nil {
				brightness = float64(*light.Dimming.Brightness)
			} else {
				brightness = 100.0
			}
		} else {
			brightness = 100.0
		}
	}

	indicatorChar := brightnessIndicator(brightness)

	// Apply color if available
	if item.IndicatorColor != "" {
		indicatorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(item.IndicatorColor))
		if selected {
			indicatorStyle = indicatorStyle.Background(styles.SelectedItem.GetBackground())
		}
		return indicatorStyle.Render(indicatorChar)
	}

	// No color - use default on color
	return lipgloss.NewStyle().Foreground(styles.Theme.OnColor).Render(indicatorChar)
}

// RenderLightIndicatorFromLight renders an indicator directly from an hueclient.LightGet.
// Used when we have the light object directly, not wrapped in an EntityItem.
func RenderLightIndicatorFromLight(light hueclient.LightGet, styles ui.Styles) string {
	if !IsLightOn(light) {
		return styles.OffIndicator.String()
	}

	// Get brightness
	brightness := 100.0
	if light.Dimming != nil && light.Dimming.Brightness != nil {
		brightness = float64(*light.Dimming.Brightness)
	}

	color := GetLightColor(light)
	return RenderBrightnessIndicator(brightness, color)
}

// RenderBrightnessIndicator renders a brightness indicator character with the given color.
// This is the most generic form, used when you have pre-calculated brightness and color.
func RenderBrightnessIndicator(brightness float64, color lipgloss.Color) string {
	indicatorChar := brightnessIndicator(brightness)
	return lipgloss.NewStyle().Foreground(color).Render(indicatorChar)
}

// RenderOffIndicator renders a muted hollow circle indicator for off/inactive state.
func RenderOffIndicator(styles ui.Styles) string {
	return styles.OffIndicator.String()
}

// PlainIndicator returns a plain (unstyled) indicator character for on/off state.
// Used for building tree node labels where coloring is handled separately.
func PlainIndicator(isOn bool) string {
	if isOn {
		return IndicatorOn
	}
	return IndicatorOff
}

// Indicator character constants for consistency across the app.
const (
	IndicatorOn      = "●" // Filled circle - on/active state
	IndicatorOff     = "○" // Hollow circle - off/inactive state
	IndicatorNeutral = "•" // Bullet - neutral/no state (devices without sensors)
)

// PlainEntityIndicator returns the appropriate plain-text indicator for an entity.
// Used for building tree node labels.
func PlainEntityIndicator(item EntityItem) string {
	switch item.Type {
	case EntityLight:
		if !item.IsOn {
			return IndicatorOff
		}
		// For lights, get brightness from RawPtr
		brightness := 100.0
		if light, ok := item.RawPtr.(hueclient.LightGet); ok {
			if light.Dimming != nil && light.Dimming.Brightness != nil {
				brightness = float64(*light.Dimming.Brightness)
			}
		}
		return brightnessIndicator(brightness)
	case EntityDevice:
		// Devices show on/off for motion sensors, neutral for others
		if item.IsOn {
			return IndicatorOn
		}
		// Check if it's a motion sensor by looking at RawPtr
		if device, ok := item.RawPtr.(hueclient.DeviceGet); ok {
			// If we have a device, check if it has motion capability
			// Devices without motion sensing get neutral indicator
			if device.Services != nil {
				for _, svc := range *device.Services {
					if svc.Rtype != nil && *svc.Rtype == "motion" {
						return IndicatorOff // Motion sensor, not detecting
					}
				}
			}
			return IndicatorNeutral // Non-motion device
		}
		return IndicatorNeutral
	case EntityScene:
		return PlainIndicator(item.IsOn)
	case EntityRoom, EntityZone, EntityEntertainment:
		return PlainIndicator(item.IsOn)
	default:
		return PlainIndicator(item.IsOn)
	}
}

// Color conversion functions moved to internal/ui/color.go:
// - ui.XyToRGB
// - ui.MirekToRGB

// GetLightColor extracts the RGB color from a light.
func GetLightColor(light hueclient.LightGet) lipgloss.Color {
	brightness := 100.0
	if light.Dimming != nil && light.Dimming.Brightness != nil {
		brightness = float64(*light.Dimming.Brightness)
	}

	// Try XY color first (color lights)
	if light.Color != nil && light.Color.Xy != nil {
		if light.Color.Xy.X != nil && light.Color.Xy.Y != nil {
			x := float64(*light.Color.Xy.X)
			y := float64(*light.Color.Xy.Y)
			r, g, b := ui.XyToRGB(x, y, brightness)
			return lipgloss.Color(fmt.Sprintf("#%02x%02x%02x", r, g, b))
		}
	}

	// Try color temperature (white ambiance lights)
	if light.ColorTemperature != nil && light.ColorTemperature.Mirek != nil {
		if light.ColorTemperature.MirekValid == nil || *light.ColorTemperature.MirekValid {
			r, g, b := ui.MirekToRGB(*light.ColorTemperature.Mirek)
			return lipgloss.Color(fmt.Sprintf("#%02x%02x%02x", r, g, b))
		}
	}

	// Default to warm white for non-color lights
	return lipgloss.Color("#ffcc66")
}

// buildLightNode creates a tree node for a light.
func buildLightNode(light hueclient.LightGet, state *hue.BridgeState) *TreeNode {
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
	var indicatorColor string
	var indicator string
	var brightness float64
	if IsLightOn(light) {
		brightness = 100.0
		if light.Dimming != nil && light.Dimming.Brightness != nil {
			brightness = float64(*light.Dimming.Brightness)
		}
		indicator = brightnessIndicator(brightness)
		indicatorColor = string(GetLightColor(light))
	} else {
		indicator = IndicatorOff
	}
	label := indicator + " " + name

	return &TreeNode{
		Label: label,
		Item: &EntityItem{
			ID:             id,
			Name:           name,
			Type:           EntityLight,
			IsOn:           IsLightOn(light),
			RawPtr:         light,
			IndicatorColor: indicatorColor,
			Brightness:     brightness,
		},
	}
}

// buildDeviceNode creates a tree node for a device.
func buildDeviceNode(device hueclient.DeviceGet, state *hue.BridgeState) *TreeNode {
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

	var indicator string
	var indicatorColor string
	isOn := false
	if hasMotion {
		if isDetecting {
			indicator = IndicatorOn
			indicatorColor = ui.ColorOn
			isOn = true
		} else {
			indicator = IndicatorOff
			indicatorColor = ui.ColorOff
		}
	} else {
		indicator = IndicatorNeutral
		indicatorColor = "" // default styling
	}
	label := indicator + " " + name

	return &TreeNode{
		Label: label,
		Item: &EntityItem{
			ID:             id,
			Name:           name,
			Type:           EntityDevice,
			IsOn:           isOn,
			RawPtr:         device,
			IndicatorColor: indicatorColor,
		},
	}
}

// buildSceneNode creates a tree node for a scene.
func buildSceneNode(scene hueclient.SceneGet, state *hue.BridgeState) *TreeNode {
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
		isActive = *scene.Status.Active != hueclient.SceneGetStatusActiveInactive
	}

	var indicator string
	var indicatorColor string
	var brightness float64

	if isActive && state != nil && scene.Group != nil && scene.Group.Rid != nil {
		// Calculate average color/brightness from the scene's room/zone lights
		var lights []hueclient.LightGet
		if room, ok := state.GetRoom(*scene.Group.Rid); ok {
			lights = state.RoomLights(room)
		} else if zone, ok := state.GetZone(*scene.Group.Rid); ok {
			lights = state.RoomLights(zone)
		}

		var totalBrightness float64
		var totalR, totalG, totalB float64
		var onCount int

		for _, light := range lights {
			if IsLightOn(light) {
				onCount++
				if light.Dimming != nil && light.Dimming.Brightness != nil {
					totalBrightness += float64(*light.Dimming.Brightness)
				} else {
					totalBrightness += 100
				}
				color := GetLightColor(light)
				colorStr := string(color)
				if len(colorStr) == 7 && colorStr[0] == '#' {
					var r, g, b int
					fmt.Sscanf(colorStr, "#%02x%02x%02x", &r, &g, &b)
					totalR += float64(r)
					totalG += float64(g)
					totalB += float64(b)
				}
			}
		}

		if onCount > 0 {
			brightness = totalBrightness / float64(onCount)
			avgR := int(totalR / float64(onCount))
			avgG := int(totalG / float64(onCount))
			avgB := int(totalB / float64(onCount))
			indicator = brightnessIndicator(brightness)
			indicatorColor = fmt.Sprintf("#%02x%02x%02x", avgR, avgG, avgB)
		} else {
			indicator = IndicatorOn
			brightness = 100
		}
	} else if isActive {
		indicator = IndicatorOn
		brightness = 100
	} else {
		indicator = IndicatorOff
		brightness = 0
	}

	label := indicator + " " + name

	return &TreeNode{
		Label: label,
		Item: &EntityItem{
			ID:             id,
			Name:           name,
			Type:           EntityScene,
			IsOn:           isActive,
			RawPtr:         scene,
			IndicatorColor: indicatorColor,
			Brightness:     brightness,
		},
	}
}
