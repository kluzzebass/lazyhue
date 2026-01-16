package app

// Field ID prefixes for UI components.
// These constants ensure consistency between field creation (rendering.go)
// and field lookup (rename.go, event handling).
const (
	// FieldPrefixName is for light device names (format: "name:<deviceID>")
	FieldPrefixName = "name:"

	// FieldPrefixDeviceName is for device names (format: "device-name:<deviceID>")
	FieldPrefixDeviceName = "device-name:"

	// FieldPrefixRoomName is for room names (format: "room-name:<roomID>")
	FieldPrefixRoomName = "room-name:"

	// FieldPrefixZoneName is for zone names (format: "zone-name:<zoneID>")
	FieldPrefixZoneName = "zone-name:"

	// FieldPrefixSceneName is for scene names (format: "scene-name:<sceneID>")
	FieldPrefixSceneName = "scene-name:"

	// FieldPrefixBridgeName is for bridge names (format: "bridge-name:<deviceID>")
	FieldPrefixBridgeName = "bridge-name:"

	// FieldPrefixArchetype is for device archetypes (format: "archetype:<deviceID>")
	FieldPrefixArchetype = "archetype:"

	// FieldPrefixRoomArchetype is for room archetypes (format: "room-archetype:<roomID>")
	FieldPrefixRoomArchetype = "room-archetype:"

	// FieldPrefixZoneArchetype is for zone archetypes (format: "zone-archetype:<zoneID>")
	FieldPrefixZoneArchetype = "zone-archetype:"

	// FieldPrefixPowerupPreset is for powerup presets (format: "powerup-preset:<lightID>")
	FieldPrefixPowerupPreset = "powerup-preset:"
)

// Field ID constructors - use these instead of string concatenation

// FieldIDName returns a field ID for a light's device name.
func FieldIDName(deviceID string) string {
	return FieldPrefixName + deviceID
}

// FieldIDDeviceName returns a field ID for a device name.
func FieldIDDeviceName(deviceID string) string {
	return FieldPrefixDeviceName + deviceID
}

// FieldIDRoomName returns a field ID for a room name.
func FieldIDRoomName(roomID string) string {
	return FieldPrefixRoomName + roomID
}

// FieldIDZoneName returns a field ID for a zone name.
func FieldIDZoneName(zoneID string) string {
	return FieldPrefixZoneName + zoneID
}

// FieldIDSceneName returns a field ID for a scene name.
func FieldIDSceneName(sceneID string) string {
	return FieldPrefixSceneName + sceneID
}

// FieldIDBridgeName returns a field ID for a bridge name.
func FieldIDBridgeName(deviceID string) string {
	return FieldPrefixBridgeName + deviceID
}

// FieldIDArchetype returns a field ID for a device archetype.
func FieldIDArchetype(deviceID string) string {
	return FieldPrefixArchetype + deviceID
}

// FieldIDRoomArchetype returns a field ID for a room archetype.
func FieldIDRoomArchetype(roomID string) string {
	return FieldPrefixRoomArchetype + roomID
}

// FieldIDZoneArchetype returns a field ID for a zone archetype.
func FieldIDZoneArchetype(zoneID string) string {
	return FieldPrefixZoneArchetype + zoneID
}

// FieldIDPowerupPreset returns a field ID for a powerup preset.
func FieldIDPowerupPreset(lightID string) string {
	return FieldPrefixPowerupPreset + lightID
}
