package app

import (
	"fmt"

	"github.com/kluzzebass/lazyhue/internal/hue"
	"github.com/kluzzebass/lazyhue/internal/hueclient"
	"github.com/kluzzebass/lazyhue/internal/ui"
	"github.com/kluzzebass/lazyhue/internal/ui/component/field"
	gridlayout "github.com/kluzzebass/lazyhue/internal/ui/component/layout"
)

// infoLabelWidth is the width for labels in info rows.
const infoLabelWidth = 14

// getDeviceForLight returns the device that owns the given light.
func (m *Model) getDeviceForLight(light hueclient.LightGet) *hueclient.DeviceGet {
	if light.Owner.Rid == "" {
		return nil
	}
	node := m.tree.SelectedNode()
	if node == nil || node.Item == nil {
		return nil
	}
	bridge := m.manager.GetBridge(node.Item.BridgeID)
	if bridge == nil {
		return nil
	}
	state := bridge.GetState()
	if state == nil {
		return nil
	}
	if device, ok := state.GetDevice(light.Owner.Rid); ok {
		return &device
	}
	return nil
}

// renderLightIndicator renders a brightness indicator for a light.
func (m *Model) renderLightIndicator(light hueclient.LightGet, isOn bool) string {
	if !isOn {
		// Return dimmed off indicator
		return m.styles.Dimmed.Render(ui.BrightnessIndicator(0))
	}

	brightness := 100.0
	if light.Dimming != nil {
		brightness = float64(light.Dimming.Brightness)
	}

	hexColor := ui.GetLightColor(light)
	return ui.RenderBrightnessIndicatorFromHex(brightness, hexColor)
}

// buildLightGridRows builds grid rows for a light's full details panel.
// This uses the Grid layout with sections for product info, controls, state, etc.
// Sections are ordered by immediacy: Controls > Settings > Info > Technical
func (m *Model) buildLightGridRows(light hueclient.LightGet) {
	var rows []gridlayout.GridRow

	// Get owning device for product info
	device := m.getDeviceForLight(light)

	// === CONTROLS (interactive, at the top) ===

	// 1. Controls section - instant adjustments (power, brightness, color, effects, identify)
	rows = append(rows, m.buildControlsRows(light)...)

	// 2. Timed Effects - interactive triggers (sunrise/sunset)
	rows = append(rows, m.buildTimedEffectsListRows(light)...)

	// 3. Signaling - interactive triggers (alerts)
	rows = append(rows, m.buildSignalingRows(light)...)

	// 4. Gradient section - interactive color gradient editing (if supported)
	lightID := light.Id
	rows = append(rows, m.buildGradientRows(light, lightID)...)

	// === SETTINGS (editable configuration) ===

	// 5. Settings section - persistent configuration (name, power-on behavior)
	rows = append(rows, m.buildLightSettingsRows(light, device)...)

	// === STATE & INFO (read-only, lower priority) ===

	// 6. State Info section - current readings
	rows = append(rows, m.buildStateRows(light)...)

	// 7. Dynamics section
	rows = append(rows, m.buildDynamicsRows(light)...)

	// 8. Capabilities section
	rows = append(rows, m.buildCapabilitiesRows(light)...)

	// 9. Product Info section
	rows = append(rows, m.buildProductInfoRows(device)...)

	// 10. Classification section
	rows = append(rows, m.buildClassificationRows(light, device)...)

	// 11. Effects list section (reference info, not interactive)
	rows = append(rows, m.buildEffectsListRows(light)...)

	// 12. Device Services section
	rows = append(rows, m.buildDeviceServicesRows(light, device)...)

	// 13. IDs section - least urgent, at the end
	rows = append(rows, m.buildIDsRows(light)...)

	m.lightGrid.SetRows(rows)
}

// buildProductInfoRows builds rows for the product info section.
func (m *Model) buildProductInfoRows(device *hueclient.DeviceGet) []gridlayout.GridRow {
	if device == nil {
		return nil
	}

	var rows []gridlayout.GridRow
	pd := device.ProductData

	// Section header
	header := field.NewHeaderComponent("product-header", "Product", &m.styles, m.zones)
	rows = append(rows, gridlayout.GridRow{
		Type:    gridlayout.RowTypeSection,
		Section: header,
	})

	if pd.ProductName != "" {
		rows = append(rows, gridlayout.NewInfoRow("Product", pd.ProductName, infoLabelWidth))
	}
	if pd.ManufacturerName != "" {
		rows = append(rows, gridlayout.NewInfoRow("Manufacturer", pd.ManufacturerName, infoLabelWidth))
	}
	if pd.ModelId != "" {
		rows = append(rows, gridlayout.NewInfoRow("Model", pd.ModelId, infoLabelWidth))
	}
	if pd.SoftwareVersion != "" {
		rows = append(rows, gridlayout.NewInfoRow("Firmware", pd.SoftwareVersion, infoLabelWidth))
	}
	if pd.HardwarePlatformType != nil {
		rows = append(rows, gridlayout.NewInfoRow("Hardware", *pd.HardwarePlatformType, infoLabelWidth))
	}

	return rows
}

// buildClassificationRows builds rows for the classification section.
func (m *Model) buildClassificationRows(light hueclient.LightGet, device *hueclient.DeviceGet) []gridlayout.GridRow {
	var rows []gridlayout.GridRow

	// Section header
	header := field.NewHeaderComponent("class-header", "Classification", &m.styles, m.zones)
	rows = append(rows, gridlayout.GridRow{
		Type:    gridlayout.RowTypeSection,
		Section: header,
	})

	// Type (read-only)
	if light.Type != "" {
		rows = append(rows, gridlayout.NewInfoRow("Type", string(light.Type), infoLabelWidth))
	}

	// Mode (read-only)
	if light.Mode != "" {
		rows = append(rows, gridlayout.NewInfoRow("Mode", string(light.Mode), infoLabelWidth))
	}

	// Alternate name (read-only) - shows light's own name if different from device name
	if light.Metadata.Name != "" {
		altName := light.Metadata.Name
		currentName := ""
		if device != nil && device.Metadata.Name != "" {
			currentName = device.Metadata.Name
		}
		if altName != currentName {
			rows = append(rows, gridlayout.NewInfoRow("Alternate name", altName, infoLabelWidth))
		}
	}

	return rows
}

// buildLightSettingsRows builds rows for the light settings section.
// Only includes editable fields: name, archetype, power-on behavior.
func (m *Model) buildLightSettingsRows(light hueclient.LightGet, device *hueclient.DeviceGet) []gridlayout.GridRow {
	var rows []gridlayout.GridRow

	// Section header
	header := field.NewHeaderComponent("settings-header", "Settings", &m.styles, m.zones)
	rows = append(rows, gridlayout.GridRow{
		Type:    gridlayout.RowTypeSection,
		Section: header,
	})

	// Device name (editable)
	deviceID := ""
	if device != nil && device.Id != "" {
		deviceID = device.Id
	}

	if device != nil && device.Metadata.Name != "" {
		textInput := field.NewTextComponent(
			FieldIDName(deviceID), "Name", device.Metadata.Name,
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
	if device != nil && device.ProductData.ProductArchetype != "" {
		deviceID := device.Id

		archetypeKeys := getSortedProductArchetypes()
		var options []field.Option
		currentIndex := 0
		currentArchetype := string(device.ProductData.ProductArchetype)

		for i, key := range archetypeKeys {
			displayName := hue.ProductArchetypeDisplayNames[key]
			options = append(options, field.Option{Label: displayName, Value: i})
			if key == currentArchetype {
				currentIndex = i
			}
		}

		selectComp := field.NewSelectComponent(
			FieldIDArchetype(deviceID), "Archetype", currentIndex, options,
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

	// Powerup preset (editable)
	if light.Powerup != nil && light.Powerup.Preset != "" {
		presetKeys := getSortedPowerupPresets()
		var options []field.Option
		currentIndex := 0
		currentPreset := string(light.Powerup.Preset)

		for i, key := range presetKeys {
			displayName := hue.PowerupPresetDisplayNames[key]
			options = append(options, field.Option{Label: displayName, Value: i})
			if key == currentPreset {
				currentIndex = i
			}
		}

		lightID := light.Id

		selectComp := field.NewSelectComponent(
			FieldIDPowerupPreset(lightID), "Power-on", currentIndex, options,
			&m.styles, m.zones,
		)
		rows = append(rows, gridlayout.GridRow{
			Type: gridlayout.RowTypeNormal,
			Cells: []gridlayout.GridCell{
				{Component: gridlayout.NewLabelWithWidth("Power-on", infoLabelWidth)},
				{Component: selectComp},
			},
		})
	}

	// Room assignment dropdown (lights move with their device)
	if deviceID != "" {
		// Get state from the selected bridge
		node := m.tree.SelectedNode()
		if node != nil && node.Item != nil {
			bridge := m.manager.GetBridge(node.Item.BridgeID)
			if bridge != nil {
				state := bridge.GetState()
				if state != nil {
					currentRoom, hasRoom := state.GetDeviceRoom(deviceID)
					currentRoomID := ""
					if hasRoom && currentRoom.Id != "" {
						currentRoomID = currentRoom.Id
					}

					// Build room options: "No Room" + all rooms
					allRooms := state.AllRooms()
					options := make([]field.Option, 0, len(allRooms)+1)
					options = append(options, field.Option{Label: "No Room", Value: 0})
					selectedIndex := 0

					for i, room := range allRooms {
						roomName := "Unknown"
						roomID := ""
						if room.Metadata.Name != "" {
							roomName = room.Metadata.Name
						}
						if room.Id != "" {
							roomID = room.Id
						}
						options = append(options, field.Option{Label: roomName, Value: i + 1, Color: m.styles.Theme.EntityRoom})
						if roomID == currentRoomID {
							selectedIndex = i + 1
						}
					}

					roomSelect := field.NewSelectComponent(
						"light-room:"+deviceID, "Room", selectedIndex, options,
						&m.styles, m.zones,
					)
					rows = append(rows, gridlayout.GridRow{
						Type: gridlayout.RowTypeNormal,
						Cells: []gridlayout.GridCell{
							{Component: gridlayout.NewLabelWithWidth("Room", infoLabelWidth)},
							{Component: roomSelect},
						},
					})
				}
			}
		}
	}

	// Zone membership (checkboxes)
	if light.Id != "" {
		zoneNode := m.tree.SelectedNode()
		if zoneNode != nil && zoneNode.Item != nil {
			zoneBridge := m.manager.GetBridge(zoneNode.Item.BridgeID)
			if zoneBridge != nil {
				zoneState := zoneBridge.GetState()
				if zoneState != nil {
					lightID := light.Id
					allZones := zoneState.AllZones()
					if len(allZones) > 0 {
						// Build a set of zone IDs this light is in
						lightZones := zoneState.GetLightZones(lightID)
						lightZoneIDs := make(map[string]bool)
						for _, zone := range lightZones {
							if zone.Id != "" {
								lightZoneIDs[zone.Id] = true
							}
						}

						// Build options and selected map for checkbox component
						var options []field.Option
						selected := make(map[int]bool)
						for i, z := range allZones {
							zoneID := ""
							zoneName := "Unknown"
							if z.Id != "" {
								zoneID = z.Id
							}
							if z.Metadata.Name != "" {
								zoneName = z.Metadata.Name
							}
							options = append(options, field.Option{Label: zoneName, Value: i, Color: m.styles.Theme.EntityZone})
							if lightZoneIDs[zoneID] {
								selected[i] = true
							}
						}

						// Add checkbox group for zones
						zoneCheckbox := field.NewCheckboxComponent(
							"light-zones:"+lightID, "Zones", selected, options, true,
							&m.styles, m.zones,
						)
						rows = append(rows, gridlayout.GridRow{
							Type: gridlayout.RowTypeNormal,
							Cells: []gridlayout.GridCell{
								{Component: gridlayout.NewLabelWithWidth("Zones", infoLabelWidth)},
								{Component: zoneCheckbox},
							},
						})
					}
				}
			}
		}
	}

	return rows
}

// buildIDsRows builds rows for the IDs section.
func (m *Model) buildIDsRows(light hueclient.LightGet) []gridlayout.GridRow {
	var rows []gridlayout.GridRow

	// Section header
	header := field.NewHeaderComponent("ids-header", "IDs", &m.styles, m.zones)
	rows = append(rows, gridlayout.GridRow{
		Type:    gridlayout.RowTypeSection,
		Section: header,
	})

	dimStyle := m.styles.Dimmed

	if light.Id != "" {
		rows = append(rows, gridlayout.NewStyledInfoRow("Light ID", light.Id, infoLabelWidth, dimStyle))
	}
	if light.Owner.Rid != "" {
		rows = append(rows, gridlayout.NewStyledInfoRow("Device ID", light.Owner.Rid, infoLabelWidth, dimStyle))
	}
	if light.IdV1 != nil {
		rows = append(rows, gridlayout.NewStyledInfoRow("V1 ID", *light.IdV1, infoLabelWidth, dimStyle))
	}

	return rows
}

// buildControlsRows builds rows for the controls section (existing functionality).
func (m *Model) buildControlsRows(light hueclient.LightGet) []gridlayout.GridRow {
	var rows []gridlayout.GridRow

	// Section header
	header := field.NewHeaderComponent("controls-header", "Controls", &m.styles, m.zones)
	rows = append(rows, gridlayout.GridRow{
		Type:    gridlayout.RowTypeSection,
		Section: header,
	})

	// On/Off toggle
	onValue := light.On.On
	toggle := field.NewToggleComponent("on", "Power", onValue, &m.styles, m.zones)
	toggle.SetLabels("On", "Off")
	rows = append(rows, gridlayout.GridRow{
		Type: gridlayout.RowTypeNormal,
		Cells: []gridlayout.GridCell{
			{Component: gridlayout.NewLabelWithWidth("Power", infoLabelWidth)},
			{Component: toggle},
		},
	})

	// Brightness (if dimmable)
	if light.Dimming != nil {
		brightness := int(light.Dimming.Brightness)
		slider := field.NewBrightnessSliderComponent(
			"brightness", "Brightness", brightness,
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

	// Color temperature (if supported)
	if light.ColorTemperature != nil {
		mirek := light.ColorTemperature.Mirek
		if mirek == 0 {
			mirek = 250 // default if not set
		}
		minMirek := light.ColorTemperature.MirekSchema.MirekMinimum
		maxMirek := light.ColorTemperature.MirekSchema.MirekMaximum
		if minMirek == 0 {
			minMirek = 153
		}
		if maxMirek == 0 {
			maxMirek = 500
		}
		slider := field.NewColorTempSliderComponent(
			"colortemp", "Color Temp", mirek, minMirek, maxMirek,
			&m.styles, m.zones,
		)
		rows = append(rows, gridlayout.GridRow{
			Type: gridlayout.RowTypeNormal,
			Cells: []gridlayout.GridCell{
				{Component: gridlayout.NewLabelWithWidth("Color Temp", infoLabelWidth)},
				{Component: slider},
			},
		})
	}

	// Color (if supported)
	if light.Color != nil {
		x := float64(light.Color.Xy.X)
		y := float64(light.Color.Xy.Y)
		if x == 0 && y == 0 {
			x, y = 0.3127, 0.329 // default white point
		}
		colorWheel := field.NewColorWheelComponent(
			"color", "Color", x, y,
			&m.styles, m.zones,
		)
		rows = append(rows, gridlayout.GridRow{
			Type: gridlayout.RowTypeNormal,
			Cells: []gridlayout.GridCell{
				{Component: gridlayout.NewLabelWithWidth("Color", infoLabelWidth)},
				{Component: colorWheel},
			},
		})
	}

	// Effect (if supported)
	if light.Effects != nil && len(light.Effects.EffectValues) > 0 {
		var options []field.Option
		currentEffect := -1
		for i, effect := range light.Effects.EffectValues {
			effectStr := string(effect)
			displayName := hue.EffectDisplayName(effectStr)
			options = append(options, field.Option{
				Label: displayName,
				Value: i,
			})
			if hueclient.SupportedSounds(light.Effects.Status) == effect {
				currentEffect = i
			}
		}
		if len(options) > 0 {
			if currentEffect == -1 {
				currentEffect = 0
			}
			selectComp := field.NewSelectComponent(
				"effect", "Effect", currentEffect, options,
				&m.styles, m.zones,
			)
			rows = append(rows, gridlayout.GridRow{
				Type: gridlayout.RowTypeNormal,
				Cells: []gridlayout.GridCell{
					{Component: gridlayout.NewLabelWithWidth("Effect", infoLabelWidth)},
					{Component: selectComp},
				},
			})
		}
	}

	// EffectsV2 speed control (when effect is active)
	if light.EffectsV2 != nil && light.EffectsV2.Status.Effect != "" && light.EffectsV2.Status.Effect != "no_effect" {
		speed := 50 // Default 50%
		if light.EffectsV2.Status.Parameters != nil {
			speed = int(light.EffectsV2.Status.Parameters.Speed * 100)
		}
		speedSlider := field.NewSliderComponent(
			"effect-speed:"+light.Id, "Speed", speed, 0, 100, 1,
			&m.styles, m.zones,
		)
		rows = append(rows, gridlayout.GridRow{
			Type: gridlayout.RowTypeNormal,
			Cells: []gridlayout.GridCell{
				{Component: gridlayout.NewLabelWithWidth("Effect Speed", infoLabelWidth)},
				{Component: speedSlider},
			},
		})
	}

	// Identify button (uses device ID from light owner)
	if light.Owner.Rid != "" {
		identifyBtn := field.NewButtonComponent(
			"identify:"+light.Owner.Rid, "Identify", "Identify",
			&m.styles, m.zones,
		)
		rows = append(rows, gridlayout.GridRow{
			Type: gridlayout.RowTypeNormal,
			Cells: []gridlayout.GridCell{
				{Component: gridlayout.NewLabelWithWidth("Identify", infoLabelWidth)},
				{Component: identifyBtn},
			},
		})
	}

	return rows
}

// buildStateRows builds rows for technical state info not shown in controls.
// Only includes details that add value beyond what the controls already display.
func (m *Model) buildStateRows(light hueclient.LightGet) []gridlayout.GridRow {
	var rows []gridlayout.GridRow

	// Collect items that provide info beyond the controls
	var hasContent bool

	// Min dim level (not shown in brightness slider)
	if light.Dimming != nil && light.Dimming.MinDimLevel != nil {
		hasContent = true
	}

	// Gamut type (not shown in color wheel)
	if light.Color != nil && light.Color.GamutType != "" {
		hasContent = true
	}

	// CT range (not shown in color temp slider)
	if light.ColorTemperature != nil {
		schema := light.ColorTemperature.MirekSchema
		if schema.MirekMinimum != 0 && schema.MirekMaximum != 0 {
			hasContent = true
		}
	}

	// Only add section if we have content
	if !hasContent {
		return rows
	}

	// Section header
	header := field.NewHeaderComponent("state-header", "Limits", &m.styles, m.zones)
	rows = append(rows, gridlayout.GridRow{
		Type:    gridlayout.RowTypeSection,
		Section: header,
	})

	// Min dim level
	if light.Dimming != nil && light.Dimming.MinDimLevel != nil {
		rows = append(rows, gridlayout.NewInfoRow("Min Dim", fmt.Sprintf("%.0f%%", *light.Dimming.MinDimLevel), infoLabelWidth))
	}

	// Gamut type
	if light.Color != nil && light.Color.GamutType != "" {
		rows = append(rows, gridlayout.NewInfoRow("Gamut", string(light.Color.GamutType), infoLabelWidth))
	}

	// CT range
	if light.ColorTemperature != nil {
		schema := light.ColorTemperature.MirekSchema
		if schema.MirekMinimum != 0 && schema.MirekMaximum != 0 {
			minK := 1000000 / schema.MirekMaximum
			maxK := 1000000 / schema.MirekMinimum
			rows = append(rows, gridlayout.NewInfoRow("CT Range", fmt.Sprintf("%dK - %dK", minK, maxK), infoLabelWidth))
		}
	}

	return rows
}

// buildDynamicsRows builds rows for the dynamics section.
func (m *Model) buildDynamicsRows(light hueclient.LightGet) []gridlayout.GridRow {
	if light.Dynamics == nil {
		return nil
	}

	var rows []gridlayout.GridRow

	// Section header
	header := field.NewHeaderComponent("dynamics-header", "Dynamics", &m.styles, m.zones)
	rows = append(rows, gridlayout.GridRow{
		Type:    gridlayout.RowTypeSection,
		Section: header,
	})

	if light.Dynamics.Status != "" {
		rows = append(rows, gridlayout.NewInfoRow("Status", string(light.Dynamics.Status), infoLabelWidth))
	}
	if light.Dynamics.SpeedValid {
		rows = append(rows, gridlayout.NewInfoRow("Speed", fmt.Sprintf("%.2f", light.Dynamics.Speed), infoLabelWidth))
	}

	return rows
}

// buildCapabilitiesRows builds rows for the capabilities section.
func (m *Model) buildCapabilitiesRows(light hueclient.LightGet) []gridlayout.GridRow {
	var rows []gridlayout.GridRow

	// Section header
	header := field.NewHeaderComponent("caps-header", "Capabilities", &m.styles, m.zones)
	rows = append(rows, gridlayout.GridRow{
		Type:    gridlayout.RowTypeSection,
		Section: header,
	})

	// Color Gamut Type (if color capable)
	if light.Color != nil {
		gamutType := string(light.Color.GamutType)
		gamutDisplay := gamutType
		switch gamutType {
		case "A":
			gamutDisplay = "A (early Philips)"
		case "B":
			gamutDisplay = "B (first gen Hue)"
		case "C":
			gamutDisplay = "C (wide gamut)"
		case "other":
			gamutDisplay = "Other (non-Hue)"
		}
		rows = append(rows, gridlayout.NewInfoRow("Color Gamut", gamutDisplay, infoLabelWidth))
	}

	// Collect capabilities
	var caps []string
	if light.Dimming != nil {
		caps = append(caps, "Dimming")
	}
	if light.Color != nil {
		caps = append(caps, "Color")
	}
	if light.ColorTemperature != nil {
		caps = append(caps, "Color Temperature")
	}
	if light.Gradient != nil {
		caps = append(caps, "Gradient")
	}
	if light.Effects != nil {
		caps = append(caps, "Effects")
	}
	if light.TimedEffects != nil {
		caps = append(caps, "Timed Effects")
	}

	if len(caps) == 0 {
		caps = append(caps, "On/Off only")
	}

	for _, cap := range caps {
		rows = append(rows, gridlayout.NewListItemRow("•", cap))
	}

	return rows
}

// buildEffectsListRows builds rows for the available effects section.
func (m *Model) buildEffectsListRows(light hueclient.LightGet) []gridlayout.GridRow {
	if light.Effects == nil || len(light.Effects.EffectValues) == 0 {
		return nil
	}

	var rows []gridlayout.GridRow

	// Section header
	header := field.NewHeaderComponent("effects-header", "Available Effects", &m.styles, m.zones)
	rows = append(rows, gridlayout.GridRow{
		Type:    gridlayout.RowTypeSection,
		Section: header,
	})

	for _, effect := range light.Effects.EffectValues {
		displayName := hue.EffectDisplayName(string(effect))
		rows = append(rows, gridlayout.NewListItemRow("•", displayName))
	}

	return rows
}

// buildTimedEffectsListRows builds rows for the available timed effects section.
func (m *Model) buildTimedEffectsListRows(light hueclient.LightGet) []gridlayout.GridRow {
	if light.TimedEffects == nil || len(light.TimedEffects.EffectValues) == 0 {
		return nil
	}

	var rows []gridlayout.GridRow

	// Section header
	header := field.NewHeaderComponent("timed-effects-header", "Timed Effects", &m.styles, m.zones)
	rows = append(rows, gridlayout.GridRow{
		Type:    gridlayout.RowTypeSection,
		Section: header,
	})

	// Show current status if active
	statusStr := string(light.TimedEffects.Status)
	if statusStr != "" && statusStr != "no_effect" {
		displayStatus := formatTimedEffect(statusStr)
		statusStyle := m.styles.Success
		rows = append(rows, gridlayout.NewStyledInfoRow("Status", displayStatus+" (active)", infoLabelWidth, statusStyle))

		// Stop button when effect is active
		stopBtn := field.NewButtonComponent(
			"timed-effect-stop:"+light.Id, "Stop", "Stop",
			&m.styles, m.zones,
		)
		rows = append(rows, gridlayout.GridRow{
			Type: gridlayout.RowTypeNormal,
			Cells: []gridlayout.GridCell{
				{Component: gridlayout.NewLabelWithWidth("", infoLabelWidth)},
				{Component: stopBtn},
			},
		})
	}

	// Duration slider (5-120 minutes, default 30)
	currentDuration := 30
	if d, ok := m.timedEffectDurations[light.Id]; ok {
		currentDuration = d
	}
	durationSlider := field.NewSliderComponent(
		"timed-effect-duration:"+light.Id, "Duration",
		currentDuration, 5, 120, 5,
		&m.styles, m.zones,
	)
	rows = append(rows, gridlayout.GridRow{
		Type: gridlayout.RowTypeNormal,
		Cells: []gridlayout.GridCell{
			{Component: gridlayout.NewLabelWithWidth("Duration (min)", infoLabelWidth)},
			{Component: durationSlider},
		},
	})

	// Effect trigger buttons for available effects
	for _, effect := range light.TimedEffects.EffectValues {
		effectStr := string(effect)
		if effectStr == "no_effect" {
			continue
		}
		displayName := formatTimedEffect(effectStr)
		triggerBtn := field.NewButtonComponent(
			"timed-effect-trigger:"+light.Id+":"+effectStr, displayName, displayName,
			&m.styles, m.zones,
		)
		rows = append(rows, gridlayout.GridRow{
			Type: gridlayout.RowTypeNormal,
			Cells: []gridlayout.GridCell{
				{Component: gridlayout.NewLabelWithWidth(displayName, infoLabelWidth)},
				{Component: triggerBtn},
			},
		})
	}

	return rows
}

// formatTimedEffect formats a timed effect name for display.
func formatTimedEffect(effect string) string {
	switch effect {
	case "sunrise":
		return "Sunrise"
	case "sunset":
		return "Sunset"
	case "no_effect":
		return "None"
	default:
		return effect
	}
}

// buildGradientRows builds rows for the gradient section.
func (m *Model) buildGradientRows(light hueclient.LightGet, lightID string) []gridlayout.GridRow {
	if light.Gradient == nil {
		return nil
	}

	var rows []gridlayout.GridRow

	// Section header
	header := field.NewHeaderComponent("gradient-header", "Gradient", &m.styles, m.zones)
	rows = append(rows, gridlayout.GridRow{
		Type:    gridlayout.RowTypeSection,
		Section: header,
	})

	// Mode selector (if ModeValues available)
	if len(light.Gradient.ModeValues) > 0 {
		options := make([]field.Option, len(light.Gradient.ModeValues))
		currentIndex := 0
		for i, mode := range light.Gradient.ModeValues {
			options[i] = field.Option{Label: formatGradientMode(hueclient.LightGetGradientMode(mode)), Value: i}
			if hueclient.SupportedSounds(light.Gradient.Mode) == mode {
				currentIndex = i
			}
		}
		modeSelect := field.NewSelectComponent(
			"gradient-mode:"+lightID, "Mode", currentIndex, options,
			&m.styles, m.zones,
		)
		rows = append(rows, gridlayout.GridRow{
			Type: gridlayout.RowTypeNormal,
			Cells: []gridlayout.GridCell{
				{Component: gridlayout.NewLabelWithWidth("Mode", infoLabelWidth)},
				{Component: modeSelect},
			},
		})
	} else if light.Gradient.Mode != "" {
		// Read-only mode display if no mode values available
		rows = append(rows, gridlayout.NewInfoRow("Mode", formatGradientMode(light.Gradient.Mode), infoLabelWidth))
	}

	// Pixel count info
	if light.Gradient.PixelCount != nil {
		rows = append(rows, gridlayout.NewInfoRow("Pixels", fmt.Sprintf("%d", *light.Gradient.PixelCount), infoLabelWidth))
	}

	// Max points capability info
	if light.Gradient.PointsCapable > 0 {
		rows = append(rows, gridlayout.NewInfoRow("Max Points", fmt.Sprintf("%d", light.Gradient.PointsCapable), infoLabelWidth))
	}

	// Gradient editor
	if len(light.Gradient.Points) > 0 {
		maxPoints := light.Gradient.PointsCapable
		if maxPoints == 0 {
			maxPoints = 5 // default
		}

		points := convertGradientPoints(light.Gradient.Points)
		editor := field.NewGradientEditorComponent(
			"gradient-points:"+lightID, "Colors", points, maxPoints,
			&m.styles, m.zones,
		)
		rows = append(rows, gridlayout.GridRow{
			Type: gridlayout.RowTypeNormal,
			Cells: []gridlayout.GridCell{
				{Component: gridlayout.NewLabelWithWidth("Colors", infoLabelWidth)},
				{Component: editor},
			},
		})
	}

	return rows
}

// formatGradientMode converts a gradient mode to a display-friendly string.
func formatGradientMode(mode hueclient.LightGetGradientMode) string {
	switch mode {
	case hueclient.LightGetGradientModeInterpolatedPalette:
		return "Interpolated"
	case hueclient.LightGetGradientModeInterpolatedPaletteMirrored:
		return "Mirrored"
	case hueclient.LightGetGradientModeRandomPixelated:
		return "Pixelated"
	default:
		return string(mode)
	}
}

// convertGradientPoints converts hueclient gradient points to field gradient points.
func convertGradientPoints(gradientPoints []hueclient.GradientPointGet) []field.GradientPoint {
	points := make([]field.GradientPoint, 0, len(gradientPoints))
	for _, gp := range gradientPoints {
		points = append(points, field.GradientPoint{
			X: float64(gp.Color.Xy.X),
			Y: float64(gp.Color.Xy.Y),
		})
	}
	return points
}

// convertToAPIPoints converts field gradient points to hueclient colors.
func convertToAPIPoints(points []field.GradientPoint) []hueclient.ActionGetActionColor {
	colors := make([]hueclient.ActionGetActionColor, len(points))
	for i, pt := range points {
		colors[i] = hueclient.ActionGetActionColor{
			Xy: hueclient.XY{X: float32(pt.X), Y: float32(pt.Y)},
		}
	}
	return colors
}

// buildSignalingRows builds rows for the signaling section.
func (m *Model) buildSignalingRows(light hueclient.LightGet) []gridlayout.GridRow {
	if light.Signaling == nil || len(light.Signaling.SignalValues) == 0 {
		return nil
	}

	var rows []gridlayout.GridRow

	// Section header
	header := field.NewHeaderComponent("signaling-header", "Signaling", &m.styles, m.zones)
	rows = append(rows, gridlayout.GridRow{
		Type:    gridlayout.RowTypeSection,
		Section: header,
	})

	// Show active signal status if present
	if light.Signaling.Status != nil {
		signalStr := string(light.Signaling.Status.Signal)
		if signalStr != "" && signalStr != "no_signal" {
			displayName := hue.SignalingModeDisplayName(signalStr)
			statusStyle := m.styles.Success
			rows = append(rows, gridlayout.NewStyledInfoRow("Status", displayName+" (active)", infoLabelWidth, statusStyle))

			// Stop button when signal is active
			stopBtn := field.NewButtonComponent(
				"signal-stop:"+light.Id, "Stop", "Stop",
				&m.styles, m.zones,
			)
			rows = append(rows, gridlayout.GridRow{
				Type: gridlayout.RowTypeNormal,
				Cells: []gridlayout.GridCell{
					{Component: gridlayout.NewLabelWithWidth("", infoLabelWidth)},
					{Component: stopBtn},
				},
			})
		}
	}

	// Duration slider (5-60 seconds, default 15)
	currentDuration := 15
	if d, ok := m.signalDurations[light.Id]; ok {
		currentDuration = d
	}
	durationSlider := field.NewSliderComponent(
		"signal-duration:"+light.Id, "Duration",
		currentDuration, 5, 60, 5,
		&m.styles, m.zones,
	)
	rows = append(rows, gridlayout.GridRow{
		Type: gridlayout.RowTypeNormal,
		Cells: []gridlayout.GridCell{
			{Component: gridlayout.NewLabelWithWidth("Duration (sec)", infoLabelWidth)},
			{Component: durationSlider},
		},
	})

	// Signal trigger buttons for available signals
	for _, sig := range light.Signaling.SignalValues {
		sigStr := string(sig)
		if sigStr == "no_signal" {
			continue
		}
		displayName := hue.SignalingModeDisplayName(sigStr)
		triggerBtn := field.NewButtonComponent(
			"signal-trigger:"+light.Id+":"+sigStr, displayName, displayName,
			&m.styles, m.zones,
		)
		rows = append(rows, gridlayout.GridRow{
			Type: gridlayout.RowTypeNormal,
			Cells: []gridlayout.GridCell{
				{Component: gridlayout.NewLabelWithWidth(displayName, infoLabelWidth)},
				{Component: triggerBtn},
			},
		})
	}

	return rows
}

// buildPowerupRows builds rows for the power-on behavior section.
func (m *Model) buildPowerupRows(light hueclient.LightGet) []gridlayout.GridRow {
	if light.Powerup == nil {
		return nil
	}

	var rows []gridlayout.GridRow

	// Section header
	header := field.NewHeaderComponent("powerup-header", "Power-on Behavior", &m.styles, m.zones)
	rows = append(rows, gridlayout.GridRow{
		Type:    gridlayout.RowTypeSection,
		Section: header,
	})

	// Powerup preset (editable)
	if light.Powerup.Preset != "" {
		presetKeys := getSortedPowerupPresets()
		var options []field.Option
		currentIndex := 0
		currentPreset := string(light.Powerup.Preset)

		for i, key := range presetKeys {
			displayName := hue.PowerupPresetDisplayNames[key]
			options = append(options, field.Option{Label: displayName, Value: i})
			if key == currentPreset {
				currentIndex = i
			}
		}

		lightID := light.Id

		selectComp := field.NewSelectComponent(
			FieldIDPowerupPreset(lightID), "Preset", currentIndex, options,
			&m.styles, m.zones,
		)
		rows = append(rows, gridlayout.GridRow{
			Type: gridlayout.RowTypeNormal,
			Cells: []gridlayout.GridCell{
				{Component: gridlayout.NewLabelWithWidth("Preset", infoLabelWidth)},
				{Component: selectComp},
			},
		})
	}

	return rows
}

// buildDeviceServicesRows builds rows for the device services section.
func (m *Model) buildDeviceServicesRows(light hueclient.LightGet, device *hueclient.DeviceGet) []gridlayout.GridRow {
	if device == nil || len(device.Services) == 0 {
		return nil
	}

	var rows []gridlayout.GridRow

	// Section header
	header := field.NewHeaderComponent("services-header", "Device Services", &m.styles, m.zones)
	rows = append(rows, gridlayout.GridRow{
		Type:    gridlayout.RowTypeSection,
		Section: header,
	})

	for _, svc := range device.Services {
		rtype := string(svc.Rtype)
		displayName := hue.DeviceServiceDisplayName(rtype)
		if svc.Rid == light.Id {
			displayName = displayName + " " + m.styles.Dimmed.Render("(this)")
		}
		rows = append(rows, gridlayout.NewListItemRow("•", displayName))
	}

	return rows
}
