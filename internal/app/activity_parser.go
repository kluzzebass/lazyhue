package app

import (
	"encoding/json"
	"time"

	"github.com/kluzzebass/lazyhue/internal/hue"
)

// parseEventFromBridgeCallback creates an Event from the bridge callback parameters.
// This is a helper function that works with the current bridge callback signature
// which doesn't pass raw event data. The event is created and enriched with state data.
func parseEventFromBridgeCallback(bridgeID, bridgeName, resourceType, resourceID, eventType string, receivedAt time.Time, state *hue.BridgeState) Event {
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
	case "relative_rotary":
		evt := &RelativeRotaryEvent{}
		event, err = evt.Parse(bridgeID, bridgeName, eventType, data, state)
	case "contact":
		evt := &ContactEvent{}
		event, err = evt.Parse(bridgeID, bridgeName, eventType, data, state)
	case "tamper":
		evt := &TamperEvent{}
		event, err = evt.Parse(bridgeID, bridgeName, eventType, data, state)
	case "zigbee_connectivity":
		evt := &ZigbeeConnectivityEvent{}
		event, err = evt.Parse(bridgeID, bridgeName, eventType, data, state)
	case "zgp_connectivity":
		evt := &ZgpConnectivityEvent{}
		event, err = evt.Parse(bridgeID, bridgeName, eventType, data, state)
	case "wifi_connectivity":
		evt := &WifiConnectivityEvent{}
		event, err = evt.Parse(bridgeID, bridgeName, eventType, data, state)
	case "device_software_update":
		evt := &DeviceSoftwareUpdateEvent{}
		event, err = evt.Parse(bridgeID, bridgeName, eventType, data, state)
	case "smart_scene":
		evt := &SmartSceneEvent{}
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
	// Device-owned simple events (show device name + event type)
	case "bell_button", "camera_motion", "speaker":
		evt := &SimpleDeviceEvent{}
		event, err = evt.ParseWithType(bridgeID, bridgeName, resourceType, eventType, data, state)
	// Named simple events (show metadata name + event type)
	case "entertainment", "entertainment_configuration", "behavior_script", "behavior_instance",
		"geofence_client", "service_group":
		evt := &SimpleNamedEvent{}
		event, err = evt.ParseWithType(bridgeID, bridgeName, resourceType, eventType, data, state)
	// Simple events without specific data (show resource type + event type)
	case "bridge", "homekit", "matter", "matter_fabric", "geolocation", "public_image",
		"auth_v1", "motion_area_configuration", "motion_area_candidate", "zigbee_device_discovery",
		"convenience_area_motion", "security_area_motion", "clip":
		evt := &SimpleEvent{}
		event, err = evt.ParseWithType(bridgeID, bridgeName, resourceType, eventType, data, state)
	default:
		evt := &UnhandledEvent{}
		event, err = evt.Parse(bridgeID, bridgeName, eventType, data, state)
	}

	if err != nil {
		// Fallback to unhandled event
		evt := &UnhandledEvent{}
		event, _ = evt.Parse(bridgeID, bridgeName, eventType, data, state)
	}

	// Set the received timestamp from the bridge callback
	if setter, ok := event.(interface{ SetTimestamp(time.Time) }); ok {
		setter.SetTimestamp(receivedAt)
	}

	return event
}
