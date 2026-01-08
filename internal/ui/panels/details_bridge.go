package panels

import (
	"fmt"

	"github.com/kluzzebass/lazyhue/internal/hue"
	"github.com/kluzzebass/lazyhue/internal/ui/panels/details"
)

// buildBridgeView creates a details view for a bridge entity.
func (p *DetailsPanel) buildBridgeView(data BridgeData) *details.View {
	bridge := data.Bridge
	if bridge == nil {
		return details.NewView(p.styles)
	}

	view := details.NewView(p.styles)
	state := bridge.GetState()

	// Connection overview
	view.Add(details.Header("Connection"))
	conn := details.NewFields()

	var statusStyle = p.styles.Muted
	var statusText string
	switch bridge.Status {
	case hue.StatusConnected:
		statusStyle, statusText = p.styles.Success, "connected"
	case hue.StatusConnecting:
		statusStyle, statusText = p.styles.Warning, "connecting"
	case hue.StatusPairing:
		statusStyle, statusText = p.styles.Warning, "pairing"
	case hue.StatusError:
		statusStyle, statusText = p.styles.Error, "error"
	default:
		statusStyle, statusText = p.styles.Muted, "disconnected"
	}
	conn.AddStyled("Status", statusText, statusStyle)
	conn.Add("Bridge ID", bridge.Info.ID)
	if !bridge.LastSync.IsZero() {
		conn.Add("Last sync", bridge.LastSync.Format("15:04:05"))
	}
	if bridge.LastErr != nil {
		conn.AddStyled("Last error", bridge.LastErr.Error(), p.styles.Error)
	}
	view.Add(conn)

	// Wired Network
	view.Add(details.Header("Wired Network"))
	wired := details.NewFields()
	wired.Add("IP Address", bridge.Info.IPAddress)
	wired.AddStyled("Status", "connected", p.styles.Success)
	view.Add(wired)

	// WiFi Network (Bridge Pro only)
	if state != nil {
		wifi := state.GetWifiConnectivity()
		if len(wifi) > 0 {
			view.Add(details.Header("WiFi Network"))
			wifiFields := details.NewFields()
			for _, wc := range wifi {
				if wc.Status == "connected" {
					wifiFields.AddStyled("Status", "connected", p.styles.Success)
				} else {
					wifiFields.AddStyled("Status", "disconnected", p.styles.Muted)
				}
				wifiFields.Add("SSID configured", fmt.Sprintf("%v", wc.HasSSID))
			}
			view.Add(wifiFields)
		}
	}

	// Hardware info
	if state != nil {
		if bridgeDevice, ok := state.GetBridgeDevice(); ok && bridgeDevice.ProductData != nil {
			pd := bridgeDevice.ProductData
			view.Add(details.Header("Hardware"))
			hw := details.NewFields()
			if pd.ManufacturerName != nil {
				hw.Add("Manufacturer", *pd.ManufacturerName)
			}
			if pd.ProductName != nil {
				hw.Add("Product", *pd.ProductName)
			}
			if pd.ModelId != nil {
				hw.Add("Model", *pd.ModelId)
			}
			if pd.HardwarePlatformType != nil {
				hw.Add("Platform", *pd.HardwarePlatformType)
			}
			if pd.SoftwareVersion != nil {
				hw.Add("Firmware", *pd.SoftwareVersion)
			}
			if pd.Certified != nil {
				hw.Add("Certified", fmt.Sprintf("%v", *pd.Certified))
			}
			view.Add(hw)
		}
	}

	// Bridge resource info (timezone only)
	if state != nil {
		bridgeRes := state.GetBridgeResource()
		if bridgeRes != nil && bridgeRes.TimeZone != nil && bridgeRes.TimeZone.TimeZone != nil {
			view.Add(details.Header("Location"))
			loc := details.NewFields()
			loc.Add("Timezone", *bridgeRes.TimeZone.TimeZone)
			view.Add(loc)
		}
	}

	// Services
	if state != nil {
		if bridgeDevice, ok := state.GetBridgeDevice(); ok {
			if bridgeDevice.Services != nil && len(*bridgeDevice.Services) > 0 {
				view.Add(details.Header("Services"))
				servicesList := details.NewList()
				for _, svc := range *bridgeDevice.Services {
					if svc.Rtype != nil {
						servicesList.Add(string(*svc.Rtype))
					}
				}
				view.Add(servicesList)
			}
		}
	}

	// Summary statistics
	if state != nil {
		view.Add(details.Header("Summary"))
		summary := details.NewFields()

		rooms := state.AllRooms()
		zones := state.AllZones()
		lights := state.AllLights()
		scenes := state.AllScenes()
		devices := state.AllDevices()

		summary.Add("Rooms", fmt.Sprintf("%d", len(rooms)))
		summary.Add("Zones", fmt.Sprintf("%d", len(zones)))
		summary.Add("Lights", fmt.Sprintf("%d", len(lights)))
		summary.Add("Scenes", fmt.Sprintf("%d", len(scenes)))
		summary.Add("Devices", fmt.Sprintf("%d", len(devices)))

		// Count lights that are on
		lightsOn := 0
		for _, light := range lights {
			if light.IsOn() {
				lightsOn++
			}
		}
		summary.Add("Lights on", fmt.Sprintf("%d/%d", lightsOn, len(lights)))
		view.Add(summary)

		// Authenticated applications
		authApps := state.GetAuthApps()
		if len(authApps) > 0 {
			view.Add(details.Header(fmt.Sprintf("Authenticated Apps (%d)", len(authApps))))
			for _, app := range authApps {
				appList := details.NewList()
				appList.Add(app.AppName)
				view.Add(appList)

				if app.LastUseDate != "" {
					appInfo := details.NewFields()
					appInfo.AddSubMuted("Last used", app.LastUseDate)
					view.Add(appInfo)
				}
			}
		}
	}

	return view
}
