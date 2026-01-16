package hue

// EffectDisplayNames maps Hue API effect names to user-friendly display names.
var EffectDisplayNames = map[string]string{
	"no_effect":  "None",
	"candle":     "Candle",
	"fire":       "Fire",
	"prism":      "Prism",
	"sparkle":    "Sparkle",
	"opal":       "Opal",
	"glisten":    "Glisten",
	"cosmos":     "Cosmos",
	"sunbeam":    "Sunbeam",
	"enchant":    "Enchant",
	"underwater": "Underwater",
}

// EffectDisplayName returns the user-friendly display name for an effect, or the raw name if unknown.
func EffectDisplayName(effect string) string {
	return displayName(EffectDisplayNames, effect)
}

// SignalingModeDisplayNames maps Hue API signaling mode names to user-friendly display names.
var SignalingModeDisplayNames = map[string]string{
	"no_signal":    "No Signal",
	"on_off":       "On/Off",
	"on_off_color": "On/Off Color",
	"alternating":  "Alternating",
}

// SignalingModeDisplayName returns the user-friendly display name for a signaling mode, or the raw name if unknown.
func SignalingModeDisplayName(mode string) string {
	return displayName(SignalingModeDisplayNames, mode)
}

// PowerupPresetDisplayNames maps Hue API power-on preset names to user-friendly display names.
var PowerupPresetDisplayNames = map[string]string{
	"safety":        "Safety",
	"powerfail":     "Power Fail",
	"last_on_state": "Last On State",
	"custom":        "Custom",
}

// PowerupPresetDisplayName returns the user-friendly display name for a power-on preset, or the raw name if unknown.
func PowerupPresetDisplayName(preset string) string {
	return displayName(PowerupPresetDisplayNames, preset)
}

// DeviceServiceDisplayNames maps Hue API device service types to user-friendly display names.
var DeviceServiceDisplayNames = map[string]string{
	"device":                      "Device",
	"bridge_home":                 "Bridge Home",
	"room":                        "Room",
	"zone":                        "Zone",
	"service_group":               "Service Group",
	"light":                       "Light",
	"button":                      "Button",
	"bell_button":                 "Bell Button",
	"relative_rotary":             "Rotary Dial",
	"temperature":                 "Temperature Sensor",
	"light_level":                 "Light Level Sensor",
	"motion":                      "Motion Sensor",
	"camera_motion":               "Camera Motion",
	"entertainment":               "Entertainment",
	"contact":                     "Contact Sensor",
	"tamper":                      "Tamper Sensor",
	"convenience_area_motion":     "Convenience Area Motion",
	"security_area_motion":        "Security Area Motion",
	"speaker":                     "Speaker",
	"grouped_light":               "Grouped Light",
	"grouped_motion":              "Grouped Motion",
	"grouped_light_level":         "Grouped Light Level",
	"device_power":                "Battery",
	"device_software_update":      "Software Update",
	"zigbee_connectivity":         "Zigbee Connectivity",
	"zgp_connectivity":            "ZGP Connectivity",
	"bridge":                      "Bridge",
	"motion_area_candidate":       "Motion Area Candidate",
	"wifi_connectivity":           "WiFi Connectivity",
	"zigbee_device_discovery":     "Zigbee Discovery",
	"homekit":                     "HomeKit",
	"matter":                      "Matter",
	"matter_fabric":               "Matter Fabric",
	"scene":                       "Scene",
	"entertainment_configuration": "Entertainment Config",
	"public_image":                "Public Image",
	"auth_v1":                     "Auth V1",
	"behavior_script":             "Behavior Script",
	"behavior_instance":           "Behavior Instance",
	"geofence_client":             "Geofence Client",
	"geolocation":                 "Geolocation",
	"smart_scene":                 "Smart Scene",
	"motion_area_configuration":   "Motion Area Config",
	"clip":                        "CLIP",
}

// DeviceServiceDisplayName returns the user-friendly display name for a device service type, or the raw name if unknown.
func DeviceServiceDisplayName(serviceType string) string {
	return displayName(DeviceServiceDisplayNames, serviceType)
}

// ProductArchetypeDisplayNames maps Hue API product archetype names to user-friendly display names.
var ProductArchetypeDisplayNames = map[string]string{
	"bridge_v2":              "Bridge V2",
	"bridge_v3":              "Bridge V3",
	"unknown_archetype":      "Unknown Archetype",
	"classic_bulb":           "Classic Bulb",
	"sultan_bulb":            "Sultan Bulb",
	"flood_bulb":             "Flood Bulb",
	"spot_bulb":              "Spot Bulb",
	"candle_bulb":            "Candle Bulb",
	"luster_bulb":            "Luster Bulb",
	"pendant_round":          "Pendant Round",
	"pendant_long":           "Pendant Long",
	"ceiling_round":          "Ceiling Round",
	"ceiling_square":         "Ceiling Square",
	"floor_shade":            "Floor Shade",
	"floor_lantern":          "Floor Lantern",
	"table_shade":            "Table Shade",
	"recessed_ceiling":       "Recessed Ceiling",
	"recessed_floor":         "Recessed Floor",
	"single_spot":            "Single Spot",
	"double_spot":            "Double Spot",
	"table_wash":             "Table Wash",
	"wall_lantern":           "Wall Lantern",
	"wall_shade":             "Wall Shade",
	"flexible_lamp":          "Flexible Lamp",
	"ground_spot":            "Ground Spot",
	"wall_spot":              "Wall Spot",
	"plug":                   "Plug",
	"hue_go":                 "Hue Go",
	"hue_lightstrip":         "Hue Lightstrip",
	"hue_iris":               "Hue Iris",
	"hue_bloom":              "Hue Bloom",
	"bollard":                "Bollard",
	"wall_washer":            "Wall Washer",
	"hue_play":               "Hue Play",
	"hue_chime":              "Hue Chime",
	"vintage_bulb":           "Vintage Bulb",
	"vintage_candle_bulb":    "Vintage Candle Bulb",
	"ellipse_bulb":           "Ellipse Bulb",
	"triangle_bulb":          "Triangle Bulb",
	"small_globe_bulb":       "Small Globe Bulb",
	"large_globe_bulb":       "Large Globe Bulb",
	"edison_bulb":            "Edison Bulb",
	"christmas_tree":         "Christmas Tree",
	"string_light":           "String Light",
	"hue_centris":            "Hue Centris",
	"hue_lightstrip_tv":      "Hue Lightstrip TV",
	"hue_lightstrip_pc":      "Hue Lightstrip PC",
	"hue_tube":               "Hue Tube",
	"hue_signe":              "Hue Signe",
	"pendant_spot":           "Pendant Spot",
	"ceiling_horizontal":     "Ceiling Horizontal",
	"ceiling_tube":           "Ceiling Tube",
	"up_and_down":            "Up and Down",
	"up_and_down_up":         "Up and Down (Up)",
	"up_and_down_down":       "Up and Down (Down)",
	"hue_floodlight_camera":  "Hue Floodlight Camera",
	"twilight":               "Twilight",
	"twilight_front":         "Twilight Front",
	"twilight_back":          "Twilight Back",
	"hue_play_wallwasher":    "Hue Play Wallwasher",
	"hue_omniglow":           "Hue Omniglow",
	"hue_neon":               "Hue Neon",
	"string_globe":           "String Globe",
	"string_permanent":       "String Permanent",
}

// ProductArchetypeDisplayName returns the user-friendly display name for a product archetype, or the raw name if unknown.
func ProductArchetypeDisplayName(archetype string) string {
	return displayName(ProductArchetypeDisplayNames, archetype)
}

// RoomArchetypeDisplayNames maps Hue API room archetype names to user-friendly display names.
var RoomArchetypeDisplayNames = map[string]string{
	"attic":        "Attic",
	"balcony":      "Balcony",
	"barbecue":     "Barbecue",
	"bathroom":     "Bathroom",
	"bedroom":      "Bedroom",
	"carport":      "Carport",
	"closet":       "Closet",
	"computer":     "Computer",
	"dining":       "Dining Room",
	"downstairs":   "Downstairs",
	"driveway":     "Driveway",
	"front_door":   "Front Door",
	"garage":       "Garage",
	"garden":       "Garden",
	"guest_room":   "Guest Room",
	"gym":          "Gym",
	"hallway":      "Hallway",
	"home":         "Home",
	"kids_bedroom": "Kids Bedroom",
	"kitchen":      "Kitchen",
	"laundry_room": "Laundry Room",
	"living_room":  "Living Room",
	"lounge":       "Lounge",
	"man_cave":     "Man Cave",
	"music":        "Music Room",
	"nursery":      "Nursery",
	"office":       "Office",
	"other":        "Other",
	"pool":         "Pool",
	"porch":        "Porch",
	"reading":      "Reading Room",
	"recreation":   "Recreation",
	"staircase":    "Staircase",
	"storage":      "Storage",
	"studio":       "Studio",
	"terrace":      "Terrace",
	"toilet":       "Toilet",
	"top_floor":    "Top Floor",
	"tv":           "TV Room",
	"upstairs":     "Upstairs",
}

// RoomArchetypeDisplayName returns the user-friendly display name for a room archetype.
func RoomArchetypeDisplayName(archetype string) string {
	return displayName(RoomArchetypeDisplayNames, archetype)
}

// RoomArchetypeList returns a sorted list of all room archetypes for use in selection UIs.
func RoomArchetypeList() []string {
	return []string{
		"living_room", "bedroom", "bathroom", "kitchen", "dining",
		"office", "hallway", "staircase", "closet", "storage",
		"laundry_room", "guest_room", "kids_bedroom", "nursery",
		"lounge", "tv", "reading", "music", "computer", "gym",
		"recreation", "man_cave", "studio", "garage", "carport",
		"garden", "terrace", "balcony", "porch", "pool", "barbecue",
		"driveway", "front_door", "attic", "top_floor", "upstairs",
		"downstairs", "home", "other",
	}
}

