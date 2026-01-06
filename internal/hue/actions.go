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

	return home.UpdateLight(lightID, openhue.LightPut{
		On: &openhue.On{On: &newState},
	})
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

	return home.UpdateLight(lightID, openhue.LightPut{
		On: &openhue.On{On: &on},
	})
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
	return home.UpdateLight(lightID, openhue.LightPut{
		Dimming: &openhue.Dimming{Brightness: &br},
	})
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

	return home.UpdateGroupedLight(groupedLightID, openhue.GroupedLightPut{
		On: &openhue.On{On: &newState},
	})
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

	return home.UpdateGroupedLight(groupedLightID, openhue.GroupedLightPut{
		On: &openhue.On{On: &on},
	})
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
	return home.UpdateGroupedLight(groupedLightID, openhue.GroupedLightPut{
		Dimming: &openhue.Dimming{Brightness: &br},
	})
}

// RecallScene activates a scene.
func (b *Bridge) RecallScene(sceneID string) error {
	b.mu.RLock()
	home := b.home
	b.mu.RUnlock()

	if home == nil {
		return ErrAuthFailed
	}

	action := openhue.SceneRecallAction("active")
	return home.UpdateScene(sceneID, openhue.ScenePut{
		Recall: &openhue.SceneRecall{
			Action: &action,
		},
	})
}
