package hue

import (
	"context"
	"errors"
	"fmt"
	"time"

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

	lightName := "Unknown"
	if light, ok := b.state.GetLight(lightID); ok {
		lightName = b.state.GetLightName(light)
	}
	action := "on"
	if !newState {
		action = "off"
	}
	b.logRequest(fmt.Sprintf("%s: %s", lightName, action))
	httpResp, err := client.UpdateLightWithResponse(context.Background(), lightID, hueclient.UpdateLightJSONRequestBody{
		On: &hueclient.On{On: &newState},
	})
	if err != nil {
		b.logError(fmt.Sprintf("%s: %s failed: %v", lightName, action, err))
		return err
	}
	if httpResp != nil && httpResp.HTTPResponse != nil && httpResp.HTTPResponse.StatusCode >= 400 {
		errMsg := fmt.Sprintf("%s: %s failed: HTTP %d", lightName, action, httpResp.HTTPResponse.StatusCode)
		b.logError(errMsg)
		return errors.New(errMsg)
	}
	return nil
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

	// Get light name for logging - try multiple approaches
	lightName := "Unknown"
	if light, ok := b.state.GetLight(lightID); ok {
		lightName = b.state.GetLightName(light)
		if lightName == "Unknown" {
			// If GetLightName returned "Unknown", try to provide a more descriptive fallback
			lightName = fmt.Sprintf("Light %s", lightID)
		}
	} else {
		// Fallback: try to find light by searching all lights
		allLights := b.state.AllLights()
		for _, l := range allLights {
			if l.Id != nil && *l.Id == lightID {
				lightName = b.state.GetLightName(l)
				if lightName == "Unknown" {
					lightName = fmt.Sprintf("Light %s", lightID)
				}
				break
			}
		}
		// If still unknown, use a descriptive format instead of raw UUID
		if lightName == "Unknown" {
			lightName = fmt.Sprintf("Light %s", lightID)
		}
	}
	action := "on"
	if !on {
		action = "off"
	}
	b.logRequest(fmt.Sprintf("%s: %s", lightName, action))
	httpResp, err := client.UpdateLightWithResponse(context.Background(), lightID, hueclient.UpdateLightJSONRequestBody{
		On: &hueclient.On{On: &on},
	})
	if err != nil {
		b.logError(fmt.Sprintf("%s: %s failed: %v", lightName, action, err))
		return err
	}
	if httpResp != nil && httpResp.HTTPResponse != nil && httpResp.HTTPResponse.StatusCode >= 400 {
		errMsg := fmt.Sprintf("%s: %s failed: HTTP %d", lightName, action, httpResp.HTTPResponse.StatusCode)
		b.logError(errMsg)
		return errors.New(errMsg)
	}
	return nil
}

// SetLightBrightness sets a light's brightness (0-100).
// Rapid calls are debounced - state is updated immediately, but API call is delayed.
func (b *Bridge) SetLightBrightness(lightID string, brightness float64) error {
	b.mu.RLock()
	client := b.client
	b.mu.RUnlock()

	if client == nil {
		return ErrAuthFailed
	}

	// Optimistic update: apply to cache immediately so rapid keypresses accumulate
	b.state.SetLightBrightness(lightID, brightness)

	// Debounce the actual API call
	b.debounceMu.Lock()
	if timer, ok := b.brightnessDebounce[lightID]; ok {
		timer.Stop()
	}
	
	lightName := "Unknown"
	if light, ok := b.state.GetLight(lightID); ok {
		lightName = b.state.GetLightName(light)
	}

	br := float32(brightness)
	b.brightnessDebounce[lightID] = time.AfterFunc(50*time.Millisecond, func() {
		b.mu.RLock()
		client := b.client
		b.mu.RUnlock()

		if client != nil {
			b.logRequest(fmt.Sprintf("%s: brightness %.0f%%", lightName, brightness))
			httpResp, err := client.UpdateLightWithResponse(context.Background(), lightID, hueclient.UpdateLightJSONRequestBody{
				Dimming: &hueclient.Dimming{Brightness: &br},
			})
			if err != nil {
				b.logError(fmt.Sprintf("%s: brightness failed: %v", lightName, err))
			} else if httpResp != nil && httpResp.HTTPResponse != nil && httpResp.HTTPResponse.StatusCode >= 400 {
				b.logError(fmt.Sprintf("%s: brightness failed: HTTP %d", lightName, httpResp.HTTPResponse.StatusCode))
			}
		}
		
		// Clean up timer
		b.debounceMu.Lock()
		delete(b.brightnessDebounce, lightID)
		b.debounceMu.Unlock()
	})
	b.debounceMu.Unlock()

	return nil
}

// SetLightColor sets a light's color using CIE XY coordinates.
// Rapid calls are debounced - state is updated immediately, but API call is delayed.
func (b *Bridge) SetLightColor(lightID string, x, y float64) error {
	b.mu.RLock()
	client := b.client
	b.mu.RUnlock()

	if client == nil {
		return ErrAuthFailed
	}

	// Optimistic update
	b.state.SetLightColor(lightID, x, y)

	// Debounce the actual API call
	b.debounceMu.Lock()
	if timer, ok := b.colorDebounce[lightID]; ok {
		timer.Stop()
	}
	
	lightName := "Unknown"
	if light, ok := b.state.GetLight(lightID); ok {
		lightName = b.state.GetLightName(light)
	}
	
	xf, yf := float32(x), float32(y)
	b.colorDebounce[lightID] = time.AfterFunc(50*time.Millisecond, func() {
		b.mu.RLock()
		client := b.client
		b.mu.RUnlock()
		
		if client != nil {
			b.logRequest(fmt.Sprintf("%s: color", lightName))
			httpResp, err := client.UpdateLightWithResponse(context.Background(), lightID, hueclient.UpdateLightJSONRequestBody{
				Color: &hueclient.Color{
					Xy: &hueclient.GamutPosition{X: &xf, Y: &yf},
				},
			})
			if err != nil {
				b.logError(fmt.Sprintf("%s: color failed: %v", lightName, err))
			} else if httpResp != nil && httpResp.HTTPResponse != nil && httpResp.HTTPResponse.StatusCode >= 400 {
				b.logError(fmt.Sprintf("%s: color failed: HTTP %d", lightName, httpResp.HTTPResponse.StatusCode))
			}
		}
		
		// Clean up timer
		b.debounceMu.Lock()
		delete(b.colorDebounce, lightID)
		b.debounceMu.Unlock()
	})
	b.debounceMu.Unlock()

	return nil
}

// SetLightColorTemperature sets a light's color temperature in mirek (153-500).
// Rapid calls are debounced - state is updated immediately, but API call is delayed.
func (b *Bridge) SetLightColorTemperature(lightID string, mirek int) error {
	b.mu.RLock()
	client := b.client
	b.mu.RUnlock()

	if client == nil {
		return ErrAuthFailed
	}

	// Optimistic update
	b.state.SetLightColorTemperature(lightID, mirek)

	// Debounce the actual API call
	b.debounceMu.Lock()
	if timer, ok := b.colorTempDebounce[lightID]; ok {
		timer.Stop()
	}
	
	lightName := "Unknown"
	if light, ok := b.state.GetLight(lightID); ok {
		lightName = b.state.GetLightName(light)
	}
	
	b.colorTempDebounce[lightID] = time.AfterFunc(50*time.Millisecond, func() {
		b.mu.RLock()
		client := b.client
		b.mu.RUnlock()
		
		if client != nil {
			b.logRequest(fmt.Sprintf("%s: color temp %d mirek", lightName, mirek))
			httpResp, err := client.UpdateLightWithResponse(context.Background(), lightID, hueclient.UpdateLightJSONRequestBody{
				ColorTemperature: &hueclient.ColorTemperature{Mirek: &mirek},
			})
			if err != nil {
				b.logError(fmt.Sprintf("%s: color temp failed: %v", lightName, err))
			} else if httpResp != nil && httpResp.HTTPResponse != nil && httpResp.HTTPResponse.StatusCode >= 400 {
				b.logError(fmt.Sprintf("%s: color temp failed: HTTP %d", lightName, httpResp.HTTPResponse.StatusCode))
			}
		}
		
		// Clean up timer
		b.debounceMu.Lock()
		delete(b.colorTempDebounce, lightID)
		b.debounceMu.Unlock()
	})
	b.debounceMu.Unlock()

	return nil
}

// SetLightEffect sets a light's effect (candle, fire, prism, etc.).
func (b *Bridge) SetLightEffect(lightID string, effect hueclient.SupportedEffects) error {
	b.mu.RLock()
	client := b.client
	b.mu.RUnlock()

	if client == nil {
		return ErrAuthFailed
	}

	// Optimistic update
	b.state.SetLightEffect(lightID, effect)

	lightName := "Unknown"
	if light, ok := b.state.GetLight(lightID); ok {
		lightName = b.state.GetLightName(light)
	}
	effectName := EffectDisplayName(string(effect))
	b.logRequest(fmt.Sprintf("%s: effect %s", lightName, effectName))
	httpResp, err := client.UpdateLightWithResponse(context.Background(), lightID, hueclient.UpdateLightJSONRequestBody{
		Effects: &hueclient.Effects{Effect: &effect},
	})
	if err != nil {
		b.logError(fmt.Sprintf("%s: effect %s failed: %v", lightName, effectName, err))
		return err
	}
	if httpResp != nil && httpResp.HTTPResponse != nil && httpResp.HTTPResponse.StatusCode >= 400 {
		errMsg := fmt.Sprintf("%s: effect %s failed: HTTP %d", lightName, effectName, httpResp.HTTPResponse.StatusCode)
		b.logError(errMsg)
		return errors.New(errMsg)
	}
	return nil
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

	groupName := b.state.GetGroupedLightName(groupedLightID)
	if groupName == "" {
		groupName = "Unknown"
	}
	action := "on"
	if !newState {
		action = "off"
	}
	b.logRequest(fmt.Sprintf("%s: %s", groupName, action))
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

	groupName := b.state.GetGroupedLightName(groupedLightID)
	if groupName == "" {
		groupName = "Unknown"
	}
	action := "on"
	if !on {
		action = "off"
	}
	b.logRequest(fmt.Sprintf("%s: %s", groupName, action))
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

	groupName := b.state.GetGroupedLightName(groupedLightID)
	if groupName == "" {
		groupName = "Unknown"
	}
	b.logRequest(fmt.Sprintf("%s: brightness %.0f%%", groupName, brightness))
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

	sceneName := "Unknown"
	if scene, ok := b.state.GetScene(sceneID); ok {
		sceneName = b.state.GetSceneName(scene)
	}
	b.logRequest(fmt.Sprintf("%s: activated", sceneName))
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

	deviceName := "Unknown"
	if motion.Owner != nil && motion.Owner.Rid != nil {
	if device, ok := b.state.GetDevice(*motion.Owner.Rid); ok {
		deviceName = b.state.GetDeviceName(device)
		}
	}
	action := "sensor enabled"
	if !newState {
		action = "sensor disabled"
	}
	b.logRequest(fmt.Sprintf("%s: %s", deviceName, action))
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

	deviceName := "Unknown"
	if motion, ok := b.state.GetMotionSensor(motionID); ok {
	if motion.Owner != nil && motion.Owner.Rid != nil {
		if device, ok := b.state.GetDevice(*motion.Owner.Rid); ok {
			deviceName = b.state.GetDeviceName(device)
		}
		}
	}
	action := "sensor enabled"
	if !enabled {
		action = "sensor disabled"
	}
	b.logRequest(fmt.Sprintf("%s: %s", deviceName, action))
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

	deviceName := "Unknown"
	if motion, ok := b.state.GetMotionSensor(motionID); ok {
	if motion.Owner != nil && motion.Owner.Rid != nil {
		if device, ok := b.state.GetDevice(*motion.Owner.Rid); ok {
			deviceName = b.state.GetDeviceName(device)
		}
		}
	}
	b.logRequest(fmt.Sprintf("%s: sensitivity %d", deviceName, sensitivity))
	_, err := client.UpdateMotionSensor(context.Background(), motionID, hueclient.UpdateMotionSensorJSONRequestBody{
		Sensitivity: &struct {
		Sensitivity *int `json:"sensitivity,omitempty"`
		}{
		Sensitivity: &sensitivity,
		},
	})
	return err
}

// SetTemperatureSensorEnabled enables or disables a temperature sensor.
func (b *Bridge) SetTemperatureSensorEnabled(temperatureID string, enabled bool) error {
	b.mu.RLock()
	client := b.client
	b.mu.RUnlock()

	if client == nil {
		return ErrAuthFailed
	}

	// Optimistic update
	b.state.SetTemperatureSensorEnabled(temperatureID, enabled)

	deviceName := "Unknown"
	if temp, ok := b.state.GetTemperature(temperatureID); ok {
		if temp.Owner != nil && temp.Owner.Rid != nil {
			if device, ok := b.state.GetDevice(*temp.Owner.Rid); ok {
				deviceName = b.state.GetDeviceName(device)
			}
		}
	}
	action := "temperature sensor enabled"
	if !enabled {
		action = "temperature sensor disabled"
	}
	b.logRequest(fmt.Sprintf("%s: %s", deviceName, action))
	_, err := client.UpdateTemperature(context.Background(), temperatureID, hueclient.UpdateTemperatureJSONRequestBody{
		Enabled: &enabled,
	})
	return err
}

// SetLightLevelSensorEnabled enables or disables a light level sensor.
func (b *Bridge) SetLightLevelSensorEnabled(lightLevelID string, enabled bool) error {
	b.mu.RLock()
	client := b.client
	b.mu.RUnlock()

	if client == nil {
		return ErrAuthFailed
	}

	// Optimistic update
	b.state.SetLightLevelSensorEnabled(lightLevelID, enabled)

	deviceName := "Unknown"
	if ll, ok := b.state.GetLightLevel(lightLevelID); ok {
		if ll.Owner != nil && ll.Owner.Rid != nil {
			if device, ok := b.state.GetDevice(*ll.Owner.Rid); ok {
				deviceName = b.state.GetDeviceName(device)
			}
		}
	}
	action := "light level sensor enabled"
	if !enabled {
		action = "light level sensor disabled"
	}
	b.logRequest(fmt.Sprintf("%s: %s", deviceName, action))
	_, err := client.UpdateLightLevel(context.Background(), lightLevelID, hueclient.UpdateLightLevelJSONRequestBody{
		Enabled: &enabled,
	})
	return err
}

// RenameDevice renames a device.
func (b *Bridge) RenameDevice(deviceID, newName string) error {
	b.mu.RLock()
	client := b.client
	b.mu.RUnlock()

	if client == nil {
		return ErrAuthFailed
	}

	deviceName := "Unknown"
	if device, ok := b.state.GetDevice(deviceID); ok {
		deviceName = b.state.GetDeviceName(device)
	}
	b.logRequest(fmt.Sprintf("%s: renamed to \"%s\"", deviceName, newName))
	_, err := client.UpdateDevice(context.Background(), deviceID, hueclient.UpdateDeviceJSONRequestBody{
		Metadata: &struct {
		Archetype *hueclient.ProductArchetype `json:"archetype,omitempty"`
		Name      *string                     `json:"name,omitempty"`
		}{
		Name: &newName,
		},
	})
	return err
}

// IdentifyDevice triggers a visual identification sequence on a device.
// The bridge performs Zigbee LED identification cycles for 5 seconds.
// Lights perform one breathe cycle. Sensors perform LED identification cycles for 15 seconds.
func (b *Bridge) IdentifyDevice(deviceID string) error {
	b.mu.RLock()
	client := b.client
	b.mu.RUnlock()

	if client == nil {
		return ErrAuthFailed
	}

	deviceName := "Unknown"
	if device, ok := b.state.GetDevice(deviceID); ok {
		deviceName = b.state.GetDeviceName(device)
	}
	b.logRequest(fmt.Sprintf("%s: identify", deviceName))

	action := hueclient.DevicePutIdentifyAction("identify")
	_, err := client.UpdateDevice(context.Background(), deviceID, hueclient.UpdateDeviceJSONRequestBody{
		Identify: &struct {
			Action *hueclient.DevicePutIdentifyAction `json:"action,omitempty"`
		}{
			Action: &action,
		},
	})
	return err
}

// RenameRoom renames a room.
func (b *Bridge) RenameRoom(roomID, newName string) error {
	b.mu.RLock()
	client := b.client
	b.mu.RUnlock()

	if client == nil {
		return ErrAuthFailed
	}

	roomName := "Unknown"
	if room, ok := b.state.GetRoom(roomID); ok {
		roomName = b.state.GetRoomName(room)
	}
	b.logRequest(fmt.Sprintf("%s: renamed to \"%s\"", roomName, newName))
	_, err := client.UpdateRoom(context.Background(), roomID, hueclient.UpdateRoomJSONRequestBody{
		Metadata: &struct {
			Archetype *hueclient.RoomArchetype `json:"archetype,omitempty"`
			Name      *string                  `json:"name,omitempty"`
		}{
			Name: &newName,
		},
	})
	return err
}

// RenameZone renames a zone.
func (b *Bridge) RenameZone(zoneID, newName string) error {
	b.mu.RLock()
	client := b.client
	b.mu.RUnlock()

	if client == nil {
		return ErrAuthFailed
	}

	zoneName := "Unknown"
	if zone, ok := b.state.GetZone(zoneID); ok {
		zoneName = zone.RoomName("")
	}
	b.logRequest(fmt.Sprintf("%s: renamed to \"%s\"", zoneName, newName))
	_, err := client.UpdateZone(context.Background(), zoneID, hueclient.UpdateZoneJSONRequestBody{
		Metadata: &struct {
			Archetype *hueclient.RoomArchetype `json:"archetype,omitempty"`
			Name      *string                  `json:"name,omitempty"`
		}{
			Name: &newName,
		},
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
	_, err := client.UpdateScene(context.Background(), sceneID, hueclient.UpdateSceneJSONRequestBody{
		Metadata: &hueclient.SceneMetadata{
			Name: &newName,
		},
	})
	return err
}

// CreateRoom creates a new room with the given name and archetype.
// deviceIDs should be the IDs of devices to add to the room.
func (b *Bridge) CreateRoom(name string, archetype hueclient.RoomArchetype, deviceIDs []string) error {
	b.mu.RLock()
	client := b.client
	b.mu.RUnlock()

	if client == nil {
		return ErrAuthFailed
	}

	// Build children list from device IDs
	children := make([]hueclient.ResourceIdentifier, len(deviceIDs))
	deviceType := hueclient.ResourceIdentifierRtypeDevice
	for i, id := range deviceIDs {
		idCopy := id
		children[i] = hueclient.ResourceIdentifier{
		Rid:   &idCopy,
		Rtype: &deviceType,
		}
	}

	b.logRequest(fmt.Sprintf("Room \"%s\": created", name))
	_, err := client.CreateRoom(context.Background(), hueclient.CreateRoomJSONRequestBody{
		Metadata: &struct {
		Archetype *hueclient.RoomArchetype `json:"archetype,omitempty"`
		Name      *string                  `json:"name,omitempty"`
		}{
		Name:      &name,
		Archetype: &archetype,
		},
		Children: &children,
	})
	return err
}

// CreateZone creates a new zone with the given name and archetype.
// serviceIDs should be the IDs of services (lights, grouped_lights) to add to the zone.
func (b *Bridge) CreateZone(name string, archetype hueclient.RoomArchetype, serviceIDs []string) error {
	b.mu.RLock()
	client := b.client
	b.mu.RUnlock()

	if client == nil {
		return ErrAuthFailed
	}

	// Build children list from service IDs (typically lights)
	children := make([]hueclient.ResourceIdentifier, len(serviceIDs))
	lightType := hueclient.ResourceIdentifierRtypeLight
	for i, id := range serviceIDs {
		idCopy := id
		children[i] = hueclient.ResourceIdentifier{
		Rid:   &idCopy,
		Rtype: &lightType,
		}
	}

	b.logRequest(fmt.Sprintf("Zone \"%s\": created", name))
	_, err := client.CreateZone(context.Background(), hueclient.CreateZoneJSONRequestBody{
		Metadata: &struct {
		Archetype *hueclient.RoomArchetype `json:"archetype,omitempty"`
		Name      *string                  `json:"name,omitempty"`
		}{
		Name:      &name,
		Archetype: &archetype,
		},
		Children: &children,
	})
	return err
}

// DeleteRoom deletes a room by its ID.
func (b *Bridge) DeleteRoom(roomID string) error {
	b.mu.RLock()
	client := b.client
	b.mu.RUnlock()

	if client == nil {
		return ErrAuthFailed
	}

	roomName := "Unknown"
	if room, ok := b.state.GetRoom(roomID); ok {
		roomName = b.state.GetRoomName(room)
		}
	b.logRequest(fmt.Sprintf("%s: deleted", roomName))
	_, err := client.DeleteRoom(context.Background(), roomID)
	return err
}

// DeleteZone deletes a zone by its ID.
func (b *Bridge) DeleteZone(zoneID string) error {
	b.mu.RLock()
	client := b.client
	b.mu.RUnlock()

	if client == nil {
		return ErrAuthFailed
	}

	zoneName := "Unknown"
	if zone, ok := b.state.GetZone(zoneID); ok {
		zoneName = zone.RoomName("")
		}
	b.logRequest(fmt.Sprintf("%s: deleted", zoneName))
	_, err := client.DeleteZone(context.Background(), zoneID)
	return err
}

// UpdateRoomDevices updates the devices assigned to a room.
func (b *Bridge) UpdateRoomDevices(roomID string, deviceIDs []string) error {
	b.mu.RLock()
	client := b.client
	b.mu.RUnlock()

	if client == nil {
		return ErrAuthFailed
	}

	roomName := "Unknown"
	if room, ok := b.state.GetRoom(roomID); ok {
		roomName = b.state.GetRoomName(room)
	}
	b.logRequest(fmt.Sprintf("%s: updated devices (%d)", roomName, len(deviceIDs)))

	// Build children list from device IDs
	children := make([]hueclient.ResourceIdentifier, len(deviceIDs))
	deviceType := hueclient.ResourceIdentifierRtypeDevice
	for i, id := range deviceIDs {
		idCopy := id
		children[i] = hueclient.ResourceIdentifier{
		Rid:   &idCopy,
		Rtype: &deviceType,
		}
	}

	_, err := client.UpdateRoom(context.Background(), roomID, hueclient.UpdateRoomJSONRequestBody{
		Children: &children,
	})
	return err
}

// MoveDeviceToRoom moves a device from its current room to a new room.
// If newRoomID is empty, the device is removed from its current room without being added to another.
func (b *Bridge) MoveDeviceToRoom(deviceID, newRoomID string) error {
	// Find current room
	oldRoom, hasOldRoom := b.state.GetDeviceRoom(deviceID)
	oldRoomID := ""
	if hasOldRoom && oldRoom.Id != nil {
		oldRoomID = *oldRoom.Id
	}

	// If already in the target room, nothing to do
	if oldRoomID == newRoomID {
		return nil
	}

	// Remove from old room if it was in one
	if hasOldRoom && oldRoom.Children != nil {
		var remainingDevices []string
		for _, child := range *oldRoom.Children {
			if child.Rid != nil && child.Rtype != nil &&
				*child.Rtype == hueclient.ResourceIdentifierRtypeDevice &&
				*child.Rid != deviceID {
				remainingDevices = append(remainingDevices, *child.Rid)
			}
		}
		if err := b.UpdateRoomDevices(oldRoomID, remainingDevices); err != nil {
			return fmt.Errorf("remove device from old room: %w", err)
		}
	}

	// Add to new room if specified
	if newRoomID != "" {
		newRoom, ok := b.state.GetRoom(newRoomID)
		if !ok {
			return fmt.Errorf("room not found: %s", newRoomID)
		}

		var newDevices []string
		if newRoom.Children != nil {
			for _, child := range *newRoom.Children {
				if child.Rid != nil && child.Rtype != nil &&
					*child.Rtype == hueclient.ResourceIdentifierRtypeDevice {
					newDevices = append(newDevices, *child.Rid)
				}
			}
		}
		newDevices = append(newDevices, deviceID)

		if err := b.UpdateRoomDevices(newRoomID, newDevices); err != nil {
			return fmt.Errorf("add device to new room: %w", err)
		}
	}

	return nil
}

// UpdateZoneServices updates the services (lights) assigned to a zone.
func (b *Bridge) UpdateZoneServices(zoneID string, serviceIDs []string) error {
	b.mu.RLock()
	client := b.client
	b.mu.RUnlock()

	if client == nil {
		return ErrAuthFailed
	}

	zoneName := "Unknown"
	if zone, ok := b.state.GetZone(zoneID); ok {
		zoneName = zone.RoomName("")
	}
	b.logRequest(fmt.Sprintf("%s: updated services (%d)", zoneName, len(serviceIDs)))

	// Build children list from service IDs
	children := make([]hueclient.ResourceIdentifier, len(serviceIDs))
	lightType := hueclient.ResourceIdentifierRtypeLight
	for i, id := range serviceIDs {
		idCopy := id
		children[i] = hueclient.ResourceIdentifier{
		Rid:   &idCopy,
		Rtype: &lightType,
		}
	}

	_, err := client.UpdateZone(context.Background(), zoneID, hueclient.UpdateZoneJSONRequestBody{
		Children: &children,
	})
	return err
}

// AddLightToZone adds a light to a zone.
func (b *Bridge) AddLightToZone(lightID, zoneID string) error {
	zone, ok := b.state.GetZone(zoneID)
	if !ok {
		return fmt.Errorf("zone not found: %s", zoneID)
	}

	// Get current light IDs in the zone
	var lightIDs []string
	if zone.Children != nil {
		for _, child := range *zone.Children {
			if child.Rid != nil && child.Rtype != nil &&
				*child.Rtype == hueclient.ResourceIdentifierRtypeLight {
				// Check if light is already in the zone
				if *child.Rid == lightID {
					return nil // Already in zone, nothing to do
				}
				lightIDs = append(lightIDs, *child.Rid)
			}
		}
	}

	// Add the new light
	lightIDs = append(lightIDs, lightID)
	return b.UpdateZoneServices(zoneID, lightIDs)
}

// RemoveLightFromZone removes a light from a zone.
func (b *Bridge) RemoveLightFromZone(lightID, zoneID string) error {
	zone, ok := b.state.GetZone(zoneID)
	if !ok {
		return fmt.Errorf("zone not found: %s", zoneID)
	}

	// Get current light IDs in the zone, excluding the one to remove
	var lightIDs []string
	found := false
	if zone.Children != nil {
		for _, child := range *zone.Children {
			if child.Rid != nil && child.Rtype != nil &&
				*child.Rtype == hueclient.ResourceIdentifierRtypeLight {
				if *child.Rid == lightID {
					found = true
					continue // Skip this light
				}
				lightIDs = append(lightIDs, *child.Rid)
			}
		}
	}

	if !found {
		return nil // Light wasn't in zone, nothing to do
	}

	return b.UpdateZoneServices(zoneID, lightIDs)
}

// SetRoomArchetype updates a room's archetype.
func (b *Bridge) SetRoomArchetype(roomID string, archetype hueclient.RoomArchetype) error {
	b.mu.RLock()
	client := b.client
	b.mu.RUnlock()

	if client == nil {
		return ErrAuthFailed
	}

	roomName := "Unknown"
	if room, ok := b.state.GetRoom(roomID); ok {
		roomName = b.state.GetRoomName(room)
		}
	b.logRequest(fmt.Sprintf("%s: renamed to \"%s\"", roomName, roomName))
	_, err := client.UpdateRoom(context.Background(), roomID, hueclient.UpdateRoomJSONRequestBody{
		Metadata: &struct {
		Archetype *hueclient.RoomArchetype `json:"archetype,omitempty"`
		Name      *string                  `json:"name,omitempty"`
		}{
		Archetype: &archetype,
		},
	})
	return err
}

// SetZoneArchetype updates a zone's archetype.
func (b *Bridge) SetZoneArchetype(zoneID string, archetype hueclient.RoomArchetype) error {
	b.mu.RLock()
	client := b.client
	b.mu.RUnlock()

	if client == nil {
		return ErrAuthFailed
	}

	zoneName := "Unknown"
	if zone, ok := b.state.GetZone(zoneID); ok {
		zoneName = zone.RoomName("")
	}
	b.logRequest(fmt.Sprintf("%s: archetype changed", zoneName))
	_, err := client.UpdateZone(context.Background(), zoneID, hueclient.UpdateZoneJSONRequestBody{
		Metadata: &struct {
		Archetype *hueclient.RoomArchetype `json:"archetype,omitempty"`
		Name      *string                  `json:"name,omitempty"`
		}{
		Archetype: &archetype,
		},
	})
	return err
}

// SetDeviceName updates a device's name.
func (b *Bridge) SetDeviceName(deviceID string, name string) error {
	b.mu.RLock()
	client := b.client
	b.mu.RUnlock()

	if client == nil {
		return ErrAuthFailed
	}

	oldName := "Unknown"
	if device, ok := b.state.GetDevice(deviceID); ok && device.Metadata != nil && device.Metadata.Name != nil {
		oldName = *device.Metadata.Name
	}
	b.logRequest(fmt.Sprintf("%s: renamed to \"%s\"", oldName, name))
	_, err := client.UpdateDevice(context.Background(), deviceID, hueclient.UpdateDeviceJSONRequestBody{
		Metadata: &struct {
			Archetype *hueclient.ProductArchetype `json:"archetype,omitempty"`
			Name      *string                     `json:"name,omitempty"`
		}{
			Name: &name,
		},
	})
	return err
}

// SetDeviceArchetype updates a device's archetype.
func (b *Bridge) SetDeviceArchetype(deviceID string, archetype hueclient.ProductArchetype) error {
	b.mu.RLock()
	client := b.client
	b.mu.RUnlock()

	if client == nil {
		return ErrAuthFailed
	}

	deviceName := "Unknown"
	if device, ok := b.state.GetDevice(deviceID); ok && device.Metadata != nil && device.Metadata.Name != nil {
		deviceName = *device.Metadata.Name
	}

	// Optimistic update: apply to cache immediately
	b.state.SetDeviceArchetype(deviceID, archetype)

	b.logRequest(fmt.Sprintf("%s: archetype changed to \"%s\"", deviceName, string(archetype)))
	_, err := client.UpdateDevice(context.Background(), deviceID, hueclient.UpdateDeviceJSONRequestBody{
		Metadata: &struct {
			Archetype *hueclient.ProductArchetype `json:"archetype,omitempty"`
			Name      *string                     `json:"name,omitempty"`
		}{
			Archetype: &archetype,
		},
	})
	return err
}

// SetLightPowerupPreset updates a light's power-on behavior preset.
func (b *Bridge) SetLightPowerupPreset(lightID string, preset hueclient.PowerupPreset) error {
	b.mu.RLock()
	client := b.client
	b.mu.RUnlock()

	if client == nil {
		return ErrAuthFailed
	}

	// Get light name BEFORE optimistic update to ensure we can retrieve it
	lightName := "Unknown"
	if light, ok := b.state.GetLight(lightID); ok {
		lightName = b.state.GetLightName(light)
	}

	// Optimistic update: apply to cache immediately
	b.state.SetLightPowerupPreset(lightID, preset)

	b.logRequest(fmt.Sprintf("%s: power-on preset changed to \"%s\"", lightName, string(preset)))
	_, err := client.UpdateLight(context.Background(), lightID, hueclient.UpdateLightJSONRequestBody{
		Powerup: &hueclient.Powerup{
			Preset: &preset,
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

	_, err := client.DeleteScene(context.Background(), sceneID)
	return err
}

// DeleteDevice removes a device from the bridge (factory reset).
func (b *Bridge) DeleteDevice(deviceID string) error {
	b.mu.RLock()
	client := b.client
	b.mu.RUnlock()

	if client == nil {
		return ErrAuthFailed
	}

	deviceName := "Unknown"
	if device, ok := b.state.GetDevice(deviceID); ok {
		deviceName = b.state.GetDeviceName(device)
	}

	b.logRequest(fmt.Sprintf("Deleting device: %s", deviceName))

	// Remove from state immediately (optimistic)
	b.state.DeleteDevice(deviceID)

	_, err := client.DeleteDevice(context.Background(), deviceID)
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

	action := hueclient.SmartSceneOptionalRecallActionActivate
	_, err := client.UpdateSmartScene(context.Background(), sceneID, hueclient.UpdateSmartSceneJSONRequestBody{
		Recall: &hueclient.SmartSceneOptionalRecall{
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

	action := hueclient.SmartSceneOptionalRecallActionDeactivate
	_, err := client.UpdateSmartScene(context.Background(), sceneID, hueclient.UpdateSmartSceneJSONRequestBody{
		Recall: &hueclient.SmartSceneOptionalRecall{
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

	_, err := client.DeleteSmartScene(context.Background(), sceneID)
	return err
}

// CreateSceneFromCurrentState creates a new scene for a room/zone using the current light states.
func (b *Bridge) CreateSceneFromCurrentState(groupID string, isZone bool, sceneName string) error {
	b.mu.RLock()
	client := b.client
	b.mu.RUnlock()

	if client == nil {
		return ErrAuthFailed
	}

	// Get the room/zone to find its lights
	var lights []hueclient.LightGet
	var groupType hueclient.ResourceIdentifierRtype
	if isZone {
		zone, ok := b.state.GetZone(groupID)
		if !ok {
			return fmt.Errorf("zone not found: %s", groupID)
		}
		lights = b.state.ZoneLights(zone)
		groupType = hueclient.ResourceIdentifierRtypeZone
	} else {
		room, ok := b.state.GetRoom(groupID)
		if !ok {
			return fmt.Errorf("room not found: %s", groupID)
		}
		lights = b.state.RoomLights(room)
		groupType = hueclient.ResourceIdentifierRtypeRoom
	}

	if len(lights) == 0 {
		return fmt.Errorf("no lights in group")
	}

	// Build actions from current light states
	var actions []hueclient.ActionPost
	lightType := hueclient.ResourceIdentifierRtypeLight
	for _, light := range lights {
		if light.Id == nil {
			continue
		}
		lightID := *light.Id

		action := hueclient.ActionPost{
			Target: hueclient.ResourceIdentifier{
				Rid:   &lightID,
				Rtype: &lightType,
			},
		}

		// Capture on/off state
		if light.On != nil && light.On.On != nil {
			action.Action.On = &hueclient.On{On: light.On.On}
		}

		// Capture brightness
		if light.Dimming != nil && light.Dimming.Brightness != nil {
			action.Action.Dimming = &hueclient.Dimming{Brightness: light.Dimming.Brightness}
		}

		// Capture color (XY)
		if light.Color != nil && light.Color.Xy != nil {
			action.Action.Color = &hueclient.Color{
				Xy: light.Color.Xy,
			}
		}

		// Capture color temperature (mirek)
		if light.ColorTemperature != nil && light.ColorTemperature.Mirek != nil {
			action.Action.ColorTemperature = &struct {
				Mirek *hueclient.Mirek `json:"mirek,omitempty"`
			}{
				Mirek: light.ColorTemperature.Mirek,
			}
		}

		actions = append(actions, action)
	}

	// Create the scene
	b.logRequest(fmt.Sprintf("Creating scene \"%s\" with %d lights", sceneName, len(actions)))

	_, err := client.CreateScene(context.Background(), hueclient.CreateSceneJSONRequestBody{
		Metadata: hueclient.SceneMetadata{
			Name: &sceneName,
		},
		Group: hueclient.ResourceIdentifier{
			Rid:   &groupID,
			Rtype: &groupType,
		},
		Actions: actions,
	})

	return err
}
