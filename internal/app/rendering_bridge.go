package app

import (
	"fmt"

	"github.com/charmbracelet/lipgloss/v2"
	"github.com/kluzzebass/lazyhue/internal/hue"
	"github.com/kluzzebass/lazyhue/internal/ui/component/field"
	gridlayout "github.com/kluzzebass/lazyhue/internal/ui/component/layout"
	"github.com/kluzzebass/lazyhue/internal/ui/panels"
)

// buildBridgeGridRows builds grid rows for a bridge details panel.
func (m *Model) buildBridgeGridRows(bridgeID string) []gridlayout.GridRow {
	var rows []gridlayout.GridRow

	bridge := m.manager.GetBridge(bridgeID)
	if bridge == nil {
		return rows
	}

	state := bridge.GetState()

	// 1. Controls section - instant actions (identify)
	if state != nil {
		if bridgeDevice, ok := state.GetBridgeDevice(); ok {
			deviceID := bridgeDevice.Id

			controlsHeader := field.NewHeaderComponent("controls-header", "Controls", &m.styles, m.zones)
			rows = append(rows, gridlayout.GridRow{
				Type:    gridlayout.RowTypeSection,
				Section: controlsHeader,
			})

			// Identify button
			if deviceID != "" {
				identifyBtn := field.NewButtonComponent(
					"bridge-identify:"+deviceID, "Identify", "Identify",
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

			// 2. Settings section (name)
			settingsHeader := field.NewHeaderComponent("settings-header", "Settings", &m.styles, m.zones)
			rows = append(rows, gridlayout.GridRow{
				Type:    gridlayout.RowTypeSection,
				Section: settingsHeader,
			})

			// Name field
			if bridgeDevice.Metadata.Name != "" && deviceID != "" {
				textInput := field.NewTextComponent(
					FieldIDBridgeName(deviceID), "Name", bridgeDevice.Metadata.Name,
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
		}
	}

	// Connection section
	connHeader := field.NewHeaderComponent("conn-header", "Connection", &m.styles, m.zones)
	rows = append(rows, gridlayout.GridRow{
		Type:    gridlayout.RowTypeSection,
		Section: connHeader,
	})

	statusStyle := m.styles.Dimmed
	statusText := "disconnected"
	switch bridge.Status {
	case hue.StatusConnected:
		statusStyle, statusText = m.styles.Success, "connected"
	case hue.StatusConnecting:
		statusStyle, statusText = m.styles.Warning, "connecting"
	case hue.StatusPairing:
		statusStyle, statusText = m.styles.Warning, "pairing"
	case hue.StatusError:
		statusStyle, statusText = m.styles.Error, "error"
	}
	rows = append(rows, gridlayout.NewStyledInfoRow("Status", statusText, infoLabelWidth, statusStyle))
	rows = append(rows, gridlayout.NewInfoRow("IP Address", bridge.Info.IPAddress, infoLabelWidth))
	if !bridge.LastSync.IsZero() {
		rows = append(rows, gridlayout.NewInfoRow("Last Sync", bridge.LastSync.Format("15:04:05"), infoLabelWidth))
	}

	// Hardware section
	if state != nil {
		if bridgeDevice, ok := state.GetBridgeDevice(); ok {
			pd := bridgeDevice.ProductData
			if pd.ProductName != "" || pd.ModelId != "" || pd.SoftwareVersion != "" {
				hwHeader := field.NewHeaderComponent("hw-header", "Hardware", &m.styles, m.zones)
				rows = append(rows, gridlayout.GridRow{
					Type:    gridlayout.RowTypeSection,
					Section: hwHeader,
				})

				if pd.ProductName != "" {
					rows = append(rows, gridlayout.NewInfoRow("Product", pd.ProductName, infoLabelWidth))
				}
				if pd.ModelId != "" {
					rows = append(rows, gridlayout.NewInfoRow("Model", pd.ModelId, infoLabelWidth))
				}
				if pd.SoftwareVersion != "" {
					rows = append(rows, gridlayout.NewInfoRow("Firmware", pd.SoftwareVersion, infoLabelWidth))
				}
				if pd.ProductArchetype != "" {
					rows = append(rows, gridlayout.NewStyledInfoRow("Archetype", hue.ProductArchetypeDisplayName(string(pd.ProductArchetype)), infoLabelWidth, m.styles.Dimmed))
				}
			}
		}
	}

	// Summary section
	if state != nil {
		summaryHeader := field.NewHeaderComponent("summary-header", "Summary", &m.styles, m.zones)
		rows = append(rows, gridlayout.GridRow{
			Type:    gridlayout.RowTypeSection,
			Section: summaryHeader,
		})

		rooms := state.AllRooms()
		zones := state.AllZones()
		lights := state.AllLights()
		scenes := state.AllScenes()
		devices := state.AllDevices()

		rows = append(rows, gridlayout.NewInfoRow("Rooms", fmt.Sprintf("%d", len(rooms)), infoLabelWidth))
		rows = append(rows, gridlayout.NewInfoRow("Zones", fmt.Sprintf("%d", len(zones)), infoLabelWidth))
		rows = append(rows, gridlayout.NewInfoRow("Lights", fmt.Sprintf("%d", len(lights)), infoLabelWidth))
		rows = append(rows, gridlayout.NewInfoRow("Scenes", fmt.Sprintf("%d", len(scenes)), infoLabelWidth))
		rows = append(rows, gridlayout.NewInfoRow("Devices", fmt.Sprintf("%d", len(devices)), infoLabelWidth))

		lightsOn := 0
		for _, light := range lights {
			if panels.IsLightOn(light) {
				lightsOn++
			}
		}
		rows = append(rows, gridlayout.NewInfoRow("Lights On", fmt.Sprintf("%d/%d", lightsOn, len(lights)), infoLabelWidth))
	}

	// IDs section
	idsHeader := field.NewHeaderComponent("ids-header", "IDs", &m.styles, m.zones)
	rows = append(rows, gridlayout.GridRow{
		Type:    gridlayout.RowTypeSection,
		Section: idsHeader,
	})
	rows = append(rows, gridlayout.NewStyledInfoRow("Bridge ID", bridge.Info.ID, infoLabelWidth, m.styles.Dimmed))

	return rows
}

// buildEntertainmentGridRows builds grid rows for an entertainment configuration details panel.
func (m *Model) buildEntertainmentGridRows(cfg hue.EntertainmentConfiguration, state *hue.BridgeState) []gridlayout.GridRow {
	var rows []gridlayout.GridRow

	// Status section
	header := field.NewHeaderComponent("status-header", "Status", &m.styles, m.zones)
	rows = append(rows, gridlayout.GridRow{
		Type:    gridlayout.RowTypeSection,
		Section: header,
	})

	if cfg.Status != "" {
		statusStyle := m.styles.Dimmed
		if cfg.Status == "active" {
			statusStyle = m.styles.Success
		}
		rows = append(rows, gridlayout.NewStyledInfoRow("Status", cfg.Status, infoLabelWidth, statusStyle))
	}
	if cfg.ConfigurationType != "" {
		rows = append(rows, gridlayout.NewInfoRow("Type", cfg.ConfigurationType, infoLabelWidth))
	}

	// Lights section
	if len(cfg.Lights) > 0 {
		cfgLightsHeaderText := fmt.Sprintf("Lights %s%d%s", m.styles.Dimmed.Render("["), len(cfg.Lights), m.styles.Dimmed.Render("]"))
		lightsHeader := field.NewHeaderComponent("lights-header", cfgLightsHeaderText, &m.styles, m.zones)
		rows = append(rows, gridlayout.GridRow{
			Type:    gridlayout.RowTypeSection,
			Section: lightsHeader,
		})

		cfgLightStyle := lipgloss.NewStyle().Foreground(m.styles.Theme.EntityLight)
		for _, light := range cfg.Lights {
			if light.Service != nil && light.Service.RID != "" {
				lightName := light.Service.RID
				if state != nil {
					if l, ok := state.GetLight(light.Service.RID); ok {
						lightName = state.GetLightName(l)
					}
				}
				rows = append(rows, gridlayout.NewListItemRow("•", cfgLightStyle.Render(lightName)))
			}
		}
	}

	// Channels section
	if len(cfg.Channels) > 0 {
		channelsHeaderText := fmt.Sprintf("Channels %s%d%s", m.styles.Dimmed.Render("["), len(cfg.Channels), m.styles.Dimmed.Render("]"))
		channelsHeader := field.NewHeaderComponent("channels-header", channelsHeaderText, &m.styles, m.zones)
		rows = append(rows, gridlayout.GridRow{
			Type:    gridlayout.RowTypeSection,
			Section: channelsHeader,
		})

		for _, ch := range cfg.Channels {
			posStr := ""
			if ch.Position != nil {
				posStr = fmt.Sprintf(" (%.1f, %.1f, %.1f)", ch.Position.X, ch.Position.Y, ch.Position.Z)
			}
			rows = append(rows, gridlayout.NewListItemRow("•", fmt.Sprintf("Channel %d%s", ch.ChannelID, m.styles.Dimmed.Render(posStr))))
		}
	}

	// IDs section
	idsHeader := field.NewHeaderComponent("ids-header", "IDs", &m.styles, m.zones)
	rows = append(rows, gridlayout.GridRow{
		Type:    gridlayout.RowTypeSection,
		Section: idsHeader,
	})
	rows = append(rows, gridlayout.NewStyledInfoRow("ID", cfg.ID, infoLabelWidth, m.styles.Dimmed))

	return rows
}
