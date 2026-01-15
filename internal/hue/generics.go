package hue

// Generic helper functions for map operations with mutex locking.
// These reduce boilerplate in BridgeState methods.

// getFromMap retrieves a value from a map with read locking.
// The caller must pass the appropriate map field.
func getFromMap[V any](s *BridgeState, m map[string]V, id string) (V, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	v, ok := m[id]
	return v, ok
}

// addToMap adds a value to a map with write locking.
// Initializes the map if nil.
func addToMap[V any](s *BridgeState, m *map[string]V, id string, value V) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if *m == nil {
		*m = make(map[string]V)
	}
	(*m)[id] = value
}

// removeFromMap removes a value from a map with write locking.
// Returns true if the value existed.
func removeFromMap[V any](s *BridgeState, m map[string]V, id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := m[id]; ok {
		delete(m, id)
		return true
	}
	return false
}

// updateMap replaces an entire map with write locking.
func updateMap[V any](s *BridgeState, m *map[string]V, newMap map[string]V) {
	s.mu.Lock()
	defer s.mu.Unlock()
	*m = newMap
}

// displayName looks up a display name from a map, returning the key as fallback.
func displayName(m map[string]string, key string) string {
	if name, ok := m[key]; ok {
		return name
	}
	return key
}
