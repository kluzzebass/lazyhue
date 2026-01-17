package hue

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/kluzzebass/lazyhue/internal/hueclient"
)

// RecallScene activates a scene.
func (b *Bridge) RecallScene(sceneID string) error {
	b.mu.RLock()
	client := b.client
	b.mu.RUnlock()

	if client == nil {
		return ErrAuthFailed
	}

	sceneName := "Unknown"
	if scene, ok := b.state.GetScene(sceneID); ok {
		sceneName = b.state.GetSceneName(scene)
	}
	b.logRequest(fmt.Sprintf("%s: activated", sceneName))
	action := hueclient.UpdateSceneJSONBodyRecallActionActive
	_, err := client.UpdateScene(context.Background(), toResourceId(sceneID), hueclient.UpdateSceneJSONRequestBody{
		Recall: &struct {
			Action  *hueclient.UpdateSceneJSONBodyRecallAction `json:"action,omitempty"`
			Dimming *struct {
				Brightness *float32 `json:"brightness,omitempty"`
			} `json:"dimming,omitempty"`
			Duration *int `json:"duration,omitempty"`
		}{Action: &action},
	})
	return err
}

// RenameScene renames a scene.
func (b *Bridge) RenameScene(sceneID, newName string) error {
	b.mu.RLock()
	client := b.client
	b.mu.RUnlock()

	if client == nil {
		return ErrAuthFailed
	}

	sceneName := "Unknown"
	if scene, ok := b.state.GetScene(sceneID); ok {
		sceneName = b.state.GetSceneName(scene)
	}
	b.logRequest(fmt.Sprintf("%s: renamed to \"%s\"", sceneName, newName))
	_, err := client.UpdateScene(context.Background(), toResourceId(sceneID), hueclient.UpdateSceneJSONRequestBody{
		Metadata: &struct {
			Appdata *string `json:"appdata,omitempty"`
			Name    *string `json:"name,omitempty"`
		}{
			Name: &newName,
		},
	})
	return err
}

// DeleteScene deletes a scene from the bridge.
func (b *Bridge) DeleteScene(sceneID string) error {
	b.mu.RLock()
	client := b.client
	b.mu.RUnlock()

	if client == nil {
		return ErrAuthFailed
	}

	sceneName := "Unknown"
	if scene, ok := b.state.GetScene(sceneID); ok {
		sceneName = b.state.GetSceneName(scene)
	}

	b.logRequest(fmt.Sprintf("Deleting scene: %s", sceneName))

	// Remove from state immediately (optimistic)
	b.state.DeleteScene(sceneID)

	_, err := client.DeleteScene(context.Background(), toResourceId(sceneID))
	return err
}

// ActivateSmartScene activates a smart (automation) scene.
func (b *Bridge) ActivateSmartScene(sceneID string) error {
	b.mu.RLock()
	client := b.client
	b.mu.RUnlock()

	if client == nil {
		return ErrAuthFailed
	}

	sceneName := "Unknown"
	if scene, ok := b.state.GetSmartScene(sceneID); ok {
		sceneName = b.state.GetSmartSceneName(scene)
	}

	b.logRequest(fmt.Sprintf("Activating smart scene: %s", sceneName))

	action := hueclient.UpdateSmartSceneJSONBodyRecallActionActivate
	_, err := client.UpdateSmartScene(context.Background(), toResourceId(sceneID), hueclient.UpdateSmartSceneJSONRequestBody{
		Recall: &struct {
			Action *hueclient.UpdateSmartSceneJSONBodyRecallAction `json:"action,omitempty"`
		}{
			Action: &action,
		},
	})
	return err
}

// DeactivateSmartScene deactivates a smart (automation) scene.
func (b *Bridge) DeactivateSmartScene(sceneID string) error {
	b.mu.RLock()
	client := b.client
	b.mu.RUnlock()

	if client == nil {
		return ErrAuthFailed
	}

	sceneName := "Unknown"
	if scene, ok := b.state.GetSmartScene(sceneID); ok {
		sceneName = b.state.GetSmartSceneName(scene)
	}

	b.logRequest(fmt.Sprintf("Deactivating smart scene: %s", sceneName))

	action := hueclient.UpdateSmartSceneJSONBodyRecallActionDeactivate
	_, err := client.UpdateSmartScene(context.Background(), toResourceId(sceneID), hueclient.UpdateSmartSceneJSONRequestBody{
		Recall: &struct {
			Action *hueclient.UpdateSmartSceneJSONBodyRecallAction `json:"action,omitempty"`
		}{
			Action: &action,
		},
	})
	return err
}

// DeleteSmartScene deletes a smart scene from the bridge.
func (b *Bridge) DeleteSmartScene(sceneID string) error {
	b.mu.RLock()
	client := b.client
	b.mu.RUnlock()

	if client == nil {
		return ErrAuthFailed
	}

	sceneName := "Unknown"
	if scene, ok := b.state.GetSmartScene(sceneID); ok {
		sceneName = b.state.GetSmartSceneName(scene)
	}

	b.logRequest(fmt.Sprintf("Deleting smart scene: %s", sceneName))

	// Remove from state immediately (optimistic)
	b.state.DeleteSmartScene(sceneID)

	_, err := client.DeleteSmartScene(context.Background(), toResourceId(sceneID))
	return err
}

// sceneAction defines the structure for a scene action when creating a scene.
// This matches the CreateSceneJSONBody.Actions structure.
type sceneAction struct {
	Action sceneActionDetail `json:"action"`
	Target sceneTarget       `json:"target"`
}

type sceneTarget struct {
	Rid   string                 `json:"rid"`
	Rtype hueclient.ResourceType `json:"rtype"`
}

type sceneActionDetail struct {
	On               *sceneOn               `json:"on,omitempty"`
	Dimming          *sceneDimming          `json:"dimming,omitempty"`
	Color            *sceneColor            `json:"color,omitempty"`
	ColorTemperature *sceneColorTemperature `json:"color_temperature,omitempty"`
	Gradient         *sceneGradient         `json:"gradient,omitempty"`
}

type sceneOn struct {
	On bool `json:"on"`
}

type sceneDimming struct {
	Brightness float32 `json:"brightness"`
}

type sceneColor struct {
	Xy sceneXY `json:"xy"`
}

type sceneColorTemperature struct {
	Mirek int `json:"mirek"`
}

type sceneXY struct {
	X float32 `json:"x"`
	Y float32 `json:"y"`
}

type sceneGradient struct {
	Mode   *hueclient.CreateSceneJSONBodyActionsActionGradientMode `json:"mode,omitempty"`
	Points []sceneGradientPoint                                    `json:"points"`
}

type sceneGradientPoint struct {
	Color sceneColor `json:"color"`
}

// Types for UPDATE operations - use pointers to match UpdateSceneJSONBody structure
type sceneActionUpdate struct {
	Action sceneActionDetailUpdate `json:"action"`
	Target sceneTarget             `json:"target"`
}

type sceneActionDetailUpdate struct {
	On               *sceneOnUpdate               `json:"on,omitempty"`
	Dimming          *sceneDimmingUpdate          `json:"dimming,omitempty"`
	Color            *sceneColorUpdate            `json:"color,omitempty"`
	ColorTemperature *sceneColorTemperatureUpdate `json:"color_temperature,omitempty"`
}

type sceneOnUpdate struct {
	On *bool `json:"on,omitempty"`
}

type sceneDimmingUpdate struct {
	Brightness *float32 `json:"brightness,omitempty"`
}

type sceneColorUpdate struct {
	Xy *sceneXYUpdate `json:"xy,omitempty"`
}

type sceneXYUpdate struct {
	X float32 `json:"x"`
	Y float32 `json:"y"`
}

type sceneColorTemperatureUpdate struct {
	Mirek *int `json:"mirek,omitempty"`
}

// CreateSceneFromCurrentState creates a new scene for a room/zone using the current light states.
func (b *Bridge) CreateSceneFromCurrentState(groupID string, isZone bool, sceneName string) error {
	b.mu.RLock()
	client := b.client
	b.mu.RUnlock()

	if client == nil {
		return ErrAuthFailed
	}

	// Get the lights for this room/zone
	var lights []hueclient.LightGet
	var groupType hueclient.ResourceType

	if isZone {
		zone, ok := b.state.GetZone(groupID)
		if !ok {
			return fmt.Errorf("zone %s not found", groupID)
		}
		lights = b.state.ZoneLights(zone)
		groupType = hueclient.ResourceTypeZone
	} else {
		room, ok := b.state.GetRoom(groupID)
		if !ok {
			return fmt.Errorf("room %s not found", groupID)
		}
		lights = b.state.RoomLights(room)
		groupType = hueclient.ResourceTypeRoom
	}

	if len(lights) == 0 {
		return fmt.Errorf("no lights found in group")
	}

	// Build actions from current light states
	actions := make([]sceneAction, 0, len(lights))

	for _, light := range lights {
		action := sceneAction{
			Target: sceneTarget{
				Rid:   light.Id,
				Rtype: hueclient.ResourceTypeLight,
			},
			Action: sceneActionDetail{
				On: &sceneOn{On: light.On.On},
			},
		}

		// Capture dimming if present
		if light.Dimming != nil {
			action.Action.Dimming = &sceneDimming{Brightness: light.Dimming.Brightness}
		}

		// Capture gradient if present (takes precedence over color)
		if light.Gradient != nil && len(light.Gradient.Points) > 0 && light.On.On {
			gradientMode := hueclient.CreateSceneJSONBodyActionsActionGradientMode(light.Gradient.Mode)
			points := make([]sceneGradientPoint, len(light.Gradient.Points))

			for i, pt := range light.Gradient.Points {
				points[i] = sceneGradientPoint{
					Color: sceneColor{
						Xy: sceneXY{X: pt.Color.Xy.X, Y: pt.Color.Xy.Y},
					},
				}
			}

			action.Action.Gradient = &sceneGradient{
				Mode:   &gradientMode,
				Points: points,
			}
		} else if light.Color != nil && light.On.On {
			// Capture color if present (only if light is on and no gradient)
			action.Action.Color = &sceneColor{
				Xy: sceneXY{X: light.Color.Xy.X, Y: light.Color.Xy.Y},
			}
		} else if light.ColorTemperature != nil && light.ColorTemperature.MirekValid && light.On.On {
			// Capture color temperature if present (only if light is on and not using color/gradient)
			action.Action.ColorTemperature = &sceneColorTemperature{Mirek: light.ColorTemperature.Mirek}
		}

		actions = append(actions, action)
	}

	b.logRequest(fmt.Sprintf("Creating scene \"%s\" with %d lights", sceneName, len(lights)))

	// Create the scene using raw JSON body since the generated types don't match our custom action type
	sceneType := hueclient.CreateSceneJSONBodyTypeScene
	body := struct {
		Type     *hueclient.CreateSceneJSONBodyType `json:"type,omitempty"`
		Group    sceneTarget                        `json:"group"`
		Metadata struct {
			Name string `json:"name"`
		} `json:"metadata"`
		Actions []sceneAction `json:"actions"`
	}{
		Type: &sceneType,
		Group: sceneTarget{
			Rid:   groupID,
			Rtype: groupType,
		},
		Metadata: struct {
			Name string `json:"name"`
		}{
			Name: sceneName,
		},
		Actions: actions,
	}

	// Use CreateSceneWithBody with JSON marshaling
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("failed to marshal scene body: %w", err)
	}

	_, err = client.CreateSceneWithBody(context.Background(), "application/json", bytes.NewReader(jsonBody))
	return err
}

// buildSceneActionsWithPending builds a complete actions list from an existing scene,
// applying all pending modifications for the specified light.
func (b *Bridge) buildSceneActionsWithPending(sceneID, lightID string, pending *pendingSceneAction) ([]sceneActionUpdate, error) {
	scene, ok := b.state.GetScene(sceneID)
	if !ok {
		return nil, fmt.Errorf("scene %s not found", sceneID)
	}

	var actions []sceneActionUpdate
	found := false

	for _, existingAction := range scene.Actions {
		// Convert existing action to update format
		action := sceneActionUpdate{
			Target: sceneTarget{
				Rid:   existingAction.Target.Rid,
				Rtype: existingAction.Target.Rtype,
			},
			Action: sceneActionDetailUpdate{},
		}

		// Copy existing values
		if existingAction.Action.On != nil {
			on := existingAction.Action.On.On
			action.Action.On = &sceneOnUpdate{On: &on}
		}
		if existingAction.Action.Dimming != nil {
			bri := existingAction.Action.Dimming.Brightness
			action.Action.Dimming = &sceneDimmingUpdate{Brightness: &bri}
		}
		if existingAction.Action.Color != nil {
			action.Action.Color = &sceneColorUpdate{
				Xy: &sceneXYUpdate{
					X: existingAction.Action.Color.Xy.X,
					Y: existingAction.Action.Color.Xy.Y,
				},
			}
		}
		if existingAction.Action.ColorTemperature != nil && existingAction.Action.ColorTemperature.Mirek != 0 {
			mirek := existingAction.Action.ColorTemperature.Mirek
			action.Action.ColorTemperature = &sceneColorTemperatureUpdate{Mirek: &mirek}
		}

		// Apply pending modifications if this is the target light
		if existingAction.Target.Rid == lightID {
			if pending.on != nil {
				action.Action.On = &sceneOnUpdate{On: pending.on}
			}
			if pending.brightness != nil {
				action.Action.Dimming = &sceneDimmingUpdate{Brightness: pending.brightness}
			}
			if pending.colorX != nil && pending.colorY != nil {
				action.Action.Color = &sceneColorUpdate{Xy: &sceneXYUpdate{X: *pending.colorX, Y: *pending.colorY}}
			}
			if pending.colorTemp != nil {
				action.Action.ColorTemperature = &sceneColorTemperatureUpdate{Mirek: pending.colorTemp}
			}
			// Clear the opposite color mode when switching
			if pending.clearColorTemp {
				action.Action.ColorTemperature = nil
			}
			if pending.clearColor {
				action.Action.Color = nil
			}
			found = true
		}

		actions = append(actions, action)
	}

	if !found {
		return nil, fmt.Errorf("light %s not found in scene %s actions", lightID, sceneID)
	}

	return actions, nil
}

// sendDebouncedSceneUpdate sends the accumulated scene action changes.
func (b *Bridge) sendDebouncedSceneUpdate(sceneID, lightID string, pending *pendingSceneAction) {
	b.mu.RLock()
	client := b.client
	b.mu.RUnlock()

	if client == nil {
		return
	}

	sceneName := "Unknown"
	lightName := "Unknown"
	if scene, ok := b.state.GetScene(sceneID); ok {
		sceneName = b.state.GetSceneName(scene)
	}
	if light, ok := b.state.GetLight(lightID); ok {
		lightName = b.state.GetLightName(light)
	}

	// Build description of what changed
	var changes []string
	if pending.on != nil {
		if *pending.on {
			changes = append(changes, "on")
		} else {
			changes = append(changes, "off")
		}
	}
	if pending.brightness != nil {
		changes = append(changes, fmt.Sprintf("brightness %.0f%%", *pending.brightness))
	}
	if pending.colorX != nil && pending.colorY != nil {
		changes = append(changes, fmt.Sprintf("color (%.3f, %.3f)", *pending.colorX, *pending.colorY))
	}
	if pending.colorTemp != nil {
		kelvin := 1000000 / *pending.colorTemp
		changes = append(changes, fmt.Sprintf("color temp %dK", kelvin))
	}
	if len(changes) > 0 {
		b.logRequest(fmt.Sprintf("Scene \"%s\": %s → %s", sceneName, lightName, changes[0]))
	}

	// Build complete actions list with pending modifications
	actions, err := b.buildSceneActionsWithPending(sceneID, lightID, pending)
	if err != nil {
		b.logError(fmt.Sprintf("Scene update failed: %v", err))
		return
	}

	body := struct {
		Actions []sceneActionUpdate `json:"actions"`
	}{
		Actions: actions,
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		b.logError(fmt.Sprintf("Scene update marshal failed: %v", err))
		return
	}

	resp, err := client.UpdateSceneWithBody(context.Background(), toResourceId(sceneID), "application/json", bytes.NewReader(jsonBody))
	if err != nil {
		b.logError(fmt.Sprintf("Scene update failed: %v", err))
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(resp.Body)
		b.logError(fmt.Sprintf("Scene update failed: HTTP %d: %s", resp.StatusCode, string(respBody)))
	}
}

// scheduleSceneUpdate schedules a debounced scene action update.
// Multiple rapid changes to the same scene/light are accumulated and sent in one API call.
func (b *Bridge) scheduleSceneUpdate(sceneID, lightID string, update func(p *pendingSceneAction)) {
	key := sceneID + ":" + lightID

	b.debounceMu.Lock()
	defer b.debounceMu.Unlock()

	entry, exists := b.sceneDebounce[key]
	if exists {
		// Stop existing timer and accumulate the change
		entry.timer.Stop()
	} else {
		// Create new entry with pending action
		entry = &sceneDebounceEntry{
			pending: &pendingSceneAction{},
		}
		b.sceneDebounce[key] = entry
	}

	// Apply the update to pending
	update(entry.pending)

	// Schedule the API call
	entry.timer = time.AfterFunc(50*time.Millisecond, func() {
		// Copy pending data before cleanup
		b.debounceMu.Lock()
		pendingCopy := *entry.pending
		delete(b.sceneDebounce, key)
		b.debounceMu.Unlock()

		// Send the update
		b.sendDebouncedSceneUpdate(sceneID, lightID, &pendingCopy)
	})
}

// UpdateSceneActionOn updates the on/off state for a light within a scene.
// Rapid calls are debounced - changes are accumulated and sent in one API call.
func (b *Bridge) UpdateSceneActionOn(sceneID, lightID string, on bool) error {
	b.mu.RLock()
	client := b.client
	b.mu.RUnlock()

	if client == nil {
		return ErrAuthFailed
	}

	onCopy := on
	b.scheduleSceneUpdate(sceneID, lightID, func(p *pendingSceneAction) {
		p.on = &onCopy
	})

	return nil
}

// UpdateSceneActionBrightness updates the brightness for a light within a scene.
// Rapid calls are debounced - changes are accumulated and sent in one API call.
func (b *Bridge) UpdateSceneActionBrightness(sceneID, lightID string, brightness float32) error {
	b.mu.RLock()
	client := b.client
	b.mu.RUnlock()

	if client == nil {
		return ErrAuthFailed
	}

	briCopy := brightness
	b.scheduleSceneUpdate(sceneID, lightID, func(p *pendingSceneAction) {
		p.brightness = &briCopy
	})

	return nil
}

// UpdateSceneActionColor updates the color (XY) for a light within a scene.
// Rapid calls are debounced - changes are accumulated and sent in one API call.
// Returns (error, modeSwitched) - modeSwitched is true if this switched from color temp to color mode.
func (b *Bridge) UpdateSceneActionColor(sceneID, lightID string, x, y float32) (error, bool) {
	b.mu.RLock()
	client := b.client
	b.mu.RUnlock()

	if client == nil {
		return ErrAuthFailed, false
	}

	// Optimistic update: apply to local state immediately
	_, modeSwitched := b.state.ApplySceneActionColor(sceneID, lightID, x, y)

	xCopy, yCopy := x, y
	b.scheduleSceneUpdate(sceneID, lightID, func(p *pendingSceneAction) {
		p.colorX = &xCopy
		p.colorY = &yCopy
		p.clearColorTemp = true // Switch to color mode
	})

	return nil, modeSwitched
}

// UpdateSceneActionColorTemp updates the color temperature for a light within a scene.
// Rapid calls are debounced - changes are accumulated and sent in one API call.
// Returns (error, modeSwitched) - modeSwitched is true if this switched from color to color temp mode.
func (b *Bridge) UpdateSceneActionColorTemp(sceneID, lightID string, mirek int) (error, bool) {
	b.mu.RLock()
	client := b.client
	b.mu.RUnlock()

	if client == nil {
		return ErrAuthFailed, false
	}

	// Optimistic update: apply to local state immediately
	_, modeSwitched := b.state.ApplySceneActionColorTemp(sceneID, lightID, mirek)

	mirekCopy := mirek
	b.scheduleSceneUpdate(sceneID, lightID, func(p *pendingSceneAction) {
		p.colorTemp = &mirekCopy
		p.clearColor = true // Switch to color temp mode
	})

	return nil, modeSwitched
}
