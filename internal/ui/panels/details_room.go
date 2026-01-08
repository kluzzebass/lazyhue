package panels

import (
	"fmt"
	"math"
	"strings"

	"github.com/kluzzebass/lazyhue/internal/hueclient"
	"github.com/kluzzebass/lazyhue/internal/ui/panels/details"
)

// buildRoomView creates a details view for a room or zone entity.
func (p *DetailsPanel) buildRoomView(roomAny interface{}, isZone bool) *details.View {
	room, ok := roomAny.(hueclient.RoomGet)
	if !ok || p.state == nil {
		return details.NewView(p.styles)
	}

	view := details.NewView(p.styles)

	groupType := "room"
	if isZone {
		groupType = "zone"
	}

	// Basic info section
	ids := details.NewFields()
	if room.Id != nil {
		ids.AddMuted("ID", *room.Id)
	}
	if room.IdV1 != nil && *room.IdV1 != "" {
		ids.AddMuted("ID (v1)", *room.IdV1)
	}
	ids.AddMuted("Type", groupType)
	if room.Metadata != nil && room.Metadata.Archetype != nil {
		ids.AddMuted("Archetype", string(*room.Metadata.Archetype))
	}
	view.Add(ids)

	// Status section
	lights := p.state.RoomLights(room)
	onCount := 0
	for _, l := range lights {
		if IsLightOn(l) {
			onCount++
		}
	}
	view.Add(details.Blank())
	status := details.NewFields()
	status.Add("Status", fmt.Sprintf("%d/%d lights on", onCount, len(lights)))
	if gl, ok := p.state.RoomGroupedLight(room); ok {
		if gl.Dimming != nil && gl.Dimming.Brightness != nil {
			status.Add("Brightness", fmt.Sprintf("%.0f%%", float64(*gl.Dimming.Brightness)))
		}
	}
	view.Add(status)

	// Lights section
	if len(lights) > 0 {
		view.Add(details.Header("Lights"))
		lightsList := details.NewList()
		for _, light := range lights {
			name := p.state.GetLightName(light)
			indicator := RenderLightIndicatorFromLight(light, p.styles)

			var detailParts []string
			if IsLightOn(light) {
				if light.Dimming != nil && light.Dimming.Brightness != nil {
					detailParts = append(detailParts, fmt.Sprintf("%.0f%%", float64(*light.Dimming.Brightness)))
				}
				if light.ColorTemperature != nil && light.ColorTemperature.Mirek != nil {
					mirek := *light.ColorTemperature.Mirek
					kelvin := 1000000 / mirek
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
	}

	// Devices section (non-light devices only)
	if room.Children != nil && len(*room.Children) > 0 {
		var nonLightDevices []string
		for _, child := range *room.Children {
			if child.Rid == nil || child.Rtype == nil || *child.Rtype != hueclient.ResourceIdentifierRtypeDevice {
				continue
			}
			device, ok := p.state.GetDevice(*child.Rid)
			if !ok {
				continue
			}
			// Skip light devices
			isLight := false
			if device.Services != nil {
				for _, svc := range *device.Services {
					if svc.Rtype != nil && *svc.Rtype == hueclient.ResourceIdentifierRtypeLight {
						isLight = true
						break
					}
				}
			}
			if isLight {
				continue
			}
			name := *child.Rid
			if device.Metadata != nil && device.Metadata.Name != nil {
				name = *device.Metadata.Name
			}
			nonLightDevices = append(nonLightDevices, name)
		}

		if len(nonLightDevices) > 0 {
			view.Add(details.Header(fmt.Sprintf("Devices (%d)", len(nonLightDevices))))
			devicesList := details.NewList()
			for _, name := range nonLightDevices {
				devicesList.Add(name)
			}
			view.Add(devicesList)
		}
	}

	// Scenes section
	roomID := ""
	if room.Id != nil {
		roomID = *room.Id
	}
	scenes := p.state.RoomScenes(roomID)
	if len(scenes) > 0 {
		view.Add(details.Header("Scenes"))
		scenesList := details.NewList()
		for _, scene := range scenes {
			name := ""
			if scene.Metadata != nil && scene.Metadata.Name != nil {
				name = *scene.Metadata.Name
			}
			scenesList.Add(name)
		}
		view.Add(scenesList)
	}

	// Services section
	if room.Services != nil && len(*room.Services) > 0 {
		view.Add(details.Header("Services"))
		servicesList := details.NewList()

		for _, svc := range *room.Services {
			if svc.Rtype == nil || svc.Rid == nil {
				continue
			}
			rtype := *svc.Rtype
			rid := *svc.Rid
			rtypeStr := string(rtype)

			switch rtype {
			case hueclient.ResourceIdentifierRtypeGroupedLight:
				if gl, ok := p.state.GetGroupedLight(rid); ok {
					state := "off"
					if gl.On != nil && gl.On.On != nil && *gl.On.On {
						state = "on"
						if gl.Dimming != nil && gl.Dimming.Brightness != nil {
							state = fmt.Sprintf("on, %.0f%%", *gl.Dimming.Brightness)
						}
					}
					servicesList.Add(fmt.Sprintf("Grouped Light: %s", state))
				} else {
					servicesList.Add("Grouped Light")
				}

			case hueclient.ResourceIdentifierRtypeLightLevel:
				status := ""
				if ll, ok := p.state.GetLightLevel(rid); ok && ll.Light != nil {
					var level int
					if ll.Light.LightLevelReport != nil && ll.Light.LightLevelReport.LightLevel != nil {
						level = *ll.Light.LightLevelReport.LightLevel
					} else if ll.Light.LightLevel != nil {
						level = *ll.Light.LightLevel
					}
					if level > 0 {
						lux := math.Pow(10, float64(level-1)/10000)
						status = fmt.Sprintf("%.0f lux", lux)
					}
				}
				if status != "" {
					servicesList.Add(fmt.Sprintf("Light Level: %s", status))
				} else {
					servicesList.Add("Light Level")
				}

			case hueclient.ResourceIdentifierRtypeMotion:
				status := ""
				if m, ok := p.state.GetMotion(rid); ok && m.Motion != nil {
					detected := false
					if m.Motion.MotionReport != nil && m.Motion.MotionReport.Motion != nil {
						detected = *m.Motion.MotionReport.Motion
					} else if m.Motion.Motion != nil {
						detected = *m.Motion.Motion
					}
					if detected {
						status = "detected"
					} else {
						status = "clear"
					}
				}
				if status != "" {
					servicesList.Add(fmt.Sprintf("Motion: %s", status))
				} else {
					servicesList.Add("Motion")
				}

			default:
				if rtypeStr == "grouped_motion" {
					status := "no sensors"
					if room.Children != nil {
						for _, child := range *room.Children {
							if child.Rid == nil || child.Rtype == nil || *child.Rtype != hueclient.ResourceIdentifierRtypeDevice {
								continue
							}
							if device, ok := p.state.GetDevice(*child.Rid); ok {
								if hasMotion, isDetecting := p.state.GetDeviceMotionState(device); hasMotion {
									if isDetecting {
										status = "detected"
										break
									}
									status = "clear"
								}
							}
						}
					}
					servicesList.Add(fmt.Sprintf("Grouped Motion: %s", status))
					continue
				}

				if rtypeStr == "grouped_light_level" {
					var totalLux float64
					var count int
					if room.Children != nil {
						for _, child := range *room.Children {
							if child.Rid == nil || child.Rtype == nil || *child.Rtype != hueclient.ResourceIdentifierRtypeDevice {
								continue
							}
							if device, ok := p.state.GetDevice(*child.Rid); ok {
								if hasLevel, level := p.state.GetDeviceLightLevel(device); hasLevel && level > 0 {
									totalLux += math.Pow(10, float64(level-1)/10000)
									count++
								}
							}
						}
					}
					status := "no sensors"
					if count > 0 {
						status = fmt.Sprintf("%.0f lux", totalLux/float64(count))
					}
					servicesList.Add(fmt.Sprintf("Grouped Light Level: %s", status))
					continue
				}

				typeName := strings.ReplaceAll(rtypeStr, "_", " ")
				words := strings.Fields(typeName)
				for i, word := range words {
					if len(word) > 0 {
						words[i] = strings.ToUpper(word[:1]) + word[1:]
					}
				}
				servicesList.Add(strings.Join(words, " "))
			}
		}
		view.Add(servicesList)
	}

	return view
}
