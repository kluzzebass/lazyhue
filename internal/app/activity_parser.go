package app

import (
	"encoding/json"

	"github.com/kluzzebass/lazyhue/internal/hue"
)

// parseEventFromBridgeCallback creates an Event from the bridge callback parameters.
// This is a helper function that works with the current bridge callback signature
// which doesn't pass raw event data. The event is created and enriched with state data.
func parseEventFromBridgeCallback(bridgeID, bridgeName, resourceType, resourceID, eventType string, state *hue.BridgeState) Event {
	// Create a mock json.RawMessage for the event data
	// Since we don't have raw data from the callback, we create a minimal structure
	// The actual data will come from state lookup
	data := json.RawMessage(`[{"id":"` + resourceID + `","type":"` + resourceType + `"}]`)

	var event Event
	var err error

	switch resourceType {
	case "light":
		evt := &LightEvent{}
		event, err = evt.Parse(bridgeID, bridgeName, eventType, data, state)
	case "grouped_light":
		evt := &GroupedLightEvent{}
		event, err = evt.Parse(bridgeID, bridgeName, eventType, data, state)
	case "scene":
		evt := &SceneEvent{}
		event, err = evt.Parse(bridgeID, bridgeName, eventType, data, state)
	case "motion":
		evt := &MotionEvent{}
		event, err = evt.Parse(bridgeID, bridgeName, eventType, data, state)
	case "grouped_motion":
		evt := &GroupedMotionEvent{}
		event, err = evt.Parse(bridgeID, bridgeName, eventType, data, state)
	case "button":
		evt := &ButtonEvent{}
		event, err = evt.Parse(bridgeID, bridgeName, eventType, data, state)
	case "device_power":
		evt := &DevicePowerEvent{}
		event, err = evt.Parse(bridgeID, bridgeName, eventType, data, state)
	case "temperature":
		evt := &TemperatureEvent{}
		event, err = evt.Parse(bridgeID, bridgeName, eventType, data, state)
	case "light_level":
		evt := &LightLevelEvent{}
		event, err = evt.Parse(bridgeID, bridgeName, eventType, data, state)
	case "grouped_light_level":
		evt := &GroupedLightLevelEvent{}
		event, err = evt.Parse(bridgeID, bridgeName, eventType, data, state)
	case "device":
		evt := &DeviceEvent{}
		event, err = evt.Parse(bridgeID, bridgeName, eventType, data, state)
	case "room":
		evt := &RoomEvent{}
		event, err = evt.Parse(bridgeID, bridgeName, eventType, data, state)
	case "zone":
		evt := &ZoneEvent{}
		event, err = evt.Parse(bridgeID, bridgeName, eventType, data, state)
	case "bridge_home":
		evt := &BridgeHomeEvent{}
		event, err = evt.Parse(bridgeID, bridgeName, eventType, data, state)
	default:
		evt := &UnhandledEvent{}
		event, err = evt.Parse(bridgeID, bridgeName, eventType, data, state)
	}

	if err != nil {
		// Fallback to unhandled event
		evt := &UnhandledEvent{}
		event, _ = evt.Parse(bridgeID, bridgeName, eventType, data, state)
	}

	return event
}
