package app

import (
	"fmt"
	"math"
	"time"

	"github.com/kluzzebass/lazyhue/internal/hue"
	"github.com/kluzzebass/lazyhue/internal/hueclient"
	"github.com/kluzzebass/lazyhue/internal/ui/component/field"
	gridlayout "github.com/kluzzebass/lazyhue/internal/ui/component/layout"
)

// buildDeviceGridRows builds grid rows for a device details panel.
// Sections are ordered by immediacy: Controls > Settings > Sensors > Product > Zigbee > IDs
func (m *Model) buildDeviceGridRows(device hueclient.DeviceGet, state *hue.BridgeState) []gridlayout.GridRow {
	var rows []gridlayout.GridRow

	deviceID := device.Id

	// 1. Controls section - instant actions (identify)
	controlsHeader := field.NewHeaderComponent("controls-header", "Controls", &m.styles, m.zones)
	rows = append(rows, gridlayout.GridRow{
		Type:    gridlayout.RowTypeSection,
		Section: controlsHeader,
	})

	if deviceID != "" {
		identifyBtn := field.NewButtonComponent(
			"identify:"+deviceID, "Identify", "Identify",
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

	// 2. Settings section (name, sensor toggles)
	settingsHeader := field.NewHeaderComponent("settings-header", "Settings", &m.styles, m.zones)
	rows = append(rows, gridlayout.GridRow{
		Type:    gridlayout.RowTypeSection,
		Section: settingsHeader,
	})

	if device.Metadata.Name != "" {
		textInput := field.NewTextComponent(
			FieldIDDeviceName(deviceID), "Name", device.Metadata.Name,
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

	// Room assignment dropdown
	if state != nil && deviceID != "" {
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
			"device-room:"+deviceID, "Room", selectedIndex, options,
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

	// Sensor enable/disable toggles in Settings
	if state != nil {
		// Motion sensor toggle and sensitivity
		if motionID, motion, found := state.GetDeviceMotionSensor(device); found {
			enabled := motion.Enabled
			motionToggle := field.NewToggleComponent(
				"motion-enabled:"+motionID, "Motion Sensor", enabled,
				&m.styles, m.zones,
			)
			rows = append(rows, gridlayout.GridRow{
				Type: gridlayout.RowTypeNormal,
				Cells: []gridlayout.GridCell{
					{Component: gridlayout.NewLabelWithWidth("Motion Sensor", infoLabelWidth)},
					{Component: motionToggle},
				},
			})

			// Sensitivity slider
			if motion.Sensitivity != nil && motion.Sensitivity.SensitivityMax != nil {
				sensitivity := motion.Sensitivity.Sensitivity
				maxSensitivity := *motion.Sensitivity.SensitivityMax
				sensitivitySlider := field.NewSliderComponent(
					"motion-sensitivity:"+motionID, "Sensitivity",
					sensitivity, 0, maxSensitivity, 1,
					&m.styles, m.zones,
				)
				rows = append(rows, gridlayout.GridRow{
					Type: gridlayout.RowTypeNormal,
					Cells: []gridlayout.GridCell{
						{Component: gridlayout.NewLabelWithWidth("Sensitivity", infoLabelWidth)},
						{Component: sensitivitySlider},
					},
				})
			}
		}

		// Temperature sensor toggle
		if tempID, tempSensor, found := state.GetDeviceTemperatureSensor(device); found {
			enabled := tempSensor.Enabled
			tempToggle := field.NewToggleComponent(
				"temp-enabled:"+tempID, "Temp Sensor", enabled,
				&m.styles, m.zones,
			)
			rows = append(rows, gridlayout.GridRow{
				Type: gridlayout.RowTypeNormal,
				Cells: []gridlayout.GridCell{
					{Component: gridlayout.NewLabelWithWidth("Temp Sensor", infoLabelWidth)},
					{Component: tempToggle},
				},
			})
		}

		// Light level sensor toggle
		if llID, llSensor, found := state.GetDeviceLightLevelSensor(device); found {
			enabled := llSensor.Enabled
			llToggle := field.NewToggleComponent(
				"ll-enabled:"+llID, "Light Sensor", enabled,
				&m.styles, m.zones,
			)
			rows = append(rows, gridlayout.GridRow{
				Type: gridlayout.RowTypeNormal,
				Cells: []gridlayout.GridCell{
					{Component: gridlayout.NewLabelWithWidth("Light Sensor", infoLabelWidth)},
					{Component: llToggle},
				},
			})
		}
	}

	// 3. Product section
	pd := device.ProductData
	if pd.ProductName != "" || pd.ManufacturerName != "" || pd.ModelId != "" {
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
		// Firmware update status
		if state != nil {
			if update, found := state.GetDeviceSoftwareUpdateStatus(device); found {
				updateStatus := string(update.State)
				statusStyle := m.styles.Dimmed
				displayStatus := "Up to date"
				switch updateStatus {
				case "no_update":
					displayStatus = "Up to date"
					statusStyle = m.styles.Success
				case "update_pending":
					displayStatus = "Update pending"
					statusStyle = m.styles.Warning
				case "ready_to_install":
					displayStatus = "Ready to install"
					statusStyle = m.styles.Warning
				case "installing":
					displayStatus = "Installing..."
					statusStyle = m.styles.Success
				}
				rows = append(rows, gridlayout.NewStyledInfoRow("Update", displayStatus, infoLabelWidth, statusStyle))
			}
		}
		if pd.ProductArchetype != "" {
			rows = append(rows, gridlayout.NewStyledInfoRow("Archetype", hue.ProductArchetypeDisplayName(string(pd.ProductArchetype)), infoLabelWidth, m.styles.Dimmed))
		}
	}

	// Sensors section
	if state != nil {
		var sensorRows []gridlayout.GridRow

		// Motion sensor (read-only status)
		if _, motion, found := state.GetDeviceMotionSensor(device); found {
			isDetecting := false
			if motion.Motion.MotionReport != nil {
				isDetecting = motion.Motion.MotionReport.Motion
			} else {
				isDetecting = motion.Motion.Motion
			}
			status := "clear"
			statusStyle := m.styles.Dimmed
			if isDetecting {
				status = "detected"
				statusStyle = m.styles.Success
			}
			sensorRows = append(sensorRows, gridlayout.NewStyledInfoRow("Motion", status, infoLabelWidth, statusStyle))
		}

		// Temperature (read-only value)
		if _, tempSensor, found := state.GetDeviceTemperatureSensor(device); found {
			tempValue := "N/A"
			if tempSensor.Temperature.TemperatureReport != nil {
				tempValue = fmt.Sprintf("%.1f°C", tempSensor.Temperature.TemperatureReport.Temperature)
			} else {
				tempValue = fmt.Sprintf("%.1f°C", tempSensor.Temperature.Temperature)
			}
			sensorRows = append(sensorRows, gridlayout.NewInfoRow("Temperature", tempValue, infoLabelWidth))
		}

		// Light level (read-only value)
		if _, llSensor, found := state.GetDeviceLightLevelSensor(device); found {
			llValue := "N/A"
			level := 0
			if llSensor.Light.LightLevelReport != nil {
				level = llSensor.Light.LightLevelReport.LightLevel
			} else {
				level = llSensor.Light.LightLevel
			}
			if level > 1 {
				lux := math.Pow(10, float64(level-1)/10000.0)
				llValue = fmt.Sprintf("%.0f lux", lux)
			} else {
				llValue = "0 lux"
			}
			sensorRows = append(sensorRows, gridlayout.NewInfoRow("Light Level", llValue, infoLabelWidth))
		}

		// Battery
		if hasBattery, battLevel, battState := state.GetDeviceBattery(device); hasBattery {
			value := fmt.Sprintf("%d%%", battLevel)
			if battState != "" {
				value += fmt.Sprintf(" (%s)", battState)
			}
			sensorRows = append(sensorRows, gridlayout.NewInfoRow("Battery", value, infoLabelWidth))
		}

		if len(sensorRows) > 0 {
			sensorsHeader := field.NewHeaderComponent("sensors-header", "Sensors", &m.styles, m.zones)
			rows = append(rows, gridlayout.GridRow{
				Type:    gridlayout.RowTypeSection,
				Section: sensorsHeader,
			})
			rows = append(rows, sensorRows...)
		}

		// Buttons section - for switches/remotes/dials
		buttons := state.GetDeviceButtons(device)
		if len(buttons) > 0 {
			buttonsHeader := field.NewHeaderComponent("buttons-header", "Buttons", &m.styles, m.zones)
			rows = append(rows, gridlayout.GridRow{
				Type:    gridlayout.RowTypeSection,
				Section: buttonsHeader,
			})

			for _, btn := range buttons {
				// Button label: "Button 1", "Button 2", etc.
				label := fmt.Sprintf("Button %d", btn.Metadata.ControlId)

				// Get last event and format it
				lastEvent := "—"
				if btn.Button.ButtonReport != nil && btn.Button.ButtonReport.Event != "" {
					lastEvent = formatButtonEventForDetails(string(btn.Button.ButtonReport.Event))
					// Add timestamp - Updated is time.Time
					if !btn.Button.ButtonReport.Updated.IsZero() {
						lastEvent += " " + formatButtonTime(btn.Button.ButtonReport.Updated)
					}
				} else if btn.Button.LastEvent != nil && *btn.Button.LastEvent != "" {
					lastEvent = formatButtonEventForDetails(string(*btn.Button.LastEvent))
				}

				rows = append(rows, gridlayout.NewInfoRow(label, lastEvent, infoLabelWidth))
			}
		}

		// Zigbee connectivity
		if zc, ok := state.GetDeviceZigbeeConnectivity(device); ok {
			zigbeeHeader := field.NewHeaderComponent("zigbee-header", "Zigbee", &m.styles, m.zones)
			rows = append(rows, gridlayout.GridRow{
				Type:    gridlayout.RowTypeSection,
				Section: zigbeeHeader,
			})

			statusStyle := m.styles.Dimmed
			if zc.Status == "connected" {
				statusStyle = m.styles.Success
			} else if zc.Status == "connectivity_issue" || zc.Status == "disconnected" {
				statusStyle = m.styles.Error
			}
			rows = append(rows, gridlayout.NewStyledInfoRow("Status", zc.Status, infoLabelWidth, statusStyle))

			if zc.MacAddress != "" {
				rows = append(rows, gridlayout.NewStyledInfoRow("MAC", zc.MacAddress, infoLabelWidth, m.styles.Dimmed))
			}

			if zc.Channel != nil && zc.Channel.Value != "" {
				// Extract channel number from "channel_25" format
				channelDisplay := zc.Channel.Value
				if len(channelDisplay) > 8 && channelDisplay[:8] == "channel_" {
					channelDisplay = channelDisplay[8:]
				}
				if zc.Channel.Status == "changing" {
					channelDisplay += " (changing)"
				}
				rows = append(rows, gridlayout.NewStyledInfoRow("Channel", channelDisplay, infoLabelWidth, m.styles.Dimmed))
			}
		}
	}

	// IDs section
	idsHeader := field.NewHeaderComponent("ids-header", "IDs", &m.styles, m.zones)
	rows = append(rows, gridlayout.GridRow{
		Type:    gridlayout.RowTypeSection,
		Section: idsHeader,
	})

	dimStyle := m.styles.Dimmed
	if device.Id != "" {
		rows = append(rows, gridlayout.NewStyledInfoRow("ID", device.Id, infoLabelWidth, dimStyle))
	}
	if device.Type != "" {
		rows = append(rows, gridlayout.NewStyledInfoRow("Type", string(device.Type), infoLabelWidth, dimStyle))
	}

	return rows
}

// formatButtonEventForDetails converts Hue button event names to human-readable form for the details panel.
func formatButtonEventForDetails(event string) string {
	switch event {
	case "initial_press":
		return "pressed"
	case "repeat":
		return "held"
	case "short_release":
		return "short press"
	case "long_release":
		return "long press"
	case "double_short_release":
		return "double press"
	case "long_press":
		return "long press (held)"
	default:
		return event
	}
}

// formatButtonTime formats a button event time for display.
func formatButtonTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}

	// Calculate time since event
	since := time.Since(t)
	switch {
	case since < time.Minute:
		return fmt.Sprintf("(%ds ago)", int(since.Seconds()))
	case since < time.Hour:
		return fmt.Sprintf("(%dm ago)", int(since.Minutes()))
	case since < 24*time.Hour:
		return fmt.Sprintf("(%dh ago)", int(since.Hours()))
	default:
		return fmt.Sprintf("(%dd ago)", int(since.Hours()/24))
	}
}
