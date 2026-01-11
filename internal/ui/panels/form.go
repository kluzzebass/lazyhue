package panels

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/kluzzebass/lazyhue/internal/ui"
)

// Form handles rendering and interaction for a list of form fields.
// This is the shared form logic used by both PopupPanel and DetailsPanel.
type Form struct {
	Fields   []FormField
	Cursor   int  // Currently focused field
	Editing  bool // Whether we're in edit mode for current field
	LiveMode bool // Whether changes are applied immediately

	// Text inputs for text fields
	TextInputs map[int]*textinput.Model

	// Color wheel state
	ColorWheel     *ColorWheel
	ColorOriginalX float64
	ColorOriginalY float64

	// Select dropdown state
	DropdownOpen   bool
	DropdownScroll int

	// Original values for cancel
	OriginalValue int
	OriginalText  string

	// Mouse capture for dragging
	MouseCaptureIdx  int           // Field index being dragged (-1 if none)
	MouseCaptureType FormFieldType // Type of field being captured
	CachedFieldStart int           // Cached fieldStartY for motion events
	CachedLabelWidth int           // Cached labelWidth for motion events

	// Debouncing
	DebounceTimer *time.Timer
	DebounceField *FormField

	// Callbacks
	OnChange func(field FormField) // Called when a field value changes
	OnSave   func(field FormField) // Called when a field is saved (Enter pressed)

	// Styling
	Styles ui.Styles
}

// NewForm creates a new form with the given fields.
func NewForm(styles ui.Styles) *Form {
	return &Form{
		TextInputs:      make(map[int]*textinput.Model),
		ColorWheel:      NewColorWheel(),
		Styles:          styles,
		LiveMode:        true, // Default to live mode
		MouseCaptureIdx: -1,
	}
}

// SetFields sets the form fields, preserving cursor position if possible.
func (f *Form) SetFields(fields []FormField) {
	prevLen := len(f.Fields)
	f.Fields = fields
	f.TextInputs = make(map[int]*textinput.Model)

	// Reset color wheel position when field count changes (new entity)
	// but preserve editing state to not interrupt user
	if len(fields) != prevLen {
		f.ColorWheel.PosValid = false
	}

	// Initialize text inputs
	for i, field := range f.Fields {
		if field.Type == FormFieldText {
			ti := textinput.New()
			ti.SetValue(field.TextValue)
			f.TextInputs[i] = &ti
		}
	}

	// Preserve cursor if field count unchanged
	if len(fields) != prevLen || f.Cursor >= len(fields) {
		f.Cursor = 0
	}
}

// CurrentField returns the currently focused field, or nil if none.
func (f *Form) CurrentField() *FormField {
	if f.Cursor >= 0 && f.Cursor < len(f.Fields) {
		return &f.Fields[f.Cursor]
	}
	return nil
}

// HandleKey handles keyboard input. Returns true if the input was consumed.
func (f *Form) HandleKey(key string) bool {
	if len(f.Fields) == 0 {
		return false
	}

	field := f.CurrentField()
	if field == nil {
		return false
	}

	// Handle dropdown mode
	if f.DropdownOpen {
		return f.handleDropdownKey(key, field)
	}

	// Handle color wheel edit mode
	if f.Editing && field.Type == FormFieldColor {
		return f.handleColorKey(key, field)
	}

	// Handle text edit mode
	if f.Editing && field.Type == FormFieldText {
		return f.handleTextKey(key, field)
	}

	// Normal navigation
	switch key {
	case "up", "k":
		if f.Cursor > 0 {
			f.Cursor--
			return true
		}
	case "down", "j":
		if f.Cursor < len(f.Fields)-1 {
			f.Cursor++
			return true
		}
	case "enter", " ":
		f.activateField(field)
		return true
	case "left", "h":
		f.adjustLeft(field)
		return true
	case "right", "l":
		f.adjustRight(field)
		return true
	case "esc":
		if f.Editing {
			f.cancelEdit(field)
			return true
		}
	}

	return false
}

// HandleMouse handles mouse input. Returns true if the input was consumed.
func (f *Form) HandleMouse(msg tea.MouseMsg, fieldStartY, labelWidth int) bool {
	if len(f.Fields) == 0 {
		return false
	}

	// Cache these for motion events
	f.CachedFieldStart = fieldStartY
	f.CachedLabelWidth = labelWidth
	valueStartX := labelWidth + 4 // "  " prefix + label + ": "

	// Handle mouse release - clear capture
	if msg.Action == tea.MouseActionRelease {
		f.MouseCaptureIdx = -1
		return false
	}

	// Handle mouse motion for dragging
	if msg.Type == tea.MouseMotion && f.MouseCaptureIdx >= 0 {
		field := &f.Fields[f.MouseCaptureIdx]
		switch field.Type {
		case FormFieldColor:
			if f.Editing {
				// Calculate field row start for color field
				currentRow := 0
				for i := 0; i < f.MouseCaptureIdx; i++ {
					currentRow += f.fieldHeight(i)
				}
				wheelStartX := valueStartX + 4
				wheelY := msg.Y - fieldStartY - currentRow - 1
				col := msg.X - wheelStartX
				if f.ColorWheel.HandleClick(wheelY, col) {
					field.ColorX = f.ColorWheel.ColorX
					field.ColorY = f.ColorWheel.ColorY
					f.debounceSave(field)
				}
			}
		case FormFieldBrightness:
			f.handleBrightnessClick(field, msg.X, valueStartX)
		case FormFieldColorTemp:
			f.handleColorTempClick(field, msg.X, valueStartX)
		}
		return true
	}

	// Find which field was clicked
	clickedRow := msg.Y - fieldStartY
	if clickedRow < 0 {
		return false
	}

	fieldIdx := -1
	currentRow := 0
	for i := range f.Fields {
		height := f.fieldHeight(i)
		if clickedRow >= currentRow && clickedRow < currentRow+height {
			fieldIdx = i
			break
		}
		currentRow += height
	}

	if fieldIdx < 0 || fieldIdx >= len(f.Fields) {
		return false
	}

	field := &f.Fields[fieldIdx]

	switch msg.Type {
	case tea.MouseLeft:
		f.Cursor = fieldIdx

		switch field.Type {
		case FormFieldToggle:
			field.Value = 1 - field.Value
			f.notifyChange(field)

		case FormFieldBrightness:
			f.handleBrightnessClick(field, msg.X, valueStartX)
			f.MouseCaptureIdx = fieldIdx
			f.MouseCaptureType = FormFieldBrightness

		case FormFieldColorTemp:
			f.handleColorTempClick(field, msg.X, valueStartX)
			f.MouseCaptureIdx = fieldIdx
			f.MouseCaptureType = FormFieldColorTemp

		case FormFieldSelect:
			if !f.DropdownOpen {
				f.DropdownOpen = true
				f.DropdownScroll = 0
			}

		case FormFieldColor:
			// Enter edit mode if not already
			if !f.Editing {
				f.Editing = true
				f.ColorWheel.SetOriginal(field.ColorX, field.ColorY)
				f.ColorWheel.SetColor(field.ColorX, field.ColorY)
			}
			// Process the click on the wheel
			// The wheel starts after the label line, so subtract 1 from the row
			wheelStartX := valueStartX + 4
			wheelY := msg.Y - fieldStartY - currentRow - 1
			col := msg.X - wheelStartX
			if f.ColorWheel.HandleClick(wheelY, col) {
				field.ColorX = f.ColorWheel.ColorX
				field.ColorY = f.ColorWheel.ColorY
				f.debounceSave(field)
			}
			// Capture for dragging
			f.MouseCaptureIdx = fieldIdx
			f.MouseCaptureType = FormFieldColor
		}
		return true

	case tea.MouseWheelUp, tea.MouseWheelDown:
		f.Cursor = fieldIdx
		delta := 1
		if msg.Type == tea.MouseWheelDown {
			delta = -1
		}

		switch field.Type {
		case FormFieldBrightness:
			field.Value += delta * 5
			field.Value = clamp(field.Value, 0, 100)
			f.debounceSave(field)

		case FormFieldColorTemp:
			field.Value += delta * 10
			field.Value = clamp(field.Value, field.Min, field.Max)
			f.debounceSave(field)

		case FormFieldSelect:
			field.Value -= delta
			field.Value = clamp(field.Value, 0, len(field.Options)-1)
			f.notifyChange(field)
		}
		return true
	}

	return false
}

// Render renders the form fields and returns the string output.
func (f *Form) Render(width int) string {
	if len(f.Fields) == 0 {
		return ""
	}

	// Find max label width
	maxLabel := 0
	for _, field := range f.Fields {
		if len(field.Label) > maxLabel {
			maxLabel = len(field.Label)
		}
	}

	accentStyle := lipgloss.NewStyle().Foreground(f.Styles.Theme.Accent).Bold(true)

	var out strings.Builder
	for i, field := range f.Fields {
		isFocused := i == f.Cursor

		// Cursor indicator
		cursor := "  "
		if isFocused {
			cursor = "> "
		}

		// Label with padding
		label := field.Label + ":"
		padding := maxLabel - len(field.Label) + 1
		label += strings.Repeat(" ", padding)

		// Value rendering
		valueStr := f.renderFieldValue(&field, isFocused, width-maxLabel-4)

		// Build line with styling
		if isFocused {
			out.WriteString(accentStyle.Render(cursor + label))
		} else {
			out.WriteString(cursor + label)
		}
		out.WriteString(valueStr)
		out.WriteString("\n")
	}

	return out.String()
}

// renderFieldValue renders the value portion of a field.
func (f *Form) renderFieldValue(field *FormField, isFocused bool, maxWidth int) string {
	switch field.Type {
	case FormFieldToggle:
		onLabel := field.ToggleOnLabel
		offLabel := field.ToggleOffLabel
		if onLabel == "" {
			onLabel = "On"
		}
		if offLabel == "" {
			offLabel = "Off"
		}
		if field.Value != 0 {
			return "[●] " + onLabel
		}
		return "[ ] " + offLabel

	case FormFieldBrightness:
		barWidth := 20
		filled := field.Value * barWidth / 100
		bar := strings.Repeat("█", filled) + strings.Repeat("░", barWidth-filled)
		return fmt.Sprintf("%s %3d%%", bar, field.Value)

	case FormFieldColorTemp:
		return f.renderColorTempSlider(field)

	case FormFieldColor:
		if isFocused && f.Editing {
			return f.renderColorWheel(field)
		}
		r, g, b := ui.XyToRGB(field.ColorX, field.ColorY, 1.0)
		style := lipgloss.NewStyle().Foreground(lipgloss.Color(fmt.Sprintf("#%02x%02x%02x", r, g, b)))
		return style.Render("███") + " (Enter to edit)"

	case FormFieldSelect:
		if isFocused && f.DropdownOpen {
			return f.renderDropdown(field)
		}
		currentLabel := "---"
		for _, opt := range field.Options {
			if opt.Value == field.Value {
				currentLabel = opt.Label
				break
			}
		}
		return fmt.Sprintf("[%s] ▼", currentLabel)

	case FormFieldText:
		if f.Editing && isFocused {
			if ti, ok := f.TextInputs[f.Cursor]; ok {
				return ti.View()
			}
		}
		return field.TextValue

	default:
		return fmt.Sprintf("%d", field.Value)
	}
}

// renderColorTempSlider renders the color temperature slider.
func (f *Form) renderColorTempSlider(field *FormField) string {
	barWidth := 20
	if field.Max <= field.Min {
		return fmt.Sprintf("%d", field.Value)
	}

	pos := (field.Value - field.Min) * barWidth / (field.Max - field.Min)
	pos = clamp(pos, 0, barWidth-1)

	var bar strings.Builder
	for i := 0; i < barWidth; i++ {
		mirekAtPos := field.Min + (field.Max-field.Min)*i/barWidth
		r, g, b := ui.MirekToRGB(mirekAtPos)
		style := lipgloss.NewStyle().Foreground(lipgloss.Color(fmt.Sprintf("#%02x%02x%02x", r, g, b)))
		if i == pos {
			bar.WriteString(style.Render("●"))
		} else {
			bar.WriteString(style.Render("─"))
		}
	}
	kelvin := 1000000 / field.Value
	return fmt.Sprintf("%s %dK", bar.String(), kelvin)
}

// renderColorWheel renders the color wheel for editing.
func (f *Form) renderColorWheel(field *FormField) string {
	f.ColorWheel.ColorX = field.ColorX
	f.ColorWheel.ColorY = field.ColorY
	f.ColorWheel.BlinkOn = true

	var out strings.Builder
	out.WriteString("\n")
	for _, line := range strings.Split(f.ColorWheel.Render(), "\n") {
		if line != "" {
			out.WriteString("    " + line + "\n")
		}
	}
	return out.String()
}

// renderDropdown renders the select dropdown.
func (f *Form) renderDropdown(field *FormField) string {
	maxVisible := 5
	start := f.DropdownScroll
	end := min(start+maxVisible, len(field.Options))

	maxLen := 0
	for _, opt := range field.Options {
		if len(opt.Label) > maxLen {
			maxLen = len(opt.Label)
		}
	}

	var out strings.Builder
	out.WriteString("┌" + strings.Repeat("─", maxLen+2) + "┐\n")
	for i := start; i < end; i++ {
		opt := field.Options[i]
		prefix := "  "
		if i == field.Value {
			prefix = "> "
		}
		label := opt.Label + strings.Repeat(" ", maxLen-len(opt.Label))
		out.WriteString(fmt.Sprintf("    │%s%s│\n", prefix, label))
	}
	out.WriteString("    └" + strings.Repeat("─", maxLen+2) + "┘")
	return out.String()
}

// fieldHeight returns the height of a field in rows.
func (f *Form) fieldHeight(idx int) int {
	if idx < 0 || idx >= len(f.Fields) {
		return 1
	}
	field := &f.Fields[idx]
	if field.Type == FormFieldColor && f.Editing && f.Cursor == idx {
		return f.ColorWheel.Height() + 2
	}
	return 1
}

// handleDropdownKey handles keyboard input when dropdown is open.
func (f *Form) handleDropdownKey(key string, field *FormField) bool {
	switch key {
	case "up", "k":
		if field.Value > 0 {
			field.Value--
			if field.Value < f.DropdownScroll {
				f.DropdownScroll = field.Value
			}
		}
	case "down", "j":
		if field.Value < len(field.Options)-1 {
			field.Value++
			if field.Value >= f.DropdownScroll+5 {
				f.DropdownScroll = field.Value - 4
			}
		}
	case "enter", " ":
		f.DropdownOpen = false
		f.notifyChange(field)
	case "esc":
		f.DropdownOpen = false
	default:
		return false
	}
	return true
}

// handleColorKey handles keyboard input when editing color.
func (f *Form) handleColorKey(key string, field *FormField) bool {
	switch key {
	case "esc", "enter":
		f.Editing = false
		return true
	case "left", "h":
		if f.ColorWheel.MoveLeft() {
			field.ColorX = f.ColorWheel.ColorX
			field.ColorY = f.ColorWheel.ColorY
			f.notifyChange(field)
		}
		return true
	case "right", "l":
		if f.ColorWheel.MoveRight() {
			field.ColorX = f.ColorWheel.ColorX
			field.ColorY = f.ColorWheel.ColorY
			f.notifyChange(field)
		}
		return true
	case "up", "k":
		if f.ColorWheel.MoveUp() {
			field.ColorX = f.ColorWheel.ColorX
			field.ColorY = f.ColorWheel.ColorY
			f.notifyChange(field)
		}
		return true
	case "down", "j":
		if f.ColorWheel.MoveDown() {
			field.ColorX = f.ColorWheel.ColorX
			field.ColorY = f.ColorWheel.ColorY
			f.notifyChange(field)
		}
		return true
	}
	return false
}

// handleTextKey handles keyboard input when editing text.
func (f *Form) handleTextKey(key string, field *FormField) bool {
	ti, ok := f.TextInputs[f.Cursor]
	if !ok {
		return false
	}

	switch key {
	case "enter":
		field.TextValue = ti.Value()
		f.Editing = false
		f.notifyChange(field)
		return true
	case "esc":
		f.Editing = false
		return true
	default:
		var cmd tea.Cmd
		*ti, cmd = ti.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)})
		_ = cmd
		field.TextValue = ti.Value()
		return true
	}
}

// activateField enters edit mode or toggles the field.
func (f *Form) activateField(field *FormField) {
	switch field.Type {
	case FormFieldToggle:
		field.Value = 1 - field.Value
		f.notifyChange(field)

	case FormFieldSelect:
		f.DropdownOpen = true
		f.DropdownScroll = 0

	case FormFieldColor:
		f.ColorWheel.SetOriginal(field.ColorX, field.ColorY)
		f.ColorWheel.SetColor(field.ColorX, field.ColorY)
		f.Editing = true

	case FormFieldText:
		f.OriginalText = field.TextValue
		f.Editing = true
		if ti, ok := f.TextInputs[f.Cursor]; ok {
			ti.Focus()
		}

	case FormFieldBrightness, FormFieldColorTemp:
		// Already adjusted with arrows, enter just saves
		f.notifyChange(field)
	}
}

// adjustLeft adjusts the field value left/down.
func (f *Form) adjustLeft(field *FormField) {
	switch field.Type {
	case FormFieldBrightness:
		field.Value = max(0, field.Value-5)
		f.debounceSave(field)
	case FormFieldColorTemp:
		field.Value = max(field.Min, field.Value-10)
		f.debounceSave(field)
	case FormFieldSelect:
		if field.Value > 0 {
			field.Value--
			f.notifyChange(field)
		}
	}
}

// adjustRight adjusts the field value right/up.
func (f *Form) adjustRight(field *FormField) {
	switch field.Type {
	case FormFieldBrightness:
		field.Value = min(100, field.Value+5)
		f.debounceSave(field)
	case FormFieldColorTemp:
		field.Value = min(field.Max, field.Value+10)
		f.debounceSave(field)
	case FormFieldSelect:
		if field.Value < len(field.Options)-1 {
			field.Value++
			f.notifyChange(field)
		}
	}
}

// cancelEdit cancels the current edit and restores original value.
func (f *Form) cancelEdit(field *FormField) {
	switch field.Type {
	case FormFieldText:
		field.TextValue = f.OriginalText
	case FormFieldColor:
		f.ColorWheel.RestoreOriginal()
		field.ColorX = f.ColorWheel.ColorX
		field.ColorY = f.ColorWheel.ColorY
	}
	f.Editing = false
}

// handleBrightnessClick handles mouse click on brightness slider.
func (f *Form) handleBrightnessClick(field *FormField, mouseX, sliderStartX int) {
	sliderWidth := 20
	clickPos := mouseX - sliderStartX
	clickPos = clamp(clickPos, 0, sliderWidth-1)
	field.Value = clickPos * 100 / sliderWidth
	f.debounceSave(field)
}

// handleColorTempClick handles mouse click on color temp slider.
func (f *Form) handleColorTempClick(field *FormField, mouseX, sliderStartX int) {
	sliderWidth := 20
	clickPos := mouseX - sliderStartX
	clickPos = clamp(clickPos, 0, sliderWidth-1)
	field.Value = field.Min + clickPos*(field.Max-field.Min)/sliderWidth
	f.debounceSave(field)
}

// notifyChange calls the OnChange callback if set.
func (f *Form) notifyChange(field *FormField) {
	if f.OnChange != nil {
		f.OnChange(*field)
	}
}

// debounceSave debounces save calls for rapid adjustments.
func (f *Form) debounceSave(field *FormField) {
	if f.DebounceTimer != nil {
		f.DebounceTimer.Stop()
	}

	fieldCopy := *field
	f.DebounceField = &fieldCopy

	f.DebounceTimer = time.AfterFunc(50*time.Millisecond, func() {
		if f.OnChange != nil {
			f.OnChange(fieldCopy)
		}
	})
}

// clamp constrains a value to a range.
func clamp(v, minV, maxV int) int {
	if v < minV {
		return minV
	}
	if v > maxV {
		return maxV
	}
	return v
}

// min returns the minimum of two ints.
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// max returns the maximum of two ints.
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// ToggleBlink toggles the blink state for the color wheel indicator.
func (f *Form) ToggleBlink() {
	f.ColorWheel.BlinkOn = !f.ColorWheel.BlinkOn
}

// IsEditingColor returns true if we're currently editing a color field.
func (f *Form) IsEditingColor() bool {
	if !f.Editing {
		return false
	}
	field := f.CurrentField()
	return field != nil && field.Type == FormFieldColor
}

// Unused but kept for compatibility
var _ = math.Pi
