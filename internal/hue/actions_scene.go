package hue

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"

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

// sceneActionModifier is a function that modifies a scene action in place.
type sceneActionModifier func(action *sceneActionUpdate)

// buildSceneActionsWithModification builds a complete actions list from an existing scene,
// applying a modification to the action with the specified target lightID.
// This is needed because the Hue API replaces ALL actions when you send an update,
// so we must include all existing actions with the modification applied.
func (b *Bridge) buildSceneActionsWithModification(sceneID, lightID string, modifier sceneActionModifier) ([]sceneActionUpdate, error) {
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

		// Apply modification if this is the target light
		if existingAction.Target.Rid == lightID {
			modifier(&action)
			found = true
		}

		actions = append(actions, action)
	}

	if !found {
		return nil, fmt.Errorf("light %s not found in scene %s actions", lightID, sceneID)
	}

	return actions, nil
}

// UpdateSceneActionOn updates the on/off state for a light within a scene.
func (b *Bridge) UpdateSceneActionOn(sceneID, lightID string, on bool) error {
	b.mu.RLock()
	client := b.client
	b.mu.RUnlock()

	if client == nil {
		return ErrAuthFailed
	}

	sceneName := "Unknown"
	lightName := "Unknown"
	if scene, ok := b.state.GetScene(sceneID); ok {
		sceneName = b.state.GetSceneName(scene)
	}
	if light, ok := b.state.GetLight(lightID); ok {
		lightName = b.state.GetLightName(light)
	}

	state := "off"
	if on {
		state = "on"
	}
	b.logRequest(fmt.Sprintf("Scene \"%s\": setting %s to %s", sceneName, lightName, state))

	// Build complete actions list with modification
	actions, err := b.buildSceneActionsWithModification(sceneID, lightID, func(action *sceneActionUpdate) {
		action.Action.On = &sceneOnUpdate{On: &on}
	})
	if err != nil {
		return err
	}

	body := struct {
		Actions []sceneActionUpdate `json:"actions"`
	}{
		Actions: actions,
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("failed to marshal scene update: %w", err)
	}

	resp, err := client.UpdateSceneWithBody(context.Background(), toResourceId(sceneID), "application/json", bytes.NewReader(jsonBody))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("scene update failed: HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	return nil
}

// UpdateSceneActionBrightness updates the brightness for a light within a scene.
func (b *Bridge) UpdateSceneActionBrightness(sceneID, lightID string, brightness float32) error {
	b.mu.RLock()
	client := b.client
	b.mu.RUnlock()

	if client == nil {
		return ErrAuthFailed
	}

	sceneName := "Unknown"
	lightName := "Unknown"
	if scene, ok := b.state.GetScene(sceneID); ok {
		sceneName = b.state.GetSceneName(scene)
	}
	if light, ok := b.state.GetLight(lightID); ok {
		lightName = b.state.GetLightName(light)
	}

	b.logRequest(fmt.Sprintf("Scene \"%s\": setting %s brightness to %.0f%%", sceneName, lightName, brightness))

	// Build complete actions list with modification
	actions, err := b.buildSceneActionsWithModification(sceneID, lightID, func(action *sceneActionUpdate) {
		action.Action.Dimming = &sceneDimmingUpdate{Brightness: &brightness}
	})
	if err != nil {
		return err
	}

	body := struct {
		Actions []sceneActionUpdate `json:"actions"`
	}{
		Actions: actions,
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("failed to marshal scene update: %w", err)
	}

	resp, err := client.UpdateSceneWithBody(context.Background(), toResourceId(sceneID), "application/json", bytes.NewReader(jsonBody))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("scene update failed: HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	return nil
}

// UpdateSceneActionColor updates the color (XY) for a light within a scene.
func (b *Bridge) UpdateSceneActionColor(sceneID, lightID string, x, y float32) error {
	b.mu.RLock()
	client := b.client
	b.mu.RUnlock()

	if client == nil {
		return ErrAuthFailed
	}

	sceneName := "Unknown"
	lightName := "Unknown"
	if scene, ok := b.state.GetScene(sceneID); ok {
		sceneName = b.state.GetSceneName(scene)
	}
	if light, ok := b.state.GetLight(lightID); ok {
		lightName = b.state.GetLightName(light)
	}

	b.logRequest(fmt.Sprintf("Scene \"%s\": setting %s color to (%.3f, %.3f)", sceneName, lightName, x, y))

	// Build complete actions list with modification
	actions, err := b.buildSceneActionsWithModification(sceneID, lightID, func(action *sceneActionUpdate) {
		action.Action.Color = &sceneColorUpdate{Xy: &sceneXYUpdate{X: x, Y: y}}
	})
	if err != nil {
		return err
	}

	body := struct {
		Actions []sceneActionUpdate `json:"actions"`
	}{
		Actions: actions,
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("failed to marshal scene update: %w", err)
	}

	resp, err := client.UpdateSceneWithBody(context.Background(), toResourceId(sceneID), "application/json", bytes.NewReader(jsonBody))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("scene update failed: HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	return nil
}

// UpdateSceneActionColorTemp updates the color temperature for a light within a scene.
func (b *Bridge) UpdateSceneActionColorTemp(sceneID, lightID string, mirek int) error {
	b.mu.RLock()
	client := b.client
	b.mu.RUnlock()

	if client == nil {
		return ErrAuthFailed
	}

	sceneName := "Unknown"
	lightName := "Unknown"
	if scene, ok := b.state.GetScene(sceneID); ok {
		sceneName = b.state.GetSceneName(scene)
	}
	if light, ok := b.state.GetLight(lightID); ok {
		lightName = b.state.GetLightName(light)
	}

	// Convert mirek to Kelvin for display (K = 1,000,000 / mirek)
	kelvin := 1000000 / mirek
	b.logRequest(fmt.Sprintf("Scene \"%s\": setting %s color temp to %dK", sceneName, lightName, kelvin))

	// Build complete actions list with modification
	actions, err := b.buildSceneActionsWithModification(sceneID, lightID, func(action *sceneActionUpdate) {
		action.Action.ColorTemperature = &sceneColorTemperatureUpdate{Mirek: &mirek}
	})
	if err != nil {
		return err
	}

	body := struct {
		Actions []sceneActionUpdate `json:"actions"`
	}{
		Actions: actions,
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("failed to marshal scene update: %w", err)
	}

	resp, err := client.UpdateSceneWithBody(context.Background(), toResourceId(sceneID), "application/json", bytes.NewReader(jsonBody))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("scene update failed: HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	return nil
}
