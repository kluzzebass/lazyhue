package hue

import (
	"github.com/openhue/openhue-go"
)

// ToggleLight toggles a light on or off.
func (b *Bridge) ToggleLight(lightID string) error {
	b.mu.RLock()
	home := b.home
	b.mu.RUnlock()

	if home == nil {
		return ErrAuthFailed
	}

	light, ok := b.state.GetLight(lightID)
	if !ok {
		return ErrAuthFailed
	}

	newState := !light.IsOn()

	// Optimistic update: apply to cache immediately
	b.state.SetLightOn(lightID, newState)

	if err := home.UpdateLight(lightID, openhue.LightPut{
		On: &openhue.On{On: &newState},
	}); err != nil {
		return err
	}

	// Fetch actual state to verify/correct
	return b.refreshLight(lightID)
}

// SetLightOn turns a light on or off.
func (b *Bridge) SetLightOn(lightID string, on bool) error {
	b.mu.RLock()
	home := b.home
	b.mu.RUnlock()

	if home == nil {
		return ErrAuthFailed
	}

	// Optimistic update: apply to cache immediately
	b.state.SetLightOn(lightID, on)

	if err := home.UpdateLight(lightID, openhue.LightPut{
		On: &openhue.On{On: &on},
	}); err != nil {
		return err
	}

	// Fetch actual state to verify/correct
	return b.refreshLight(lightID)
}

// SetLightBrightness sets a light's brightness (0-100).
func (b *Bridge) SetLightBrightness(lightID string, brightness float64) error {
	b.mu.RLock()
	home := b.home
	b.mu.RUnlock()

	if home == nil {
		return ErrAuthFailed
	}

	// Optimistic update: apply to cache immediately so rapid keypresses accumulate
	b.state.SetLightBrightness(lightID, brightness)

	br := openhue.Brightness(brightness)
	if err := home.UpdateLight(lightID, openhue.LightPut{
		Dimming: &openhue.Dimming{Brightness: &br},
	}); err != nil {
		return err
	}

	// Fetch actual state to verify/correct
	return b.refreshLight(lightID)
}

// ToggleGroupedLight toggles a grouped light (room/zone).
func (b *Bridge) ToggleGroupedLight(groupedLightID string) error {
	b.mu.RLock()
	home := b.home
	b.mu.RUnlock()

	if home == nil {
		return ErrAuthFailed
	}

	gl, ok := b.state.GetGroupedLight(groupedLightID)
	if !ok {
		return ErrAuthFailed
	}

	newState := !gl.IsOn()

	// Optimistic update: apply to cache immediately
	b.state.SetGroupedLightOn(groupedLightID, newState)

	if err := home.UpdateGroupedLight(groupedLightID, openhue.GroupedLightPut{
		On: &openhue.On{On: &newState},
	}); err != nil {
		return err
	}

	// Fetch actual state to verify/correct
	return b.refreshGroupedLight(groupedLightID)
}

// SetGroupedLightOn turns a grouped light on or off.
func (b *Bridge) SetGroupedLightOn(groupedLightID string, on bool) error {
	b.mu.RLock()
	home := b.home
	b.mu.RUnlock()

	if home == nil {
		return ErrAuthFailed
	}

	// Optimistic update: apply to cache immediately
	b.state.SetGroupedLightOn(groupedLightID, on)

	if err := home.UpdateGroupedLight(groupedLightID, openhue.GroupedLightPut{
		On: &openhue.On{On: &on},
	}); err != nil {
		return err
	}

	// Fetch actual state to verify/correct
	return b.refreshGroupedLight(groupedLightID)
}

// SetGroupedLightBrightness sets a grouped light's brightness.
func (b *Bridge) SetGroupedLightBrightness(groupedLightID string, brightness float64) error {
	b.mu.RLock()
	home := b.home
	b.mu.RUnlock()

	if home == nil {
		return ErrAuthFailed
	}

	// Optimistic update: apply to cache immediately so rapid keypresses accumulate
	b.state.SetGroupedLightBrightness(groupedLightID, brightness)

	br := openhue.Brightness(brightness)
	if err := home.UpdateGroupedLight(groupedLightID, openhue.GroupedLightPut{
		Dimming: &openhue.Dimming{Brightness: &br},
	}); err != nil {
		return err
	}

	// Fetch actual state to verify/correct
	return b.refreshGroupedLight(groupedLightID)
}

// RecallScene activates a scene.
func (b *Bridge) RecallScene(sceneID string) error {
	b.mu.RLock()
	home := b.home
	b.mu.RUnlock()

	if home == nil {
		return ErrAuthFailed
	}

	// Get the scene to find its target room/zone
	scene, ok := b.state.GetScene(sceneID)
	if !ok {
		return ErrAuthFailed
	}

	action := openhue.SceneRecallAction("active")
	if err := home.UpdateScene(sceneID, openhue.ScenePut{
		Recall: &openhue.SceneRecall{
			Action: &action,
		},
	}); err != nil {
		return err
	}

	// Refresh all lights (scene affects multiple lights)
	lights, err := home.GetLights()
	if err != nil {
		return err
	}
	b.state.UpdateLights(lights)

	// Refresh the grouped light for the scene's room/zone
	if scene.Group != nil && scene.Group.Rid != nil {
		// Find the grouped light for this room
		if room, ok := b.state.GetRoom(*scene.Group.Rid); ok {
			if gl, ok := b.state.RoomGroupedLight(room); ok && gl.Id != nil {
				if err := b.refreshGroupedLight(*gl.Id); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

// refreshLight fetches the current state of a single light and updates the cache.
// Note: openhue-go doesn't have GetLightById, so we fetch all lights.
func (b *Bridge) refreshLight(lightID string) error {
	b.mu.RLock()
	home := b.home
	b.mu.RUnlock()

	if home == nil {
		return ErrAuthFailed
	}

	lights, err := home.GetLights()
	if err != nil {
		return err
	}

	// Update only the specific light in our cache
	if light, ok := lights[lightID]; ok {
		b.state.SetLight(lightID, light)
	}

	return nil
}

// refreshGroupedLight fetches the current state of a grouped light and updates the cache.
func (b *Bridge) refreshGroupedLight(groupedLightID string) error {
	b.mu.RLock()
	home := b.home
	b.mu.RUnlock()

	if home == nil {
		return ErrAuthFailed
	}

	gl, err := home.GetGroupedLightById(groupedLightID)
	if err != nil {
		return err
	}

	if gl != nil {
		b.state.SetGroupedLight(groupedLightID, *gl)
	}

	return nil
}
