// Package panels provides UI panel components.
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

// debounceDelay is the delay before sending slider changes to the bridge.
const debounceDelay = 50 * time.Millisecond

// PopupMode defines the type of popup.
type PopupMode int

const (
	PopupModeDisplay PopupMode = iota // Scrollable read-only content
	PopupModeInput                    // Text input with prompt
	PopupModeConfirm                  // Yes/No confirmation
	PopupModeSelect                   // Selection from list of options
	PopupModeForm                     // Form with multiple editable fields
)

// PopupResult is sent when the popup closes.
type PopupResult struct {
	Confirmed bool   // true if user confirmed/submitted, false if cancelled
	Value     string // input value (for Input mode)
	Index     int    // selected index (for Select mode)
}

// SelectOption represents an option in a selection popup.
type SelectOption struct {
	Label string // Display label
	Value string // Value returned in result
}

// FormFieldType defines the type of form field.
type FormFieldType int

const (
	FormFieldToggle     FormFieldType = iota // Boolean toggle (on/off)
	FormFieldSlider                          // Integer slider with min/max
	FormFieldText                            // Text input
	FormFieldBrightness                      // Brightness slider (0-100) with visual bar
	FormFieldColorTemp                       // Color temperature slider (warm to cool)
	FormFieldColor                           // Color picker (XY color space)
	FormFieldSelect                          // Dropdown selection from options
	FormFieldRadio                           // Radio button group (inline options)
	FormFieldHSL                             // HSL color picker (3 sliders: Hue, Saturation, Lightness)
	FormFieldRGB                             // RGB color picker (3 sliders: Red, Green, Blue)
)

// SelectOption represents an option for FormFieldSelect.
type FormSelectOption struct {
	Label string // Display label
	Value int    // Value when selected
}

// FormField represents a single field in a form.
type FormField struct {
	ID           string        // Unique identifier
	Label        string        // Display label
	Type         FormFieldType // Field type
	Value        int           // Current value (0/1 for toggle, actual value for slider/brightness/temp)
	Min          int           // Minimum value (for sliders)
	Max          int           // Maximum value (for sliders)
	Original     int           // Original value (to detect changes)
	TextValue    string        // Current text value (for text fields)
	OriginalText string        // Original text value (to detect changes)

	// Color fields (for FormFieldColor)
	ColorX         float64 // CIE x coordinate (0.0-1.0)
	ColorY         float64 // CIE y coordinate (0.0-1.0)
	OriginalColorX float64 // Original x
	OriginalColorY float64 // Original y
	ColorMode      int     // 0=hue, 1=saturation adjustment mode

	// Select field options
	Options  []FormSelectOption // Available options for select field
	Vertical bool               // For radio buttons: true = vertical layout, false = horizontal

	// HSL fields (for FormFieldHSL)
	Hue            int // 0-360 degrees
	Saturation     int // 0-100 percent
	Lightness      int // 0-100 percent
	OriginalHue    int // Original hue
	OriginalSat    int // Original saturation
	OriginalLight  int // Original lightness
	HSLSliderFocus int // Which slider is focused: 0=hue, 1=sat, 2=light

	// RGB fields (for FormFieldRGB)
	Red            int // 0-255
	Green          int // 0-255
	Blue           int // 0-255
	OriginalRed    int // Original red
	OriginalGreen  int // Original green
	OriginalBlue   int // Original blue
	RGBSliderFocus int // Which slider is focused: 0=red, 1=green, 2=blue

	// Toggle labels (for FormFieldToggle) - if empty, defaults to "On"/"Off"
	ToggleOnLabel  string
	ToggleOffLabel string
}

// PopupPanel is a generic modal dialog.
type PopupPanel struct {
	styles ui.Styles

	// State
	visible bool
	mode    PopupMode
	title   string

	// Display mode
	lines  []string
	scroll int

	// Input mode
	input  textinput.Model
	prompt string

	// Confirm mode
	confirmMsg string
	yesLabel   string
	noLabel    string
	selected   int // 0 = yes, 1 = no

	// Select mode
	selectOptions []SelectOption
	selectCursor  int

	// Form mode
	formFields       []FormField
	formCursor       int  // Which field is focused
	formOnButtons    bool // true when focus is on Save/Cancel buttons
	formBtnIndex     int  // 0 = Save, 1 = Cancel
	formTextInputs   map[int]textinput.Model // Text inputs for FormFieldText fields (indexed by field index)
	formLiveMode     bool                    // true = changes apply immediately, false = apply on Save
	formOnChange     func(field FormField)   // Callback for live mode changes
	formFieldEditing bool                    // true when current field is in edit mode (captures keys)
	debouncePending  *FormField              // Pending field change waiting for debounce
	debounceTime     time.Time               // Timestamp of last change (for debounce)
	mouseCaptureIdx  int                     // Field index that has mouse capture (-1 = none)
	mouseCaptureType FormFieldType           // Type of captured field for validation

	// Select dropdown state
	selectDropdownOpen       bool // Whether a select dropdown is currently open
	selectDropdownCursor     int  // Highlighted option in dropdown
	selectDropdownScroll     int  // Scroll offset for long dropdown lists
	selectDropdownJustOpened bool // Flag to ignore the first mouse event after opening

	// Radio edit state
	radioEditCursor    int // Cursor position when editing radio (index into Options)
	radioOriginalValue int // Original value before editing (for cancel)

	// Color edit state
	colorOriginalX float64 // Original X before editing (for cancel)
	colorOriginalY float64 // Original Y before editing (for cancel)
	colorSelRow    int     // Selected row on disc (screen coordinates)
	colorSelCol    int     // Selected column on disc (screen coordinates)
	colorPosValid  bool    // True if colorSelRow/colorSelCol are valid (have been set by user interaction)
	colorBlinkOn   bool    // Blink state for color wheel indicator

	// HSL/RGB slider capture state
	capturedSliderRow int // Which sub-slider (0,1,2) is captured during drag (-1 = none)

	// Form scrolling
	formScroll int // Scroll offset for form content

	// Sizing
	screenWidth  int
	screenHeight int
	widthRatio   float64
	heightRatio  float64

	// Callbacks
	onClose func(PopupResult)
}

// NewPopupPanel creates a new popup panel.
func NewPopupPanel(styles ui.Styles) *PopupPanel {
	ti := textinput.New()
	ti.Focus()
	ti.CharLimit = 256
	ti.Width = 40

	return &PopupPanel{
		styles:      styles,
		widthRatio:  0.5,
		heightRatio: 0.5,
		input:       ti,
		yesLabel:    "Yes",
		noLabel:     "No",
	}
}

// ShowDisplay shows the popup in display mode with scrollable content.
func (p *PopupPanel) ShowDisplay(title string, content string, onClose func(PopupResult)) {
	p.mode = PopupModeDisplay
	p.title = title
	p.lines = strings.Split(content, "\n")
	p.scroll = 0
	p.onClose = onClose
	p.visible = true
}

// ShowInput shows the popup in input mode.
func (p *PopupPanel) ShowInput(title string, prompt string, defaultValue string, onClose func(PopupResult)) {
	p.mode = PopupModeInput
	p.title = title
	p.prompt = prompt
	p.input.SetValue(defaultValue)
	p.input.CursorEnd()
	p.onClose = onClose
	p.visible = true
}

// ShowConfirm shows the popup in confirm mode.
func (p *PopupPanel) ShowConfirm(title string, message string, onClose func(PopupResult)) {
	p.mode = PopupModeConfirm
	p.title = title
	p.confirmMsg = message
	p.selected = 1 // Default to "No" for safety
	p.onClose = onClose
	p.visible = true
}

// ShowSelect shows the popup in select mode with a list of options.
func (p *PopupPanel) ShowSelect(title string, options []SelectOption, onClose func(PopupResult)) {
	p.mode = PopupModeSelect
	p.title = title
	p.selectOptions = options
	p.selectCursor = 0
	p.onClose = onClose
	p.visible = true
}

// ShowForm shows the popup in form mode with editable fields.
// The onClose callback receives the modified fields when Confirmed is true.
// ShowForm displays a form with Save/Cancel buttons (edit mode).
func (p *PopupPanel) ShowForm(title string, fields []FormField, onClose func(PopupResult, []FormField)) {
	p.showFormInternal(title, fields, onClose, nil, false)
}

// ShowFormLive displays a form in live mode where changes apply immediately.
// onChange is called immediately when values change.
func (p *PopupPanel) ShowFormLive(title string, fields []FormField, onClose func(PopupResult, []FormField), onChange func(field FormField)) {
	p.showFormInternal(title, fields, onClose, onChange, true)
}

func (p *PopupPanel) showFormInternal(title string, fields []FormField, onClose func(PopupResult, []FormField), onChange func(field FormField), startLive bool) {
	p.mode = PopupModeForm
	p.title = title
	p.formOnChange = onChange
	p.formLiveMode = startLive

	// Copy fields so we can track original values
	p.formFields = make([]FormField, len(fields))
	p.formTextInputs = make(map[int]textinput.Model)

	for i, f := range fields {
		f.Original = f.Value
		f.OriginalText = f.TextValue
		f.OriginalColorX = f.ColorX
		f.OriginalColorY = f.ColorY
		p.formFields[i] = f

		// Initialize text input for text fields
		if f.Type == FormFieldText {
			ti := textinput.New()
			ti.Prompt = ""        // Remove default "> " prompt
			ti.Placeholder = ""
			ti.SetValue(f.TextValue)
			ti.CharLimit = 64
			ti.Width = 30
			p.formTextInputs[i] = ti
		}
	}

	p.formCursor = 0
	p.formOnButtons = false
	p.formBtnIndex = 0
	p.formFieldEditing = false  // Not editing any field initially
	p.mouseCaptureIdx = -1      // No mouse capture initially
	p.capturedSliderRow = -1    // No slider row captured
	p.colorPosValid = false     // Color position needs to be calculated from XY initially
	p.selectDropdownOpen = false
	p.selectDropdownCursor = 0
	p.selectDropdownScroll = 0
	p.formScroll = 0            // Reset scroll position for new form

	// Focus and enter edit mode for the first text input if applicable
	if len(p.formFields) > 0 && p.formFields[0].Type == FormFieldText {
		if ti, ok := p.formTextInputs[0]; ok {
			ti.Focus()
			p.formTextInputs[0] = ti
			p.formFieldEditing = true // Enable edit mode so typing works
		}
	}

	p.onClose = func(result PopupResult) {
		// Sync text values back to fields before calling callback
		for i := range p.formFields {
			if p.formFields[i].Type == FormFieldText {
				if ti, ok := p.formTextInputs[i]; ok {
					p.formFields[i].TextValue = ti.Value()
				}
			}
		}
		onClose(result, p.formFields)
	}
	p.visible = true
}

// GetFormFields returns the current form field values.
func (p *PopupPanel) GetFormFields() []FormField {
	return p.formFields
}

// UpdateFormField updates a form field's value by ID without triggering onChange.
// Used for live state sync from external sources (e.g., SSE events).
func (p *PopupPanel) UpdateFormField(id string, value int, colorX, colorY float64) {
	if !p.visible || p.mode != PopupModeForm {
		return
	}
	for i := range p.formFields {
		if p.formFields[i].ID == id {
			p.formFields[i].Value = value
			p.formFields[i].ColorX = colorX
			p.formFields[i].ColorY = colorY
			break
		}
	}
}

// Hide hides the popup.
func (p *PopupPanel) Hide() {
	p.visible = false
}

// IsVisible returns whether the popup is visible.
func (p *PopupPanel) IsVisible() bool {
	return p.visible
}

// SetSize updates screen dimensions.
func (p *PopupPanel) SetSize(width, height int) {
	p.screenWidth = width
	p.screenHeight = height
	p.input.Width = p.contentWidth() - 2
}

// SetRatio sets the popup size ratio (0.0 to 1.0).
func (p *PopupPanel) SetRatio(widthRatio, heightRatio float64) {
	p.widthRatio = widthRatio
	p.heightRatio = heightRatio
}

// ToggleBlink toggles the blink state for color wheel indicator.
func (p *PopupPanel) ToggleBlink() {
	p.colorBlinkOn = !p.colorBlinkOn
}

func (p *PopupPanel) width() int {
	if p.screenWidth <= 0 {
		return 1
	}
	return max(1, int(float64(p.screenWidth)*p.widthRatio))
}

func (p *PopupPanel) height() int {
	if p.screenHeight <= 0 {
		return 1
	}
	return max(1, int(float64(p.screenHeight)*p.heightRatio))
}

func (p *PopupPanel) contentWidth() int {
	return p.width() - 4 // borders + padding
}

func (p *PopupPanel) contentHeight() int {
	h := p.height() - 2 // borders
	if h < 1 {
		return 1
	}
	return h
}

// Width returns the popup width.
func (p *PopupPanel) Width() int {
	return p.width()
}

// Height returns the popup height.
func (p *PopupPanel) Height() int {
	return p.height()
}

// Update handles input.
func (p *PopupPanel) Update(msg tea.Msg) tea.Cmd {
	if !p.visible {
		return nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		return p.handleKey(msg)
	case tea.MouseMsg:
		return p.handleMouse(msg)
	}

	// Update input field if in input mode
	if p.mode == PopupModeInput {
		var cmd tea.Cmd
		p.input, cmd = p.input.Update(msg)
		return cmd
	}

	return nil
}

func (p *PopupPanel) handleKey(msg tea.KeyMsg) tea.Cmd {
	switch p.mode {
	case PopupModeDisplay:
		return p.handleDisplayKey(msg)
	case PopupModeInput:
		return p.handleInputKey(msg)
	case PopupModeConfirm:
		return p.handleConfirmKey(msg)
	case PopupModeSelect:
		return p.handleSelectKey(msg)
	case PopupModeForm:
		return p.handleFormKey(msg)
	}
	return nil
}

func (p *PopupPanel) handleDisplayKey(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "esc", "q", "enter":
		p.close(PopupResult{Confirmed: true})
	case "up", "k":
		if p.scroll > 0 {
			p.scroll--
		}
	case "down", "j":
		if p.scroll < p.maxScroll() {
			p.scroll++
		}
	case "home", "g":
		p.scroll = 0
	case "end", "G":
		p.scroll = p.maxScroll()
	case "pgup":
		p.scroll = max(0, p.scroll-p.contentHeight())
	case "pgdown":
		p.scroll = min(p.maxScroll(), p.scroll+p.contentHeight())
	}
	return nil
}

func (p *PopupPanel) handleInputKey(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "esc":
		p.close(PopupResult{Confirmed: false})
		return nil
	case "enter":
		p.close(PopupResult{Confirmed: true, Value: p.input.Value()})
		return nil
	}

	// Pass to text input
	var cmd tea.Cmd
	p.input, cmd = p.input.Update(msg)
	return cmd
}

func (p *PopupPanel) handleConfirmKey(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "esc", "n":
		p.close(PopupResult{Confirmed: false})
	case "enter":
		p.close(PopupResult{Confirmed: p.selected == 0})
	case "y":
		p.close(PopupResult{Confirmed: true})
	case "left", "h":
		p.selected = 0
	case "right", "l":
		p.selected = 1
	case "tab":
		p.selected = (p.selected + 1) % 2
	}
	return nil
}

func (p *PopupPanel) handleSelectKey(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "esc", "q":
		p.close(PopupResult{Confirmed: false, Index: -1})
	case "enter":
		if len(p.selectOptions) > 0 && p.selectCursor < len(p.selectOptions) {
			p.close(PopupResult{
				Confirmed: true,
				Value:     p.selectOptions[p.selectCursor].Value,
				Index:     p.selectCursor,
			})
		}
	case "up", "k":
		if p.selectCursor > 0 {
			p.selectCursor--
		}
	case "down", "j":
		if p.selectCursor < len(p.selectOptions)-1 {
			p.selectCursor++
		}
	case "home", "g":
		p.selectCursor = 0
	case "end", "G":
		p.selectCursor = len(p.selectOptions) - 1
	}
	return nil
}

func (p *PopupPanel) handleFormKey(msg tea.KeyMsg) tea.Cmd {
	key := msg.String()

	// Handle select dropdown if open
	if p.selectDropdownOpen {
		return p.handleSelectDropdownKey(msg)
	}

	// Handle edit mode for text inputs and vertical radio
	if p.formFieldEditing {
		return p.handleFieldEditMode(msg)
	}

	// Handle escape - close form
	if key == "esc" {
		if p.formLiveMode {
			p.close(PopupResult{Confirmed: true})
		} else {
			p.close(PopupResult{Confirmed: false})
		}
		return nil
	}

	// Handle enter
	if key == "enter" {
		if p.formOnButtons {
			// In live mode, there's only Close (changes already applied)
			if p.formLiveMode {
				p.close(PopupResult{Confirmed: true})
			} else if p.formBtnIndex == 0 {
				p.close(PopupResult{Confirmed: true})
			} else {
				p.close(PopupResult{Confirmed: false})
			}
			return nil
		}
		// Enter on a field - activate editing or perform action
		if p.formCursor < len(p.formFields) {
			field := &p.formFields[p.formCursor]
			switch field.Type {
			case FormFieldText:
				// Enter edit mode for text
				p.formFieldEditing = true
				p.focusCurrentTextField()
				return nil
			case FormFieldRadio:
				// Enter edit mode for radio, set cursor to current value
				p.formFieldEditing = true
				p.radioOriginalValue = field.Value // Save for cancel
				p.radioEditCursor = 0
				for i, opt := range field.Options {
					if opt.Value == field.Value {
						p.radioEditCursor = i
						break
					}
				}
				return nil
			case FormFieldColor:
				// Enter edit mode for color
				p.formFieldEditing = true
				p.colorOriginalX = field.ColorX
				p.colorOriginalY = field.ColorY
				
				// Only calculate screen position from XY if we don't have a valid position
				// (i.e., first time entering edit mode for this color field)
				if !p.colorPosValid {
					// Initialize screen position from current color
					// Must use inverse of the forward conversion (screen→color)
					radiusY := 4
					radiusX := 9
					
					// Convert XY to hue/sat (in XY space)
					hue, sat := ui.XyToHueSat(field.ColorX, field.ColorY)
					
					// The wheel uses: hue = -angle*180/π + 90
					// So: angle = (90 - hue) * π/180
					angleRad := float64(90-hue) * math.Pi / 180
					dist := float64(sat) / 100.0
					xNorm := dist * math.Cos(angleRad)
					yNorm := -dist * math.Sin(angleRad)
					
					p.colorSelCol = radiusX + int(xNorm*float64(radiusX)+0.5)
					p.colorSelRow = radiusY + int(yNorm*float64(radiusY)+0.5)
					
					// Clamp to valid range
					if p.colorSelRow < 0 {
						p.colorSelRow = 0
					}
					if p.colorSelRow > radiusY*2 {
						p.colorSelRow = radiusY * 2
					}
					if p.colorSelCol < 0 {
						p.colorSelCol = 0
					}
					if p.colorSelCol > radiusX*2 {
						p.colorSelCol = radiusX * 2
					}
				}
				return nil
			case FormFieldHSL:
				// Enter edit mode for HSL
				p.formFieldEditing = true
				field.OriginalHue = field.Hue
				field.OriginalSat = field.Saturation
				field.OriginalLight = field.Lightness
				field.HSLSliderFocus = 0 // Start on Hue slider
				return nil
			case FormFieldRGB:
				// Enter edit mode for RGB
				p.formFieldEditing = true
				field.OriginalRed = field.Red
				field.OriginalGreen = field.Green
				field.OriginalBlue = field.Blue
				field.RGBSliderFocus = 0 // Start on Red slider
				return nil
			case FormFieldToggle:
				// Toggle immediately
				field.Value = 1 - field.Value
				p.notifyLiveChange(*field)
			case FormFieldSelect:
				// Open dropdown
				p.openSelectDropdown()
				return nil
			default:
				// Toggle live mode for sliders etc
				if p.formOnChange != nil {
					p.formLiveMode = !p.formLiveMode
				}
			}
		}
		return nil
	}

	// Space on select opens dropdown, toggles for toggle fields
	if key == " " {
		if !p.formOnButtons && p.formCursor < len(p.formFields) {
			field := &p.formFields[p.formCursor]
			if field.Type == FormFieldSelect {
				p.openSelectDropdown()
				return nil
			} else if field.Type == FormFieldToggle {
				field.Value = 1 - field.Value
				p.notifyLiveChange(*field)
				return nil
			}
		}
	}

	// Standard field navigation (not in edit mode)
	switch key {
	case "tab", "down", "j":
		if p.formOnButtons {
			p.formBtnIndex = (p.formBtnIndex + 1) % 2
		} else {
			if p.formCursor < len(p.formFields)-1 {
				p.formCursor++
			} else {
				p.formOnButtons = true
				p.formBtnIndex = 0
			}
		}
		p.ensureFocusedFieldVisible()

	case "shift+tab", "up", "k":
		if p.formOnButtons {
			if p.formBtnIndex > 0 {
				p.formBtnIndex--
			} else {
				p.formOnButtons = false
				p.formCursor = len(p.formFields) - 1
			}
		} else {
			if p.formCursor > 0 {
				p.formCursor--
			}
		}
		p.ensureFocusedFieldVisible()

	case "left", "h":
		if !p.formOnButtons && p.formCursor < len(p.formFields) {
			field := &p.formFields[p.formCursor]
			changed := false
			switch field.Type {
			case FormFieldSlider, FormFieldBrightness:
				if field.Value > field.Min {
					field.Value--
					changed = true
				}
			case FormFieldColorTemp:
				// Inverted: left = toward warm = increase mirek
				if field.Value < field.Max {
					field.Value += 2
					if field.Value > field.Max {
						field.Value = field.Max
					}
					changed = true
				}
			// Note: Color requires Enter to edit
			// Note: Radio and Select require Enter to edit
			}
			if changed {
				p.notifyLiveChange(*field)
			}
		} else if p.formOnButtons {
			p.formBtnIndex = 0
		}

	case "right", "l":
		if !p.formOnButtons && p.formCursor < len(p.formFields) {
			field := &p.formFields[p.formCursor]
			changed := false
			switch field.Type {
			case FormFieldSlider, FormFieldBrightness:
				if field.Value < field.Max {
					field.Value++
					changed = true
				}
			case FormFieldColorTemp:
				// Inverted: right = toward cool = decrease mirek
				if field.Value > field.Min {
					field.Value -= 2
					if field.Value < field.Min {
						field.Value = field.Min
					}
					changed = true
				}
			// Note: Color requires Enter to edit
			// Note: Radio and Select require Enter to edit
			}
			if changed {
				p.notifyLiveChange(*field)
			}
		} else if p.formOnButtons {
			p.formBtnIndex = 1
		}

	}

	return nil
}

// blurAllTextInputs removes focus from all text inputs.
// handleFieldEditMode handles keys when a field is in edit mode.
func (p *PopupPanel) handleFieldEditMode(msg tea.KeyMsg) tea.Cmd {
	key := msg.String()
	field := &p.formFields[p.formCursor]

	switch field.Type {
	case FormFieldText:
		// Handle text editing
		switch key {
		case "esc":
			// Exit edit mode for text, stay on field
			p.formFieldEditing = false
			p.blurAllTextInputs()
			return nil
		case "enter":
			// Exit edit mode and advance to next field (or Save if last field)
			p.formFieldEditing = false
			p.blurAllTextInputs()
			if p.formCursor < len(p.formFields)-1 {
				p.formCursor++
				// If next field is also a text field, enter edit mode
				if p.formFields[p.formCursor].Type == FormFieldText {
					if ti, ok := p.formTextInputs[p.formCursor]; ok {
						ti.Focus()
						p.formTextInputs[p.formCursor] = ti
						p.formFieldEditing = true
					}
				}
			} else {
				p.formOnButtons = true
				p.formBtnIndex = 0 // Save button
			}
			return nil
		case "ctrl+t":
			if ti, ok := p.formTextInputs[p.formCursor]; ok {
				ti.SetValue(ui.ToTitleCase(ti.Value()))
				p.formTextInputs[p.formCursor] = ti
				p.formFields[p.formCursor].TextValue = ti.Value()
			}
			return nil
		case "ctrl+l":
			if ti, ok := p.formTextInputs[p.formCursor]; ok {
				ti.SetValue(ui.ToSentenceCase(ti.Value()))
				p.formTextInputs[p.formCursor] = ti
				p.formFields[p.formCursor].TextValue = ti.Value()
			}
			return nil
		default:
			// Pass to text input
			if ti, ok := p.formTextInputs[p.formCursor]; ok {
				var cmd tea.Cmd
				ti, cmd = ti.Update(msg)
				p.formTextInputs[p.formCursor] = ti
				p.formFields[p.formCursor].TextValue = ti.Value()
				return cmd
			}
		}

	case FormFieldRadio:
		switch key {
		case "esc":
			// Cancel - restore original value and exit
			field.Value = p.radioOriginalValue
			p.formFieldEditing = false
			return nil
		case "enter":
			// Confirm - value already set, just exit
			p.notifyLiveChange(*field)
			p.formFieldEditing = false
			return nil
		case "up", "k":
			// Move cursor up (vertical)
			if field.Vertical && p.radioEditCursor > 0 {
				p.radioEditCursor--
				field.Value = field.Options[p.radioEditCursor].Value
			}
			return nil
		case "down", "j":
			// Move cursor down (vertical)
			if field.Vertical && p.radioEditCursor < len(field.Options)-1 {
				p.radioEditCursor++
				field.Value = field.Options[p.radioEditCursor].Value
			}
			return nil
		case "left", "h":
			// Move cursor left (horizontal)
			if !field.Vertical && p.radioEditCursor > 0 {
				p.radioEditCursor--
				field.Value = field.Options[p.radioEditCursor].Value
			}
			return nil
		case "right", "l":
			// Move cursor right (horizontal)
			if !field.Vertical && p.radioEditCursor < len(field.Options)-1 {
				p.radioEditCursor++
				field.Value = field.Options[p.radioEditCursor].Value
			}
			return nil
		}

	case FormFieldColor:
		// Move in screen coordinates, then derive color
		radiusY := 4
		radiusX := 9
		diameterY := radiusY*2 + 1
		// Use same overshoot as rendering for navigation bounds
		effectiveRadiusX := float64(radiusX) + 0.5
		hiResRadiusY := float64(radiusY * 2)
		hiResCenterY := float64(radiusY*2) + 0.5
		effectiveRadiusY := hiResRadiusY + 1.0

		// Helper to check if a cell is inside the overshot ellipse
		cellIsInside := func(row, col int) bool {
			dx := col - radiusX
			// Check both sub-pixels for this cell
			hiResRowTop := float64(row * 2)
			hiResRowBot := float64(row*2 + 1)
			dyTop := hiResRowTop - hiResCenterY
			dyBot := hiResRowBot - hiResCenterY
			topIn := float64(dx*dx)/(effectiveRadiusX*effectiveRadiusX)+(dyTop*dyTop)/(effectiveRadiusY*effectiveRadiusY) <= 1.0
			botIn := float64(dx*dx)/(effectiveRadiusX*effectiveRadiusX)+(dyBot*dyBot)/(effectiveRadiusY*effectiveRadiusY) <= 1.0
			return topIn || botIn
		}

		switch key {
		case "esc":
			// In live mode, changes are already applied, just exit edit mode
			// In non-live mode, revert to original
			if !p.formLiveMode {
				field.ColorX = p.colorOriginalX
				field.ColorY = p.colorOriginalY
			}
			p.formFieldEditing = false
			return nil
		case "enter":
			// Enter just exits edit mode (same as Esc in live mode)
			p.formFieldEditing = false
			return nil
		case "left", "h":
			newCol := p.colorSelCol - 1
			if newCol >= 0 && cellIsInside(p.colorSelRow, newCol) {
				p.colorSelCol = newCol
			}
		case "right", "l":
			newCol := p.colorSelCol + 1
			if newCol < radiusX*2+1 && cellIsInside(p.colorSelRow, newCol) {
				p.colorSelCol = newCol
			}
		case "up", "k":
			if p.colorSelRow > 0 {
				newRow := p.colorSelRow - 1
				if cellIsInside(newRow, p.colorSelCol) {
					p.colorSelRow = newRow
				} else {
					// Try to find a valid column in the new row
					for offset := 1; offset <= radiusX; offset++ {
						if cellIsInside(newRow, p.colorSelCol-offset) {
							p.colorSelRow = newRow
							p.colorSelCol -= offset
							break
						}
						if cellIsInside(newRow, p.colorSelCol+offset) {
							p.colorSelRow = newRow
							p.colorSelCol += offset
							break
						}
					}
				}
			}
		case "down", "j":
			if p.colorSelRow < diameterY-1 {
				newRow := p.colorSelRow + 1
				if cellIsInside(newRow, p.colorSelCol) {
					p.colorSelRow = newRow
				} else {
					// Try to find a valid column in the new row
					for offset := 1; offset <= radiusX; offset++ {
						if cellIsInside(newRow, p.colorSelCol-offset) {
							p.colorSelRow = newRow
							p.colorSelCol -= offset
							break
						}
						if cellIsInside(newRow, p.colorSelCol+offset) {
							p.colorSelRow = newRow
							p.colorSelCol += offset
							break
						}
					}
				}
			}
		}
		
		// Mark position as valid since user interacted with it
		p.colorPosValid = true
		
		// Derive ColorX/ColorY using same HSV calculation as rendering
		dy := float64(p.colorSelRow - radiusY)
		dx := float64(p.colorSelCol - radiusX)
		xNorm := dx / float64(radiusX)
		yNorm := dy / float64(radiusY)
		
		// Calculate hue and saturation same as rendering
		// Reverse direction and rotate to put Red at right
		dist := math.Sqrt(xNorm*xNorm + yNorm*yNorm)
		angle := math.Atan2(-yNorm, xNorm)
		hue := -int(angle*180/math.Pi) + 90
		hue = ((hue % 360) + 360) % 360
		sat := int(dist * 100)
		if sat > 100 {
			sat = 100
		}
		
		// Convert HSV to XY
		field.ColorX, field.ColorY = ui.HueSatToXY(hue, sat)
		
		// Clamp to valid range
		if field.ColorX < 0.05 {
			field.ColorX = 0.05
		}
		if field.ColorX > 0.65 {
			field.ColorX = 0.65
		}
		if field.ColorY < 0.05 {
			field.ColorY = 0.05
		}
		if field.ColorY > 0.6 {
			field.ColorY = 0.6
		}
		
		// Notify live change for real-time updates
		p.notifyLiveChange(*field)
		return nil

	case FormFieldHSL:
		switch key {
		case "esc":
			// Cancel - restore original values
			field.Hue = field.OriginalHue
			field.Saturation = field.OriginalSat
			field.Lightness = field.OriginalLight
			p.formFieldEditing = false
			return nil
		case "enter":
			// Confirm
			p.notifyLiveChange(*field)
			p.formFieldEditing = false
			return nil
		case "up", "k":
			// Move to previous slider
			if field.HSLSliderFocus > 0 {
				field.HSLSliderFocus--
			}
			return nil
		case "down", "j":
			// Move to next slider
			if field.HSLSliderFocus < 2 {
				field.HSLSliderFocus++
			}
			return nil
		case "left", "h":
			// Decrease current slider value
			switch field.HSLSliderFocus {
			case 0: // Hue
				field.Hue -= 5
				if field.Hue < 0 {
					field.Hue = 0
				}
			case 1: // Saturation
				field.Saturation -= 5
				if field.Saturation < 0 {
					field.Saturation = 0
				}
			case 2: // Lightness
				field.Lightness -= 5
				if field.Lightness < 0 {
					field.Lightness = 0
				}
			}
			return nil
		case "right", "l":
			// Increase current slider value
			switch field.HSLSliderFocus {
			case 0: // Hue
				field.Hue += 5
				if field.Hue > 360 {
					field.Hue = 360
				}
			case 1: // Saturation
				field.Saturation += 5
				if field.Saturation > 100 {
					field.Saturation = 100
				}
			case 2: // Lightness
				field.Lightness += 5
				if field.Lightness > 100 {
					field.Lightness = 100
				}
			}
			return nil
		}

	case FormFieldRGB:
		switch key {
		case "esc":
			// Cancel - restore original values
			field.Red = field.OriginalRed
			field.Green = field.OriginalGreen
			field.Blue = field.OriginalBlue
			p.formFieldEditing = false
			return nil
		case "enter":
			// Confirm
			p.notifyLiveChange(*field)
			p.formFieldEditing = false
			return nil
		case "up", "k":
			// Move to previous slider
			if field.RGBSliderFocus > 0 {
				field.RGBSliderFocus--
			}
			return nil
		case "down", "j":
			// Move to next slider
			if field.RGBSliderFocus < 2 {
				field.RGBSliderFocus++
			}
			return nil
		case "left", "h":
			// Decrease current slider value
			switch field.RGBSliderFocus {
			case 0: // Red
				field.Red -= 5
				if field.Red < 0 {
					field.Red = 0
				}
			case 1: // Green
				field.Green -= 5
				if field.Green < 0 {
					field.Green = 0
				}
			case 2: // Blue
				field.Blue -= 5
				if field.Blue < 0 {
					field.Blue = 0
				}
			}
			return nil
		case "right", "l":
			// Increase current slider value
			switch field.RGBSliderFocus {
			case 0: // Red
				field.Red += 5
				if field.Red > 255 {
					field.Red = 255
				}
			case 1: // Green
				field.Green += 5
				if field.Green > 255 {
					field.Green = 255
				}
			case 2: // Blue
				field.Blue += 5
				if field.Blue > 255 {
					field.Blue = 255
				}
			}
			return nil
		}
	}

	return nil
}

func (p *PopupPanel) blurAllTextInputs() {
	for i, ti := range p.formTextInputs {
		ti.Blur()
		p.formTextInputs[i] = ti
	}
}

// notifyLiveChange schedules a debounced callback if in live mode.
// The callback is debounced to avoid flooding the API during rapid slider adjustments.
func (p *PopupPanel) notifyLiveChange(field FormField) {
	if !p.formLiveMode || p.formOnChange == nil {
		return
	}
	
	// Store the pending change and current timestamp
	fieldCopy := field
	p.debouncePending = &fieldCopy
	p.debounceTime = time.Now()
	
	// Start a goroutine that will fire the callback after the debounce delay
	// if no newer changes have come in
	timestamp := p.debounceTime
	callback := p.formOnChange
	go func() {
		time.Sleep(debounceDelay)
		// Only fire if this is still the most recent change
		if p.debouncePending != nil && timestamp.Equal(p.debounceTime) {
			callback(fieldCopy)
		}
	}()
}

// IsLiveMode returns whether the form is in live mode.
func (p *PopupPanel) IsLiveMode() bool {
	return p.formLiveMode
}

// SetLiveMode sets the form's live mode.
func (p *PopupPanel) SetLiveMode(live bool) {
	p.formLiveMode = live
}

// focusCurrentTextField focuses the text input at the current cursor position if applicable.
func (p *PopupPanel) focusCurrentTextField() {
	p.blurAllTextInputs()
	if p.formCursor < len(p.formFields) && p.formFields[p.formCursor].Type == FormFieldText {
		if ti, ok := p.formTextInputs[p.formCursor]; ok {
			ti.Focus()
			p.formTextInputs[p.formCursor] = ti
		}
	}
}

// updateSliderFromMouse updates a slider field value based on mouse X position.
func (p *PopupPanel) updateSliderFromMouse(field *FormField, mouseX int, valueStartX int) {
	var sliderWidth int
	var sliderStartX int

	switch field.Type {
	case FormFieldSlider:
		sliderWidth = 10
		sliderStartX = valueStartX + 1 // After "["
	case FormFieldBrightness:
		sliderWidth = 20
		sliderStartX = valueStartX // Slider starts directly at value position
	case FormFieldColorTemp:
		sliderWidth = 20
		sliderStartX = valueStartX - 1 // Account for rendering offset
	default:
		return
	}

	// Calculate position within slider
	clickPos := mouseX - sliderStartX
	if clickPos < 0 {
		clickPos = 0
	}
	if clickPos > sliderWidth {
		clickPos = sliderWidth
	}

	// Map to value
	if field.Max > field.Min {
		if field.Type == FormFieldColorTemp {
			// ColorTemp is inverted: left = Max (warm), right = Min (cool)
			field.Value = field.Max - (clickPos * (field.Max - field.Min) / sliderWidth)
		} else {
			field.Value = field.Min + (clickPos * (field.Max - field.Min) / sliderWidth)
		}
		if field.Value < field.Min {
			field.Value = field.Min
		}
		if field.Value > field.Max {
			field.Value = field.Max
		}
	}

	p.notifyLiveChange(*field)
}

// getFieldRowY returns the Y position of a field's first row on screen.
// Takes into account scrolling when the form is larger than the viewport.
func (p *PopupPanel) getFieldRowY(fieldIdx int, popupY int, startY int) int {
	// Use rowForField which accounts for spacing between fields
	logicalRow := p.rowForField(fieldIdx)

	// Check if scrolling is active
	contentHeight := p.contentHeight()
	totalFieldRows := p.totalFormRows()
	totalRows := totalFieldRows + 2 // fields + blank + button row
	needsScroll := totalRows > contentHeight

	if needsScroll {
		// In scroll mode: convert logical row to screen row by subtracting scroll offset
		screenRow := logicalRow - p.formScroll
		return popupY + 1 + screenRow // +1 for border
	}

	// In centered mode
	return popupY + 1 + startY + logicalRow // +1 for border
}

// updateColorFromMouse updates a color field based on mouse position.
func (p *PopupPanel) updateColorFromMouse(field *FormField, mouseX, mouseY, valueStartX, fieldRowY int) {
	radiusY := 4
	radiusX := 9
	centerRow := radiusY
	diameterY := radiusY*2 + 1
	
	// Use same overshoot as rendering
	effectiveRadiusX := float64(radiusX) + 0.5
	hiResRadiusY := float64(radiusY * 2)
	hiResCenterY := float64(radiusY*2) + 0.5
	effectiveRadiusY := hiResRadiusY + 1.0
	
	// Calculate clicked position relative to wheel
	// fieldRowY points to the first row of the color field (which is wheel row 0)
	clickedCol := mouseX - valueStartX
	clickedRow := mouseY - fieldRowY
	
	// Check if within bounds
	if clickedRow < 0 || clickedRow >= diameterY {
		return
	}
	
	// Check if cell is inside the overshot ellipse
	dx := clickedCol - radiusX
	hiResRowTop := float64(clickedRow * 2)
	hiResRowBot := float64(clickedRow*2 + 1)
	dyTop := hiResRowTop - hiResCenterY
	dyBot := hiResRowBot - hiResCenterY
	topIn := float64(dx*dx)/(effectiveRadiusX*effectiveRadiusX)+(dyTop*dyTop)/(effectiveRadiusY*effectiveRadiusY) <= 1.0
	botIn := float64(dx*dx)/(effectiveRadiusX*effectiveRadiusX)+(dyBot*dyBot)/(effectiveRadiusY*effectiveRadiusY) <= 1.0
	
	if !topIn && !botIn {
		return
	}
	
	// Update selection
	p.colorSelRow = clickedRow
	p.colorSelCol = clickedCol
	p.colorPosValid = true
	
	// Derive ColorX/ColorY using same HSV calculation as rendering
	dxf := float64(clickedCol - radiusX)
	dy := float64(clickedRow - centerRow)
	xNorm := dxf / float64(radiusX)
	yNorm := dy / float64(radiusY)
	
	// Calculate hue and saturation same as rendering
	// Reverse direction and rotate to put Red at right
	dist := math.Sqrt(xNorm*xNorm + yNorm*yNorm)
	angle := math.Atan2(-yNorm, xNorm)
	hue := -int(angle*180/math.Pi) + 90
	hue = ((hue % 360) + 360) % 360
	sat := int(dist * 100)
	if sat > 100 {
		sat = 100
	}
	
	// Convert HSV to XY using the same function as elsewhere
	field.ColorX, field.ColorY = ui.HueSatToXY(hue, sat)
	
	// Clamp
	if field.ColorX < 0.05 {
		field.ColorX = 0.05
	}
	if field.ColorX > 0.65 {
		field.ColorX = 0.65
	}
	if field.ColorY < 0.05 {
		field.ColorY = 0.05
	}
	if field.ColorY > 0.6 {
		field.ColorY = 0.6
	}
	
	p.notifyLiveChange(*field)
}

// updateHSLFromMouseRow updates an HSL field for a specific slider row.
func (p *PopupPanel) updateHSLFromMouseRow(field *FormField, mouseX, valueStartX, sliderRow int) {
	if sliderRow < 0 || sliderRow > 2 {
		return
	}

	sliderWidth := 20
	sliderStartX := valueStartX + 3 // After "H: " label

	// Calculate value from mouse X
	clickPos := mouseX - sliderStartX
	if clickPos < 0 {
		clickPos = 0
	}
	if clickPos > sliderWidth {
		clickPos = sliderWidth
	}

	// Map to value based on which slider
	switch sliderRow {
	case 0: // Hue (0-360)
		field.Hue = clickPos * 360 / sliderWidth
		field.HSLSliderFocus = 0
	case 1: // Saturation (0-100)
		field.Saturation = clickPos * 100 / sliderWidth
		field.HSLSliderFocus = 1
	case 2: // Lightness (0-100)
		field.Lightness = clickPos * 100 / sliderWidth
		field.HSLSliderFocus = 2
	}

	p.notifyLiveChange(*field)
}

// updateRGBFromMouseRow updates an RGB field for a specific slider row.
func (p *PopupPanel) updateRGBFromMouseRow(field *FormField, mouseX, valueStartX, sliderRow int) {
	if sliderRow < 0 || sliderRow > 2 {
		return
	}

	sliderWidth := 20
	sliderStartX := valueStartX + 3 // After "R: " label

	// Calculate value from mouse X
	clickPos := mouseX - sliderStartX
	if clickPos < 0 {
		clickPos = 0
	}
	if clickPos > sliderWidth {
		clickPos = sliderWidth
	}

	// Map to value (0-255)
	value := clickPos * 255 / sliderWidth

	switch sliderRow {
	case 0: // Red
		field.Red = value
		field.RGBSliderFocus = 0
	case 1: // Green
		field.Green = value
		field.RGBSliderFocus = 1
	case 2: // Blue
		field.Blue = value
		field.RGBSliderFocus = 2
	}

	p.notifyLiveChange(*field)
}

// openSelectDropdown opens the dropdown for the current select field.
func (p *PopupPanel) openSelectDropdown() {
	if p.formCursor >= len(p.formFields) {
		return
	}
	field := p.formFields[p.formCursor]
	if field.Type != FormFieldSelect || len(field.Options) == 0 {
		return
	}

	// Find current selection index
	p.selectDropdownCursor = 0
	for i, opt := range field.Options {
		if opt.Value == field.Value {
			p.selectDropdownCursor = i
			break
		}
	}
	p.selectDropdownScroll = 0
	p.selectDropdownOpen = true
	p.selectDropdownJustOpened = true // Ignore next mouse event

	// Ensure cursor is visible
	p.ensureDropdownCursorVisible()
}

// closeSelectDropdown closes the dropdown without selecting.
func (p *PopupPanel) closeSelectDropdown() {
	p.selectDropdownOpen = false
}

// selectDropdownMaxVisible returns the max visible items in dropdown.
func (p *PopupPanel) selectDropdownMaxVisible() int {
	// Limit dropdown height
	maxHeight := 8
	contentHeight := p.contentHeight()
	if maxHeight > contentHeight-4 {
		maxHeight = contentHeight - 4
	}
	if maxHeight < 3 {
		maxHeight = 3
	}
	return maxHeight
}

// ensureDropdownCursorVisible ensures the cursor is within the visible scroll area.
func (p *PopupPanel) ensureDropdownCursorVisible() {
	maxVisible := p.selectDropdownMaxVisible()
	if p.selectDropdownCursor < p.selectDropdownScroll {
		p.selectDropdownScroll = p.selectDropdownCursor
	} else if p.selectDropdownCursor >= p.selectDropdownScroll+maxVisible {
		p.selectDropdownScroll = p.selectDropdownCursor - maxVisible + 1
	}
}

// handleSelectDropdownKey handles keyboard input when dropdown is open.
func (p *PopupPanel) handleSelectDropdownKey(msg tea.KeyMsg) tea.Cmd {
	if p.formCursor >= len(p.formFields) {
		p.closeSelectDropdown()
		return nil
	}
	field := &p.formFields[p.formCursor]
	numOptions := len(field.Options)

	switch msg.String() {
	case "esc":
		p.closeSelectDropdown()
		return nil

	case "enter", " ":
		// Select current option
		if p.selectDropdownCursor >= 0 && p.selectDropdownCursor < numOptions {
			field.Value = field.Options[p.selectDropdownCursor].Value
			p.notifyLiveChange(*field)
		}
		p.closeSelectDropdown()
		return nil

	case "up", "k":
		if p.selectDropdownCursor > 0 {
			p.selectDropdownCursor--
			p.ensureDropdownCursorVisible()
		}
		return nil

	case "down", "j":
		if p.selectDropdownCursor < numOptions-1 {
			p.selectDropdownCursor++
			p.ensureDropdownCursorVisible()
		}
		return nil

	case "home", "g":
		p.selectDropdownCursor = 0
		p.selectDropdownScroll = 0
		return nil

	case "end", "G":
		p.selectDropdownCursor = numOptions - 1
		p.ensureDropdownCursorVisible()
		return nil

	case "pgup":
		p.selectDropdownCursor -= p.selectDropdownMaxVisible()
		if p.selectDropdownCursor < 0 {
			p.selectDropdownCursor = 0
		}
		p.ensureDropdownCursorVisible()
		return nil

	case "pgdown":
		p.selectDropdownCursor += p.selectDropdownMaxVisible()
		if p.selectDropdownCursor >= numOptions {
			p.selectDropdownCursor = numOptions - 1
		}
		p.ensureDropdownCursorVisible()
		return nil
	}

	return nil
}

func (p *PopupPanel) handleMouse(msg tea.MouseMsg) tea.Cmd {
	switch p.mode {
	case PopupModeDisplay:
		// Handle scroll wheel without checking Action
		switch msg.Button {
		case tea.MouseButtonWheelUp:
			if p.scroll > 0 {
				p.scroll--
			}
		case tea.MouseButtonWheelDown:
			if p.scroll < p.maxScroll() {
				p.scroll++
			}
		}
	case PopupModeConfirm:
		if msg.Type == tea.MouseLeft {
			// Check if click is on a button
			// Calculate popup position (centered on screen)
			popupX := (p.screenWidth - p.width()) / 2
			popupY := (p.screenHeight - p.height()) / 2

			// Calculate button row (midY+1 in content, +1 for top border, +1 for content padding)
			contentHeight := p.contentHeight()
			midY := contentHeight / 2
			buttonRowY := popupY + 1 + midY + 1 // +1 border, +1 to get to midY+1

			// Check if click is on button row
			if msg.Y == buttonRowY {
				// Calculate button positions within the popup
				contentWidth := p.contentWidth()
				yesBtn := " " + p.yesLabel + " "
				noBtn := " " + p.noLabel + " "
				buttons := yesBtn + "  " + noBtn
				buttonsLen := lipgloss.Width(buttons)
				buttonsStartX := popupX + 2 + (contentWidth-buttonsLen)/2 // +2 for border+padding

				// Check Yes button
				yesEndX := buttonsStartX + lipgloss.Width(yesBtn)
				if msg.X >= buttonsStartX && msg.X < yesEndX {
					p.selected = 0
					p.close(PopupResult{Confirmed: true})
					return nil
				}

				// Check No button (after Yes + 2 spaces gap)
				noStartX := yesEndX + 2
				noEndX := noStartX + lipgloss.Width(noBtn)
				if msg.X >= noStartX && msg.X < noEndX {
					p.selected = 1
					p.close(PopupResult{Confirmed: false})
					return nil
				}
			}
		}
	case PopupModeSelect:
		// Calculate popup position
		popupX := (p.screenWidth - p.width()) / 2
		popupY := (p.screenHeight - p.height()) / 2

		// Handle scroll wheel without checking Action
		switch msg.Button {
		case tea.MouseButtonWheelUp:
			if p.selectCursor > 0 {
				p.selectCursor--
			}
		case tea.MouseButtonWheelDown:
			if p.selectCursor < len(p.selectOptions)-1 {
				p.selectCursor++
			}
		case tea.MouseButtonLeft:
			if msg.Action == tea.MouseActionPress {
				// Check if click is on an option
				contentStartY := popupY + 1 // +1 for top border
				for i := range p.selectOptions {
					optionY := contentStartY + i
					if msg.Y == optionY && msg.X > popupX && msg.X < popupX+p.width()-1 {
						p.selectCursor = i
						p.close(PopupResult{
							Confirmed: true,
							Value:     p.selectOptions[i].Value,
							Index:     i,
						})
						return nil
					}
				}
			}
		}

	case PopupModeForm:
		// Calculate popup position (centered on screen)
		popupX := (p.screenWidth - p.width()) / 2
		popupY := (p.screenHeight - p.height()) / 2

		// Calculate layout - must match renderFormContent logic
		contentHeight := p.contentHeight()
		totalFieldRows := p.totalFormRows()
		totalRows := totalFieldRows + 2 // fields + blank + button row
		needsScroll := totalRows > contentHeight

		var startY int
		if needsScroll {
			startY = 0
		} else {
			startY = (contentHeight - totalRows) / 2
			if startY < 0 {
				startY = 0
			}
		}

		contentStartX := popupX + 2 // border + padding
		labelWidth := 18
		valueStartX := contentStartX + labelWidth

		// Handle scroll wheel first - scroll form content if scrolling is needed
		// Note: Don't check msg.Action for scroll wheel - just check msg.Button
		if msg.Button == tea.MouseButtonWheelUp || msg.Button == tea.MouseButtonWheelDown {
			maxScroll := p.formMaxScroll(contentHeight)
			if maxScroll > 0 {
				// Scroll the form
				if msg.Button == tea.MouseButtonWheelUp {
					p.formScroll -= 3
					if p.formScroll < 0 {
						p.formScroll = 0
					}
				} else {
					p.formScroll += 3
					if p.formScroll > maxScroll {
						p.formScroll = maxScroll
					}
				}
				return nil
			}
		}

		// Handle dropdown mouse events first
		if p.selectDropdownOpen {
			if cmd := p.handleDropdownMouse(msg, popupX, popupY, startY, valueStartX); cmd != nil {
				return cmd
			}
			// If dropdown is open, consume all mouse events to prevent click-through
			if msg.Type == tea.MouseLeft || msg.Type == tea.MouseRelease {
				return nil
			}
		}

		// Handle mouse release - end any capture
		if msg.Type == tea.MouseRelease {
			p.mouseCaptureIdx = -1
			p.capturedSliderRow = -1
			return nil
		}

		// Handle dragging when we have a captured field
		// Note: Some terminals send MouseLeft repeatedly during drag, others send MouseMotion
		if p.mouseCaptureIdx >= 0 && (msg.Type == tea.MouseMotion || msg.Type == tea.MouseLeft) {
			field := &p.formFields[p.mouseCaptureIdx]
			switch field.Type {
			case FormFieldSlider, FormFieldBrightness, FormFieldColorTemp:
				p.updateSliderFromMouse(field, msg.X, valueStartX)
			case FormFieldColor:
				// Calculate field's Y position for row calculation
				fieldRowY := p.getFieldRowY(p.mouseCaptureIdx, popupY, startY)
				p.updateColorFromMouse(field, msg.X, msg.Y, valueStartX, fieldRowY)
			case FormFieldHSL:
				// Use the captured slider row, not the current mouse Y
				p.updateHSLFromMouseRow(field, msg.X, valueStartX, p.capturedSliderRow)
			case FormFieldRGB:
				// Use the captured slider row, not the current mouse Y
				p.updateRGBFromMouseRow(field, msg.X, valueStartX, p.capturedSliderRow)
			}
			// All captured fields consume the event to prevent re-triggering
			return nil
		}

		// Handle left click (new click, no capture active)
		if msg.Type == tea.MouseLeft {
			// Calculate button row position - in scroll mode, check if visible
			buttonLogicalRow := totalFieldRows + 1
			var buttonRowY int
			if needsScroll {
				// Button row is at logical position, convert to screen position
				buttonScreenRow := buttonLogicalRow - p.formScroll
				if buttonScreenRow >= 0 && buttonScreenRow < contentHeight {
					buttonRowY = popupY + 1 + buttonScreenRow // +1 for top border
				} else {
					buttonRowY = -1 // Button not visible
				}
			} else {
				buttonRowY = popupY + 1 + startY + buttonLogicalRow // +1 for top border
			}

			// Check if click is on button row
			if buttonRowY >= 0 && msg.Y == buttonRowY {
				contentWidth := p.contentWidth()
				saveBtn := " Save "
				cancelBtn := " Cancel "
				buttons := saveBtn + "  " + cancelBtn
				buttonsLen := lipgloss.Width(buttons)
				buttonsStartX := popupX + 2 + (contentWidth-buttonsLen)/2 // +2 for border+padding

				// Check Save button
				saveEndX := buttonsStartX + lipgloss.Width(saveBtn)
				if msg.X >= buttonsStartX && msg.X < saveEndX {
					p.formBtnIndex = 0
					p.formOnButtons = true
					p.close(PopupResult{Confirmed: true})
					return nil
				}

				// Check Cancel button (after Save + 2 spaces gap)
				cancelStartX := saveEndX + 2
				cancelEndX := cancelStartX + lipgloss.Width(cancelBtn)
				if msg.X >= cancelStartX && msg.X < cancelEndX {
					p.formBtnIndex = 1
					p.formOnButtons = true
					p.close(PopupResult{Confirmed: false})
					return nil
				}
			}

			// Check if click is on a form field row
			// Calculate the logical row from the screen position
			screenRowInContent := msg.Y - popupY - 1 - startY // -1 for top border
			var logicalRow int
			if needsScroll {
				logicalRow = screenRowInContent + p.formScroll
			} else {
				logicalRow = screenRowInContent
			}

			// Check if this is a valid field row
			if logicalRow >= 0 && logicalRow < totalFieldRows && msg.X > popupX && msg.X < popupX+p.width()-1 {
				// Map logical row to field index and subrow
				fieldIdx, subRow := p.fieldAtRow(logicalRow)
				if fieldIdx >= 0 && fieldIdx < len(p.formFields) {
					wasAlreadySelected := p.formCursor == fieldIdx && !p.formOnButtons

					// Select this field
					p.formCursor = fieldIdx
					p.formOnButtons = false
					p.focusCurrentTextField()

					field := &p.formFields[fieldIdx]

					// Always capture mouse to prevent repeated actions during drag
					p.mouseCaptureIdx = fieldIdx
					p.mouseCaptureType = field.Type

					switch field.Type {
					case FormFieldToggle:
						// Only toggle if clicking on an already-selected field
						if wasAlreadySelected && msg.X >= valueStartX {
							if field.Value != 0 {
								field.Value = 0
							} else {
								field.Value = 1
							}
							p.notifyLiveChange(*field)
						}

					case FormFieldSlider, FormFieldBrightness, FormFieldColorTemp:
						// Set initial value from click position
						p.updateSliderFromMouse(field, msg.X, valueStartX)

					case FormFieldColor:
						// Click on color wheel to select color
						radiusY := 4
						radiusX := 9
						centerRow := radiusY
						diameterY := radiusY*2 + 1
						effectiveRadiusY := float64(radiusY) + 0.5
						
						// Calculate clicked position
						// For color fields, subRow 0 is wheel row 0 (no separate label)
						clickedCol := msg.X - valueStartX
						clickedRow := subRow
						
						// Skip if click is on label row or outside wheel bounds
						if clickedRow < 0 || clickedRow >= diameterY {
							break
						}
						
						// Check if click is within the ellipse
						dy := float64(clickedRow - centerRow)
						yNorm := dy / effectiveRadiusY
						if yNorm*yNorm <= 1.0 {
							halfWidth := int(float64(radiusX) * math.Sqrt(1.0-yNorm*yNorm))
							minCol := radiusX - halfWidth
							maxCol := radiusX + halfWidth
							
							if clickedCol >= minCol && clickedCol <= maxCol {
								// Valid click - update selection
								p.colorSelRow = clickedRow
								p.colorSelCol = clickedCol
								p.colorPosValid = true
								
								// Derive ColorX/ColorY using same HSV calculation as rendering
								dx := float64(clickedCol - radiusX)
								xNorm := dx / float64(radiusX)
								yNormColor := dy / float64(radiusY)
								
								// Calculate hue and saturation same as rendering
								// Reverse direction and rotate to put Red at right
								dist := math.Sqrt(xNorm*xNorm + yNormColor*yNormColor)
								angle := math.Atan2(-yNormColor, xNorm)
								hue := -int(angle*180/math.Pi) + 90
								hue = ((hue % 360) + 360) % 360
								sat := int(dist * 100)
								if sat > 100 {
									sat = 100
								}
								
								// Convert HSV to XY
								field.ColorX, field.ColorY = ui.HueSatToXY(hue, sat)
								
								// Clamp
								if field.ColorX < 0.05 {
									field.ColorX = 0.05
								}
								if field.ColorX > 0.65 {
									field.ColorX = 0.65
								}
								if field.ColorY < 0.05 {
									field.ColorY = 0.05
								}
								if field.ColorY > 0.6 {
									field.ColorY = 0.6
								}
								
								// Enter edit mode if not already
								if !p.formFieldEditing {
									p.formFieldEditing = true
									p.colorOriginalX = p.formFields[p.formCursor].ColorX
									p.colorOriginalY = p.formFields[p.formCursor].ColorY
								}
								
								p.notifyLiveChange(*field)
								
								// Set mouse capture for dragging
								p.mouseCaptureIdx = p.formCursor
								p.mouseCaptureType = FormFieldColor
							}
						}

					case FormFieldHSL:
						// Click on HSL sliders
						// subRow tells us which slider (0=H, 1=S, 2=L) was clicked
						p.capturedSliderRow = subRow
						p.updateHSLFromMouseRow(field, msg.X, valueStartX, subRow)
						// Enter edit mode
						if !p.formFieldEditing {
							p.formFieldEditing = true
							field.OriginalHue = field.Hue
							field.OriginalSat = field.Saturation
							field.OriginalLight = field.Lightness
						}
						// Capture for dragging
						p.mouseCaptureIdx = p.formCursor
						p.mouseCaptureType = FormFieldHSL

					case FormFieldRGB:
						// Click on RGB sliders
						// subRow tells us which slider (0=R, 1=G, 2=B) was clicked
						p.capturedSliderRow = subRow
						p.updateRGBFromMouseRow(field, msg.X, valueStartX, subRow)
						// Enter edit mode
						if !p.formFieldEditing {
							p.formFieldEditing = true
							field.OriginalRed = field.Red
							field.OriginalGreen = field.Green
							field.OriginalBlue = field.Blue
						}
						// Capture for dragging
						p.mouseCaptureIdx = p.formCursor
						p.mouseCaptureType = FormFieldRGB

					case FormFieldSelect:
						// Click opens dropdown
						if msg.X >= valueStartX && len(field.Options) > 0 {
							p.openSelectDropdown()
						}

					case FormFieldRadio:
						// Click on radio button
						if msg.X >= valueStartX && len(field.Options) > 0 {
							if field.Vertical {
								// Vertical: subRow corresponds to option index
								if subRow < len(field.Options) {
									opt := field.Options[subRow]
									if field.Value != opt.Value {
										field.Value = opt.Value
										p.notifyLiveChange(*field)
									}
								}
							} else {
								// Horizontal: calculate option positions
								clickX := msg.X - valueStartX
								currentX := 0
								for _, opt := range field.Options {
									// Each option: "○ Label" + "  " spacing
									optWidth := 2 + len(opt.Label) + 2 // indicator + space + label + spacing
									if clickX >= currentX && clickX < currentX+optWidth-2 {
										if field.Value != opt.Value {
											field.Value = opt.Value
											p.notifyLiveChange(*field)
										}
										break
									}
									currentX += optWidth
								}
							}
						}

					case FormFieldText:
						// Position cursor at click location
						if ti, ok := p.formTextInputs[fieldIdx]; ok {
							// Calculate cursor position from click
							// Text starts at valueStartX
							clickOffset := msg.X - valueStartX
							if clickOffset < 0 {
								clickOffset = 0
							}
							textLen := len(ti.Value())
							cursorPos := clickOffset
							if cursorPos > textLen {
								cursorPos = textLen
							}
							ti.SetCursor(cursorPos)
							p.formTextInputs[fieldIdx] = ti
						}
					}
					return nil
				}
			}
		}

	}
	return nil
}

func (p *PopupPanel) close(result PopupResult) {
	p.visible = false
	if p.onClose != nil {
		p.onClose(result)
	}
}

func (p *PopupPanel) maxScroll() int {
	return max(0, len(p.lines)-p.contentHeight())
}

// View renders the popup.
func (p *PopupPanel) View() string {
	if !p.visible {
		return ""
	}

	width := p.width()
	contentWidth := p.contentWidth()
	contentHeight := p.contentHeight()

	// Border characters and colors
	tl, tr, bl, br := "╭", "╮", "╰", "╯"
	horiz, vert := "─", "│"
	borderFg := p.styles.Theme.Primary

	// Build top border with title
	titleLen := len(p.title) + 2 // +2 for spaces
	leftPad := max(0, (width-2-titleLen)/2)
	rightPad := max(0, width-2-titleLen-leftPad)
	topBorder := ui.Colorize(tl, borderFg) +
		ui.Colorize(strings.Repeat(horiz, leftPad), borderFg) +
		ui.Colorize(" "+p.title+" ", p.styles.Theme.Accent) +
		ui.Colorize(strings.Repeat(horiz, rightPad), borderFg) +
		ui.Colorize(tr, borderFg)

	// Build content based on mode
	var contentLines []string
	switch p.mode {
	case PopupModeDisplay:
		contentLines = p.renderDisplayContent(contentWidth, contentHeight)
	case PopupModeInput:
		contentLines = p.renderInputContent(contentWidth, contentHeight)
	case PopupModeConfirm:
		contentLines = p.renderConfirmContent(contentWidth, contentHeight)
	case PopupModeSelect:
		contentLines = p.renderSelectContent(contentWidth, contentHeight)
	case PopupModeForm:
		contentLines = p.renderFormContent(contentWidth, contentHeight)
	}

	// Wrap content lines with borders
	var wrappedLines []string
	for i, line := range contentLines {
		// Right border - scroll indicator for display and form modes
		rightBorderStr := ui.Colorize(vert, borderFg)
		if p.mode == PopupModeDisplay && p.maxScroll() > 0 {
			thumbPos, thumbSize := p.scrollThumb(len(contentLines))
			if i >= thumbPos && i < thumbPos+thumbSize {
				rightBorderStr = ui.Colorize("┃", borderFg)
			}
		} else if p.mode == PopupModeForm && p.formMaxScroll(contentHeight) > 0 {
			thumbPos, thumbSize := p.formScrollThumb(contentHeight)
			if i >= thumbPos && i < thumbPos+thumbSize {
				rightBorderStr = ui.Colorize("┃", borderFg)
			}
		}
		wrappedLines = append(wrappedLines,
			ui.Colorize(vert, borderFg)+" "+line+" "+rightBorderStr)
	}

	// Build bottom border with hints on the right
	hints := p.getHints()
	hintsLen := lipgloss.Width(hints)
	bottomPad := max(0, width-2-hintsLen)
	bottomBorder := ui.Colorize(bl, borderFg) +
		ui.Colorize(strings.Repeat(horiz, bottomPad), borderFg) +
		hints +
		ui.Colorize(br, borderFg)

	// Combine all
	var result []string
	result = append(result, topBorder)
	result = append(result, wrappedLines...)
	result = append(result, bottomBorder)

	return strings.Join(result, "\n")
}

func (p *PopupPanel) renderDisplayContent(width, height int) []string {
	lines := make([]string, height)
	endLine := min(p.scroll+height, len(p.lines))

	for i := 0; i < height; i++ {
		lineIdx := p.scroll + i
		if lineIdx < endLine {
			lines[i] = p.padRight(p.lines[lineIdx], width)
		} else {
			lines[i] = strings.Repeat(" ", width)
		}
	}
	return lines
}

func (p *PopupPanel) renderInputContent(width, height int) []string {
	lines := make([]string, height)

	// Center the prompt and input vertically
	midY := height / 2

	for i := 0; i < height; i++ {
		if i == midY-1 && p.prompt != "" {
			lines[i] = p.padRight(p.prompt, width)
		} else if i == midY {
			inputView := p.input.View()
			lines[i] = p.padRight(inputView, width)
		} else {
			lines[i] = strings.Repeat(" ", width)
		}
	}
	return lines
}

func (p *PopupPanel) renderConfirmContent(width, height int) []string {
	lines := make([]string, height)

	midY := height / 2

	// Render buttons
	yesStyle := lipgloss.NewStyle()
	noStyle := lipgloss.NewStyle()
	if p.selected == 0 {
		yesStyle = yesStyle.Reverse(true)
	} else {
		noStyle = noStyle.Reverse(true)
	}

	buttons := yesStyle.Render(" "+p.yesLabel+" ") + "  " + noStyle.Render(" "+p.noLabel+" ")

	for i := 0; i < height; i++ {
		if i == midY-1 {
			// Center the confirm message
			lines[i] = p.centerText(p.confirmMsg, width)
		} else if i == midY+1 {
			// Center buttons
			lines[i] = p.centerText(buttons, width)
		} else {
			lines[i] = strings.Repeat(" ", width)
		}
	}
	return lines
}

func (p *PopupPanel) renderSelectContent(width, height int) []string {
	lines := make([]string, height)

	// Calculate starting position to center the options vertically
	numOptions := len(p.selectOptions)
	startY := (height - numOptions) / 2
	if startY < 0 {
		startY = 0
	}

	for i := 0; i < height; i++ {
		optionIdx := i - startY
		if optionIdx >= 0 && optionIdx < numOptions {
			option := p.selectOptions[optionIdx]
			prefix := "  "
			if optionIdx == p.selectCursor {
				prefix = "> "
			}

			label := prefix + option.Label
			if optionIdx == p.selectCursor {
				// Highlight selected option
				label = lipgloss.NewStyle().
					Foreground(p.styles.Theme.Accent).
					Bold(true).
					Render(label)
			}
			lines[i] = p.padRight(label, width)
		} else {
			lines[i] = strings.Repeat(" ", width)
		}
	}
	return lines
}

// fieldHeight returns the number of visual rows a field takes.
func (p *PopupPanel) fieldHeight(field FormField) int {
	if field.Type == FormFieldRadio && field.Vertical {
		return len(field.Options)
	}
	if field.Type == FormFieldColor {
		// radiusY=4 means diameterY=9 (2*R+1)
		return 9
	}
	if field.Type == FormFieldHSL {
		return 3 // Hue, Saturation, Lightness
	}
	if field.Type == FormFieldRGB {
		return 3 // Red, Green, Blue
	}
	return 1
}

// totalFormRows returns the total visual rows for all fields including spacing.
func (p *PopupPanel) totalFormRows() int {
	total := 0
	for i, f := range p.formFields {
		total += p.fieldHeight(f)
		// Add blank line after each field except the last
		if i < len(p.formFields)-1 {
			total++
		}
	}
	return total
}

// fieldAtRow returns the field index and sub-row within that field for a given visual row.
// Returns -1 for fieldIdx if the row is a spacing row between fields.
func (p *PopupPanel) fieldAtRow(row int) (fieldIdx int, subRow int) {
	currentRow := 0
	for i, f := range p.formFields {
		h := p.fieldHeight(f)
		if row < currentRow+h {
			return i, row - currentRow
		}
		currentRow += h
		// Account for spacing row after each field except the last
		if i < len(p.formFields)-1 {
			if row == currentRow {
				// This is a spacing row
				return -1, 0
			}
			currentRow++
		}
	}
	return -1, 0
}


// rowForField returns the starting visual row for a field.
func (p *PopupPanel) rowForField(fieldIdx int) int {
	row := 0
	for i := 0; i < fieldIdx && i < len(p.formFields); i++ {
		row += p.fieldHeight(p.formFields[i])
		// Account for blank line after each field (except last)
		if i < len(p.formFields)-1 {
			row++
		}
	}
	return row
}

func (p *PopupPanel) renderFormContent(width, height int) []string {
	lines := make([]string, height)

	// Calculate layout: fields + blank line + buttons
	totalFieldRows := p.totalFormRows()
	totalRows := totalFieldRows + 2 // fields + blank + button row

	// Determine if scrolling is needed
	needsScroll := totalRows > height

	var startY int
	var viewOffset int
	if needsScroll {
		// Scrolling mode: content starts at row 0, apply scroll offset
		startY = 0
		viewOffset = p.formScroll

		// Clamp scroll (don't modify p.formScroll during render)
		maxScroll := totalRows - height
		if viewOffset > maxScroll {
			viewOffset = maxScroll
		}
		if viewOffset < 0 {
			viewOffset = 0
		}
	} else {
		// Centered mode (no scroll needed)
		startY = (height - totalRows) / 2
		if startY < 0 {
			startY = 0
		}
		viewOffset = 0
	}

	accentStyle := lipgloss.NewStyle().Foreground(p.styles.Theme.Accent).Bold(true)
	mutedStyle := lipgloss.NewStyle().Foreground(p.styles.Theme.Muted)

	for i := 0; i < height; i++ {
		// Calculate which logical row this screen line corresponds to
		// In scroll mode: logical row = screen line + viewOffset
		// In centered mode: logical row = screen line - startY
		var rowIdx int
		if needsScroll {
			rowIdx = i + viewOffset
		} else {
			rowIdx = i - startY
		}

		if rowIdx >= 0 && rowIdx < totalFieldRows {
			// Find which field this row belongs to
			fieldIdx, subRow := p.fieldAtRow(rowIdx)
			if fieldIdx < 0 || fieldIdx >= len(p.formFields) {
				lines[i] = strings.Repeat(" ", width)
				continue
			}

			field := p.formFields[fieldIdx]
			isFocused := !p.formOnButtons && p.formCursor == fieldIdx

			// For multi-row fields, only show label on first row
			prefix := "  "
			if isFocused && subRow == 0 {
				prefix = "> "
			} else if isFocused {
				prefix = "  " // Subsequent rows of focused field
			}

			var valueStr string
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
					valueStr = "[●] " + onLabel
				} else {
					valueStr = "[ ] " + offLabel
				}
			case FormFieldSlider:
				// Render a slider: [====●----] 3/5
				sliderWidth := 10
				if field.Max > 0 {
					filled := field.Value * sliderWidth / field.Max
					empty := sliderWidth - filled
					valueStr = "[" + strings.Repeat("=", filled) + "●" + strings.Repeat("-", empty) + "] "
					valueStr += fmt.Sprintf("%d/%d", field.Value, field.Max)
				} else {
					valueStr = fmt.Sprintf("%d", field.Value)
				}
			case FormFieldText:
				// Render text input - handle separately due to ANSI codes
				var label string
				if isFocused {
					label = accentStyle.Render(fmt.Sprintf("%s%-15s ", prefix, field.Label+":"))
				} else {
					label = fmt.Sprintf("%s%-15s ", prefix, field.Label+":")
				}
				labelWidth := lipgloss.Width(label)

				if ti, ok := p.formTextInputs[rowIdx]; ok {
					// Set text input width to fill remaining space
					inputWidth := width - labelWidth - 2
					if inputWidth < 10 {
						inputWidth = 10
					}
					ti.Width = inputWidth
					p.formTextInputs[rowIdx] = ti
					valueStr = ti.View()
				} else {
					valueStr = field.TextValue
				}

				// Build the line with proper padding
				fullLine := label + valueStr
				lineWidth := lipgloss.Width(fullLine)
				if lineWidth < width {
					fullLine += strings.Repeat(" ", width-lineWidth)
				}
				lines[i] = fullLine
				continue // Skip the common line assignment below

			case FormFieldBrightness:
				// Render brightness slider with visual gradient bar
				sliderWidth := 20
				pct := 0
				if field.Max > 0 {
					pct = field.Value * 100 / field.Max
				}
				filled := field.Value * sliderWidth / max(1, field.Max)

				// Build gradient bar with brightness indication
				bar := ""
				for j := 0; j < sliderWidth; j++ {
					if j < filled {
						// Gradient from dim to bright
						intensity := 180 + (j * 75 / sliderWidth)
						bar += lipgloss.NewStyle().Foreground(lipgloss.Color(fmt.Sprintf("#%02x%02x%02x", intensity, intensity, intensity))).Render("█")
					} else {
						bar += lipgloss.NewStyle().Foreground(p.styles.Theme.Muted).Render("░")
					}
				}
				valueStr = bar + fmt.Sprintf(" %3d%%", pct)

			case FormFieldColorTemp:
				// Render color temperature slider (warm orange to cool blue)
				sliderWidth := 20
				// Mirek range: 153 (cool/6500K) to 500 (warm/2000K)
				// Warm (high mirek) = left side, Cool (low mirek) = right side
				// Invert position: high mirek → left, low mirek → right
				pos := 0
				if field.Max > field.Min {
					pos = (field.Max - field.Value) * sliderWidth / (field.Max - field.Min)
				}
				if pos < 0 {
					pos = 0
				}
				if pos >= sliderWidth {
					pos = sliderWidth - 1
				}

				// Build gradient bar from warm (left) to cool (right)
				bar := ""
				for j := 0; j < sliderWidth; j++ {
					// Gradient from warm orange to cool blue
					warmR, warmG, warmB := 255, 180, 100 // Warm (left, high mirek)
					coolR, coolG, coolB := 150, 200, 255 // Cool (right, low mirek)
					r := warmR + (coolR-warmR)*j/sliderWidth
					g := warmG + (coolG-warmG)*j/sliderWidth
					b := warmB + (coolB-warmB)*j/sliderWidth

					char := "─"
					if j == pos {
						char = "●"
					}
					bar += lipgloss.NewStyle().Foreground(lipgloss.Color(fmt.Sprintf("#%02x%02x%02x", r, g, b))).Render(char)
				}
				// Show Kelvin approximation
				kelvin := 1000000 / max(1, field.Value) // Approximate Kelvin from Mirek
				valueStr = bar + fmt.Sprintf(" %dK", kelvin)

			case FormFieldColor:
				// Color wheel using 2x vertical resolution for smooth edges
				radiusY := 4
				radiusX := 9
				diameterY := 2*radiusY + 1
				centerRow := radiusY
				
				// High-res parameters (2x vertical) with overshoot for rounder appearance
				hiResRadiusY := float64(radiusY * 2)
				// Center is between sub-rows, so use 0.5 offset for symmetry
				hiResCenterY := float64(radiusY*2) + 0.5
				// Overshoot: make effective radii larger for rounder edges
				effectiveRadiusX := float64(radiusX) + 0.5
				effectiveRadiusY := hiResRadiusY + 1.0
				
				// Calculate selection position
				var selRow, selCol int
				if p.formFieldEditing || p.colorPosValid {
					// Use stored position from user interaction
					selRow = p.colorSelRow
					selCol = p.colorSelCol
				} else {
					// First time showing - calculate from XY (approximate)
					// This won't be perfect but is only used before first interaction
					hue, sat := ui.XyToHueSat(field.ColorX, field.ColorY)
					angleRad := float64(90-hue) * math.Pi / 180
					dist := float64(sat) / 100.0
					xNorm := dist * math.Cos(angleRad)
					yNorm := -dist * math.Sin(angleRad)
					
					selCol = radiusX + int(xNorm*float64(radiusX)+0.5)
					selRow = centerRow + int(yNorm*float64(radiusY)+0.5)
					
					if selRow < 0 {
						selRow = 0
					}
					if selRow >= diameterY {
						selRow = diameterY - 1
					}
					if selCol < 0 {
						selCol = 0
					}
					if selCol > radiusX*2 {
						selCol = radiusX * 2
					}
				}
				
				if subRow < diameterY {
					maxCells := radiusX*2 + 1
					valueStr = ""
					
					// For each column, check both sub-pixels (top half and bottom half)
					for col := 0; col < maxCells; col++ {
						dx := col - radiusX // distance from center column
						
						// Check top sub-pixel (subRow*2)
						hiResRowTop := float64(subRow*2)
						dyTop := hiResRowTop - hiResCenterY
						// Ellipse equation with overshoot: (dx/rx)^2 + (dy/ry)^2 <= 1
						topInside := float64(dx*dx)/(effectiveRadiusX*effectiveRadiusX)+(dyTop*dyTop)/(effectiveRadiusY*effectiveRadiusY) <= 1.0
						
						// Check bottom sub-pixel (subRow*2 + 1)
						hiResRowBot := float64(subRow*2 + 1)
						dyBot := hiResRowBot - hiResCenterY
						botInside := float64(dx*dx)/(effectiveRadiusX*effectiveRadiusX)+(dyBot*dyBot)/(effectiveRadiusY*effectiveRadiusY) <= 1.0
						
						if !topInside && !botInside {
							// Neither half is inside - empty
							valueStr += " "
						} else {
							// Calculate color for this cell
							// Use the sub-pixel that's farther from center (more saturated)
							var useY float64
							if topInside && botInside {
								// Both inside - use average Y for color calculation
								useY = float64(subRow - centerRow)
							} else if topInside {
								useY = float64(dyTop) / 2.0 // Scale back to normal resolution
							} else {
								useY = float64(dyBot) / 2.0
							}
							
							xNorm := float64(dx) / float64(radiusX)
							yNorm := useY / float64(radiusY)
							dist := math.Sqrt(xNorm*xNorm + yNorm*yNorm)
							
							// Calculate angle and reverse direction for standard color wheel
							// Red at right, going counter-clockwise: Red→Magenta→Blue→Cyan→Green→Yellow→Red
							angle := math.Atan2(-yNorm, xNorm)
							blockHue := -int(angle*180/math.Pi) + 90 // Rotate to put Red at right
							blockHue = ((blockHue % 360) + 360) % 360
							
							blockSat := int(dist * 100)
							if blockSat > 100 {
								blockSat = 100
							}
							
							cr, cg, cb := ui.HsvToRGB(blockHue, blockSat, 100)
							blockColor := fmt.Sprintf("#%02X%02X%02X", cr, cg, cb)
							
							// Check if this is the selected cell
							isSelected := subRow == selRow && col == selCol
							
							if isSelected {
								// Blink the selected cell - alternate between color and background
								bgColor := "#1F2937" // Theme background color
								if p.colorBlinkOn {
									// Blink ON - show the color
									if topInside && botInside {
										valueStr += lipgloss.NewStyle().Background(lipgloss.Color(blockColor)).Render(" ")
									} else if topInside {
										valueStr += lipgloss.NewStyle().Foreground(lipgloss.Color(blockColor)).Render("▀")
									} else {
										valueStr += lipgloss.NewStyle().Foreground(lipgloss.Color(blockColor)).Render("▄")
									}
								} else {
									// Blink OFF - show background color (hide the selection)
									if topInside && botInside {
										valueStr += lipgloss.NewStyle().Background(lipgloss.Color(bgColor)).Render(" ")
									} else if topInside {
										// Top half was color, now show bg; bottom is already bg
										valueStr += lipgloss.NewStyle().Foreground(lipgloss.Color(bgColor)).Render("▀")
									} else {
										// Bottom half was color, now show bg; top is already bg
										valueStr += lipgloss.NewStyle().Foreground(lipgloss.Color(bgColor)).Render("▄")
									}
								}
							} else if topInside && botInside {
								// Both halves inside - full block
								valueStr += lipgloss.NewStyle().Background(lipgloss.Color(blockColor)).Render(" ")
							} else if topInside {
								// Only top half inside
								valueStr += lipgloss.NewStyle().Foreground(lipgloss.Color(blockColor)).Render("▀")
							} else {
								// Only bottom half inside
								valueStr += lipgloss.NewStyle().Foreground(lipgloss.Color(blockColor)).Render("▄")
							}
						}
					}
				}

			case FormFieldSelect:
				// Render select - show current selection with dropdown indicator
				currentLabel := "---"
				for _, opt := range field.Options {
					if opt.Value == field.Value {
						currentLabel = opt.Label
						break
					}
				}
				if isFocused {
					valueStr = fmt.Sprintf("[%s] ▼", currentLabel)
				} else {
					valueStr = fmt.Sprintf("[%s]", currentLabel)
				}

			case FormFieldRadio:
				if field.Vertical {
					// Vertical: each subRow is one option
					if subRow < len(field.Options) {
						opt := field.Options[subRow]
						indicator := "○"
						if opt.Value == field.Value {
							indicator = "●"
						}
						optStr := fmt.Sprintf("%s %s", indicator, opt.Label)
						// Highlight selected option when in edit mode
						if isFocused && p.formFieldEditing && opt.Value == field.Value {
							optStr = accentStyle.Render(optStr)
						}
						valueStr = optStr
					}
				} else {
					// Horizontal: render all options inline
					var parts []string
					for _, opt := range field.Options {
						indicator := "○"
						if opt.Value == field.Value {
							indicator = "●"
						}
						optStr := fmt.Sprintf("%s %s", indicator, opt.Label)
						// Highlight selected option when in edit mode
						if isFocused && p.formFieldEditing && opt.Value == field.Value {
							optStr = accentStyle.Render(optStr)
						}
						parts = append(parts, optStr)
					}
					valueStr = strings.Join(parts, "  ")
				}

			case FormFieldHSL:
				// HSL picker - 3 rows: Hue, Saturation, Lightness
				sliderWidth := 20
				rowLabels := []string{"H", "S", "L"}
				rowValues := []int{field.Hue, field.Saturation, field.Lightness}
				rowMaxes := []int{360, 100, 100}

				if subRow < 3 {
					rowVal := rowValues[subRow]
					rowMax := rowMaxes[subRow]
					pos := rowVal * sliderWidth / max(1, rowMax)
					if pos >= sliderWidth {
						pos = sliderWidth - 1
					}

					var bar string
					if subRow == 0 {
						// Hue - rainbow gradient
						for j := 0; j < sliderWidth; j++ {
							h := j * 360 / sliderWidth
							r, g, b := ui.HsvToRGB(h, 100, 100)
							char := "─"
							if j == pos {
								char = "●"
							}
							bar += lipgloss.NewStyle().Foreground(lipgloss.Color(fmt.Sprintf("#%02x%02x%02x", r, g, b))).Render(char)
						}
					} else if subRow == 1 {
						// Saturation - gray to full color
						for j := 0; j < sliderWidth; j++ {
							s := j * 100 / sliderWidth
							r, g, b := ui.HsvToRGB(field.Hue, s, 100)
							char := "─"
							if j == pos {
								char = "●"
							}
							bar += lipgloss.NewStyle().Foreground(lipgloss.Color(fmt.Sprintf("#%02x%02x%02x", r, g, b))).Render(char)
						}
					} else {
						// Lightness - black to white through color
						for j := 0; j < sliderWidth; j++ {
							l := j * 100 / sliderWidth
							r, g, b := ui.HsvToRGB(field.Hue, field.Saturation, l)
							char := "─"
							if j == pos {
								char = "●"
							}
							bar += lipgloss.NewStyle().Foreground(lipgloss.Color(fmt.Sprintf("#%02x%02x%02x", r, g, b))).Render(char)
						}
					}

					// Row highlight when editing
					rowLabel := rowLabels[subRow]
					if isFocused && p.formFieldEditing && field.HSLSliderFocus == subRow {
						rowLabel = accentStyle.Render(rowLabel)
					}
					valueStr = fmt.Sprintf("%s: %s %3d", rowLabel, bar, rowVal)

					// Show color preview as vertical stripe with rounded corners
					pr, pg, pb := ui.HsvToRGB(field.Hue, field.Saturation, field.Lightness)
					colorHex := fmt.Sprintf("#%02x%02x%02x", pr, pg, pb)
					colorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(colorHex))
					bgStyle := lipgloss.NewStyle().Background(lipgloss.Color(colorHex))
					var preview string
					switch subRow {
					case 0: // Top row - rounded corners
						preview = " " + colorStyle.Render("▄▄") + " "
					case 2: // Bottom row - rounded corners
						preview = " " + colorStyle.Render("▀▀") + " "
					default: // Middle row - full block
						preview = bgStyle.Render("    ")
					}
					valueStr += " " + preview
				}

			case FormFieldRGB:
				// RGB picker - 3 rows: Red, Green, Blue
				sliderWidth := 20
				rowLabels := []string{"R", "G", "B"}
				rowValues := []int{field.Red, field.Green, field.Blue}

				if subRow < 3 {
					rowVal := rowValues[subRow]
					pos := rowVal * sliderWidth / 255
					if pos >= sliderWidth {
						pos = sliderWidth - 1
					}

					var bar string
					for j := 0; j < sliderWidth; j++ {
						intensity := j * 255 / sliderWidth
						var r, g, b int
						switch subRow {
						case 0: // Red slider
							r, g, b = intensity, 0, 0
						case 1: // Green slider
							r, g, b = 0, intensity, 0
						case 2: // Blue slider
							r, g, b = 0, 0, intensity
						}
						char := "─"
						if j == pos {
							char = "●"
						}
						bar += lipgloss.NewStyle().Foreground(lipgloss.Color(fmt.Sprintf("#%02x%02x%02x", r, g, b))).Render(char)
					}

					// Row highlight when editing
					rowLabel := rowLabels[subRow]
					if isFocused && p.formFieldEditing && field.RGBSliderFocus == subRow {
						rowLabel = accentStyle.Render(rowLabel)
					}
					valueStr = fmt.Sprintf("%s: %s %3d", rowLabel, bar, rowVal)

					// Show color preview as vertical stripe with rounded corners
					colorHex := fmt.Sprintf("#%02x%02x%02x", field.Red, field.Green, field.Blue)
					colorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(colorHex))
					bgStyle := lipgloss.NewStyle().Background(lipgloss.Color(colorHex))
					var preview string
					switch subRow {
					case 0: // Top row - rounded corners
						preview = " " + colorStyle.Render("▄▄") + " "
					case 2: // Bottom row - rounded corners
						preview = " " + colorStyle.Render("▀▀") + " "
					default: // Middle row - full block
						preview = bgStyle.Render("    ")
					}
					valueStr += " " + preview
				}
			}

			// Build label - only show on first row of multi-row fields
			var label string
			if subRow == 0 {
				label = fmt.Sprintf("%s%-15s ", prefix, field.Label+":")
				if isFocused {
					label = accentStyle.Render(label)
				}
			} else {
				// Subsequent rows: just indent to match label width
				label = strings.Repeat(" ", 18)
			}
			lines[i] = p.padRight(label+valueStr, width)

		} else if rowIdx == totalFieldRows {
			// Blank line before buttons
			lines[i] = strings.Repeat(" ", width)

		} else if rowIdx == totalFieldRows+1 {
			// Render buttons
			var buttons string
			if p.formLiveMode {
				// Live mode - just show Close (changes already applied)
				closeStyle := lipgloss.NewStyle()
				if p.formOnButtons {
					closeStyle = closeStyle.Reverse(true)
				}
				closeLabel := closeStyle.Render(" Close ")
				buttons = closeLabel
			} else {
				// Edit mode - show Save/Cancel
				saveStyle := lipgloss.NewStyle()
				cancelStyle := lipgloss.NewStyle()

				if p.formOnButtons && p.formBtnIndex == 0 {
					saveStyle = saveStyle.Reverse(true)
				} else if p.formOnButtons && p.formBtnIndex == 1 {
					cancelStyle = cancelStyle.Reverse(true)
				}

				hasChanges := p.formHasChanges()
				saveLabel := " Save "
				if !hasChanges {
					saveLabel = mutedStyle.Render(saveLabel)
				} else {
					saveLabel = saveStyle.Render(saveLabel)
				}
				cancelLabel := cancelStyle.Render(" Cancel ")
				buttons = saveLabel + "  " + cancelLabel
			}
			lines[i] = p.centerText(buttons, width)

		} else {
			lines[i] = strings.Repeat(" ", width)
		}
	}

	// Overlay dropdown if open
	if p.selectDropdownOpen && p.formCursor < len(p.formFields) {
		field := p.formFields[p.formCursor]
		if field.Type == FormFieldSelect && len(field.Options) > 0 {
			lines = p.overlayDropdown(lines, width, startY)
		}
	}

	return lines
}

// handleDropdownMouse handles mouse events when the dropdown is open.
func (p *PopupPanel) handleDropdownMouse(msg tea.MouseMsg, popupX, popupY, startY, valueStartX int) tea.Cmd {
	if p.formCursor >= len(p.formFields) {
		return nil
	}
	field := &p.formFields[p.formCursor]
	numOptions := len(field.Options)
	if numOptions == 0 {
		return nil
	}

	// Ignore mouse events immediately after opening (prevents open-then-close on same click)
	if p.selectDropdownJustOpened {
		if msg.Type == tea.MouseLeft || msg.Type == tea.MouseRelease {
			p.selectDropdownJustOpened = false
			return func() tea.Msg { return nil } // Consume the event
		}
	}

	// Calculate dropdown position
	fieldRow := p.rowForField(p.formCursor)
	dropdownStartRow := startY + fieldRow + 1
	dropdownStartY := popupY + 1 + dropdownStartRow // +1 for top border
	maxVisible := p.selectDropdownMaxVisible()
	if maxVisible > numOptions {
		maxVisible = numOptions
	}

	// Handle scroll wheel without checking Action
	switch msg.Button {
	case tea.MouseButtonWheelUp:
		if p.selectDropdownCursor > 0 {
			p.selectDropdownCursor--
			p.ensureDropdownCursorVisible()
		}
		return func() tea.Msg { return nil } // Return non-nil to indicate handled

	case tea.MouseButtonWheelDown:
		if p.selectDropdownCursor < numOptions-1 {
			p.selectDropdownCursor++
			p.ensureDropdownCursorVisible()
		}
		return func() tea.Msg { return nil }

	case tea.MouseButtonLeft:
		if msg.Action == tea.MouseActionPress {
			// Check if click is on a dropdown option
			for i := 0; i < maxVisible; i++ {
				optIdx := p.selectDropdownScroll + i
				if optIdx >= numOptions {
					break
				}
				optionY := dropdownStartY + i
				if msg.Y == optionY && msg.X >= valueStartX {
					// Select this option
					field.Value = field.Options[optIdx].Value
					p.notifyLiveChange(*field)
					p.closeSelectDropdown()
					return func() tea.Msg { return nil }
				}
			}

			// Click outside dropdown - close it
			p.closeSelectDropdown()
			return func() tea.Msg { return nil }
		}
	}

	return nil
}

// overlayDropdown renders the dropdown options over the form lines.
func (p *PopupPanel) overlayDropdown(lines []string, width int, startY int) []string {
	if p.formCursor >= len(p.formFields) {
		return lines
	}
	field := p.formFields[p.formCursor]
	numOptions := len(field.Options)
	if numOptions == 0 {
		return lines
	}

	// Calculate dropdown position (below the select field)
	fieldRow := p.rowForField(p.formCursor)
	dropdownStartRow := startY + fieldRow + 1
	maxVisible := p.selectDropdownMaxVisible()
	if maxVisible > numOptions {
		maxVisible = numOptions
	}

	// Calculate label width to align dropdown with value area
	labelWidth := 18 // "  " + "%-15s " = 2 + 15 + 1 = 18

	// Dropdown box styling
	borderColor := p.styles.Theme.Primary
	bgStyle := lipgloss.NewStyle().Background(lipgloss.Color("#1a1a2e"))
	selectedStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#ffffff")).
		Background(p.styles.Theme.Accent).
		Bold(true)

	// Calculate dropdown width
	maxLabelLen := 0
	for _, opt := range field.Options {
		if len(opt.Label) > maxLabelLen {
			maxLabelLen = len(opt.Label)
		}
	}
	dropdownWidth := maxLabelLen + 4 // padding
	if dropdownWidth > width-labelWidth-3 { // -3 for borders and scrollbar
		dropdownWidth = width - labelWidth - 3
	}

	// Calculate scrollbar position if needed
	needsScrollbar := numOptions > maxVisible
	var scrollThumbStart, scrollThumbSize int
	if needsScrollbar {
		// Calculate thumb size (minimum 1)
		scrollThumbSize = max(1, maxVisible*maxVisible/numOptions)
		// Calculate thumb position
		scrollRange := numOptions - maxVisible
		thumbRange := maxVisible - scrollThumbSize
		if scrollRange > 0 && thumbRange > 0 {
			scrollThumbStart = p.selectDropdownScroll * thumbRange / scrollRange
		}
	}

	// Render dropdown options
	for i := 0; i < maxVisible; i++ {
		optIdx := p.selectDropdownScroll + i
		if optIdx >= numOptions {
			break
		}

		lineIdx := dropdownStartRow + i
		if lineIdx >= len(lines) {
			break
		}

		opt := field.Options[optIdx]
		optLabel := opt.Label
		if len(optLabel) > dropdownWidth-2 {
			optLabel = optLabel[:dropdownWidth-3] + "…"
		}

		// Pad the option label
		paddedLabel := fmt.Sprintf(" %-*s", dropdownWidth-2, optLabel)

		// Apply styling
		var styledOption string
		if optIdx == p.selectDropdownCursor {
			styledOption = selectedStyle.Render(paddedLabel)
		} else {
			styledOption = bgStyle.Foreground(lipgloss.Color("#cccccc")).Render(paddedLabel)
		}

		// Build the dropdown border
		leftBorder := lipgloss.NewStyle().Foreground(borderColor).Render("│")

		// Right border with scrollbar
		var rightBorder string
		if needsScrollbar {
			// Determine if this row is part of the scrollbar thumb
			if i >= scrollThumbStart && i < scrollThumbStart+scrollThumbSize {
				rightBorder = lipgloss.NewStyle().Foreground(p.styles.Theme.Accent).Render("┃")
			} else {
				rightBorder = lipgloss.NewStyle().Foreground(p.styles.Theme.Muted).Render("│")
			}
		} else {
			rightBorder = lipgloss.NewStyle().Foreground(borderColor).Render("│")
		}

		// Overlay on the line
		line := lines[lineIdx]
		// Keep the label portion, overlay the dropdown
		labelPart := ""
		if len(line) >= labelWidth {
			labelPart = line[:labelWidth]
		} else {
			labelPart = p.padRight(line, labelWidth)
		}

		// Build new line with dropdown
		dropdownContent := leftBorder + styledOption + rightBorder
		remainingWidth := width - labelWidth - lipgloss.Width(dropdownContent)
		if remainingWidth < 0 {
			remainingWidth = 0
		}
		lines[lineIdx] = labelPart + dropdownContent + strings.Repeat(" ", remainingWidth)
	}

	return lines
}

// formHasChanges returns true if any form field has been modified.
func (p *PopupPanel) formHasChanges() bool {
	for i, f := range p.formFields {
		switch f.Type {
		case FormFieldText:
			if ti, ok := p.formTextInputs[i]; ok {
				if ti.Value() != f.OriginalText {
					return true
				}
			} else if f.TextValue != f.OriginalText {
				return true
			}
		case FormFieldColor:
			if f.ColorX != f.OriginalColorX || f.ColorY != f.OriginalColorY {
				return true
			}
		default:
			if f.Value != f.Original {
				return true
			}
		}
	}
	return false
}

func (p *PopupPanel) getHints() string {
	var hint string
	switch p.mode {
	case PopupModeDisplay:
		// Just scroll position - close hint goes in status bar
		hint = fmt.Sprintf(" %d of %d ", p.scroll+1, max(1, len(p.lines)))
	case PopupModeInput:
		hint = " Enter:submit  Esc:cancel "
	case PopupModeConfirm:
		hint = " y:yes  n:no  Enter:confirm "
	case PopupModeSelect:
		hint = " ↑↓:select  Enter:confirm  Esc:cancel "
	case PopupModeForm:
		// Check if dropdown is open
		if p.selectDropdownOpen {
			hint = " ↑↓:select  Enter:confirm  Esc:close "
		} else if p.formFieldEditing {
			// In edit mode
			isTextFocused := p.formCursor < len(p.formFields) &&
				p.formFields[p.formCursor].Type == FormFieldText
			isRadio := p.formCursor < len(p.formFields) &&
				p.formFields[p.formCursor].Type == FormFieldRadio
			isVerticalRadio := isRadio && p.formFields[p.formCursor].Vertical
			isColor := p.formCursor < len(p.formFields) &&
				p.formFields[p.formCursor].Type == FormFieldColor
			if isTextFocused {
				hint = " ^T:Title  ^L:sentence  Enter/Esc:done "
			} else if isVerticalRadio {
				hint = " ↑↓:select  Enter:confirm  Esc:cancel "
			} else if isRadio {
				hint = " ←→:select  Enter:confirm  Esc:cancel "
			} else if isColor {
				hint = " ←→↑↓:move  Enter:confirm  Esc:cancel "
			} else {
				hint = " Enter/Esc:done "
			}
		} else {
			// Not editing - normal navigation
			isSelectFocused := !p.formOnButtons && p.formCursor < len(p.formFields) &&
				p.formFields[p.formCursor].Type == FormFieldSelect
			isTextFocused := !p.formOnButtons && p.formCursor < len(p.formFields) &&
				p.formFields[p.formCursor].Type == FormFieldText
			isRadioFocused := !p.formOnButtons && p.formCursor < len(p.formFields) &&
				p.formFields[p.formCursor].Type == FormFieldRadio
			isColorFocused := !p.formOnButtons && p.formCursor < len(p.formFields) &&
				p.formFields[p.formCursor].Type == FormFieldColor
			if isSelectFocused {
				hint = " Enter/Space:open  ↑↓:navigate  Esc:cancel "
			} else if isTextFocused || isRadioFocused || isColorFocused {
				hint = " Enter:edit  ↑↓:navigate  Esc:cancel "
			} else if p.formLiveMode {
				hint = " ←→:adjust  Enter:toggle live  Esc:done "
			} else if p.formOnChange != nil {
				hint = " ←→:adjust  Enter:live mode  ↑↓:navigate  Esc:cancel "
			} else {
				hint = " ←→:adjust  ↑↓:navigate  Enter:save  Esc:cancel "
			}
		}
	}
	return ui.Colorize(hint, p.styles.Theme.Muted)
}

func (p *PopupPanel) scrollThumb(viewHeight int) (pos int, size int) {
	total := len(p.lines)
	if total <= viewHeight {
		return 0, viewHeight
	}
	size = max(1, viewHeight*viewHeight/total)
	scrollRange := total - viewHeight
	posRange := viewHeight - size
	if scrollRange > 0 {
		pos = p.scroll * posRange / scrollRange
	}
	return pos, size
}

// formMaxScroll returns the maximum scroll offset for form mode.
func (p *PopupPanel) formMaxScroll(viewHeight int) int {
	totalRows := p.totalFormRows() + 2 // fields + blank + button row
	if totalRows <= viewHeight {
		return 0
	}
	return totalRows - viewHeight
}

// ensureFocusedFieldVisible adjusts formScroll to keep the focused field in view.
func (p *PopupPanel) ensureFocusedFieldVisible() {
	height := p.contentHeight()
	totalRows := p.totalFormRows() + 2 // fields + blank + button row
	if totalRows <= height {
		return // No scrolling needed
	}

	maxScroll := totalRows - height

	if !p.formOnButtons && p.formCursor < len(p.formFields) {
		focusedFieldRow := p.rowForField(p.formCursor)
		focusedFieldHeight := p.fieldHeight(p.formFields[p.formCursor])
		// Scroll up if needed
		if focusedFieldRow < p.formScroll {
			p.formScroll = focusedFieldRow
		}
		// Scroll down if needed
		if focusedFieldRow+focusedFieldHeight > p.formScroll+height {
			p.formScroll = focusedFieldRow + focusedFieldHeight - height
		}
	} else if p.formOnButtons {
		// Buttons are focused, ensure button row is visible
		buttonRow := p.totalFormRows() + 1
		if buttonRow >= p.formScroll+height {
			p.formScroll = buttonRow - height + 1
		}
	}

	// Clamp scroll
	if p.formScroll > maxScroll {
		p.formScroll = maxScroll
	}
	if p.formScroll < 0 {
		p.formScroll = 0
	}
}

// formScrollThumb returns the position and size of the scroll thumb for form mode.
func (p *PopupPanel) formScrollThumb(viewHeight int) (pos int, size int) {
	totalRows := p.totalFormRows() + 2 // fields + blank + button row
	if totalRows <= viewHeight {
		return 0, viewHeight
	}
	size = max(1, viewHeight*viewHeight/totalRows)
	scrollRange := totalRows - viewHeight
	posRange := viewHeight - size
	if scrollRange > 0 {
		pos = p.formScroll * posRange / scrollRange
	}
	return pos, size
}

func (p *PopupPanel) padRight(s string, width int) string {
	sLen := lipgloss.Width(s)
	if sLen >= width {
		return s[:min(len(s), width)]
	}
	return s + strings.Repeat(" ", width-sLen)
}

func (p *PopupPanel) centerText(s string, width int) string {
	sLen := lipgloss.Width(s)
	if sLen >= width {
		return s[:min(len(s), width)]
	}
	leftPad := (width - sLen) / 2
	rightPad := width - sLen - leftPad
	return strings.Repeat(" ", leftPad) + s + strings.Repeat(" ", rightPad)
}

// Color and text helper functions moved to internal/ui/color.go:
// - ui.HsvToRGB, ui.XyToRGB, ui.MirekToRGB
// - ui.XyToHueSat, ui.HueSatToXY
// - ui.RotateColor, ui.AdjustSaturation, ui.ClampFloat
// - ui.ToTitleCase, ui.ToSentenceCase
