package app2

import (
	"encoding/json"

	"github.com/kluzzebass/lazyhue/internal/hue"
)

// parseEventFromBridgeCallback creates an Event from the bridge callback parameters.
// This is a helper function that works with the current bridge callback signature
// which doesn't pass raw event data. The event is created and enriched with state data.
func parseEventFromBridgeCallback(bridgeID, resourceType, resourceID, eventType string, state *hue.BridgeState) Event {
	// Create a mock json.RawMessage for the event data
	// Since we don't have raw data from the callback, we create a minimal structure
	// The actual data will come from state lookup
	data := json.RawMessage(`[{"id":"` + resourceID + `","type":"` + resourceType + `"}]`)

	var event Event
	var err error

	switch resourceType {
	case "light":
		evt := &LightEvent{}
		event, err = evt.Parse(bridgeID, eventType, data, state)
	case "grouped_light":
		evt := &GroupedLightEvent{}
		event, err = evt.Parse(bridgeID, eventType, data, state)
	case "scene":
		evt := &SceneEvent{}
		event, err = evt.Parse(bridgeID, eventType, data, state)
	case "motion":
		evt := &MotionEvent{}
		event, err = evt.Parse(bridgeID, eventType, data, state)
	case "temperature":
		evt := &TemperatureEvent{}
		event, err = evt.Parse(bridgeID, eventType, data, state)
	case "light_level":
		evt := &LightLevelEvent{}
		event, err = evt.Parse(bridgeID, eventType, data, state)
	case "grouped_light_level":
		evt := &GroupedLightLevelEvent{}
		event, err = evt.Parse(bridgeID, eventType, data, state)
	case "device":
		evt := &DeviceEvent{}
		event, err = evt.Parse(bridgeID, eventType, data, state)
	default:
		evt := &UnhandledEvent{}
		event, err = evt.Parse(bridgeID, eventType, data, state)
	}

	if err != nil {
		// Fallback to unhandled event
		evt := &UnhandledEvent{}
		event, _ = evt.Parse(bridgeID, eventType, data, state)
	}

	return event
}
