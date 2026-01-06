package panels

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/kluzzebass/lazyhue/internal/hue"
	"github.com/kluzzebass/lazyhue/internal/ui"
	"github.com/openhue/openhue-go"
)

// DetailsPanel shows details for the selected entity.
type DetailsPanel struct {
	viewport viewport.Model
	styles   ui.Styles
	item     *EntityItem
	state    *hue.BridgeState
	width    int
	height   int
}

// NewDetailsPanel creates a new details panel.
func NewDetailsPanel(styles ui.Styles) *DetailsPanel {
	vp := viewport.New(0, 0)
	return &DetailsPanel{
		viewport: vp,
		styles:   styles,
	}
}

// SetItem updates the displayed entity.
func (p *DetailsPanel) SetItem(item *EntityItem, state *hue.BridgeState) {
	p.item = item
	p.state = state
	p.updateContent()
}

// SetSize updates the panel dimensions.
func (p *DetailsPanel) SetSize(width, height int) {
	p.width = width
	p.height = height
	p.viewport.Width = width - 4
	p.viewport.Height = height - 4
	p.updateContent()
}

func (p *DetailsPanel) updateContent() {
	if p.item == nil {
		p.viewport.SetContent("No selection")
		return
	}

	var content strings.Builder

	title := p.styles.Title.Render(p.item.Name)
	content.WriteString(title)
	content.WriteString("\n")
	content.WriteString(strings.Repeat("─", min(30, p.width-6)))
	content.WriteString("\n\n")

	switch p.item.Type {
	case EntityRoom:
		if room, ok := GetRoomFromItem(*p.item); ok {
			p.renderRoom(&content, room)
		}
	case EntityLight:
		if light, ok := GetLightFromItem(*p.item); ok {
			p.renderLight(&content, light)
		}
	default:
		content.WriteString(p.styles.Muted.Render(fmt.Sprintf("Type: %d", p.item.Type)))
	}

	p.viewport.SetContent(content.String())
}

func (p *DetailsPanel) renderRoom(w *strings.Builder, room openhue.RoomGet) {
	if p.state == nil {
		return
	}

	lights := p.state.RoomLights(room)
	onCount := 0
	for _, l := range lights {
		if l.IsOn() {
			onCount++
		}
	}

	status := fmt.Sprintf("Status: %d/%d lights on", onCount, len(lights))
	w.WriteString(status)
	w.WriteString("\n")

	if gl, ok := p.state.RoomGroupedLight(room); ok {
		if gl.Dimming != nil && gl.Dimming.Brightness != nil {
			brightness := fmt.Sprintf("Brightness: %.0f%%", float64(*gl.Dimming.Brightness))
			w.WriteString(brightness)
			w.WriteString("\n")
		}
	}

	w.WriteString("\n")
	w.WriteString(p.styles.Subtitle.Render("Lights:"))
	w.WriteString("\n")

	for _, light := range lights {
		name := ""
		if light.Metadata != nil && light.Metadata.Name != nil {
			name = *light.Metadata.Name
		}
		indicator := p.styles.OnOffIndicator(light.IsOn())
		brightness := ""
		if light.IsOn() {
			if light.Dimming != nil && light.Dimming.Brightness != nil {
				brightness = fmt.Sprintf(" %.0f%%", float64(*light.Dimming.Brightness))
			}
		} else {
			brightness = " Off"
		}
		line := fmt.Sprintf("  %s %s%s", indicator, name, brightness)
		w.WriteString(line)
		w.WriteString("\n")
	}

	roomID := ""
	if room.Id != nil {
		roomID = *room.Id
	}
	scenes := p.state.RoomScenes(roomID)
	if len(scenes) > 0 {
		w.WriteString("\n")
		w.WriteString(p.styles.Subtitle.Render("Scenes:"))
		w.WriteString("\n")
		for _, scene := range scenes {
			name := ""
			if scene.Metadata != nil && scene.Metadata.Name != nil {
				name = *scene.Metadata.Name
			}
			w.WriteString(fmt.Sprintf("  • %s\n", name))
		}
	}
}

func (p *DetailsPanel) renderLight(w *strings.Builder, light openhue.LightGet) {
	indicator := p.styles.OnOffIndicator(light.IsOn())
	status := "Off"
	if light.IsOn() {
		status = "On"
	}
	w.WriteString(fmt.Sprintf("Status: %s %s\n", indicator, status))

	if light.IsOn() && light.Dimming != nil && light.Dimming.Brightness != nil {
		w.WriteString(fmt.Sprintf("Brightness: %.0f%%\n", float64(*light.Dimming.Brightness)))
	}

	w.WriteString("\n")
	w.WriteString(p.styles.Subtitle.Render("Capabilities:"))
	w.WriteString("\n")

	caps := []string{}
	if light.Dimming != nil {
		caps = append(caps, "Dimming")
	}
	if light.Color != nil {
		caps = append(caps, "Color")
	}
	if light.ColorTemperature != nil {
		caps = append(caps, "Temperature")
	}

	if len(caps) > 0 {
		w.WriteString(fmt.Sprintf("  %s\n", strings.Join(caps, ", ")))
	} else {
		w.WriteString("  On/Off only\n")
	}
}

// Update handles input for the details panel.
func (p *DetailsPanel) Update(msg tea.Msg) (*DetailsPanel, tea.Cmd) {
	var cmd tea.Cmd
	p.viewport, cmd = p.viewport.Update(msg)
	return p, cmd
}

// View renders the details panel.
func (p *DetailsPanel) View(active bool) string {
	var panelStyle lipgloss.Style
	if active {
		panelStyle = p.styles.ActivePanel
	} else {
		panelStyle = p.styles.RightPanel
	}

	return panelStyle.Width(p.width).Height(p.height).Render(p.viewport.View())
}
