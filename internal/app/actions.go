package app

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/kluzzebass/lazyhue/internal/ui/panels"
)

func (m *Model) toggleSelected() tea.Cmd {
	bridge := m.manager.GetActiveBridge()
	if bridge == nil || m.selectedItem == nil {
		return nil
	}

	switch m.selectedItem.Type {
	case panels.EntityRoom, panels.EntityZone:
		if room, ok := panels.GetRoomFromItem(*m.selectedItem); ok {
			if gl, ok := bridge.GetState().RoomGroupedLight(room); ok {
				if gl.Id != nil {
					return toggleGroupedLight(bridge, *gl.Id)
				}
			}
		}
	case panels.EntityLight:
		return toggleLight(bridge, m.selectedItem.ID)
	}
	return nil
}

func (m *Model) setSelectedOn(on bool) tea.Cmd {
	bridge := m.manager.GetActiveBridge()
	if bridge == nil || m.selectedItem == nil {
		return nil
	}

	switch m.selectedItem.Type {
	case panels.EntityRoom, panels.EntityZone:
		if room, ok := panels.GetRoomFromItem(*m.selectedItem); ok {
			if gl, ok := bridge.GetState().RoomGroupedLight(room); ok {
				if gl.Id != nil {
					return setGroupedLightOn(bridge, *gl.Id, on)
				}
			}
		}
	case panels.EntityLight:
		return setLightOn(bridge, m.selectedItem.ID, on)
	}
	return nil
}

func (m *Model) adjustBrightness(delta float64) tea.Cmd {
	bridge := m.manager.GetActiveBridge()
	if bridge == nil || m.selectedItem == nil {
		return nil
	}

	switch m.selectedItem.Type {
	case panels.EntityRoom, panels.EntityZone:
		if room, ok := panels.GetRoomFromItem(*m.selectedItem); ok {
			if gl, ok := bridge.GetState().RoomGroupedLight(room); ok {
				if gl.Id != nil {
					current := float64(0)
					if gl.Dimming != nil && gl.Dimming.Brightness != nil {
						current = float64(*gl.Dimming.Brightness)
					}
					return setGroupedLightBrightness(bridge, *gl.Id, clamp(current+delta, 0, 100))
				}
			}
		}
	case panels.EntityLight:
		if light, ok := panels.GetLightFromItem(*m.selectedItem); ok {
			current := float64(0)
			if light.Dimming != nil && light.Dimming.Brightness != nil {
				current = float64(*light.Dimming.Brightness)
			}
			return setLightBrightness(bridge, m.selectedItem.ID, clamp(current+delta, 0, 100))
		}
	}
	return nil
}

func clamp(value, min, max float64) float64 {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}
