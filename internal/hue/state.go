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
	Scenes        map[string]openhue.SceneGet
	GroupedLights map[string]openhue.GroupedLightGet
	Devices       map[string]openhue.DeviceGet
}

// NewBridgeState creates an empty bridge state.
func NewBridgeState() *BridgeState {
	return &BridgeState{
		Lights:        make(map[string]openhue.LightGet),
		Rooms:         make(map[string]openhue.RoomGet),
		Scenes:        make(map[string]openhue.SceneGet),
		GroupedLights: make(map[string]openhue.GroupedLightGet),
		Devices:       make(map[string]openhue.DeviceGet),
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

// RoomBrightness returns the grouped light brightness for a room.
func (s *BridgeState) RoomBrightness(room openhue.RoomGet) float64 {
	if gl, ok := s.RoomGroupedLight(room); ok {
		if gl.Dimming != nil && gl.Dimming.Brightness != nil {
			return float64(*gl.Dimming.Brightness)
		}
	}
	return 0
}

