package hueclient

// DeviceName returns the name of a device, or fallback if not available.
func (device *DeviceGet) DeviceName(fallback string) string {
	if device.Metadata.Name != "" {
		return device.Metadata.Name
	}
	return fallback
}
