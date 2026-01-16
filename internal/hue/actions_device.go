package hue

import (
	"context"
	"fmt"

	"github.com/kluzzebass/lazyhue/internal/hueclient"
)

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
	if motion.Owner.Rid != "" {
		if device, ok := b.state.GetDevice(motion.Owner.Rid); ok {
			deviceName = b.state.GetDeviceName(device)
		}
	}
	action := "sensor enabled"
	if !newState {
		action = "sensor disabled"
	}
	b.logRequest(fmt.Sprintf("%s: %s", deviceName, action))
	_, err := client.UpdateMotionWithResponse(context.Background(), toResourceId(motionID), hueclient.UpdateMotionJSONRequestBody{
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
		if motion.Owner.Rid != "" {
			if device, ok := b.state.GetDevice(motion.Owner.Rid); ok {
				deviceName = b.state.GetDeviceName(device)
			}
		}
	}
	action := "sensor enabled"
	if !enabled {
		action = "sensor disabled"
	}
	b.logRequest(fmt.Sprintf("%s: %s", deviceName, action))
	_, err := client.UpdateMotionWithResponse(context.Background(), toResourceId(motionID), hueclient.UpdateMotionJSONRequestBody{
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
		if motion.Owner.Rid != "" {
			if device, ok := b.state.GetDevice(motion.Owner.Rid); ok {
				deviceName = b.state.GetDeviceName(device)
			}
		}
	}
	b.logRequest(fmt.Sprintf("%s: sensitivity %d", deviceName, sensitivity))
	_, err := client.UpdateMotionWithResponse(context.Background(), toResourceId(motionID), hueclient.UpdateMotionJSONRequestBody{
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
		if temp.Owner.Rid != "" {
			if device, ok := b.state.GetDevice(temp.Owner.Rid); ok {
				deviceName = b.state.GetDeviceName(device)
			}
		}
	}
	action := "temperature sensor enabled"
	if !enabled {
		action = "temperature sensor disabled"
	}
	b.logRequest(fmt.Sprintf("%s: %s", deviceName, action))
	_, err := client.UpdateTemperatureWithResponse(context.Background(), toResourceId(temperatureID), hueclient.UpdateTemperatureJSONRequestBody{
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
		if ll.Owner.Rid != "" {
			if device, ok := b.state.GetDevice(ll.Owner.Rid); ok {
				deviceName = b.state.GetDeviceName(device)
			}
		}
	}
	action := "light level sensor enabled"
	if !enabled {
		action = "light level sensor disabled"
	}
	b.logRequest(fmt.Sprintf("%s: %s", deviceName, action))
	_, err := client.UpdateLightLevel(context.Background(), toResourceId(lightLevelID), hueclient.UpdateLightLevelJSONRequestBody{
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
	_, err := client.UpdateDevice(context.Background(), toResourceId(deviceID), hueclient.UpdateDeviceJSONRequestBody{
		Metadata: &struct {
			Archetype *hueclient.DeviceArchetype `json:"archetype,omitempty"`
			Name      *string                    `json:"name,omitempty"`
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

	action := hueclient.UpdateDeviceJSONBodyIdentifyAction("identify")
	_, err := client.UpdateDevice(context.Background(), toResourceId(deviceID), hueclient.UpdateDeviceJSONRequestBody{
		Identify: &struct {
			Action   hueclient.UpdateDeviceJSONBodyIdentifyAction `json:"action"`
			Duration *int                                         `json:"duration,omitempty"`
		}{
			Action: action,
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
	if device, ok := b.state.GetDevice(deviceID); ok && device.Metadata.Name != "" {
		oldName = device.Metadata.Name
	}
	b.logRequest(fmt.Sprintf("%s: renamed to \"%s\"", oldName, name))
	_, err := client.UpdateDevice(context.Background(), toResourceId(deviceID), hueclient.UpdateDeviceJSONRequestBody{
		Metadata: &struct {
			Archetype *hueclient.DeviceArchetype `json:"archetype,omitempty"`
			Name      *string                    `json:"name,omitempty"`
		}{
			Name: &name,
		},
	})
	return err
}

// SetDeviceArchetype updates a device's archetype.
func (b *Bridge) SetDeviceArchetype(deviceID string, archetype hueclient.DeviceArchetype) error {
	b.mu.RLock()
	client := b.client
	b.mu.RUnlock()

	if client == nil {
		return ErrAuthFailed
	}

	deviceName := "Unknown"
	if device, ok := b.state.GetDevice(deviceID); ok && device.Metadata.Name != "" {
		deviceName = device.Metadata.Name
	}

	// Optimistic update: apply to cache immediately
	b.state.SetDeviceArchetype(deviceID, archetype)

	b.logRequest(fmt.Sprintf("%s: archetype changed to \"%s\"", deviceName, string(archetype)))
	_, err := client.UpdateDevice(context.Background(), toResourceId(deviceID), hueclient.UpdateDeviceJSONRequestBody{
		Metadata: &struct {
			Archetype *hueclient.DeviceArchetype `json:"archetype,omitempty"`
			Name      *string                    `json:"name,omitempty"`
		}{
			Archetype: &archetype,
		},
	})
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

	_, err := client.DeleteDevice(context.Background(), toResourceId(deviceID))
	return err
}
