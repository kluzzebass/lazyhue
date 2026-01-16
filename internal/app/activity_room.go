package app

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/kluzzebass/lazyhue/internal/hue"
	"github.com/kluzzebass/lazyhue/internal/ui"
)

// DeviceEvent represents a device update event.
type DeviceEvent struct {
	baseEvent
	Name    string
	Details string
}

func (e *DeviceEvent) Parse(bridgeID, bridgeName, eventType string, data json.RawMessage, state *hue.BridgeState) (Event, error) {
	var updates []hue.ResourceUpdate
	if err := json.Unmarshal(data, &updates); err != nil {
		return nil, fmt.Errorf("parse device event: %w", err)
	}

	if len(updates) == 0 {
		return nil, fmt.Errorf("no updates in device event")
	}

	update := updates[0]

	e.baseEvent = baseEvent{
		bridgeID:     bridgeID,
		bridgeName:   bridgeName,
		resourceType: update.Type,
		resourceID:   update.ID,
		eventType:    eventType,
		timestamp:    time.Now(),
	}

	// Look up device name from state
	if state != nil {
		if device, ok := state.GetDevice(update.ID); ok {
			e.Name = state.GetDeviceName(device)
		}
	}

	if e.Name == "" {
		e.Name = update.ID[:8] + "…"
	}

	e.Details = eventType

	return e, nil
}

func (e *DeviceEvent) Render(styles *ui.Styles, width int) string {
	return renderEvent(styles, width, e.baseEvent, e.Name, e.Details, "", 0, false)
}

// RoomEvent represents a room update event.
type RoomEvent struct {
	baseEvent
	Name    string
	Details string
}

func (e *RoomEvent) Parse(bridgeID, bridgeName, eventType string, data json.RawMessage, state *hue.BridgeState) (Event, error) {
	var updates []hue.ResourceUpdate
	if err := json.Unmarshal(data, &updates); err != nil {
		return nil, fmt.Errorf("parse room event: %w", err)
	}

	if len(updates) == 0 {
		return nil, fmt.Errorf("no updates in room event")
	}

	update := updates[0]

	e.baseEvent = baseEvent{
		bridgeID:     bridgeID,
		bridgeName:   bridgeName,
		resourceType: update.Type,
		resourceID:   update.ID,
		eventType:    eventType,
		timestamp:    time.Now(),
	}

	// Look up room name from state
	if state != nil {
		if room, ok := state.GetRoom(update.ID); ok {
			if room.Metadata.Name != "" {
				e.Name = room.Metadata.Name
			}
		}
	}

	if e.Name == "" {
		e.Name = update.ID[:8] + "…"
	}

	// Set details based on event type
	switch eventType {
	case "add":
		e.Details = "created"
	case "update":
		e.Details = "updated"
	case "delete":
		e.Details = "deleted"
	default:
		e.Details = eventType
	}

	return e, nil
}

func (e *RoomEvent) Render(styles *ui.Styles, width int) string {
	return renderEvent(styles, width, e.baseEvent, e.Name, e.Details, "", 0, false)
}

// ZoneEvent represents a zone update event.
type ZoneEvent struct {
	baseEvent
	Name    string
	Details string
}

func (e *ZoneEvent) Parse(bridgeID, bridgeName, eventType string, data json.RawMessage, state *hue.BridgeState) (Event, error) {
	var updates []hue.ResourceUpdate
	if err := json.Unmarshal(data, &updates); err != nil {
		return nil, fmt.Errorf("parse zone event: %w", err)
	}

	if len(updates) == 0 {
		return nil, fmt.Errorf("no updates in zone event")
	}

	update := updates[0]

	e.baseEvent = baseEvent{
		bridgeID:     bridgeID,
		bridgeName:   bridgeName,
		resourceType: update.Type,
		resourceID:   update.ID,
		eventType:    eventType,
		timestamp:    time.Now(),
	}

	// Look up zone name from state
	if state != nil {
		if zone, ok := state.GetZone(update.ID); ok {
			if zone.Metadata.Name != "" {
				e.Name = zone.Metadata.Name
			}
		}
	}

	if e.Name == "" {
		e.Name = update.ID[:8] + "…"
	}

	// Set details based on event type
	switch eventType {
	case "add":
		e.Details = "created"
	case "update":
		e.Details = "updated"
	case "delete":
		e.Details = "deleted"
	default:
		e.Details = eventType
	}

	return e, nil
}

func (e *ZoneEvent) Render(styles *ui.Styles, width int) string {
	return renderEvent(styles, width, e.baseEvent, e.Name, e.Details, "", 0, false)
}

// BridgeHomeEvent represents a bridge_home update event.
type BridgeHomeEvent struct {
	baseEvent
	Details string
}

func (e *BridgeHomeEvent) Parse(bridgeID, bridgeName, eventType string, data json.RawMessage, state *hue.BridgeState) (Event, error) {
	var updates []hue.ResourceUpdate
	if err := json.Unmarshal(data, &updates); err != nil {
		return nil, fmt.Errorf("parse bridge_home event: %w", err)
	}

	if len(updates) == 0 {
		return nil, fmt.Errorf("no updates in bridge_home event")
	}

	update := updates[0]

	e.baseEvent = baseEvent{
		bridgeID:     bridgeID,
		bridgeName:   bridgeName,
		resourceType: update.Type,
		resourceID:   update.ID,
		eventType:    eventType,
		timestamp:    time.Now(),
	}

	// Set details based on event type
	switch eventType {
	case "add":
		e.Details = "added"
	case "update":
		e.Details = "updated"
	case "delete":
		e.Details = "deleted"
	default:
		e.Details = eventType
	}

	return e, nil
}

func (e *BridgeHomeEvent) Render(styles *ui.Styles, width int) string {
	return renderEvent(styles, width, e.baseEvent, "Bridge Home", e.Details, "", 0, false)
}
