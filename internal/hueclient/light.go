package hueclient

// LightName returns the name from a light's metadata, or fallback if not available.
// Note: For user-assigned names, prefer BridgeState.GetLightName which checks the owning device.
func (light *LightGet) LightName(fallback string) string {
	if light.Metadata.Name != "" {
		return light.Metadata.Name
	}
	return fallback
}
