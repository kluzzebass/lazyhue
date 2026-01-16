package hue

import (
	"github.com/google/uuid"
	"github.com/kluzzebass/lazyhue/internal/hueclient"
)

// toResourceId converts a string ID to a ResourceId (UUID).
// Panics if the string is not a valid UUID.
func toResourceId(id string) hueclient.ResourceId {
	return uuid.MustParse(id)
}

// isLightOn checks if a light is on.
func isLightOn(light hueclient.LightGet) bool {
	return light.On.On
}

// isGroupedLightOn checks if a grouped light is on.
func isGroupedLightOn(gl hueclient.GroupedLightGet) bool {
	return gl.On != nil && gl.On.On
}

// isMotionEnabled checks if a motion sensor is enabled.
func isMotionEnabled(motion hueclient.MotionGet) bool {
	return motion.Enabled
}
