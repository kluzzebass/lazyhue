package app

// Field ID prefixes for UI components.
// These constants ensure consistency between field creation (rendering*.go)
// and field lookup (event handling in rendering.go).
const (
	// Name fields
	FieldPrefixName       = "name:"        // light device names (format: "name:<deviceID>")
	FieldPrefixDeviceName = "device-name:" // device names (format: "device-name:<deviceID>")
	FieldPrefixRoomName   = "room-name:"   // room names (format: "room-name:<roomID>")
	FieldPrefixZoneName   = "zone-name:"   // zone names (format: "zone-name:<zoneID>")
	FieldPrefixSceneName  = "scene-name:"  // scene names (format: "scene-name:<sceneID>")
	FieldPrefixBridgeName = "bridge-name:" // bridge names (format: "bridge-name:<deviceID>")

	// Archetype fields
	FieldPrefixArchetype     = "archetype:"      // device archetypes (format: "archetype:<deviceID>")
	FieldPrefixRoomArchetype = "room-archetype:" // room archetypes (format: "room-archetype:<roomID>")
	FieldPrefixZoneArchetype = "zone-archetype:" // zone archetypes (format: "zone-archetype:<zoneID>")

	// Light control fields
	FieldPrefixPowerupPreset = "powerup-preset:" // powerup presets (format: "powerup-preset:<lightID>")
	FieldPrefixLightRoom     = "light-room:"     // light room assignment (format: "light-room:<lightID>")
	FieldPrefixLightZones    = "light-zones:"    // light zone assignment (format: "light-zones:<lightID>")
	FieldPrefixEffectSpeed   = "effect-speed:"   // effect speed (format: "effect-speed:<lightID>")

	// Device control fields
	FieldPrefixDeviceRoom = "device-room:" // device room assignment (format: "device-room:<deviceID>")
	FieldPrefixIdentify   = "identify:"    // identify device (format: "identify:<deviceID>")

	// Bridge fields
	FieldPrefixBridgeIdentify = "bridge-identify:" // identify bridge (format: "bridge-identify:<bridgeID>")

	// Room/Zone control fields
	FieldPrefixRoomPower       = "room-power:"       // room power (format: "room-power:<roomID>")
	FieldPrefixZonePower       = "zone-power:"       // zone power (format: "zone-power:<zoneID>")
	FieldPrefixRoomBrightness  = "room-brightness:"  // room brightness (format: "room-brightness:<roomID>")
	FieldPrefixZoneBrightness  = "zone-brightness:"  // zone brightness (format: "zone-brightness:<zoneID>")
	FieldPrefixRoomCreateScene = "room-create-scene:" // create scene for room (format: "room-create-scene:<roomID>")
	FieldPrefixZoneCreateScene = "zone-create-scene:" // create scene for zone (format: "zone-create-scene:<zoneID>")

	// Scene fields
	FieldPrefixSceneRecall     = "scene-recall:"      // recall scene (format: "scene-recall:<sceneID>")
	FieldPrefixSmartSceneToggle = "smartscene-toggle:" // toggle smart scene (format: "smartscene-toggle:<sceneID>")

	// Timed effects fields
	FieldPrefixTimedEffectStop     = "timed-effect-stop:"     // stop timed effect (format: "timed-effect-stop:<lightID>")
	FieldPrefixTimedEffectDuration = "timed-effect-duration:" // timed effect duration (format: "timed-effect-duration:<lightID>")
	FieldPrefixTimedEffectTrigger  = "timed-effect-trigger:"  // trigger timed effect (format: "timed-effect-trigger:<lightID>")

	// Signaling fields
	FieldPrefixSignalStop     = "signal-stop:"     // stop signal (format: "signal-stop:<lightID>")
	FieldPrefixSignalDuration = "signal-duration:" // signal duration (format: "signal-duration:<lightID>")
	FieldPrefixSignalTrigger  = "signal-trigger:"  // trigger signal (format: "signal-trigger:<lightID>")

	// Gradient fields
	FieldPrefixGradientMode   = "gradient-mode:"   // gradient mode (format: "gradient-mode:<lightID>")
	FieldPrefixGradientPoints = "gradient-points:" // gradient points (format: "gradient-points:<lightID>")

	// Sensor fields
	FieldPrefixMotionEnabled     = "motion-enabled:"     // motion sensor enabled (format: "motion-enabled:<sensorID>")
	FieldPrefixMotionSensitivity = "motion-sensitivity:" // motion sensitivity (format: "motion-sensitivity:<sensorID>")
	FieldPrefixTempEnabled       = "temp-enabled:"       // temperature sensor enabled (format: "temp-enabled:<sensorID>")
	FieldPrefixLightLevelEnabled = "ll-enabled:"         // light level sensor enabled (format: "ll-enabled:<sensorID>")
)

// Field ID constructors - use these instead of string concatenation

// Name field constructors

func FieldIDName(deviceID string) string {
	return FieldPrefixName + deviceID
}

func FieldIDDeviceName(deviceID string) string {
	return FieldPrefixDeviceName + deviceID
}

func FieldIDRoomName(roomID string) string {
	return FieldPrefixRoomName + roomID
}

func FieldIDZoneName(zoneID string) string {
	return FieldPrefixZoneName + zoneID
}

func FieldIDSceneName(sceneID string) string {
	return FieldPrefixSceneName + sceneID
}

func FieldIDBridgeName(deviceID string) string {
	return FieldPrefixBridgeName + deviceID
}

// Archetype field constructors

func FieldIDArchetype(deviceID string) string {
	return FieldPrefixArchetype + deviceID
}

func FieldIDRoomArchetype(roomID string) string {
	return FieldPrefixRoomArchetype + roomID
}

func FieldIDZoneArchetype(zoneID string) string {
	return FieldPrefixZoneArchetype + zoneID
}

// Light control field constructors

func FieldIDPowerupPreset(lightID string) string {
	return FieldPrefixPowerupPreset + lightID
}

func FieldIDLightRoom(lightID string) string {
	return FieldPrefixLightRoom + lightID
}

func FieldIDLightZones(lightID string) string {
	return FieldPrefixLightZones + lightID
}

func FieldIDEffectSpeed(lightID string) string {
	return FieldPrefixEffectSpeed + lightID
}

// Device control field constructors

func FieldIDDeviceRoom(deviceID string) string {
	return FieldPrefixDeviceRoom + deviceID
}

func FieldIDIdentify(deviceID string) string {
	return FieldPrefixIdentify + deviceID
}

// Bridge field constructors

func FieldIDBridgeIdentify(bridgeID string) string {
	return FieldPrefixBridgeIdentify + bridgeID
}

// Room/Zone control field constructors

func FieldIDRoomPower(roomID string) string {
	return FieldPrefixRoomPower + roomID
}

func FieldIDZonePower(zoneID string) string {
	return FieldPrefixZonePower + zoneID
}

func FieldIDRoomBrightness(roomID string) string {
	return FieldPrefixRoomBrightness + roomID
}

func FieldIDZoneBrightness(zoneID string) string {
	return FieldPrefixZoneBrightness + zoneID
}

func FieldIDRoomCreateScene(roomID string) string {
	return FieldPrefixRoomCreateScene + roomID
}

func FieldIDZoneCreateScene(zoneID string) string {
	return FieldPrefixZoneCreateScene + zoneID
}

// Scene field constructors

func FieldIDSceneRecall(sceneID string) string {
	return FieldPrefixSceneRecall + sceneID
}

func FieldIDSmartSceneToggle(sceneID string) string {
	return FieldPrefixSmartSceneToggle + sceneID
}

// Timed effects field constructors

func FieldIDTimedEffectStop(lightID string) string {
	return FieldPrefixTimedEffectStop + lightID
}

func FieldIDTimedEffectDuration(lightID string) string {
	return FieldPrefixTimedEffectDuration + lightID
}

func FieldIDTimedEffectTrigger(lightID string) string {
	return FieldPrefixTimedEffectTrigger + lightID
}

// Signaling field constructors

func FieldIDSignalStop(lightID string) string {
	return FieldPrefixSignalStop + lightID
}

func FieldIDSignalDuration(lightID string) string {
	return FieldPrefixSignalDuration + lightID
}

func FieldIDSignalTrigger(lightID string) string {
	return FieldPrefixSignalTrigger + lightID
}

// Gradient field constructors

func FieldIDGradientMode(lightID string) string {
	return FieldPrefixGradientMode + lightID
}

func FieldIDGradientPoints(lightID string) string {
	return FieldPrefixGradientPoints + lightID
}

// Sensor field constructors

func FieldIDMotionEnabled(sensorID string) string {
	return FieldPrefixMotionEnabled + sensorID
}

func FieldIDMotionSensitivity(sensorID string) string {
	return FieldPrefixMotionSensitivity + sensorID
}

func FieldIDTempEnabled(sensorID string) string {
	return FieldPrefixTempEnabled + sensorID
}

func FieldIDLightLevelEnabled(sensorID string) string {
	return FieldPrefixLightLevelEnabled + sensorID
}
