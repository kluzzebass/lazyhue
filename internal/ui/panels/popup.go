// Package panels provides UI panel components.
package panels

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/kluzzebass/lazyhue/internal/ui"
)

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
	FormFieldToggle FormFieldType = iota // Boolean toggle
	FormFieldSlider                      // Integer slider with min/max
)

// FormField represents a single field in a form.
type FormField struct {
	ID       string        // Unique identifier
	Label    string        // Display label
	Type     FormFieldType // Field type
	Value    int           // Current value (0/1 for toggle, actual value for slider)
	Min      int           // Minimum value (for slider)
	Max      int           // Maximum value (for slider)
	Original int           // Original value (to detect changes)
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
	formFields    []FormField
	formCursor    int  // Which field is focused
	formOnButtons bool // true when focus is on Save/Cancel buttons
	formBtnIndex  int  // 0 = Save, 1 = Cancel

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
func (p *PopupPanel) ShowForm(title string, fields []FormField, onClose func(PopupResult, []FormField)) {
	p.mode = PopupModeForm
	p.title = title
	// Copy fields so we can track original values
	p.formFields = make([]FormField, len(fields))
	for i, f := range fields {
		f.Original = f.Value
		p.formFields[i] = f
	}
	p.formCursor = 0
	p.formOnButtons = false
	p.formBtnIndex = 0
	p.onClose = func(result PopupResult) {
		onClose(result, p.formFields)
	}
	p.visible = true
}

// GetFormFields returns the current form field values.
func (p *PopupPanel) GetFormFields() []FormField {
	return p.formFields
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
	return p.height() - 2 // borders
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

	switch key {
	case "esc":
		p.close(PopupResult{Confirmed: false})
		return nil

	case "enter":
		if p.formOnButtons {
			if p.formBtnIndex == 0 {
				// Save
				p.close(PopupResult{Confirmed: true})
			} else {
				// Cancel
				p.close(PopupResult{Confirmed: false})
			}
			return nil
		}
		// Enter on a field - for toggle, toggle it; for slider, do nothing special
		if p.formCursor < len(p.formFields) {
			field := &p.formFields[p.formCursor]
			if field.Type == FormFieldToggle {
				field.Value = 1 - field.Value
			}
		}

	case "tab", "down", "j":
		if p.formOnButtons {
			// Cycle between Save/Cancel
			p.formBtnIndex = (p.formBtnIndex + 1) % 2
		} else {
			// Move to next field, or to buttons
			if p.formCursor < len(p.formFields)-1 {
				p.formCursor++
			} else {
				p.formOnButtons = true
				p.formBtnIndex = 0
			}
		}

	case "shift+tab", "up", "k":
		if p.formOnButtons {
			if p.formBtnIndex > 0 {
				p.formBtnIndex--
			} else {
				// Move back to last field
				p.formOnButtons = false
				p.formCursor = len(p.formFields) - 1
			}
		} else {
			if p.formCursor > 0 {
				p.formCursor--
			}
		}

	case " ":
		// Space toggles boolean fields
		if !p.formOnButtons && p.formCursor < len(p.formFields) {
			field := &p.formFields[p.formCursor]
			if field.Type == FormFieldToggle {
				field.Value = 1 - field.Value
			}
		}

	case "left", "h":
		// Decrease slider value
		if !p.formOnButtons && p.formCursor < len(p.formFields) {
			field := &p.formFields[p.formCursor]
			if field.Type == FormFieldSlider && field.Value > field.Min {
				field.Value--
			}
		} else if p.formOnButtons {
			p.formBtnIndex = 0
		}

	case "right", "l":
		// Increase slider value
		if !p.formOnButtons && p.formCursor < len(p.formFields) {
			field := &p.formFields[p.formCursor]
			if field.Type == FormFieldSlider && field.Value < field.Max {
				field.Value++
			}
		} else if p.formOnButtons {
			p.formBtnIndex = 1
		}
	}

	return nil
}

func (p *PopupPanel) handleMouse(msg tea.MouseMsg) tea.Cmd {
	switch p.mode {
	case PopupModeDisplay:
		switch msg.Type {
		case tea.MouseWheelUp:
			if p.scroll > 0 {
				p.scroll--
			}
		case tea.MouseWheelDown:
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

		switch msg.Type {
		case tea.MouseWheelUp:
			if p.selectCursor > 0 {
				p.selectCursor--
			}
		case tea.MouseWheelDown:
			if p.selectCursor < len(p.selectOptions)-1 {
				p.selectCursor++
			}
		case tea.MouseLeft:
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
	topBorder := p.colorize(tl, borderFg) +
		p.colorize(strings.Repeat(horiz, leftPad), borderFg) +
		p.colorize(" "+p.title+" ", p.styles.Theme.Accent) +
		p.colorize(strings.Repeat(horiz, rightPad), borderFg) +
		p.colorize(tr, borderFg)

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
		// Right border - scroll indicator for display mode
		rightBorderStr := p.colorize(vert, borderFg)
		if p.mode == PopupModeDisplay && p.maxScroll() > 0 {
			thumbPos, thumbSize := p.scrollThumb(len(contentLines))
			if i >= thumbPos && i < thumbPos+thumbSize {
				rightBorderStr = p.colorize("┃", borderFg)
			}
		}
		wrappedLines = append(wrappedLines,
			p.colorize(vert, borderFg)+" "+line+" "+rightBorderStr)
	}

	// Build bottom border with hints on the right
	hints := p.getHints()
	hintsLen := lipgloss.Width(hints)
	bottomPad := max(0, width-2-hintsLen)
	bottomBorder := p.colorize(bl, borderFg) +
		p.colorize(strings.Repeat(horiz, bottomPad), borderFg) +
		hints +
		p.colorize(br, borderFg)

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

func (p *PopupPanel) renderFormContent(width, height int) []string {
	lines := make([]string, height)

	// Calculate layout: fields + blank line + buttons
	numFields := len(p.formFields)
	totalRows := numFields + 2 // fields + blank + button row
	startY := (height - totalRows) / 2
	if startY < 0 {
		startY = 0
	}

	accentStyle := lipgloss.NewStyle().Foreground(p.styles.Theme.Accent).Bold(true)
	mutedStyle := lipgloss.NewStyle().Foreground(p.styles.Theme.Muted)

	for i := 0; i < height; i++ {
		rowIdx := i - startY

		if rowIdx >= 0 && rowIdx < numFields {
			// Render a form field
			field := p.formFields[rowIdx]
			isFocused := !p.formOnButtons && p.formCursor == rowIdx

			prefix := "  "
			if isFocused {
				prefix = "> "
			}

			var valueStr string
			switch field.Type {
			case FormFieldToggle:
				if field.Value != 0 {
					valueStr = "[●] On "
				} else {
					valueStr = "[ ] Off"
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
			}

			label := fmt.Sprintf("%s%-20s %s", prefix, field.Label+":", valueStr)
			if isFocused {
				label = accentStyle.Render(label)
			}
			lines[i] = p.padRight(label, width)

		} else if rowIdx == numFields {
			// Blank line before buttons
			lines[i] = strings.Repeat(" ", width)

		} else if rowIdx == numFields+1 {
			// Render Save/Cancel buttons
			saveStyle := lipgloss.NewStyle()
			cancelStyle := lipgloss.NewStyle()

			if p.formOnButtons && p.formBtnIndex == 0 {
				saveStyle = saveStyle.Reverse(true)
			} else if p.formOnButtons && p.formBtnIndex == 1 {
				cancelStyle = cancelStyle.Reverse(true)
			}

			// Check if any fields changed
			hasChanges := false
			for _, f := range p.formFields {
				if f.Value != f.Original {
					hasChanges = true
					break
				}
			}

			saveLabel := " Save "
			if !hasChanges {
				saveLabel = mutedStyle.Render(saveLabel)
			} else {
				saveLabel = saveStyle.Render(saveLabel)
			}
			cancelLabel := cancelStyle.Render(" Cancel ")

			buttons := saveLabel + "  " + cancelLabel
			lines[i] = p.centerText(buttons, width)

		} else {
			lines[i] = strings.Repeat(" ", width)
		}
	}
	return lines
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
		hint = " ↑↓:navigate  ←→:adjust  Space:toggle  Tab:next "
	}
	return p.colorize(hint, p.styles.Theme.Muted)
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

func (p *PopupPanel) colorize(s string, color lipgloss.Color) string {
	return lipgloss.NewStyle().Foreground(color).Render(s)
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
