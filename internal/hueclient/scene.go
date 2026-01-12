package hueclient

// SceneName returns the name of a scene, or fallback if not available.
func (scene *SceneGet) SceneName(fallback string) string {
	if scene.Metadata != nil && scene.Metadata.Name != nil {
		return *scene.Metadata.Name
	}
	return fallback
}
