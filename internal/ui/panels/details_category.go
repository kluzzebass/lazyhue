package panels

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/kluzzebass/lazyhue/internal/hue"
	"github.com/kluzzebass/lazyhue/internal/hueclient"
	"github.com/kluzzebass/lazyhue/internal/ui/panels/details"
)

// buildLightsCategoryView creates a details view for a lights category aggregate.
func (p *DetailsPanel) buildLightsCategoryView(data LightsCategoryData) *details.View {
	view := details.NewView(p.styles)

	// Title
	view.Add(details.Text(p.styles.Subtitle.Render("Lights in "+data.ParentName) + "\n"))

	// Refresh lights from current state if possible
	if p.state != nil {
		for _, room := range p.state.AllRooms() {
			if hue.RoomName(room, "") == data.ParentName {
				data.Lights = p.state.RoomLights(room)
				break
			}
		}
		if len(data.Lights) == 0 {
			for _, zone := range p.state.AllZones() {
				if hue.RoomName(zone, "") == data.ParentName {
					data.Lights = p.state.RoomLights(zone)
					break
				}
			}
		}
	}

	if len(data.Lights) == 0 {
		view.Add(details.Text(p.styles.Muted.Render("  No lights\n")))
		return view
	}

	// Calculate aggregates
	onCount := 0
	totalBrightness := 0.0
	brightnessCount := 0
	for _, light := range data.Lights {
		if IsLightOn(light) {
			onCount++
			if light.Dimming != nil && light.Dimming.Brightness != nil {
				totalBrightness += float64(*light.Dimming.Brightness)
				brightnessCount++
			}
		}
	}

	// Summary
	summary := details.NewFields()
	summary.Add("Total", fmt.Sprintf("%d lights", len(data.Lights)))
	summary.Add("On/Off", fmt.Sprintf("%d / %d", onCount, len(data.Lights)-onCount))
	if brightnessCount > 0 {
		summary.Add("Avg Brightness", fmt.Sprintf("%.0f%%", totalBrightness/float64(brightnessCount)))
	}
	view.Add(summary)

	// List lights
	view.Add(details.Blank())
	lightsList := details.NewList()
	for _, light := range data.Lights {
		name := p.state.GetLightName(light)
		indicator := RenderLightIndicatorFromLight(light, p.styles)

		var detailParts []string
		if IsLightOn(light) {
			if light.Dimming != nil && light.Dimming.Brightness != nil {
				detailParts = append(detailParts, fmt.Sprintf("%.0f%%", float64(*light.Dimming.Brightness)))
			}
			if light.ColorTemperature != nil && light.ColorTemperature.Mirek != nil {
				kelvin := 1000000 / *light.ColorTemperature.Mirek
				detailParts = append(detailParts, fmt.Sprintf("%dK", kelvin))
			}
			if light.Color != nil && light.Color.Xy != nil && light.Color.Xy.X != nil && light.Color.Xy.Y != nil {
				if light.ColorTemperature == nil || light.ColorTemperature.Mirek == nil {
					detailParts = append(detailParts, fmt.Sprintf("xy(%.2f,%.2f)", *light.Color.Xy.X, *light.Color.Xy.Y))
				}
			}
		} else {
			detailParts = append(detailParts, "off")
		}

		suffix := ""
		if len(detailParts) > 0 {
			suffix = " " + strings.Join(detailParts, ", ")
		}
		lightsList.AddCustomFull(details.ListItem{
			Bullet: indicator,
			Text:   name,
			Suffix: suffix,
		})
	}
	view.Add(lightsList)

	return view
}

// buildDevicesCategoryView creates a details view for a devices category aggregate.
func (p *DetailsPanel) buildDevicesCategoryView(data DevicesCategoryData) *details.View {
	view := details.NewView(p.styles)

	// Title
	view.Add(details.Text(p.styles.Subtitle.Render("Devices in "+data.ParentName) + "\n"))

	if len(data.Devices) == 0 {
		view.Add(details.Text(p.styles.Muted.Render("  No devices\n")))
		return view
	}

	// Group by product type
	productCounts := make(map[string]int)
	for _, device := range data.Devices {
		productName := "Unknown"
		if device.ProductData != nil && device.ProductData.ProductName != nil {
			productName = *device.ProductData.ProductName
		}
		productCounts[productName]++
	}

	// Summary
	summary := details.NewFields()
	summary.Add("Total", fmt.Sprintf("%d devices", len(data.Devices)))
	view.Add(summary)

	if len(productCounts) > 1 {
		products := make([]string, 0, len(productCounts))
		for product := range productCounts {
			products = append(products, product)
		}
		sort.Strings(products)

		view.Add(details.Header("By Product"))
		byProduct := details.NewFields()
		for _, product := range products {
			byProduct.Add(product, fmt.Sprintf("%d", productCounts[product]))
		}
		view.Add(byProduct)
	}

	// List devices
	view.Add(details.Blank())
	devicesList := details.NewList()
	for _, device := range data.Devices {
		name := hue.DeviceName(device, "")
		productName := ""
		if device.ProductData != nil && device.ProductData.ProductName != nil {
			productName = *device.ProductData.ProductName
		}
		if productName != "" {
			devicesList.Add(fmt.Sprintf("%s (%s)", name, productName))
		} else {
			devicesList.Add(name)
		}
	}
	view.Add(devicesList)

	return view
}

// buildScenesCategoryView creates a details view for a scenes category aggregate.
func (p *DetailsPanel) buildScenesCategoryView(data ScenesCategoryData) *details.View {
	view := details.NewView(p.styles)

	// Title
	view.Add(details.Text(p.styles.Subtitle.Render("Scenes in "+data.ParentName) + "\n"))

	// Refresh scenes from current state if possible
	if p.state != nil {
		for _, room := range p.state.AllRooms() {
			if hue.RoomName(room, "") == data.ParentName {
				if room.Id != nil {
					data.Scenes = p.state.RoomScenes(*room.Id)
				}
				break
			}
		}
		if len(data.Scenes) == 0 {
			for _, zone := range p.state.AllZones() {
				if hue.RoomName(zone, "") == data.ParentName {
					if zone.Id != nil {
						data.Scenes = p.state.RoomScenes(*zone.Id)
					}
					break
				}
			}
		}
	}

	if len(data.Scenes) == 0 {
		view.Add(details.Text(p.styles.Muted.Render("  No scenes\n")))
		return view
	}

	// Summary
	summary := details.NewFields()
	summary.Add("Total", fmt.Sprintf("%d scenes", len(data.Scenes)))
	view.Add(summary)

	// List scenes with status
	view.Add(details.Blank())
	scenesList := details.NewList()

	for _, scene := range data.Scenes {
		name := hue.SceneName(scene, "")

		isActive := false
		statusStr := ""
		if scene.Status != nil && scene.Status.Active != nil {
			status := *scene.Status.Active
			if status == hueclient.SceneGetStatusActiveStatic {
				isActive = true
				statusStr = "static"
			} else if status == hueclient.SceneGetStatusActiveDynamicPalette {
				isActive = true
				statusStr = "dynamic"
			}
		}

		if isActive {
			var avgBrightness float64
			var avgColor lipgloss.Color = p.styles.Theme.OnColor

			if p.state != nil && scene.Group != nil && scene.Group.Rid != nil {
				var lights []hueclient.LightGet
				if room, ok := p.state.GetRoom(*scene.Group.Rid); ok {
					lights = p.state.RoomLights(room)
				} else if zone, ok := p.state.GetZone(*scene.Group.Rid); ok {
					lights = p.state.RoomLights(zone)
				}

				var totalBrightness, totalR, totalG, totalB float64
				var onCount int
				for _, light := range lights {
					if IsLightOn(light) {
						onCount++
						if light.Dimming != nil && light.Dimming.Brightness != nil {
							totalBrightness += float64(*light.Dimming.Brightness)
						} else {
							totalBrightness += 100
						}
						color := GetLightColor(light)
						colorStr := string(color)
						if len(colorStr) == 7 && colorStr[0] == '#' {
							var r, g, b int
							fmt.Sscanf(colorStr, "#%02x%02x%02x", &r, &g, &b)
							totalR += float64(r)
							totalG += float64(g)
							totalB += float64(b)
						}
					}
				}
				if onCount > 0 {
					avgBrightness = totalBrightness / float64(onCount)
					avgColor = lipgloss.Color(fmt.Sprintf("#%02x%02x%02x",
						int(totalR/float64(onCount)),
						int(totalG/float64(onCount)),
						int(totalB/float64(onCount))))
				}
			}

			indicator := RenderBrightnessIndicator(avgBrightness, avgColor)
			scenesList.AddCustomFull(details.ListItem{
				Bullet: indicator,
				Text:   name,
				Suffix: fmt.Sprintf(" (%s)", statusStr),
			})
		} else {
			indicator := RenderOffIndicator(p.styles)
			scenesList.AddCustomFull(details.ListItem{
				Bullet: indicator,
				Text:   name,
			})
		}
	}
	view.Add(scenesList)

	return view
}
