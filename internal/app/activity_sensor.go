package app

import (
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/kluzzebass/lazyhue/internal/hue"
	"github.com/kluzzebass/lazyhue/internal/ui"
)

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
			if motion.Owner.Rid != "" {
				if device, ok := state.GetDevice(motion.Owner.Rid); ok {
					e.Name = device.DeviceName("")
				}
			}
			if motion.Motion.Motion {
				e.Details = "motion detected"
			} else {
				e.Details = "clear"
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
	Name      string
	Details   string
	ControlID int // Which button on the device (1-4 for dimmer switch)
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

	// Get device name from owner in event data
	if state != nil && update.Owner != nil {
		if device, ok := state.GetDevice(update.Owner.Rid); ok {
			e.Name = device.DeviceName("")
		}
	}

	// Get control ID from button resource in state (event data doesn't include metadata)
	if state != nil {
		if btn, ok := state.GetButton(update.ID); ok {
			e.ControlID = btn.Metadata.ControlId
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
	// Include button number in the details if we have it
	details := e.Details
	if e.ControlID > 0 {
		details = fmt.Sprintf("btn %d → %s", e.ControlID, e.Details)
	}
	return renderEvent(styles, width, e.baseEvent, e.Name, details, "", 0, false)
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
			if temp.Owner.Rid != "" {
				if device, ok := state.GetDevice(temp.Owner.Rid); ok {
					e.Name = device.DeviceName("")
				}
			}
			e.Details = fmt.Sprintf("%.1f°C", temp.Temperature.Temperature)
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
			if ll.Owner.Rid != "" {
				if device, ok := state.GetDevice(ll.Owner.Rid); ok {
					e.Name = device.DeviceName("")
				}
			}
			// Convert from Hue's log scale to approximate lux
			lux := float64(ll.Light.LightLevel-1) / 10000.0
			lux = 100 * (lux * lux * lux) // Rough approximation
			e.Details = fmt.Sprintf("%.0f lux", lux)
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
			for _, svc := range room.Services {
				if svc.Rtype == "grouped_light_level" && svc.Rid == update.ID {
					if room.Metadata.Name != "" {
						e.Name = room.Metadata.Name
					}
					var totalLux float64
					var count int
					for _, child := range room.Children {
						if string(child.Rtype) != "device" {
							continue
						}
						if device, ok := state.GetDevice(child.Rid); ok {
							if hasLevel, level := state.GetDeviceLightLevel(device); hasLevel && level > 0 {
								// Convert from Hue's log scale to lux: 10^((level-1)/10000)
								lux := math.Pow(10, float64(level-1)/10000)
								totalLux += lux
								count++
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
		allZones := state.AllZones()
		for _, zone := range allZones {
			for _, svc := range zone.Services {
				if svc.Rtype == "grouped_light_level" && svc.Rid == update.ID {
					if zone.Metadata.Name != "" {
						e.Name = zone.Metadata.Name
					}
					var totalLux float64
					var count int
					for _, child := range zone.Children {
						if string(child.Rtype) != "device" {
							continue
						}
						if device, ok := state.GetDevice(child.Rid); ok {
							if hasLevel, level := state.GetDeviceLightLevel(device); hasLevel && level > 0 {
								// Convert from Hue's log scale to lux: 10^((level-1)/10000)
								lux := math.Pow(10, float64(level-1)/10000)
								totalLux += lux
								count++
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
			for _, svc := range room.Services {
				if svc.Rtype == "grouped_motion" && svc.Rid == update.ID {
					if room.Metadata.Name != "" {
						e.Name = room.Metadata.Name
					}
					var anyMotion bool
					var sensorCount int
					for _, child := range room.Children {
						if string(child.Rtype) != "device" {
							continue
						}
						if device, ok := state.GetDevice(child.Rid); ok {
							if hasMotion, isDetecting := state.GetDeviceMotionState(device); hasMotion {
								sensorCount++
								if isDetecting {
									anyMotion = true
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
		// Check zones for this grouped_motion service
		allZones := state.AllZones()
		for _, zone := range allZones {
			for _, svc := range zone.Services {
				if svc.Rtype == "grouped_motion" && svc.Rid == update.ID {
					if zone.Metadata.Name != "" {
						e.Name = zone.Metadata.Name
					}
					var anyMotion bool
					var sensorCount int
					for _, child := range zone.Children {
						if string(child.Rtype) != "device" {
							continue
						}
						if device, ok := state.GetDevice(child.Rid); ok {
							if hasMotion, isDetecting := state.GetDeviceMotionState(device); hasMotion {
								sensorCount++
								if isDetecting {
									anyMotion = true
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
		e.Details = "motion updated"
	}

	return e, nil
}

func (e *GroupedMotionEvent) Render(styles *ui.Styles, width int) string {
	return renderEvent(styles, width, e.baseEvent, e.Name, e.Details, "", 0, false)
}
