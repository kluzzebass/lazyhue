package app

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss/v2"
	"github.com/kluzzebass/lazyhue/internal/hue"
	"github.com/kluzzebass/lazyhue/internal/hueclient"
	"github.com/kluzzebass/lazyhue/internal/ui/component/field"
	gridlayout "github.com/kluzzebass/lazyhue/internal/ui/component/layout"
	"github.com/kluzzebass/lazyhue/internal/ui/panels"
)

// buildRoomGridRows builds grid rows for a room or zone details panel.
// Sections are ordered by immediacy: Controls > Settings > Status > Lists > IDs
func (m *Model) buildRoomGridRows(room hueclient.RoomGet, isZone bool, state *hue.BridgeState) []gridlayout.GridRow {
	var rows []gridlayout.GridRow

	if state == nil {
		return rows
	}

	roomID := room.Id

	var lights []hueclient.LightGet
	if isZone {
		// For zones, we need to convert back to ZoneGet to get the lights
		zone := hueclient.ZoneGet{
			Children: room.Children,
			Id:       room.Id,
			IdV1:     room.IdV1,
			Metadata: room.Metadata,
			Services: room.Services,
		}
		lights = state.ZoneLights(zone)
	} else {
		lights = state.RoomLights(room)
	}

	// 1. Controls section - instant adjustments (grouped light power, brightness)
	if gl, ok := state.RoomGroupedLight(room); ok {
		controlsHeader := field.NewHeaderComponent("controls-header", "Controls", &m.styles, m.zones)
		rows = append(rows, gridlayout.GridRow{
			Type:    gridlayout.RowTypeSection,
			Section: controlsHeader,
		})

		// Use different field ID prefix for rooms vs zones
		glID := gl.Id

		// Power toggle
		var powerFieldID string
		if isZone {
			powerFieldID = FieldIDZonePower(glID)
		} else {
			powerFieldID = FieldIDRoomPower(glID)
		}
		onValue := gl.On != nil && gl.On.On
		toggle := field.NewToggleComponent(powerFieldID, "Power", onValue, &m.styles, m.zones)
		toggle.SetLabels("On", "Off")
		rows = append(rows, gridlayout.GridRow{
			Type: gridlayout.RowTypeNormal,
			Cells: []gridlayout.GridCell{
				{Component: gridlayout.NewLabelWithWidth("Power", infoLabelWidth)},
				{Component: toggle},
			},
		})

		// Brightness slider
		if gl.Dimming != nil {
			var brightnessFieldID string
			if isZone {
				brightnessFieldID = FieldIDZoneBrightness(glID)
			} else {
				brightnessFieldID = FieldIDRoomBrightness(glID)
			}
			brightness := int(gl.Dimming.Brightness)
			slider := field.NewBrightnessSliderComponent(
				brightnessFieldID, "Brightness", brightness,
				&m.styles, m.zones,
			)
			rows = append(rows, gridlayout.GridRow{
				Type: gridlayout.RowTypeNormal,
				Cells: []gridlayout.GridCell{
					{Component: gridlayout.NewLabelWithWidth("Brightness", infoLabelWidth)},
					{Component: slider},
				},
			})
		}

		// Create Scene button (captures current light states)
		var createSceneFieldID string
		if isZone {
			createSceneFieldID = FieldIDZoneCreateScene(roomID)
		} else {
			createSceneFieldID = FieldIDRoomCreateScene(roomID)
		}
		createSceneBtn := field.NewButtonComponent(
			createSceneFieldID, "Create Scene", "Create Scene",
			&m.styles, m.zones,
		)
		rows = append(rows, gridlayout.GridRow{
			Type: gridlayout.RowTypeNormal,
			Cells: []gridlayout.GridCell{
				{Component: gridlayout.NewLabelWithWidth("Scene", infoLabelWidth)},
				{Component: createSceneBtn},
			},
		})
	}

	// 2. Settings section (name, archetype)
	settingsHeader := field.NewHeaderComponent("settings-header", "Settings", &m.styles, m.zones)
	rows = append(rows, gridlayout.GridRow{
		Type:    gridlayout.RowTypeSection,
		Section: settingsHeader,
	})

	if room.Metadata.Name != "" {
		// Use different field ID for rooms vs zones
		var nameFieldID string
		if isZone {
			nameFieldID = FieldIDZoneName(roomID)
		} else {
			nameFieldID = FieldIDRoomName(roomID)
		}

		textInput := field.NewTextComponent(
			nameFieldID, "Name", room.Metadata.Name,
			&m.styles, m.zones,
		)
		rows = append(rows, gridlayout.GridRow{
			Type: gridlayout.RowTypeNormal,
			Cells: []gridlayout.GridCell{
				{Component: gridlayout.NewLabelWithWidth("Name", infoLabelWidth)},
				{Component: textInput},
			},
		})
	}

	// Archetype (editable)
	if room.Metadata.Archetype != "" {
		archetypeKeys := hue.RoomArchetypeList()
		var options []field.Option
		currentIndex := 0
		currentArchetype := string(room.Metadata.Archetype)

		for i, key := range archetypeKeys {
			displayName := hue.RoomArchetypeDisplayNames[key]
			options = append(options, field.Option{Label: displayName, Value: i})
			if key == currentArchetype {
				currentIndex = i
			}
		}

		// Use different field ID for rooms vs zones
		var archetypeFieldID string
		if isZone {
			archetypeFieldID = FieldIDZoneArchetype(roomID)
		} else {
			archetypeFieldID = FieldIDRoomArchetype(roomID)
		}

		selectComp := field.NewSelectComponent(
			archetypeFieldID, "Archetype", currentIndex, options,
			&m.styles, m.zones,
		)
		rows = append(rows, gridlayout.GridRow{
			Type: gridlayout.RowTypeNormal,
			Cells: []gridlayout.GridCell{
				{Component: gridlayout.NewLabelWithWidth("Archetype", infoLabelWidth)},
				{Component: selectComp},
			},
		})
	}

	// 3. Status section
	statusHeader := field.NewHeaderComponent("status-header", "Status", &m.styles, m.zones)
	rows = append(rows, gridlayout.GridRow{
		Type:    gridlayout.RowTypeSection,
		Section: statusHeader,
	})

	onCount := 0
	for _, l := range lights {
		if panels.IsLightOn(l) {
			onCount++
		}
	}
	rows = append(rows, gridlayout.NewInfoRow("Lights", fmt.Sprintf("%d/%d on", onCount, len(lights)), infoLabelWidth))

	// Motion sensor status - check devices in the room for motion sensors
	if len(room.Children) > 0 {
		var motionSensors int
		var motionDetected bool
		for _, child := range room.Children {
			if child.Rid == "" || child.Rtype != hueclient.ResourceTypeDevice {
				continue
			}
			device, ok := state.GetDevice(child.Rid)
			if !ok {
				continue
			}
			if hasMotion, detecting := state.GetDeviceMotionState(device); hasMotion {
				motionSensors++
				if detecting {
					motionDetected = true
				}
			}
		}
		if motionSensors > 0 {
			motionStyle := lipgloss.NewStyle()
			var motionStatus string
			if motionDetected {
				motionStyle = motionStyle.Foreground(m.styles.Theme.Warning)
				motionStatus = motionStyle.Render("● motion detected")
			} else {
				motionStyle = motionStyle.Foreground(m.styles.Theme.TextMuted)
				motionStatus = motionStyle.Render("○ clear")
			}
			if motionSensors > 1 {
				motionStatus += m.styles.Dimmed.Render(fmt.Sprintf(" (%d sensors)", motionSensors))
			}
			rows = append(rows, gridlayout.NewInfoRow("Motion", motionStatus, infoLabelWidth))
		}
	}

	// Lights section
	if len(lights) > 0 {
		lightsHeaderText := fmt.Sprintf("Lights %s%d%s", m.styles.Dimmed.Render("["), len(lights), m.styles.Dimmed.Render("]"))
		lightsHeader := field.NewHeaderComponent("lights-header", lightsHeaderText, &m.styles, m.zones)
		rows = append(rows, gridlayout.GridRow{
			Type:    gridlayout.RowTypeSection,
			Section: lightsHeader,
		})

		lightStyle := lipgloss.NewStyle().Foreground(m.styles.Theme.EntityLight)
		for _, light := range lights {
			name := state.GetLightName(light)
			isOn := panels.IsLightOn(light)
			indicator := m.renderLightIndicator(light, isOn)

			var detailParts []string
			if isOn {
				if light.Dimming != nil {
					detailParts = append(detailParts, fmt.Sprintf("%.0f%%", light.Dimming.Brightness))
				}
				if light.ColorTemperature != nil && light.ColorTemperature.Mirek != 0 {
					mirek := light.ColorTemperature.Mirek
					kelvin := 1000000 / mirek
					detailParts = append(detailParts, fmt.Sprintf("%dK", kelvin))
				}
			} else {
				detailParts = append(detailParts, "off")
			}

			suffix := ""
			if len(detailParts) > 0 {
				suffix = " " + m.styles.Dimmed.Render(strings.Join(detailParts, ", "))
			}
			rows = append(rows, gridlayout.NewListItemRow(indicator, lightStyle.Render(name)+suffix))
		}
	}

	// Non-light devices
	if len(room.Children) > 0 {
		var nonLightDevices []hueclient.DeviceGet
		for _, child := range room.Children {
			if child.Rid == "" || child.Rtype != hueclient.ResourceTypeDevice {
				continue
			}
			device, ok := state.GetDevice(child.Rid)
			if !ok {
				continue
			}
			isLight := false
			for _, svc := range device.Services {
				if svc.Rtype == hueclient.ResourceTypeLight {
					isLight = true
					break
				}
			}
			if isLight {
				continue
			}
			nonLightDevices = append(nonLightDevices, device)
		}

		if len(nonLightDevices) > 0 {
			devicesHeaderText := fmt.Sprintf("Devices %s%d%s", m.styles.Dimmed.Render("["), len(nonLightDevices), m.styles.Dimmed.Render("]"))
			devicesHeader := field.NewHeaderComponent("devices-header", devicesHeaderText, &m.styles, m.zones)
			rows = append(rows, gridlayout.GridRow{
				Type:    gridlayout.RowTypeSection,
				Section: devicesHeader,
			})
			deviceStyle := lipgloss.NewStyle().Foreground(m.styles.Theme.EntityDevice)
			for _, device := range nonLightDevices {
				name := device.DeviceName("")
				indicator := "•"
				suffix := ""

				// Check for motion sensor
				if hasMotion, detecting := state.GetDeviceMotionState(device); hasMotion {
					if detecting {
						indicator = lipgloss.NewStyle().Foreground(m.styles.Theme.Warning).Render("●")
						suffix = " " + m.styles.Dimmed.Render("motion")
					} else {
						indicator = lipgloss.NewStyle().Foreground(m.styles.Theme.TextMuted).Render("○")
						suffix = " " + m.styles.Dimmed.Render("clear")
					}
				}

				rows = append(rows, gridlayout.NewListItemRow(indicator, deviceStyle.Render(name)+suffix))
			}
		}
	}

	// Scenes
	scenes := state.RoomScenes(roomID)
	if len(scenes) > 0 {
		scenesHeaderText := fmt.Sprintf("Scenes %s%d%s", m.styles.Dimmed.Render("["), len(scenes), m.styles.Dimmed.Render("]"))
		scenesHeader := field.NewHeaderComponent("scenes-header", scenesHeaderText, &m.styles, m.zones)
		rows = append(rows, gridlayout.GridRow{
			Type:    gridlayout.RowTypeSection,
			Section: scenesHeader,
		})
		sceneStyle := lipgloss.NewStyle().Foreground(m.styles.Theme.EntityScene)
		for _, scene := range scenes {
			rows = append(rows, gridlayout.NewListItemRow("•", sceneStyle.Render(scene.SceneName(""))))
		}
	}

	// IDs section
	idsHeader := field.NewHeaderComponent("ids-header", "IDs", &m.styles, m.zones)
	rows = append(rows, gridlayout.GridRow{
		Type:    gridlayout.RowTypeSection,
		Section: idsHeader,
	})

	dimStyle := m.styles.Dimmed
	if room.Id != "" {
		rows = append(rows, gridlayout.NewStyledInfoRow("ID", room.Id, infoLabelWidth, dimStyle))
	}
	if room.IdV1 != nil && *room.IdV1 != "" {
		rows = append(rows, gridlayout.NewStyledInfoRow("V1 ID", *room.IdV1, infoLabelWidth, dimStyle))
	}
	typeStr := "Room"
	if isZone {
		typeStr = "Zone"
	}
	rows = append(rows, gridlayout.NewStyledInfoRow("Type", typeStr, infoLabelWidth, dimStyle))

	return rows
}
