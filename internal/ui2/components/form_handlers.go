// Package components provides reusable UI components for the v2 UI.
package components

// handleRadioKey handles keyboard input when editing radio buttons.
func (f *Form) handleRadioKey(keyStr string, field *FormField) bool {
	switch keyStr {
	case "esc":
		field.Value = f.RadioOriginalValue
		f.Editing = false
		return true
	case "enter":
		f.notifyChange(*field)
		f.Editing = false
		return true
	case "up", "k":
		if field.Vertical && f.RadioEditCursor > 0 {
			f.RadioEditCursor--
			field.Value = field.Options[f.RadioEditCursor].Value
		}
		return true
	case "down", "j":
		if field.Vertical && f.RadioEditCursor < len(field.Options)-1 {
			f.RadioEditCursor++
			field.Value = field.Options[f.RadioEditCursor].Value
		}
		return true
	case "left", "h":
		if !field.Vertical && f.RadioEditCursor > 0 {
			f.RadioEditCursor--
			field.Value = field.Options[f.RadioEditCursor].Value
		}
		return true
	case "right", "l":
		if !field.Vertical && f.RadioEditCursor < len(field.Options)-1 {
			f.RadioEditCursor++
			field.Value = field.Options[f.RadioEditCursor].Value
		}
		return true
	}
	return false
}

// handleHSLKey handles keyboard input when editing HSL color picker.
func (f *Form) handleHSLKey(keyStr string, field *FormField) bool {
	sliderFocus := f.HSLSliderFocus[f.Cursor]
	switch keyStr {
	case "esc":
		if orig, ok := f.HSLOriginalValues[f.Cursor]; ok {
			field.Hue = orig.Hue
			field.Saturation = orig.Sat
			field.Lightness = orig.Light
		}
		f.Editing = false
		return true
	case "enter":
		f.notifyChange(*field)
		f.Editing = false
		return true
	case "up", "k":
		if sliderFocus > 0 {
			f.HSLSliderFocus[f.Cursor] = sliderFocus - 1
		}
		return true
	case "down", "j":
		if sliderFocus < 2 {
			f.HSLSliderFocus[f.Cursor] = sliderFocus + 1
		}
		return true
	case "left", "h":
		switch sliderFocus {
		case 0: // Hue
			field.Hue = max(0, field.Hue-5)
		case 1: // Saturation
			field.Saturation = max(0, field.Saturation-5)
		case 2: // Lightness
			field.Lightness = max(0, field.Lightness-5)
		}
		f.debounceSave(field)
		return true
	case "right", "l":
		switch sliderFocus {
		case 0: // Hue
			field.Hue = min(360, field.Hue+5)
		case 1: // Saturation
			field.Saturation = min(100, field.Saturation+5)
		case 2: // Lightness
			field.Lightness = min(100, field.Lightness+5)
		}
		f.debounceSave(field)
		return true
	}
	return false
}

// handleRGBKey handles keyboard input when editing RGB color picker.
func (f *Form) handleRGBKey(keyStr string, field *FormField) bool {
	sliderFocus := f.RGBSliderFocus[f.Cursor]
	switch keyStr {
	case "esc":
		if orig, ok := f.RGBOriginalValues[f.Cursor]; ok {
			field.Red = orig.Red
			field.Green = orig.Green
			field.Blue = orig.Blue
		}
		f.Editing = false
		return true
	case "enter":
		f.notifyChange(*field)
		f.Editing = false
		return true
	case "up", "k":
		if sliderFocus > 0 {
			f.RGBSliderFocus[f.Cursor] = sliderFocus - 1
		}
		return true
	case "down", "j":
		if sliderFocus < 2 {
			f.RGBSliderFocus[f.Cursor] = sliderFocus + 1
		}
		return true
	case "left", "h":
		switch sliderFocus {
		case 0: // Red
			field.Red = max(0, field.Red-5)
		case 1: // Green
			field.Green = max(0, field.Green-5)
		case 2: // Blue
			field.Blue = max(0, field.Blue-5)
		}
		f.debounceSave(field)
		return true
	case "right", "l":
		switch sliderFocus {
		case 0: // Red
			field.Red = min(255, field.Red+5)
		case 1: // Green
			field.Green = min(255, field.Green+5)
		case 2: // Blue
			field.Blue = min(255, field.Blue+5)
		}
		f.debounceSave(field)
		return true
	}
	return false
}
