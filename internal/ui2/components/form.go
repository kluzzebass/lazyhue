// Package components provides reusable UI components for the v2 UI.
package components

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/v2/textinput"
	tea "github.com/charmbracelet/bubbletea/v2"
	zone "github.com/lrstanley/bubblezone/v2"

	"github.com/kluzzebass/lazyhue/internal/ui2"
)

// Form handles rendering and interaction for a list of form fields.
type Form struct {
	Fields  []FormField
	Cursor  int  // Currently focused field
	Editing bool // Whether we're in edit mode for current field

	// Text inputs for text fields
	TextInputs map[int]*textinput.Model

	// Color wheel state
	ColorWheel     *ColorWheel
	ColorOriginalX float64
	ColorOriginalY float64

	// Select dropdown state
	DropdownOpen   bool
	DropdownScroll int
	DropdownCursor int // Current cursor position in dropdown (index into Options)

	// Radio button edit state
	RadioEditCursor    int
	RadioOriginalValue int

	// HSL/RGB slider focus state (field index -> slider focus: 0,1,2)
	HSLSliderFocus    map[int]int
	RGBSliderFocus    map[int]int
	HSLOriginalValues map[int]struct{ Hue, Sat, Light int }
	RGBOriginalValues map[int]struct{ Red, Green, Blue int }

	// Original values for cancel
	OriginalText string

	// Mouse capture for dragging
	MouseCaptureIdx  int           // Field index being dragged (-1 if none)
	MouseCaptureType FormFieldType // Type of field being captured

	// Callbacks
	OnChange func(field FormField) // Called when a field value changes

	// Styling and zones
	Styles *ui2.Styles
	Zones  *zone.Manager

	// Blink timer
	blinkTimerActive bool
	blinkTimerScheduled bool // Track if timer is already scheduled to avoid duplicates
}

// blinkTickMsg is sent by the blink timer to toggle the indicator.
type blinkTickMsg struct{}

// BlinkTickMsg is the exported type for blink tick messages (for use in app2).
type BlinkTickMsg = blinkTickMsg

// NewForm creates a new form.
func NewForm(styles *ui2.Styles, zones *zone.Manager) *Form {
	return &Form{
		TextInputs:        make(map[int]*textinput.Model),
		ColorWheel:        NewColorWheel(),
		Styles:            styles,
		Zones:             zones,
		MouseCaptureIdx:   -1,
		HSLSliderFocus:    make(map[int]int),
		RGBSliderFocus:    make(map[int]int),
		HSLOriginalValues: make(map[int]struct{ Hue, Sat, Light int }),
		RGBOriginalValues: make(map[int]struct{ Red, Green, Blue int }),
	}
}

// SetFields sets the form fields, preserving cursor position if possible.
func (f *Form) SetFields(fields []FormField) {
	prevLen := len(f.Fields)
	f.Fields = fields
	f.TextInputs = make(map[int]*textinput.Model)

	// Reset color wheel position when field count changes
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

// Update handles messages for the form.
func (f *Form) Update(msg tea.Msg) (*Form, tea.Cmd) {
	if len(f.Fields) == 0 {
		return f, nil
	}

	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case blinkTickMsg:
		// Toggle blink state if timer is active and we're editing a color field
		if f.blinkTimerActive && f.Editing && f.Cursor < len(f.Fields) {
			if f.Fields[f.Cursor].Type == FormFieldColor {
				f.ColorWheel.BlinkOn = !f.ColorWheel.BlinkOn
				f.blinkTimerScheduled = false // Timer just fired, can schedule next one
				// Schedule next tick
				cmds = append(cmds, tea.Tick(500*time.Millisecond, func(t time.Time) tea.Msg {
					return blinkTickMsg{}
				}))
				f.blinkTimerScheduled = true
			} else {
				// Not editing color anymore, stop timer
				f.blinkTimerActive = false
				f.blinkTimerScheduled = false
			}
		}
		return f, tea.Batch(cmds...)

	case tea.KeyMsg:
		wasEditingColor := f.Editing && f.Cursor < len(f.Fields) && f.Fields[f.Cursor].Type == FormFieldColor
		if f.handleKey(msg) {
			// Check if we just entered edit mode for color field
			isNowEditingColor := f.Editing && f.Cursor < len(f.Fields) && f.Fields[f.Cursor].Type == FormFieldColor
			if !wasEditingColor && isNowEditingColor && f.blinkTimerActive && !f.blinkTimerScheduled {
				// Start blink timer (only if not already scheduled)
				cmds = append(cmds, tea.Tick(500*time.Millisecond, func(t time.Time) tea.Msg {
					return blinkTickMsg{}
				}))
				f.blinkTimerScheduled = true
			}
			return f, tea.Batch(cmds...)
		}

	case tea.MouseClickMsg:
		wasEditingColor := f.Editing && f.Cursor < len(f.Fields) && f.Fields[f.Cursor].Type == FormFieldColor
		if f.handleMouseClick(msg) {
			// Check if we just entered edit mode for color field
			isNowEditingColor := f.Editing && f.Cursor < len(f.Fields) && f.Fields[f.Cursor].Type == FormFieldColor
			if !wasEditingColor && isNowEditingColor && f.blinkTimerActive && !f.blinkTimerScheduled {
				// Start blink timer (only if not already scheduled)
				cmds = append(cmds, tea.Tick(500*time.Millisecond, func(t time.Time) tea.Msg {
					return blinkTickMsg{}
				}))
				f.blinkTimerScheduled = true
			}
			return f, tea.Batch(cmds...)
		}
	}

	// Update text inputs if in edit mode
	if f.Editing {
		if field := f.CurrentField(); field != nil && field.Type == FormFieldText {
			if ti, ok := f.TextInputs[f.Cursor]; ok {
				var cmd tea.Cmd
				*ti, cmd = ti.Update(msg)
				field.TextValue = ti.Value()
				cmds = append(cmds, cmd)
			}
		}
	}

	return f, tea.Batch(cmds...)
}

// handleKey handles keyboard input.
func (f *Form) handleKey(msg tea.KeyMsg) bool {
	field := f.CurrentField()
	if field == nil {
		return false
	}

	keyStr := msg.String()

	// Handle dropdown mode
	if f.DropdownOpen {
		return f.handleDropdownKey(keyStr, field)
	}

	// Handle color wheel edit mode
	if f.Editing && field.Type == FormFieldColor {
		return f.handleColorKey(keyStr, field)
	}

	// Handle text edit mode
	if f.Editing && field.Type == FormFieldText {
		return f.handleTextKey(keyStr, field)
	}

	// Handle radio button edit mode
	if f.Editing && field.Type == FormFieldRadio {
		return f.handleRadioKey(keyStr, field)
	}

	// Handle HSL edit mode
	if f.Editing && field.Type == FormFieldHSL {
		return f.handleHSLKey(keyStr, field)
	}

	// Handle RGB edit mode
	if f.Editing && field.Type == FormFieldRGB {
		return f.handleRGBKey(keyStr, field)
	}

	// Normal navigation
	switch keyStr {
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

// handleMouseClick handles mouse click input using zones.
func (f *Form) handleMouseClick(msg tea.MouseClickMsg) bool {
	if f.Zones == nil {
		return false
	}

	// Check zones to find which field was clicked
	for i := range f.Fields {
		field := &f.Fields[i]
		zoneID := ui2.FormFieldZone(field.ID)
		if f.Zones.Get(zoneID).InBounds(msg) {
			return f.handleFieldClick(i, field, msg)
		}
	}

	return false
}

// handleFieldClick handles a mouse click on a field.
func (f *Form) handleFieldClick(fieldIdx int, field *FormField, msg tea.MouseClickMsg) bool {
	f.Cursor = fieldIdx

	if msg.Button == tea.MouseLeft {
		switch field.Type {
		case FormFieldToggle:
			field.Value = 1 - field.Value
			f.notifyChange(*field)

		case FormFieldBrightness, FormFieldColorTemp:
			// Slider clicks need coordinate calculation
			// This will be handled by special zone marking for slider tracks
			f.MouseCaptureIdx = fieldIdx
			f.MouseCaptureType = field.Type

		case FormFieldSelect:
			if !f.DropdownOpen {
				f.DropdownOpen = true
				f.DropdownScroll = 0
			}

		case FormFieldColor:
			// Enter edit mode if not already
			if !f.Editing {
				f.Editing = true
				f.blinkTimerActive = true
				f.ColorWheel.SetOriginal(field.ColorX, field.ColorY)
				f.ColorWheel.SetColor(field.ColorX, field.ColorY)
			}
			f.MouseCaptureIdx = fieldIdx
			f.MouseCaptureType = FormFieldColor
		}
		return true
	}

	return false
}

// handleDropdownKey handles keyboard input when dropdown is open.
func (f *Form) handleDropdownKey(keyStr string, field *FormField) bool {
	switch keyStr {
	case "up", "k":
		if f.DropdownCursor > 0 {
			f.DropdownCursor--
			f.ensureDropdownCursorVisible()
		}
	case "down", "j":
		if f.DropdownCursor < len(field.Options)-1 {
			f.DropdownCursor++
			f.ensureDropdownCursorVisible()
		}
	case "enter", " ":
		// Select current option
		if f.DropdownCursor >= 0 && f.DropdownCursor < len(field.Options) {
			field.Value = field.Options[f.DropdownCursor].Value
			f.notifyChange(*field)
		}
		f.DropdownOpen = false
	case "esc":
		f.DropdownOpen = false
	default:
		return false
	}
	return true
}

// handleColorKey handles keyboard input when editing color.
func (f *Form) handleColorKey(keyStr string, field *FormField) bool {
	switch keyStr {
	case "esc", "enter":
		f.Editing = false
		f.blinkTimerActive = false
		return true
	case "left", "h":
		if f.ColorWheel.MoveLeft() {
			field.ColorX = f.ColorWheel.ColorX
			field.ColorY = f.ColorWheel.ColorY
			f.notifyChange(*field)
		}
		return true
	case "right", "l":
		if f.ColorWheel.MoveRight() {
			field.ColorX = f.ColorWheel.ColorX
			field.ColorY = f.ColorWheel.ColorY
			f.notifyChange(*field)
		}
		return true
	case "up", "k":
		if f.ColorWheel.MoveUp() {
			field.ColorX = f.ColorWheel.ColorX
			field.ColorY = f.ColorWheel.ColorY
			f.notifyChange(*field)
		}
		return true
	case "down", "j":
		if f.ColorWheel.MoveDown() {
			field.ColorX = f.ColorWheel.ColorX
			field.ColorY = f.ColorWheel.ColorY
			f.notifyChange(*field)
		}
		return true
	}
	return false
}

// handleTextKey handles keyboard input when editing text.
func (f *Form) handleTextKey(keyStr string, field *FormField) bool {
	switch keyStr {
	case "enter":
		f.Editing = false
		f.notifyChange(*field)
		return true
	case "esc":
		field.TextValue = f.OriginalText
		f.Editing = false
		return true
	}
	// Other keys are handled by textinput.Update
	return false
}

// activateField enters edit mode or toggles the field.
func (f *Form) activateField(field *FormField) {
	switch field.Type {
	case FormFieldToggle:
		field.Value = 1 - field.Value
		f.notifyChange(*field)

	case FormFieldSelect:
		f.DropdownOpen = true
		// Find current selection index
		f.DropdownCursor = 0
		for i, opt := range field.Options {
			if opt.Value == field.Value {
				f.DropdownCursor = i
				break
			}
		}
		// Ensure cursor is visible (scroll to show selected item)
		f.ensureDropdownCursorVisible()

	case FormFieldColor:
		f.ColorWheel.SetOriginal(field.ColorX, field.ColorY)
		// Only reset position if colors don't match (indicating we need to recalculate)
		// Otherwise preserve the existing position to avoid jumping
		if f.ColorWheel.ColorX != field.ColorX || f.ColorWheel.ColorY != field.ColorY {
			f.ColorWheel.PosValid = false // Reset so SetColor calculates position from color
			f.ColorWheel.SetColor(field.ColorX, field.ColorY)
		}
		// Sync color values even if position is preserved
		f.ColorWheel.ColorX = field.ColorX
		f.ColorWheel.ColorY = field.ColorY
		f.ColorWheel.BlinkOn = false // Start with indicator visible (dark bg), timer will toggle for blinking
		f.Editing = true
		f.blinkTimerActive = true

	case FormFieldText:
		f.OriginalText = field.TextValue
		f.Editing = true
		if ti, ok := f.TextInputs[f.Cursor]; ok {
			ti.Focus()
		}

	case FormFieldBrightness, FormFieldColorTemp, FormFieldSlider:
		// Already adjusted with arrows, enter just saves
		f.notifyChange(*field)

	case FormFieldRadio:
		// Enter edit mode for radio
		f.RadioOriginalValue = field.Value
		f.RadioEditCursor = 0
		for i, opt := range field.Options {
			if opt.Value == field.Value {
				f.RadioEditCursor = i
				break
			}
		}
		f.Editing = true

	case FormFieldHSL:
		// Enter edit mode for HSL
		f.HSLOriginalValues[f.Cursor] = struct{ Hue, Sat, Light int }{
			Hue:   field.Hue,
			Sat:   field.Saturation,
			Light: field.Lightness,
		}
		f.HSLSliderFocus[f.Cursor] = 0
		f.Editing = true

	case FormFieldRGB:
		// Enter edit mode for RGB
		f.RGBOriginalValues[f.Cursor] = struct{ Red, Green, Blue int }{
			Red:   field.Red,
			Green: field.Green,
			Blue:  field.Blue,
		}
		f.RGBSliderFocus[f.Cursor] = 0
		f.Editing = true
	}
}

// adjustLeft adjusts the field value left/down.
func (f *Form) adjustLeft(field *FormField) {
	switch field.Type {
	case FormFieldBrightness:
		field.Value = max(0, field.Value-5)
		f.debounceSave(field)
	case FormFieldColorTemp:
		// Left = warm = increase mirek
		// Step size: (Max - Min) / (barWidth - 1) for even distribution
		barWidth := 20
		if barWidth > 1 {
			step := (field.Max - field.Min) / (barWidth - 1)
			if field.Value < field.Max {
				field.Value = min(field.Max, field.Value+step)
				f.debounceSave(field)
			}
		}
	case FormFieldSlider:
		if field.Value > field.Min {
			field.Value = max(field.Min, field.Value-1)
			f.debounceSave(field)
		}
	case FormFieldSelect:
		if field.Value > 0 {
			field.Value--
			f.notifyChange(*field)
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
		// Right = cool = decrease mirek
		// Step size: (Max - Min) / (barWidth - 1) for even distribution
		barWidth := 20
		if barWidth > 1 {
			step := (field.Max - field.Min) / (barWidth - 1)
			if field.Value > field.Min {
				field.Value = max(field.Min, field.Value-step)
				f.debounceSave(field)
			}
		}
	case FormFieldSlider:
		if field.Value < field.Max {
			field.Value = min(field.Max, field.Value+1)
			f.debounceSave(field)
		}
	case FormFieldSelect:
		if field.Value < len(field.Options)-1 {
			field.Value++
			f.notifyChange(*field)
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
	case FormFieldRadio:
		field.Value = f.RadioOriginalValue
	case FormFieldHSL:
		if orig, ok := f.HSLOriginalValues[f.Cursor]; ok {
			field.Hue = orig.Hue
			field.Saturation = orig.Sat
			field.Lightness = orig.Light
		}
	case FormFieldRGB:
		if orig, ok := f.RGBOriginalValues[f.Cursor]; ok {
			field.Red = orig.Red
			field.Green = orig.Green
			field.Blue = orig.Blue
		}
	}
	f.blinkTimerActive = false
	f.blinkTimerScheduled = false
	f.Editing = false
}

// notifyChange calls the OnChange callback if set.
func (f *Form) notifyChange(field FormField) {
	if f.OnChange != nil {
		f.OnChange(field)
	}
}

// debounceSave immediately calls OnChange (debouncing is handled in service layer).
func (f *Form) debounceSave(field *FormField) {
	// Service layer handles debouncing, so we can call OnChange immediately
	if f.OnChange != nil {
		f.OnChange(*field)
	}
}

// View renders the form fields.
func (f *Form) View() string {
	if len(f.Fields) == 0 {
		return ""
	}

	var content strings.Builder

	// Find max label width
	maxLabelWidth := 0
	for _, field := range f.Fields {
		if len(field.Label) > maxLabelWidth {
			maxLabelWidth = len(field.Label)
		}
	}

	// Render each field
	for i, field := range f.Fields {
		isFocused := i == f.Cursor

		// Determine how many rows this field needs
		rows := f.fieldHeight(i)

		// Special handling for dropdown - render inline with label
		if field.Type == FormFieldSelect && f.DropdownOpen && isFocused {
			// Label on first line
			cursor := "> "
			labelStr := field.Label + ":"
			padding := maxLabelWidth - len(field.Label) + 1
			labelPadded := labelStr + strings.Repeat(" ", padding)
			label := cursor + f.Styles.Accent.Render(labelPadded)

			// Dropdown starts on same line (inline)
			dropdownContent := f.renderSelectDropdown(&field, f.DropdownScroll, f.DropdownCursor)
			lines := strings.Split(dropdownContent, "\n")
			if len(lines) > 0 {
				// First line of dropdown goes on same line as label
				content.WriteString(label + lines[0] + "\n")
				// Subsequent lines indented to value column
				valueStart := 2 + maxLabelWidth + 2 // "> " + label + ": " = 2 + maxLabelWidth + 2
				indent := strings.Repeat(" ", valueStart)
				for _, line := range lines[1:] {
					if line != "" {
						content.WriteString(indent + line + "\n")
					}
				}
			}
			continue
		}

		// Special handling for color wheel - always visible
		if field.Type == FormFieldColor {
			// Label row with cursor indicator
			cursor := "  "
			if isFocused {
				cursor = "> "
			}
			labelStr := field.Label + ":"
			padding := maxLabelWidth - len(field.Label) + 1
			labelPadded := labelStr + strings.Repeat(" ", padding)
			var label string
			if isFocused {
				label = cursor + f.Styles.Accent.Render(labelPadded)
			} else {
				label = cursor + f.Styles.Label.Render(labelPadded)
			}
			content.WriteString(label + "\n")

			// Always sync color wheel with field color
			if f.Editing && isFocused {
				// When editing and focused
				if f.ColorWheel.PosValid {
					// Position is valid (user has interacted) - preserve position, just sync color values
					f.ColorWheel.ColorX = field.ColorX
					f.ColorWheel.ColorY = field.ColorY
					// Don't set BlinkOn here - let it toggle naturally or remain as set
				} else {
					// Position not yet set - initialize from field color
					f.ColorWheel.SetColor(field.ColorX, field.ColorY)
					// Don't set BlinkOn here - let it toggle naturally or remain as set
				}
			} else {
				// Not editing or not focused - sync color values but preserve position
				// This prevents the indicator from jumping when exiting edit mode
				f.ColorWheel.ColorX = field.ColorX
				f.ColorWheel.ColorY = field.ColorY
				// Ensure indicator is visible (not blinking) when not editing
				f.ColorWheel.BlinkOn = false
				// Only recalculate position if PosValid is false (shouldn't happen here, but just in case)
				if !f.ColorWheel.PosValid {
					f.ColorWheel.SetColor(field.ColorX, field.ColorY)
				}
			}

			// Don't force BlinkOn - let it be set by activateField or previous state
			// The indicator will be visible when BlinkOn=false (shows bg color) and invisible when BlinkOn=true (shows block color)
			// A blink timer (not yet implemented) should toggle this

			wheelLines := strings.Split(f.ColorWheel.Render(), "\n")
			valueStart := 2 + maxLabelWidth + 2 // "> " + label + ": " = 2 + maxLabelWidth + 2
			indent := strings.Repeat(" ", valueStart)
			for _, line := range wheelLines {
				if line != "" {
					content.WriteString(indent + line + "\n")
				}
			}
			continue
		}

		for row := 0; row < rows; row++ {
			// Label only on first row
			var label string
			if row == 0 {
				// Cursor indicator
				cursor := "  "
				if isFocused {
					cursor = "> "
				}

				labelStr := field.Label + ":"
				padding := maxLabelWidth - len(field.Label) + 1
				labelPadded := labelStr + strings.Repeat(" ", padding)

				// Style label with accent color when focused
				if isFocused {
					label = cursor + f.Styles.Accent.Render(labelPadded)
				} else {
					label = cursor + f.Styles.Label.Render(labelPadded)
				}
			} else {
				// Subsequent rows: indent to match label width (include cursor space)
				label = strings.Repeat(" ", maxLabelWidth+4) // "> " + label + ": " = 4 extra spaces
			}

			// Render field value
			var valueStr string
			if field.Type == FormFieldText && f.Editing && isFocused {
				if ti, ok := f.TextInputs[i]; ok {
					valueStr = ti.View()
				} else {
					valueStr = field.TextValue
				}
			} else if field.Type == FormFieldSelect && f.DropdownOpen && isFocused {
				// This branch should never be reached (handled in special case above)
				valueStr = f.renderSelectDropdown(&field, f.DropdownScroll, f.DropdownCursor)
			} else if field.Type == FormFieldRadio {
				// Radio needs edit mode highlighting
				valueStr = f.renderRadioWithEditMode(&field, isFocused, row)
			} else if field.Type == FormFieldHSL {
				// HSL needs slider focus for highlighting
				sliderFocus := f.HSLSliderFocus[i]
				valueStr = f.renderHSLWithFocus(&field, isFocused, row, sliderFocus)
			} else if field.Type == FormFieldRGB {
				// RGB needs slider focus for highlighting
				sliderFocus := f.RGBSliderFocus[i]
				valueStr = f.renderRGBWithFocus(&field, isFocused, row, sliderFocus)
			} else {
				valueStr = RenderFieldValue(&field, isFocused, f.Styles, row)
			}

			// Mark with zone
			zoneID := ui2.FormFieldZone(field.ID)
			if f.Zones != nil {
				valueStr = f.Zones.Mark(zoneID, valueStr)
			}

			line := label + valueStr + "\n"
			content.WriteString(line)
		}
	}

	return content.String()
}

// fieldHeight returns the height of a field in rows.
func (f *Form) fieldHeight(idx int) int {
	if idx < 0 || idx >= len(f.Fields) {
		return 1
	}
	field := &f.Fields[idx]
	if field.Type == FormFieldHSL || field.Type == FormFieldRGB {
		return 3
	}
	if field.Type == FormFieldRadio && field.Vertical {
		return len(field.Options)
	}
	if field.Type == FormFieldColor {
		return f.ColorWheel.Height() + 1
	}
	if field.Type == FormFieldSelect && f.DropdownOpen && f.Cursor == idx {
		return 6 // 1 for label + 5 for dropdown (maxVisible)
	}
	return 1
}

// renderSelectDropdown renders the dropdown menu for a select field.
func (f *Form) renderSelectDropdown(field *FormField, scroll int, cursor int) string {
	maxVisible := 5
	start := scroll
	end := start + maxVisible
	if end > len(field.Options) {
		end = len(field.Options)
	}
	if start < 0 {
		start = 0
	}

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
		if i == cursor {
			prefix = "> "
		}
		label := opt.Label + strings.Repeat(" ", maxLen-len(opt.Label))
		out.WriteString(fmt.Sprintf("│%s%s│\n", prefix, label))
	}
	out.WriteString("└" + strings.Repeat("─", maxLen+2) + "┘")
	return out.String()
}

// renderHSLWithFocus renders HSL with slider focus highlighting.
func (f *Form) renderHSLWithFocus(field *FormField, isFocused bool, subRow int, sliderFocus int) string {
	result := renderHSL(field, isFocused, f.Styles, subRow)
	if isFocused && f.Editing && sliderFocus == subRow {
		// Highlight the row label
		rowLabels := []string{"H", "S", "L"}
		if subRow < 3 {
			oldLabel := rowLabels[subRow] + ":"
			newLabel := f.Styles.Accent.Render(rowLabels[subRow]) + ":"
			result = strings.Replace(result, oldLabel, newLabel, 1)
		}
	}
	return result
}

// renderRGBWithFocus renders RGB with slider focus highlighting.
func (f *Form) renderRGBWithFocus(field *FormField, isFocused bool, subRow int, sliderFocus int) string {
	result := renderRGB(field, isFocused, f.Styles, subRow)
	if isFocused && f.Editing && sliderFocus == subRow {
		// Highlight the row label
		rowLabels := []string{"R", "G", "B"}
		if subRow < 3 {
			oldLabel := rowLabels[subRow] + ":"
			newLabel := f.Styles.Accent.Render(rowLabels[subRow]) + ":"
			result = strings.Replace(result, oldLabel, newLabel, 1)
		}
	}
	return result
}

// ensureDropdownCursorVisible ensures the dropdown cursor is within the visible scroll area.
func (f *Form) ensureDropdownCursorVisible() {
	maxVisible := 5
	if f.DropdownCursor < f.DropdownScroll {
		f.DropdownScroll = f.DropdownCursor
	} else if f.DropdownCursor >= f.DropdownScroll+maxVisible {
		f.DropdownScroll = f.DropdownCursor - maxVisible + 1
	}
}

// renderRadioWithEditMode renders radio buttons with edit mode highlighting.
func (f *Form) renderRadioWithEditMode(field *FormField, isFocused bool, subRow int) string {
	result := renderRadio(field, isFocused, f.Styles, subRow)
	if isFocused && f.Editing {
		// Highlight the selected option
		if field.Vertical {
			if subRow < len(field.Options) {
				opt := field.Options[subRow]
				if opt.Value == field.Value {
					// Highlight the entire option string
					result = f.Styles.Accent.Render(result)
				}
			}
		} else {
			// Horizontal: highlight the selected option
			if subRow == 0 { // Only render once for horizontal
				parts := strings.Split(result, "  ")
				for i, part := range parts {
					if i < len(field.Options) && field.Options[i].Value == field.Value {
						parts[i] = f.Styles.Accent.Render(part)
					}
				}
				result = strings.Join(parts, "  ")
			}
		}
	}
	return result
}
