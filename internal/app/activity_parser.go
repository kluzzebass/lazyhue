package app

import (
	"encoding/json"
	"time"

	"github.com/kluzzebass/lazyhue/internal/hue"
)

// parseEventFromUpdate creates an Event from a ResourceUpdate.
// The update contains all the actual event data from the SSE stream.
func parseEventFromUpdate(bridgeID, bridgeName string, update hue.ResourceUpdate, eventType string, receivedAt time.Time, state *hue.BridgeState) Event {
	// Serialize the update to JSON for the Parse methods
	// This preserves all the event data (owner, button info, etc.)
	data, err := json.Marshal([]hue.ResourceUpdate{update})
	if err != nil {
		// Fallback to unhandled event
		evt := &UnhandledEvent{}
		evt.baseEvent = baseEvent{
			bridgeID:     bridgeID,
			bridgeName:   bridgeName,
			resourceType: update.Type,
			resourceID:   update.ID,
			eventType:    eventType,
			timestamp:    receivedAt,
		}
		return evt
	}

	var event Event

	switch update.Type {
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
	case "homekit":
		evt := &HomekitEvent{}
		event, err = evt.Parse(bridgeID, bridgeName, eventType, data, state)
	case "matter":
		evt := &MatterEvent{}
		event, err = evt.Parse(bridgeID, bridgeName, eventType, data, state)
	case "matter_fabric":
		evt := &MatterFabricEvent{}
		event, err = evt.Parse(bridgeID, bridgeName, eventType, data, state)
	case "geolocation":
		evt := &GeolocationEvent{}
		event, err = evt.Parse(bridgeID, bridgeName, eventType, data, state)
	case "zigbee_device_discovery":
		evt := &ZigbeeDiscoveryEvent{}
		event, err = evt.Parse(bridgeID, bridgeName, eventType, data, state)
	// Device-owned simple events (show device name + event type)
	case "bell_button", "camera_motion", "speaker":
		evt := &SimpleDeviceEvent{}
		event, err = evt.ParseWithType(bridgeID, bridgeName, update.Type, eventType, data, state)
	// Named simple events (show metadata name + event type)
	case "entertainment", "entertainment_configuration", "behavior_script", "behavior_instance",
		"geofence_client", "service_group":
		evt := &SimpleNamedEvent{}
		event, err = evt.ParseWithType(bridgeID, bridgeName, update.Type, eventType, data, state)
	// Simple events without specific data (show resource type + event type)
	case "bridge", "public_image", "auth_v1", "motion_area_configuration", "motion_area_candidate",
		"convenience_area_motion", "security_area_motion", "clip":
		evt := &SimpleEvent{}
		event, err = evt.ParseWithType(bridgeID, bridgeName, update.Type, eventType, data, state)
	default:
		evt := &UnhandledEvent{}
		event, err = evt.Parse(bridgeID, bridgeName, eventType, data, state)
	}

	if err != nil {
		// Fallback to unhandled event
		evt := &UnhandledEvent{}
		event, _ = evt.Parse(bridgeID, bridgeName, eventType, data, state)
	}

	// Set the received timestamp
	if setter, ok := event.(interface{ SetTimestamp(time.Time) }); ok {
		setter.SetTimestamp(receivedAt)
	}

	return event
}
