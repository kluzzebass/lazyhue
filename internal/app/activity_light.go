package app

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/kluzzebass/lazyhue/internal/hue"
	"github.com/kluzzebass/lazyhue/internal/hueclient"
	"github.com/kluzzebass/lazyhue/internal/ui"
)

// LightEvent represents a light resource event.
type LightEvent struct {
	baseEvent
	Name           string
	IsOn           bool
	Brightness     float64
	IndicatorColor string
	Details        string
}

func (e *LightEvent) Parse(bridgeID, bridgeName, eventType string, data json.RawMessage, state *hue.BridgeState) (Event, error) {
	var updates []hue.ResourceUpdate
	if err := json.Unmarshal(data, &updates); err != nil {
		return nil, fmt.Errorf("parse light event: %w", err)
	}

	if len(updates) == 0 {
		return nil, fmt.Errorf("no updates in light event")
	}

	update := updates[0]
	e.baseEvent = baseEvent{
		bridgeID:     bridgeID,
		bridgeName:   bridgeName,
		resourceType: "light",
		resourceID:   update.ID,
		eventType:    eventType,
		timestamp:    time.Now(),
	}

	if state != nil {
		if light, ok := state.GetLight(update.ID); ok {
			e.Name = state.GetLightName(light)
			e.IsOn = light.On.On
			if e.IsOn {
				if light.Dimming != nil {
					e.Brightness = float64(light.Dimming.Brightness)
					e.Details = fmt.Sprintf("on %.0f%%", light.Dimming.Brightness)
				} else {
					e.Details = "on"
				}
				// Get color for indicator
				brightnessForColor := 100.0
				if light.Dimming != nil {
					brightnessForColor = float64(light.Dimming.Brightness)
				}
				if light.Color != nil {
					r, g, b := ui.XyToRGB(float64(light.Color.Xy.X), float64(light.Color.Xy.Y), brightnessForColor)
					e.IndicatorColor = fmt.Sprintf("#%02x%02x%02x", r, g, b)
				} else if light.ColorTemperature != nil {
					r, g, b := ui.MirekToRGB(light.ColorTemperature.Mirek)
					e.IndicatorColor = fmt.Sprintf("#%02x%02x%02x", r, g, b)
				}
			} else {
				e.Details = "off"
			}
		} else {
			// Light not in cache - show truncated ID
			e.Name = update.ID[:8] + "…"
		}
	}

	return e, nil
}

func (e *LightEvent) Render(styles *ui.Styles, width int) string {
	return renderEvent(styles, width, e.baseEvent, e.Name, e.Details, e.IndicatorColor, e.Brightness, e.IsOn)
}

// GroupedLightEvent represents a grouped_light resource event.
type GroupedLightEvent struct {
	baseEvent
	Name           string
	IsOn           bool
	Brightness     float64
	IndicatorColor string
	Details        string
}

func (e *GroupedLightEvent) Parse(bridgeID, bridgeName, eventType string, data json.RawMessage, state *hue.BridgeState) (Event, error) {
	var updates []hue.ResourceUpdate
	if err := json.Unmarshal(data, &updates); err != nil {
		return nil, fmt.Errorf("parse grouped_light event: %w", err)
	}

	if len(updates) == 0 {
		return nil, fmt.Errorf("no updates in grouped_light event")
	}

	update := updates[0]
	e.baseEvent = baseEvent{
		bridgeID:     bridgeID,
		bridgeName:   bridgeName,
		resourceType: "grouped_light",
		resourceID:   update.ID,
		eventType:    eventType,
		timestamp:    time.Now(),
	}

	if state != nil {
		if gl, ok := state.GetGroupedLight(update.ID); ok {
			if name := state.GetGroupedLightName(update.ID); name != "" {
				e.Name = name
			}
			if gl.On != nil {
				e.IsOn = gl.On.On
				if e.IsOn {
					if gl.Dimming != nil {
						e.Brightness = float64(gl.Dimming.Brightness)
						e.Details = fmt.Sprintf("on %.0f%%", gl.Dimming.Brightness)
					} else {
						e.Details = "on"
					}
					// Calculate aggregated color from lights in the room/zone/bridge home
					var lights []hueclient.LightGet
					for _, room := range state.AllRooms() {
						for _, svc := range room.Services {
							if svc.Rtype == "grouped_light" && svc.Rid == update.ID {
								lights = state.RoomLights(room)
								break
							}
						}
						if len(lights) > 0 {
							break
						}
					}
					if len(lights) == 0 {
						for _, zone := range state.AllZones() {
							for _, svc := range zone.Services {
								if svc.Rtype == "grouped_light" && svc.Rid == update.ID {
									lights = state.ZoneLights(zone)
									break
								}
							}
							if len(lights) > 0 {
								break
							}
						}
					}
					if len(lights) == 0 {
						// Try bridge_home
						if bh := state.GetBridgeHome(); bh != nil {
							for _, svc := range bh.Services {
								if svc.Rtype == "grouped_light" && svc.Rid == update.ID {
									lights = state.AllLights()
									break
								}
							}
						}
					}
					if len(lights) > 0 {
						_, indicatorColor := hue.CalculateRoomAggregate(lights)
						e.IndicatorColor = indicatorColor
					}
				} else {
					e.Details = "off"
				}
			}
		} else {
			// Grouped light not in cache - show truncated ID
			e.Name = update.ID[:8] + "…"
		}
	}

	return e, nil
}

func (e *GroupedLightEvent) Render(styles *ui.Styles, width int) string {
	return renderEvent(styles, width, e.baseEvent, e.Name, e.Details, e.IndicatorColor, e.Brightness, e.IsOn)
}
