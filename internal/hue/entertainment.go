package hue

// EntertainmentName returns the name of an entertainment configuration, or fallback if not available.
func (cfg *EntertainmentConfiguration) EntertainmentName(fallback string) string {
	if cfg.Metadata != nil && cfg.Metadata.Name != "" {
		return cfg.Metadata.Name
	}
	if cfg.Name != "" {
		return cfg.Name
	}
	return fallback
}
