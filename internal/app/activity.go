package app

import (
	"encoding/json"
	"fmt"
	"image/color"
	"time"

	"github.com/charmbracelet/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/kluzzebass/lazyhue/internal/hue"
	"github.com/kluzzebass/lazyhue/internal/ui"
)

// Activity represents any loggable activity (events, requests, errors, etc.)
type Activity interface {
	// Render returns the formatted string representation of this activity.
	// width is the available width for rendering (for truncation).
	Render(styles *ui.Styles, width int) string
	// Time returns when this activity occurred.
	Time() time.Time
}

// Event represents a bridge event that can be parsed from raw event data.
type Event interface {
	Activity
	// Parse parses the raw event data and returns an Event instance.
	// bridgeID is the ID of the bridge that sent the event.
	// bridgeName is the name of the bridge that sent the event.
	// eventType is the type of event ("update", "add", "delete", etc.).
	// data is the raw JSON data for this event.
	// state is the bridge state for looking up resource names and details.
	Parse(bridgeID, bridgeName, eventType string, data json.RawMessage, state *hue.BridgeState) (Event, error)
	// ResourceType returns the type of resource this event is for (e.g., "light", "scene").
	ResourceType() string
	// ResourceID returns the ID of the resource this event is for.
	ResourceID() string
}

// baseEvent contains common fields for all events.
type baseEvent struct {
	bridgeID     string
	bridgeName   string
	resourceType string
	resourceID   string
	eventType    string
	timestamp    time.Time
}

func (e *baseEvent) ResourceType() string {
	return e.resourceType
}

func (e *baseEvent) ResourceID() string {
	return e.resourceID
}

func (e *baseEvent) Time() time.Time {
	return e.timestamp
}

func (e *baseEvent) SetTimestamp(t time.Time) {
	e.timestamp = t
}

// SimpleDeviceEvent is a generic event for device-owned resources that just need name + event type.
type SimpleDeviceEvent struct {
	baseEvent
	Name    string
	Details string
}

func (e *SimpleDeviceEvent) Parse(bridgeID, bridgeName, eventType string, data json.RawMessage, state *hue.BridgeState) (Event, error) {
	// This should not be called directly - use ParseWithType instead
	return nil, fmt.Errorf("SimpleDeviceEvent.Parse should not be called directly")
}

func (e *SimpleDeviceEvent) ParseWithType(bridgeID, bridgeName, resourceType, eventType string, data json.RawMessage, state *hue.BridgeState) (Event, error) {
	var updates []hue.ResourceUpdate
	if err := json.Unmarshal(data, &updates); err != nil {
		return nil, fmt.Errorf("parse %s event: %w", resourceType, err)
	}
	if len(updates) == 0 {
		return nil, fmt.Errorf("no updates in %s event", resourceType)
	}
	update := updates[0]
	e.baseEvent = baseEvent{
		bridgeID:     bridgeID,
		bridgeName:   bridgeName,
		resourceType: resourceType,
		resourceID:   update.ID,
		eventType:    eventType,
		timestamp:    time.Now(),
	}
	e.Details = eventType
	if state != nil && update.Owner != nil {
		if device, ok := state.GetDevice(update.Owner.Rid); ok {
			e.Name = device.DeviceName("")
		}
	}
	return e, nil
}

func (e *SimpleDeviceEvent) Render(styles *ui.Styles, width int) string {
	return renderEvent(styles, width, e.baseEvent, e.Name, e.Details, "", 0, false)
}

// SimpleNamedEvent is a generic event for resources with metadata name.
type SimpleNamedEvent struct {
	baseEvent
	Name    string
	Details string
}

func (e *SimpleNamedEvent) Parse(bridgeID, bridgeName, eventType string, data json.RawMessage, state *hue.BridgeState) (Event, error) {
	// This should not be called directly - use ParseWithType instead
	return nil, fmt.Errorf("SimpleNamedEvent.Parse should not be called directly")
}

func (e *SimpleNamedEvent) ParseWithType(bridgeID, bridgeName, resourceType, eventType string, data json.RawMessage, state *hue.BridgeState) (Event, error) {
	var updates []hue.ResourceUpdate
	if err := json.Unmarshal(data, &updates); err != nil {
		return nil, fmt.Errorf("parse %s event: %w", resourceType, err)
	}
	if len(updates) == 0 {
		return nil, fmt.Errorf("no updates in %s event", resourceType)
	}
	update := updates[0]
	e.baseEvent = baseEvent{
		bridgeID:     bridgeID,
		bridgeName:   bridgeName,
		resourceType: resourceType,
		resourceID:   update.ID,
		eventType:    eventType,
		timestamp:    time.Now(),
	}
	e.Details = eventType
	if update.Metadata != nil && update.Metadata.Name != nil {
		e.Name = *update.Metadata.Name
	}
	return e, nil
}

func (e *SimpleNamedEvent) Render(styles *ui.Styles, width int) string {
	return renderEvent(styles, width, e.baseEvent, e.Name, e.Details, "", 0, false)
}

// SimpleEvent is a generic event for resources without specific data.
type SimpleEvent struct {
	baseEvent
	Details string
}

func (e *SimpleEvent) Parse(bridgeID, bridgeName, eventType string, data json.RawMessage, state *hue.BridgeState) (Event, error) {
	// This should not be called directly - use ParseWithType instead
	return nil, fmt.Errorf("SimpleEvent.Parse should not be called directly")
}

func (e *SimpleEvent) ParseWithType(bridgeID, bridgeName, resourceType, eventType string, data json.RawMessage, state *hue.BridgeState) (Event, error) {
	var updates []hue.ResourceUpdate
	if err := json.Unmarshal(data, &updates); err != nil {
		return nil, fmt.Errorf("parse %s event: %w", resourceType, err)
	}
	if len(updates) == 0 {
		return nil, fmt.Errorf("no updates in %s event", resourceType)
	}
	update := updates[0]
	e.baseEvent = baseEvent{
		bridgeID:     bridgeID,
		bridgeName:   bridgeName,
		resourceType: resourceType,
		resourceID:   update.ID,
		eventType:    eventType,
		timestamp:    time.Now(),
	}
	e.Details = eventType
	return e, nil
}

func (e *SimpleEvent) Render(styles *ui.Styles, width int) string {
	return renderEvent(styles, width, e.baseEvent, "", e.Details, "", 0, false)
}

// UnhandledEvent represents an event type that we don't have a specific handler for.
type UnhandledEvent struct {
	baseEvent
	RawData json.RawMessage
}

func (e *UnhandledEvent) Parse(bridgeID, bridgeName, eventType string, data json.RawMessage, state *hue.BridgeState) (Event, error) {
	var updates []hue.ResourceUpdate
	if err := json.Unmarshal(data, &updates); err != nil {
		return nil, fmt.Errorf("parse unhandled event: %w", err)
	}

	if len(updates) == 0 {
		return nil, fmt.Errorf("no updates in unhandled event")
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
	e.RawData = data

	return e, nil
}

func (e *UnhandledEvent) Render(styles *ui.Styles, width int) string {
	return renderEvent(styles, width, e.baseEvent, "", fmt.Sprintf("unhandled %s event", e.resourceType), "", 0, false)
}

// RequestActivity represents a request to the bridge (not an event).
type RequestActivity struct {
	timestamp  time.Time
	bridgeID   string
	bridgeName string
	message    string
}

func (a *RequestActivity) Time() time.Time {
	return a.timestamp
}

func (a *RequestActivity) Render(styles *ui.Styles, width int) string {
	timeStr := a.timestamp.Format("15:04:05")
	typeStyle := lipgloss.NewStyle().Foreground(styles.Theme.Secondary)
	typeIndicator := typeStyle.Render("→")
	bridgeStr := ""
	if a.bridgeName != "" {
		bridgeStyle := lipgloss.NewStyle().Foreground(styles.Theme.EntityBridge)
		bridgeStr = styles.Dimmed.Render("[") + bridgeStyle.Render(a.bridgeName) + styles.Dimmed.Render("]") + " "
	}
	line := fmt.Sprintf("%s %s %s%s", styles.Dimmed.Render(timeStr), typeIndicator, bridgeStr, a.message)

	// Truncate if width is provided and line exceeds it
	if width > 0 && lipgloss.Width(line) > width {
		line = ansi.Truncate(line, width, "…")
	}
	return line
}

// ErrorActivity represents an error activity.
type ErrorActivity struct {
	timestamp  time.Time
	bridgeID   string
	bridgeName string
	message    string
}

func (a *ErrorActivity) Time() time.Time {
	return a.timestamp
}

func (a *ErrorActivity) Render(styles *ui.Styles, width int) string {
	timeStr := a.timestamp.Format("15:04:05")
	typeStyle := lipgloss.NewStyle().Foreground(styles.Theme.Error)
	typeIndicator := typeStyle.Render("!")
	bridgeStr := ""
	if a.bridgeName != "" {
		bridgeStyle := lipgloss.NewStyle().Foreground(styles.Theme.EntityBridge)
		bridgeStr = styles.Dimmed.Render("[") + bridgeStyle.Render(a.bridgeName) + styles.Dimmed.Render("]") + " "
	}
	line := fmt.Sprintf("%s %s %s%s", styles.Dimmed.Render(timeStr), typeIndicator, bridgeStr, a.message)

	// Truncate if width is provided and line exceeds it
	if width > 0 && lipgloss.Width(line) > width {
		line = ansi.Truncate(line, width, "…")
	}
	return line
}

// getResourceTypeColor returns the appropriate entity color for a resource type.
func getResourceTypeColor(styles *ui.Styles, resourceType string) color.Color {
	switch resourceType {
	case "light":
		return styles.Theme.EntityLight
	case "grouped_light", "grouped_light_level", "grouped_motion":
		return styles.Theme.EntityRoom // grouped resources are room-level
	case "room":
		return styles.Theme.EntityRoom
	case "zone":
		return styles.Theme.EntityZone
	case "scene":
		return styles.Theme.EntityScene
	case "smart_scene":
		return styles.Theme.EntityScene // same color as regular scenes
	case "device", "device_power", "device_software_update":
		return styles.Theme.EntityDevice
	case "zigbee_connectivity", "zgp_connectivity", "wifi_connectivity":
		return styles.Theme.EntityDevice // connectivity is device-related
	case "button", "relative_rotary", "contact", "tamper":
		return styles.Theme.EntityDevice // input sensors are device-related
	case "motion", "light_level", "temperature":
		return styles.Theme.EntityDevice // environmental sensors are device-related
	case "bridge", "bridge_home":
		return styles.Theme.EntityBridge
	case "entertainment", "entertainment_configuration":
		return styles.Theme.EntityEntertainment
	default:
		return styles.Theme.TextMuted
	}
}

// renderEvent is a helper function to render events with consistent formatting.
func renderEvent(styles *ui.Styles, width int, base baseEvent, name, details, indicatorColor string, brightness float64, isOn bool) string {
	timeStr := base.timestamp.Format("15:04:05.000000") // Microsecond precision to debug duplicates
	typeStyle := lipgloss.NewStyle().Foreground(styles.Theme.Success)
	typeIndicator := typeStyle.Render("e")

	var indicator string
	if (base.resourceType == "light" || base.resourceType == "grouped_light") && isOn {
		indicatorChar := brightnessIndicatorLog(brightness)
		if indicatorColor != "" {
			indicatorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(indicatorColor))
			indicator = indicatorStyle.Render(indicatorChar) + " "
		} else {
			// Fallback to entity color if no specific color available
			fallbackColor := getResourceTypeColor(styles, base.resourceType)
			indicator = lipgloss.NewStyle().Foreground(fallbackColor).Render(indicatorChar) + " "
		}
	} else if (base.resourceType == "light" || base.resourceType == "grouped_light") && !isOn {
		indicator = styles.Dimmed.Render("○") + " "
	}

	// Bridge name with entity color, dimmed brackets
	bridgeStr := ""
	if base.bridgeName != "" {
		bridgeStyle := lipgloss.NewStyle().Foreground(styles.Theme.EntityBridge)
		bridgeStr = styles.Dimmed.Render("[") + bridgeStyle.Render(base.bridgeName) + styles.Dimmed.Render("]") + " "
	}

	// Entity name with entity color (the actual name like "Office Shelf Corner")
	entityColor := getResourceTypeColor(styles, base.resourceType)
	entityStyle := lipgloss.NewStyle().Foreground(entityColor)

	var line string
	if name != "" {
		if details != "" {
			line = fmt.Sprintf("%s %s %s%s%s %s → %s",
				styles.Dimmed.Render(timeStr),
				typeIndicator,
				bridgeStr,
				indicator,
				base.resourceType,
				entityStyle.Render(name),
				details)
		} else {
			line = fmt.Sprintf("%s %s %s%s%s %s",
				styles.Dimmed.Render(timeStr),
				typeIndicator,
				bridgeStr,
				indicator,
				base.resourceType,
				entityStyle.Render(name))
		}
	} else {
		if details != "" {
			line = fmt.Sprintf("%s %s %s%s%s → %s",
				styles.Dimmed.Render(timeStr),
				typeIndicator,
				bridgeStr,
				indicator,
				base.resourceType,
				details)
		} else {
			line = fmt.Sprintf("%s %s %s%s%s update",
				styles.Dimmed.Render(timeStr),
				typeIndicator,
				bridgeStr,
				indicator,
				base.resourceType)
		}
	}

	// Truncate if width is provided and line exceeds it
	if width > 0 && lipgloss.Width(line) > width {
		line = ansi.Truncate(line, width, "…")
	}

	return line
}

// brightnessIndicatorLog returns a character representing the brightness level.
func brightnessIndicatorLog(brightness float64) string {
	switch {
	case brightness <= 0:
		return "○" // off/empty
	case brightness < 37.5:
		return "◔" // quarter (1-37%)
	case brightness < 62.5:
		return "◑" // half (38-62%)
	case brightness < 87.5:
		return "◕" // three-quarters (63-87%)
	default:
		return "●" // full (88-100%)
	}
}
