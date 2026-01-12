package hue

import (
	"sort"
	"sync"

	"github.com/kluzzebass/lazyhue/internal/hueclient"
)

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
	if name, ok := EffectDisplayNames[effect]; ok {
		return name
	}
	return effect
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
	if name, ok := RoomArchetypeDisplayNames[archetype]; ok {
		return name
	}
	return archetype
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

// BridgeState holds cached entities from a bridge using hueclient types.
// Relationships are stored as IDs and resolved on-demand.
type BridgeState struct {
	mu sync.RWMutex

	Lights                      map[string]hueclient.LightGet
	Rooms                       map[string]hueclient.RoomGet
	Zones                       map[string]hueclient.RoomGet // Zones use the same type as Rooms
	Scenes                      map[string]hueclient.SceneGet
	GroupedLights               map[string]hueclient.GroupedLightGet
	Devices                     map[string]hueclient.DeviceGet
	MotionSensors               map[string]hueclient.MotionGet
	Temperatures                map[string]hueclient.TemperatureGet
	LightLevels                 map[string]hueclient.LightLevelGet
	DevicePowers                map[string]hueclient.DevicePowerGet
	EntertainmentConfigurations map[string]EntertainmentConfiguration
	WifiConnectivity            []WifiConnectivity            // WiFi status (Bridge Pro only)
	ZigbeeConnectivity          map[string]ZigbeeConnectivity // Zigbee connectivity per device

	// Bridge-specific resources
	BridgeResource *hueclient.BridgeGet     // The bridge resource itself
	BridgeHome     *hueclient.BridgeHomeGet // The home associated with the bridge
	AuthApps       []AuthV1Entry            // Authenticated applications
}

// NewBridgeState creates an empty bridge state.
func NewBridgeState() *BridgeState {
	return &BridgeState{
		Lights:                      make(map[string]hueclient.LightGet),
		Rooms:                       make(map[string]hueclient.RoomGet),
		Zones:                       make(map[string]hueclient.RoomGet),
		Scenes:                      make(map[string]hueclient.SceneGet),
		GroupedLights:               make(map[string]hueclient.GroupedLightGet),
		Devices:                     make(map[string]hueclient.DeviceGet),
		MotionSensors:               make(map[string]hueclient.MotionGet),
		Temperatures:                make(map[string]hueclient.TemperatureGet),
		LightLevels:                 make(map[string]hueclient.LightLevelGet),
		DevicePowers:                make(map[string]hueclient.DevicePowerGet),
		EntertainmentConfigurations: make(map[string]EntertainmentConfiguration),
		ZigbeeConnectivity:          make(map[string]ZigbeeConnectivity),
	}
}

// GetLight returns a light by ID.
func (s *BridgeState) GetLight(id string) (hueclient.LightGet, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	l, ok := s.Lights[id]
	return l, ok
}

// GetRoom returns a room by ID.
func (s *BridgeState) GetRoom(id string) (hueclient.RoomGet, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	r, ok := s.Rooms[id]
	return r, ok
}

// GetZone returns a zone by ID.
func (s *BridgeState) GetZone(id string) (hueclient.RoomGet, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	z, ok := s.Zones[id]
	return z, ok
}

// GetGroupedLight returns a grouped light by ID.
func (s *BridgeState) GetGroupedLight(id string) (hueclient.GroupedLightGet, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	g, ok := s.GroupedLights[id]
	return g, ok
}

// GetScene returns a scene by ID.
func (s *BridgeState) GetScene(id string) (hueclient.SceneGet, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	sc, ok := s.Scenes[id]
	return sc, ok
}

// GetDevice returns a device by ID.
func (s *BridgeState) GetDevice(id string) (hueclient.DeviceGet, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	d, ok := s.Devices[id]
	return d, ok
}

// GetAllDevices returns a copy of all devices.
func (s *BridgeState) GetAllDevices() []hueclient.DeviceGet {
	s.mu.RLock()
	defer s.mu.RUnlock()
	devices := make([]hueclient.DeviceGet, 0, len(s.Devices))
	for _, d := range s.Devices {
		devices = append(devices, d)
	}
	return devices
}

// GetLightName returns the user-assigned name for a light.
// The name comes from the owning device, not the light's deprecated Metadata.Name.
// Caller must hold the lock or call this on data that won't change.
func (s *BridgeState) GetLightName(light hueclient.LightGet) string {
	// Get name from owning device (this is where user-assigned names are stored)
	if light.Owner != nil && light.Owner.Rid != nil {
		if device, ok := s.Devices[*light.Owner.Rid]; ok {
			if device.Metadata != nil && device.Metadata.Name != nil {
				return *device.Metadata.Name
			}
		}
	}
	// Fallback to light's own metadata (deprecated, but better than nothing)
	if light.Metadata != nil && light.Metadata.Name != nil {
		return *light.Metadata.Name
	}
	return "Unknown"
}

// GetDeviceName returns the name of a device, or "Unknown" if not available.
func (s *BridgeState) GetDeviceName(device hueclient.DeviceGet) string {
	if device.Metadata != nil && device.Metadata.Name != nil {
		return *device.Metadata.Name
	}
	return "Unknown"
}

// GetRoomName returns the name of a room, or "Unknown" if not available.
func (s *BridgeState) GetRoomName(room hueclient.RoomGet) string {
	if room.Metadata != nil && room.Metadata.Name != nil {
		return *room.Metadata.Name
	}
	return "Unknown"
}

// GetSceneName returns the name of a scene, or "Unknown" if not available.
func (s *BridgeState) GetSceneName(scene hueclient.SceneGet) string {
	if scene.Metadata != nil && scene.Metadata.Name != nil {
		return *scene.Metadata.Name
	}
	return "Unknown"
}

// RoomLights returns the lights belonging to a room by resolving device references, sorted by name.
func (s *BridgeState) RoomLights(room hueclient.RoomGet) []hueclient.LightGet {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Build device -> lights mapping
	deviceLights := make(map[string][]hueclient.LightGet)
	for _, light := range s.Lights {
		if light.Owner != nil && light.Owner.Rid != nil {
			deviceLights[*light.Owner.Rid] = append(deviceLights[*light.Owner.Rid], light)
		}
	}

	// Room children are devices
	var lights []hueclient.LightGet
	if room.Children != nil {
		for _, child := range *room.Children {
			if child.Rid != nil {
				if deviceLightList, ok := deviceLights[*child.Rid]; ok {
					lights = append(lights, deviceLightList...)
				}
			}
		}
	}

	// Sort by name for stable ordering (using device name, not deprecated light metadata)
	sort.Slice(lights, func(i, j int) bool {
		return s.GetLightName(lights[i]) < s.GetLightName(lights[j])
	})

	return lights
}

// RoomGroupedLight returns the grouped light for a room.
func (s *BridgeState) RoomGroupedLight(room hueclient.RoomGet) (hueclient.GroupedLightGet, bool) {
	if room.Services == nil {
		return hueclient.GroupedLightGet{}, false
	}
	for _, svc := range *room.Services {
		if svc.Rtype != nil && *svc.Rtype == hueclient.ResourceIdentifierRtypeGroupedLight && svc.Rid != nil {
			return s.GetGroupedLight(*svc.Rid)
		}
	}
	return hueclient.GroupedLightGet{}, false
}

// RoomScenes returns scenes belonging to a room, sorted by name.
func (s *BridgeState) RoomScenes(roomID string) []hueclient.SceneGet {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var scenes []hueclient.SceneGet
	for _, scene := range s.Scenes {
		if scene.Group != nil && scene.Group.Rid != nil && *scene.Group.Rid == roomID {
			scenes = append(scenes, scene)
		}
	}

	// Sort by name for stable ordering
	sort.Slice(scenes, func(i, j int) bool {
		nameI := ""
		nameJ := ""
		if scenes[i].Metadata != nil && scenes[i].Metadata.Name != nil {
			nameI = *scenes[i].Metadata.Name
		}
		if scenes[j].Metadata != nil && scenes[j].Metadata.Name != nil {
			nameJ = *scenes[j].Metadata.Name
		}
		return nameI < nameJ
	})

	return scenes
}

// ZoneScenes returns scenes belonging to a zone, sorted by name.
func (s *BridgeState) ZoneScenes(zoneID string) []hueclient.SceneGet {
	// Zones use the same scene grouping as rooms
	return s.RoomScenes(zoneID)
}

// ZoneGroupedLight returns the grouped light for a zone.
func (s *BridgeState) ZoneGroupedLight(zone hueclient.RoomGet) (hueclient.GroupedLightGet, bool) {
	// Zones use the same services structure as rooms
	return s.RoomGroupedLight(zone)
}

// IsZoneOn returns true if any light in the zone is on.
func (s *BridgeState) IsZoneOn(zone hueclient.RoomGet) bool {
	// Zones work the same way as rooms
	return s.IsRoomOn(zone)
}

// ZoneBrightness returns the grouped light brightness for a zone.
func (s *BridgeState) ZoneBrightness(zone hueclient.RoomGet) float64 {
	// Zones work the same way as rooms
	return s.RoomBrightness(zone)
}

// AllRooms returns all rooms sorted by name.
func (s *BridgeState) AllRooms() []hueclient.RoomGet {
	s.mu.RLock()
	defer s.mu.RUnlock()

	rooms := make([]hueclient.RoomGet, 0, len(s.Rooms))
	for _, r := range s.Rooms {
		rooms = append(rooms, r)
	}

	// Sort by name for stable ordering
	sort.Slice(rooms, func(i, j int) bool {
		nameI := ""
		nameJ := ""
		if rooms[i].Metadata != nil && rooms[i].Metadata.Name != nil {
			nameI = *rooms[i].Metadata.Name
		}
		if rooms[j].Metadata != nil && rooms[j].Metadata.Name != nil {
			nameJ = *rooms[j].Metadata.Name
		}
		return nameI < nameJ
	})

	return rooms
}

// AllZones returns all zones sorted by name.
func (s *BridgeState) AllZones() []hueclient.RoomGet {
	s.mu.RLock()
	defer s.mu.RUnlock()

	zones := make([]hueclient.RoomGet, 0, len(s.Zones))
	for _, z := range s.Zones {
		zones = append(zones, z)
	}

	// Sort by name for stable ordering
	sort.Slice(zones, func(i, j int) bool {
		nameI := ""
		nameJ := ""
		if zones[i].Metadata != nil && zones[i].Metadata.Name != nil {
			nameI = *zones[i].Metadata.Name
		}
		if zones[j].Metadata != nil && zones[j].Metadata.Name != nil {
			nameJ = *zones[j].Metadata.Name
		}
		return nameI < nameJ
	})

	return zones
}

// AllLights returns all lights sorted by name.
func (s *BridgeState) AllLights() []hueclient.LightGet {
	s.mu.RLock()
	defer s.mu.RUnlock()

	lights := make([]hueclient.LightGet, 0, len(s.Lights))
	for _, l := range s.Lights {
		lights = append(lights, l)
	}

	// Sort by name for stable ordering (using device name, not deprecated light metadata)
	sort.Slice(lights, func(i, j int) bool {
		return s.GetLightName(lights[i]) < s.GetLightName(lights[j])
	})

	return lights
}

// AllScenes returns all scenes sorted by name.
func (s *BridgeState) AllScenes() []hueclient.SceneGet {
	s.mu.RLock()
	defer s.mu.RUnlock()

	scenes := make([]hueclient.SceneGet, 0, len(s.Scenes))
	for _, sc := range s.Scenes {
		scenes = append(scenes, sc)
	}

	// Helper to get group name (room or zone name) for a scene
	getGroupName := func(sc hueclient.SceneGet) string {
		if sc.Group == nil || sc.Group.Rid == nil {
			return ""
		}
		rid := *sc.Group.Rid
		// Check rooms first
		if room, ok := s.Rooms[rid]; ok {
			if room.Metadata != nil && room.Metadata.Name != nil {
				return *room.Metadata.Name
			}
		}
		// Check zones
		if zone, ok := s.Zones[rid]; ok {
			if zone.Metadata != nil && zone.Metadata.Name != nil {
				return *zone.Metadata.Name
			}
		}
		return ""
	}

	// Sort by scene name first, then by group name (room/zone), then by ID for stable ordering
	sort.Slice(scenes, func(i, j int) bool {
		nameI := ""
		nameJ := ""
		if scenes[i].Metadata != nil && scenes[i].Metadata.Name != nil {
			nameI = *scenes[i].Metadata.Name
		}
		if scenes[j].Metadata != nil && scenes[j].Metadata.Name != nil {
			nameJ = *scenes[j].Metadata.Name
		}
		if nameI != nameJ {
			return nameI < nameJ
		}
		groupI := getGroupName(scenes[i])
		groupJ := getGroupName(scenes[j])
		if groupI != groupJ {
			return groupI < groupJ
		}
		// Tertiary sort by ID for full stability
		idI := ""
		idJ := ""
		if scenes[i].Id != nil {
			idI = *scenes[i].Id
		}
		if scenes[j].Id != nil {
			idJ = *scenes[j].Id
		}
		return idI < idJ
	})

	return scenes
}

// UpdateLights replaces the lights cache.
func (s *BridgeState) UpdateLights(lights map[string]hueclient.LightGet) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Lights = lights
}

// UpdateRooms replaces the rooms cache.
func (s *BridgeState) UpdateRooms(rooms map[string]hueclient.RoomGet) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Rooms = rooms
}

// ApplyRoomMetadata applies a metadata update (e.g., rename) to a room.
func (s *BridgeState) ApplyRoomMetadata(id string, name *string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	room, ok := s.Rooms[id]
	if !ok {
		return false
	}

	if name != nil && room.Metadata != nil {
		room.Metadata.Name = name
		s.Rooms[id] = room
		return true
	}
	return false
}

// UpdateZones replaces the zones cache.
func (s *BridgeState) UpdateZones(zones map[string]hueclient.RoomGet) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Zones = zones
}

// ApplyZoneMetadata applies a metadata update (e.g., rename) to a zone.
func (s *BridgeState) ApplyZoneMetadata(id string, name *string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	zone, ok := s.Zones[id]
	if !ok {
		return false
	}

	if name != nil && zone.Metadata != nil {
		zone.Metadata.Name = name
		s.Zones[id] = zone
		return true
	}
	return false
}

// UpdateGroupedLights replaces the grouped lights cache.
func (s *BridgeState) UpdateGroupedLights(grouped map[string]hueclient.GroupedLightGet) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.GroupedLights = grouped
}

// UpdateScenes replaces the scenes cache.
func (s *BridgeState) UpdateScenes(scenes map[string]hueclient.SceneGet) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Scenes = scenes
}

// ApplySceneMetadata applies a metadata update (e.g., rename) to a scene.
func (s *BridgeState) ApplySceneMetadata(id string, name *string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	scene, ok := s.Scenes[id]
	if !ok {
		return false
	}

	if name != nil && scene.Metadata != nil {
		scene.Metadata.Name = name
		s.Scenes[id] = scene
		return true
	}
	return false
}

// UpdateDevices replaces the devices cache.
func (s *BridgeState) UpdateDevices(devices map[string]hueclient.DeviceGet) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Devices = devices
}

// ApplyDeviceMetadata applies a metadata update (e.g., rename) to a device.
func (s *BridgeState) ApplyDeviceMetadata(id string, name *string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	device, ok := s.Devices[id]
	if !ok {
		return false
	}

	if name != nil && device.Metadata != nil {
		device.Metadata.Name = name
		s.Devices[id] = device
		return true
	}
	return false
}

// UpdateMotionSensors replaces the motion sensors cache.
func (s *BridgeState) UpdateMotionSensors(sensors map[string]hueclient.MotionGet) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.MotionSensors = sensors
}

// GetMotionSensor returns a motion sensor by ID.
func (s *BridgeState) GetMotionSensor(id string) (hueclient.MotionGet, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	m, ok := s.MotionSensors[id]
	return m, ok
}

// GetDeviceMotionState returns true if the device has a motion sensor service that detects motion.
func (s *BridgeState) GetDeviceMotionState(device hueclient.DeviceGet) (hasMotion bool, isDetecting bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	deviceID := ""
	if device.Id != nil {
		deviceID = *device.Id
	}

	// First, try to find via device services
	if device.Services != nil {
		for _, svc := range *device.Services {
			if svc.Rtype != nil && *svc.Rtype == hueclient.ResourceIdentifierRtypeMotion && svc.Rid != nil {
				// Found motion service, check its state
				if motion, ok := s.MotionSensors[*svc.Rid]; ok {
					if motion.Motion != nil {
						// Check motion_report first (newer API), then fallback to motion
						if motion.Motion.MotionReport != nil && motion.Motion.MotionReport.Motion != nil {
							return true, *motion.Motion.MotionReport.Motion
						}
						if motion.Motion.Motion != nil {
							return true, *motion.Motion.Motion
						}
					}
					return true, false
				}
				return true, false
			}
		}
	}

	// Fallback: check if any motion sensor is owned by this device
	if deviceID != "" {
		for _, motion := range s.MotionSensors {
			if motion.Owner != nil && motion.Owner.Rid != nil && *motion.Owner.Rid == deviceID {
				if motion.Motion != nil {
					if motion.Motion.MotionReport != nil && motion.Motion.MotionReport.Motion != nil {
						return true, *motion.Motion.MotionReport.Motion
					}
					if motion.Motion.Motion != nil {
						return true, *motion.Motion.Motion
					}
				}
				return true, false
			}
		}
	}

	return false, false
}

// GetDeviceMotionSensor returns the motion sensor resource for a device if it has one.
// Returns the motion sensor ID and the full MotionGet struct.
func (s *BridgeState) GetDeviceMotionSensor(device hueclient.DeviceGet) (motionID string, motion hueclient.MotionGet, found bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	deviceID := ""
	if device.Id != nil {
		deviceID = *device.Id
	}

	// First, try to find via device services
	if device.Services != nil {
		for _, svc := range *device.Services {
			if svc.Rtype != nil && *svc.Rtype == hueclient.ResourceIdentifierRtypeMotion && svc.Rid != nil {
				if m, ok := s.MotionSensors[*svc.Rid]; ok {
					return *svc.Rid, m, true
				}
				return *svc.Rid, hueclient.MotionGet{}, true
			}
		}
	}

	// Fallback: check if any motion sensor is owned by this device
	if deviceID != "" {
		for id, m := range s.MotionSensors {
			if m.Owner != nil && m.Owner.Rid != nil && *m.Owner.Rid == deviceID {
				return id, m, true
			}
		}
	}

	return "", hueclient.MotionGet{}, false
}

// GetDeviceTemperature returns the temperature reading for a device if it has a temperature service.
func (s *BridgeState) GetDeviceTemperature(device hueclient.DeviceGet) (hasTemp bool, tempC float32) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	deviceID := ""
	if device.Id != nil {
		deviceID = *device.Id
	}

	// First, try to find via device services
	if device.Services != nil {
		for _, svc := range *device.Services {
			if svc.Rtype != nil && *svc.Rtype == hueclient.ResourceIdentifierRtypeTemperature && svc.Rid != nil {
				if temp, ok := s.Temperatures[*svc.Rid]; ok {
					if temp.Temperature != nil {
						if temp.Temperature.TemperatureReport != nil && temp.Temperature.TemperatureReport.Temperature != nil {
							return true, *temp.Temperature.TemperatureReport.Temperature
						}
						if temp.Temperature.Temperature != nil {
							return true, *temp.Temperature.Temperature
						}
					}
					return true, 0
				}
				return true, 0
			}
		}
	}

	// Fallback: check if any temperature sensor is owned by this device
	if deviceID != "" {
		for _, temp := range s.Temperatures {
			if temp.Owner != nil && temp.Owner.Rid != nil && *temp.Owner.Rid == deviceID {
				if temp.Temperature != nil {
					if temp.Temperature.TemperatureReport != nil && temp.Temperature.TemperatureReport.Temperature != nil {
						return true, *temp.Temperature.TemperatureReport.Temperature
					}
					if temp.Temperature.Temperature != nil {
						return true, *temp.Temperature.Temperature
					}
				}
				return true, 0
			}
		}
	}

	return false, 0
}

// GetDeviceLightLevel returns the light level reading for a device if it has a light_level service.
func (s *BridgeState) GetDeviceLightLevel(device hueclient.DeviceGet) (hasLevel bool, level int) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	deviceID := ""
	if device.Id != nil {
		deviceID = *device.Id
	}

	// First, try to find via device services
	if device.Services != nil {
		for _, svc := range *device.Services {
			if svc.Rtype != nil && *svc.Rtype == hueclient.ResourceIdentifierRtypeLightLevel && svc.Rid != nil {
				if ll, ok := s.LightLevels[*svc.Rid]; ok {
					if ll.Light != nil {
						if ll.Light.LightLevelReport != nil && ll.Light.LightLevelReport.LightLevel != nil {
							return true, *ll.Light.LightLevelReport.LightLevel
						}
						if ll.Light.LightLevel != nil {
							return true, *ll.Light.LightLevel
						}
					}
					return true, 0
				}
				return true, 0
			}
		}
	}

	// Fallback: check if any light level sensor is owned by this device
	if deviceID != "" {
		for _, ll := range s.LightLevels {
			if ll.Owner != nil && ll.Owner.Rid != nil && *ll.Owner.Rid == deviceID {
				if ll.Light != nil {
					if ll.Light.LightLevelReport != nil && ll.Light.LightLevelReport.LightLevel != nil {
						return true, *ll.Light.LightLevelReport.LightLevel
					}
					if ll.Light.LightLevel != nil {
						return true, *ll.Light.LightLevel
					}
				}
				return true, 0
			}
		}
	}

	return false, 0
}

// GetDeviceBattery returns the battery status for a device if it has a device_power service.
func (s *BridgeState) GetDeviceBattery(device hueclient.DeviceGet) (hasBattery bool, level int, state string) {
	if device.Services == nil {
		return false, 0, ""
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, svc := range *device.Services {
		if svc.Rtype != nil && *svc.Rtype == hueclient.ResourceIdentifierRtypeDevicePower && svc.Rid != nil {
			if power, ok := s.DevicePowers[*svc.Rid]; ok {
				if power.PowerState != nil {
					lvl := 0
					st := ""
					if power.PowerState.BatteryLevel != nil {
						lvl = *power.PowerState.BatteryLevel
					}
					if power.PowerState.BatteryState != nil {
						st = string(*power.PowerState.BatteryState)
					}
					return true, lvl, st
				}
				return true, 0, ""
			}
			return true, 0, ""
		}
	}
	return false, 0, ""
}

// UpdateTemperatures replaces the temperatures cache.
func (s *BridgeState) UpdateTemperatures(temps map[string]hueclient.TemperatureGet) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Temperatures = temps
}

// UpdateLightLevels replaces the light levels cache.
func (s *BridgeState) UpdateLightLevels(levels map[string]hueclient.LightLevelGet) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.LightLevels = levels
}

// GetLightLevel returns a light level sensor reading by its service ID.
func (s *BridgeState) GetLightLevel(id string) (hueclient.LightLevelGet, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	ll, ok := s.LightLevels[id]
	return ll, ok
}

// GetMotion returns a motion sensor reading by its service ID.
func (s *BridgeState) GetMotion(id string) (hueclient.MotionGet, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	m, ok := s.MotionSensors[id]
	return m, ok
}

// UpdateDevicePowers replaces the device powers cache.
func (s *BridgeState) UpdateDevicePowers(powers map[string]hueclient.DevicePowerGet) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.DevicePowers = powers
}

// UpdateEntertainmentConfigurations replaces entertainment configuration data.
func (s *BridgeState) UpdateEntertainmentConfigurations(configs map[string]EntertainmentConfiguration) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.EntertainmentConfigurations = configs
}

// GetEntertainmentConfiguration returns an entertainment configuration by ID.
func (s *BridgeState) GetEntertainmentConfiguration(id string) (EntertainmentConfiguration, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	ent, ok := s.EntertainmentConfigurations[id]
	return ent, ok
}

// UpdateWifiConnectivity replaces WiFi connectivity data.
func (s *BridgeState) UpdateWifiConnectivity(wifi []WifiConnectivity) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.WifiConnectivity = wifi
}

// GetWifiConnectivity returns WiFi connectivity status (empty if not supported).
func (s *BridgeState) GetWifiConnectivity() []WifiConnectivity {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.WifiConnectivity
}

// UpdateZigbeeConnectivity replaces Zigbee connectivity data.
func (s *BridgeState) UpdateZigbeeConnectivity(zigbee map[string]ZigbeeConnectivity) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ZigbeeConnectivity = zigbee
}

// GetZigbeeConnectivity returns a Zigbee connectivity resource by ID.
func (s *BridgeState) GetZigbeeConnectivity(id string) (ZigbeeConnectivity, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	zc, ok := s.ZigbeeConnectivity[id]
	return zc, ok
}

// GetDeviceZigbeeConnectivity returns the Zigbee connectivity for a device.
func (s *BridgeState) GetDeviceZigbeeConnectivity(device hueclient.DeviceGet) (ZigbeeConnectivity, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	deviceID := ""
	if device.Id != nil {
		deviceID = *device.Id
	}

	// First, try to find via device services
	if device.Services != nil {
		for _, svc := range *device.Services {
			if svc.Rtype != nil && string(*svc.Rtype) == "zigbee_connectivity" && svc.Rid != nil {
				if zc, ok := s.ZigbeeConnectivity[*svc.Rid]; ok {
					return zc, true
				}
			}
		}
	}

	// Fallback: check if any zigbee connectivity is owned by this device
	if deviceID != "" {
		for _, zc := range s.ZigbeeConnectivity {
			if zc.Owner != nil && zc.Owner.Rid == deviceID {
				return zc, true
			}
		}
	}

	return ZigbeeConnectivity{}, false
}

// UpdateBridgeResource sets the bridge resource.
func (s *BridgeState) UpdateBridgeResource(bridge *hueclient.BridgeGet) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.BridgeResource = bridge
}

// UpdateBridgeHome sets the bridge home resource.
func (s *BridgeState) UpdateBridgeHome(home *hueclient.BridgeHomeGet) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.BridgeHome = home
}

// GetBridgeResource returns the bridge resource.
func (s *BridgeState) GetBridgeResource() *hueclient.BridgeGet {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.BridgeResource
}

// GetBridgeHome returns the bridge home resource.
func (s *BridgeState) GetBridgeHome() *hueclient.BridgeHomeGet {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.BridgeHome
}

// GetBridgeDevice returns the device that owns the bridge service.
func (s *BridgeState) GetBridgeDevice() (hueclient.DeviceGet, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, device := range s.Devices {
		if device.Services == nil {
			continue
		}
		for _, svc := range *device.Services {
			if svc.Rtype != nil && *svc.Rtype == hueclient.ResourceIdentifierRtypeBridge {
				return device, true
			}
		}
	}
	return hueclient.DeviceGet{}, false
}

// UpdateAuthApps sets the authenticated applications list.
func (s *BridgeState) UpdateAuthApps(apps []AuthV1Entry) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.AuthApps = apps
}

// GetAuthApps returns the authenticated applications list.
func (s *BridgeState) GetAuthApps() []AuthV1Entry {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.AuthApps
}

// AllDevices returns all devices sorted by name.
func (s *BridgeState) AllDevices() []hueclient.DeviceGet {
	s.mu.RLock()
	defer s.mu.RUnlock()

	devices := make([]hueclient.DeviceGet, 0, len(s.Devices))
	for _, d := range s.Devices {
		devices = append(devices, d)
	}

	sort.Slice(devices, func(i, j int) bool {
		nameI := ""
		nameJ := ""
		if devices[i].Metadata != nil && devices[i].Metadata.Name != nil {
			nameI = *devices[i].Metadata.Name
		}
		if devices[j].Metadata != nil && devices[j].Metadata.Name != nil {
			nameJ = *devices[j].Metadata.Name
		}
		return nameI < nameJ
	})

	return devices
}

// AllEntertainmentConfigurations returns all entertainment configurations, sorted by name.
func (s *BridgeState) AllEntertainmentConfigurations() []EntertainmentConfiguration {
	s.mu.RLock()
	defer s.mu.RUnlock()

	configs := make([]EntertainmentConfiguration, 0, len(s.EntertainmentConfigurations))
	for _, c := range s.EntertainmentConfigurations {
		configs = append(configs, c)
	}

	sort.Slice(configs, func(i, j int) bool {
		nameI := ""
		nameJ := ""
		if configs[i].Metadata != nil {
			nameI = configs[i].Metadata.Name
		}
		if configs[j].Metadata != nil {
			nameJ = configs[j].Metadata.Name
		}
		return nameI < nameJ
	})

	return configs
}

// IsRoomOn returns true if any light in the room is on.
func (s *BridgeState) IsRoomOn(room hueclient.RoomGet) bool {
	if gl, ok := s.RoomGroupedLight(room); ok {
		return isGroupedLightOn(gl)
	}
	for _, light := range s.RoomLights(room) {
		if isLightOn(light) {
			return true
		}
	}
	return false
}

// BridgeName returns the name of the bridge device, if found.
func (s *BridgeState) BridgeName() string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, device := range s.Devices {
		if device.Services == nil {
			continue
		}
		for _, svc := range *device.Services {
			if svc.Rtype != nil && *svc.Rtype == hueclient.ResourceIdentifierRtypeBridge {
				if device.Metadata != nil && device.Metadata.Name != nil {
					return *device.Metadata.Name
				}
			}
		}
	}
	return ""
}

// RoomBrightness returns the grouped light brightness for a room.
func (s *BridgeState) RoomBrightness(room hueclient.RoomGet) float64 {
	if gl, ok := s.RoomGroupedLight(room); ok {
		if gl.Dimming != nil && gl.Dimming.Brightness != nil {
			return float64(*gl.Dimming.Brightness)
		}
	}
	return 0
}

// SetLight updates a single light in the cache.
func (s *BridgeState) SetLight(id string, light hueclient.LightGet) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Lights[id] = light
}

// SetGroupedLight updates a single grouped light in the cache.
func (s *BridgeState) SetGroupedLight(id string, gl hueclient.GroupedLightGet) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.GroupedLights[id] = gl
}

// SetLightBrightness optimistically updates a light's brightness in the cache.
func (s *BridgeState) SetLightBrightness(id string, brightness float64) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if light, ok := s.Lights[id]; ok {
		if light.Dimming != nil {
			br := hueclient.Brightness(brightness)
			light.Dimming.Brightness = &br
			s.Lights[id] = light
		}
	}
}

// SetLightColor optimistically updates a light's color in the cache.
func (s *BridgeState) SetLightColor(id string, x, y float64) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if light, ok := s.Lights[id]; ok {
		if light.Color != nil && light.Color.Xy != nil {
			xf, yf := float32(x), float32(y)
			light.Color.Xy.X = &xf
			light.Color.Xy.Y = &yf
			s.Lights[id] = light
		}
	}
}

// SetLightColorTemperature optimistically updates a light's color temperature in the cache.
func (s *BridgeState) SetLightColorTemperature(id string, mirek int) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if light, ok := s.Lights[id]; ok {
		if light.ColorTemperature != nil {
			light.ColorTemperature.Mirek = &mirek
			s.Lights[id] = light
		}
	}
}

// SetLightEffect optimistically updates a light's effect in the cache.
func (s *BridgeState) SetLightEffect(id string, effect hueclient.SupportedEffects) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if light, ok := s.Lights[id]; ok {
		if light.Effects != nil {
			light.Effects.Effect = &effect
			light.Effects.Status = &effect
			s.Lights[id] = light
		}
	}
}

// SetGroupedLightBrightness optimistically updates a grouped light's brightness in the cache.
func (s *BridgeState) SetGroupedLightBrightness(id string, brightness float64) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if gl, ok := s.GroupedLights[id]; ok {
		if gl.Dimming != nil {
			br := hueclient.Brightness(brightness)
			gl.Dimming.Brightness = &br
			s.GroupedLights[id] = gl
		}
	}
}

// SetLightOn optimistically updates a light's on state in the cache.
func (s *BridgeState) SetLightOn(id string, on bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if light, ok := s.Lights[id]; ok {
		if light.On == nil {
			light.On = &hueclient.On{}
		}
		light.On.On = &on
		s.Lights[id] = light
	}
}

// SetGroupedLightOn optimistically updates a grouped light's on state in the cache.
func (s *BridgeState) SetGroupedLightOn(id string, on bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if gl, ok := s.GroupedLights[id]; ok {
		if gl.On == nil {
			gl.On = &hueclient.On{}
		}
		gl.On.On = &on
		s.GroupedLights[id] = gl
	}
}

// SetMotionSensorEnabled optimistically updates a motion sensor's enabled state in the cache.
func (s *BridgeState) SetMotionSensorEnabled(id string, enabled bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if motion, ok := s.MotionSensors[id]; ok {
		motion.Enabled = &enabled
		s.MotionSensors[id] = motion
	}
}

// SetMotionSensorSensitivity optimistically updates a motion sensor's sensitivity in the cache.
func (s *BridgeState) SetMotionSensorSensitivity(id string, sensitivity int) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if motion, ok := s.MotionSensors[id]; ok {
		if motion.Sensitivity == nil {
			motion.Sensitivity = &struct {
				Sensitivity    *int                                  `json:"sensitivity,omitempty"`
				SensitivityMax *int                                  `json:"sensitivity_max,omitempty"`
				Status         *hueclient.MotionGetSensitivityStatus `json:"status,omitempty"`
			}{}
		}
		motion.Sensitivity.Sensitivity = &sensitivity
		s.MotionSensors[id] = motion
	}
}

// LightUpdate contains partial update data for a light from SSE events.
type LightUpdate struct {
	On         *bool
	Brightness *float64
	ColorXY    *[2]float64 // X, Y coordinates
	Mirek      *int        // Color temperature in mirek
}

// ApplyLightUpdate applies a partial update from an SSE event to a light.
// Returns true if the light was found and updated.
func (s *BridgeState) ApplyLightUpdate(id string, update LightUpdate) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	light, ok := s.Lights[id]
	if !ok {
		return false
	}

	if update.On != nil {
		if light.On == nil {
			light.On = &hueclient.On{}
		}
		light.On.On = update.On
	}

	if update.Brightness != nil && light.Dimming != nil {
		b := float32(*update.Brightness)
		light.Dimming.Brightness = &b
	}

	if update.ColorXY != nil && light.Color != nil && light.Color.Xy != nil {
		x := float32(update.ColorXY[0])
		y := float32(update.ColorXY[1])
		light.Color.Xy.X = &x
		light.Color.Xy.Y = &y
		// Clear mirek when switching to XY color mode
		if light.ColorTemperature != nil {
			light.ColorTemperature.Mirek = nil
		}
	}

	if update.Mirek != nil && light.ColorTemperature != nil {
		light.ColorTemperature.Mirek = update.Mirek
		// Clear XY when switching to color temperature mode
		if light.Color != nil && light.Color.Xy != nil {
			light.Color.Xy.X = nil
			light.Color.Xy.Y = nil
		}
	}

	s.Lights[id] = light
	return true
}

// ApplyGroupedLightUpdate applies a partial update from an SSE event to a grouped light.
func (s *BridgeState) ApplyGroupedLightUpdate(id string, on *bool, brightness *float64) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	gl, ok := s.GroupedLights[id]
	if !ok {
		return false
	}

	if on != nil {
		if gl.On == nil {
			gl.On = &hueclient.On{}
		}
		gl.On.On = on
	}

	if brightness != nil && gl.Dimming != nil {
		b := hueclient.Brightness(*brightness)
		gl.Dimming.Brightness = &b
	}

	s.GroupedLights[id] = gl
	return true
}

// ApplyMotionUpdate applies a partial update from an SSE event to a motion sensor.
func (s *BridgeState) ApplyMotionUpdate(id string, motion bool) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	m, ok := s.MotionSensors[id]
	if !ok {
		return false
	}

	// Only update if the Motion struct exists
	if m.Motion != nil {
		m.Motion.Motion = &motion
		// Also update MotionReport if it exists (keeps both in sync)
		if m.Motion.MotionReport != nil {
			m.Motion.MotionReport.Motion = &motion
		}
	}

	s.MotionSensors[id] = m
	return true
}

// ApplySceneStatus applies a status update from an SSE event to a scene.
func (s *BridgeState) ApplySceneStatus(id string, status string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	scene, ok := s.Scenes[id]
	if !ok {
		return false
	}

	// Only update if the Status struct exists
	if scene.Status != nil {
		active := hueclient.SceneGetStatusActive(status)
		scene.Status.Active = &active
	}

	s.Scenes[id] = scene
	return true
}

// GetTemperature returns a temperature sensor by ID.
func (s *BridgeState) GetTemperature(id string) (hueclient.TemperatureGet, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	t, ok := s.Temperatures[id]
	return t, ok
}

// GetGroupedLightName returns the room or zone name for a grouped light.
func (s *BridgeState) GetGroupedLightName(groupedLightID string) string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Check rooms
	for _, room := range s.Rooms {
		if room.Services != nil {
			for _, svc := range *room.Services {
				if svc.Rtype != nil && *svc.Rtype == hueclient.ResourceIdentifierRtypeGroupedLight &&
					svc.Rid != nil && *svc.Rid == groupedLightID {
					if room.Metadata != nil && room.Metadata.Name != nil {
						return *room.Metadata.Name
					}
				}
			}
		}
	}

	// Check zones
	for _, zone := range s.Zones {
		if zone.Services != nil {
			for _, svc := range *zone.Services {
				if svc.Rtype != nil && *svc.Rtype == hueclient.ResourceIdentifierRtypeGroupedLight &&
					svc.Rid != nil && *svc.Rid == groupedLightID {
					if zone.Metadata != nil && zone.Metadata.Name != nil {
						return *zone.Metadata.Name
					}
				}
			}
		}
	}

	return ""
}
