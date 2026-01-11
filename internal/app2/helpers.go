package app2

import (
	"fmt"
	"image/color"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss/v2"

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

		// Create bridge node
		bridgeNode := &panels.TreeNode{
			ID:       bridge.Info.ID,
			Label:    bridgeName,
			Depth:    0,
			Expanded: true, // Start expanded
			Item: &panels.EntityItem{
				ID:     bridge.Info.ID,
				Name:   bridgeName,
				Type:   panels.EntityBridge,
				RawPtr: bridge,
			},
			Children: make([]*panels.TreeNode, 0),
		}

		// Only add children if bridge is connected
		if bridge.IsConnected() {
			state := bridge.GetState()
			if state != nil {
				// Build rooms, zones, and entertainment areas as children
				m.buildBridgeChildren(bridgeNode, state)
			}
		}

		nodes = append(nodes, bridgeNode)
	}

	m.tree.SetRoots(nodes)
}

// buildBridgeChildren adds Rooms, Zones, and Entertainment Areas as children to a bridge node.
func (m *Model) buildBridgeChildren(bridgeNode *panels.TreeNode, state *hue.BridgeState) {
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

			roomNode := &panels.TreeNode{
				ID:       roomID,
				Label:    name,
				Depth:    2,
				Expanded: false,
				Item: &panels.EntityItem{
					ID:   roomID,
					Name: name,
					Type: panels.EntityRoom,
				},
				Children: make([]*panels.TreeNode, 0),
			}

			// Get lights, devices, and scenes for this room
			lights := state.RoomLights(room)
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
					lightsNode.Children = append(lightsNode.Children, &panels.TreeNode{
						ID:    lightID,
						Label: lightName,
						Depth: 4,
						Item: &panels.EntityItem{
							ID:   lightID,
							Name: lightName,
							Type: panels.EntityLight,
							IsOn: isOn,
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
					deviceName := hue.DeviceName(device, "")
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
							ID:   deviceID,
							Name: deviceName,
							Type: panels.EntityDevice,
							IsOn: isOn,
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
					sceneName := hue.SceneName(scene, "")
					if scene.Id != nil {
						sceneID = *scene.Id
					}
					scenesNode.Children = append(scenesNode.Children, &panels.TreeNode{
						ID:    sceneID,
						Label: sceneName,
						Depth: 4,
						Item: &panels.EntityItem{
							ID:   sceneID,
							Name: sceneName,
							Type: panels.EntityScene,
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

			zoneNode := &panels.TreeNode{
				ID:       *zone.Id,
				Label:    name,
				Depth:    2,
				Expanded: false,
				Item: &panels.EntityItem{
					ID:   *zone.Id,
					Name: name,
					Type: panels.EntityZone,
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
										zoneNode.Children = append(zoneNode.Children, &panels.TreeNode{
											ID:    *svc.Rid,
											Label: *device.Metadata.Name,
											Depth: 3,
											Item: &panels.EntityItem{
												ID:   *svc.Rid,
												Name: *device.Metadata.Name,
												Type: panels.EntityLight,
												IsOn: isOn,
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
			name := hue.EntertainmentName(ent, "Unknown")

			entItemNode := &panels.TreeNode{
				ID:       ent.ID,
				Label:    name,
				Depth:    2,
				Expanded: false,
				Item: &panels.EntityItem{
					ID:   ent.ID,
					Name: name,
					Type: panels.EntityEntertainment,
				},
				Children: make([]*panels.TreeNode, 0),
			}

			// Add lights in entertainment configuration
			for _, lightEntry := range ent.Lights {
				if lightEntry.Service != nil && lightEntry.Service.RID != "" {
					light, lightFound := state.GetLight(lightEntry.Service.RID)
					if lightFound {
						isOn := light.On != nil && light.On.On != nil && *light.On.On
						lightName := "Unknown"
						if light.Metadata != nil && light.Metadata.Name != nil {
							lightName = *light.Metadata.Name
						} else {
							// Try to get name from device
							for _, device := range state.AllDevices() {
								if device.Services != nil {
									for _, svc := range *device.Services {
										if svc.Rtype != nil && *svc.Rtype == "light" && svc.Rid != nil && *svc.Rid == lightEntry.Service.RID {
											if device.Metadata != nil && device.Metadata.Name != nil {
												lightName = *device.Metadata.Name
												break
											}
										}
									}
								}
							}
						}

						entItemNode.Children = append(entItemNode.Children, &panels.TreeNode{
							ID:    lightEntry.Service.RID,
							Label: lightName,
							Depth: 3,
							Item: &panels.EntityItem{
								ID:   lightEntry.Service.RID,
								Name: lightName,
								Type: panels.EntityLight,
								IsOn: isOn,
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

// buildLightsTree builds the tree panel showing all Lights.
func (m *Model) buildLightsTree(state *hue.BridgeState) {
	if state == nil {
		return
	}

	var nodes []*panels.TreeNode

	lights := state.AllLights()
	for _, light := range lights {
		name := "Unknown"
		if light.Metadata != nil && light.Metadata.Name != nil {
			name = *light.Metadata.Name
		}
		isOn := light.On != nil && light.On.On != nil && *light.On.On

		nodes = append(nodes, &panels.TreeNode{
			ID:    *light.Id,
			Label: name,
			Depth: 0,
			Item: &panels.EntityItem{
				ID:   *light.Id,
				Name: name,
				Type: panels.EntityLight,
				IsOn: isOn,
			},
		})
	}

	m.tree.SetRoots(nodes)
}

// buildDevicesTree builds the tree panel showing all Devices.
func (m *Model) buildDevicesTree(state *hue.BridgeState) {
	if state == nil {
		return
	}

	var nodes []*panels.TreeNode

	devices := state.AllDevices()

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
				if svc.Rtype != nil && *svc.Rtype == "bridge" {
					isBridge = true
					break
				}
			}
		}
		if isBridge {
			continue
		}

		name := hue.DeviceName(device, "")

		// Check if device has motion sensor and its state
		hasMotion, isDetecting := state.GetDeviceMotionState(device)
		isOn := hasMotion && isDetecting

		nodes = append(nodes, &panels.TreeNode{
			ID:    id,
			Label: name,
			Depth: 0,
			Item: &panels.EntityItem{
				ID:   id,
				Name: name,
				Type: panels.EntityDevice,
				IsOn: isOn,
			},
		})
	}

	m.tree.SetRoots(nodes)
}

// buildScenesTree builds the tree panel showing all Scenes.
func (m *Model) buildScenesTree(state *hue.BridgeState) {
	if state == nil {
		return
	}

	var nodes []*panels.TreeNode

	scenes := state.AllScenes()
	for _, scene := range scenes {
		name := hue.SceneName(scene, "")
		id := ""
		if scene.Id != nil {
			id = *scene.Id
		}

		nodes = append(nodes, &panels.TreeNode{
			ID:    id,
			Label: name,
			Depth: 0,
			Item: &panels.EntityItem{
				ID:   id,
				Name: name,
				Type: panels.EntityScene,
			},
		})
	}

	m.tree.SetRoots(nodes)
}

// updateDetailContent updates the detail viewport content based on the selected node.
func (m *Model) updateDetailContent() {
	node := m.tree.SelectedNode()
	if node == nil || node.Item == nil {
		m.detailViewport.SetContent("No selection")
		return
	}

	// Build content based on entity type
	var content strings.Builder
	content.WriteString(node.Item.Name)
	content.WriteString("\n")
	content.WriteString(strings.Repeat("─", 30))
	content.WriteString("\n\n")
	content.WriteString(fmt.Sprintf("Type: %s\n", node.Item.Type))
	content.WriteString(fmt.Sprintf("ID: %s\n", node.Item.ID))

	m.detailViewport.SetContent(content.String())
}

// EventDetails contains rich information about an SSE event.
type EventDetails struct {
	ResourceType   string  // "light", "scene", "motion", etc.
	ResourceName   string  // Human-readable name
	EventType      string  // "update", "add", "delete"
	Details        string  // What changed: "on", "off", "50%", "activated", "motion detected"
	IndicatorColor string  // Hex color for brightness indicator (lights)
	Brightness     float64 // Brightness 0-100 for indicator
	IsOn           bool    // Whether the light/entity is on
}

// LogEntry represents a single log entry.
type LogEntry struct {
	Time           time.Time
	Type           string  // "event", "request", "response", "error"
	Message        string  // For non-event entries
	ResourceType   string  // For events: "light", "scene", etc.
	ResourceName   string  // For events: the human-readable name
	Details        string  // For events: "on 50%", "activated", etc.
	IndicatorColor string  // Hex color for brightness indicator (lights)
	Brightness     float64 // Brightness 0-100 for indicator
	IsOn           bool    // Whether the light/entity is on
}

// buildEventDetails creates rich event details by looking up resource info from bridge state.
func (m *Model) buildEventDetails(msg bridgeEventMsg) EventDetails {
	details := EventDetails{
		ResourceType: msg.resourceType,
		EventType:    msg.eventType,
	}

	bridge := m.manager.GetBridge(msg.bridgeID)
	if bridge == nil || bridge.GetState() == nil {
		return details
	}
	state := bridge.GetState()

	switch msg.resourceType {
	case "light":
		if light, ok := state.GetLight(msg.resourceID); ok {
			// Use GetLightName to get the user-assigned name from the owning device
			details.ResourceName = state.GetLightName(light)
			// Show current state
			if light.On != nil && light.On.On != nil {
				details.IsOn = *light.On.On
				if *light.On.On {
					if light.Dimming != nil && light.Dimming.Brightness != nil {
						details.Brightness = float64(*light.Dimming.Brightness)
						details.Details = fmt.Sprintf("on %.0f%%", *light.Dimming.Brightness)
					} else {
						details.Details = "on"
					}
					// Get color for indicator
					if light.Color != nil && light.Color.Xy != nil &&
						light.Color.Xy.X != nil && light.Color.Xy.Y != nil {
						r, g, b := ui2.XyToRGB(float64(*light.Color.Xy.X), float64(*light.Color.Xy.Y), 1.0)
						details.IndicatorColor = fmt.Sprintf("#%02x%02x%02x", r, g, b)
					} else if light.ColorTemperature != nil && light.ColorTemperature.Mirek != nil {
						r, g, b := ui2.MirekToRGB(*light.ColorTemperature.Mirek)
						details.IndicatorColor = fmt.Sprintf("#%02x%02x%02x", r, g, b)
					}
				} else {
					details.Details = "off"
				}
			}
		}

	case "grouped_light":
		if gl, ok := state.GetGroupedLight(msg.resourceID); ok {
			// Try to find the room/zone name for this grouped light
			if name := state.GetGroupedLightName(msg.resourceID); name != "" {
				details.ResourceName = name
			}
			if gl.On != nil && gl.On.On != nil {
				details.IsOn = *gl.On.On
				if *gl.On.On {
					if gl.Dimming != nil && gl.Dimming.Brightness != nil {
						details.Brightness = float64(*gl.Dimming.Brightness)
						details.Details = fmt.Sprintf("on %.0f%%", *gl.Dimming.Brightness)
					} else {
						details.Details = "on"
					}
				} else {
					details.Details = "off"
				}
			}
		}

	case "scene":
		if scene, ok := state.GetScene(msg.resourceID); ok {
			details.ResourceName = hue.SceneName(scene, details.ResourceName)
			if scene.Status != nil && scene.Status.Active != nil {
				if *scene.Status.Active == "active" {
					details.Details = "activated"
				} else {
					details.Details = "deactivated"
				}
			}
		}

	case "motion":
		if motion, ok := state.GetMotion(msg.resourceID); ok {
			// Try to get device name
			if motion.Owner != nil && motion.Owner.Rid != nil {
				if device, ok := state.GetDevice(*motion.Owner.Rid); ok {
					details.ResourceName = hue.DeviceName(device, details.ResourceName)
				}
			}
			if motion.Motion != nil && motion.Motion.Motion != nil {
				if *motion.Motion.Motion {
					details.Details = "motion detected"
				} else {
					details.Details = "clear"
				}
			}
		}
	}

	return details
}

// brightnessIndicatorLog returns a character representing the brightness level.
func brightnessIndicatorLog(brightness float64) string {
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

// formatLogEntry formats a log entry with timestamps, colors, and styling like v1.
func (m *Model) formatLogEntry(entry LogEntry, width int) string {
	// Format: HH:MM:SS [TYPE] message
	timeStr := entry.Time.Format("15:04:05")

	var typeStyle lipgloss.Style
	switch entry.Type {
	case "event":
		typeStyle = lipgloss.NewStyle().Foreground(m.styles.Theme.Success)
	case "request":
		typeStyle = lipgloss.NewStyle().Foreground(m.styles.Theme.Secondary)
	case "response":
		typeStyle = lipgloss.NewStyle().Foreground(m.styles.Theme.TextMuted)
	case "error":
		typeStyle = lipgloss.NewStyle().Foreground(m.styles.Theme.Error)
	default:
		typeStyle = m.styles.Dimmed
	}

	typeIndicator := typeStyle.Render(string(entry.Type[0]))

	var line string
	if entry.Type == "event" && entry.ResourceType != "" {
		// For events, style the resource name in a faded color
		nameStyle := lipgloss.NewStyle().Foreground(m.styles.Theme.TextMuted)

		// Build brightness/color indicator for lights
		var indicator string
		if (entry.ResourceType == "light" || entry.ResourceType == "grouped_light") && entry.IsOn {
			indicatorChar := brightnessIndicatorLog(entry.Brightness)
			if entry.IndicatorColor != "" {
				indicatorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(entry.IndicatorColor))
				indicator = indicatorStyle.Render(indicatorChar) + " "
			} else {
				indicator = lipgloss.NewStyle().Foreground(m.styles.Theme.Success).Render(indicatorChar) + " "
			}
		} else if (entry.ResourceType == "light" || entry.ResourceType == "grouped_light") && !entry.IsOn {
			indicator = m.styles.Dimmed.Render("○") + " "
		}

		if entry.ResourceName != "" {
			if entry.Details != "" {
				line = fmt.Sprintf("%s %s %s%s %s → %s",
					m.styles.Dimmed.Render(timeStr),
					typeIndicator,
					indicator,
					entry.ResourceType,
					nameStyle.Render(entry.ResourceName),
					entry.Details)
			} else {
				line = fmt.Sprintf("%s %s %s%s %s",
					m.styles.Dimmed.Render(timeStr),
					typeIndicator,
					indicator,
					entry.ResourceType,
					nameStyle.Render(entry.ResourceName))
			}
		} else {
			// No resource name - show details if available
			if entry.Details != "" {
				line = fmt.Sprintf("%s %s %s%s → %s",
					m.styles.Dimmed.Render(timeStr),
					typeIndicator,
					indicator,
					entry.ResourceType,
					entry.Details)
			} else {
				line = fmt.Sprintf("%s %s %s%s update",
					m.styles.Dimmed.Render(timeStr),
					typeIndicator,
					indicator,
					entry.ResourceType)
			}
		}
	} else {
		line = fmt.Sprintf("%s %s %s", m.styles.Dimmed.Render(timeStr), typeIndicator, entry.Message)
	}

	// Truncate if needed (simple truncation - v1 has more sophisticated ANSI-aware truncation)
	if lipgloss.Width(line) > width {
		// Simple truncation - in production you'd want ANSI-aware truncation like v1
		maxLen := len(line)
		if width-1 < maxLen {
			maxLen = width - 1
		}
		line = line[:maxLen] + "…"
	}

	return line
}

// updateLogContent updates the log viewport content from logEntries.
func (m *Model) updateLogContent() {
	var content strings.Builder
	logBounds := m.layout.Bounds(PanelLog)
	contentWidth := logBounds.Width - 2 // Account for border
	if contentWidth < 1 {
		contentWidth = 1
	}

	for i, entry := range m.logEntries {
		line := m.formatLogEntry(entry, contentWidth)
		content.WriteString(line)
		if i < len(m.logEntries)-1 {
			content.WriteString("\n")
		}
	}
	m.logViewport.SetContent(content.String())
	// Scroll to bottom
	m.logViewport.LineDown(len(m.logEntries))
}

// renderDetailPanel renders the detail panel with border and header.
func (m *Model) renderDetailPanel(width, height int, focused bool, key string) string {
	borderColor := m.styles.Theme.Border
	if focused {
		borderColor = m.styles.Theme.Accent
	}

	// Get the selected item's name for the panel title
	title := "Details"
	node := m.tree.SelectedNode()
	if node != nil && node.Item != nil && node.Item.Name != "" {
		title = node.Item.Name
	}

	topBorder := m.renderPanelHeader(width, key, title, focused, borderColor)

	content := m.detailViewport.View()

	innerWidth := width - 2
	if innerWidth < 1 {
		innerWidth = 1
	}
	innerHeight := height - 2
	if innerHeight < 1 {
		innerHeight = 1
	}

	contentLines := strings.Split(content, "\n")
	if len(contentLines) > innerHeight {
		contentLines = contentLines[:innerHeight]
		content = strings.Join(contentLines, "\n")
	}

	border := lipgloss.RoundedBorder()
	borderStyleColor := lipgloss.NewStyle().Foreground(borderColor)

	leftBorder := borderStyleColor.Render(border.Left)
	rightBorder := borderStyleColor.Render(border.Right)
	bottomBorder := borderStyleColor.Render(border.BottomLeft) +
		borderStyleColor.Render(strings.Repeat(border.Bottom, innerWidth)) +
		borderStyleColor.Render(border.BottomRight)

	var lines []string
	lines = append(lines, topBorder)

	for _, line := range contentLines {
		if lipgloss.Width(line) > innerWidth {
			line = lipgloss.Place(innerWidth, 1, lipgloss.Left, lipgloss.Top, line)
		}
		paddedLine := lipgloss.Place(innerWidth, 1, lipgloss.Left, lipgloss.Top, line)
		lines = append(lines, leftBorder+paddedLine+rightBorder)
	}

	for len(lines) < height-1 {
		paddedLine := strings.Repeat(" ", innerWidth)
		lines = append(lines, leftBorder+paddedLine+rightBorder)
	}

	lines = append(lines, bottomBorder)

	return strings.Join(lines, "\n")
}

// renderLogPanel renders the log panel with border and header.
func (m *Model) renderLogPanel(width, height int, focused bool, key string) string {
	borderColor := m.styles.Theme.Border
	if focused {
		borderColor = m.styles.Theme.Accent
	}

	topBorder := m.renderPanelHeader(width, key, "Activity", focused, borderColor)

	innerWidth := width - 2
	if innerWidth < 1 {
		innerWidth = 1
	}
	innerHeight := height - 2
	if innerHeight < 1 {
		innerHeight = 1
	}

	content := m.logViewport.View()
	// Split content - viewport may add trailing newline
	contentLines := strings.Split(content, "\n")
	// Remove trailing empty line if present
	if len(contentLines) > 0 && contentLines[len(contentLines)-1] == "" {
		contentLines = contentLines[:len(contentLines)-1]
	}
	// Viewport should return exactly innerHeight lines, but cap it just in case
	if len(contentLines) > innerHeight {
		contentLines = contentLines[:innerHeight]
	}

	border := lipgloss.RoundedBorder()
	borderStyleColor := lipgloss.NewStyle().Foreground(borderColor)

	leftBorder := borderStyleColor.Render(border.Left)
	rightBorder := borderStyleColor.Render(border.Right)
	bottomBorder := borderStyleColor.Render(border.BottomLeft) +
		borderStyleColor.Render(strings.Repeat(border.Bottom, innerWidth)) +
		borderStyleColor.Render(border.BottomRight)

	var lines []string
	lines = append(lines, topBorder)

	// Render exactly the lines the viewport provides (should be innerHeight)
	// Limit to innerHeight to prevent overflow
	for i := 0; i < len(contentLines) && i < innerHeight; i++ {
		line := contentLines[i]
		if lipgloss.Width(line) > innerWidth {
			line = lipgloss.Place(innerWidth, 1, lipgloss.Left, lipgloss.Top, line)
		}
		paddedLine := lipgloss.Place(innerWidth, 1, lipgloss.Left, lipgloss.Top, line)
		lines = append(lines, leftBorder+paddedLine+rightBorder)
	}

	lines = append(lines, bottomBorder)

	return strings.Join(lines, "\n")
}

// renderPanelHeader renders a panel header with hotkey and title.
func (m *Model) renderPanelHeader(width int, keyStr, title string, focused bool, borderColor color.Color) string {
	border := lipgloss.RoundedBorder()

	keyRendered := ""
	keyWidth := 0
	if keyStr != "" {
		keyStyle := lipgloss.NewStyle().
			Foreground(m.styles.Theme.Primary).
			Bold(true)
		keyRendered = keyStyle.Render("[" + keyStr + "]")
		keyWidth = lipgloss.Width(keyRendered)
	}

	titleStyle := lipgloss.NewStyle().
		Foreground(m.styles.Theme.Primary).
		Bold(true)
	titleRendered := titleStyle.Render(title)
	titleWidth := lipgloss.Width(titleRendered)

	leftPadding := 1
	middlePadding := 1
	remainingWidth := width - keyWidth - titleWidth - leftPadding - middlePadding - 2

	if remainingWidth < 0 {
		remainingWidth = 0
	}

	borderStyle := lipgloss.NewStyle().Foreground(borderColor)

	if keyRendered != "" {
		return borderStyle.Render(border.TopLeft) +
			borderStyle.Render(strings.Repeat(border.Top, leftPadding)) +
			keyRendered +
			borderStyle.Render(strings.Repeat(border.Top, middlePadding)) +
			titleRendered +
			borderStyle.Render(strings.Repeat(border.Top, remainingWidth)) +
			borderStyle.Render(border.TopRight)
	}

	return borderStyle.Render(border.TopLeft) +
		borderStyle.Render(strings.Repeat(border.Top, leftPadding)) +
		titleRendered +
		borderStyle.Render(strings.Repeat(border.Top, remainingWidth+middlePadding)) +
		borderStyle.Render(border.TopRight)
}
