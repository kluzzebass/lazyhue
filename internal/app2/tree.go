package app2

import (
	"fmt"
	"sort"
	"time"

	"github.com/kluzzebass/lazyhue/internal/hue"
	"github.com/kluzzebass/lazyhue/internal/hueclient"
	"github.com/kluzzebass/lazyhue/internal/ui2"
	"github.com/kluzzebass/lazyhue/internal/ui2/panels"
)

// buildHomeTree builds the tree panel showing Bridges as root nodes,
// with Rooms, Zones, and Entertainment Areas as children of each bridge.
func (m *Model) buildHomeTree(_ *hue.BridgeState) {
	allBridges := m.manager.AllBridges()

	var nodes []*panels.TreeNode

	// Sort bridges by name for consistent ordering
	sort.Slice(allBridges, func(i, j int) bool {
		return allBridges[i].Info.Name < allBridges[j].Info.Name
	})

	// Create a node for each bridge
	for _, bridge := range allBridges {
		bridgeName := bridge.Info.Name
		if bridgeName == "" {
			bridgeName = bridge.Info.ID
		}

		// Check if bridge is currently blinking
		isBlinking := false
		if until, ok := m.bridgeBlinkUntil[bridge.Info.ID]; ok {
			isBlinking = time.Now().Before(until)
		}

		// Create bridge node
		bridgeNode := &panels.TreeNode{
			ID:       bridge.Info.ID,
			Label:    bridgeName,
			Depth:    0,
			Expanded: true, // Start expanded
			Item: &panels.EntityItem{
				ID:       bridge.Info.ID,
				Name:     bridgeName,
				Type:     panels.EntityBridge,
				IsOn:     bridge.IsConnected(), // Use connection status for IsOn
				RawPtr:   bridge,
				BridgeID: bridge.Info.ID,
				Brightness: func() float64 {
					if isBlinking {
						return 100.0 // Full brightness when blinking
					}
					return 0 // No brightness indicator when not blinking
				}(),
			},
			Children: make([]*panels.TreeNode, 0),
		}

		// Only add children if bridge is connected
		if bridge.IsConnected() {
			state := bridge.GetState()
			if state != nil {
				// Build rooms, zones, and entertainment areas as children
				m.buildBridgeChildren(bridgeNode, state, bridge.Info.ID)
			}
		}

		nodes = append(nodes, bridgeNode)
	}

	m.tree.SetRoots(nodes)
}

// buildBridgeChildren adds Rooms, Zones, and Entertainment Areas as children to a bridge node.
func (m *Model) buildBridgeChildren(bridgeNode *panels.TreeNode, state *hue.BridgeState, bridgeID string) {
	// Rooms category
	rooms := state.AllRooms()
	if len(rooms) > 0 {
		roomsNode := &panels.TreeNode{
			ID:       bridgeNode.ID + ":rooms",
			Label:    fmt.Sprintf("Rooms (%d)", len(rooms)),
			Depth:    1,
			Expanded: true,
			Children: make([]*panels.TreeNode, 0, len(rooms)),
		}

		for _, room := range rooms {
			name := "Unknown"
			if room.Metadata != nil && room.Metadata.Name != nil {
				name = *room.Metadata.Name
			}

			roomID := ""
			if room.Id != nil {
				roomID = *room.Id
			}

			// Get lights for this room
			lights := state.RoomLights(room)

			// Calculate aggregate brightness and color from room lights
			brightness, indicatorColor := hue.CalculateRoomAggregate(lights)

			roomNode := &panels.TreeNode{
				ID:       roomID,
				Label:    name,
				Depth:    2,
				Expanded: false,
				Item: &panels.EntityItem{
					ID:             roomID,
					Name:           name,
					Type:           panels.EntityRoom,
					IsOn:           brightness > 0,
					Brightness:     brightness,
					IndicatorColor: indicatorColor,
					RawPtr:         room,
					BridgeID:       bridgeID,
				},
				Children: make([]*panels.TreeNode, 0),
			}

			// Get devices and scenes for this room
			scenes := state.RoomScenes(roomID)

			// Get non-light devices in room
			var nonLightDevices []hueclient.DeviceGet
			if room.Children != nil {
				for _, child := range *room.Children {
					if child.Rtype != nil && *child.Rtype == "device" && child.Rid != nil {
						device, found := state.GetDevice(*child.Rid)
						if !found {
							continue
						}
						// Check if device has lights (if so, skip - it's a light device)
						hasLightService := false
						if device.Services != nil {
							for _, svc := range *device.Services {
								if svc.Rtype != nil && *svc.Rtype == "light" {
									hasLightService = true
									break
								}
							}
						}
						if !hasLightService {
							nonLightDevices = append(nonLightDevices, device)
						}
					}
				}
			}

			// Lights subsection
			if len(lights) > 0 {
				lightsNode := &panels.TreeNode{
					ID:       roomID + ":lights",
					Label:    fmt.Sprintf("Lights (%d)", len(lights)),
					Depth:    3,
					Expanded: false,
					Children: make([]*panels.TreeNode, 0, len(lights)),
				}
				for _, light := range lights {
					lightID := ""
					if light.Id != nil {
						lightID = *light.Id
					}
					lightName := state.GetLightName(light)
					isOn := light.On != nil && light.On.On != nil && *light.On.On
					brightness, indicatorColor := getLightBrightnessAndColor(light)
					// Store a copy of the light in RawPtr for later access
					lightCopy := light
					lightsNode.Children = append(lightsNode.Children, &panels.TreeNode{
						ID:    lightID,
						Label: lightName,
						Depth: 4,
						Item: &panels.EntityItem{
							ID:             lightID,
							Name:           lightName,
							Type:           panels.EntityLight,
							IsOn:           isOn,
							Brightness:     brightness,
							IndicatorColor: indicatorColor,
							RawPtr:         lightCopy,
							BridgeID:       bridgeID,
						},
					})
				}
				roomNode.Children = append(roomNode.Children, lightsNode)
			}

			// Devices subsection
			if len(nonLightDevices) > 0 {
				devicesNode := &panels.TreeNode{
					ID:       roomID + ":devices",
					Label:    fmt.Sprintf("Devices (%d)", len(nonLightDevices)),
					Depth:    3,
					Expanded: false,
					Children: make([]*panels.TreeNode, 0, len(nonLightDevices)),
				}
				for _, device := range nonLightDevices {
					deviceID := ""
					deviceName := device.DeviceName("")
					if device.Id != nil {
						deviceID = *device.Id
					}
					// Check if device has motion sensor and its state
					hasMotion, isDetecting := state.GetDeviceMotionState(device)
					isOn := hasMotion && isDetecting
					devicesNode.Children = append(devicesNode.Children, &panels.TreeNode{
						ID:    deviceID,
						Label: deviceName,
						Depth: 4,
						Item: &panels.EntityItem{
							ID:       deviceID,
							Name:     deviceName,
							Type:     panels.EntityDevice,
							IsOn:     isOn,
							BridgeID: bridgeID,
						},
					})
				}
				roomNode.Children = append(roomNode.Children, devicesNode)
			}

			// Scenes subsection
			if len(scenes) > 0 {
				scenesNode := &panels.TreeNode{
					ID:       roomID + ":scenes",
					Label:    fmt.Sprintf("Scenes (%d)", len(scenes)),
					Depth:    3,
					Expanded: false,
					Children: make([]*panels.TreeNode, 0, len(scenes)),
				}
				for _, scene := range scenes {
					sceneID := ""
					sceneName := scene.SceneName("")
					if scene.Id != nil {
						sceneID = *scene.Id
					}
					scenesNode.Children = append(scenesNode.Children, &panels.TreeNode{
						ID:    sceneID,
						Label: sceneName,
						Depth: 4,
						Item: &panels.EntityItem{
							ID:       sceneID,
							Name:     sceneName,
							Type:     panels.EntityScene,
							BridgeID: bridgeID,
						},
					})
				}
				roomNode.Children = append(roomNode.Children, scenesNode)
			}

			roomsNode.Children = append(roomsNode.Children, roomNode)
		}
		bridgeNode.Children = append(bridgeNode.Children, roomsNode)
	}

	// Zones category
	zones := state.AllZones()
	if len(zones) > 0 {
		zonesNode := &panels.TreeNode{
			ID:       bridgeNode.ID + ":zones",
			Label:    fmt.Sprintf("Zones (%d)", len(zones)),
			Depth:    1,
			Expanded: false,
			Children: make([]*panels.TreeNode, 0, len(zones)),
		}

		for _, zone := range zones {
			name := "Unknown"
			if zone.Metadata != nil && zone.Metadata.Name != nil {
				name = *zone.Metadata.Name
			}

			zoneID := ""
			if zone.Id != nil {
				zoneID = *zone.Id
			}

			// Get lights for this zone
			lights := state.RoomLights(zone)

			// Calculate aggregate brightness and color from zone lights
			brightness, indicatorColor := hue.CalculateRoomAggregate(lights)

			zoneNode := &panels.TreeNode{
				ID:       zoneID,
				Label:    name,
				Depth:    2,
				Expanded: false,
				Item: &panels.EntityItem{
					ID:             zoneID,
					Name:           name,
					Type:           panels.EntityZone,
					IsOn:           brightness > 0,
					Brightness:     brightness,
					IndicatorColor: indicatorColor,
					RawPtr:         zone,
					BridgeID:       bridgeID,
				},
				Children: make([]*panels.TreeNode, 0),
			}

			// Add lights in zone
			if zone.Children != nil {
				for _, child := range *zone.Children {
					if child.Rtype != nil && *child.Rtype == "device" && child.Rid != nil {
						device, found := state.GetDevice(*child.Rid)
						if found && device.Metadata != nil && device.Metadata.Name != nil && device.Services != nil {
							for _, svc := range *device.Services {
								if svc.Rtype != nil && *svc.Rtype == "light" && svc.Rid != nil {
									light, lightFound := state.GetLight(*svc.Rid)
									if lightFound {
										isOn := light.On != nil && light.On.On != nil && *light.On.On
										brightness, indicatorColor := getLightBrightnessAndColor(light)
										// Store a copy of the light in RawPtr for later access
										lightCopy := light
										zoneNode.Children = append(zoneNode.Children, &panels.TreeNode{
											ID:    *svc.Rid,
											Label: *device.Metadata.Name,
											Depth: 3,
											Item: &panels.EntityItem{
												ID:             *svc.Rid,
												Name:           *device.Metadata.Name,
												Type:           panels.EntityLight,
												IsOn:           isOn,
												Brightness:     brightness,
												IndicatorColor: indicatorColor,
												RawPtr:         lightCopy,
												BridgeID:       bridgeID,
											},
										})
									}
								}
							}
						}
					}
				}
			}

			zonesNode.Children = append(zonesNode.Children, zoneNode)
		}
		bridgeNode.Children = append(bridgeNode.Children, zonesNode)
	}

	// Entertainment configurations category
	entConfigs := state.AllEntertainmentConfigurations()
	if len(entConfigs) > 0 {
		entNode := &panels.TreeNode{
			ID:       bridgeNode.ID + ":entertainment",
			Label:    fmt.Sprintf("Entertainment Areas (%d)", len(entConfigs)),
			Depth:    1,
			Expanded: false,
			Children: make([]*panels.TreeNode, 0, len(entConfigs)),
		}

		for _, ent := range entConfigs {
			name := ent.EntertainmentName("Unknown")

			entItemNode := &panels.TreeNode{
				ID:       ent.ID,
				Label:    name,
				Depth:    2,
				Expanded: false,
				Item: &panels.EntityItem{
					ID:       ent.ID,
					Name:     name,
					Type:     panels.EntityEntertainment,
					BridgeID: bridgeID,
				},
				Children: make([]*panels.TreeNode, 0),
			}

			// Add lights in entertainment configuration
			for _, lightEntry := range ent.Lights {
				if lightEntry.Service != nil && lightEntry.Service.RID != "" {
					light, lightFound := state.GetLight(lightEntry.Service.RID)
					if lightFound {
						isOn := light.On != nil && light.On.On != nil && *light.On.On
						lightName := state.GetLightName(light)

						brightness, indicatorColor := getLightBrightnessAndColor(light)
						// Store a copy of the light in RawPtr for later access
						lightCopy := light
						entItemNode.Children = append(entItemNode.Children, &panels.TreeNode{
							ID:    lightEntry.Service.RID,
							Label: lightName,
							Depth: 3,
							Item: &panels.EntityItem{
								ID:             lightEntry.Service.RID,
								Name:           lightName,
								Type:           panels.EntityLight,
								IsOn:           isOn,
								Brightness:     brightness,
								IndicatorColor: indicatorColor,
								RawPtr:         lightCopy,
								BridgeID:       bridgeID,
							},
						})
					}
				}
			}

			entNode.Children = append(entNode.Children, entItemNode)
		}
		bridgeNode.Children = append(bridgeNode.Children, entNode)
	}
}

// buildLightsTree builds the tree panel showing all Lights from all bridges.
func (m *Model) buildLightsTree(_ *hue.BridgeState) {
	allBridges := m.manager.AllBridges()
	var nodes []*panels.TreeNode

	for _, bridge := range allBridges {
		if !bridge.IsConnected() {
			continue
		}
		state := bridge.GetState()
		if state == nil {
			continue
		}

		bridgeName := bridge.Info.Name
		lights := state.AllLights()
		for _, light := range lights {
			name := state.GetLightName(light)
			isOn := light.On != nil && light.On.On != nil && *light.On.On
			brightness, indicatorColor := getLightBrightnessAndColor(light)

			// Store a copy of the light in RawPtr for later access
			lightCopy := light
			nodes = append(nodes, &panels.TreeNode{
				ID:           *light.Id,
				Label:        name,
				BridgeSuffix: "[" + bridgeName + "]",
				Depth:        0,
				Item: &panels.EntityItem{
					ID:             *light.Id,
					Name:           name,
					Type:           panels.EntityLight,
					IsOn:           isOn,
					Brightness:     brightness,
					IndicatorColor: indicatorColor,
					RawPtr:         lightCopy,
					BridgeID:       bridge.Info.ID,
				},
			})
		}
	}

	// Sort by name
	sort.Slice(nodes, func(i, j int) bool {
		return nodes[i].Item.Name < nodes[j].Item.Name
	})

	m.tree.SetRoots(nodes)
}

// buildDevicesTree builds the tree panel showing all Devices from all bridges.
func (m *Model) buildDevicesTree(_ *hue.BridgeState) {
	allBridges := m.manager.AllBridges()
	var nodes []*panels.TreeNode

	for _, bridge := range allBridges {
		if !bridge.IsConnected() {
			continue
		}
		state := bridge.GetState()
		if state == nil {
			continue
		}

		devices := state.AllDevices()
		rooms := state.AllRooms()

		// Build device -> room name mapping
		deviceRoomNames := make(map[string]string)
		for _, room := range rooms {
			if room.Children == nil {
				continue
			}
			roomName := ""
			if room.Metadata != nil && room.Metadata.Name != nil {
				roomName = *room.Metadata.Name
			}
			for _, child := range *room.Children {
				if child.Rtype != nil && *child.Rtype == "device" && child.Rid != nil {
					deviceRoomNames[*child.Rid] = roomName
				}
			}
		}

		// Build a set of device IDs that own lights (to exclude them)
		lightOwnerIDs := make(map[string]bool)
		for _, light := range state.AllLights() {
			if light.Owner != nil && light.Owner.Rid != nil {
				lightOwnerIDs[*light.Owner.Rid] = true
			}
		}

		bridgeName := bridge.Info.Name

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
			isBridgeDevice := false
			if device.Services != nil {
				for _, svc := range *device.Services {
					if svc.Rtype != nil && *svc.Rtype == "bridge" {
						isBridgeDevice = true
						break
					}
				}
			}
			if isBridgeDevice {
				continue
			}

			name := device.DeviceName("")

			// Check if device has motion sensor and its state
			hasMotion, isDetecting := state.GetDeviceMotionState(device)
			isOn := hasMotion && isDetecting

			groupSuffix := ""
			if roomName := deviceRoomNames[id]; roomName != "" {
				groupSuffix = "(" + roomName + ")"
			}

			nodes = append(nodes, &panels.TreeNode{
				ID:           id,
				Label:        name,
				GroupSuffix:  groupSuffix,
				BridgeSuffix: "[" + bridgeName + "]",
				Depth:        0,
				Item: &panels.EntityItem{
					ID:       id,
					Name:     name,
					Type:     panels.EntityDevice,
					IsOn:     isOn,
					BridgeID: bridge.Info.ID,
				},
			})
		}
	}

	// Sort by name
	sort.Slice(nodes, func(i, j int) bool {
		return nodes[i].Item.Name < nodes[j].Item.Name
	})

	m.tree.SetRoots(nodes)
}

// buildScenesTree builds the tree panel showing all Scenes from all bridges.
func (m *Model) buildScenesTree(_ *hue.BridgeState) {
	allBridges := m.manager.AllBridges()
	var nodes []*panels.TreeNode

	for _, bridge := range allBridges {
		if !bridge.IsConnected() {
			continue
		}
		state := bridge.GetState()
		if state == nil {
			continue
		}

		bridgeName := bridge.Info.Name
		scenes := state.AllScenes()
		for _, scene := range scenes {
			name := scene.SceneName("")
			id := ""
			if scene.Id != nil {
				id = *scene.Id
			}

			// Get room/zone name for the scene
			groupName := ""
			if scene.Group != nil && scene.Group.Rid != nil {
				groupID := *scene.Group.Rid
				if room, ok := state.GetRoom(groupID); ok {
					groupName = state.GetRoomName(room)
				} else if zone, ok := state.GetZone(groupID); ok {
					// Zones use RoomGet type, so use GetRoomName
					groupName = state.GetRoomName(zone)
				}
			}

			// Build suffixes
			groupSuffix := ""
			if groupName != "" {
				groupSuffix = "(" + groupName + ")"
			}

			nodes = append(nodes, &panels.TreeNode{
				ID:           id,
				Label:        name,
				GroupSuffix:  groupSuffix,
				BridgeSuffix: "[" + bridgeName + "]",
				Depth:        0,
				Item: &panels.EntityItem{
					ID:       id,
					Name:     name,
					Type:     panels.EntityScene,
					BridgeID: bridge.Info.ID,
				},
			})
		}
	}

	// Sort by name
	sort.Slice(nodes, func(i, j int) bool {
		return nodes[i].Item.Name < nodes[j].Item.Name
	})

	m.tree.SetRoots(nodes)
}

// getLightBrightnessAndColor extracts brightness and color from a light.
func getLightBrightnessAndColor(light hueclient.LightGet) (float64, string) {
	brightness := 0.0
	indicatorColor := ""

	isOn := light.On != nil && light.On.On != nil && *light.On.On
	if isOn {
		if light.Dimming != nil && light.Dimming.Brightness != nil {
			brightness = float64(*light.Dimming.Brightness)
		} else {
			brightness = 100.0 // Default to full brightness if on but no dimming
		}
		indicatorColor = ui2.GetLightColor(light)
	}

	return brightness, indicatorColor
}
