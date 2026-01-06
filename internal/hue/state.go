package hue

import (
	"sort"
	"sync"

	"github.com/openhue/openhue-go"
)

// BridgeState holds cached entities from a bridge using openhue-go types directly.
// Relationships are stored as IDs and resolved on-demand.
type BridgeState struct {
	mu sync.RWMutex

	Lights        map[string]openhue.LightGet
	Rooms         map[string]openhue.RoomGet
	Zones         map[string]openhue.RoomGet // Zones use the same type as Rooms
	Scenes        map[string]openhue.SceneGet
	GroupedLights map[string]openhue.GroupedLightGet
	Devices       map[string]openhue.DeviceGet
	MotionSensors map[string]openhue.MotionGet
	Temperatures  map[string]openhue.TemperatureGet
	LightLevels   map[string]openhue.LightLevelGet
	DevicePowers  map[string]openhue.DevicePowerGet

	// Bridge-specific resources
	BridgeResource *openhue.BridgeGet     // The bridge resource itself
	BridgeHome     *openhue.BridgeHomeGet // The home associated with the bridge
	AuthApps       []AuthV1Entry          // Authenticated applications
}

// NewBridgeState creates an empty bridge state.
func NewBridgeState() *BridgeState {
	return &BridgeState{
		Lights:        make(map[string]openhue.LightGet),
		Rooms:         make(map[string]openhue.RoomGet),
		Zones:         make(map[string]openhue.RoomGet),
		Scenes:        make(map[string]openhue.SceneGet),
		GroupedLights: make(map[string]openhue.GroupedLightGet),
		Devices:       make(map[string]openhue.DeviceGet),
		MotionSensors: make(map[string]openhue.MotionGet),
		Temperatures:  make(map[string]openhue.TemperatureGet),
		LightLevels:   make(map[string]openhue.LightLevelGet),
		DevicePowers:  make(map[string]openhue.DevicePowerGet),
	}
}

// GetLight returns a light by ID.
func (s *BridgeState) GetLight(id string) (openhue.LightGet, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	l, ok := s.Lights[id]
	return l, ok
}

// GetRoom returns a room by ID.
func (s *BridgeState) GetRoom(id string) (openhue.RoomGet, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	r, ok := s.Rooms[id]
	return r, ok
}

// GetZone returns a zone by ID.
func (s *BridgeState) GetZone(id string) (openhue.RoomGet, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	z, ok := s.Zones[id]
	return z, ok
}

// GetGroupedLight returns a grouped light by ID.
func (s *BridgeState) GetGroupedLight(id string) (openhue.GroupedLightGet, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	g, ok := s.GroupedLights[id]
	return g, ok
}

// GetScene returns a scene by ID.
func (s *BridgeState) GetScene(id string) (openhue.SceneGet, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	sc, ok := s.Scenes[id]
	return sc, ok
}

// GetDevice returns a device by ID.
func (s *BridgeState) GetDevice(id string) (openhue.DeviceGet, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	d, ok := s.Devices[id]
	return d, ok
}

// RoomLights returns the lights belonging to a room by resolving device references, sorted by name.
func (s *BridgeState) RoomLights(room openhue.RoomGet) []openhue.LightGet {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Build device -> lights mapping
	deviceLights := make(map[string][]openhue.LightGet)
	for _, light := range s.Lights {
		if light.Owner != nil && light.Owner.Rid != nil {
			deviceLights[*light.Owner.Rid] = append(deviceLights[*light.Owner.Rid], light)
		}
	}

	// Room children are devices
	var lights []openhue.LightGet
	if room.Children != nil {
		for _, child := range *room.Children {
			if child.Rid != nil {
				if deviceLightList, ok := deviceLights[*child.Rid]; ok {
					lights = append(lights, deviceLightList...)
				}
			}
		}
	}
	
	// Sort by name for stable ordering
	sort.Slice(lights, func(i, j int) bool {
		nameI := ""
		nameJ := ""
		if lights[i].Metadata != nil && lights[i].Metadata.Name != nil {
			nameI = *lights[i].Metadata.Name
		}
		if lights[j].Metadata != nil && lights[j].Metadata.Name != nil {
			nameJ = *lights[j].Metadata.Name
		}
		return nameI < nameJ
	})
	
	return lights
}

// RoomGroupedLight returns the grouped light for a room.
func (s *BridgeState) RoomGroupedLight(room openhue.RoomGet) (openhue.GroupedLightGet, bool) {
	for svcID, svcType := range room.GetServices() {
		if svcType == openhue.ResourceIdentifierRtypeGroupedLight {
			return s.GetGroupedLight(svcID)
		}
	}
	return openhue.GroupedLightGet{}, false
}

// RoomScenes returns scenes belonging to a room, sorted by name.
func (s *BridgeState) RoomScenes(roomID string) []openhue.SceneGet {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var scenes []openhue.SceneGet
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
func (s *BridgeState) ZoneScenes(zoneID string) []openhue.SceneGet {
	// Zones use the same scene grouping as rooms
	return s.RoomScenes(zoneID)
}

// ZoneGroupedLight returns the grouped light for a zone.
func (s *BridgeState) ZoneGroupedLight(zone openhue.RoomGet) (openhue.GroupedLightGet, bool) {
	// Zones use the same services structure as rooms
	return s.RoomGroupedLight(zone)
}

// IsZoneOn returns true if any light in the zone is on.
func (s *BridgeState) IsZoneOn(zone openhue.RoomGet) bool {
	// Zones work the same way as rooms
	return s.IsRoomOn(zone)
}

// ZoneBrightness returns the grouped light brightness for a zone.
func (s *BridgeState) ZoneBrightness(zone openhue.RoomGet) float64 {
	// Zones work the same way as rooms
	return s.RoomBrightness(zone)
}

// AllRooms returns all rooms sorted by name.
func (s *BridgeState) AllRooms() []openhue.RoomGet {
	s.mu.RLock()
	defer s.mu.RUnlock()

	rooms := make([]openhue.RoomGet, 0, len(s.Rooms))
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
func (s *BridgeState) AllZones() []openhue.RoomGet {
	s.mu.RLock()
	defer s.mu.RUnlock()

	zones := make([]openhue.RoomGet, 0, len(s.Zones))
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
func (s *BridgeState) AllLights() []openhue.LightGet {
	s.mu.RLock()
	defer s.mu.RUnlock()

	lights := make([]openhue.LightGet, 0, len(s.Lights))
	for _, l := range s.Lights {
		lights = append(lights, l)
	}
	
	// Sort by name for stable ordering
	sort.Slice(lights, func(i, j int) bool {
		nameI := ""
		nameJ := ""
		if lights[i].Metadata != nil && lights[i].Metadata.Name != nil {
			nameI = *lights[i].Metadata.Name
		}
		if lights[j].Metadata != nil && lights[j].Metadata.Name != nil {
			nameJ = *lights[j].Metadata.Name
		}
		return nameI < nameJ
	})
	
	return lights
}

// AllScenes returns all scenes sorted by name.
func (s *BridgeState) AllScenes() []openhue.SceneGet {
	s.mu.RLock()
	defer s.mu.RUnlock()

	scenes := make([]openhue.SceneGet, 0, len(s.Scenes))
	for _, sc := range s.Scenes {
		scenes = append(scenes, sc)
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

// UpdateLights replaces the lights cache.
func (s *BridgeState) UpdateLights(lights map[string]openhue.LightGet) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Lights = lights
}

// UpdateRooms replaces the rooms cache.
func (s *BridgeState) UpdateRooms(rooms map[string]openhue.RoomGet) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Rooms = rooms
}

// UpdateZones replaces the zones cache.
func (s *BridgeState) UpdateZones(zones map[string]openhue.RoomGet) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Zones = zones
}

// UpdateGroupedLights replaces the grouped lights cache.
func (s *BridgeState) UpdateGroupedLights(grouped map[string]openhue.GroupedLightGet) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.GroupedLights = grouped
}

// UpdateScenes replaces the scenes cache.
func (s *BridgeState) UpdateScenes(scenes map[string]openhue.SceneGet) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Scenes = scenes
}

// UpdateDevices replaces the devices cache.
func (s *BridgeState) UpdateDevices(devices map[string]openhue.DeviceGet) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Devices = devices
}

// UpdateMotionSensors replaces the motion sensors cache.
func (s *BridgeState) UpdateMotionSensors(sensors map[string]openhue.MotionGet) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.MotionSensors = sensors
}

// GetMotionSensor returns a motion sensor by ID.
func (s *BridgeState) GetMotionSensor(id string) (openhue.MotionGet, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	m, ok := s.MotionSensors[id]
	return m, ok
}

// GetDeviceMotionState returns true if the device has a motion sensor service that detects motion.
func (s *BridgeState) GetDeviceMotionState(device openhue.DeviceGet) (hasMotion bool, isDetecting bool) {
	if device.Services == nil {
		return false, false
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, svc := range *device.Services {
		if svc.Rtype != nil && *svc.Rtype == openhue.ResourceIdentifierRtypeMotion && svc.Rid != nil {
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
	return false, false
}

// GetDeviceTemperature returns the temperature reading for a device if it has a temperature service.
func (s *BridgeState) GetDeviceTemperature(device openhue.DeviceGet) (hasTemp bool, tempC float32) {
	if device.Services == nil {
		return false, 0
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, svc := range *device.Services {
		if svc.Rtype != nil && *svc.Rtype == openhue.ResourceIdentifierRtypeTemperature && svc.Rid != nil {
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
	return false, 0
}

// GetDeviceLightLevel returns the light level reading for a device if it has a light_level service.
func (s *BridgeState) GetDeviceLightLevel(device openhue.DeviceGet) (hasLevel bool, level int) {
	if device.Services == nil {
		return false, 0
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, svc := range *device.Services {
		if svc.Rtype != nil && *svc.Rtype == openhue.ResourceIdentifierRtypeLightLevel && svc.Rid != nil {
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
	return false, 0
}

// GetDeviceBattery returns the battery status for a device if it has a device_power service.
func (s *BridgeState) GetDeviceBattery(device openhue.DeviceGet) (hasBattery bool, level int, state string) {
	if device.Services == nil {
		return false, 0, ""
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, svc := range *device.Services {
		if svc.Rtype != nil && *svc.Rtype == openhue.ResourceIdentifierRtypeDevicePower && svc.Rid != nil {
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
func (s *BridgeState) UpdateTemperatures(temps map[string]openhue.TemperatureGet) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Temperatures = temps
}

// UpdateLightLevels replaces the light levels cache.
func (s *BridgeState) UpdateLightLevels(levels map[string]openhue.LightLevelGet) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.LightLevels = levels
}

// UpdateDevicePowers replaces the device powers cache.
func (s *BridgeState) UpdateDevicePowers(powers map[string]openhue.DevicePowerGet) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.DevicePowers = powers
}

// UpdateBridgeResource sets the bridge resource.
func (s *BridgeState) UpdateBridgeResource(bridge *openhue.BridgeGet) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.BridgeResource = bridge
}

// UpdateBridgeHome sets the bridge home resource.
func (s *BridgeState) UpdateBridgeHome(home *openhue.BridgeHomeGet) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.BridgeHome = home
}

// GetBridgeResource returns the bridge resource.
func (s *BridgeState) GetBridgeResource() *openhue.BridgeGet {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.BridgeResource
}

// GetBridgeHome returns the bridge home resource.
func (s *BridgeState) GetBridgeHome() *openhue.BridgeHomeGet {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.BridgeHome
}

// GetBridgeDevice returns the device that owns the bridge service.
func (s *BridgeState) GetBridgeDevice() (openhue.DeviceGet, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, device := range s.Devices {
		if device.Services == nil {
			continue
		}
		for _, svc := range *device.Services {
			if svc.Rtype != nil && *svc.Rtype == openhue.ResourceIdentifierRtypeBridge {
				return device, true
			}
		}
	}
	return openhue.DeviceGet{}, false
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
func (s *BridgeState) AllDevices() []openhue.DeviceGet {
	s.mu.RLock()
	defer s.mu.RUnlock()

	devices := make([]openhue.DeviceGet, 0, len(s.Devices))
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

// IsRoomOn returns true if any light in the room is on.
func (s *BridgeState) IsRoomOn(room openhue.RoomGet) bool {
	if gl, ok := s.RoomGroupedLight(room); ok {
		return gl.IsOn()
	}
	for _, light := range s.RoomLights(room) {
		if light.IsOn() {
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
			if svc.Rtype != nil && *svc.Rtype == openhue.ResourceIdentifierRtypeBridge {
				if device.Metadata != nil && device.Metadata.Name != nil {
					return *device.Metadata.Name
				}
			}
		}
	}
	return ""
}

// RoomBrightness returns the grouped light brightness for a room.
func (s *BridgeState) RoomBrightness(room openhue.RoomGet) float64 {
	if gl, ok := s.RoomGroupedLight(room); ok {
		if gl.Dimming != nil && gl.Dimming.Brightness != nil {
			return float64(*gl.Dimming.Brightness)
		}
	}
	return 0
}

// SetLight updates a single light in the cache.
func (s *BridgeState) SetLight(id string, light openhue.LightGet) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Lights[id] = light
}

// SetGroupedLight updates a single grouped light in the cache.
func (s *BridgeState) SetGroupedLight(id string, gl openhue.GroupedLightGet) {
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
			br := openhue.Brightness(brightness)
			light.Dimming.Brightness = &br
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
			br := openhue.Brightness(brightness)
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
			light.On = &openhue.On{}
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
			gl.On = &openhue.On{}
		}
		gl.On.On = &on
		s.GroupedLights[id] = gl
	}
}

