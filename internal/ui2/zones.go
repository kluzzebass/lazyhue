package ui2

import (
	zone "github.com/lrstanley/bubblezone/v2"
)

// Zone IDs for clickable regions.
// Using constants ensures consistency and enables autocomplete.
const (
	// Panel zones
	ZonePanelBridges   = "panel-bridges"
	ZonePanelHierarchy = "panel-hierarchy"
	ZonePanelDetails   = "panel-details"
	ZonePanelLog       = "panel-log"

	// Bridge list zones (prefix with index)
	ZoneBridgePrefix = "bridge-"

	// Tree/list item zones (prefix with path/id)
	ZoneTreeItemPrefix = "tree-"

	// Details panel zones
	ZoneDetailField  = "detail-field-"
	ZoneDetailToggle = "detail-toggle-"
	ZoneDetailSlider = "detail-slider-"
	ZoneDetailSelect = "detail-select-"
	ZoneDetailColor  = "detail-color-"

	// Form zones
	ZoneFormField   = "form-field-"
	ZoneFormButton  = "form-button-"
	ZoneFormSubmit  = "form-submit"
	ZoneFormCancel  = "form-cancel"
	ZoneColorWheel  = "color-wheel"
	ZoneSliderTrack = "slider-track-"

	// Popup zones
	ZonePopupClose = "popup-close"

	// Tab zones
	ZoneTabPrefix = "tab-"

	// Help zones
	ZoneHelpClose = "help-close"
)

// ZoneManager wraps bubblezone for easier zone management.
type ZoneManager struct {
	manager *zone.Manager
}

// NewZoneManager creates a new zone manager.
func NewZoneManager() *ZoneManager {
	return &ZoneManager{
		manager: zone.New(),
	}
}

// Manager returns the underlying bubblezone manager.
func (zm *ZoneManager) Manager() *zone.Manager {
	return zm.manager
}

// Mark wraps content in a clickable zone.
func (zm *ZoneManager) Mark(id string, content string) string {
	return zm.manager.Mark(id, content)
}

// Scan processes rendered output to enable zone tracking.
// Call this on the final View() output before returning.
func (zm *ZoneManager) Scan(output string) string {
	return zm.manager.Scan(output)
}

// TreeItemZone returns a zone ID for a tree item.
func TreeItemZone(id string) string {
	return ZoneTreeItemPrefix + id
}

// BridgeZone returns a zone ID for a bridge item.
func BridgeZone(id string) string {
	return ZoneBridgePrefix + id
}

// FormFieldZone returns a zone ID for a form field.
func FormFieldZone(id string) string {
	return ZoneFormField + id
}

// TabZone returns a zone ID for a tab.
func TabZone(index int) string {
	return ZoneTabPrefix + string(rune('0'+index))
}

// SliderZone returns a zone ID for a slider track.
func SliderZone(id string) string {
	return ZoneSliderTrack + id
}

// DetailFieldZone returns a zone ID for a detail field.
func DetailFieldZone(id string) string {
	return ZoneDetailField + id
}
