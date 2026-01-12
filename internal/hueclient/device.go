package hueclient

// DeviceName returns the name of a device, or fallback if not available.
func (device *DeviceGet) DeviceName(fallback string) string {
	if device.Metadata != nil && device.Metadata.Name != nil {
		return *device.Metadata.Name
	}
	return fallback
}

// DeviceDeprecatedName is a helper that returns the deprecated name from associated lights.
// Since devices don't have direct access to their lights, this function should be called
// with a function that can retrieve lights for the device, or use BridgeState.GetDeviceDeprecatedName.
// The deprecated name comes from light.Metadata.Name (which is deprecated in favor of device.Metadata.Name).
