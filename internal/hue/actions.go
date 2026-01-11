package hue

import (
	"context"
	"fmt"

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

	lightName := "Unknown"
	if light, ok := b.state.GetLight(lightID); ok {
		lightName = b.state.GetLightName(light)
	}
	action := "on"
	if !on {
		action = "off"
	}
	b.logRequest(fmt.Sprintf("%s: %s", lightName, action))
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

	lightName := "Unknown"
	if light, ok := b.state.GetLight(lightID); ok {
		lightName = b.state.GetLightName(light)
	}
	b.logRequest(fmt.Sprintf("%s: brightness %.0f%%", lightName, brightness))
	br := float32(brightness)
	_, err := client.UpdateLight(context.Background(), lightID, hueclient.UpdateLightJSONRequestBody{
		Dimming: &hueclient.Dimming{Brightness: &br},
	})
	return err
}

// SetLightColor sets a light's color using CIE XY coordinates.
func (b *Bridge) SetLightColor(lightID string, x, y float64) error {
	b.mu.RLock()
	client := b.client
	b.mu.RUnlock()

	if client == nil {
		return ErrAuthFailed
	}

	// Optimistic update
	b.state.SetLightColor(lightID, x, y)

	lightName := "Unknown"
	if light, ok := b.state.GetLight(lightID); ok {
		lightName = b.state.GetLightName(light)
	}
	b.logRequest(fmt.Sprintf("%s: color", lightName))
	xf, yf := float32(x), float32(y)
	_, err := client.UpdateLight(context.Background(), lightID, hueclient.UpdateLightJSONRequestBody{
		Color: &hueclient.Color{
		Xy: &hueclient.GamutPosition{X: &xf, Y: &yf},
		},
	})
	return err
}

// SetLightColorTemperature sets a light's color temperature in mirek (153-500).
func (b *Bridge) SetLightColorTemperature(lightID string, mirek int) error {
	b.mu.RLock()
	client := b.client
	b.mu.RUnlock()

	if client == nil {
		return ErrAuthFailed
	}

	// Optimistic update
	b.state.SetLightColorTemperature(lightID, mirek)

	lightName := "Unknown"
	if light, ok := b.state.GetLight(lightID); ok {
		lightName = b.state.GetLightName(light)
	}
	b.logRequest(fmt.Sprintf("%s: color temp %d mirek", lightName, mirek))
	_, err := client.UpdateLight(context.Background(), lightID, hueclient.UpdateLightJSONRequestBody{
		ColorTemperature: &hueclient.ColorTemperature{Mirek: &mirek},
	})
	return err
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
	_, err := client.UpdateLight(context.Background(), lightID, hueclient.UpdateLightJSONRequestBody{
		Effects: &hueclient.Effects{Effect: &effect},
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
		zoneName = b.state.GetRoomName(zone)
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
		zoneName = b.state.GetRoomName(zone)
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

// UpdateZoneServices updates the services (lights) assigned to a zone.
func (b *Bridge) UpdateZoneServices(zoneID string, serviceIDs []string) error {
	b.mu.RLock()
	client := b.client
	b.mu.RUnlock()

	if client == nil {
		return ErrAuthFailed
	}

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
		zoneName = b.state.GetRoomName(zone)
		}
	b.logRequest(fmt.Sprintf("%s: renamed to \"%s\"", zoneName, zoneName))
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
