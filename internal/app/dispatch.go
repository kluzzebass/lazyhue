package app

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/kluzzebass/lazyhue/internal/ui"
	"github.com/kluzzebass/lazyhue/internal/ui/panels"
)

// dispatch executes an action and returns a tea.Cmd if any.
// This is the single place where actions map to actual behavior.
func (m *Model) dispatch(action ui.Action) tea.Cmd {
	switch action {
	// Navigation
	case ui.ActionUp:
		m.moveFocusedPanel(-1)
	case ui.ActionDown:
		m.moveFocusedPanel(1)
	case ui.ActionTop:
		m.moveFocusedPanelToTop()
	case ui.ActionBottom:
		m.moveFocusedPanelToBottom()
	case ui.ActionPageUp:
		m.moveFocusedPanelPage(-1)
	case ui.ActionPageDown:
		m.moveFocusedPanelPage(1)

	// Panel focus
	case ui.ActionNextPanel:
		m.nextPanel()
	case ui.ActionPrevPanel:
		m.prevPanel()
	case ui.ActionBack:
		m.returnFromDetail()

	// Bridge
	case ui.ActionNextBridge:
		m.manager.NextBridge()
		m.updateBridgePanel()
		m.refreshAllPanels()
	case ui.ActionPrevBridge:
		m.manager.PrevBridge()
		m.updateBridgePanel()
		m.refreshAllPanels()
	case ui.ActionPairBridge:
		return m.startBridgePairing()

	// Entity actions
	case ui.ActionSelect:
		return m.handleSelect()
	case ui.ActionToggle:
		return m.toggleSelected()
	case ui.ActionTurnOn:
		return m.setSelectedOn(true)
	case ui.ActionTurnOff:
		return m.setSelectedOn(false)
	case ui.ActionBrightnessUp:
		return m.adjustBrightness(10)
	case ui.ActionBrightnessDown:
		return m.adjustBrightness(-10)
	case ui.ActionExpandCollapse:
		if m.focusedPanelID() == PanelIDHierarchy {
			m.hierarchyPanel().ToggleSelected()
		}
	case ui.ActionExpand:
		// Pass to the hierarchy panel's Update which handles right/l key for expand
		if m.focusedPanelID() == PanelIDHierarchy {
			m.hierarchyPanel().Update(tea.KeyMsg{Type: tea.KeyRight})
			m.syncSelectionFromFocusedPanel()
		}
	case ui.ActionCollapse:
		// Pass to the hierarchy panel's Update which handles left/h key for collapse
		if m.focusedPanelID() == PanelIDHierarchy {
			m.hierarchyPanel().Update(tea.KeyMsg{Type: tea.KeyLeft})
			m.syncSelectionFromFocusedPanel()
		}

	// Global
	case ui.ActionHelp:
		m.toggleHelp()
	case ui.ActionQuit:
		m.quitting = true
		return tea.Quit
	}

	return nil
}

// Navigation helpers

func (m *Model) returnFromDetail() {
	if m.focusIndex < 0 {
		m.focusIndex = m.lastFocusIndex
		if m.focusIndex < 0 || m.focusIndex >= len(m.panelOrder) {
			m.focusIndex = 0
		}
		m.syncSelectionFromFocusedPanel()
	}
}

func (m *Model) moveFocusedPanel(delta int) {
	if panel := m.focusedPanel(); panel != nil {
		if delta < 0 {
			panel.Update(tea.KeyMsg{Type: tea.KeyUp})
		} else {
			panel.Update(tea.KeyMsg{Type: tea.KeyDown})
		}
		m.syncSelectionFromFocusedPanel()
	}
}

func (m *Model) moveFocusedPanelToTop() {
	if panel := m.focusedPanel(); panel != nil {
		panel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}})
		m.syncSelectionFromFocusedPanel()
	}
}

func (m *Model) moveFocusedPanelToBottom() {
	if panel := m.focusedPanel(); panel != nil {
		panel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'G'}})
		m.syncSelectionFromFocusedPanel()
	}
}

func (m *Model) moveFocusedPanelPage(direction int) {
	if panel := m.focusedPanel(); panel != nil {
		if direction < 0 {
			panel.Update(tea.KeyMsg{Type: tea.KeyPgUp})
		} else {
			panel.Update(tea.KeyMsg{Type: tea.KeyPgDown})
		}
		m.syncSelectionFromFocusedPanel()
	}
}

func (m *Model) nextPanel() {
	if m.focusIndex < 0 {
		m.focusIndex = 0
	} else {
		m.focusIndex++
		if m.focusIndex >= len(m.panelOrder) {
			m.focusIndex = 0
		}
	}
	m.syncSelectionFromFocusedPanel()
}

func (m *Model) prevPanel() {
	if m.focusIndex < 0 {
		m.focusIndex = len(m.panelOrder) - 1
	} else {
		m.focusIndex--
		if m.focusIndex < 0 {
			m.focusIndex = len(m.panelOrder) - 1
		}
	}
	m.syncSelectionFromFocusedPanel()
}

func (m *Model) toggleHelp() {
	if panel := m.focusedPanel(); panel != nil {
		panelID := m.focusedPanelID()
		bindings := m.panelBindings[panelID]
		// Filter bindings based on selection for hierarchy panel
		if panelID == PanelIDHierarchy {
			bindings = m.filterBindingsForSelection(bindings)
		}
		m.helpPanel.SetPanelBindings(panel.Title(), bindings)
	} else {
		m.helpPanel.SetPanelBindings("", nil)
	}
	m.helpPanel.SetGlobalBindings(m.globalBindings)

	// Build panel focus keys dynamically
	panelKeys := []panels.PanelInfo{
		{Key: "0", Title: "Details"},
	}
	for _, id := range m.panelOrder {
		if panel := m.panelMap[id]; panel != nil {
			panelKeys = append(panelKeys, panels.PanelInfo{
				Key:   panel.Key(),
				Title: panel.Title(),
			})
		}
	}
	m.helpPanel.SetPanelKeys(panelKeys)

	m.helpPanel.Toggle()
	if m.helpPanel.IsVisible() {
		m.statusBar.SetPopupHints("Esc close  ↑/↓ scroll")
	} else {
		m.statusBar.ClearPopupHints()
	}
}

