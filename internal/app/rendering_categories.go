package app

import (
	"fmt"

	"github.com/charmbracelet/lipgloss/v2"
	"github.com/kluzzebass/lazyhue/internal/hue"
	"github.com/kluzzebass/lazyhue/internal/ui"
	"github.com/kluzzebass/lazyhue/internal/ui/component/field"
	gridlayout "github.com/kluzzebass/lazyhue/internal/ui/component/layout"
	"github.com/kluzzebass/lazyhue/internal/ui/panels"
)

// buildRoomsCategoryGridRows builds a list of all rooms in a category.
func (m *Model) buildRoomsCategoryGridRows(data panels.RoomsCategoryData, state *hue.BridgeState) []gridlayout.GridRow {
	var rows []gridlayout.GridRow

	// Header with count
	roomsHeaderText := fmt.Sprintf("Rooms %s%d%s", m.styles.Dimmed.Render("["), len(data.Rooms), m.styles.Dimmed.Render("]"))
	header := field.NewHeaderComponent("rooms-header", roomsHeaderText, &m.styles, m.zones)
	rows = append(rows, gridlayout.GridRow{
		Type:    gridlayout.RowTypeSection,
		Section: header,
	})

	// Count on/off rooms
	onCount := 0
	for _, room := range data.Rooms {
		lights := state.RoomLights(room)
		for _, light := range lights {
			if panels.IsLightOn(light) {
				onCount++
				break
			}
		}
	}

	rows = append(rows, gridlayout.NewInfoRow("Active", fmt.Sprintf("%d of %d", onCount, len(data.Rooms)), infoLabelWidth))
	rows = append(rows, gridlayout.NewEmptyRow())

	// List each room
	roomStyle := lipgloss.NewStyle().Foreground(m.styles.Theme.EntityRoom)
	for _, room := range data.Rooms {
		name := "Unknown"
		if room.Metadata.Name != "" {
			name = room.Metadata.Name
		}

		// Get room's lights and calculate state
		lights := state.RoomLights(room)
		brightness, indicatorColor := hue.CalculateRoomAggregate(lights)

		// Build indicator
		indicator := m.styles.Dimmed.Render("○")
		if brightness > 0 {
			indicator = ui.RenderBrightnessIndicatorFromHex(brightness, indicatorColor)
		}

		rows = append(rows, gridlayout.NewListItemRow(indicator, roomStyle.Render(name)))
	}

	return rows
}

// buildZonesCategoryGridRows builds a list of all zones in a category.
func (m *Model) buildZonesCategoryGridRows(data panels.ZonesCategoryData, state *hue.BridgeState) []gridlayout.GridRow {
	var rows []gridlayout.GridRow

	// Header with count
	zonesHeaderText := fmt.Sprintf("Zones %s%d%s", m.styles.Dimmed.Render("["), len(data.Zones), m.styles.Dimmed.Render("]"))
	header := field.NewHeaderComponent("zones-header", zonesHeaderText, &m.styles, m.zones)
	rows = append(rows, gridlayout.GridRow{
		Type:    gridlayout.RowTypeSection,
		Section: header,
	})

	// Count on/off zones
	onCount := 0
	for _, zone := range data.Zones {
		lights := state.ZoneLights(zone)
		for _, light := range lights {
			if panels.IsLightOn(light) {
				onCount++
				break
			}
		}
	}

	rows = append(rows, gridlayout.NewInfoRow("Active", fmt.Sprintf("%d of %d", onCount, len(data.Zones)), infoLabelWidth))
	rows = append(rows, gridlayout.NewEmptyRow())

	// List each zone
	zoneStyle := lipgloss.NewStyle().Foreground(m.styles.Theme.EntityZone)
	for _, zone := range data.Zones {
		name := "Unknown"
		if zone.Metadata.Name != "" {
			name = zone.Metadata.Name
		}

		// Get zone's lights and calculate state
		lights := state.ZoneLights(zone)
		brightness, indicatorColor := hue.CalculateRoomAggregate(lights)

		// Build indicator
		indicator := m.styles.Dimmed.Render("○")
		if brightness > 0 {
			indicator = ui.RenderBrightnessIndicatorFromHex(brightness, indicatorColor)
		}

		rows = append(rows, gridlayout.NewListItemRow(indicator, zoneStyle.Render(name)))
	}

	return rows
}

// buildLightsCategoryGridRows builds a list of all lights in a category.
func (m *Model) buildLightsCategoryGridRows(data panels.LightsCategoryData, _ *hue.BridgeState) []gridlayout.GridRow {
	var rows []gridlayout.GridRow

	// Header with parent context
	headerText := fmt.Sprintf("Lights %s%d%s", m.styles.Dimmed.Render("["), len(data.Lights), m.styles.Dimmed.Render("]"))
	if data.ParentName != "" {
		headerText = fmt.Sprintf("Lights in %s %s%d%s", data.ParentName, m.styles.Dimmed.Render("["), len(data.Lights), m.styles.Dimmed.Render("]"))
	}
	header := field.NewHeaderComponent("lights-header", headerText, &m.styles, m.zones)
	rows = append(rows, gridlayout.GridRow{
		Type:    gridlayout.RowTypeSection,
		Section: header,
	})

	// Count on/off lights
	onCount := 0
	for _, light := range data.Lights {
		if panels.IsLightOn(light) {
			onCount++
		}
	}

	rows = append(rows, gridlayout.NewInfoRow("On", fmt.Sprintf("%d of %d", onCount, len(data.Lights)), infoLabelWidth))
	rows = append(rows, gridlayout.NewEmptyRow())

	// List each light
	for _, light := range data.Lights {
		name := "Unknown"
		if light.Metadata.Name != "" {
			name = light.Metadata.Name
		}

		isOn := panels.IsLightOn(light)
		brightness := 0.0
		indicatorColor := ""
		if isOn {
			if light.Dimming != nil {
				brightness = float64(light.Dimming.Brightness)
			} else {
				brightness = 100.0
			}
			indicatorColor = ui.GetLightColor(light)
		}

		// Build indicator
		indicator := m.styles.Dimmed.Render("○")
		if brightness > 0 {
			indicator = ui.RenderBrightnessIndicatorFromHex(brightness, indicatorColor)
		}

		catLightStyle := lipgloss.NewStyle().Foreground(m.styles.Theme.EntityLight)
		rows = append(rows, gridlayout.NewListItemRow(indicator, catLightStyle.Render(name)))
	}

	return rows
}

// buildDevicesCategoryGridRows builds a list of all devices in a category.
func (m *Model) buildDevicesCategoryGridRows(data panels.DevicesCategoryData, state *hue.BridgeState) []gridlayout.GridRow {
	var rows []gridlayout.GridRow

	// Header with parent context
	headerText := fmt.Sprintf("Devices %s%d%s", m.styles.Dimmed.Render("["), len(data.Devices), m.styles.Dimmed.Render("]"))
	if data.ParentName != "" {
		headerText = fmt.Sprintf("Devices in %s %s%d%s", data.ParentName, m.styles.Dimmed.Render("["), len(data.Devices), m.styles.Dimmed.Render("]"))
	}
	header := field.NewHeaderComponent("devices-header", headerText, &m.styles, m.zones)
	rows = append(rows, gridlayout.GridRow{
		Type:    gridlayout.RowTypeSection,
		Section: header,
	})

	rows = append(rows, gridlayout.NewInfoRow("Total", fmt.Sprintf("%d", len(data.Devices)), infoLabelWidth))
	rows = append(rows, gridlayout.NewEmptyRow())

	// List each device
	catDeviceStyle := lipgloss.NewStyle().Foreground(m.styles.Theme.EntityDevice)
	for _, device := range data.Devices {
		name := device.DeviceName("Unknown")

		// Check if device has motion sensor and its state
		hasMotion, isDetecting := state.GetDeviceMotionState(device)

		// Build indicator based on motion state
		indicator := m.styles.Dimmed.Render("○")
		if hasMotion && isDetecting {
			indicator = lipgloss.NewStyle().Foreground(m.styles.Theme.Warning).Render("●")
		}

		rows = append(rows, gridlayout.NewListItemRow(indicator, catDeviceStyle.Render(name)))
	}

	return rows
}

// buildScenesCategoryGridRows builds a list of all scenes in a category.
func (m *Model) buildScenesCategoryGridRows(data panels.ScenesCategoryData, _ *hue.BridgeState) []gridlayout.GridRow {
	var rows []gridlayout.GridRow

	// Header with parent context
	headerText := fmt.Sprintf("Scenes %s%d%s", m.styles.Dimmed.Render("["), len(data.Scenes), m.styles.Dimmed.Render("]"))
	if data.ParentName != "" {
		headerText = fmt.Sprintf("Scenes in %s %s%d%s", data.ParentName, m.styles.Dimmed.Render("["), len(data.Scenes), m.styles.Dimmed.Render("]"))
	}
	header := field.NewHeaderComponent("scenes-header", headerText, &m.styles, m.zones)
	rows = append(rows, gridlayout.GridRow{
		Type:    gridlayout.RowTypeSection,
		Section: header,
	})

	rows = append(rows, gridlayout.NewInfoRow("Total", fmt.Sprintf("%d", len(data.Scenes)), infoLabelWidth))
	rows = append(rows, gridlayout.NewEmptyRow())

	// List each scene
	catSceneStyle := lipgloss.NewStyle().Foreground(m.styles.Theme.EntityScene)
	for _, scene := range data.Scenes {
		name := scene.SceneName("Unknown")
		rows = append(rows, gridlayout.NewListItemRow("•", catSceneStyle.Render(name)))
	}

	return rows
}

// buildSmartScenesCategoryGridRows builds a list of all smart scenes in a category.
func (m *Model) buildSmartScenesCategoryGridRows(data panels.SmartScenesCategoryData, _ *hue.BridgeState) []gridlayout.GridRow {
	var rows []gridlayout.GridRow

	// Header with parent context
	headerText := fmt.Sprintf("Smart Scenes %s%d%s", m.styles.Dimmed.Render("["), len(data.SmartScenes), m.styles.Dimmed.Render("]"))
	if data.ParentName != "" {
		headerText = fmt.Sprintf("Smart Scenes on %s %s%d%s", data.ParentName, m.styles.Dimmed.Render("["), len(data.SmartScenes), m.styles.Dimmed.Render("]"))
	}
	header := field.NewHeaderComponent("smartscenes-header", headerText, &m.styles, m.zones)
	rows = append(rows, gridlayout.GridRow{
		Type:    gridlayout.RowTypeSection,
		Section: header,
	})

	rows = append(rows, gridlayout.NewInfoRow("Total", fmt.Sprintf("%d", len(data.SmartScenes)), infoLabelWidth))
	rows = append(rows, gridlayout.NewEmptyRow())

	// List each smart scene with status
	smartSceneStyle := lipgloss.NewStyle().Foreground(m.styles.Theme.EntityScene)
	for _, scene := range data.SmartScenes {
		name := "Unknown"
		if scene.Metadata.Name != "" {
			name = scene.Metadata.Name
		}
		statusIndicator := m.styles.Dimmed.Render("○") // inactive
		if scene.State == "active" {
			statusIndicator = m.styles.Success.Render("●") // active
		}
		rows = append(rows, gridlayout.NewListItemRow(statusIndicator, smartSceneStyle.Render(name)))
	}

	return rows
}

// buildEntertainmentCategoryGridRows builds a list of all entertainment configurations in a category.
func (m *Model) buildEntertainmentCategoryGridRows(data panels.EntertainmentCategoryData) []gridlayout.GridRow {
	var rows []gridlayout.GridRow

	// Header with count
	entHeaderText := fmt.Sprintf("Entertainment Areas %s%d%s", m.styles.Dimmed.Render("["), len(data.Configurations), m.styles.Dimmed.Render("]"))
	header := field.NewHeaderComponent("entertainment-header", entHeaderText, &m.styles, m.zones)
	rows = append(rows, gridlayout.GridRow{
		Type:    gridlayout.RowTypeSection,
		Section: header,
	})

	rows = append(rows, gridlayout.NewInfoRow("Total", fmt.Sprintf("%d", len(data.Configurations)), infoLabelWidth))
	rows = append(rows, gridlayout.NewEmptyRow())

	// List each entertainment configuration
	entStyle := lipgloss.NewStyle().Foreground(m.styles.Theme.EntityEntertainment)
	for _, cfg := range data.Configurations {
		rows = append(rows, gridlayout.NewListItemRow("•", entStyle.Render(cfg.Name)))
	}

	return rows
}
