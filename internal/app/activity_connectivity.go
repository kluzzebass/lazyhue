package app

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/kluzzebass/lazyhue/internal/hue"
	"github.com/kluzzebass/lazyhue/internal/ui"
)

// ZigbeeConnectivityEvent represents a Zigbee connectivity status change.
type ZigbeeConnectivityEvent struct {
	baseEvent
	Name    string
	Details string
}

func (e *ZigbeeConnectivityEvent) Parse(bridgeID, bridgeName, eventType string, data json.RawMessage, state *hue.BridgeState) (Event, error) {
	var updates []hue.ResourceUpdate
	if err := json.Unmarshal(data, &updates); err != nil {
		return nil, fmt.Errorf("parse zigbee_connectivity event: %w", err)
	}
	if len(updates) == 0 {
		return nil, fmt.Errorf("no updates in zigbee_connectivity event")
	}
	update := updates[0]
	e.baseEvent = baseEvent{
		bridgeID:     bridgeID,
		bridgeName:   bridgeName,
		resourceType: "zigbee_connectivity",
		resourceID:   update.ID,
		eventType:    eventType,
		timestamp:    time.Now(),
	}
	if update.ConnectivityStatus != nil {
		e.Details = strings.ReplaceAll(update.ConnectivityStatus.Status, "_", " ")
	}
	if state != nil && update.Owner != nil {
		if device, ok := state.GetDevice(update.Owner.Rid); ok {
			e.Name = device.DeviceName("")
		}
	}
	return e, nil
}

func (e *ZigbeeConnectivityEvent) Render(styles *ui.Styles, width int) string {
	return renderEvent(styles, width, e.baseEvent, e.Name, e.Details, "", 0, false)
}

// ZgpConnectivityEvent represents a ZGP (Zigbee Green Power) connectivity status change.
type ZgpConnectivityEvent struct {
	baseEvent
	Name    string
	Details string
}

func (e *ZgpConnectivityEvent) Parse(bridgeID, bridgeName, eventType string, data json.RawMessage, state *hue.BridgeState) (Event, error) {
	var updates []hue.ResourceUpdate
	if err := json.Unmarshal(data, &updates); err != nil {
		return nil, fmt.Errorf("parse zgp_connectivity event: %w", err)
	}
	if len(updates) == 0 {
		return nil, fmt.Errorf("no updates in zgp_connectivity event")
	}
	update := updates[0]
	e.baseEvent = baseEvent{
		bridgeID:     bridgeID,
		bridgeName:   bridgeName,
		resourceType: "zgp_connectivity",
		resourceID:   update.ID,
		eventType:    eventType,
		timestamp:    time.Now(),
	}
	if update.ConnectivityStatus != nil {
		e.Details = strings.ReplaceAll(update.ConnectivityStatus.Status, "_", " ")
	}
	if state != nil && update.Owner != nil {
		if device, ok := state.GetDevice(update.Owner.Rid); ok {
			e.Name = device.DeviceName("")
		}
	}
	return e, nil
}

func (e *ZgpConnectivityEvent) Render(styles *ui.Styles, width int) string {
	return renderEvent(styles, width, e.baseEvent, e.Name, e.Details, "", 0, false)
}

// WifiConnectivityEvent represents a WiFi connectivity status change.
type WifiConnectivityEvent struct {
	baseEvent
	Name    string
	Details string
}

func (e *WifiConnectivityEvent) Parse(bridgeID, bridgeName, eventType string, data json.RawMessage, state *hue.BridgeState) (Event, error) {
	var updates []hue.ResourceUpdate
	if err := json.Unmarshal(data, &updates); err != nil {
		return nil, fmt.Errorf("parse wifi_connectivity event: %w", err)
	}
	if len(updates) == 0 {
		return nil, fmt.Errorf("no updates in wifi_connectivity event")
	}
	update := updates[0]
	e.baseEvent = baseEvent{
		bridgeID:     bridgeID,
		bridgeName:   bridgeName,
		resourceType: "wifi_connectivity",
		resourceID:   update.ID,
		eventType:    eventType,
		timestamp:    time.Now(),
	}
	if update.ConnectivityStatus != nil {
		e.Details = strings.ReplaceAll(update.ConnectivityStatus.Status, "_", " ")
	}
	if state != nil && update.Owner != nil {
		if device, ok := state.GetDevice(update.Owner.Rid); ok {
			e.Name = device.DeviceName("")
		}
	}
	return e, nil
}

func (e *WifiConnectivityEvent) Render(styles *ui.Styles, width int) string {
	return renderEvent(styles, width, e.baseEvent, e.Name, e.Details, "", 0, false)
}

// DeviceSoftwareUpdateEvent represents a device software update event.
type DeviceSoftwareUpdateEvent struct {
	baseEvent
	Name    string
	Details string
}

func (e *DeviceSoftwareUpdateEvent) Parse(bridgeID, bridgeName, eventType string, data json.RawMessage, state *hue.BridgeState) (Event, error) {
	var updates []hue.ResourceUpdate
	if err := json.Unmarshal(data, &updates); err != nil {
		return nil, fmt.Errorf("parse device_software_update event: %w", err)
	}
	if len(updates) == 0 {
		return nil, fmt.Errorf("no updates in device_software_update event")
	}
	update := updates[0]
	e.baseEvent = baseEvent{
		bridgeID:     bridgeID,
		bridgeName:   bridgeName,
		resourceType: "device_software_update",
		resourceID:   update.ID,
		eventType:    eventType,
		timestamp:    time.Now(),
	}
	if update.SoftwareUpdate != nil {
		e.Details = strings.ReplaceAll(update.SoftwareUpdate.State, "_", " ")
	}
	if state != nil && update.Owner != nil {
		if device, ok := state.GetDevice(update.Owner.Rid); ok {
			e.Name = device.DeviceName("")
		}
	}
	return e, nil
}

func (e *DeviceSoftwareUpdateEvent) Render(styles *ui.Styles, width int) string {
	return renderEvent(styles, width, e.baseEvent, e.Name, e.Details, "", 0, false)
}
