package hueclient

// ZoneName returns the name of a zone, or fallback if not available.
func (zone *ZoneGet) ZoneName(fallback string) string {
	if zone.Metadata.Name != "" {
		return zone.Metadata.Name
	}
	return fallback
}
