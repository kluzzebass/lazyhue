package hue

import (
	"sort"
	"strings"
	"sync"

	"github.com/kluzzebass/lazyhue/internal/hueclient"
)

// BridgeState holds cached entities from a bridge using hueclient types.
// Relationships are stored as IDs and resolved on-demand.
type BridgeState struct {
	mu sync.RWMutex

	Lights                      map[string]hueclient.LightGet
	Rooms                       map[string]hueclient.RoomGet
	Zones                       map[string]hueclient.ZoneGet
	Scenes                      map[string]hueclient.SceneGet
	SmartScenes                 map[string]hueclient.SmartSceneGet
	GroupedLights               map[string]hueclient.GroupedLightGet
	Devices                     map[string]hueclient.DeviceGet
	MotionSensors               map[string]hueclient.MotionGet
	Temperatures                map[string]hueclient.TemperatureGet
	LightLevels                 map[string]hueclient.LightLevelGet
	DevicePowers                map[string]hueclient.DevicePowerGet
	DeviceSoftwareUpdates       map[string]hueclient.DeviceSoftwareUpdateGet
	Buttons                     map[string]hueclient.ButtonGet // Button resources for switches/remotes
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
		Zones:                       make(map[string]hueclient.ZoneGet),
		Scenes:                      make(map[string]hueclient.SceneGet),
		SmartScenes:                 make(map[string]hueclient.SmartSceneGet),
		GroupedLights:               make(map[string]hueclient.GroupedLightGet),
		Devices:                     make(map[string]hueclient.DeviceGet),
		MotionSensors:               make(map[string]hueclient.MotionGet),
		Temperatures:                make(map[string]hueclient.TemperatureGet),
		LightLevels:                 make(map[string]hueclient.LightLevelGet),
		DevicePowers:                make(map[string]hueclient.DevicePowerGet),
		DeviceSoftwareUpdates:       make(map[string]hueclient.DeviceSoftwareUpdateGet),
		Buttons:                     make(map[string]hueclient.ButtonGet),
		EntertainmentConfigurations: make(map[string]EntertainmentConfiguration),
		ZigbeeConnectivity:          make(map[string]ZigbeeConnectivity),
	}
}

// GetLight returns a light by ID.
func (s *BridgeState) GetLight(id string) (hueclient.LightGet, bool) {
	return getFromMap(s, s.Lights, id)
}

// GetRoom returns a room by ID.
func (s *BridgeState) GetRoom(id string) (hueclient.RoomGet, bool) {
	return getFromMap(s, s.Rooms, id)
}

// GetZone returns a zone by ID.
func (s *BridgeState) GetZone(id string) (hueclient.ZoneGet, bool) {
	return getFromMap(s, s.Zones, id)
}

// GetGroupedLight returns a grouped light by ID.
func (s *BridgeState) GetGroupedLight(id string) (hueclient.GroupedLightGet, bool) {
	return getFromMap(s, s.GroupedLights, id)
}

// GetScene returns a scene by ID.
func (s *BridgeState) GetScene(id string) (hueclient.SceneGet, bool) {
	return getFromMap(s, s.Scenes, id)
}

// GetDevice returns a device by ID.
func (s *BridgeState) GetDevice(id string) (hueclient.DeviceGet, bool) {
	return getFromMap(s, s.Devices, id)
}

// DeleteScene removes a scene from the state.
func (s *BridgeState) DeleteScene(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.Scenes, id)
}

// GetSmartScene returns a smart scene by ID.
func (s *BridgeState) GetSmartScene(id string) (hueclient.SmartSceneGet, bool) {
	return getFromMap(s, s.SmartScenes, id)
}

// GetSmartSceneName returns the name of a smart scene.
func (s *BridgeState) GetSmartSceneName(scene hueclient.SmartSceneGet) string {
	if scene.Metadata.Name != "" {
		return scene.Metadata.Name
	}
	return "Unknown"
}

// UpdateSmartScenes replaces the smart scenes cache.
func (s *BridgeState) UpdateSmartScenes(scenes map[string]hueclient.SmartSceneGet) {
	updateMap(s, &s.SmartScenes, scenes)
}

// DeleteSmartScene removes a smart scene from the state.
func (s *BridgeState) DeleteSmartScene(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.SmartScenes, id)
}

// RemoveSmartScene removes a smart scene and returns whether it existed.
func (s *BridgeState) RemoveSmartScene(id string) bool {
	return removeFromMap(s, s.SmartScenes, id)
}

// AddSmartScene adds a smart scene to the state.
func (s *BridgeState) AddSmartScene(id string, scene hueclient.SmartSceneGet) {
	addToMap(s, &s.SmartScenes, id, scene)
}

// AllSmartScenes returns all smart scenes, sorted by name.
func (s *BridgeState) AllSmartScenes() []hueclient.SmartSceneGet {
	s.mu.RLock()
	defer s.mu.RUnlock()

	scenes := make([]hueclient.SmartSceneGet, 0, len(s.SmartScenes))
	for _, sc := range s.SmartScenes {
		scenes = append(scenes, sc)
	}
	sort.Slice(scenes, func(i, j int) bool {
		return strings.ToLower(scenes[i].Metadata.Name) < strings.ToLower(scenes[j].Metadata.Name)
	})
	return scenes
}

// DeleteDevice removes a device and its associated resources from the state.
func (s *BridgeState) DeleteDevice(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	device, ok := s.Devices[id]
	if !ok {
		return
	}

	// Remove associated services (lights, sensors, etc.)
	for _, svc := range device.Services {
		if svc.Rid == "" {
			continue
		}
		switch svc.Rtype {
		case hueclient.ResourceTypeLight:
			delete(s.Lights, svc.Rid)
		case hueclient.ResourceTypeMotion:
			delete(s.MotionSensors, svc.Rid)
		case hueclient.ResourceTypeTemperature:
			delete(s.Temperatures, svc.Rid)
		case hueclient.ResourceTypeLightLevel:
			delete(s.LightLevels, svc.Rid)
		case hueclient.ResourceTypeDevicePower:
			delete(s.DevicePowers, svc.Rid)
		}
	}

	delete(s.Devices, id)
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
// The name comes from the owning device, not the light's alternate Metadata.Name.
// Caller must hold the lock or call this on data that won't change.
func (s *BridgeState) GetLightName(light hueclient.LightGet) string {
	// Get name from owning device (this is where user-assigned names are stored)
	if light.Owner.Rid != "" {
		if device, ok := s.Devices[light.Owner.Rid]; ok {
			if device.Metadata.Name != "" {
				return device.Metadata.Name
			}
		}
	}
	// Fallback to light's own metadata (alternate name)
	if light.Metadata.Name != "" {
		return light.Metadata.Name
	}
	return "Unknown"
}

// GetDeviceName returns the name of a device, or "Unknown" if not available.
func (s *BridgeState) GetDeviceName(device hueclient.DeviceGet) string {
	if device.Metadata.Name != "" {
		return device.Metadata.Name
	}
	return "Unknown"
}

// GetDeviceAlternateName returns the alternate name from the device's lights, or empty string if not available.
// The alternate name is stored in the light's Metadata.Name field.
// If the device has multiple lights, returns the alternate name from the first light that has one.
func (s *BridgeState) GetDeviceAlternateName(device hueclient.DeviceGet) string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if device.Id == "" {
		return ""
	}

	// Find lights owned by this device
	for _, light := range s.Lights {
		if light.Owner.Rid == device.Id {
			// Return the first alternate name we find
			if light.Metadata.Name != "" {
				return light.Metadata.Name
			}
		}
	}
	return ""
}

// GetRoomName returns the name of a room, or "Unknown" if not available.
func (s *BridgeState) GetRoomName(room hueclient.RoomGet) string {
	if room.Metadata.Name != "" {
		return room.Metadata.Name
	}
	return "Unknown"
}

// GetSceneName returns the name of a scene, or "Unknown" if not available.
func (s *BridgeState) GetSceneName(scene hueclient.SceneGet) string {
	if scene.Metadata.Name != "" {
		return scene.Metadata.Name
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
		if light.Owner.Rid != "" {
			deviceLights[light.Owner.Rid] = append(deviceLights[light.Owner.Rid], light)
		}
	}

	// Room children are devices
	var lights []hueclient.LightGet
	for _, child := range room.Children {
		if child.Rid != "" {
			if deviceLightList, ok := deviceLights[child.Rid]; ok {
				lights = append(lights, deviceLightList...)
			}
		}
	}

	// Sort by name for stable ordering (using device name, not alternate light name)
	sort.Slice(lights, func(i, j int) bool {
		return s.GetLightName(lights[i]) < s.GetLightName(lights[j])
	})

	return lights
}

// ZoneLights returns all lights in a zone.
// Zones reference lights directly in their children (unlike rooms which reference devices).
func (s *BridgeState) ZoneLights(zone hueclient.ZoneGet) []hueclient.LightGet {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var lights []hueclient.LightGet
	for _, child := range zone.Children {
		if child.Rtype == hueclient.ResourceTypeLight {
			if light, ok := s.Lights[child.Rid]; ok {
				lights = append(lights, light)
			}
		}
	}

	// Sort by name for stable ordering
	sort.Slice(lights, func(i, j int) bool {
		return s.GetLightName(lights[i]) < s.GetLightName(lights[j])
	})

	return lights
}

// RoomGroupedLight returns the grouped light for a room.
func (s *BridgeState) RoomGroupedLight(room hueclient.RoomGet) (hueclient.GroupedLightGet, bool) {
	for _, svc := range room.Services {
		if svc.Rtype == hueclient.ResourceTypeGroupedLight && svc.Rid != "" {
			return s.GetGroupedLight(svc.Rid)
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
		if scene.Group.Rid == roomID {
			scenes = append(scenes, scene)
		}
	}

	// Sort by name for stable ordering
	sort.Slice(scenes, func(i, j int) bool {
		return scenes[i].Metadata.Name < scenes[j].Metadata.Name
	})

	return scenes
}

// ZoneScenes returns scenes belonging to a zone, sorted by name.
func (s *BridgeState) ZoneScenes(zoneID string) []hueclient.SceneGet {
	// Zones use the same scene grouping as rooms
	return s.RoomScenes(zoneID)
}

// RoomSmartScenes returns smart scenes belonging to a room, sorted by name.
func (s *BridgeState) RoomSmartScenes(roomID string) []hueclient.SmartSceneGet {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var scenes []hueclient.SmartSceneGet
	for _, scene := range s.SmartScenes {
		if scene.Group.Rid == roomID {
			scenes = append(scenes, scene)
		}
	}

	// Sort by name for stable ordering
	sort.Slice(scenes, func(i, j int) bool {
		return scenes[i].Metadata.Name < scenes[j].Metadata.Name
	})

	return scenes
}

// ZoneSmartScenes returns smart scenes belonging to a zone, sorted by name.
func (s *BridgeState) ZoneSmartScenes(zoneID string) []hueclient.SmartSceneGet {
	// Zones use the same scene grouping as rooms
	return s.RoomSmartScenes(zoneID)
}

// ZoneGroupedLight returns the grouped light for a zone.
func (s *BridgeState) ZoneGroupedLight(zone hueclient.ZoneGet) (hueclient.GroupedLightGet, bool) {
	// Zones have their own services structure
	for _, svc := range zone.Services {
		if svc.Rtype == hueclient.ResourceTypeGroupedLight && svc.Rid != "" {
			return s.GetGroupedLight(svc.Rid)
		}
	}
	return hueclient.GroupedLightGet{}, false
}

// IsZoneOn returns true if any light in the zone is on.
func (s *BridgeState) IsZoneOn(zone hueclient.ZoneGet) bool {
	lights := s.ZoneLights(zone)
	for _, light := range lights {
		if light.On.On {
			return true
		}
	}
	return false
}

// ZoneBrightness returns the grouped light brightness for a zone.
func (s *BridgeState) ZoneBrightness(zone hueclient.ZoneGet) float64 {
	gl, ok := s.ZoneGroupedLight(zone)
	if !ok {
		return 0
	}
	if gl.Dimming != nil {
		return float64(gl.Dimming.Brightness)
	}
	return 0
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
		return rooms[i].Metadata.Name < rooms[j].Metadata.Name
	})

	return rooms
}

// AllZones returns all zones sorted by name.
func (s *BridgeState) AllZones() []hueclient.ZoneGet {
	s.mu.RLock()
	defer s.mu.RUnlock()

	zones := make([]hueclient.ZoneGet, 0, len(s.Zones))
	for _, z := range s.Zones {
		zones = append(zones, z)
	}

	// Sort by name for stable ordering
	sort.Slice(zones, func(i, j int) bool {
		return zones[i].Metadata.Name < zones[j].Metadata.Name
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

	// Sort by name for stable ordering (using device name, not alternate light name)
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
		if sc.Group.Rid == "" {
			return ""
		}
		rid := sc.Group.Rid
		// Check rooms first
		if room, ok := s.Rooms[rid]; ok {
			if room.Metadata.Name != "" {
				return room.Metadata.Name
			}
		}
		// Check zones
		if zone, ok := s.Zones[rid]; ok {
			if zone.Metadata.Name != "" {
				return zone.Metadata.Name
			}
		}
		return ""
	}

	// Sort by scene name first, then by group name (room/zone), then by ID for stable ordering
	sort.Slice(scenes, func(i, j int) bool {
		nameI := scenes[i].Metadata.Name
		nameJ := scenes[j].Metadata.Name
		if nameI != nameJ {
			return nameI < nameJ
		}
		groupI := getGroupName(scenes[i])
		groupJ := getGroupName(scenes[j])
		if groupI != groupJ {
			return groupI < groupJ
		}
		// Tertiary sort by ID for full stability
		return scenes[i].Id < scenes[j].Id
	})

	return scenes
}

// UpdateLights replaces the lights cache.
func (s *BridgeState) UpdateLights(lights map[string]hueclient.LightGet) {
	updateMap(s, &s.Lights, lights)
}

// UpdateRooms replaces the rooms cache.
func (s *BridgeState) UpdateRooms(rooms map[string]hueclient.RoomGet) {
	updateMap(s, &s.Rooms, rooms)
}

// ApplyRoomMetadata applies a metadata update (e.g., rename, archetype change) to a room.
func (s *BridgeState) ApplyRoomMetadata(id string, name *string, archetype *string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	room, ok := s.Rooms[id]
	if !ok {
		return false
	}

	updated := false
	if name != nil {
		room.Metadata.Name = *name
		updated = true
	}
	if archetype != nil {
		room.Metadata.Archetype = hueclient.RoomArchetype(*archetype)
		updated = true
	}

	if updated {
		s.Rooms[id] = room
	}
	return updated
}

// ApplyRoomChildren applies a children update (devices) to a room.
func (s *BridgeState) ApplyRoomChildren(id string, children []struct {
	Rid   string `json:"rid"`
	Rtype string `json:"rtype"`
}) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	room, ok := s.Rooms[id]
	if !ok {
		return false
	}

	// Convert to ResourceIdentifier slice
	newChildren := make([]hueclient.ResourceIdentifier, len(children))
	for i, c := range children {
		newChildren[i] = hueclient.ResourceIdentifier{
			Rid:   c.Rid,
			Rtype: hueclient.ResourceType(c.Rtype),
		}
	}

	room.Children = newChildren
	s.Rooms[id] = room
	return true
}

// UpdateZones replaces the zones cache.
func (s *BridgeState) UpdateZones(zones map[string]hueclient.ZoneGet) {
	updateMap(s, &s.Zones, zones)
}

// ApplyZoneMetadata applies a metadata update (e.g., rename, archetype change) to a zone.
func (s *BridgeState) ApplyZoneMetadata(id string, name *string, archetype *string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	zone, ok := s.Zones[id]
	if !ok {
		return false
	}

	updated := false
	if name != nil {
		zone.Metadata.Name = *name
		updated = true
	}
	if archetype != nil {
		zone.Metadata.Archetype = hueclient.RoomArchetype(*archetype)
		updated = true
	}

	if updated {
		s.Zones[id] = zone
	}
	return updated
}

// ApplyZoneServices applies a services update (lights) to a zone.
func (s *BridgeState) ApplyZoneServices(id string, services []struct {
	Rid   string `json:"rid"`
	Rtype string `json:"rtype"`
}) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	zone, ok := s.Zones[id]
	if !ok {
		return false
	}

	// Convert to ResourceIdentifier slice
	newServices := make([]hueclient.ResourceIdentifier, len(services))
	for i, svc := range services {
		newServices[i] = hueclient.ResourceIdentifier{
			Rid:   svc.Rid,
			Rtype: hueclient.ResourceType(svc.Rtype),
		}
	}

	zone.Services = newServices
	s.Zones[id] = zone
	return true
}

// UpdateGroupedLights replaces the grouped lights cache.
func (s *BridgeState) UpdateGroupedLights(grouped map[string]hueclient.GroupedLightGet) {
	updateMap(s, &s.GroupedLights, grouped)
}

// UpdateScenes replaces the scenes cache.
func (s *BridgeState) UpdateScenes(scenes map[string]hueclient.SceneGet) {
	updateMap(s, &s.Scenes, scenes)
}

// ApplySceneMetadata applies a metadata update (e.g., rename) to a scene.
func (s *BridgeState) ApplySceneMetadata(id string, name *string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	scene, ok := s.Scenes[id]
	if !ok {
		return false
	}

	if name != nil {
		scene.Metadata.Name = *name
		s.Scenes[id] = scene
		return true
	}
	return false
}

// UpdateDevices replaces the devices cache.
func (s *BridgeState) UpdateDevices(devices map[string]hueclient.DeviceGet) {
	updateMap(s, &s.Devices, devices)
}

// ApplyDeviceMetadata applies a metadata update (e.g., rename) to a device.
func (s *BridgeState) ApplyDeviceMetadata(id string, name *string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	device, ok := s.Devices[id]
	if !ok {
		return false
	}

	if name != nil {
		device.Metadata.Name = *name
		s.Devices[id] = device
		return true
	}
	return false
}

// RemoveLight removes a light from the cache.
func (s *BridgeState) RemoveLight(id string) bool {
	return removeFromMap(s, s.Lights, id)
}

// RemoveRoom removes a room from the cache.
func (s *BridgeState) RemoveRoom(id string) bool {
	return removeFromMap(s, s.Rooms, id)
}

// RemoveZone removes a zone from the cache.
func (s *BridgeState) RemoveZone(id string) bool {
	return removeFromMap(s, s.Zones, id)
}

// RemoveScene removes a scene from the cache.
func (s *BridgeState) RemoveScene(id string) bool {
	return removeFromMap(s, s.Scenes, id)
}

// RemoveDevice removes a device from the cache.
func (s *BridgeState) RemoveDevice(id string) bool {
	return removeFromMap(s, s.Devices, id)
}

// RemoveGroupedLight removes a grouped light from the cache.
func (s *BridgeState) RemoveGroupedLight(id string) bool {
	return removeFromMap(s, s.GroupedLights, id)
}

// AddRoom adds a room to the cache.
func (s *BridgeState) AddRoom(id string, room hueclient.RoomGet) {
	addToMap(s, &s.Rooms, id, room)
}

// AddZone adds a zone to the cache.
func (s *BridgeState) AddZone(id string, zone hueclient.ZoneGet) {
	addToMap(s, &s.Zones, id, zone)
}

// AddScene adds a scene to the cache.
func (s *BridgeState) AddScene(id string, scene hueclient.SceneGet) {
	addToMap(s, &s.Scenes, id, scene)
}

// AddGroupedLight adds a grouped light to the cache.
func (s *BridgeState) AddGroupedLight(id string, gl hueclient.GroupedLightGet) {
	addToMap(s, &s.GroupedLights, id, gl)
}

// AddLight adds a light to the cache.
func (s *BridgeState) AddLight(id string, light hueclient.LightGet) {
	addToMap(s, &s.Lights, id, light)
}

// AddDevice adds a device to the cache.
func (s *BridgeState) AddDevice(id string, device hueclient.DeviceGet) {
	addToMap(s, &s.Devices, id, device)
}

// UpdateMotionSensors replaces the motion sensors cache.
func (s *BridgeState) UpdateMotionSensors(sensors map[string]hueclient.MotionGet) {
	updateMap(s, &s.MotionSensors, sensors)
}

// GetMotionSensor returns a motion sensor by ID.
func (s *BridgeState) GetMotionSensor(id string) (hueclient.MotionGet, bool) {
	return getFromMap(s, s.MotionSensors, id)
}

// GetDeviceMotionState returns true if the device has a motion sensor service that detects motion.
func (s *BridgeState) GetDeviceMotionState(device hueclient.DeviceGet) (hasMotion bool, isDetecting bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	deviceID := device.Id

	// First, try to find via device services
	for _, svc := range device.Services {
		if svc.Rtype == hueclient.ResourceTypeMotion && svc.Rid != "" {
			// Found motion service, check its state
			if motion, ok := s.MotionSensors[svc.Rid]; ok {
				// Check motion_report first (newer API), then fallback to motion
				if motion.Motion.MotionReport != nil {
					return true, motion.Motion.MotionReport.Motion
				}
				return true, motion.Motion.Motion
			}
			return true, false
		}
	}

	// Fallback: check if any motion sensor is owned by this device
	if deviceID != "" {
		for _, motion := range s.MotionSensors {
			if motion.Owner.Rid == deviceID {
				if motion.Motion.MotionReport != nil {
					return true, motion.Motion.MotionReport.Motion
				}
				return true, motion.Motion.Motion
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

	deviceID := device.Id

	// First, try to find via device services
	for _, svc := range device.Services {
		if svc.Rtype == hueclient.ResourceTypeMotion && svc.Rid != "" {
			if m, ok := s.MotionSensors[svc.Rid]; ok {
				return svc.Rid, m, true
			}
			return svc.Rid, hueclient.MotionGet{}, true
		}
	}

	// Fallback: check if any motion sensor is owned by this device
	if deviceID != "" {
		for id, m := range s.MotionSensors {
			if m.Owner.Rid == deviceID {
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

	deviceID := device.Id

	// First, try to find via device services
	for _, svc := range device.Services {
		if svc.Rtype == hueclient.ResourceTypeTemperature && svc.Rid != "" {
			if temp, ok := s.Temperatures[svc.Rid]; ok {
				if temp.Temperature.TemperatureReport != nil {
					return true, temp.Temperature.TemperatureReport.Temperature
				}
				return true, temp.Temperature.Temperature
			}
			return true, 0
		}
	}

	// Fallback: check if any temperature sensor is owned by this device
	if deviceID != "" {
		for _, temp := range s.Temperatures {
			if temp.Owner.Rid == deviceID {
				if temp.Temperature.TemperatureReport != nil {
					return true, temp.Temperature.TemperatureReport.Temperature
				}
				return true, temp.Temperature.Temperature
			}
		}
	}

	return false, 0
}

// GetDeviceLightLevel returns the light level reading for a device if it has a light_level service.
func (s *BridgeState) GetDeviceLightLevel(device hueclient.DeviceGet) (hasLevel bool, level int) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	deviceID := device.Id

	// First, try to find via device services
	for _, svc := range device.Services {
		if svc.Rtype == hueclient.ResourceTypeLightLevel && svc.Rid != "" {
			if ll, ok := s.LightLevels[svc.Rid]; ok {
				if ll.Light.LightLevelReport != nil {
					return true, ll.Light.LightLevelReport.LightLevel
				}
				return true, ll.Light.LightLevel
			}
			return true, 0
		}
	}

	// Fallback: check if any light level sensor is owned by this device
	if deviceID != "" {
		for _, ll := range s.LightLevels {
			if ll.Owner.Rid == deviceID {
				if ll.Light.LightLevelReport != nil {
					return true, ll.Light.LightLevelReport.LightLevel
				}
				return true, ll.Light.LightLevel
			}
		}
	}

	return false, 0
}

// GetDeviceTemperatureSensor returns the temperature sensor for a device if it has one.
// Returns the sensor ID and the full sensor object.
func (s *BridgeState) GetDeviceTemperatureSensor(device hueclient.DeviceGet) (sensorID string, sensor hueclient.TemperatureGet, found bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	deviceID := device.Id

	// First, try to find via device services
	for _, svc := range device.Services {
		if svc.Rtype == hueclient.ResourceTypeTemperature && svc.Rid != "" {
			if temp, ok := s.Temperatures[svc.Rid]; ok {
				return svc.Rid, temp, true
			}
		}
	}

	// Fallback: check if any temperature sensor is owned by this device
	if deviceID != "" {
		for id, temp := range s.Temperatures {
			if temp.Owner.Rid == deviceID {
				return id, temp, true
			}
		}
	}

	return "", hueclient.TemperatureGet{}, false
}

// GetDeviceLightLevelSensor returns the light level sensor for a device if it has one.
// Returns the sensor ID and the full sensor object.
func (s *BridgeState) GetDeviceLightLevelSensor(device hueclient.DeviceGet) (sensorID string, sensor hueclient.LightLevelGet, found bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	deviceID := device.Id

	// First, try to find via device services
	for _, svc := range device.Services {
		if svc.Rtype == hueclient.ResourceTypeLightLevel && svc.Rid != "" {
			if ll, ok := s.LightLevels[svc.Rid]; ok {
				return svc.Rid, ll, true
			}
		}
	}

	// Fallback: check if any light level sensor is owned by this device
	if deviceID != "" {
		for id, ll := range s.LightLevels {
			if ll.Owner.Rid == deviceID {
				return id, ll, true
			}
		}
	}

	return "", hueclient.LightLevelGet{}, false
}

// GetDeviceBattery returns the battery status for a device if it has a device_power service.
func (s *BridgeState) GetDeviceBattery(device hueclient.DeviceGet) (hasBattery bool, level int, state string) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, svc := range device.Services {
		if svc.Rtype == hueclient.ResourceTypeDevicePower && svc.Rid != "" {
			if power, ok := s.DevicePowers[svc.Rid]; ok {
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
	}
	return false, 0, ""
}

// UpdateTemperatures replaces the temperatures cache.
func (s *BridgeState) UpdateTemperatures(temps map[string]hueclient.TemperatureGet) {
	updateMap(s, &s.Temperatures, temps)
}

// UpdateLightLevels replaces the light levels cache.
func (s *BridgeState) UpdateLightLevels(levels map[string]hueclient.LightLevelGet) {
	updateMap(s, &s.LightLevels, levels)
}

// GetLightLevel returns a light level sensor reading by its service ID.
func (s *BridgeState) GetLightLevel(id string) (hueclient.LightLevelGet, bool) {
	return getFromMap(s, s.LightLevels, id)
}

// GetMotion returns a motion sensor reading by its service ID.
func (s *BridgeState) GetMotion(id string) (hueclient.MotionGet, bool) {
	return getFromMap(s, s.MotionSensors, id)
}

// UpdateDevicePowers replaces the device powers cache.
func (s *BridgeState) UpdateDevicePowers(powers map[string]hueclient.DevicePowerGet) {
	updateMap(s, &s.DevicePowers, powers)
}

// UpdateDeviceSoftwareUpdates replaces device software update data.
func (s *BridgeState) UpdateDeviceSoftwareUpdates(updates map[string]hueclient.DeviceSoftwareUpdateGet) {
	updateMap(s, &s.DeviceSoftwareUpdates, updates)
}

// GetDeviceSoftwareUpdate returns a device software update by ID.
func (s *BridgeState) GetDeviceSoftwareUpdate(id string) (hueclient.DeviceSoftwareUpdateGet, bool) {
	return getFromMap(s, s.DeviceSoftwareUpdates, id)
}

// GetDeviceSoftwareUpdateStatus returns the software update status for a device.
// Returns the update resource and whether an update is available.
func (s *BridgeState) GetDeviceSoftwareUpdateStatus(device hueclient.DeviceGet) (update hueclient.DeviceSoftwareUpdateGet, found bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, svc := range device.Services {
		if svc.Rtype == "device_software_update" {
			if upd, ok := s.DeviceSoftwareUpdates[svc.Rid]; ok {
				return upd, true
			}
		}
	}
	return hueclient.DeviceSoftwareUpdateGet{}, false
}

// UpdateButtons replaces the buttons cache.
func (s *BridgeState) UpdateButtons(buttons map[string]hueclient.ButtonGet) {
	updateMap(s, &s.Buttons, buttons)
}

// GetButton returns a button by ID.
func (s *BridgeState) GetButton(id string) (hueclient.ButtonGet, bool) {
	return getFromMap(s, s.Buttons, id)
}

// GetDeviceButtons returns all buttons owned by a device, sorted by control ID.
func (s *BridgeState) GetDeviceButtons(device hueclient.DeviceGet) []hueclient.ButtonGet {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var buttons []hueclient.ButtonGet
	deviceID := device.Id

	// Find buttons via device services
	for _, svc := range device.Services {
		if svc.Rtype == hueclient.ResourceTypeButton && svc.Rid != "" {
			if btn, ok := s.Buttons[svc.Rid]; ok {
				buttons = append(buttons, btn)
			}
		}
	}

	// Fallback: check if any button is owned by this device
	if len(buttons) == 0 && deviceID != "" {
		for _, btn := range s.Buttons {
			if btn.Owner.Rid == deviceID {
				buttons = append(buttons, btn)
			}
		}
	}

	// Sort by control ID for consistent ordering
	sort.Slice(buttons, func(i, j int) bool {
		return buttons[i].Metadata.ControlId < buttons[j].Metadata.ControlId
	})

	return buttons
}

// ApplyButtonUpdate applies a button event update from SSE.
// Returns true if the button was found and updated.
func (s *BridgeState) ApplyButtonUpdate(buttonID string, lastEvent *string, buttonReport *hueclient.BellButtonGetButtonButtonReport) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	btn, ok := s.Buttons[buttonID]
	if !ok {
		return false
	}

	if lastEvent != nil {
		evt := hueclient.Event(*lastEvent)
		btn.Button.LastEvent = &evt
	}
	if buttonReport != nil {
		btn.Button.ButtonReport = buttonReport
	}

	s.Buttons[buttonID] = btn
	return true
}

// GetButtonOwnerDeviceID returns the device ID that owns a button.
func (s *BridgeState) GetButtonOwnerDeviceID(buttonID string) string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if btn, ok := s.Buttons[buttonID]; ok {
		return btn.Owner.Rid
	}
	return ""
}

// UpdateEntertainmentConfigurations replaces entertainment configuration data.
func (s *BridgeState) UpdateEntertainmentConfigurations(configs map[string]EntertainmentConfiguration) {
	updateMap(s, &s.EntertainmentConfigurations, configs)
}

// GetEntertainmentConfiguration returns an entertainment configuration by ID.
func (s *BridgeState) GetEntertainmentConfiguration(id string) (EntertainmentConfiguration, bool) {
	return getFromMap(s, s.EntertainmentConfigurations, id)
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
	updateMap(s, &s.ZigbeeConnectivity, zigbee)
}

// GetZigbeeConnectivity returns a Zigbee connectivity resource by ID.
func (s *BridgeState) GetZigbeeConnectivity(id string) (ZigbeeConnectivity, bool) {
	return getFromMap(s, s.ZigbeeConnectivity, id)
}

// GetDeviceZigbeeConnectivity returns the Zigbee connectivity for a device.
func (s *BridgeState) GetDeviceZigbeeConnectivity(device hueclient.DeviceGet) (ZigbeeConnectivity, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	deviceID := device.Id

	// First, try to find via device services
	for _, svc := range device.Services {
		if svc.Rtype == hueclient.ResourceTypeZigbeeConnectivity && svc.Rid != "" {
			if zc, ok := s.ZigbeeConnectivity[svc.Rid]; ok {
				return zc, true
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
		for _, svc := range device.Services {
			if svc.Rtype == hueclient.ResourceTypeBridge {
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
		return devices[i].Metadata.Name < devices[j].Metadata.Name
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
		for _, svc := range device.Services {
			if svc.Rtype == hueclient.ResourceTypeBridge {
				if device.Metadata.Name != "" {
					return device.Metadata.Name
				}
			}
		}
	}
	return ""
}

// RoomBrightness returns the grouped light brightness for a room.
func (s *BridgeState) RoomBrightness(room hueclient.RoomGet) float64 {
	if gl, ok := s.RoomGroupedLight(room); ok {
		if gl.Dimming != nil {
			return float64(gl.Dimming.Brightness)
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
			light.Dimming.Brightness = float32(brightness)
			s.Lights[id] = light
		}
	}
}

// SetLightColor optimistically updates a light's color in the cache.
func (s *BridgeState) SetLightColor(id string, x, y float64) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if light, ok := s.Lights[id]; ok {
		if light.Color != nil {
			light.Color.Xy.X = float32(x)
			light.Color.Xy.Y = float32(y)
			if light.ColorTemperature != nil {
				light.ColorTemperature.MirekValid = false
			}
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
			light.ColorTemperature.Mirek = mirek
			light.ColorTemperature.MirekValid = true
			s.Lights[id] = light
		}
	}
}

// SetLightEffect optimistically updates a light's effect in the cache.
func (s *BridgeState) SetLightEffect(id string, effect hueclient.Effect) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if light, ok := s.Lights[id]; ok {
		if light.Effects != nil {
			light.Effects.Status = effect
			s.Lights[id] = light
		}
	}
}

// SetLightGradientMode optimistically updates a light's gradient mode in the cache.
func (s *BridgeState) SetLightGradientMode(id string, mode hueclient.LightGetGradientMode) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if light, ok := s.Lights[id]; ok {
		if light.Gradient != nil {
			light.Gradient.Mode = mode
			if light.ColorTemperature != nil {
				light.ColorTemperature.MirekValid = false
			}
			s.Lights[id] = light
		}
	}
}

// SetLightGradientPoints optimistically updates a light's gradient points in the cache.
func (s *BridgeState) SetLightGradientPoints(id string, points []hueclient.GradientPointGet) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if light, ok := s.Lights[id]; ok {
		if light.Gradient != nil {
			light.Gradient.Points = points
			if light.ColorTemperature != nil {
				light.ColorTemperature.MirekValid = false
			}
			s.Lights[id] = light
		}
	}
}

// SetLightPowerupPreset optimistically updates a light's power-on preset in the cache.
func (s *BridgeState) SetLightPowerupPreset(id string, preset hueclient.LightGetPowerupPreset) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if light, ok := s.Lights[id]; ok {
		if light.Powerup != nil {
			light.Powerup.Preset = preset
			s.Lights[id] = light
		}
	}
}

// SetDeviceArchetype optimistically updates a device's archetype in the cache.
func (s *BridgeState) SetDeviceArchetype(id string, archetype hueclient.DeviceArchetype) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if device, ok := s.Devices[id]; ok {
		// Update metadata archetype (user-changeable, what the API actually updates)
		device.Metadata.Archetype = archetype
		s.Devices[id] = device
	}
}

// SetGroupedLightBrightness optimistically updates a grouped light's brightness in the cache.
func (s *BridgeState) SetGroupedLightBrightness(id string, brightness float64) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if gl, ok := s.GroupedLights[id]; ok {
		if gl.Dimming != nil {
			gl.Dimming.Brightness = float32(brightness)
			s.GroupedLights[id] = gl
		}
	}
}

// SetLightOn optimistically updates a light's on state in the cache.
func (s *BridgeState) SetLightOn(id string, on bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if light, ok := s.Lights[id]; ok {
		light.On.On = on
		s.Lights[id] = light
	}
}

// SetGroupedLightOn optimistically updates a grouped light's on state in the cache.
func (s *BridgeState) SetGroupedLightOn(id string, on bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if gl, ok := s.GroupedLights[id]; ok {
		if gl.On == nil {
			gl.On = &hueclient.GroupedLightGetOn{}
		}
		gl.On.On = on
		s.GroupedLights[id] = gl
	}
}

// SetMotionSensorEnabled optimistically updates a motion sensor's enabled state in the cache.
func (s *BridgeState) SetMotionSensorEnabled(id string, enabled bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if motion, ok := s.MotionSensors[id]; ok {
		motion.Enabled = enabled
		s.MotionSensors[id] = motion
	}
}

// SetMotionSensorSensitivity optimistically updates a motion sensor's sensitivity in the cache.
func (s *BridgeState) SetMotionSensorSensitivity(id string, sensitivity int) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if motion, ok := s.MotionSensors[id]; ok {
		if motion.Sensitivity == nil {
			motion.Sensitivity = &hueclient.CameraMotionGetSensitivity{}
		}
		motion.Sensitivity.Sensitivity = sensitivity
		s.MotionSensors[id] = motion
	}
}

// SetTemperatureSensorEnabled optimistically updates a temperature sensor's enabled state in the cache.
func (s *BridgeState) SetTemperatureSensorEnabled(id string, enabled bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if temp, ok := s.Temperatures[id]; ok {
		temp.Enabled = enabled
		s.Temperatures[id] = temp
	}
}

// SetLightLevelSensorEnabled optimistically updates a light level sensor's enabled state in the cache.
func (s *BridgeState) SetLightLevelSensorEnabled(id string, enabled bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if ll, ok := s.LightLevels[id]; ok {
		ll.Enabled = enabled
		s.LightLevels[id] = ll
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
		light.On.On = *update.On
	}

	if update.Brightness != nil && light.Dimming != nil {
		light.Dimming.Brightness = float32(*update.Brightness)
	}

	if update.ColorXY != nil && light.Color != nil {
		light.Color.Xy.X = float32(update.ColorXY[0])
		light.Color.Xy.Y = float32(update.ColorXY[1])
	}

	if update.Mirek != nil && light.ColorTemperature != nil {
		light.ColorTemperature.Mirek = *update.Mirek
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
			gl.On = &hueclient.GroupedLightGetOn{}
		}
		gl.On.On = *on
	}

	if brightness != nil && gl.Dimming != nil {
		gl.Dimming.Brightness = float32(*brightness)
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

	// Update Motion field (value type in new API)
	m.Motion.Motion = motion
	// Also update MotionReport if it exists (keeps both in sync)
	if m.Motion.MotionReport != nil {
		m.Motion.MotionReport.Motion = motion
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

	active := hueclient.SceneGetStatusActive(status)
	scene.Status.Active = &active

	s.Scenes[id] = scene
	return true
}

// ApplySceneActionColor updates a scene action to use color mode (clears color temp).
// Returns (found, modeSwitched) - modeSwitched is true if this switched from color temp to color mode.
func (s *BridgeState) ApplySceneActionColor(sceneID, lightID string, x, y float32) (bool, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	scene, ok := s.Scenes[sceneID]
	if !ok {
		return false, false
	}

	for i, action := range scene.Actions {
		if action.Target.Rid == lightID {
			// Check if we're switching modes (was using color temp)
			modeSwitched := action.Action.ColorTemperature != nil
			// Set color
			scene.Actions[i].Action.Color = &hueclient.ActionGetActionColor{
				Xy: hueclient.XY{X: x, Y: y},
			}
			// Clear color temperature
			scene.Actions[i].Action.ColorTemperature = nil
			s.Scenes[sceneID] = scene
			return true, modeSwitched
		}
	}
	return false, false
}

// ApplySceneActionColorTemp updates a scene action to use color temp mode (clears color).
// Returns (found, modeSwitched) - modeSwitched is true if this switched from color to color temp mode.
func (s *BridgeState) ApplySceneActionColorTemp(sceneID, lightID string, mirek int) (bool, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	scene, ok := s.Scenes[sceneID]
	if !ok {
		return false, false
	}

	for i, action := range scene.Actions {
		if action.Target.Rid == lightID {
			// Check if we're switching modes (was using color)
			modeSwitched := action.Action.Color != nil
			// Set color temperature
			scene.Actions[i].Action.ColorTemperature = &hueclient.ActionGetActionColorTemperature{
				Mirek: mirek,
			}
			// Clear color
			scene.Actions[i].Action.Color = nil
			s.Scenes[sceneID] = scene
			return true, modeSwitched
		}
	}
	return false, false
}

// GetTemperature returns a temperature sensor by ID.
func (s *BridgeState) GetTemperature(id string) (hueclient.TemperatureGet, bool) {
	return getFromMap(s, s.Temperatures, id)
}

// GetDeviceRoom returns the room that contains a device, if any.
func (s *BridgeState) GetDeviceRoom(deviceID string) (hueclient.RoomGet, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, room := range s.Rooms {
		for _, child := range room.Children {
			if child.Rtype == hueclient.ResourceTypeDevice && child.Rid == deviceID {
				return room, true
			}
		}
	}
	return hueclient.RoomGet{}, false
}

// GetLightZones returns all zones that contain a light.
func (s *BridgeState) GetLightZones(lightID string) []hueclient.ZoneGet {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var zones []hueclient.ZoneGet
	for _, zone := range s.Zones {
		for _, child := range zone.Children {
			if child.Rtype == hueclient.ResourceTypeLight && child.Rid == lightID {
				zones = append(zones, zone)
				break
			}
		}
	}

	// Sort by name for stable ordering
	sort.Slice(zones, func(i, j int) bool {
		return zones[i].Metadata.Name < zones[j].Metadata.Name
	})

	return zones
}

// GetGroupedLightName returns the room or zone name for a grouped light.
func (s *BridgeState) GetGroupedLightName(groupedLightID string) string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Check rooms
	for _, room := range s.Rooms {
		for _, svc := range room.Services {
			if svc.Rtype == hueclient.ResourceTypeGroupedLight && svc.Rid == groupedLightID {
				if room.Metadata.Name != "" {
					return room.Metadata.Name
				}
			}
		}
	}

	// Check zones
	for _, zone := range s.Zones {
		for _, svc := range zone.Services {
			if svc.Rtype == hueclient.ResourceTypeGroupedLight && svc.Rid == groupedLightID {
				if zone.Metadata.Name != "" {
					return zone.Metadata.Name
				}
			}
		}
	}

	return ""
}
