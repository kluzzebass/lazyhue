package app

import (
	"fmt"
	"sort"
	"time"

	"github.com/kluzzebass/lazyhue/internal/hue"
	"github.com/kluzzebass/lazyhue/internal/hueclient"
	"github.com/kluzzebass/lazyhue/internal/ui"
	"github.com/kluzzebass/lazyhue/internal/ui/panels"
)

// nodeID creates a unique tree node ID by combining bridge ID and entity ID.
// This ensures click zones don't conflict across bridges.
func nodeID(bridgeID, entityID string) string {
	return bridgeID + ":" + entityID
}

// getBridgeDisplayName returns the best available display name for a bridge.
// It prefers the device metadata name (updated via API) over the discovery name.
func getBridgeDisplayName(bridge *hue.Bridge, state *hue.BridgeState) string {
	// Try to get name from bridge device metadata (updated via API)
	if state != nil {
		if bridgeDevice, ok := state.GetBridgeDevice(); ok {
			if deviceName := state.GetDeviceName(bridgeDevice); deviceName != "Unknown" {
				return deviceName
			}
		}
	}
	// Fall back to discovery name
	if bridge.Info.Name != "" {
		return bridge.Info.Name
	}
	// Last resort: use bridge ID
	return bridge.Info.ID
}

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
		state := bridge.GetState()
		bridgeName := getBridgeDisplayName(bridge, state)

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
			Item: &panels.EntityItem{
				ID:       bridgeNode.ID + ":rooms",
				Name:     "Rooms",
				Type:     panels.EntityRoomsCategory,
				BridgeID: bridgeID,
				RawPtr: panels.RoomsCategoryData{
					BridgeID: bridgeID,
					Rooms:    rooms,
				},
			},
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
				ID:       nodeID(bridgeID, roomID),
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
					ID:       nodeID(bridgeID, roomID+":lights"),
					Label:    fmt.Sprintf("Lights (%d)", len(lights)),
					Depth:    3,
					Expanded: false,
					Item: &panels.EntityItem{
						ID:             roomID + ":lights",
						Name:           fmt.Sprintf("Lights in %s", name),
						Type:           panels.EntityLightsCategory,
						IsOn:           brightness > 0,
						Brightness:     brightness,
						IndicatorColor: indicatorColor,
						BridgeID:       bridgeID,
						RawPtr: panels.LightsCategoryData{
							ParentName: name,
							Lights:     lights,
						},
					},
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
						ID:    nodeID(bridgeID, lightID),
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
					ID:       nodeID(bridgeID, roomID+":devices"),
					Label:    fmt.Sprintf("Devices (%d)", len(nonLightDevices)),
					Depth:    3,
					Expanded: false,
					Item: &panels.EntityItem{
						ID:       roomID + ":devices",
						Name:     fmt.Sprintf("Devices in %s", name),
						Type:     panels.EntityDevicesCategory,
						BridgeID: bridgeID,
						RawPtr: panels.DevicesCategoryData{
							ParentName: name,
							Devices:    nonLightDevices,
						},
					},
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
						ID:    nodeID(bridgeID, deviceID),
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
					ID:       nodeID(bridgeID, roomID+":scenes"),
					Label:    fmt.Sprintf("Scenes (%d)", len(scenes)),
					Depth:    3,
					Expanded: false,
					Item: &panels.EntityItem{
						ID:       roomID + ":scenes",
						Name:     fmt.Sprintf("Scenes in %s", name),
						Type:     panels.EntityScenesCategory,
						BridgeID: bridgeID,
						RawPtr: panels.ScenesCategoryData{
							ParentName: name,
							Scenes:     scenes,
						},
					},
					Children: make([]*panels.TreeNode, 0, len(scenes)),
				}
				for _, scene := range scenes {
					sceneID := ""
					sceneName := scene.SceneName("")
					if scene.Id != nil {
						sceneID = *scene.Id
					}
					scenesNode.Children = append(scenesNode.Children, &panels.TreeNode{
						ID:    nodeID(bridgeID, sceneID),
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

			// Smart scenes subsection
			smartScenes := state.RoomSmartScenes(roomID)
			if len(smartScenes) > 0 {
				smartScenesNode := &panels.TreeNode{
					ID:       nodeID(bridgeID, roomID+":smart_scenes"),
					Label:    fmt.Sprintf("Smart Scenes (%d)", len(smartScenes)),
					Depth:    3,
					Expanded: false,
					Item: &panels.EntityItem{
						ID:       roomID + ":smart_scenes",
						Name:     fmt.Sprintf("Smart Scenes in %s", name),
						Type:     panels.EntitySmartScenesCategory,
						BridgeID: bridgeID,
						RawPtr: panels.SmartScenesCategoryData{
							ParentName:  name,
							SmartScenes: smartScenes,
						},
					},
					Children: make([]*panels.TreeNode, 0, len(smartScenes)),
				}
				for _, scene := range smartScenes {
					sceneID := ""
					sceneName := state.GetSmartSceneName(scene)
					if scene.Id != nil {
						sceneID = *scene.Id
					}
					isActive := scene.State == "active"
					smartScenesNode.Children = append(smartScenesNode.Children, &panels.TreeNode{
						ID:    nodeID(bridgeID, sceneID),
						Label: sceneName,
						Depth: 4,
						Item: &panels.EntityItem{
							ID:       sceneID,
							Name:     sceneName,
							Type:     panels.EntitySmartScene,
							IsOn:     isActive,
							BridgeID: bridgeID,
						},
					})
				}
				roomNode.Children = append(roomNode.Children, smartScenesNode)
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
			Item: &panels.EntityItem{
				ID:       bridgeNode.ID + ":zones",
				Name:     "Zones",
				Type:     panels.EntityZonesCategory,
				BridgeID: bridgeID,
				RawPtr: panels.ZonesCategoryData{
					BridgeID: bridgeID,
					Zones:    zones,
				},
			},
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
			lights := state.ZoneLights(zone)

			// Calculate aggregate brightness and color from zone lights
			brightness, indicatorColor := hue.CalculateRoomAggregate(lights)

			zoneNode := &panels.TreeNode{
				ID:       nodeID(bridgeID, zoneID),
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

			// Lights subsection for zone (like rooms have)
			if len(lights) > 0 {
				lightsNode := &panels.TreeNode{
					ID:       nodeID(bridgeID, zoneID+":lights"),
					Label:    fmt.Sprintf("Lights (%d)", len(lights)),
					Depth:    3,
					Expanded: false,
					Item: &panels.EntityItem{
						ID:             zoneID + ":lights",
						Name:           fmt.Sprintf("Lights in %s", name),
						Type:           panels.EntityLightsCategory,
						IsOn:           brightness > 0,
						Brightness:     brightness,
						IndicatorColor: indicatorColor,
						BridgeID:       bridgeID,
						RawPtr: panels.LightsCategoryData{
							ParentName: name,
							Lights:     lights,
						},
					},
					Children: make([]*panels.TreeNode, 0, len(lights)),
				}
				for _, light := range lights {
					lightID := ""
					if light.Id != nil {
						lightID = *light.Id
					}
					lightName := state.GetLightName(light)
					isOn := light.On != nil && light.On.On != nil && *light.On.On
					lightBrightness, lightIndicatorColor := getLightBrightnessAndColor(light)
					// Store a copy of the light in RawPtr for later access
					lightCopy := light
					lightsNode.Children = append(lightsNode.Children, &panels.TreeNode{
						ID:    nodeID(bridgeID, lightID),
						Label: lightName,
						Depth: 4,
						Item: &panels.EntityItem{
							ID:             lightID,
							Name:           lightName,
							Type:           panels.EntityLight,
							IsOn:           isOn,
							Brightness:     lightBrightness,
							IndicatorColor: lightIndicatorColor,
							RawPtr:         lightCopy,
							BridgeID:       bridgeID,
						},
					})
				}
				zoneNode.Children = append(zoneNode.Children, lightsNode)
			}

			// Scenes subsection for zone
			zoneScenes := state.ZoneScenes(zoneID)
			if len(zoneScenes) > 0 {
				scenesNode := &panels.TreeNode{
					ID:       nodeID(bridgeID, zoneID+":scenes"),
					Label:    fmt.Sprintf("Scenes (%d)", len(zoneScenes)),
					Depth:    3,
					Expanded: false,
					Item: &panels.EntityItem{
						ID:       zoneID + ":scenes",
						Name:     fmt.Sprintf("Scenes in %s", name),
						Type:     panels.EntityScenesCategory,
						BridgeID: bridgeID,
						RawPtr: panels.ScenesCategoryData{
							ParentName: name,
							Scenes:     zoneScenes,
						},
					},
					Children: make([]*panels.TreeNode, 0, len(zoneScenes)),
				}
				for _, scene := range zoneScenes {
					sceneID := ""
					sceneName := scene.SceneName("")
					if scene.Id != nil {
						sceneID = *scene.Id
					}
					scenesNode.Children = append(scenesNode.Children, &panels.TreeNode{
						ID:    nodeID(bridgeID, sceneID),
						Label: sceneName,
						Depth: 4,
						Item: &panels.EntityItem{
							ID:       sceneID,
							Name:     sceneName,
							Type:     panels.EntityZoneScene,
							BridgeID: bridgeID,
						},
					})
				}
				zoneNode.Children = append(zoneNode.Children, scenesNode)
			}

			// Smart scenes subsection for zone
			zoneSmartScenes := state.ZoneSmartScenes(zoneID)
			if len(zoneSmartScenes) > 0 {
				smartScenesNode := &panels.TreeNode{
					ID:       nodeID(bridgeID, zoneID+":smart_scenes"),
					Label:    fmt.Sprintf("Smart Scenes (%d)", len(zoneSmartScenes)),
					Depth:    3,
					Expanded: false,
					Item: &panels.EntityItem{
						ID:       zoneID + ":smart_scenes",
						Name:     fmt.Sprintf("Smart Scenes in %s", name),
						Type:     panels.EntitySmartScenesCategory,
						BridgeID: bridgeID,
						RawPtr: panels.SmartScenesCategoryData{
							ParentName:  name,
							SmartScenes: zoneSmartScenes,
						},
					},
					Children: make([]*panels.TreeNode, 0, len(zoneSmartScenes)),
				}
				for _, scene := range zoneSmartScenes {
					sceneID := ""
					sceneName := state.GetSmartSceneName(scene)
					if scene.Id != nil {
						sceneID = *scene.Id
					}
					isActive := scene.State == "active"
					smartScenesNode.Children = append(smartScenesNode.Children, &panels.TreeNode{
						ID:    nodeID(bridgeID, sceneID),
						Label: sceneName,
						Depth: 4,
						Item: &panels.EntityItem{
							ID:       sceneID,
							Name:     sceneName,
							Type:     panels.EntityZoneSmartScene,
							IsOn:     isActive,
							BridgeID: bridgeID,
						},
					})
				}
				zoneNode.Children = append(zoneNode.Children, smartScenesNode)
			}

			zonesNode.Children = append(zonesNode.Children, zoneNode)
		}
		bridgeNode.Children = append(bridgeNode.Children, zonesNode)
	}

	// Entertainment configurations category
	entConfigs := state.AllEntertainmentConfigurations()
	if len(entConfigs) > 0 {
		// Build simplified config list for category data
		var configList []panels.EntertainmentConfig
		for _, ent := range entConfigs {
			configList = append(configList, panels.EntertainmentConfig{
				ID:   ent.ID,
				Name: ent.EntertainmentName("Unknown"),
			})
		}

		entNode := &panels.TreeNode{
			ID:       bridgeNode.ID + ":entertainment",
			Label:    fmt.Sprintf("Entertainment Areas (%d)", len(entConfigs)),
			Depth:    1,
			Expanded: false,
			Item: &panels.EntityItem{
				ID:       bridgeNode.ID + ":entertainment",
				Name:     "Entertainment Areas",
				Type:     panels.EntityEntertainmentCategory,
				BridgeID: bridgeID,
				RawPtr: panels.EntertainmentCategoryData{
					BridgeID:       bridgeID,
					Configurations: configList,
				},
			},
			Children: make([]*panels.TreeNode, 0, len(entConfigs)),
		}

		for _, ent := range entConfigs {
			name := ent.EntertainmentName("Unknown")

			entItemNode := &panels.TreeNode{
				ID:       nodeID(bridgeID, ent.ID),
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
							ID:    nodeID(bridgeID, lightEntry.Service.RID),
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

// buildLightsTree builds the tree panel showing Lights grouped by bridge.
func (m *Model) buildLightsTree(_ *hue.BridgeState) {
	allBridges := m.manager.AllBridges()
	var bridgeNodes []*panels.TreeNode

	// Sort bridges by name for consistent ordering
	sort.Slice(allBridges, func(i, j int) bool {
		return allBridges[i].Info.Name < allBridges[j].Info.Name
	})

	for _, bridge := range allBridges {
		if !bridge.IsConnected() {
			continue
		}
		state := bridge.GetState()
		if state == nil {
			continue
		}

		bridgeName := getBridgeDisplayName(bridge, state)
		lights := state.AllLights()
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

		// Calculate aggregate brightness/color for bridge
		bridgeBrightness, bridgeIndicatorColor := hue.CalculateRoomAggregate(lights)

		// Create bridge node
		bridgeNode := &panels.TreeNode{
			ID:       bridge.Info.ID,
			Label:    fmt.Sprintf("%s (%d)", bridgeName, len(lights)),
			Depth:    0,
			Expanded: true,
			Item: &panels.EntityItem{
				ID:             bridge.Info.ID,
				Name:           bridgeName,
				Type:           panels.EntityBridge,
				IsOn:           bridge.IsConnected(),
				Brightness:     bridgeBrightness,
				IndicatorColor: bridgeIndicatorColor,
				RawPtr:         bridge,
				BridgeID:       bridge.Info.ID,
			},
			Children: make([]*panels.TreeNode, 0, len(lights)),
		}

		// Sort lights by name
		sort.Slice(lights, func(i, j int) bool {
			return state.GetLightName(lights[i]) < state.GetLightName(lights[j])
		})

		for _, light := range lights {
			name := state.GetLightName(light)
			isOn := light.On != nil && light.On.On != nil && *light.On.On
			brightness, indicatorColor := getLightBrightnessAndColor(light)

			// Get room name via owning device
			groupSuffix := ""
			if light.Owner != nil && light.Owner.Rid != nil {
				if roomName := deviceRoomNames[*light.Owner.Rid]; roomName != "" {
					groupSuffix = "(" + roomName + ")"
				}
			}

			// Store a copy of the light in RawPtr for later access
			lightCopy := light
			bridgeNode.Children = append(bridgeNode.Children, &panels.TreeNode{
				ID:          nodeID(bridge.Info.ID, *light.Id),
				Label:       name,
				GroupSuffix: groupSuffix,
				Depth:       1,
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

		bridgeNodes = append(bridgeNodes, bridgeNode)
	}

	m.tree.SetRoots(bridgeNodes)
}

// buildDevicesTree builds the tree panel showing Devices grouped by bridge.
func (m *Model) buildDevicesTree(_ *hue.BridgeState) {
	allBridges := m.manager.AllBridges()
	var bridgeNodes []*panels.TreeNode

	// Sort bridges by name for consistent ordering
	sort.Slice(allBridges, func(i, j int) bool {
		return allBridges[i].Info.Name < allBridges[j].Info.Name
	})

	for _, bridge := range allBridges {
		if !bridge.IsConnected() {
			continue
		}
		state := bridge.GetState()
		if state == nil {
			continue
		}

		bridgeName := getBridgeDisplayName(bridge, state)
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

		// Filter and collect non-light devices
		var nonLightDevices []hueclient.DeviceGet
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

			nonLightDevices = append(nonLightDevices, device)
		}

		// Sort devices by name
		sort.Slice(nonLightDevices, func(i, j int) bool {
			return nonLightDevices[i].DeviceName("") < nonLightDevices[j].DeviceName("")
		})

		// Create bridge node
		bridgeNode := &panels.TreeNode{
			ID:       bridge.Info.ID,
			Label:    fmt.Sprintf("%s (%d)", bridgeName, len(nonLightDevices)),
			Depth:    0,
			Expanded: true,
			Item: &panels.EntityItem{
				ID:       bridge.Info.ID,
				Name:     bridgeName,
				Type:     panels.EntityBridge,
				IsOn:     bridge.IsConnected(),
				RawPtr:   bridge,
				BridgeID: bridge.Info.ID,
			},
			Children: make([]*panels.TreeNode, 0, len(nonLightDevices)),
		}

		for _, device := range nonLightDevices {
			id := ""
			if device.Id != nil {
				id = *device.Id
			}

			name := device.DeviceName("")

			// Check if device has motion sensor and its state
			hasMotion, isDetecting := state.GetDeviceMotionState(device)
			isOn := hasMotion && isDetecting

			groupSuffix := ""
			if roomName := deviceRoomNames[id]; roomName != "" {
				groupSuffix = "(" + roomName + ")"
			}

			bridgeNode.Children = append(bridgeNode.Children, &panels.TreeNode{
				ID:          nodeID(bridge.Info.ID, id),
				Label:       name,
				GroupSuffix: groupSuffix,
				Depth:       1,
				Item: &panels.EntityItem{
					ID:       id,
					Name:     name,
					Type:     panels.EntityDevice,
					IsOn:     isOn,
					BridgeID: bridge.Info.ID,
				},
			})
		}

		bridgeNodes = append(bridgeNodes, bridgeNode)
	}

	m.tree.SetRoots(bridgeNodes)
}

// buildScenesTree builds the tree panel showing all Scenes and Smart Scenes from all bridges.
func (m *Model) buildScenesTree(_ *hue.BridgeState) {
	allBridges := m.manager.AllBridges()
	var bridgeNodes []*panels.TreeNode

	for _, bridge := range allBridges {
		if !bridge.IsConnected() {
			continue
		}
		state := bridge.GetState()
		if state == nil {
			continue
		}

		bridgeName := getBridgeDisplayName(bridge, state)
		scenes := state.AllScenes()
		smartScenes := state.AllSmartScenes()
		totalScenes := len(scenes) + len(smartScenes)

		// Create bridge node
		bridgeNode := &panels.TreeNode{
			ID:       bridge.Info.ID,
			Label:    fmt.Sprintf("%s (%d)", bridgeName, totalScenes),
			Depth:    0,
			Expanded: true,
			Item: &panels.EntityItem{
				ID:       bridge.Info.ID,
				Name:     bridgeName,
				Type:     panels.EntityBridge,
				IsOn:     bridge.IsConnected(),
				RawPtr:   bridge,
				BridgeID: bridge.Info.ID,
			},
			Children: make([]*panels.TreeNode, 0, totalScenes),
		}

		// Collect all scene nodes for sorting
		type sceneNode struct {
			name string
			node *panels.TreeNode
		}
		var sceneNodes []sceneNode

		// Regular scenes
		for _, scene := range scenes {
			name := scene.SceneName("")
			id := ""
			if scene.Id != nil {
				id = *scene.Id
			}

			// Get room/zone name for the scene and track the type
			groupName := ""
			groupType := panels.EntityRoom // Default to room
			if scene.Group != nil && scene.Group.Rid != nil {
				groupID := *scene.Group.Rid
				if room, ok := state.GetRoom(groupID); ok {
					groupName = state.GetRoomName(room)
					groupType = panels.EntityRoom
				} else if zone, ok := state.GetZone(groupID); ok {
					groupName = state.GetRoomName(zone)
					groupType = panels.EntityZone
				}
			}

			groupSuffix := ""
			if groupName != "" {
				groupSuffix = "(" + groupName + ")"
			}

			// Use zone scene type if it belongs to a zone
			sceneType := panels.EntityScene
			if groupType == panels.EntityZone {
				sceneType = panels.EntityZoneScene
			}

			sceneNodes = append(sceneNodes, sceneNode{
				name: name,
				node: &panels.TreeNode{
					ID:          nodeID(bridge.Info.ID, id),
					Label:       name,
					GroupSuffix: groupSuffix,
					GroupType:   groupType,
					Depth:       1,
					Item: &panels.EntityItem{
						ID:       id,
						Name:     name,
						Type:     sceneType,
						BridgeID: bridge.Info.ID,
					},
				},
			})
		}

		// Smart scenes
		for _, scene := range smartScenes {
			name := state.GetSmartSceneName(scene)
			id := ""
			if scene.Id != nil {
				id = *scene.Id
			}

			// Get room/zone name for the smart scene and track the type
			groupName := ""
			groupType := panels.EntityRoom // Default to room
			if scene.Group.Rid != nil {
				groupID := *scene.Group.Rid
				if room, ok := state.GetRoom(groupID); ok {
					groupName = state.GetRoomName(room)
					groupType = panels.EntityRoom
				} else if zone, ok := state.GetZone(groupID); ok {
					groupName = state.GetRoomName(zone)
					groupType = panels.EntityZone
				}
			}

			groupSuffix := ""
			if groupName != "" {
				groupSuffix = "(" + groupName + ")"
			}

			isActive := scene.State == "active"

			// Use zone smart scene type if it belongs to a zone
			smartSceneType := panels.EntitySmartScene
			if groupType == panels.EntityZone {
				smartSceneType = panels.EntityZoneSmartScene
			}

			sceneNodes = append(sceneNodes, sceneNode{
				name: name,
				node: &panels.TreeNode{
					ID:          nodeID(bridge.Info.ID, id),
					Label:       name,
					GroupSuffix: groupSuffix,
					GroupType:   groupType,
					Depth:       1,
					Item: &panels.EntityItem{
						ID:       id,
						Name:     name,
						Type:     smartSceneType,
						IsOn:     isActive,
						BridgeID: bridge.Info.ID,
					},
				},
			})
		}

		// Sort scenes by name within this bridge
		sort.Slice(sceneNodes, func(i, j int) bool {
			return sceneNodes[i].name < sceneNodes[j].name
		})

		// Add sorted scenes to bridge node
		for _, sn := range sceneNodes {
			bridgeNode.Children = append(bridgeNode.Children, sn.node)
		}

		bridgeNodes = append(bridgeNodes, bridgeNode)
	}

	m.tree.SetRoots(bridgeNodes)
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
		indicatorColor = ui.GetLightColor(light)
	}

	return brightness, indicatorColor
}
