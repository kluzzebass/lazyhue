package panels

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/kluzzebass/lazyhue/internal/hue"
	"github.com/kluzzebass/lazyhue/internal/ui"
	"github.com/kluzzebass/lazyhue/internal/ui/panels/details"
)

// DetailsPanel shows details for the selected entity using the new view system.
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

	// Content area: width minus borders (2), height minus borders (2)
	p.viewport.Width = max(1, width-2)
	p.viewport.Height = max(1, height-2)
	p.updateContent()
}

func (p *DetailsPanel) updateContent() {
	if p.item == nil {
		p.viewport.SetContent("No selection")
		return
	}

	// Build header
	var header strings.Builder
	title := p.styles.Title.Render(p.item.Name)
	header.WriteString(title)
	header.WriteString("\n")
	header.WriteString(strings.Repeat("─", max(0, min(30, p.width-6))))
	header.WriteString("\n\n")

	// Build view based on entity type
	var view *details.View
	switch p.item.Type {
	case EntityRoom:
		if room, ok := GetRoomFromItem(*p.item); ok {
			view = p.buildRoomView(room, false)
		}
	case EntityZone:
		if zone, ok := GetRoomFromItem(*p.item); ok {
			view = p.buildRoomView(zone, true)
		}
	case EntityLight:
		if light, ok := GetLightFromItem(*p.item); ok {
			view = p.buildLightView(light)
		}
	case EntityScene:
		if scene, ok := GetSceneFromItem(*p.item); ok {
			view = p.buildSceneView(scene)
		}
	case EntityDevice:
		if device, ok := GetDeviceFromItem(*p.item); ok {
			view = p.buildDeviceView(device)
		}
	case EntityEntertainment:
		if cfg, ok := GetEntertainmentFromItem(*p.item); ok {
			view = p.buildEntertainmentView(cfg)
		}
	case EntityLightsCategory:
		if data, ok := p.item.RawPtr.(LightsCategoryData); ok {
			view = p.buildLightsCategoryView(data)
		}
	case EntityDevicesCategory:
		if data, ok := p.item.RawPtr.(DevicesCategoryData); ok {
			view = p.buildDevicesCategoryView(data)
		}
	case EntityScenesCategory:
		if data, ok := p.item.RawPtr.(ScenesCategoryData); ok {
			view = p.buildScenesCategoryView(data)
		}
	case EntityBridge:
		if bridgeData, ok := GetBridgeFromItem(*p.item); ok {
			view = p.buildBridgeView(bridgeData)
		}
	default:
		// Fallback for unknown types
		view = details.NewView(p.styles)
		fields := details.NewFields()
		fields.AddMuted("Type", fmt.Sprintf("%d", p.item.Type))
		view.Add(fields)
	}

	// Combine header and view content
	content := header.String()
	if view != nil {
		content += view.Render()
	}

	p.viewport.SetContent(content)
}

// Update handles input for the details panel.
func (p *DetailsPanel) Update(msg tea.Msg) (*DetailsPanel, tea.Cmd) {
	// Handle g/G for top/bottom (viewport uses home/end)
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		switch keyMsg.String() {
		case "g":
			p.viewport.GotoTop()
			return p, nil
		case "G":
			p.viewport.GotoBottom()
			return p, nil
		}
	}

	var cmd tea.Cmd
	p.viewport, cmd = p.viewport.Update(msg)
	return p, cmd
}

// View renders the details panel.
func (p *DetailsPanel) View(active bool) string {
	content := p.viewport.View()

	// Count content lines for scroll indicator
	contentLines := strings.Count(p.viewport.View(), "\n") + 1
	totalLines := p.viewport.TotalLineCount()

	cfg := ui.BorderConfig{
		Title:       "[0] Details",
		ScrollPos:   p.viewport.YOffset,
		TotalHeight: totalLines,
		ViewHeight:  contentLines,
	}

	return ui.RenderBorderedPanel(content, p.width, p.height, active, p.styles, cfg)
}

// Title returns the panel title.
func (p *DetailsPanel) Title() string {
	return "Details"
}

// HasItems returns true if there is content to display.
func (p *DetailsPanel) HasItems() bool {
	return p.item != nil
}

// HandleScroll handles scroll events.
func (p *DetailsPanel) HandleScroll(lines int) {
	if lines > 0 {
		p.viewport.LineDown(lines)
	} else {
		p.viewport.LineUp(-lines)
	}
}

// Builder methods are implemented in separate files:
// - details_device.go: buildDeviceView
// - details_room.go: buildRoomView
// - details_light.go: buildLightView
// - details_scene.go: buildSceneView
// - details_entertainment.go: buildEntertainmentView
// - details_category.go: buildLightsCategoryView, buildDevicesCategoryView, buildScenesCategoryView
// - details_bridge.go: buildBridgeView
