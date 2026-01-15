package hue

import "testing"

func TestEffectDisplayName(t *testing.T) {
	tests := []struct {
		effect, want string
	}{
		{"no_effect", "None"},
		{"candle", "Candle"},
		{"fire", "Fire"},
		{"prism", "Prism"},
		{"sparkle", "Sparkle"},
		{"opal", "Opal"},
		{"glisten", "Glisten"},
		{"unknown_effect", "unknown_effect"}, // Returns raw name if unknown
	}

	for _, tt := range tests {
		t.Run(tt.effect, func(t *testing.T) {
			got := EffectDisplayName(tt.effect)
			if got != tt.want {
				t.Errorf("EffectDisplayName(%q) = %q, want %q", tt.effect, got, tt.want)
			}
		})
	}
}

func TestSignalingModeDisplayName(t *testing.T) {
	tests := []struct {
		mode, want string
	}{
		{"no_signal", "No Signal"},
		{"on_off", "On/Off"},
		{"on_off_color", "On/Off Color"},
		{"alternating", "Alternating"},
		{"unknown_mode", "unknown_mode"}, // Returns raw name if unknown
	}

	for _, tt := range tests {
		t.Run(tt.mode, func(t *testing.T) {
			got := SignalingModeDisplayName(tt.mode)
			if got != tt.want {
				t.Errorf("SignalingModeDisplayName(%q) = %q, want %q", tt.mode, got, tt.want)
			}
		})
	}
}

func TestPowerupPresetDisplayName(t *testing.T) {
	tests := []struct {
		preset, want string
	}{
		{"safety", "Safety"},
		{"powerfail", "Power Fail"},
		{"last_on_state", "Last On State"},
		{"custom", "Custom"},
		{"unknown_preset", "unknown_preset"}, // Returns raw name if unknown
	}

	for _, tt := range tests {
		t.Run(tt.preset, func(t *testing.T) {
			got := PowerupPresetDisplayName(tt.preset)
			if got != tt.want {
				t.Errorf("PowerupPresetDisplayName(%q) = %q, want %q", tt.preset, got, tt.want)
			}
		})
	}
}

func TestDeviceServiceDisplayName(t *testing.T) {
	tests := []struct {
		service, want string
	}{
		{"light", "Light"},
		{"grouped_light", "Grouped Light"},
		{"motion", "Motion Sensor"},
		{"temperature", "Temperature Sensor"},
		{"light_level", "Light Level Sensor"},
		{"button", "Button"},
		{"device_power", "Battery"},
		{"zigbee_connectivity", "Zigbee Connectivity"},
		{"unknown_service", "unknown_service"}, // Returns raw name if unknown
	}

	for _, tt := range tests {
		t.Run(tt.service, func(t *testing.T) {
			got := DeviceServiceDisplayName(tt.service)
			if got != tt.want {
				t.Errorf("DeviceServiceDisplayName(%q) = %q, want %q", tt.service, got, tt.want)
			}
		})
	}
}

func TestProductArchetypeDisplayName(t *testing.T) {
	tests := []struct {
		archetype, want string
	}{
		{"bridge_v2", "Bridge V2"},
		{"unknown_archetype", "Unknown Archetype"},
		{"hue_lightstrip", "Hue Lightstrip"},
		{"hue_bloom", "Hue Bloom"},
		{"really_unknown", "really_unknown"}, // Returns raw name if not in map
	}

	for _, tt := range tests {
		t.Run(tt.archetype, func(t *testing.T) {
			got := ProductArchetypeDisplayName(tt.archetype)
			if got != tt.want {
				t.Errorf("ProductArchetypeDisplayName(%q) = %q, want %q", tt.archetype, got, tt.want)
			}
		})
	}
}

func TestRoomArchetypeDisplayName(t *testing.T) {
	tests := []struct {
		archetype, want string
	}{
		{"living_room", "Living Room"},
		{"bedroom", "Bedroom"},
		{"kitchen", "Kitchen"},
		{"bathroom", "Bathroom"},
		{"office", "Office"},
		{"unknown_room", "unknown_room"}, // Returns raw name if unknown
	}

	for _, tt := range tests {
		t.Run(tt.archetype, func(t *testing.T) {
			got := RoomArchetypeDisplayName(tt.archetype)
			if got != tt.want {
				t.Errorf("RoomArchetypeDisplayName(%q) = %q, want %q", tt.archetype, got, tt.want)
			}
		})
	}
}

func TestRoomArchetypeList(t *testing.T) {
	list := RoomArchetypeList()
	if len(list) == 0 {
		t.Error("RoomArchetypeList() returned empty list")
	}
	// Check that common room types are in the list
	expected := []string{"living_room", "bedroom", "kitchen", "bathroom", "office"}
	for _, want := range expected {
		found := false
		for _, got := range list {
			if got == want {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("RoomArchetypeList() missing expected archetype %q", want)
		}
	}
}
