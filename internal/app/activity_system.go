package app

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/kluzzebass/lazyhue/internal/hue"
	"github.com/kluzzebass/lazyhue/internal/ui"
)

// HomekitEvent represents a HomeKit status event.
type HomekitEvent struct {
	baseEvent
	Status string // "paired", "pairing", "unpaired"
}

func (e *HomekitEvent) Parse(bridgeID, bridgeName, eventType string, data json.RawMessage, state *hue.BridgeState) (Event, error) {
	var updates []hue.ResourceUpdate
	if err := json.Unmarshal(data, &updates); err != nil {
		return nil, fmt.Errorf("parse homekit event: %w", err)
	}
	if len(updates) == 0 {
		return nil, fmt.Errorf("no updates in homekit event")
	}
	update := updates[0]
	e.baseEvent = baseEvent{
		bridgeID:     bridgeID,
		bridgeName:   bridgeName,
		resourceType: "homekit",
		resourceID:   update.ID,
		eventType:    eventType,
		timestamp:    time.Now(),
	}

	// Get status from the update - homekit status comes as a string in the Status field
	if len(update.Status) > 0 {
		var status string
		if err := json.Unmarshal(update.Status, &status); err == nil {
			e.Status = status
		}
	}

	return e, nil
}

func (e *HomekitEvent) Render(styles *ui.Styles, width int) string {
	details := e.Status
	if details == "" {
		details = e.eventType
	}
	return renderEvent(styles, width, e.baseEvent, "HomeKit", details, "", 0, false)
}

// MatterEvent represents a Matter configuration event.
type MatterEvent struct {
	baseEvent
	HasQrCode       bool
	MaxFabrics      int
	SoftwareVersion string
}

func (e *MatterEvent) Parse(bridgeID, bridgeName, eventType string, data json.RawMessage, state *hue.BridgeState) (Event, error) {
	var updates []hue.ResourceUpdate
	if err := json.Unmarshal(data, &updates); err != nil {
		return nil, fmt.Errorf("parse matter event: %w", err)
	}
	if len(updates) == 0 {
		return nil, fmt.Errorf("no updates in matter event")
	}
	update := updates[0]
	e.baseEvent = baseEvent{
		bridgeID:     bridgeID,
		bridgeName:   bridgeName,
		resourceType: "matter",
		resourceID:   update.ID,
		eventType:    eventType,
		timestamp:    time.Now(),
	}

	// Get matter data from the update
	if update.Matter != nil {
		e.HasQrCode = update.Matter.HasQrCode
		e.MaxFabrics = update.Matter.MaxFabrics
		e.SoftwareVersion = update.Matter.SoftwareVersionString
	}

	return e, nil
}

func (e *MatterEvent) Render(styles *ui.Styles, width int) string {
	details := e.eventType
	if e.SoftwareVersion != "" {
		details = fmt.Sprintf("v%s", e.SoftwareVersion)
	}
	return renderEvent(styles, width, e.baseEvent, "Matter", details, "", 0, false)
}

// MatterFabricEvent represents a Matter fabric event.
type MatterFabricEvent struct {
	baseEvent
	Status       string // "paired", "timedout"
	CreationTime string
	Label        string
	VendorID     int
}

func (e *MatterFabricEvent) Parse(bridgeID, bridgeName, eventType string, data json.RawMessage, state *hue.BridgeState) (Event, error) {
	var updates []hue.ResourceUpdate
	if err := json.Unmarshal(data, &updates); err != nil {
		return nil, fmt.Errorf("parse matter_fabric event: %w", err)
	}
	if len(updates) == 0 {
		return nil, fmt.Errorf("no updates in matter_fabric event")
	}
	update := updates[0]
	e.baseEvent = baseEvent{
		bridgeID:     bridgeID,
		bridgeName:   bridgeName,
		resourceType: "matter_fabric",
		resourceID:   update.ID,
		eventType:    eventType,
		timestamp:    time.Now(),
	}

	// Get matter fabric data from the update
	if update.MatterFabric != nil {
		e.Status = update.MatterFabric.Status
		e.CreationTime = update.MatterFabric.CreationTime
		if update.MatterFabric.FabricData != nil {
			e.Label = update.MatterFabric.FabricData.Label
			e.VendorID = update.MatterFabric.FabricData.VendorID
		}
	}

	return e, nil
}

func (e *MatterFabricEvent) Render(styles *ui.Styles, width int) string {
	name := "Matter Fabric"
	if e.Label != "" {
		name = e.Label
	}
	details := e.Status
	if details == "" {
		details = e.eventType
	}
	return renderEvent(styles, width, e.baseEvent, name, details, "", 0, false)
}

// GeolocationEvent represents a geolocation configuration event.
type GeolocationEvent struct {
	baseEvent
	IsConfigured bool
	DayType      string
	SunsetTime   string
}

func (e *GeolocationEvent) Parse(bridgeID, bridgeName, eventType string, data json.RawMessage, state *hue.BridgeState) (Event, error) {
	var updates []hue.ResourceUpdate
	if err := json.Unmarshal(data, &updates); err != nil {
		return nil, fmt.Errorf("parse geolocation event: %w", err)
	}
	if len(updates) == 0 {
		return nil, fmt.Errorf("no updates in geolocation event")
	}
	update := updates[0]
	e.baseEvent = baseEvent{
		bridgeID:     bridgeID,
		bridgeName:   bridgeName,
		resourceType: "geolocation",
		resourceID:   update.ID,
		eventType:    eventType,
		timestamp:    time.Now(),
	}

	// Get geolocation data from the update
	if update.IsConfigured != nil {
		e.IsConfigured = *update.IsConfigured
	}
	if update.SunToday != nil {
		e.DayType = update.SunToday.DayType
		e.SunsetTime = update.SunToday.SunsetTime
	}

	return e, nil
}

func (e *GeolocationEvent) Render(styles *ui.Styles, width int) string {
	var details string
	if e.SunsetTime != "" {
		details = fmt.Sprintf("sunset %s", e.SunsetTime)
	} else if e.IsConfigured {
		details = "configured"
	} else {
		details = "not configured"
	}
	return renderEvent(styles, width, e.baseEvent, "Geolocation", details, "", 0, false)
}

// ZigbeeDiscoveryEvent represents a Zigbee device discovery event.
type ZigbeeDiscoveryEvent struct {
	baseEvent
	Status string // "active", "ready"
}

func (e *ZigbeeDiscoveryEvent) Parse(bridgeID, bridgeName, eventType string, data json.RawMessage, state *hue.BridgeState) (Event, error) {
	var updates []hue.ResourceUpdate
	if err := json.Unmarshal(data, &updates); err != nil {
		return nil, fmt.Errorf("parse zigbee_device_discovery event: %w", err)
	}
	if len(updates) == 0 {
		return nil, fmt.Errorf("no updates in zigbee_device_discovery event")
	}
	update := updates[0]
	e.baseEvent = baseEvent{
		bridgeID:     bridgeID,
		bridgeName:   bridgeName,
		resourceType: "zigbee_device_discovery",
		resourceID:   update.ID,
		eventType:    eventType,
		timestamp:    time.Now(),
	}

	// Get status from the update - discovery status comes as a string in the Status field
	if len(update.Status) > 0 {
		var status string
		if err := json.Unmarshal(update.Status, &status); err == nil {
			e.Status = status
		}
	}

	return e, nil
}

func (e *ZigbeeDiscoveryEvent) Render(styles *ui.Styles, width int) string {
	details := e.Status
	if details == "" {
		details = e.eventType
	} else if details == "active" {
		details = "searching..."
	} else if details == "ready" {
		details = "complete"
	}
	return renderEvent(styles, width, e.baseEvent, "Zigbee Discovery", details, "", 0, false)
}
