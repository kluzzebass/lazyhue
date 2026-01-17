package field

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/kluzzebass/lazyhue/internal/ui"
	"github.com/kluzzebass/lazyhue/internal/ui/component"
	zone "github.com/lrstanley/bubblezone/v2"
)

// FormComponent is a container for form field components.
// It handles navigation between fields and routes events to the focused field.
type FormComponent struct {
	*component.BaseComponent

	// Field components
	fields []component.Component

	// Cursor position (index of focused field)
	cursor int

	// Mouse capture state for drag operations
	captureField int // Field index being captured (-1 if none)

	// Shared resources
	Styles *ui.Styles
	Zones  *zone.Manager

	// Optional section header
	SectionHeader string
}

// NewFormComponent creates a new form component.
func NewFormComponent(styles *ui.Styles, zones *zone.Manager) *FormComponent {
	base := component.NewBaseComponent()
	base.SetFocusState(component.FocusPassive)

	return &FormComponent{
		BaseComponent: base,
		fields:        nil,
		cursor:        0,
		captureField:  -1,
		Styles:        styles,
		Zones:         zones,
	}
}

// SetFields sets the form's field components.
func (f *FormComponent) SetFields(fields []component.Component) {
	f.fields = fields

	// Set parent for all fields
	for _, fld := range fields {
		fld.SetParent(f)
	}

	// Calculate max label width for alignment
	maxLabelWidth := 0
	for _, fld := range fields {
		labelWidth := getFieldLabelWidth(fld)
		if labelWidth > maxLabelWidth {
			maxLabelWidth = labelWidth
		}
	}

	// Apply max label width to all fields
	for _, fld := range fields {
		setFieldMaxLabelWidth(fld, maxLabelWidth)
	}

	// Reset cursor to first focusable field
	f.cursor = 0
	f.moveCursor(0) // This will find the first focusable field

	// Clear capture state
	f.captureField = -1
}

// CurrentField returns the currently focused field, or nil if none.
func (f *FormComponent) CurrentField() component.Component {
	if f.cursor >= 0 && f.cursor < len(f.fields) {
		return f.fields[f.cursor]
	}
	return nil
}

// Children returns the field components.
func (f *FormComponent) Children() []component.Component {
	return f.fields
}

// Update handles events for the form.
func (f *FormComponent) Update(msg tea.Msg) (component.Component, tea.Cmd) {
	if len(f.fields) == 0 {
		return f, nil
	}

	var cmds []tea.Cmd

	// Handle blink tick messages - route to appropriate field
	if blinkMsg, ok := msg.(BlinkTickMsg); ok {
		for _, fld := range f.fields {
			if cw, ok := fld.(*ColorWheelComponent); ok && cw.ID == blinkMsg.FieldID {
				_, cmd := cw.Update(msg)
				if cmd != nil {
					cmds = append(cmds, cmd)
				}
				return f, tea.Batch(cmds...)
			}
		}
	}

	// Handle capture field messages
	if _, ok := msg.(StartCaptureMsg); ok {
		f.captureField = f.cursor
	}
	if _, ok := msg.(EndCaptureMsg); ok {
		f.captureField = -1
	}

	return f, tea.Batch(cmds...)
}

// RouteEvent routes events to the appropriate field.
func (f *FormComponent) RouteEvent(msg tea.Msg) (bool, tea.Cmd) {
	if len(f.fields) == 0 {
		return false, nil
	}

	// If capturing mouse, route all mouse events to capture field
	if f.captureField >= 0 && f.captureField < len(f.fields) {
		switch msg.(type) {
		case tea.MouseMsg, tea.MouseClickMsg, tea.MouseReleaseMsg, tea.MouseMotionMsg:
			if handled, cmd := f.fields[f.captureField].RouteEvent(msg); handled {
				return true, cmd
			}
		}
	}

	// Route to focused field first
	if f.cursor >= 0 && f.cursor < len(f.fields) {
		if handled, cmd := f.fields[f.cursor].RouteEvent(msg); handled {
			return true, cmd
		}
	}

	// Handle form-level navigation
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return f.handleKey(msg)

	case tea.MouseClickMsg:
		return f.handleMouseClick(msg)
	}

	return false, nil
}

func (f *FormComponent) handleKey(msg tea.KeyMsg) (bool, tea.Cmd) {
	// Check if focused field is editing - if so, don't handle navigation
	if f.cursor >= 0 && f.cursor < len(f.fields) {
		currentField := f.fields[f.cursor]
		if isFieldEditing(currentField) {
			return false, nil
		}
	}

	switch msg.String() {
	case "up", "k":
		f.moveCursor(-1)
		return true, nil

	case "down", "j":
		f.moveCursor(1)
		return true, nil
	}

	return false, nil
}

func (f *FormComponent) handleMouseClick(msg tea.MouseClickMsg) (bool, tea.Cmd) {
	if f.Zones == nil {
		return false, nil
	}

	// Check if click is on any field
	for i, fld := range f.fields {
		zoneID := getFieldZoneID(fld)
		if zoneID != "" {
			if z := f.Zones.Get(zoneID); z != nil && z.InBounds(msg) {
				// Focus this field
				f.focusField(i)
				// Route the click to the field
				if handled, cmd := fld.RouteEvent(msg); handled {
					return true, cmd
				}
				return true, nil
			}
		}
	}

	return false, nil
}

// moveCursor moves the cursor by delta, skipping non-focusable fields.
func (f *FormComponent) moveCursor(delta int) {
	if len(f.fields) == 0 {
		return
	}

	// Blur current field
	if f.cursor >= 0 && f.cursor < len(f.fields) {
		f.fields[f.cursor].Blur()
	}

	// Find next focusable field
	newCursor := f.cursor + delta
	for newCursor >= 0 && newCursor < len(f.fields) {
		if f.fields[newCursor].CanFocus() {
			f.cursor = newCursor
			f.fields[f.cursor].Focus()
			return
		}
		if delta >= 0 {
			newCursor++
		} else {
			newCursor--
		}
	}

	// If delta is 0 (initialization), find first focusable field
	if delta == 0 {
		for i, fld := range f.fields {
			if fld.CanFocus() {
				f.cursor = i
				f.fields[f.cursor].Focus()
				return
			}
		}
	}

	// Restore focus to current if no valid target found
	if f.cursor >= 0 && f.cursor < len(f.fields) {
		f.fields[f.cursor].Focus()
	}
}

// focusField focuses the field at the given index.
func (f *FormComponent) focusField(index int) {
	if index < 0 || index >= len(f.fields) {
		return
	}
	if !f.fields[index].CanFocus() {
		return
	}

	// Blur current field
	if f.cursor >= 0 && f.cursor < len(f.fields) {
		f.fields[f.cursor].Blur()
	}

	f.cursor = index
	f.fields[f.cursor].Focus()
}

// View renders all form fields.
func (f *FormComponent) View() string {
	if len(f.fields) == 0 {
		return ""
	}

	var out strings.Builder

	// Optional section header
	if f.SectionHeader != "" {
		out.WriteString(f.Styles.Subtitle.Render(f.SectionHeader))
		out.WriteString("\n")
	}

	// Calculate max label width for alignment
	maxLabelWidth := 0
	for _, fld := range f.fields {
		if cr, ok := fld.(ControlRenderer); ok {
			labelLen := len(cr.FieldLabel())
			if labelLen > maxLabelWidth {
				maxLabelWidth = labelLen
			}
		}
	}

	for i, fld := range f.fields {
		if i > 0 {
			out.WriteString("\n")
		}

		// Check if field implements ControlRenderer for separated label/control
		if cr, ok := fld.(ControlRenderer); ok {
			fieldView := f.renderWithControlRenderer(cr, i == f.cursor && fld.CanFocus(), maxLabelWidth)
			out.WriteString(fieldView)
		} else {
			// Fallback to View() for components without ControlRenderer
			fieldView := fld.View()

			// Add focus indicator for focused field
			if i == f.cursor && fld.CanFocus() {
				lines := strings.Split(fieldView, "\n")
				if len(lines) > 0 {
					firstLine := lines[0]
					if len(firstLine) >= 2 && firstLine[:2] == "  " {
						lines[0] = "> " + firstLine[2:]
					} else {
						lines[0] = "> " + firstLine
					}
					fieldView = strings.Join(lines, "\n")
				}
			}

			out.WriteString(fieldView)
		}
	}

	return out.String()
}

// renderWithControlRenderer renders a field using its ControlRenderer interface
// for proper label/control separation.
func (f *FormComponent) renderWithControlRenderer(cr ControlRenderer, focused bool, maxLabelWidth int) string {
	label := cr.FieldLabel()
	control := cr.ViewControl()
	height := cr.FieldHeight()

	// Handle header case (empty control)
	if control == "" {
		// Headers just render their label with style (they handle their own rendering)
		if header, ok := cr.(*HeaderComponent); ok {
			return header.View()
		}
		return label
	}

	var out strings.Builder

	// Focus indicator prefix
	focusPrefix := "  "
	if focused {
		focusPrefix = "> "
	}

	// Pad label to max width
	paddedLabel := fmt.Sprintf("%-*s", maxLabelWidth, label)

	// Split control into lines for multi-row handling
	controlLines := strings.Split(control, "\n")

	for row := 0; row < height && row < len(controlLines); row++ {
		if row > 0 {
			out.WriteString("\n")
		}

		if row == 0 {
			// First row: focus indicator + label + gap + control
			out.WriteString(focusPrefix)
			out.WriteString(paddedLabel)
			out.WriteString("  ")
			out.WriteString(controlLines[row])
		} else {
			// Subsequent rows: indent + spacer + gap + control
			out.WriteString("  ")
			out.WriteString(strings.Repeat(" ", maxLabelWidth))
			out.WriteString("  ")
			out.WriteString(controlLines[row])
		}
	}

	return out.String()
}

// Helper functions to work with field components

func getFieldLabelWidth(c component.Component) int {
	switch v := c.(type) {
	case *ToggleComponent:
		return len(v.Label)
	case *SliderComponent:
		return len(v.Label)
	case *BrightnessSliderComponent:
		return len(v.Label)
	case *ColorTempSliderComponent:
		return len(v.Label)
	case *SelectComponent:
		return len(v.Label)
	case *RadioComponent:
		return len(v.Label)
	case *TextComponent:
		return len(v.Label)
	case *ColorWheelComponent:
		return len(v.Label)
	case *HSLComponent:
		return len(v.Label)
	case *RGBComponent:
		return len(v.Label)
	case *HeaderComponent:
		return 0 // Headers don't need label alignment
	}
	return 0
}

func setFieldMaxLabelWidth(c component.Component, width int) {
	switch v := c.(type) {
	case *ToggleComponent:
		v.MaxLabelWidth = width
	case *SliderComponent:
		v.MaxLabelWidth = width
	case *BrightnessSliderComponent:
		v.MaxLabelWidth = width
	case *ColorTempSliderComponent:
		v.MaxLabelWidth = width
	case *SelectComponent:
		v.MaxLabelWidth = width
	case *RadioComponent:
		v.MaxLabelWidth = width
	case *TextComponent:
		v.MaxLabelWidth = width
	case *ColorWheelComponent:
		v.MaxLabelWidth = width
	case *HSLComponent:
		v.MaxLabelWidth = width
	case *RGBComponent:
		v.MaxLabelWidth = width
	}
}

func getFieldZoneID(c component.Component) string {
	switch v := c.(type) {
	case *ToggleComponent:
		return v.ZoneID()
	case *SliderComponent:
		return v.ZoneID()
	case *BrightnessSliderComponent:
		return v.ZoneID()
	case *ColorTempSliderComponent:
		return v.ZoneID()
	case *SelectComponent:
		return v.ZoneID()
	case *RadioComponent:
		return v.ZoneID()
	case *TextComponent:
		return v.ZoneID()
	case *ColorWheelComponent:
		return v.ZoneID()
	case *HSLComponent:
		return v.ZoneID()
	case *RGBComponent:
		return v.ZoneID()
	case *HSVComponent:
		return v.ZoneID()
	}
	return ""
}

func isFieldEditing(c component.Component) bool {
	switch v := c.(type) {
	case *ToggleComponent:
		return v.Editing
	case *SliderComponent:
		return v.Editing
	case *BrightnessSliderComponent:
		return v.Editing
	case *ColorTempSliderComponent:
		return v.Editing
	case *SelectComponent:
		return v.IsOpen()
	case *RadioComponent:
		return v.Editing
	case *TextComponent:
		return v.Editing
	case *ColorWheelComponent:
		return v.Editing
	case *HSLComponent:
		return v.Editing
	case *RGBComponent:
		return v.Editing
	case *HSVComponent:
		return v.Editing
	}
	return false
}
