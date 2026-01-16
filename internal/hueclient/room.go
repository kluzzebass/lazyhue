package hueclient

// RoomName returns the name of a room, or fallback if not available.
func (room *RoomGet) RoomName(fallback string) string {
	if room.Metadata.Name != "" {
		return room.Metadata.Name
	}
	return fallback
}
