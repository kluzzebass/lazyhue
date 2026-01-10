package hue

import (
	"context"

	"github.com/kluzzebass/lazyhue/internal/hueclient"
)

// isLightOn checks if a light is on.
func isLightOn(light hueclient.LightGet) bool {
	return light.On != nil && light.On.On != nil && *light.On.On
}

// isGroupedLightOn checks if a grouped light is on.
func isGroupedLightOn(gl hueclient.GroupedLightGet) bool {
	return gl.On != nil && gl.On.On != nil && *gl.On.On
}

// ToggleLight toggles a light on or off.
func (b *Bridge) ToggleLight(lightID string) error {
	b.mu.RLock()
	client := b.client
	b.mu.RUnlock()

	if client == nil {
		return ErrAuthFailed
	}

	light, ok := b.state.GetLight(lightID)
	if !ok {
		return ErrAuthFailed
	}

	newState := !isLightOn(light)

	// Optimistic update: apply to cache immediately
	b.state.SetLightOn(lightID, newState)

	_, err := client.UpdateLight(context.Background(), lightID, hueclient.UpdateLightJSONRequestBody{
		On: &hueclient.On{On: &newState},
	})
	return err
}

// SetLightOn turns a light on or off.
func (b *Bridge) SetLightOn(lightID string, on bool) error {
	b.mu.RLock()
	client := b.client
	b.mu.RUnlock()

	if client == nil {
		return ErrAuthFailed
	}

	// Optimistic update: apply to cache immediately
	b.state.SetLightOn(lightID, on)

	_, err := client.UpdateLight(context.Background(), lightID, hueclient.UpdateLightJSONRequestBody{
		On: &hueclient.On{On: &on},
	})
	return err
}

// SetLightBrightness sets a light's brightness (0-100).
func (b *Bridge) SetLightBrightness(lightID string, brightness float64) error {
	b.mu.RLock()
	client := b.client
	b.mu.RUnlock()

	if client == nil {
		return ErrAuthFailed
	}

	// Optimistic update: apply to cache immediately so rapid keypresses accumulate
	b.state.SetLightBrightness(lightID, brightness)

	br := float32(brightness)
	_, err := client.UpdateLight(context.Background(), lightID, hueclient.UpdateLightJSONRequestBody{
		Dimming: &hueclient.Dimming{Brightness: &br},
	})
	return err
}

// ToggleGroupedLight toggles a grouped light (room/zone).
func (b *Bridge) ToggleGroupedLight(groupedLightID string) error {
	b.mu.RLock()
	client := b.client
	b.mu.RUnlock()

	if client == nil {
		return ErrAuthFailed
	}

	gl, ok := b.state.GetGroupedLight(groupedLightID)
	if !ok {
		return ErrAuthFailed
	}

	newState := !isGroupedLightOn(gl)

	// Optimistic update: apply to cache immediately
	b.state.SetGroupedLightOn(groupedLightID, newState)

	_, err := client.UpdateGroupedLight(context.Background(), groupedLightID, hueclient.UpdateGroupedLightJSONRequestBody{
		On: &hueclient.On{On: &newState},
	})
	return err
}

// SetGroupedLightOn turns a grouped light on or off.
func (b *Bridge) SetGroupedLightOn(groupedLightID string, on bool) error {
	b.mu.RLock()
	client := b.client
	b.mu.RUnlock()

	if client == nil {
		return ErrAuthFailed
	}

	// Optimistic update: apply to cache immediately
	b.state.SetGroupedLightOn(groupedLightID, on)

	_, err := client.UpdateGroupedLight(context.Background(), groupedLightID, hueclient.UpdateGroupedLightJSONRequestBody{
		On: &hueclient.On{On: &on},
	})
	return err
}

// SetGroupedLightBrightness sets a grouped light's brightness.
func (b *Bridge) SetGroupedLightBrightness(groupedLightID string, brightness float64) error {
	b.mu.RLock()
	client := b.client
	b.mu.RUnlock()

	if client == nil {
		return ErrAuthFailed
	}

	// Optimistic update: apply to cache immediately so rapid keypresses accumulate
	b.state.SetGroupedLightBrightness(groupedLightID, brightness)

	br := float32(brightness)
	_, err := client.UpdateGroupedLight(context.Background(), groupedLightID, hueclient.UpdateGroupedLightJSONRequestBody{
		Dimming: &hueclient.Dimming{Brightness: &br},
	})
	return err
}

// RecallScene activates a scene.
func (b *Bridge) RecallScene(sceneID string) error {
	b.mu.RLock()
	client := b.client
	b.mu.RUnlock()

	if client == nil {
		return ErrAuthFailed
	}

	action := hueclient.SceneRecallAction("active")
	_, err := client.UpdateScene(context.Background(), sceneID, hueclient.UpdateSceneJSONRequestBody{
		Recall: &hueclient.SceneRecall{
			Action: &action,
		},
	})
	return err
}

// isMotionEnabled checks if a motion sensor is enabled.
func isMotionEnabled(motion hueclient.MotionGet) bool {
	return motion.Enabled != nil && *motion.Enabled
}

// ToggleMotionSensor toggles a motion sensor's enabled state.
func (b *Bridge) ToggleMotionSensor(motionID string) error {
	b.mu.RLock()
	client := b.client
	b.mu.RUnlock()

	if client == nil {
		return ErrAuthFailed
	}

	motion, ok := b.state.GetMotionSensor(motionID)
	if !ok {
		return ErrAuthFailed
	}

	newState := !isMotionEnabled(motion)

	// Optimistic update
	b.state.SetMotionSensorEnabled(motionID, newState)

	_, err := client.UpdateMotionSensor(context.Background(), motionID, hueclient.UpdateMotionSensorJSONRequestBody{
		Enabled: &newState,
	})
	return err
}

// SetMotionSensorEnabled sets a motion sensor's enabled state.
func (b *Bridge) SetMotionSensorEnabled(motionID string, enabled bool) error {
	b.mu.RLock()
	client := b.client
	b.mu.RUnlock()

	if client == nil {
		return ErrAuthFailed
	}

	// Optimistic update
	b.state.SetMotionSensorEnabled(motionID, enabled)

	_, err := client.UpdateMotionSensor(context.Background(), motionID, hueclient.UpdateMotionSensorJSONRequestBody{
		Enabled: &enabled,
	})
	return err
}

// SetMotionSensorSensitivity sets a motion sensor's sensitivity (0 to sensitivity_max).
func (b *Bridge) SetMotionSensorSensitivity(motionID string, sensitivity int) error {
	b.mu.RLock()
	client := b.client
	b.mu.RUnlock()

	if client == nil {
		return ErrAuthFailed
	}

	// Optimistic update
	b.state.SetMotionSensorSensitivity(motionID, sensitivity)

	_, err := client.UpdateMotionSensor(context.Background(), motionID, hueclient.UpdateMotionSensorJSONRequestBody{
		Sensitivity: &struct {
			Sensitivity *int `json:"sensitivity,omitempty"`
		}{
			Sensitivity: &sensitivity,
		},
	})
	return err
}
