package app

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss/v2"
	"github.com/kluzzebass/lazyhue/internal/hue"
	"github.com/kluzzebass/lazyhue/internal/hueclient"
	"github.com/kluzzebass/lazyhue/internal/ui"
	"github.com/kluzzebass/lazyhue/internal/ui/component/field"
	gridlayout "github.com/kluzzebass/lazyhue/internal/ui/component/layout"
)

// buildSceneGridRows builds grid rows for a scene details panel.
// Sections are ordered by immediacy: Controls > Settings > Info > Actions > IDs
func (m *Model) buildSceneGridRows(scene hueclient.SceneGet, state *hue.BridgeState) []gridlayout.GridRow {
	var rows []gridlayout.GridRow

	sceneID := scene.Id

	// 1. Controls section - instant action (recall/activate scene)
	controlsHeader := field.NewHeaderComponent("controls-header", "Controls", &m.styles, m.zones)
	rows = append(rows, gridlayout.GridRow{
		Type:    gridlayout.RowTypeSection,
		Section: controlsHeader,
	})

	if sceneID != "" {
		activateBtn := field.NewButtonComponent(
			FieldIDSceneRecall(sceneID), "Activate", "Activate",
			&m.styles, m.zones,
		)
		rows = append(rows, gridlayout.GridRow{
			Type: gridlayout.RowTypeNormal,
			Cells: []gridlayout.GridCell{
				{Component: gridlayout.NewLabelWithWidth("Activate", infoLabelWidth)},
				{Component: activateBtn},
			},
		})
	}

	// 2. Settings section (name)
	settingsHeader := field.NewHeaderComponent("settings-header", "Settings", &m.styles, m.zones)
	rows = append(rows, gridlayout.GridRow{
		Type:    gridlayout.RowTypeSection,
		Section: settingsHeader,
	})

	if scene.Metadata.Name != "" {
		textInput := field.NewTextComponent(
			FieldIDSceneName(sceneID), "Name", scene.Metadata.Name,
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

	// 3. Info section (status, group, speed, etc.)
	infoHeader := field.NewHeaderComponent("info-header", "Info", &m.styles, m.zones)
	rows = append(rows, gridlayout.GridRow{
		Type:    gridlayout.RowTypeSection,
		Section: infoHeader,
	})

	if scene.Status.Active != nil {
		status := string(*scene.Status.Active)
		statusStyle := m.styles.Dimmed
		if status == "active" {
			statusStyle = m.styles.Success
		}
		rows = append(rows, gridlayout.NewStyledInfoRow("Status", status, infoLabelWidth, statusStyle))
	}

	// Group (room/zone)
	if scene.Group.Rid != "" && state != nil {
		groupName := ""
		groupType := string(scene.Group.Rtype)
		var groupStyle lipgloss.Style
		if room, ok := state.GetRoom(scene.Group.Rid); ok {
			groupName = state.GetRoomName(room)
			groupStyle = lipgloss.NewStyle().Foreground(m.styles.Theme.EntityRoom)
		} else if zone, ok := state.GetZone(scene.Group.Rid); ok {
			groupName = zone.ZoneName("")
			groupStyle = lipgloss.NewStyle().Foreground(m.styles.Theme.EntityZone)
		}
		if groupName != "" {
			rows = append(rows, gridlayout.NewStyledInfoRow("Group", groupStyle.Render(groupName), infoLabelWidth, lipgloss.NewStyle()))
		}
		rows = append(rows, gridlayout.NewStyledInfoRow("Group Type", groupType, infoLabelWidth, m.styles.Dimmed))
	}

	// Speed and auto dynamic
	if scene.Speed > 0 {
		rows = append(rows, gridlayout.NewInfoRow("Speed", fmt.Sprintf("%.2f", scene.Speed), infoLabelWidth))
	}
	rows = append(rows, gridlayout.NewInfoRow("Auto Dynamic", fmt.Sprintf("%v", scene.AutoDynamic), infoLabelWidth))

	// Color Palette - visual representation of colors in the scene
	if len(scene.Actions) > 0 {
		colorPalette := m.buildSceneColorPalette(scene.Actions)
		if colorPalette != "" {
			paletteHeader := field.NewHeaderComponent("palette-header", "Palette", &m.styles, m.zones)
			rows = append(rows, gridlayout.GridRow{
				Type:    gridlayout.RowTypeSection,
				Section: paletteHeader,
			})
			rows = append(rows, gridlayout.NewInfoRow("Colors", colorPalette, infoLabelWidth))
		}
	}

	// Actions
	if len(scene.Actions) > 0 {
		actionsHeaderText := fmt.Sprintf("Actions %s%d%s", m.styles.Dimmed.Render("["), len(scene.Actions), m.styles.Dimmed.Render("]"))
		actionsHeader := field.NewHeaderComponent("actions-header", actionsHeaderText, &m.styles, m.zones)
		rows = append(rows, gridlayout.GridRow{
			Type:    gridlayout.RowTypeSection,
			Section: actionsHeader,
		})

		actionLightStyle := lipgloss.NewStyle().Foreground(m.styles.Theme.EntityLight)
		for _, action := range scene.Actions {
			targetName := "unknown"
			if action.Target.Rid != "" {
				targetName = action.Target.Rid
				if state != nil {
					if light, ok := state.GetLight(action.Target.Rid); ok {
						targetName = state.GetLightName(light)
					}
				}
			}

			actionDesc := ""
			var parts []string
			if action.Action.On != nil && action.Action.On.On {
				parts = append(parts, "on")
			} else if action.Action.On != nil && !action.Action.On.On {
				parts = append(parts, "off")
			}
			if action.Action.Dimming != nil {
				parts = append(parts, fmt.Sprintf("%.0f%%", action.Action.Dimming.Brightness))
			}
			if action.Action.ColorTemperature != nil && action.Action.ColorTemperature.Mirek != 0 {
				kelvin := 1000000 / action.Action.ColorTemperature.Mirek
				parts = append(parts, fmt.Sprintf("%dK", kelvin))
			}
			if len(parts) > 0 {
				actionDesc = " → " + strings.Join(parts, ", ")
			}

			rows = append(rows, gridlayout.NewListItemRow("•", actionLightStyle.Render(targetName)+m.styles.Dimmed.Render(actionDesc)))
		}
	}

	// IDs section
	idsHeader := field.NewHeaderComponent("ids-header", "IDs", &m.styles, m.zones)
	rows = append(rows, gridlayout.GridRow{
		Type:    gridlayout.RowTypeSection,
		Section: idsHeader,
	})

	dimStyle := m.styles.Dimmed
	if scene.Id != "" {
		rows = append(rows, gridlayout.NewStyledInfoRow("ID", scene.Id, infoLabelWidth, dimStyle))
	}
	if scene.IdV1 != nil {
		rows = append(rows, gridlayout.NewStyledInfoRow("V1 ID", *scene.IdV1, infoLabelWidth, dimStyle))
	}

	return rows
}

// buildSmartSceneGridRows builds the detail grid rows for a smart scene.
func (m *Model) buildSmartSceneGridRows(scene hueclient.SmartSceneGet, state *hue.BridgeState, bridgeID string) []gridlayout.GridRow {
	var rows []gridlayout.GridRow

	sceneID := scene.Id

	// 1. Controls section - activate/deactivate toggle and delete
	controlsHeader := field.NewHeaderComponent("controls-header", "Controls", &m.styles, m.zones)
	rows = append(rows, gridlayout.GridRow{
		Type:    gridlayout.RowTypeSection,
		Section: controlsHeader,
	})

	if sceneID != "" {
		// Activate/Deactivate toggle
		isActive := scene.State == "active"
		toggle := field.NewToggleComponent(
			FieldIDSmartSceneToggle(sceneID), "Active", isActive,
			&m.styles, m.zones,
		)
		rows = append(rows, gridlayout.GridRow{
			Type: gridlayout.RowTypeNormal,
			Cells: []gridlayout.GridCell{
				{Component: gridlayout.NewLabelWithWidth("Active", infoLabelWidth)},
				{Component: toggle},
			},
		})

		}

	// 2. Info section
	infoHeader := field.NewHeaderComponent("info-header", "Info", &m.styles, m.zones)
	rows = append(rows, gridlayout.GridRow{
		Type:    gridlayout.RowTypeSection,
		Section: infoHeader,
	})

	// Name
	if scene.Metadata.Name != "" {
		rows = append(rows, gridlayout.NewInfoRow("Name", scene.Metadata.Name, infoLabelWidth))
	}

	// State
	stateStr := string(scene.State)
	stateStyle := m.styles.Dimmed
	if scene.State == "active" {
		stateStyle = m.styles.Success
	}
	rows = append(rows, gridlayout.NewStyledInfoRow("State", stateStr, infoLabelWidth, stateStyle))

	// Group (room/zone)
	if scene.Group.Rid != "" && state != nil {
		groupName := ""
		groupType := string(scene.Group.Rtype)
		var groupStyle lipgloss.Style
		switch scene.Group.Rtype {
		case hueclient.ResourceTypeRoom:
			if room, ok := state.GetRoom(scene.Group.Rid); ok {
				groupName = state.GetRoomName(room)
				groupStyle = lipgloss.NewStyle().Foreground(m.styles.Theme.EntityRoom)
			}
		case hueclient.ResourceTypeZone:
			if zone, ok := state.GetZone(scene.Group.Rid); ok {
				groupName = zone.ZoneName("Unknown")
				groupStyle = lipgloss.NewStyle().Foreground(m.styles.Theme.EntityZone)
			}
		}
		if groupName != "" {
			rows = append(rows, gridlayout.NewStyledInfoRow("Group", groupStyle.Render(groupName), infoLabelWidth, lipgloss.NewStyle()))
		}
		rows = append(rows, gridlayout.NewStyledInfoRow("Group Type", groupType, infoLabelWidth, m.styles.Dimmed))
	}

	// Transition duration (convert from ms to seconds)
	if scene.TransitionDuration > 0 {
		durationSec := float64(scene.TransitionDuration) / 1000.0
		rows = append(rows, gridlayout.NewInfoRow("Transition", fmt.Sprintf("%.1fs", durationSec), infoLabelWidth))
	}

	// Active timeslot
	if scene.ActiveTimeslot != nil {
		slotInfo := fmt.Sprintf("Slot %d (%s)", scene.ActiveTimeslot.TimeslotId, string(scene.ActiveTimeslot.Weekday))
		rows = append(rows, gridlayout.NewInfoRow("Active Slot", slotInfo, infoLabelWidth))
	}

	// Schedule section - show detailed timeslots
	if len(scene.WeekTimeslots) > 0 {
		scheduleHeaderText := fmt.Sprintf("Schedule %s%d days%s", m.styles.Dimmed.Render("["), len(scene.WeekTimeslots), m.styles.Dimmed.Render("]"))
		scheduleHeader := field.NewHeaderComponent("schedule-header", scheduleHeaderText, &m.styles, m.zones)
		rows = append(rows, gridlayout.GridRow{
			Type:    gridlayout.RowTypeSection,
			Section: scheduleHeader,
		})

		for _, daySlots := range scene.WeekTimeslots {
			// Format recurrence days
			var days []string
			for _, day := range daySlots.Recurrence {
				days = append(days, formatWeekdayShort(string(day)))
			}
			daysStr := strings.Join(days, ", ")

			// Format timeslots for this day
			var slotStrs []string
			for _, slot := range daySlots.Timeslots {
				timeStr := formatSmartSceneTime(slot.StartTime)
				// Try to get scene name from target
				sceneName := ""
				if state != nil && slot.Target.Rid != "" {
					if targetScene, ok := state.GetScene(slot.Target.Rid); ok {
						sceneName = targetScene.SceneName("")
					}
				}
				if sceneName != "" {
					slotStrs = append(slotStrs, fmt.Sprintf("%s → %s", timeStr, sceneName))
				} else {
					slotStrs = append(slotStrs, timeStr)
				}
			}

			// Show day row with slot count
			dayLabel := daysStr
			if len(daySlots.Timeslots) > 0 {
				dayLabel += m.styles.Dimmed.Render(fmt.Sprintf(" (%d slots)", len(daySlots.Timeslots)))
			}
			rows = append(rows, gridlayout.NewInfoRow("Days", dayLabel, infoLabelWidth))

			// Show individual slots
			for _, slotStr := range slotStrs {
				rows = append(rows, gridlayout.NewListItemRow("•", m.styles.Dimmed.Render(slotStr)))
			}
		}
	}

	// IDs section
	idsHeader := field.NewHeaderComponent("ids-header", "IDs", &m.styles, m.zones)
	rows = append(rows, gridlayout.GridRow{
		Type:    gridlayout.RowTypeSection,
		Section: idsHeader,
	})
	if scene.Id != "" {
		rows = append(rows, gridlayout.NewStyledInfoRow("ID", scene.Id, infoLabelWidth, m.styles.Dimmed))
	}

	return rows
}

// buildSceneColorPalette builds a visual color palette from scene actions.
// Returns a string of colored blocks representing the colors used in the scene.
func (m *Model) buildSceneColorPalette(actions []hueclient.ActionGet) string {
	var colorBlocks []string
	seenColors := make(map[string]bool) // track unique colors by hex

	for _, action := range actions {
		var hexColor string

		// Check for XY color
		if action.Action.Color != nil {
			x := float64(action.Action.Color.Xy.X)
			y := float64(action.Action.Color.Xy.Y)
			// Use default brightness for palette display
			r, g, b := ui.XyToRGB(x, y, 80)
			hexColor = ui.RGBToHex(r, g, b)
		} else if action.Action.ColorTemperature != nil && action.Action.ColorTemperature.Mirek != 0 {
			// Color temperature
			r, g, b := ui.MirekToRGB(action.Action.ColorTemperature.Mirek)
			hexColor = ui.RGBToHex(r, g, b)
		}

		// Add unique colors only
		if hexColor != "" && !seenColors[hexColor] {
			seenColors[hexColor] = true
			colorStyle := lipgloss.NewStyle().Background(lipgloss.Color(hexColor))
			colorBlocks = append(colorBlocks, colorStyle.Render("  "))
		}
	}

	if len(colorBlocks) == 0 {
		return ""
	}

	return strings.Join(colorBlocks, " ")
}

// formatWeekdayShort converts weekday names to short form.
func formatWeekdayShort(day string) string {
	switch day {
	case "monday":
		return "Mon"
	case "tuesday":
		return "Tue"
	case "wednesday":
		return "Wed"
	case "thursday":
		return "Thu"
	case "friday":
		return "Fri"
	case "saturday":
		return "Sat"
	case "sunday":
		return "Sun"
	default:
		if len(day) > 3 {
			return strings.ToUpper(day[:1]) + day[1:3]
		}
		return day
	}
}

// formatSmartSceneTime formats a smart scene start time for display.
func formatSmartSceneTime(startTime hueclient.SmartSceneTimeslotGetStartTime) string {
	if startTime.Kind == "sunset" {
		return "sunset"
	}
	// Format as HH:MM
	return fmt.Sprintf("%02d:%02d", startTime.Time.Hour, startTime.Time.Minute)
}
