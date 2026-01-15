package ui

import (
	"strings"
	"testing"
)

func TestTreeItemZone(t *testing.T) {
	tests := []struct {
		id   string
		want string
	}{
		{"room-123", "tree-room-123"},
		{"light-abc", "tree-light-abc"},
		{"", "tree-"},
	}

	for _, tt := range tests {
		t.Run(tt.id, func(t *testing.T) {
			got := TreeItemZone(tt.id)
			if got != tt.want {
				t.Errorf("TreeItemZone(%q) = %q, want %q", tt.id, got, tt.want)
			}
		})
	}
}

func TestBridgeZone(t *testing.T) {
	tests := []struct {
		id   string
		want string
	}{
		{"bridge-001", "bridge-bridge-001"},
		{"abc123", "bridge-abc123"},
	}

	for _, tt := range tests {
		t.Run(tt.id, func(t *testing.T) {
			got := BridgeZone(tt.id)
			if got != tt.want {
				t.Errorf("BridgeZone(%q) = %q, want %q", tt.id, got, tt.want)
			}
		})
	}
}

func TestBridgeItemZone(t *testing.T) {
	// BridgeItemZone is an alias for BridgeZone
	id := "test-id"
	if BridgeItemZone(id) != BridgeZone(id) {
		t.Error("BridgeItemZone should be alias for BridgeZone")
	}
}

func TestFormFieldZone(t *testing.T) {
	got := FormFieldZone("brightness")
	want := "form-field-brightness"
	if got != want {
		t.Errorf("FormFieldZone('brightness') = %q, want %q", got, want)
	}
}

func TestTabZone(t *testing.T) {
	tests := []struct {
		index int
		want  string
	}{
		{0, "tab-0"},
		{1, "tab-1"},
		{5, "tab-5"},
		{9, "tab-9"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := TabZone(tt.index)
			if got != tt.want {
				t.Errorf("TabZone(%d) = %q, want %q", tt.index, got, tt.want)
			}
		})
	}
}

func TestSliderZone(t *testing.T) {
	got := SliderZone("volume")
	want := "slider-track-volume"
	if got != want {
		t.Errorf("SliderZone('volume') = %q, want %q", got, want)
	}
}

func TestDetailFieldZone(t *testing.T) {
	got := DetailFieldZone("name")
	want := "detail-field-name"
	if got != want {
		t.Errorf("DetailFieldZone('name') = %q, want %q", got, want)
	}
}

func TestDropdownOptionZone(t *testing.T) {
	tests := []struct {
		fieldID     string
		optionIndex int
		want        string
	}{
		{"effect", 0, "dropdown-opt-effect-0"},
		{"effect", 5, "dropdown-opt-effect-5"},
		{"room-type", 12, "dropdown-opt-room-type-12"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := DropdownOptionZone(tt.fieldID, tt.optionIndex)
			if got != tt.want {
				t.Errorf("DropdownOptionZone(%q, %d) = %q, want %q",
					tt.fieldID, tt.optionIndex, got, tt.want)
			}
		})
	}
}

func TestZoneConstants(t *testing.T) {
	// Verify that zone constants have expected prefixes
	constants := map[string]string{
		"ZonePanelBridges":   ZonePanelBridges,
		"ZonePanelHierarchy": ZonePanelHierarchy,
		"ZonePanelDetails":   ZonePanelDetails,
		"ZonePanelLog":       ZonePanelLog,
	}

	for name, value := range constants {
		if !strings.HasPrefix(value, "panel-") {
			t.Errorf("Constant %s = %q, expected 'panel-' prefix", name, value)
		}
	}

	// Verify prefixes are non-empty
	prefixes := []string{
		ZoneBridgePrefix,
		ZoneTreeItemPrefix,
		ZoneDetailField,
		ZoneFormField,
		ZoneTabPrefix,
	}

	for _, prefix := range prefixes {
		if prefix == "" {
			t.Errorf("Zone prefix should not be empty")
		}
	}
}
