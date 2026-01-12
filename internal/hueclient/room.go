package hueclient

// RoomName returns the name of a room, or fallback if not available.
func (room *RoomGet) RoomName(fallback string) string {
	if room.Metadata != nil && room.Metadata.Name != nil {
		return *room.Metadata.Name
	}
	return fallback
}
