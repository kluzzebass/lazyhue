package hue

import (
	"context"
	"fmt"

	"github.com/kluzzebass/lazyhue/internal/hueclient"
)

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
	_, err := client.UpdateGroupedLight(context.Background(), toResourceId(groupedLightID), hueclient.UpdateGroupedLightJSONRequestBody{
		On: &struct {
			On *bool `json:"on,omitempty"`
		}{On: &newState},
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
	_, err := client.UpdateGroupedLight(context.Background(), toResourceId(groupedLightID), hueclient.UpdateGroupedLightJSONRequestBody{
		On: &struct {
			On *bool `json:"on,omitempty"`
		}{On: &on},
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
	_, err := client.UpdateGroupedLight(context.Background(), toResourceId(groupedLightID), hueclient.UpdateGroupedLightJSONRequestBody{
		Dimming: &struct {
			Brightness *float32 `json:"brightness,omitempty"`
		}{Brightness: &br},
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
	_, err := client.UpdateRoom(context.Background(), toResourceId(roomID), hueclient.UpdateRoomJSONRequestBody{
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
		zoneName = zone.Metadata.Name
	}
	b.logRequest(fmt.Sprintf("%s: renamed to \"%s\"", zoneName, newName))
	_, err := client.UpdateZone(context.Background(), toResourceId(zoneID), hueclient.UpdateZoneJSONRequestBody{
		Metadata: &struct {
			Archetype *hueclient.RoomArchetype `json:"archetype,omitempty"`
			Name      *string                  `json:"name,omitempty"`
		}{
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
	for i, id := range deviceIDs {
		children[i] = hueclient.ResourceIdentifier{
			Rid:   id,
			Rtype: hueclient.ResourceTypeDevice,
		}
	}

	b.logRequest(fmt.Sprintf("Room \"%s\": created", name))
	_, err := client.CreateRoom(context.Background(), hueclient.CreateRoomJSONRequestBody{
		Metadata: struct {
			Archetype hueclient.RoomArchetype `json:"archetype"`
			Name      string                  `json:"name"`
		}{
			Name:      name,
			Archetype: archetype,
		},
		Children: children,
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
	for i, id := range serviceIDs {
		children[i] = hueclient.ResourceIdentifier{
			Rid:   id,
			Rtype: hueclient.ResourceTypeLight,
		}
	}

	b.logRequest(fmt.Sprintf("Zone \"%s\": created", name))
	_, err := client.CreateZone(context.Background(), hueclient.CreateZoneJSONRequestBody{
		Metadata: struct {
			Archetype hueclient.RoomArchetype `json:"archetype"`
			Name      string                  `json:"name"`
		}{
			Name:      name,
			Archetype: archetype,
		},
		Children: children,
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
	_, err := client.DeleteRoom(context.Background(), toResourceId(roomID))
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
		zoneName = zone.Metadata.Name
	}
	b.logRequest(fmt.Sprintf("%s: deleted", zoneName))
	_, err := client.DeleteZone(context.Background(), toResourceId(zoneID))
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
	for i, id := range deviceIDs {
		children[i] = hueclient.ResourceIdentifier{
			Rid:   id,
			Rtype: hueclient.ResourceTypeDevice,
		}
	}

	_, err := client.UpdateRoom(context.Background(), toResourceId(roomID), hueclient.UpdateRoomJSONRequestBody{
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
	if hasOldRoom && oldRoom.Id != "" {
		oldRoomID = oldRoom.Id
	}

	// If already in the target room, nothing to do
	if oldRoomID == newRoomID {
		return nil
	}

	// Remove from old room if it was in one
	if hasOldRoom && len(oldRoom.Children) > 0 {
		var remainingDevices []string
		for _, child := range oldRoom.Children {
			if child.Rtype == hueclient.ResourceTypeDevice && child.Rid != deviceID {
				remainingDevices = append(remainingDevices, child.Rid)
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
		for _, child := range newRoom.Children {
			if child.Rtype == hueclient.ResourceTypeDevice {
				newDevices = append(newDevices, child.Rid)
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
		zoneName = zone.Metadata.Name
	}
	b.logRequest(fmt.Sprintf("%s: updated services (%d)", zoneName, len(serviceIDs)))

	// Build children list from service IDs
	children := make([]hueclient.ResourceIdentifier, len(serviceIDs))
	for i, id := range serviceIDs {
		children[i] = hueclient.ResourceIdentifier{
			Rid:   id,
			Rtype: hueclient.ResourceTypeLight,
		}
	}

	_, err := client.UpdateZone(context.Background(), toResourceId(zoneID), hueclient.UpdateZoneJSONRequestBody{
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
	for _, child := range zone.Children {
		if child.Rtype == hueclient.ResourceTypeLight {
			// Check if light is already in the zone
			if child.Rid == lightID {
				return nil // Already in zone, nothing to do
			}
			lightIDs = append(lightIDs, child.Rid)
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
	for _, child := range zone.Children {
		if child.Rtype == hueclient.ResourceTypeLight {
			if child.Rid == lightID {
				found = true
				continue // Skip this light
			}
			lightIDs = append(lightIDs, child.Rid)
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
	b.logRequest(fmt.Sprintf("%s: archetype changed", roomName))
	_, err := client.UpdateRoom(context.Background(), toResourceId(roomID), hueclient.UpdateRoomJSONRequestBody{
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
		zoneName = zone.Metadata.Name
	}
	b.logRequest(fmt.Sprintf("%s: archetype changed", zoneName))
	_, err := client.UpdateZone(context.Background(), toResourceId(zoneID), hueclient.UpdateZoneJSONRequestBody{
		Metadata: &struct {
			Archetype *hueclient.RoomArchetype `json:"archetype,omitempty"`
			Name      *string                  `json:"name,omitempty"`
		}{
			Archetype: &archetype,
		},
	})
	return err
}
