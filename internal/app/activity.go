package app

import (
	"encoding/json"
	"fmt"
	"image/color"
	"math"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/kluzzebass/lazyhue/internal/hue"
	"github.com/kluzzebass/lazyhue/internal/hueclient"
	"github.com/kluzzebass/lazyhue/internal/ui"
)

// Activity represents any loggable activity (events, requests, errors, etc.)
type Activity interface {
	// Render returns the formatted string representation of this activity.
	// width is the available width for rendering (for truncation).
	Render(styles *ui.Styles, width int) string
	// Time returns when this activity occurred.
	Time() time.Time
}

// Event represents a bridge event that can be parsed from raw event data.
type Event interface {
	Activity
	// Parse parses the raw event data and returns an Event instance.
	// bridgeID is the ID of the bridge that sent the event.
	// bridgeName is the name of the bridge that sent the event.
	// eventType is the type of event ("update", "add", "delete", etc.).
	// data is the raw JSON data for this event.
	// state is the bridge state for looking up resource names and details.
	Parse(bridgeID, bridgeName, eventType string, data json.RawMessage, state *hue.BridgeState) (Event, error)
	// ResourceType returns the type of resource this event is for (e.g., "light", "scene").
	ResourceType() string
	// ResourceID returns the ID of the resource this event is for.
	ResourceID() string
}

// baseEvent contains common fields for all events.
type baseEvent struct {
	bridgeID     string
	bridgeName   string
	resourceType string
	resourceID   string
	eventType    string
	timestamp    time.Time
}

func (e *baseEvent) ResourceType() string {
	return e.resourceType
}

func (e *baseEvent) ResourceID() string {
	return e.resourceID
}

func (e *baseEvent) Time() time.Time {
	return e.timestamp
}

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
			if light.On != nil && light.On.On != nil {
				e.IsOn = *light.On.On
				if e.IsOn {
					if light.Dimming != nil && light.Dimming.Brightness != nil {
						e.Brightness = float64(*light.Dimming.Brightness)
						e.Details = fmt.Sprintf("on %.0f%%", *light.Dimming.Brightness)
					} else {
						e.Details = "on"
					}
					// Get color for indicator
					brightnessForColor := 100.0
					if light.Dimming != nil && light.Dimming.Brightness != nil {
						brightnessForColor = float64(*light.Dimming.Brightness)
					}
					if light.Color != nil && light.Color.Xy != nil &&
						light.Color.Xy.X != nil && light.Color.Xy.Y != nil {
						r, g, b := ui.XyToRGB(float64(*light.Color.Xy.X), float64(*light.Color.Xy.Y), brightnessForColor)
						e.IndicatorColor = fmt.Sprintf("#%02x%02x%02x", r, g, b)
					} else if light.ColorTemperature != nil && light.ColorTemperature.Mirek != nil {
						r, g, b := ui.MirekToRGB(*light.ColorTemperature.Mirek)
						e.IndicatorColor = fmt.Sprintf("#%02x%02x%02x", r, g, b)
					}
				} else {
					e.Details = "off"
				}
			}
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
			if gl.On != nil && gl.On.On != nil {
				e.IsOn = *gl.On.On
				if e.IsOn {
					if gl.Dimming != nil && gl.Dimming.Brightness != nil {
						e.Brightness = float64(*gl.Dimming.Brightness)
						e.Details = fmt.Sprintf("on %.0f%%", *gl.Dimming.Brightness)
					} else {
						e.Details = "on"
					}
					// Calculate aggregated color from lights in the room/zone/bridge home
					var lights []hueclient.LightGet
					for _, room := range state.AllRooms() {
						if room.Services != nil {
							for _, svc := range *room.Services {
								if svc.Rtype != nil && *svc.Rtype == "grouped_light" && svc.Rid != nil && *svc.Rid == update.ID {
									lights = state.RoomLights(room)
									break
								}
							}
						}
						if len(lights) > 0 {
							break
						}
					}
					if len(lights) == 0 {
						for _, zone := range state.AllZones() {
							if zone.Services != nil {
								for _, svc := range *zone.Services {
									if svc.Rtype != nil && *svc.Rtype == "grouped_light" && svc.Rid != nil && *svc.Rid == update.ID {
										lights = state.RoomLights(zone)
										break
									}
								}
							}
							if len(lights) > 0 {
								break
							}
						}
					}
					if len(lights) == 0 {
						if bridgeHome := state.GetBridgeHome(); bridgeHome != nil {
							if bridgeHome.Services != nil {
								for _, svc := range *bridgeHome.Services {
									if svc.Rtype != nil && *svc.Rtype == "grouped_light" && svc.Rid != nil && *svc.Rid == update.ID {
										if bridgeHome.Children != nil {
											deviceLights := make(map[string][]hueclient.LightGet)
											for _, light := range state.AllLights() {
												if light.Owner != nil && light.Owner.Rid != nil {
													deviceLights[*light.Owner.Rid] = append(deviceLights[*light.Owner.Rid], light)
												}
											}
											for _, child := range *bridgeHome.Children {
												if child.Rid != nil {
													if deviceLightList, ok := deviceLights[*child.Rid]; ok {
														lights = append(lights, deviceLightList...)
													}
												}
											}
										}
										break
									}
								}
							}
						}
					}
					if len(lights) > 0 {
						_, aggregatedColor := hue.CalculateRoomAggregate(lights)
						if aggregatedColor != "" {
							e.IndicatorColor = aggregatedColor
						}
					}
				} else {
					e.Details = "off"
				}
			}
		}
	}

	return e, nil
}

func (e *GroupedLightEvent) Render(styles *ui.Styles, width int) string {
	return renderEvent(styles, width, e.baseEvent, e.Name, e.Details, e.IndicatorColor, e.Brightness, e.IsOn)
}

// SceneEvent represents a scene resource event.
type SceneEvent struct {
	baseEvent
	Name    string
	Details string
}

func (e *SceneEvent) Parse(bridgeID, bridgeName, eventType string, data json.RawMessage, state *hue.BridgeState) (Event, error) {
	var updates []hue.ResourceUpdate
	if err := json.Unmarshal(data, &updates); err != nil {
		return nil, fmt.Errorf("parse scene event: %w", err)
	}

	if len(updates) == 0 {
		return nil, fmt.Errorf("no updates in scene event")
	}

	update := updates[0]
	e.baseEvent = baseEvent{
		bridgeID:     bridgeID,
		bridgeName:   bridgeName,
		resourceType: "scene",
		resourceID:   update.ID,
		eventType:    eventType,
		timestamp:    time.Now(),
	}

	if state != nil {
		if scene, ok := state.GetScene(update.ID); ok {
			e.Name = scene.SceneName("")
			if scene.Status != nil && scene.Status.Active != nil {
				if *scene.Status.Active == "active" {
					e.Details = "activated"
				} else {
					e.Details = "deactivated"
				}
			}
		}
	}

	return e, nil
}

func (e *SceneEvent) Render(styles *ui.Styles, width int) string {
	return renderEvent(styles, width, e.baseEvent, e.Name, e.Details, "", 0, false)
}

// MotionEvent represents a motion sensor event.
type MotionEvent struct {
	baseEvent
	Name    string
	Details string
}

func (e *MotionEvent) Parse(bridgeID, bridgeName, eventType string, data json.RawMessage, state *hue.BridgeState) (Event, error) {
	var updates []hue.ResourceUpdate
	if err := json.Unmarshal(data, &updates); err != nil {
		return nil, fmt.Errorf("parse motion event: %w", err)
	}

	if len(updates) == 0 {
		return nil, fmt.Errorf("no updates in motion event")
	}

	update := updates[0]
	e.baseEvent = baseEvent{
		bridgeID:     bridgeID,
		bridgeName:   bridgeName,
		resourceType: "motion",
		resourceID:   update.ID,
		eventType:    eventType,
		timestamp:    time.Now(),
	}

	if state != nil {
		if motion, ok := state.GetMotion(update.ID); ok {
			if motion.Owner != nil && motion.Owner.Rid != nil {
				if device, ok := state.GetDevice(*motion.Owner.Rid); ok {
					e.Name = device.DeviceName("")
				}
			}
			if motion.Motion != nil && motion.Motion.Motion != nil {
				if *motion.Motion.Motion {
					e.Details = "motion detected"
				} else {
					e.Details = "clear"
				}
			}
		}
	}

	return e, nil
}

func (e *MotionEvent) Render(styles *ui.Styles, width int) string {
	return renderEvent(styles, width, e.baseEvent, e.Name, e.Details, "", 0, false)
}

// ButtonEvent represents a button press event.
type ButtonEvent struct {
	baseEvent
	Name    string
	Details string
}

func (e *ButtonEvent) Parse(bridgeID, bridgeName, eventType string, data json.RawMessage, state *hue.BridgeState) (Event, error) {
	var updates []hue.ResourceUpdate
	if err := json.Unmarshal(data, &updates); err != nil {
		return nil, fmt.Errorf("parse button event: %w", err)
	}

	if len(updates) == 0 {
		return nil, fmt.Errorf("no updates in button event")
	}

	update := updates[0]
	e.baseEvent = baseEvent{
		bridgeID:     bridgeID,
		bridgeName:   bridgeName,
		resourceType: "button",
		resourceID:   update.ID,
		eventType:    eventType,
		timestamp:    time.Now(),
	}

	// Get button action from the update
	if update.Button != nil && update.Button.LastEvent != "" {
		e.Details = formatButtonEvent(update.Button.LastEvent)
	}

	// Get device name from owner
	if state != nil && update.Owner != nil {
		if device, ok := state.GetDevice(update.Owner.Rid); ok {
			e.Name = device.DeviceName("")
		}
	}

	return e, nil
}

// formatButtonEvent converts Hue button event names to human-readable form.
func formatButtonEvent(event string) string {
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
		return "long press"
	default:
		return event
	}
}

func (e *ButtonEvent) Render(styles *ui.Styles, width int) string {
	return renderEvent(styles, width, e.baseEvent, e.Name, e.Details, "", 0, false)
}

// DevicePowerEvent represents a device power (battery) event.
type DevicePowerEvent struct {
	baseEvent
	Name    string
	Details string
}

func (e *DevicePowerEvent) Parse(bridgeID, bridgeName, eventType string, data json.RawMessage, state *hue.BridgeState) (Event, error) {
	var updates []hue.ResourceUpdate
	if err := json.Unmarshal(data, &updates); err != nil {
		return nil, fmt.Errorf("parse device_power event: %w", err)
	}

	if len(updates) == 0 {
		return nil, fmt.Errorf("no updates in device_power event")
	}

	update := updates[0]
	e.baseEvent = baseEvent{
		bridgeID:     bridgeID,
		bridgeName:   bridgeName,
		resourceType: "device_power",
		resourceID:   update.ID,
		eventType:    eventType,
		timestamp:    time.Now(),
	}

	// Get battery info from the update
	if update.PowerState != nil {
		var parts []string
		if update.PowerState.BatteryLevel != nil {
			parts = append(parts, fmt.Sprintf("%d%%", *update.PowerState.BatteryLevel))
		}
		if update.PowerState.BatteryState != nil && *update.PowerState.BatteryState != "normal" {
			parts = append(parts, *update.PowerState.BatteryState)
		}
		if len(parts) > 0 {
			e.Details = strings.Join(parts, " ")
		}
	}

	// Get device name from owner
	if state != nil && update.Owner != nil {
		if device, ok := state.GetDevice(update.Owner.Rid); ok {
			e.Name = device.DeviceName("")
		}
	}

	return e, nil
}

func (e *DevicePowerEvent) Render(styles *ui.Styles, width int) string {
	return renderEvent(styles, width, e.baseEvent, e.Name, e.Details, "", 0, false)
}

// RelativeRotaryEvent represents a rotary dial event (Hue Tap Dial).
type RelativeRotaryEvent struct {
	baseEvent
	Name    string
	Details string
}

func (e *RelativeRotaryEvent) Parse(bridgeID, bridgeName, eventType string, data json.RawMessage, state *hue.BridgeState) (Event, error) {
	var updates []hue.ResourceUpdate
	if err := json.Unmarshal(data, &updates); err != nil {
		return nil, fmt.Errorf("parse relative_rotary event: %w", err)
	}
	if len(updates) == 0 {
		return nil, fmt.Errorf("no updates in relative_rotary event")
	}
	update := updates[0]
	e.baseEvent = baseEvent{
		bridgeID:     bridgeID,
		bridgeName:   bridgeName,
		resourceType: "relative_rotary",
		resourceID:   update.ID,
		eventType:    eventType,
		timestamp:    time.Now(),
	}
	if update.RelativeRotary != nil && update.RelativeRotary.LastEvent != nil {
		evt := update.RelativeRotary.LastEvent
		if evt.Rotation != nil {
			dir := "→"
			if evt.Rotation.Direction == "counter_clock_wise" {
				dir = "←"
			}
			e.Details = fmt.Sprintf("%s %d steps", dir, evt.Rotation.Steps)
		} else {
			e.Details = evt.Action
		}
	}
	if state != nil && update.Owner != nil {
		if device, ok := state.GetDevice(update.Owner.Rid); ok {
			e.Name = device.DeviceName("")
		}
	}
	return e, nil
}

func (e *RelativeRotaryEvent) Render(styles *ui.Styles, width int) string {
	return renderEvent(styles, width, e.baseEvent, e.Name, e.Details, "", 0, false)
}

// ContactEvent represents a contact sensor event (door/window).
type ContactEvent struct {
	baseEvent
	Name    string
	Details string
}

func (e *ContactEvent) Parse(bridgeID, bridgeName, eventType string, data json.RawMessage, state *hue.BridgeState) (Event, error) {
	var updates []hue.ResourceUpdate
	if err := json.Unmarshal(data, &updates); err != nil {
		return nil, fmt.Errorf("parse contact event: %w", err)
	}
	if len(updates) == 0 {
		return nil, fmt.Errorf("no updates in contact event")
	}
	update := updates[0]
	e.baseEvent = baseEvent{
		bridgeID:     bridgeID,
		bridgeName:   bridgeName,
		resourceType: "contact",
		resourceID:   update.ID,
		eventType:    eventType,
		timestamp:    time.Now(),
	}
	if update.ContactReport != nil {
		if update.ContactReport.State == "contact" {
			e.Details = "closed"
		} else {
			e.Details = "open"
		}
	}
	if state != nil && update.Owner != nil {
		if device, ok := state.GetDevice(update.Owner.Rid); ok {
			e.Name = device.DeviceName("")
		}
	}
	return e, nil
}

func (e *ContactEvent) Render(styles *ui.Styles, width int) string {
	return renderEvent(styles, width, e.baseEvent, e.Name, e.Details, "", 0, false)
}

// TamperEvent represents a tamper sensor event.
type TamperEvent struct {
	baseEvent
	Name    string
	Details string
}

func (e *TamperEvent) Parse(bridgeID, bridgeName, eventType string, data json.RawMessage, state *hue.BridgeState) (Event, error) {
	var updates []hue.ResourceUpdate
	if err := json.Unmarshal(data, &updates); err != nil {
		return nil, fmt.Errorf("parse tamper event: %w", err)
	}
	if len(updates) == 0 {
		return nil, fmt.Errorf("no updates in tamper event")
	}
	update := updates[0]
	e.baseEvent = baseEvent{
		bridgeID:     bridgeID,
		bridgeName:   bridgeName,
		resourceType: "tamper",
		resourceID:   update.ID,
		eventType:    eventType,
		timestamp:    time.Now(),
	}
	if update.TamperReports != nil && len(*update.TamperReports) > 0 {
		if (*update.TamperReports)[0].State == "tampered" {
			e.Details = "tampered"
		} else {
			e.Details = "secure"
		}
	}
	if state != nil && update.Owner != nil {
		if device, ok := state.GetDevice(update.Owner.Rid); ok {
			e.Name = device.DeviceName("")
		}
	}
	return e, nil
}

func (e *TamperEvent) Render(styles *ui.Styles, width int) string {
	return renderEvent(styles, width, e.baseEvent, e.Name, e.Details, "", 0, false)
}

// ZigbeeConnectivityEvent represents a Zigbee connectivity status change.
type ZigbeeConnectivityEvent struct {
	baseEvent
	Name    string
	Details string
}

func (e *ZigbeeConnectivityEvent) Parse(bridgeID, bridgeName, eventType string, data json.RawMessage, state *hue.BridgeState) (Event, error) {
	var updates []hue.ResourceUpdate
	if err := json.Unmarshal(data, &updates); err != nil {
		return nil, fmt.Errorf("parse zigbee_connectivity event: %w", err)
	}
	if len(updates) == 0 {
		return nil, fmt.Errorf("no updates in zigbee_connectivity event")
	}
	update := updates[0]
	e.baseEvent = baseEvent{
		bridgeID:     bridgeID,
		bridgeName:   bridgeName,
		resourceType: "zigbee_connectivity",
		resourceID:   update.ID,
		eventType:    eventType,
		timestamp:    time.Now(),
	}
	if update.ConnectivityStatus != nil {
		e.Details = strings.ReplaceAll(update.ConnectivityStatus.Status, "_", " ")
	}
	if state != nil && update.Owner != nil {
		if device, ok := state.GetDevice(update.Owner.Rid); ok {
			e.Name = device.DeviceName("")
		}
	}
	return e, nil
}

func (e *ZigbeeConnectivityEvent) Render(styles *ui.Styles, width int) string {
	return renderEvent(styles, width, e.baseEvent, e.Name, e.Details, "", 0, false)
}

// ZgpConnectivityEvent represents a ZGP (Zigbee Green Power) connectivity status change.
type ZgpConnectivityEvent struct {
	baseEvent
	Name    string
	Details string
}

func (e *ZgpConnectivityEvent) Parse(bridgeID, bridgeName, eventType string, data json.RawMessage, state *hue.BridgeState) (Event, error) {
	var updates []hue.ResourceUpdate
	if err := json.Unmarshal(data, &updates); err != nil {
		return nil, fmt.Errorf("parse zgp_connectivity event: %w", err)
	}
	if len(updates) == 0 {
		return nil, fmt.Errorf("no updates in zgp_connectivity event")
	}
	update := updates[0]
	e.baseEvent = baseEvent{
		bridgeID:     bridgeID,
		bridgeName:   bridgeName,
		resourceType: "zgp_connectivity",
		resourceID:   update.ID,
		eventType:    eventType,
		timestamp:    time.Now(),
	}
	if update.ConnectivityStatus != nil {
		e.Details = strings.ReplaceAll(update.ConnectivityStatus.Status, "_", " ")
	}
	if state != nil && update.Owner != nil {
		if device, ok := state.GetDevice(update.Owner.Rid); ok {
			e.Name = device.DeviceName("")
		}
	}
	return e, nil
}

func (e *ZgpConnectivityEvent) Render(styles *ui.Styles, width int) string {
	return renderEvent(styles, width, e.baseEvent, e.Name, e.Details, "", 0, false)
}

// WifiConnectivityEvent represents a WiFi connectivity status change.
type WifiConnectivityEvent struct {
	baseEvent
	Name    string
	Details string
}

func (e *WifiConnectivityEvent) Parse(bridgeID, bridgeName, eventType string, data json.RawMessage, state *hue.BridgeState) (Event, error) {
	var updates []hue.ResourceUpdate
	if err := json.Unmarshal(data, &updates); err != nil {
		return nil, fmt.Errorf("parse wifi_connectivity event: %w", err)
	}
	if len(updates) == 0 {
		return nil, fmt.Errorf("no updates in wifi_connectivity event")
	}
	update := updates[0]
	e.baseEvent = baseEvent{
		bridgeID:     bridgeID,
		bridgeName:   bridgeName,
		resourceType: "wifi_connectivity",
		resourceID:   update.ID,
		eventType:    eventType,
		timestamp:    time.Now(),
	}
	if update.ConnectivityStatus != nil {
		e.Details = strings.ReplaceAll(update.ConnectivityStatus.Status, "_", " ")
	}
	if state != nil && update.Owner != nil {
		if device, ok := state.GetDevice(update.Owner.Rid); ok {
			e.Name = device.DeviceName("")
		}
	}
	return e, nil
}

func (e *WifiConnectivityEvent) Render(styles *ui.Styles, width int) string {
	return renderEvent(styles, width, e.baseEvent, e.Name, e.Details, "", 0, false)
}

// DeviceSoftwareUpdateEvent represents a device software update event.
type DeviceSoftwareUpdateEvent struct {
	baseEvent
	Name    string
	Details string
}

func (e *DeviceSoftwareUpdateEvent) Parse(bridgeID, bridgeName, eventType string, data json.RawMessage, state *hue.BridgeState) (Event, error) {
	var updates []hue.ResourceUpdate
	if err := json.Unmarshal(data, &updates); err != nil {
		return nil, fmt.Errorf("parse device_software_update event: %w", err)
	}
	if len(updates) == 0 {
		return nil, fmt.Errorf("no updates in device_software_update event")
	}
	update := updates[0]
	e.baseEvent = baseEvent{
		bridgeID:     bridgeID,
		bridgeName:   bridgeName,
		resourceType: "device_software_update",
		resourceID:   update.ID,
		eventType:    eventType,
		timestamp:    time.Now(),
	}
	if update.SoftwareUpdate != nil {
		e.Details = strings.ReplaceAll(update.SoftwareUpdate.State, "_", " ")
	}
	if state != nil && update.Owner != nil {
		if device, ok := state.GetDevice(update.Owner.Rid); ok {
			e.Name = device.DeviceName("")
		}
	}
	return e, nil
}

func (e *DeviceSoftwareUpdateEvent) Render(styles *ui.Styles, width int) string {
	return renderEvent(styles, width, e.baseEvent, e.Name, e.Details, "", 0, false)
}

// SmartSceneEvent represents a smart scene event.
type SmartSceneEvent struct {
	baseEvent
	Name    string
	Details string
}

func (e *SmartSceneEvent) Parse(bridgeID, bridgeName, eventType string, data json.RawMessage, state *hue.BridgeState) (Event, error) {
	var updates []hue.ResourceUpdate
	if err := json.Unmarshal(data, &updates); err != nil {
		return nil, fmt.Errorf("parse smart_scene event: %w", err)
	}
	if len(updates) == 0 {
		return nil, fmt.Errorf("no updates in smart_scene event")
	}
	update := updates[0]
	e.baseEvent = baseEvent{
		bridgeID:     bridgeID,
		bridgeName:   bridgeName,
		resourceType: "smart_scene",
		resourceID:   update.ID,
		eventType:    eventType,
		timestamp:    time.Now(),
	}
	if update.SmartSceneState != nil {
		e.Details = update.SmartSceneState.Active
	}
	if update.Metadata != nil && update.Metadata.Name != nil {
		e.Name = *update.Metadata.Name
	} else if state != nil {
		// Fall back to looking up the name from state
		if scene, ok := state.GetSmartScene(update.ID); ok {
			e.Name = state.GetSmartSceneName(scene)
		}
	}
	return e, nil
}

func (e *SmartSceneEvent) Render(styles *ui.Styles, width int) string {
	return renderEvent(styles, width, e.baseEvent, e.Name, e.Details, "", 0, false)
}

// SimpleDeviceEvent is a generic event for device-owned resources that just need name + event type.
type SimpleDeviceEvent struct {
	baseEvent
	Name    string
	Details string
}

func (e *SimpleDeviceEvent) Parse(bridgeID, bridgeName, eventType string, data json.RawMessage, state *hue.BridgeState) (Event, error) {
	// This should not be called directly - use ParseWithType instead
	return nil, fmt.Errorf("SimpleDeviceEvent.Parse should not be called directly")
}

func (e *SimpleDeviceEvent) ParseWithType(bridgeID, bridgeName, resourceType, eventType string, data json.RawMessage, state *hue.BridgeState) (Event, error) {
	var updates []hue.ResourceUpdate
	if err := json.Unmarshal(data, &updates); err != nil {
		return nil, fmt.Errorf("parse %s event: %w", resourceType, err)
	}
	if len(updates) == 0 {
		return nil, fmt.Errorf("no updates in %s event", resourceType)
	}
	update := updates[0]
	e.baseEvent = baseEvent{
		bridgeID:     bridgeID,
		bridgeName:   bridgeName,
		resourceType: resourceType,
		resourceID:   update.ID,
		eventType:    eventType,
		timestamp:    time.Now(),
	}
	e.Details = eventType
	if state != nil && update.Owner != nil {
		if device, ok := state.GetDevice(update.Owner.Rid); ok {
			e.Name = device.DeviceName("")
		}
	}
	return e, nil
}

func (e *SimpleDeviceEvent) Render(styles *ui.Styles, width int) string {
	return renderEvent(styles, width, e.baseEvent, e.Name, e.Details, "", 0, false)
}

// SimpleNamedEvent is a generic event for resources with metadata name.
type SimpleNamedEvent struct {
	baseEvent
	Name    string
	Details string
}

func (e *SimpleNamedEvent) Parse(bridgeID, bridgeName, eventType string, data json.RawMessage, state *hue.BridgeState) (Event, error) {
	// This should not be called directly - use ParseWithType instead
	return nil, fmt.Errorf("SimpleNamedEvent.Parse should not be called directly")
}

func (e *SimpleNamedEvent) ParseWithType(bridgeID, bridgeName, resourceType, eventType string, data json.RawMessage, state *hue.BridgeState) (Event, error) {
	var updates []hue.ResourceUpdate
	if err := json.Unmarshal(data, &updates); err != nil {
		return nil, fmt.Errorf("parse %s event: %w", resourceType, err)
	}
	if len(updates) == 0 {
		return nil, fmt.Errorf("no updates in %s event", resourceType)
	}
	update := updates[0]
	e.baseEvent = baseEvent{
		bridgeID:     bridgeID,
		bridgeName:   bridgeName,
		resourceType: resourceType,
		resourceID:   update.ID,
		eventType:    eventType,
		timestamp:    time.Now(),
	}
	e.Details = eventType
	if update.Metadata != nil && update.Metadata.Name != nil {
		e.Name = *update.Metadata.Name
	}
	return e, nil
}

func (e *SimpleNamedEvent) Render(styles *ui.Styles, width int) string {
	return renderEvent(styles, width, e.baseEvent, e.Name, e.Details, "", 0, false)
}

// SimpleEvent is a generic event for resources without specific data.
type SimpleEvent struct {
	baseEvent
	Details string
}

func (e *SimpleEvent) Parse(bridgeID, bridgeName, eventType string, data json.RawMessage, state *hue.BridgeState) (Event, error) {
	// This should not be called directly - use ParseWithType instead
	return nil, fmt.Errorf("SimpleEvent.Parse should not be called directly")
}

func (e *SimpleEvent) ParseWithType(bridgeID, bridgeName, resourceType, eventType string, data json.RawMessage, state *hue.BridgeState) (Event, error) {
	var updates []hue.ResourceUpdate
	if err := json.Unmarshal(data, &updates); err != nil {
		return nil, fmt.Errorf("parse %s event: %w", resourceType, err)
	}
	if len(updates) == 0 {
		return nil, fmt.Errorf("no updates in %s event", resourceType)
	}
	update := updates[0]
	e.baseEvent = baseEvent{
		bridgeID:     bridgeID,
		bridgeName:   bridgeName,
		resourceType: resourceType,
		resourceID:   update.ID,
		eventType:    eventType,
		timestamp:    time.Now(),
	}
	e.Details = eventType
	return e, nil
}

func (e *SimpleEvent) Render(styles *ui.Styles, width int) string {
	return renderEvent(styles, width, e.baseEvent, "", e.Details, "", 0, false)
}

// TemperatureEvent represents a temperature sensor event.
type TemperatureEvent struct {
	baseEvent
	Name    string
	Details string
}

func (e *TemperatureEvent) Parse(bridgeID, bridgeName, eventType string, data json.RawMessage, state *hue.BridgeState) (Event, error) {
	var updates []hue.ResourceUpdate
	if err := json.Unmarshal(data, &updates); err != nil {
		return nil, fmt.Errorf("parse temperature event: %w", err)
	}

	if len(updates) == 0 {
		return nil, fmt.Errorf("no updates in temperature event")
	}

	update := updates[0]
	e.baseEvent = baseEvent{
		bridgeID:     bridgeID,
		bridgeName:   bridgeName,
		resourceType: "temperature",
		resourceID:   update.ID,
		eventType:    eventType,
		timestamp:    time.Now(),
	}

	if state != nil {
		if temp, ok := state.GetTemperature(update.ID); ok {
			if temp.Owner != nil && temp.Owner.Rid != nil {
				if device, ok := state.GetDevice(*temp.Owner.Rid); ok {
					e.Name = device.DeviceName("")
				}
			}
			if temp.Temperature != nil && temp.Temperature.Temperature != nil {
				e.Details = fmt.Sprintf("%.1f°C", *temp.Temperature.Temperature)
			}
		}
	}

	return e, nil
}

func (e *TemperatureEvent) Render(styles *ui.Styles, width int) string {
	return renderEvent(styles, width, e.baseEvent, e.Name, e.Details, "", 0, false)
}

// LightLevelEvent represents a light level sensor event.
type LightLevelEvent struct {
	baseEvent
	Name    string
	Details string
}

func (e *LightLevelEvent) Parse(bridgeID, bridgeName, eventType string, data json.RawMessage, state *hue.BridgeState) (Event, error) {
	var updates []hue.ResourceUpdate
	if err := json.Unmarshal(data, &updates); err != nil {
		return nil, fmt.Errorf("parse light_level event: %w", err)
	}

	if len(updates) == 0 {
		return nil, fmt.Errorf("no updates in light_level event")
	}

	update := updates[0]
	e.baseEvent = baseEvent{
		bridgeID:     bridgeID,
		bridgeName:   bridgeName,
		resourceType: "light_level",
		resourceID:   update.ID,
		eventType:    eventType,
		timestamp:    time.Now(),
	}

	if state != nil {
		if ll, ok := state.GetLightLevel(update.ID); ok {
			if ll.Owner != nil && ll.Owner.Rid != nil {
				if device, ok := state.GetDevice(*ll.Owner.Rid); ok {
					e.Name = device.DeviceName("")
				}
			}
			if ll.Light != nil && ll.Light.LightLevel != nil {
				// Convert from Hue's log scale to approximate lux
				lux := float64(*ll.Light.LightLevel-1) / 10000.0
				lux = 100 * (lux * lux * lux) // Rough approximation
				e.Details = fmt.Sprintf("%.0f lux", lux)
			}
		}
	}

	return e, nil
}

func (e *LightLevelEvent) Render(styles *ui.Styles, width int) string {
	return renderEvent(styles, width, e.baseEvent, e.Name, e.Details, "", 0, false)
}

// GroupedLightLevelEvent represents a grouped_light_level event.
type GroupedLightLevelEvent struct {
	baseEvent
	Name    string
	Details string
}

func (e *GroupedLightLevelEvent) Parse(bridgeID, bridgeName, eventType string, data json.RawMessage, state *hue.BridgeState) (Event, error) {
	var updates []hue.ResourceUpdate
	if err := json.Unmarshal(data, &updates); err != nil {
		return nil, fmt.Errorf("parse grouped_light_level event: %w", err)
	}

	if len(updates) == 0 {
		return nil, fmt.Errorf("no updates in grouped_light_level event")
	}

	update := updates[0]
	e.baseEvent = baseEvent{
		bridgeID:     bridgeID,
		bridgeName:   bridgeName,
		resourceType: "grouped_light_level",
		resourceID:   update.ID,
		eventType:    eventType,
		timestamp:    time.Now(),
	}

	if state != nil {
		allRooms := state.AllRooms()
		for _, room := range allRooms {
			if room.Services != nil {
				for _, svc := range *room.Services {
					if svc.Rtype != nil && *svc.Rtype == "grouped_light_level" && svc.Rid != nil && *svc.Rid == update.ID {
						if room.Metadata != nil && room.Metadata.Name != nil {
							e.Name = *room.Metadata.Name
						}
						var totalLux float64
						var count int
						if room.Children != nil {
							for _, child := range *room.Children {
								if child.Rid == nil || child.Rtype == nil || *child.Rtype != "device" {
									continue
								}
								if device, ok := state.GetDevice(*child.Rid); ok {
									if hasLevel, level := state.GetDeviceLightLevel(device); hasLevel && level > 0 {
										// Convert from Hue's log scale to lux: 10^((level-1)/10000)
										lux := math.Pow(10, float64(level-1)/10000)
										totalLux += lux
										count++
									}
								}
							}
						}
						if count > 0 {
							avgLux := totalLux / float64(count)
							e.Details = fmt.Sprintf("%.0f lux", avgLux)
						} else {
							e.Details = "no sensors"
						}
						return e, nil
					}
				}
			}
		}
		allZones := state.AllZones()
		for _, zone := range allZones {
			if zone.Services != nil {
				for _, svc := range *zone.Services {
					if svc.Rtype != nil && *svc.Rtype == "grouped_light_level" && svc.Rid != nil && *svc.Rid == update.ID {
						if zone.Metadata != nil && zone.Metadata.Name != nil {
							e.Name = *zone.Metadata.Name
						}
						var totalLux float64
						var count int
						if zone.Children != nil {
							for _, child := range *zone.Children {
								if child.Rid == nil || child.Rtype == nil || *child.Rtype != "device" {
									continue
								}
								if device, ok := state.GetDevice(*child.Rid); ok {
									if hasLevel, level := state.GetDeviceLightLevel(device); hasLevel && level > 0 {
										// Convert from Hue's log scale to lux: 10^((level-1)/10000)
										lux := math.Pow(10, float64(level-1)/10000)
										totalLux += lux
										count++
									}
								}
							}
						}
						if count > 0 {
							avgLux := totalLux / float64(count)
							e.Details = fmt.Sprintf("%.0f lux", avgLux)
						} else {
							e.Details = "no sensors"
						}
						return e, nil
					}
				}
			}
		}
		e.Details = "light level updated"
	}

	return e, nil
}

func (e *GroupedLightLevelEvent) Render(styles *ui.Styles, width int) string {
	return renderEvent(styles, width, e.baseEvent, e.Name, e.Details, "", 0, false)
}

// GroupedMotionEvent represents a grouped_motion event (aggregate motion for a room/zone).
type GroupedMotionEvent struct {
	baseEvent
	Name    string
	Details string
}

func (e *GroupedMotionEvent) Parse(bridgeID, bridgeName, eventType string, data json.RawMessage, state *hue.BridgeState) (Event, error) {
	var updates []hue.ResourceUpdate
	if err := json.Unmarshal(data, &updates); err != nil {
		return nil, fmt.Errorf("parse grouped_motion event: %w", err)
	}

	if len(updates) == 0 {
		return nil, fmt.Errorf("no updates in grouped_motion event")
	}

	update := updates[0]
	e.baseEvent = baseEvent{
		bridgeID:     bridgeID,
		bridgeName:   bridgeName,
		resourceType: "grouped_motion",
		resourceID:   update.ID,
		eventType:    eventType,
		timestamp:    time.Now(),
	}

	if state != nil {
		// Check rooms for this grouped_motion service
		allRooms := state.AllRooms()
		for _, room := range allRooms {
			if room.Services != nil {
				for _, svc := range *room.Services {
					if svc.Rtype != nil && *svc.Rtype == "grouped_motion" && svc.Rid != nil && *svc.Rid == update.ID {
						if room.Metadata != nil && room.Metadata.Name != nil {
							e.Name = *room.Metadata.Name
						}
						var anyMotion bool
						var sensorCount int
						if room.Children != nil {
							for _, child := range *room.Children {
								if child.Rid == nil || child.Rtype == nil || *child.Rtype != "device" {
									continue
								}
								if device, ok := state.GetDevice(*child.Rid); ok {
									if hasMotion, isDetecting := state.GetDeviceMotionState(device); hasMotion {
										sensorCount++
										if isDetecting {
											anyMotion = true
										}
									}
								}
							}
						}
						if sensorCount > 0 {
							if anyMotion {
								e.Details = "motion detected"
							} else {
								e.Details = "clear"
							}
						} else {
							e.Details = "no sensors"
						}
						return e, nil
					}
				}
			}
		}
		// Check zones for this grouped_motion service
		allZones := state.AllZones()
		for _, zone := range allZones {
			if zone.Services != nil {
				for _, svc := range *zone.Services {
					if svc.Rtype != nil && *svc.Rtype == "grouped_motion" && svc.Rid != nil && *svc.Rid == update.ID {
						if zone.Metadata != nil && zone.Metadata.Name != nil {
							e.Name = *zone.Metadata.Name
						}
						var anyMotion bool
						var sensorCount int
						if zone.Children != nil {
							for _, child := range *zone.Children {
								if child.Rid == nil || child.Rtype == nil || *child.Rtype != "device" {
									continue
								}
								if device, ok := state.GetDevice(*child.Rid); ok {
									if hasMotion, isDetecting := state.GetDeviceMotionState(device); hasMotion {
										sensorCount++
										if isDetecting {
											anyMotion = true
										}
									}
								}
							}
						}
						if sensorCount > 0 {
							if anyMotion {
								e.Details = "motion detected"
							} else {
								e.Details = "clear"
							}
						} else {
							e.Details = "no sensors"
						}
						return e, nil
					}
				}
			}
		}
		e.Details = "motion updated"
	}

	return e, nil
}

func (e *GroupedMotionEvent) Render(styles *ui.Styles, width int) string {
	return renderEvent(styles, width, e.baseEvent, e.Name, e.Details, "", 0, false)
}

// UnhandledEvent represents an event type that we don't have a specific handler for.
type UnhandledEvent struct {
	baseEvent
	RawData json.RawMessage
}

func (e *UnhandledEvent) Parse(bridgeID, bridgeName, eventType string, data json.RawMessage, state *hue.BridgeState) (Event, error) {
	var updates []hue.ResourceUpdate
	if err := json.Unmarshal(data, &updates); err != nil {
		return nil, fmt.Errorf("parse unhandled event: %w", err)
	}

	if len(updates) == 0 {
		return nil, fmt.Errorf("no updates in unhandled event")
	}

	update := updates[0]
	e.baseEvent = baseEvent{
		bridgeID:     bridgeID,
		bridgeName:   bridgeName,
		resourceType: update.Type,
		resourceID:   update.ID,
		eventType:    eventType,
		timestamp:    time.Now(),
	}
	e.RawData = data

	return e, nil
}

func (e *UnhandledEvent) Render(styles *ui.Styles, width int) string {
	return renderEvent(styles, width, e.baseEvent, "", fmt.Sprintf("unhandled %s event", e.resourceType), "", 0, false)
}

// DeviceEvent represents a device update event.
type DeviceEvent struct {
	baseEvent
	Name    string
	Details string
}

func (e *DeviceEvent) Parse(bridgeID, bridgeName, eventType string, data json.RawMessage, state *hue.BridgeState) (Event, error) {
	var updates []hue.ResourceUpdate
	if err := json.Unmarshal(data, &updates); err != nil {
		return nil, fmt.Errorf("parse device event: %w", err)
	}

	if len(updates) == 0 {
		return nil, fmt.Errorf("no updates in device event")
	}

	update := updates[0]

	e.baseEvent = baseEvent{
		bridgeID:     bridgeID,
		bridgeName:   bridgeName,
		resourceType: update.Type,
		resourceID:   update.ID,
		eventType:    eventType,
		timestamp:    time.Now(),
	}

	// Look up device name from state
	if state != nil {
		if device, ok := state.GetDevice(update.ID); ok {
			e.Name = state.GetDeviceName(device)
		}
	}

	if e.Name == "" {
		e.Name = update.ID
	}

	e.Details = eventType

	return e, nil
}

func (e *DeviceEvent) Render(styles *ui.Styles, width int) string {
	return renderEvent(styles, width, e.baseEvent, e.Name, e.Details, "", 0, false)
}

// RequestActivity represents a request to the bridge (not an event).
type RequestActivity struct {
	timestamp  time.Time
	bridgeID   string
	bridgeName string
	message    string
}

func (a *RequestActivity) Time() time.Time {
	return a.timestamp
}

func (a *RequestActivity) Render(styles *ui.Styles, width int) string {
	timeStr := a.timestamp.Format("15:04:05")
	typeStyle := lipgloss.NewStyle().Foreground(styles.Theme.Secondary)
	typeIndicator := typeStyle.Render("→")
	bridgeStr := ""
	if a.bridgeName != "" {
		bridgeStyle := lipgloss.NewStyle().Foreground(styles.Theme.EntityBridge)
		bridgeStr = styles.Dimmed.Render("[") + bridgeStyle.Render(a.bridgeName) + styles.Dimmed.Render("]") + " "
	}
	line := fmt.Sprintf("%s %s %s%s", styles.Dimmed.Render(timeStr), typeIndicator, bridgeStr, a.message)

	// Truncate if width is provided and line exceeds it
	if width > 0 && lipgloss.Width(line) > width {
		line = ansi.Truncate(line, width, "…")
	}
	return line
}

// ErrorActivity represents an error activity.
type ErrorActivity struct {
	timestamp  time.Time
	bridgeID   string
	bridgeName string
	message    string
}

func (a *ErrorActivity) Time() time.Time {
	return a.timestamp
}

func (a *ErrorActivity) Render(styles *ui.Styles, width int) string {
	timeStr := a.timestamp.Format("15:04:05")
	typeStyle := lipgloss.NewStyle().Foreground(styles.Theme.Error)
	typeIndicator := typeStyle.Render("!")
	bridgeStr := ""
	if a.bridgeName != "" {
		bridgeStyle := lipgloss.NewStyle().Foreground(styles.Theme.EntityBridge)
		bridgeStr = styles.Dimmed.Render("[") + bridgeStyle.Render(a.bridgeName) + styles.Dimmed.Render("]") + " "
	}
	line := fmt.Sprintf("%s %s %s%s", styles.Dimmed.Render(timeStr), typeIndicator, bridgeStr, a.message)

	// Truncate if width is provided and line exceeds it
	if width > 0 && lipgloss.Width(line) > width {
		line = ansi.Truncate(line, width, "…")
	}
	return line
}

// RoomEvent represents a room update event.
type RoomEvent struct {
	baseEvent
	Name    string
	Details string
}

func (e *RoomEvent) Parse(bridgeID, bridgeName, eventType string, data json.RawMessage, state *hue.BridgeState) (Event, error) {
	var updates []hue.ResourceUpdate
	if err := json.Unmarshal(data, &updates); err != nil {
		return nil, fmt.Errorf("parse room event: %w", err)
	}

	if len(updates) == 0 {
		return nil, fmt.Errorf("no updates in room event")
	}

	update := updates[0]

	e.baseEvent = baseEvent{
		bridgeID:     bridgeID,
		bridgeName:   bridgeName,
		resourceType: update.Type,
		resourceID:   update.ID,
		eventType:    eventType,
		timestamp:    time.Now(),
	}

	// Look up room name from state
	if state != nil {
		if room, ok := state.GetRoom(update.ID); ok {
			if room.Metadata != nil && room.Metadata.Name != nil {
				e.Name = *room.Metadata.Name
			}
		}
	}

	if e.Name == "" {
		e.Name = update.ID
	}

	// Set details based on event type
	switch eventType {
	case "add":
		e.Details = "created"
	case "update":
		e.Details = "updated"
	case "delete":
		e.Details = "deleted"
	default:
		e.Details = eventType
	}

	return e, nil
}

func (e *RoomEvent) Render(styles *ui.Styles, width int) string {
	return renderEvent(styles, width, e.baseEvent, "Room \""+e.Name+"\"", e.Details, "", 0, false)
}

// ZoneEvent represents a zone update event.
type ZoneEvent struct {
	baseEvent
	Name    string
	Details string
}

func (e *ZoneEvent) Parse(bridgeID, bridgeName, eventType string, data json.RawMessage, state *hue.BridgeState) (Event, error) {
	var updates []hue.ResourceUpdate
	if err := json.Unmarshal(data, &updates); err != nil {
		return nil, fmt.Errorf("parse zone event: %w", err)
	}

	if len(updates) == 0 {
		return nil, fmt.Errorf("no updates in zone event")
	}

	update := updates[0]

	e.baseEvent = baseEvent{
		bridgeID:     bridgeID,
		bridgeName:   bridgeName,
		resourceType: update.Type,
		resourceID:   update.ID,
		eventType:    eventType,
		timestamp:    time.Now(),
	}

	// Look up zone name from state
	if state != nil {
		if zone, ok := state.GetZone(update.ID); ok {
			if zone.Metadata != nil && zone.Metadata.Name != nil {
				e.Name = *zone.Metadata.Name
			}
		}
	}

	if e.Name == "" {
		e.Name = update.ID
	}

	// Set details based on event type
	switch eventType {
	case "add":
		e.Details = "created"
	case "update":
		e.Details = "updated"
	case "delete":
		e.Details = "deleted"
	default:
		e.Details = eventType
	}

	return e, nil
}

func (e *ZoneEvent) Render(styles *ui.Styles, width int) string {
	return renderEvent(styles, width, e.baseEvent, "Zone \""+e.Name+"\"", e.Details, "", 0, false)
}

// BridgeHomeEvent represents a bridge_home update event.
type BridgeHomeEvent struct {
	baseEvent
	Details string
}

func (e *BridgeHomeEvent) Parse(bridgeID, bridgeName, eventType string, data json.RawMessage, state *hue.BridgeState) (Event, error) {
	var updates []hue.ResourceUpdate
	if err := json.Unmarshal(data, &updates); err != nil {
		return nil, fmt.Errorf("parse bridge_home event: %w", err)
	}

	if len(updates) == 0 {
		return nil, fmt.Errorf("no updates in bridge_home event")
	}

	update := updates[0]

	e.baseEvent = baseEvent{
		bridgeID:     bridgeID,
		bridgeName:   bridgeName,
		resourceType: update.Type,
		resourceID:   update.ID,
		eventType:    eventType,
		timestamp:    time.Now(),
	}

	// Set details based on event type
	switch eventType {
	case "add":
		e.Details = "added"
	case "update":
		e.Details = "updated"
	case "delete":
		e.Details = "deleted"
	default:
		e.Details = eventType
	}

	return e, nil
}

func (e *BridgeHomeEvent) Render(styles *ui.Styles, width int) string {
	return renderEvent(styles, width, e.baseEvent, "Bridge Home", e.Details, "", 0, false)
}

// renderEvent is a helper function to render events with consistent formatting.
// getResourceTypeColor returns the appropriate entity color for a resource type.
func getResourceTypeColor(styles *ui.Styles, resourceType string) color.Color {
	switch resourceType {
	case "light":
		return styles.Theme.EntityLight
	case "grouped_light", "grouped_light_level", "grouped_motion":
		return styles.Theme.EntityRoom // grouped resources are room-level
	case "room":
		return styles.Theme.EntityRoom
	case "zone":
		return styles.Theme.EntityZone
	case "scene":
		return styles.Theme.EntityScene
	case "smart_scene":
		return styles.Theme.EntityScene // same color as regular scenes
	case "device", "device_power", "device_software_update":
		return styles.Theme.EntityDevice
	case "zigbee_connectivity", "zgp_connectivity", "wifi_connectivity":
		return styles.Theme.EntityDevice // connectivity is device-related
	case "button", "relative_rotary", "contact", "tamper":
		return styles.Theme.EntityDevice // input sensors are device-related
	case "motion", "light_level", "temperature":
		return styles.Theme.EntityDevice // environmental sensors are device-related
	case "bridge", "bridge_home":
		return styles.Theme.EntityBridge
	case "entertainment", "entertainment_configuration":
		return styles.Theme.EntityEntertainment
	default:
		return styles.Theme.TextMuted
	}
}

func renderEvent(styles *ui.Styles, width int, base baseEvent, name, details, indicatorColor string, brightness float64, isOn bool) string {
	timeStr := base.timestamp.Format("15:04:05")
	typeStyle := lipgloss.NewStyle().Foreground(styles.Theme.Success)
	typeIndicator := typeStyle.Render("e")

	var indicator string
	if (base.resourceType == "light" || base.resourceType == "grouped_light") && isOn {
		indicatorChar := brightnessIndicatorLog(brightness)
		if indicatorColor != "" {
			indicatorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(indicatorColor))
			indicator = indicatorStyle.Render(indicatorChar) + " "
		} else {
			indicator = lipgloss.NewStyle().Foreground(styles.Theme.Success).Render(indicatorChar) + " "
		}
	} else if (base.resourceType == "light" || base.resourceType == "grouped_light") && !isOn {
		indicator = styles.Dimmed.Render("○") + " "
	}

	// Bridge name with entity color, dimmed brackets
	bridgeStr := ""
	if base.bridgeName != "" {
		bridgeStyle := lipgloss.NewStyle().Foreground(styles.Theme.EntityBridge)
		bridgeStr = styles.Dimmed.Render("[") + bridgeStyle.Render(base.bridgeName) + styles.Dimmed.Render("]") + " "
	}

	// Entity name with entity color (the actual name like "Office Shelf Corner")
	entityColor := getResourceTypeColor(styles, base.resourceType)
	entityStyle := lipgloss.NewStyle().Foreground(entityColor)

	var line string
	if name != "" {
		if details != "" {
			line = fmt.Sprintf("%s %s %s%s%s %s → %s",
				styles.Dimmed.Render(timeStr),
				typeIndicator,
				bridgeStr,
				indicator,
				base.resourceType,
				entityStyle.Render(name),
				details)
		} else {
			line = fmt.Sprintf("%s %s %s%s%s %s",
				styles.Dimmed.Render(timeStr),
				typeIndicator,
				bridgeStr,
				indicator,
				base.resourceType,
				entityStyle.Render(name))
		}
	} else {
		if details != "" {
			line = fmt.Sprintf("%s %s %s%s%s → %s",
				styles.Dimmed.Render(timeStr),
				typeIndicator,
				bridgeStr,
				indicator,
				base.resourceType,
				details)
		} else {
			line = fmt.Sprintf("%s %s %s%s%s update",
				styles.Dimmed.Render(timeStr),
				typeIndicator,
				bridgeStr,
				indicator,
				base.resourceType)
		}
	}

	// Truncate if width is provided and line exceeds it
	if width > 0 && lipgloss.Width(line) > width {
		line = ansi.Truncate(line, width, "…")
	}

	return line
}

// brightnessIndicatorLog returns a character representing the brightness level.
func brightnessIndicatorLog(brightness float64) string {
	switch {
	case brightness <= 0:
		return "○" // off/empty
	case brightness < 37.5:
		return "◔" // quarter (1-37%)
	case brightness < 62.5:
		return "◑" // half (38-62%)
	case brightness < 87.5:
		return "◕" // three-quarters (63-87%)
	default:
		return "●" // full (88-100%)
	}
}
