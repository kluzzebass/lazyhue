package app

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/kluzzebass/lazyhue/internal/hue"
	"github.com/kluzzebass/lazyhue/internal/ui"
)

// SceneEvent represents a scene resource event.
type SceneEvent struct {
	baseEvent
	Name    string
	Details string
}

func (e *SceneEvent) Parse(bridgeID, bridgeName, eventType string, data json.RawMessage, state *hue.BridgeState) (Event, error) {
	var updates []hue.ResourceUpdate
	if err := json.Unmarshal(data, &updates); err != nil {
		return nil, fmt.Errorf("parse scene event: %w", err)
	}

	if len(updates) == 0 {
		return nil, fmt.Errorf("no updates in scene event")
	}

	update := updates[0]
	e.baseEvent = baseEvent{
		bridgeID:     bridgeID,
		bridgeName:   bridgeName,
		resourceType: "scene",
		resourceID:   update.ID,
		eventType:    eventType,
		timestamp:    time.Now(),
	}

	if state != nil {
		if scene, ok := state.GetScene(update.ID); ok {
			e.Name = scene.SceneName("")
			if scene.Status.Active != nil && *scene.Status.Active == "active" {
				e.Details = "activated"
			} else {
				e.Details = "deactivated"
			}
		} else {
			// Scene not in cache - show truncated ID
			e.Name = update.ID[:8] + "…"
		}
	}

	return e, nil
}

func (e *SceneEvent) Render(styles *ui.Styles, width int) string {
	return renderEvent(styles, width, e.baseEvent, e.Name, e.Details, "", 0, false)
}

// SmartSceneEvent represents a smart scene event.
type SmartSceneEvent struct {
	baseEvent
	Name    string
	Details string
}

func (e *SmartSceneEvent) Parse(bridgeID, bridgeName, eventType string, data json.RawMessage, state *hue.BridgeState) (Event, error) {
	var updates []hue.ResourceUpdate
	if err := json.Unmarshal(data, &updates); err != nil {
		return nil, fmt.Errorf("parse smart_scene event: %w", err)
	}
	if len(updates) == 0 {
		return nil, fmt.Errorf("no updates in smart_scene event")
	}
	update := updates[0]
	e.baseEvent = baseEvent{
		bridgeID:     bridgeID,
		bridgeName:   bridgeName,
		resourceType: "smart_scene",
		resourceID:   update.ID,
		eventType:    eventType,
		timestamp:    time.Now(),
	}
	if len(update.State) > 0 {
		// Try to parse state as an object with "active" field
		var stateObj struct {
			Active string `json:"active"`
		}
		if err := json.Unmarshal(update.State, &stateObj); err == nil && stateObj.Active != "" {
			e.Details = stateObj.Active
		}
	}
	if update.Metadata != nil && update.Metadata.Name != nil {
		e.Name = *update.Metadata.Name
	} else if state != nil {
		// Fall back to looking up the name from state
		if scene, ok := state.GetSmartScene(update.ID); ok {
			e.Name = state.GetSmartSceneName(scene)
		}
	}
	return e, nil
}

func (e *SmartSceneEvent) Render(styles *ui.Styles, width int) string {
	return renderEvent(styles, width, e.baseEvent, e.Name, e.Details, "", 0, false)
}
