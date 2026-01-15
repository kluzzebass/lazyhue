// Package panels provides UI panel components.
package panels

import (
	"github.com/kluzzebass/lazyhue/internal/hueclient"
)

// EntityType identifies what kind of entity an item represents.
type EntityType int

const (
	EntityRoom EntityType = iota
	EntityZone
	EntityLight
	EntityScene
	EntitySmartScene
	EntityZoneScene      // Scene belonging to a zone
	EntityZoneSmartScene // Smart scene belonging to a zone
	EntityDevice
	EntityEntertainment
	EntityBridge
	EntityLightsCategory         // Aggregate for "Lights" folder
	EntityDevicesCategory        // Aggregate for "Devices" folder
	EntityScenesCategory         // Aggregate for "Scenes" folder
	EntitySmartScenesCategory    // Aggregate for "Smart Scenes" folder
	EntityRoomsCategory          // Aggregate for "Rooms" folder
	EntityZonesCategory          // Aggregate for "Zones" folder
	EntityEntertainmentCategory  // Aggregate for "Entertainment Areas" folder
)

// String returns a human-readable name for the entity type.
func (t EntityType) String() string {
	switch t {
	case EntityRoom:
		return "Room"
	case EntityZone:
		return "Zone"
	case EntityLight:
		return "Light"
	case EntityScene:
		return "Scene"
	case EntitySmartScene:
		return "Smart Scene"
	case EntityZoneScene:
		return "Zone Scene"
	case EntityZoneSmartScene:
		return "Zone Smart Scene"
	case EntityDevice:
		return "Device"
	case EntityEntertainment:
		return "Entertainment"
	case EntityBridge:
		return "Bridge"
	case EntityLightsCategory:
		return "Lights"
	case EntityDevicesCategory:
		return "Devices"
	case EntityScenesCategory:
		return "Scenes"
	case EntitySmartScenesCategory:
		return "Smart Scenes"
	case EntityRoomsCategory:
		return "Rooms"
	case EntityZonesCategory:
		return "Zones"
	case EntityEntertainmentCategory:
		return "Entertainment Areas"
	default:
		return "Unknown"
	}
}

// LightsCategoryData holds aggregate data for a lights category folder.
type LightsCategoryData struct {
	ParentName string
	Lights     []hueclient.LightGet
}

// DevicesCategoryData holds aggregate data for a devices category folder.
type DevicesCategoryData struct {
	ParentName string
	Devices    []hueclient.DeviceGet
}

// ScenesCategoryData holds aggregate data for a scenes category folder.
type ScenesCategoryData struct {
	ParentName string
	Scenes     []hueclient.SceneGet
}

// SmartScenesCategoryData holds aggregate data for a smart scenes category folder.
type SmartScenesCategoryData struct {
	ParentName  string
	SmartScenes []hueclient.SmartSceneGet
}

// RoomsCategoryData holds aggregate data for a rooms category folder.
type RoomsCategoryData struct {
	BridgeID string
	Rooms    []hueclient.RoomGet
}

// ZonesCategoryData holds aggregate data for a zones category folder.
type ZonesCategoryData struct {
	BridgeID string
	Zones    []hueclient.RoomGet
}

// EntertainmentCategoryData holds aggregate data for an entertainment areas category folder.
type EntertainmentCategoryData struct {
	BridgeID       string
	Configurations []EntertainmentConfig
}

// EntertainmentConfig is a simplified entertainment configuration for category display.
type EntertainmentConfig struct {
	ID   string
	Name string
}

// IsLightOn checks if a light is on (nil-safe).
func IsLightOn(light hueclient.LightGet) bool {
	return light.On != nil && light.On.On != nil && *light.On.On
}

// IsGroupedLightOn checks if a grouped light is on (nil-safe).
func IsGroupedLightOn(gl hueclient.GroupedLightGet) bool {
	return gl.On != nil && gl.On.On != nil && *gl.On.On
}

// EntityItem represents an entity that can be displayed in a list or tree.
type EntityItem struct {
	ID             string
	Name           string
	Type           EntityType
	IsOn           bool
	RawPtr         any     // The underlying hueclient type for access to full data
	IndicatorColor string  // Hex color for the indicator (e.g., "#ff0000"), empty for default
	Brightness     float64 // Brightness level 0-100 for brightness indicator (lights, scenes)
	BridgeID       string  // ID of the bridge this entity belongs to
}

// FilterValue returns the value used for filtering (implements list.Item).
func (i EntityItem) FilterValue() string { return i.Name }

// Title returns the item title (implements list.DefaultItem).
func (i EntityItem) Title() string { return i.Name }

// Description returns the item description (implements list.DefaultItem).
func (i EntityItem) Description() string {
	return i.Type.String()
}
