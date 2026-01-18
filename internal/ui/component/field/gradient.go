package field

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/charmbracelet/lipgloss/v2"
	"github.com/kluzzebass/lazyhue/internal/ui"
	"github.com/kluzzebass/lazyhue/internal/ui/component"
	zone "github.com/lrstanley/bubblezone/v2"
)

// GradientEditorComponent allows selecting gradient points for external editing.
type GradientEditorComponent struct {
	*BaseField

	// Gradient points
	Points    []GradientPoint
	MaxPoints int
	MinPoints int // Minimum allowed points

	// Selection state
	SelectedIndex int // -1 if none selected

	// Color wheel for editing
	Wheel *ColorWheel

	// Editing state
	Editing          bool
	blinkTimerActive bool
	blinkScheduled   bool

	// Drag state for color wheel
	Dragging     bool
	DragZoneInfo map[int]*zone.ZoneInfo
}

// NewGradientEditorComponent creates a new gradient editor.
func NewGradientEditorComponent(id, label string, points []GradientPoint, maxPoints int, styles *ui.Styles, zones *zone.Manager) *GradientEditorComponent {
	wheel := NewColorWheel()

	minPoints := 2
	if maxPoints <= 1 {
		minPoints = 1
	}

	// Ensure we have at least the minimum points
	if len(points) < minPoints {
		defaultPoint := GradientPoint{0.3127, 0.329}
		for len(points) < minPoints {
			points = append(points, defaultPoint)
		}
	}

	return &GradientEditorComponent{
		BaseField:     NewBaseField(id, label, styles, zones),
		Points:        points,
		MaxPoints:     maxPoints,
		MinPoints:     minPoints,
		SelectedIndex: -1,
		Wheel:         wheel,
	}
}

// Update handles events for the gradient editor.
func (g *GradientEditorComponent) Update(msg tea.Msg) (component.Component, tea.Cmd) {
	if g.ReadOnly {
		return g, nil
	}

	switch msg := msg.(type) {
	case BlinkTickMsg:
		if msg.FieldID == g.ID && g.blinkTimerActive && g.Editing {
			g.Wheel.BlinkOn = !g.Wheel.BlinkOn
			g.blinkScheduled = false
			return g, g.scheduleBlinkTick()
		}

	case tea.KeyMsg:
		if g.Editing {
			return g.handleEditingKey(msg)
		}
		return g.handleNormalKey(msg)

	case tea.MouseClickMsg:
		return g.handleMouseClick(msg)

	case tea.MouseMotionMsg:
		if g.Dragging {
			return g.handleMouseDrag(msg)
		}

	case tea.MouseReleaseMsg:
		if g.Dragging {
			return g.handleMouseRelease()
		}
	}

	return g, nil
}

// RouteEvent routes events to this component.
func (g *GradientEditorComponent) RouteEvent(msg tea.Msg) (bool, tea.Cmd) {
	if g.ReadOnly {
		return false, nil
	}

	switch msg := msg.(type) {
	case BlinkTickMsg:
		if msg.FieldID == g.ID {
			_, cmd := g.Update(msg)
			return true, cmd
		}

	case tea.KeyMsg:
		if g.Editing {
			_, cmd := g.handleEditingKey(msg)
			return true, cmd
		}
		// Handle point navigation
		switch msg.String() {
		case "left", "h":
			if g.SelectedIndex > 0 {
				g.SelectedIndex--
				g.syncWheelToSelected()
				return true, nil
			}
		case "right", "l":
			if g.SelectedIndex < len(g.Points)-1 {
				g.SelectedIndex++
				g.syncWheelToSelected()
				return true, nil
			}
		case "enter", " ":
			if g.SelectedIndex < 0 && len(g.Points) > 0 {
				g.SelectedIndex = 0
				g.syncWheelToSelected()
			}
			return true, nil
		case "+", "=":
			if len(g.Points) < g.MaxPoints {
				return true, g.addPoint()
			}
		case "-", "backspace", "delete":
			if len(g.Points) > g.MinPoints && g.SelectedIndex >= 0 {
				return true, g.removePoint()
			}
		}

	case tea.MouseClickMsg:
		if msg.Button == tea.MouseLeft {
			// Check swatch zones
			for i := range g.Points {
				zoneID := fmt.Sprintf("%s-swatch-%d", g.ZoneID(), i)
				if g.Zones != nil {
					if z := g.Zones.Get(zoneID); z != nil && z.InBounds(msg) {
						g.SelectedIndex = i
						g.syncWheelToSelected()
						return true, nil
					}
				}
			}

			// Check add button
			addZoneID := fmt.Sprintf("%s-add", g.ZoneID())
			if g.Zones != nil {
				if z := g.Zones.Get(addZoneID); z != nil && z.InBounds(msg) {
					if len(g.Points) < g.MaxPoints {
						_, cmd := g.Update(msg)
						return true, tea.Batch(cmd, g.addPoint())
					}
				}
			}

			// Check remove button
			removeZoneID := fmt.Sprintf("%s-remove", g.ZoneID())
			if g.Zones != nil {
				if z := g.Zones.Get(removeZoneID); z != nil && z.InBounds(msg) {
					if len(g.Points) > g.MinPoints && g.SelectedIndex >= 0 {
						_, cmd := g.Update(msg)
						return true, tea.Batch(cmd, g.removePoint())
					}
				}
			}

			// Check wheel row zones when editing
			if g.Editing {
				for row := 0; row < g.Wheel.Height(); row++ {
					rowZoneID := fmt.Sprintf("%s-wheel-row-%d", g.ZoneID(), row)
					if g.Zones != nil {
						if z := g.Zones.Get(rowZoneID); z != nil && z.InBounds(msg) {
							return true, g.handleWheelClick(msg, row, z)
						}
					}
				}
			}
		}

	case tea.MouseMotionMsg:
		if g.Dragging {
			_, cmd := g.handleMouseDrag(msg)
			return true, cmd
		}

	case tea.MouseReleaseMsg:
		if g.Dragging {
			_, cmd := g.handleMouseRelease()
			return true, cmd
		}
	}

	return false, nil
}

func (g *GradientEditorComponent) handleNormalKey(msg tea.KeyMsg) (component.Component, tea.Cmd) {
	switch msg.String() {
	case "left", "h":
		if g.SelectedIndex > 0 {
			g.SelectedIndex--
			g.syncWheelToSelected()
		}
	case "right", "l":
		if g.SelectedIndex < len(g.Points)-1 {
			g.SelectedIndex++
			g.syncWheelToSelected()
		}
	case "enter", " ":
		if g.SelectedIndex < 0 && len(g.Points) > 0 {
			g.SelectedIndex = 0
			g.syncWheelToSelected()
		}
	}
	return g, nil
}

func (g *GradientEditorComponent) handleEditingKey(msg tea.KeyMsg) (component.Component, tea.Cmd) {
	switch msg.String() {
	case "esc":
		// Cancel editing, restore original
		g.Wheel.RestoreOriginal()
		if g.SelectedIndex >= 0 && g.SelectedIndex < len(g.Points) {
			g.Points[g.SelectedIndex].X = g.Wheel.ColorX
			g.Points[g.SelectedIndex].Y = g.Wheel.ColorY
		}
		g.Editing = false
		g.blinkTimerActive = false
		return g, nil

	case "enter":
		// Confirm edit
		if g.SelectedIndex >= 0 && g.SelectedIndex < len(g.Points) {
			g.Points[g.SelectedIndex].X = g.Wheel.ColorX
			g.Points[g.SelectedIndex].Y = g.Wheel.ColorY
		}
		g.Editing = false
		g.blinkTimerActive = false
		return g, g.sendFieldChanged()

	case "left", "h":
		g.Wheel.MoveLeft()
		g.updateSelectedFromWheel()
	case "right", "l":
		g.Wheel.MoveRight()
		g.updateSelectedFromWheel()
	case "up", "k":
		g.Wheel.MoveUp()
		g.updateSelectedFromWheel()
	case "down", "j":
		g.Wheel.MoveDown()
		g.updateSelectedFromWheel()
	}

	return g, nil
}

func (g *GradientEditorComponent) handleMouseClick(msg tea.MouseClickMsg) (component.Component, tea.Cmd) {
	// Handled in RouteEvent for zone detection
	return g, nil
}

func (g *GradientEditorComponent) handleWheelClick(msg tea.MouseClickMsg, row int, z *zone.ZoneInfo) tea.Cmd {
	col := msg.X - z.StartX
	if g.Wheel.HandleClick(row, col) {
		g.updateSelectedFromWheel()

		// Start drag
		g.Dragging = true
		g.DragZoneInfo = make(map[int]*zone.ZoneInfo)
		for r := 0; r < g.Wheel.Height(); r++ {
			rowZoneID := fmt.Sprintf("%s-wheel-row-%d", g.ZoneID(), r)
			if g.Zones != nil {
				if zInfo := g.Zones.Get(rowZoneID); zInfo != nil {
					g.DragZoneInfo[r] = zInfo
				}
			}
		}

		return tea.Batch(
			func() tea.Msg { return StartCaptureMsg{FieldID: g.ID} },
			g.sendFieldChanged(),
		)
	}
	return nil
}

func (g *GradientEditorComponent) handleMouseDrag(msg tea.MouseMotionMsg) (component.Component, tea.Cmd) {
	if g.DragZoneInfo == nil {
		return g, nil
	}

	for row, zInfo := range g.DragZoneInfo {
		if msg.Y >= zInfo.StartY && msg.Y <= zInfo.EndY {
			col := msg.X - zInfo.StartX
			if g.Wheel.HandleClick(row, col) {
				g.updateSelectedFromWheel()
				return g, g.sendFieldChanged()
			}
			return g, nil
		}
	}

	return g, nil
}

func (g *GradientEditorComponent) handleMouseRelease() (component.Component, tea.Cmd) {
	g.Dragging = false
	g.DragZoneInfo = nil
	return g, tea.Batch(
		func() tea.Msg { return EndCaptureMsg{FieldID: g.ID} },
		g.sendFieldChanged(),
	)
}

func (g *GradientEditorComponent) startEditing() tea.Cmd {
	if g.SelectedIndex < 0 || g.SelectedIndex >= len(g.Points) {
		return nil
	}

	g.Editing = true
	pt := g.Points[g.SelectedIndex]
	g.Wheel.SetOriginal(pt.X, pt.Y)
	g.Wheel.SetColor(pt.X, pt.Y)
	g.Wheel.BlinkOn = true
	g.blinkTimerActive = true

	return g.scheduleBlinkTick()
}

func (g *GradientEditorComponent) syncWheelToSelected() {
	if g.SelectedIndex >= 0 && g.SelectedIndex < len(g.Points) {
		pt := g.Points[g.SelectedIndex]
		g.Wheel.SetColor(pt.X, pt.Y)
	}
}

func (g *GradientEditorComponent) updateSelectedFromWheel() {
	if g.SelectedIndex >= 0 && g.SelectedIndex < len(g.Points) {
		g.Points[g.SelectedIndex].X = g.Wheel.ColorX
		g.Points[g.SelectedIndex].Y = g.Wheel.ColorY
	}
}

func (g *GradientEditorComponent) addPoint() tea.Cmd {
	// Add a new point (default to white)
	newPoint := GradientPoint{X: 0.3127, Y: 0.329}
	g.Points = append(g.Points, newPoint)
	g.SelectedIndex = len(g.Points) - 1
	g.syncWheelToSelected()
	return g.sendFieldChanged()
}

func (g *GradientEditorComponent) removePoint() tea.Cmd {
	if len(g.Points) <= g.MinPoints || g.SelectedIndex < 0 {
		return nil
	}

	// Remove the selected point
	g.Points = append(g.Points[:g.SelectedIndex], g.Points[g.SelectedIndex+1:]...)

	// Adjust selection
	if g.SelectedIndex >= len(g.Points) {
		g.SelectedIndex = len(g.Points) - 1
	}
	if g.SelectedIndex >= 0 {
		g.syncWheelToSelected()
	}

	return g.sendFieldChanged()
}

func (g *GradientEditorComponent) sendFieldChanged() tea.Cmd {
	points := make([]GradientPoint, len(g.Points))
	copy(points, g.Points)
	return func() tea.Msg {
		return FieldChangedMsg{
			FieldID: g.ID,
			Value:   GradientValue{Points: points},
		}
	}
}

func (g *GradientEditorComponent) scheduleBlinkTick() tea.Cmd {
	if g.blinkScheduled {
		return nil
	}
	g.blinkScheduled = true
	fieldID := g.ID
	return tea.Tick(500*time.Millisecond, func(t time.Time) tea.Msg {
		return BlinkTickMsg{FieldID: fieldID}
	})
}

// FieldHeight returns the height of this component.
func (g *GradientEditorComponent) FieldHeight() int {
	// Swatches row only
	return 1
}

// ViewControl renders the control portion.
func (g *GradientEditorComponent) ViewControl() string {
	var out strings.Builder

	// Render swatches row
	out.WriteString(g.renderSwatches())

	return out.String()
}

func (g *GradientEditorComponent) renderSwatches() string {
	var parts []string

	// Render each gradient point as a colored square
	for i, pt := range g.Points {
		// Convert XY to RGB for display
		r, gr, b := ui.XyToRGB(pt.X, pt.Y, 100.0) // Full brightness for display
		color := lipgloss.Color(fmt.Sprintf("#%02x%02x%02x", r, gr, b))

		// Swatch style - use foreground color for block characters
		style := lipgloss.NewStyle().Foreground(color)

		// Selected indicator
		swatch := "██"
		if i == g.SelectedIndex {
			// Show selection with different character
			if g.Editing {
				swatch = "▓▓"
			} else {
				swatch = "▒▒"
			}
		}

		zoneID := fmt.Sprintf("%s-swatch-%d", g.ZoneID(), i)
		if g.Zones != nil {
			parts = append(parts, g.Zones.Mark(zoneID, style.Render(swatch)))
		} else {
			parts = append(parts, style.Render(swatch))
		}
	}

	// Add/remove buttons
	addStyle := g.Styles.Dimmed
	removeStyle := g.Styles.Dimmed
	if len(g.Points) < g.MaxPoints {
		addStyle = lipgloss.NewStyle().Foreground(g.Styles.Theme.Success)
	}
	if len(g.Points) > g.MinPoints && g.SelectedIndex >= 0 {
		removeStyle = lipgloss.NewStyle().Foreground(g.Styles.Theme.Error)
	}

	addZoneID := fmt.Sprintf("%s-add", g.ZoneID())
	removeZoneID := fmt.Sprintf("%s-remove", g.ZoneID())

	addBtn := addStyle.Render("[+]")
	removeBtn := removeStyle.Render("[-]")

	if g.Zones != nil {
		parts = append(parts, g.Zones.Mark(addZoneID, addBtn))
		parts = append(parts, g.Zones.Mark(removeZoneID, removeBtn))
	} else {
		parts = append(parts, addBtn, removeBtn)
	}

	return strings.Join(parts, "")
}

func (g *GradientEditorComponent) renderWheel() string {
	var out strings.Builder

	wheelLines := strings.Split(g.Wheel.Render(), "\n")
	for row, line := range wheelLines {
		if line == "" {
			continue
		}
		if row > 0 {
			out.WriteString("\n")
		}

		rowZoneID := fmt.Sprintf("%s-wheel-row-%d", g.ZoneID(), row)
		if g.Zones != nil {
			out.WriteString(g.Zones.Mark(rowZoneID, line))
		} else {
			out.WriteString(line)
		}
	}

	return out.String()
}
