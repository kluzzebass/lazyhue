package hue

import (
	"context"
	"fmt"

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

// CreateSceneFromCurrentState creates a new scene for a room/zone using the current light states.
// TODO: Re-implement with new hueclient types after oapi-hue migration
func (b *Bridge) CreateSceneFromCurrentState(groupID string, isZone bool, sceneName string) error {
	return fmt.Errorf("CreateSceneFromCurrentState: not yet implemented with new hueclient types")
}
