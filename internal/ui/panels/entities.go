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
	EntityDevice
	EntityEntertainment
	EntityBridge
	EntityLightsCategory  // Aggregate for "Lights" folder
	EntityDevicesCategory // Aggregate for "Devices" folder
	EntityScenesCategory  // Aggregate for "Scenes" folder
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
