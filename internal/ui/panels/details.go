package panels

import (
	"fmt"
	"math"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/kluzzebass/lazyhue/internal/hue"
	"github.com/kluzzebass/lazyhue/internal/ui"
	"github.com/openhue/openhue-go"
)

// DetailsPanel shows details for the selected entity.
type DetailsPanel struct {
	viewport viewport.Model
	styles   ui.Styles
	item     *EntityItem
	state    *hue.BridgeState
	width    int
	height   int
}

// NewDetailsPanel creates a new details panel.
func NewDetailsPanel(styles ui.Styles) *DetailsPanel {
	vp := viewport.New(0, 0)
	return &DetailsPanel{
		viewport: vp,
		styles:   styles,
	}
}

// SetItem updates the displayed entity.
func (p *DetailsPanel) SetItem(item *EntityItem, state *hue.BridgeState) {
	p.item = item
	p.state = state
	p.updateContent()
}

// SetSize updates the panel dimensions.
func (p *DetailsPanel) SetSize(width, height int) {
	p.width = width
	p.height = height

	// Content area: width minus borders (2), height minus borders (2)
	p.viewport.Width = max(1, width-2)
	p.viewport.Height = max(1, height-2)
	p.updateContent()
}

func (p *DetailsPanel) updateContent() {
	if p.item == nil {
		p.viewport.SetContent("No selection")
		return
	}

	var content strings.Builder

	title := p.styles.Title.Render(p.item.Name)
	content.WriteString(title)
	content.WriteString("\n")
	content.WriteString(strings.Repeat("─", max(0, min(30, p.width-6))))
	content.WriteString("\n\n")

	switch p.item.Type {
	case EntityRoom:
		if room, ok := GetRoomFromItem(*p.item); ok {
			p.renderRoom(&content, room, false)
		}
	case EntityZone:
		if zone, ok := GetRoomFromItem(*p.item); ok {
			p.renderRoom(&content, zone, true) // Zones use same type as rooms
		}
	case EntityLight:
		if light, ok := GetLightFromItem(*p.item); ok {
			p.renderLight(&content, light)
		}
	case EntityScene:
		if scene, ok := GetSceneFromItem(*p.item); ok {
			p.renderScene(&content, scene)
		}
	case EntityDevice:
		if device, ok := GetDeviceFromItem(*p.item); ok {
			p.renderDevice(&content, device)
		}
	case EntityBridge:
		if bridgeData, ok := GetBridgeFromItem(*p.item); ok {
			p.renderBridge(&content, bridgeData)
		}
	default:
		content.WriteString(p.styles.Muted.Render(fmt.Sprintf("Type: %d", p.item.Type)))
	}

	p.viewport.SetContent(content.String())
}

func (p *DetailsPanel) renderRoom(w *strings.Builder, room openhue.RoomGet, isZone bool) {
	if p.state == nil {
		return
	}

	groupType := "Room"
	if isZone {
		groupType = "Zone"
	}

	// ID
	if room.Id != nil {
		w.WriteString(p.styles.Muted.Render("ID: "))
		w.WriteString(*room.Id)
		w.WriteString("\n")
	}

	// Type
	w.WriteString(p.styles.Muted.Render("Type: "))
	w.WriteString(groupType)
	w.WriteString("\n")

	// Archetype
	if room.Metadata != nil && room.Metadata.Archetype != nil {
		w.WriteString(p.styles.Muted.Render("Archetype: "))
		w.WriteString(string(*room.Metadata.Archetype))
		w.WriteString("\n")
	}

	lights := p.state.RoomLights(room)
	onCount := 0
	for _, l := range lights {
		if l.IsOn() {
			onCount++
		}
	}

	w.WriteString("\n")
	status := fmt.Sprintf("Status: %d/%d lights on", onCount, len(lights))
	w.WriteString(status)
	w.WriteString("\n")

	if gl, ok := p.state.RoomGroupedLight(room); ok {
		if gl.Dimming != nil && gl.Dimming.Brightness != nil {
			brightness := fmt.Sprintf("Brightness: %.0f%%", float64(*gl.Dimming.Brightness))
			w.WriteString(brightness)
			w.WriteString("\n")
		}
	}

	// Lights section
	if len(lights) > 0 {
		w.WriteString("\n")
		w.WriteString(p.styles.Subtitle.Render("Lights:"))
		w.WriteString("\n")

		for _, light := range lights {
			name := ""
			if light.Metadata != nil && light.Metadata.Name != nil {
				name = *light.Metadata.Name
			}
			indicator := p.styles.OnOffIndicator(light.IsOn())
			brightness := ""
			if light.IsOn() {
				if light.Dimming != nil && light.Dimming.Brightness != nil {
					brightness = fmt.Sprintf(" %.0f%%", float64(*light.Dimming.Brightness))
				}
			} else {
				brightness = " Off"
			}
			line := fmt.Sprintf("  %s %s%s", indicator, name, brightness)
			w.WriteString(line)
			w.WriteString("\n")
		}
	}

	// Children (devices) section
	if room.Children != nil && len(*room.Children) > 0 {
		w.WriteString("\n")
		w.WriteString(p.styles.Subtitle.Render("Children:"))
		w.WriteString("\n")
		for _, child := range *room.Children {
			rtype := "unknown"
			rid := ""
			if child.Rtype != nil {
				rtype = string(*child.Rtype)
			}
			if child.Rid != nil {
				rid = *child.Rid
			}
			w.WriteString(fmt.Sprintf("  • %s: %s\n", rtype, p.styles.Muted.Render(rid)))
		}
	}

	// Services section
	if room.Services != nil && len(*room.Services) > 0 {
		w.WriteString("\n")
		w.WriteString(p.styles.Subtitle.Render("Services:"))
		w.WriteString("\n")
		for _, svc := range *room.Services {
			rtype := "unknown"
			rid := ""
			if svc.Rtype != nil {
				rtype = string(*svc.Rtype)
			}
			if svc.Rid != nil {
				rid = *svc.Rid
			}
			w.WriteString(fmt.Sprintf("  • %s: %s\n", rtype, p.styles.Muted.Render(rid)))
		}
	}

	// Scenes section
	roomID := ""
	if room.Id != nil {
		roomID = *room.Id
	}
	scenes := p.state.RoomScenes(roomID)
	if len(scenes) > 0 {
		w.WriteString("\n")
		w.WriteString(p.styles.Subtitle.Render("Scenes:"))
		w.WriteString("\n")
		for _, scene := range scenes {
			name := ""
			if scene.Metadata != nil && scene.Metadata.Name != nil {
				name = *scene.Metadata.Name
			}
			w.WriteString(fmt.Sprintf("  • %s\n", name))
		}
	}
}

func (p *DetailsPanel) renderLight(w *strings.Builder, light openhue.LightGet) {
	// Get owning device info for product details
	var device *openhue.DeviceGet
	if p.state != nil && light.Owner != nil && light.Owner.Rid != nil {
		if d, ok := p.state.GetDevice(*light.Owner.Rid); ok {
			device = &d
		}
	}

	// Product info from device
	if device != nil {
		if device.ProductData != nil {
			if device.ProductData.ProductName != nil {
				w.WriteString(p.styles.Muted.Render("Product: "))
				w.WriteString(*device.ProductData.ProductName)
				w.WriteString("\n")
			}
			if device.ProductData.ManufacturerName != nil {
				w.WriteString(p.styles.Muted.Render("Manufacturer: "))
				w.WriteString(*device.ProductData.ManufacturerName)
				w.WriteString("\n")
			}
			if device.ProductData.ModelId != nil {
				w.WriteString(p.styles.Muted.Render("Model: "))
				w.WriteString(*device.ProductData.ModelId)
				w.WriteString("\n")
			}
			if device.ProductData.ProductArchetype != nil {
				w.WriteString(p.styles.Muted.Render("Archetype: "))
				w.WriteString(string(*device.ProductData.ProductArchetype))
				w.WriteString("\n")
			}
			if device.ProductData.SoftwareVersion != nil {
				w.WriteString(p.styles.Muted.Render("Firmware: "))
				w.WriteString(*device.ProductData.SoftwareVersion)
				w.WriteString("\n")
			}
			if device.ProductData.HardwarePlatformType != nil {
				w.WriteString(p.styles.Muted.Render("Hardware: "))
				w.WriteString(*device.ProductData.HardwarePlatformType)
				w.WriteString("\n")
			}
		}
		w.WriteString("\n")
	}

	// Light ID
	if light.Id != nil {
		w.WriteString(p.styles.Muted.Render("Light ID: "))
		w.WriteString(*light.Id)
		w.WriteString("\n")
	}

	// V1 ID (for legacy API compatibility)
	if light.IdV1 != nil {
		w.WriteString(p.styles.Muted.Render("V1 ID: "))
		w.WriteString(*light.IdV1)
		w.WriteString("\n")
	}

	// Type
	if light.Type != nil {
		w.WriteString(p.styles.Muted.Render("Type: "))
		w.WriteString(string(*light.Type))
		w.WriteString("\n")
	}

	// Mode
	if light.Mode != nil {
		w.WriteString(p.styles.Muted.Render("Mode: "))
		w.WriteString(string(*light.Mode))
		w.WriteString("\n")
	}

	// Status
	indicator := p.styles.OnOffIndicator(light.IsOn())
	status := "Off"
	if light.IsOn() {
		status = "On"
	}
	w.WriteString(fmt.Sprintf("\nStatus: %s %s\n", indicator, status))

	// Dimming
	if light.Dimming != nil {
		if light.Dimming.Brightness != nil {
			w.WriteString(fmt.Sprintf("Brightness: %.0f%%\n", float64(*light.Dimming.Brightness)))
		}
		if light.Dimming.MinDimLevel != nil {
			w.WriteString(fmt.Sprintf("Min dim level: %.0f%%\n", float64(*light.Dimming.MinDimLevel)))
		}
	}

	// Color
	if light.Color != nil && light.Color.Xy != nil {
		w.WriteString(fmt.Sprintf("Color XY: (%.4f, %.4f)\n", *light.Color.Xy.X, *light.Color.Xy.Y))
		if light.Color.Gamut != nil {
			w.WriteString(p.styles.Muted.Render("Gamut: "))
			if light.Color.GamutType != nil {
				w.WriteString(string(*light.Color.GamutType))
			}
			w.WriteString("\n")
		}
	}

	// Color Temperature
	if light.ColorTemperature != nil {
		if light.ColorTemperature.Mirek != nil {
			mirek := *light.ColorTemperature.Mirek
			kelvin := 1000000 / int(mirek) // Convert mirek to Kelvin
			w.WriteString(fmt.Sprintf("Color temp: %d mirek (~%dK)\n", mirek, kelvin))
		}
		if light.ColorTemperature.MirekSchema != nil {
			if light.ColorTemperature.MirekSchema.MirekMinimum != nil && light.ColorTemperature.MirekSchema.MirekMaximum != nil {
				minK := 1000000 / int(*light.ColorTemperature.MirekSchema.MirekMaximum)
				maxK := 1000000 / int(*light.ColorTemperature.MirekSchema.MirekMinimum)
				w.WriteString(fmt.Sprintf("CT range: %dK - %dK\n", minK, maxK))
			}
		}
	}

	// Dynamics
	if light.Dynamics != nil {
		w.WriteString("\n")
		w.WriteString(p.styles.Subtitle.Render("Dynamics:"))
		w.WriteString("\n")
		if light.Dynamics.Status != nil {
			w.WriteString(fmt.Sprintf("  Status: %s\n", *light.Dynamics.Status))
		}
		if light.Dynamics.Speed != nil {
			w.WriteString(fmt.Sprintf("  Speed: %.2f\n", *light.Dynamics.Speed))
		}
	}

	// Capabilities
	w.WriteString("\n")
	w.WriteString(p.styles.Subtitle.Render("Capabilities:"))
	w.WriteString("\n")

	caps := []string{}
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

	if len(caps) > 0 {
		for _, cap := range caps {
			w.WriteString(fmt.Sprintf("  • %s\n", cap))
		}
	} else {
		w.WriteString("  On/Off only\n")
	}

	// Effects
	if light.Effects != nil && light.Effects.EffectValues != nil && len(*light.Effects.EffectValues) > 0 {
		w.WriteString("\n")
		w.WriteString(p.styles.Subtitle.Render("Available Effects:"))
		w.WriteString("\n")
		for _, effect := range *light.Effects.EffectValues {
			w.WriteString(fmt.Sprintf("  • %s\n", effect))
		}
	}

	// Gradient (for gradient-capable lights)
	if light.Gradient != nil {
		w.WriteString("\n")
		w.WriteString(p.styles.Subtitle.Render("Gradient:"))
		w.WriteString("\n")
		if light.Gradient.Mode != nil {
			w.WriteString(fmt.Sprintf("  Mode: %s\n", *light.Gradient.Mode))
		}
		if light.Gradient.PixelCount != nil {
			w.WriteString(fmt.Sprintf("  Pixels: %d\n", *light.Gradient.PixelCount))
		}
		if light.Gradient.Points != nil && len(*light.Gradient.Points) > 0 {
			w.WriteString(fmt.Sprintf("  Points: %d\n", len(*light.Gradient.Points)))
		}
	}

	// Signaling
	if light.Signaling != nil && light.Signaling.SignalValues != nil && len(*light.Signaling.SignalValues) > 0 {
		w.WriteString("\n")
		w.WriteString(p.styles.Subtitle.Render("Signaling Modes:"))
		w.WriteString("\n")
		for _, sig := range *light.Signaling.SignalValues {
			w.WriteString(fmt.Sprintf("  • %s\n", sig))
		}
	}

	// Powerup behavior
	if light.Powerup != nil && light.Powerup.Preset != nil {
		w.WriteString("\n")
		w.WriteString(p.styles.Muted.Render("Power-on behavior: "))
		w.WriteString(string(*light.Powerup.Preset))
		w.WriteString("\n")
	}

	// Device services (sibling services on the owning device)
	if device != nil && device.Services != nil && len(*device.Services) > 0 {
		w.WriteString("\n")
		w.WriteString(p.styles.Subtitle.Render("Device Services:"))
		w.WriteString("\n")
		for _, svc := range *device.Services {
			rtype := "unknown"
			if svc.Rtype != nil {
				rtype = string(*svc.Rtype)
			}
			// Mark the current light service
			if svc.Rid != nil && light.Id != nil && *svc.Rid == *light.Id {
				w.WriteString(fmt.Sprintf("  • %s %s\n", rtype, p.styles.Muted.Render("(this)")))
			} else {
				w.WriteString(fmt.Sprintf("  • %s\n", rtype))
			}
		}
	}

	// Owner device ID (for reference)
	if light.Owner != nil && light.Owner.Rid != nil {
		w.WriteString("\n")
		w.WriteString(p.styles.Muted.Render("Device ID: "))
		w.WriteString(*light.Owner.Rid)
		w.WriteString("\n")
	}
}

func (p *DetailsPanel) renderScene(w *strings.Builder, scene openhue.SceneGet) {
	// ID
	if scene.Id != nil {
		w.WriteString(p.styles.Muted.Render("ID: "))
		w.WriteString(*scene.Id)
		w.WriteString("\n")
	}

	// V1 ID
	if scene.IdV1 != nil {
		w.WriteString(p.styles.Muted.Render("V1 ID: "))
		w.WriteString(*scene.IdV1)
		w.WriteString("\n")
	}

	// Type
	if scene.Type != nil {
		w.WriteString(p.styles.Muted.Render("Type: "))
		w.WriteString(string(*scene.Type))
		w.WriteString("\n")
	}

	// Status
	if scene.Status != nil && scene.Status.Active != nil {
		w.WriteString(p.styles.Muted.Render("Active: "))
		w.WriteString(string(*scene.Status.Active))
		w.WriteString("\n")
	}

	// Owner
	if scene.Owner != nil && scene.Owner.Rid != nil {
		w.WriteString(p.styles.Muted.Render("Owner: "))
		ownerName := *scene.Owner.Rid
		if scene.Owner.Rtype != nil {
			ownerName = fmt.Sprintf("%s (%s)", ownerName, string(*scene.Owner.Rtype))
		}
		w.WriteString(ownerName)
		w.WriteString("\n")
	}

	// App data
	if scene.Metadata != nil && scene.Metadata.Appdata != nil && *scene.Metadata.Appdata != "" {
		w.WriteString(p.styles.Muted.Render("App data: "))
		w.WriteString(*scene.Metadata.Appdata)
		w.WriteString("\n")
	}

	// Group (room/zone)
	if scene.Group != nil && scene.Group.Rid != nil {
		w.WriteString("\n")
		w.WriteString(p.styles.Subtitle.Render("Group:"))
		w.WriteString("\n")
		rtype := "unknown"
		if scene.Group.Rtype != nil {
			rtype = string(*scene.Group.Rtype)
		}
		w.WriteString(fmt.Sprintf("  Type: %s\n", rtype))
		w.WriteString(fmt.Sprintf("  ID: %s\n", p.styles.Muted.Render(*scene.Group.Rid)))

		// Try to get group name
		if p.state != nil {
			if room, ok := p.state.GetRoom(*scene.Group.Rid); ok {
				if room.Metadata != nil && room.Metadata.Name != nil {
					w.WriteString(fmt.Sprintf("  Name: %s\n", *room.Metadata.Name))
				}
			} else if zone, ok := p.state.GetZone(*scene.Group.Rid); ok {
				if zone.Metadata != nil && zone.Metadata.Name != nil {
					w.WriteString(fmt.Sprintf("  Name: %s (zone)\n", *zone.Metadata.Name))
				}
			}
		}
	}

	// Speed
	if scene.Speed != nil {
		w.WriteString(fmt.Sprintf("\nTransition speed: %.2f\n", *scene.Speed))
	}

	// Auto dynamic
	if scene.AutoDynamic != nil {
		w.WriteString(fmt.Sprintf("Auto dynamic: %v\n", *scene.AutoDynamic))
	}

	// Palette
	if scene.Palette != nil {
		hasPalette := (scene.Palette.Color != nil && len(*scene.Palette.Color) > 0) ||
			(scene.Palette.Dimming != nil && len(*scene.Palette.Dimming) > 0) ||
			(scene.Palette.ColorTemperature != nil && len(*scene.Palette.ColorTemperature) > 0) ||
			(scene.Palette.Effects != nil && len(*scene.Palette.Effects) > 0)

		if hasPalette {
			w.WriteString("\n")
			w.WriteString(p.styles.Subtitle.Render("Palette:"))
			w.WriteString("\n")

			if scene.Palette.Color != nil && len(*scene.Palette.Color) > 0 {
				w.WriteString(fmt.Sprintf("  Colors: %d\n", len(*scene.Palette.Color)))
				for i, c := range *scene.Palette.Color {
					if c.Color != nil && c.Color.Xy != nil {
						w.WriteString(fmt.Sprintf("    %d: XY(%.4f, %.4f)\n", i+1, *c.Color.Xy.X, *c.Color.Xy.Y))
					}
				}
			}
			if scene.Palette.Dimming != nil && len(*scene.Palette.Dimming) > 0 {
				w.WriteString(fmt.Sprintf("  Dimming levels: %d\n", len(*scene.Palette.Dimming)))
				for i, d := range *scene.Palette.Dimming {
					if d.Brightness != nil {
						w.WriteString(fmt.Sprintf("    %d: %.0f%%\n", i+1, *d.Brightness))
					}
				}
			}
			if scene.Palette.ColorTemperature != nil && len(*scene.Palette.ColorTemperature) > 0 {
				w.WriteString(fmt.Sprintf("  Color temps: %d\n", len(*scene.Palette.ColorTemperature)))
				for i, ct := range *scene.Palette.ColorTemperature {
					if ct.ColorTemperature != nil && ct.ColorTemperature.Mirek != nil {
						w.WriteString(fmt.Sprintf("    %d: %d mirek\n", i+1, *ct.ColorTemperature.Mirek))
					}
				}
			}
			if scene.Palette.Effects != nil && len(*scene.Palette.Effects) > 0 {
				w.WriteString(fmt.Sprintf("  Effects: %d\n", len(*scene.Palette.Effects)))
				for i, e := range *scene.Palette.Effects {
					if e.Effect != nil {
						w.WriteString(fmt.Sprintf("    %d: %s\n", i+1, string(*e.Effect)))
					}
				}
			}
		}
	}

	// Actions
	if scene.Actions != nil && len(*scene.Actions) > 0 {
		w.WriteString("\n")
		w.WriteString(p.styles.Subtitle.Render(fmt.Sprintf("Actions (%d):", len(*scene.Actions))))
		w.WriteString("\n")
		for _, action := range *scene.Actions {
			targetName := "unknown"
			if action.Target != nil && action.Target.Rid != nil {
				targetName = *action.Target.Rid
				// Try to get friendly name from owning device
				if p.state != nil {
					if light, ok := p.state.GetLight(*action.Target.Rid); ok {
						// Get name from owning device (preferred)
						if light.Owner != nil && light.Owner.Rid != nil {
							if device, ok := p.state.GetDevice(*light.Owner.Rid); ok {
								if device.Metadata != nil && device.Metadata.Name != nil {
									targetName = *device.Metadata.Name
								}
							}
						}
						// Fallback to light's own metadata
						if targetName == *action.Target.Rid && light.Metadata != nil && light.Metadata.Name != nil {
							targetName = *light.Metadata.Name
						}
					}
				}
			}
			w.WriteString(fmt.Sprintf("  • %s\n", targetName))

			// Show action details
			if action.Action != nil {
				if action.Action.On != nil && action.Action.On.On != nil {
					state := "off"
					if *action.Action.On.On {
						state = "on"
					}
					w.WriteString(fmt.Sprintf("      State: %s\n", state))
				}
				if action.Action.Dimming != nil && action.Action.Dimming.Brightness != nil {
					w.WriteString(fmt.Sprintf("      Brightness: %.0f%%\n", *action.Action.Dimming.Brightness))
				}
				if action.Action.ColorTemperature != nil && action.Action.ColorTemperature.Mirek != nil {
					w.WriteString(fmt.Sprintf("      Color temp: %d mirek\n", *action.Action.ColorTemperature.Mirek))
				}
				if action.Action.Color != nil && action.Action.Color.Xy != nil {
					w.WriteString(fmt.Sprintf("      Color XY: (%.4f, %.4f)\n", *action.Action.Color.Xy.X, *action.Action.Color.Xy.Y))
				}
				if action.Action.Effects != nil && action.Action.Effects.Effect != nil {
					w.WriteString(fmt.Sprintf("      Effect: %s\n", string(*action.Action.Effects.Effect)))
				}
				if action.Action.Gradient != nil && action.Action.Gradient.Points != nil {
					w.WriteString(fmt.Sprintf("      Gradient: %d points\n", len(*action.Action.Gradient.Points)))
				}
			}
		}
	}

	// Metadata image
	if scene.Metadata != nil && scene.Metadata.Image != nil && scene.Metadata.Image.Rid != nil {
		w.WriteString("\n")
		w.WriteString(p.styles.Muted.Render("Image: "))
		w.WriteString(*scene.Metadata.Image.Rid)
		w.WriteString("\n")
	}
}

func (p *DetailsPanel) renderDevice(w *strings.Builder, device openhue.DeviceGet) {
	// ID
	if device.Id != nil {
		w.WriteString(p.styles.Muted.Render("ID: "))
		w.WriteString(*device.Id)
		w.WriteString("\n")
	}

	// Type
	if device.Type != nil {
		w.WriteString(p.styles.Muted.Render("Type: "))
		w.WriteString(string(*device.Type))
		w.WriteString("\n")
	}

	// Product data
	if device.ProductData != nil {
		w.WriteString("\n")
		w.WriteString(p.styles.Subtitle.Render("Product:"))
		w.WriteString("\n")

		if device.ProductData.ManufacturerName != nil {
			w.WriteString(fmt.Sprintf("  Manufacturer: %s\n", *device.ProductData.ManufacturerName))
		}
		if device.ProductData.ProductName != nil {
			w.WriteString(fmt.Sprintf("  Product: %s\n", *device.ProductData.ProductName))
		}
		if device.ProductData.ModelId != nil {
			w.WriteString(fmt.Sprintf("  Model ID: %s\n", *device.ProductData.ModelId))
		}
		if device.ProductData.ProductArchetype != nil {
			w.WriteString(fmt.Sprintf("  Archetype: %s\n", string(*device.ProductData.ProductArchetype)))
		}
		if device.ProductData.SoftwareVersion != nil {
			w.WriteString(fmt.Sprintf("  Firmware: %s\n", *device.ProductData.SoftwareVersion))
		}
		if device.ProductData.HardwarePlatformType != nil {
			w.WriteString(fmt.Sprintf("  Platform: %s\n", *device.ProductData.HardwarePlatformType))
		}
		if device.ProductData.Certified != nil {
			w.WriteString(fmt.Sprintf("  Certified: %v\n", *device.ProductData.Certified))
		}
	}

	// Metadata
	if device.Metadata != nil {
		if device.Metadata.Archetype != nil {
			w.WriteString("\n")
			w.WriteString(p.styles.Muted.Render("Archetype: "))
			w.WriteString(string(*device.Metadata.Archetype))
			w.WriteString("\n")
		}
	}

	// Sensor readings (if applicable)
	if p.state != nil {
		hasSensors := false

		// Motion sensor
		hasMotion, isDetecting := p.state.GetDeviceMotionState(device)
		if hasMotion {
			if !hasSensors {
				w.WriteString("\n")
				w.WriteString(p.styles.Subtitle.Render("Sensors:"))
				w.WriteString("\n")
				hasSensors = true
			}
			if isDetecting {
				w.WriteString(fmt.Sprintf("  Motion: %s Detected\n", p.styles.OnOffIndicator(true)))
			} else {
				w.WriteString(fmt.Sprintf("  Motion: %s None\n", p.styles.OnOffIndicator(false)))
			}
		}

		// Temperature sensor
		hasTemp, tempC := p.state.GetDeviceTemperature(device)
		if hasTemp {
			if !hasSensors {
				w.WriteString("\n")
				w.WriteString(p.styles.Subtitle.Render("Sensors:"))
				w.WriteString("\n")
				hasSensors = true
			}
			w.WriteString(fmt.Sprintf("  Temperature: %.1f C\n", tempC))
		}

		// Light level sensor
		hasLevel, level := p.state.GetDeviceLightLevel(device)
		if hasLevel {
			if !hasSensors {
				w.WriteString("\n")
				w.WriteString(p.styles.Subtitle.Render("Sensors:"))
				w.WriteString("\n")
				hasSensors = true
			}
			// Convert from 10000*log10(lux)+1 to approximate lux
			// level = 10000 * log10(lux) + 1
			// lux = 10^((level-1)/10000)
			lux := 0.0
			if level > 1 {
				lux = math.Pow(10, float64(level-1)/10000.0)
			}
			w.WriteString(fmt.Sprintf("  Light level: %.0f lux\n", lux))
		}

		// Battery status
		hasBattery, battLevel, battState := p.state.GetDeviceBattery(device)
		if hasBattery {
			if !hasSensors {
				w.WriteString("\n")
				w.WriteString(p.styles.Subtitle.Render("Sensors:"))
				w.WriteString("\n")
				hasSensors = true
			}
			stateStr := ""
			if battState != "" {
				stateStr = fmt.Sprintf(" (%s)", battState)
			}
			w.WriteString(fmt.Sprintf("  Battery: %d%%%s\n", battLevel, stateStr))
		}
	}

	// Services
	if device.Services != nil && len(*device.Services) > 0 {
		w.WriteString("\n")
		w.WriteString(p.styles.Subtitle.Render("Services:"))
		w.WriteString("\n")
		for _, svc := range *device.Services {
			rtype := "unknown"
			rid := ""
			if svc.Rtype != nil {
				rtype = string(*svc.Rtype)
			}
			if svc.Rid != nil {
				rid = *svc.Rid
			}
			w.WriteString(fmt.Sprintf("  • %s\n", rtype))
			w.WriteString(fmt.Sprintf("    %s\n", p.styles.Muted.Render(rid)))
		}
	}
}

func (p *DetailsPanel) renderBridge(w *strings.Builder, data BridgeData) {
	bridge := data.Bridge
	if bridge == nil {
		return
	}

	state := bridge.GetState()

	// Connection status
	w.WriteString(p.styles.Muted.Render("Status: "))
	switch bridge.Status {
	case hue.StatusConnected:
		w.WriteString(p.styles.Success.Render("Connected"))
	case hue.StatusConnecting:
		w.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#ffff00")).Render("Connecting..."))
	case hue.StatusPairing:
		w.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#ffff00")).Render("Pairing..."))
	case hue.StatusError:
		w.WriteString(p.styles.Error.Render("Error"))
	default:
		w.WriteString(p.styles.Muted.Render("Disconnected"))
	}
	w.WriteString("\n")

	// Network info
	w.WriteString("\n")
	w.WriteString(p.styles.Subtitle.Render("Network:"))
	w.WriteString("\n")
	w.WriteString(fmt.Sprintf("  IP Address: %s\n", bridge.Info.IPAddress))
	w.WriteString(fmt.Sprintf("  Bridge ID: %s\n", bridge.Info.ID))

	// Last sync time
	if !bridge.LastSync.IsZero() {
		w.WriteString(fmt.Sprintf("  Last sync: %s\n", bridge.LastSync.Format("15:04:05")))
	}

	// Error info if any
	if bridge.LastErr != nil {
		w.WriteString("\n")
		w.WriteString(p.styles.Error.Render("Last error: "))
		w.WriteString(bridge.LastErr.Error())
		w.WriteString("\n")
	}

	// Bridge resource info
	if state != nil {
		bridgeRes := state.GetBridgeResource()
		if bridgeRes != nil {
			w.WriteString("\n")
			w.WriteString(p.styles.Subtitle.Render("Bridge Resource:"))
			w.WriteString("\n")

			if bridgeRes.Id != nil {
				w.WriteString(fmt.Sprintf("  Resource ID: %s\n", *bridgeRes.Id))
			}
			if bridgeRes.BridgeId != nil {
				w.WriteString(fmt.Sprintf("  Bridge ID: %s\n", *bridgeRes.BridgeId))
			}
			if bridgeRes.IdV1 != nil {
				w.WriteString(fmt.Sprintf("  V1 ID: %s\n", *bridgeRes.IdV1))
			}
			if bridgeRes.TimeZone != nil && bridgeRes.TimeZone.TimeZone != nil {
				w.WriteString(fmt.Sprintf("  Timezone: %s\n", *bridgeRes.TimeZone.TimeZone))
			}
		}

		// Bridge device info (the physical bridge)
		if bridgeDevice, ok := state.GetBridgeDevice(); ok {
			w.WriteString("\n")
			w.WriteString(p.styles.Subtitle.Render("Hardware:"))
			w.WriteString("\n")

			if bridgeDevice.ProductData != nil {
				if bridgeDevice.ProductData.ManufacturerName != nil {
					w.WriteString(fmt.Sprintf("  Manufacturer: %s\n", *bridgeDevice.ProductData.ManufacturerName))
				}
				if bridgeDevice.ProductData.ProductName != nil {
					w.WriteString(fmt.Sprintf("  Product: %s\n", *bridgeDevice.ProductData.ProductName))
				}
				if bridgeDevice.ProductData.ModelId != nil {
					w.WriteString(fmt.Sprintf("  Model: %s\n", *bridgeDevice.ProductData.ModelId))
				}
				if bridgeDevice.ProductData.HardwarePlatformType != nil {
					w.WriteString(fmt.Sprintf("  Platform: %s\n", *bridgeDevice.ProductData.HardwarePlatformType))
				}
				if bridgeDevice.ProductData.SoftwareVersion != nil {
					w.WriteString(fmt.Sprintf("  Firmware: %s\n", *bridgeDevice.ProductData.SoftwareVersion))
				}
				if bridgeDevice.ProductData.Certified != nil {
					w.WriteString(fmt.Sprintf("  Certified: %v\n", *bridgeDevice.ProductData.Certified))
				}
			}

			// Bridge device services
			if bridgeDevice.Services != nil && len(*bridgeDevice.Services) > 0 {
				w.WriteString("\n")
				w.WriteString(p.styles.Subtitle.Render("Bridge Services:"))
				w.WriteString("\n")
				for _, svc := range *bridgeDevice.Services {
					rtype := "unknown"
					if svc.Rtype != nil {
						rtype = string(*svc.Rtype)
					}
					w.WriteString(fmt.Sprintf("  • %s\n", rtype))
				}
			}
		}

		// Bridge home info
		bridgeHome := state.GetBridgeHome()
		if bridgeHome != nil {
			w.WriteString("\n")
			w.WriteString(p.styles.Subtitle.Render("Home:"))
			w.WriteString("\n")

			if bridgeHome.Id != nil {
				w.WriteString(fmt.Sprintf("  Home ID: %s\n", *bridgeHome.Id))
			}
			if bridgeHome.IdV1 != nil {
				w.WriteString(fmt.Sprintf("  V1 ID: %s\n", *bridgeHome.IdV1))
			}

			// Children - count by type and show names
			if bridgeHome.Children != nil && len(*bridgeHome.Children) > 0 {
				w.WriteString(fmt.Sprintf("  Children: %d devices\n", len(*bridgeHome.Children)))

				// Count by type and collect names
				typeCounts := make(map[string]int)
				for _, child := range *bridgeHome.Children {
					rtype := "unknown"
					if child.Rtype != nil {
						rtype = string(*child.Rtype)
					}
					typeCounts[rtype]++
				}

				// Sort type names for stable output
				typeNames := make([]string, 0, len(typeCounts))
				for rtype := range typeCounts {
					typeNames = append(typeNames, rtype)
				}
				sort.Strings(typeNames)

				// Show breakdown by type
				for _, rtype := range typeNames {
					w.WriteString(fmt.Sprintf("    %s: %d\n", rtype, typeCounts[rtype]))
				}
			}

			// Services
			if bridgeHome.Services != nil && len(*bridgeHome.Services) > 0 {
				w.WriteString(fmt.Sprintf("  Services: %d\n", len(*bridgeHome.Services)))
			}
		}

		// Summary statistics
		w.WriteString("\n")
		w.WriteString(p.styles.Subtitle.Render("Summary:"))
		w.WriteString("\n")

		rooms := state.AllRooms()
		zones := state.AllZones()
		lights := state.AllLights()
		scenes := state.AllScenes()
		devices := state.AllDevices()

		w.WriteString(fmt.Sprintf("  Rooms: %d\n", len(rooms)))
		w.WriteString(fmt.Sprintf("  Zones: %d\n", len(zones)))
		w.WriteString(fmt.Sprintf("  Lights: %d\n", len(lights)))
		w.WriteString(fmt.Sprintf("  Scenes: %d\n", len(scenes)))
		w.WriteString(fmt.Sprintf("  Devices: %d\n", len(devices)))

		// Count lights that are on
		lightsOn := 0
		for _, light := range lights {
			if light.IsOn() {
				lightsOn++
			}
		}
		w.WriteString(fmt.Sprintf("  Lights on: %d/%d\n", lightsOn, len(lights)))

		// Authenticated applications (from V1 API whitelist)
		authApps := state.GetAuthApps()
		if len(authApps) > 0 {
			w.WriteString("\n")
			w.WriteString(p.styles.Subtitle.Render(fmt.Sprintf("Authenticated Apps (%d):", len(authApps))))
			w.WriteString("\n")
			for _, app := range authApps {
				w.WriteString(fmt.Sprintf("  • %s\n", app.AppName))
				if app.LastUseDate != "" {
					w.WriteString(fmt.Sprintf("    Last used: %s\n", app.LastUseDate))
				}
			}
		}
	}
}

// Update handles input for the details panel.
func (p *DetailsPanel) Update(msg tea.Msg) (*DetailsPanel, tea.Cmd) {
	// Handle g/G for top/bottom (viewport uses home/end)
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		switch keyMsg.String() {
		case "g":
			p.viewport.GotoTop()
			return p, nil
		case "G":
			p.viewport.GotoBottom()
			return p, nil
		}
	}

	var cmd tea.Cmd
	p.viewport, cmd = p.viewport.Update(msg)
	return p, cmd
}

// View renders the details panel.
func (p *DetailsPanel) View(active bool) string {
	content := p.viewport.View()

	// Count content lines for scroll indicator
	contentLines := strings.Count(p.viewport.View(), "\n") + 1
	totalLines := p.viewport.TotalLineCount()

	cfg := ui.BorderConfig{
		Title:       "[0] Details",
		ScrollPos:   p.viewport.YOffset,
		TotalHeight: totalLines,
		ViewHeight:  contentLines,
	}

	return ui.RenderBorderedPanel(content, p.width, p.height, active, p.styles, cfg)
}
