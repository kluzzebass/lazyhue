// Package app2 is the v2 application using bubbletea v2, bubbles v2, and lipgloss v2.
package app2

import (
	"strings"

	"github.com/kluzzebass/lazyhue/internal/ui2/components"
)

// showTestForm creates and displays a test form with all field types.
func (m *Model) showTestForm() {
	// Create test fields matching v1 test form
	fields := []components.FormField{
		{
			ID:    "toggle",
			Label: "Toggle",
			Type:  components.FormFieldToggle,
			Value: 1,
		},
		{
			ID:    "slider",
			Label: "Slider",
			Type:  components.FormFieldSlider,
			Value: 5,
			Min:   0,
			Max:   10,
		},
		{
			ID:    "brightness",
			Label: "Brightness",
			Type:  components.FormFieldBrightness,
			Value: 75,
			Min:   0,
			Max:   100,
		},
		{
			ID:    "colortemp",
			Label: "Color Temp",
			Type:  components.FormFieldColorTemp,
			Value: 326, // Mirek value (warm-ish)
			Min:   153, // Cool (6500K)
			Max:   500, // Warm (2000K)
		},
		{
			ID:     "color",
			Label:  "Color",
			Type:   components.FormFieldColor,
			ColorX: 0.5,
			ColorY: 0.3,
		},
		{
			ID:    "select",
			Label: "Select",
			Type:  components.FormFieldSelect,
			Value: 13, // M
			Options: []components.FormSelectOption{
				{Label: "Alpha", Value: 1},
				{Label: "Bravo", Value: 2},
				{Label: "Charlie", Value: 3},
				{Label: "Delta", Value: 4},
				{Label: "Echo", Value: 5},
				{Label: "Foxtrot", Value: 6},
				{Label: "Golf", Value: 7},
				{Label: "Hotel", Value: 8},
				{Label: "India", Value: 9},
				{Label: "Juliet", Value: 10},
				{Label: "Kilo", Value: 11},
				{Label: "Lima", Value: 12},
				{Label: "Mike", Value: 13},
				{Label: "November", Value: 14},
				{Label: "Oscar", Value: 15},
				{Label: "Papa", Value: 16},
				{Label: "Quebec", Value: 17},
				{Label: "Romeo", Value: 18},
				{Label: "Sierra", Value: 19},
				{Label: "Tango", Value: 20},
				{Label: "Uniform", Value: 21},
				{Label: "Victor", Value: 22},
				{Label: "Whiskey", Value: 23},
				{Label: "X-ray", Value: 24},
				{Label: "Yankee", Value: 25},
				{Label: "Zulu", Value: 26},
			},
		},
		{
			ID:        "text",
			Label:     "Text",
			Type:      components.FormFieldText,
			TextValue: "Hello World",
		},
		{
			ID:    "radio",
			Label: "Radio (horiz)",
			Type:  components.FormFieldRadio,
			Value: 2,
			Options: []components.FormSelectOption{
				{Label: "Small", Value: 1},
				{Label: "Medium", Value: 2},
				{Label: "Large", Value: 3},
			},
			Vertical: false,
		},
		{
			ID:    "radio_vert",
			Label: "Radio (vert)",
			Type:  components.FormFieldRadio,
			Value: 2,
			Options: []components.FormSelectOption{
				{Label: "Low", Value: 1},
				{Label: "Medium", Value: 2},
				{Label: "High", Value: 3},
				{Label: "Ultra", Value: 4},
			},
			Vertical: true,
		},
		{
			ID:         "hsl",
			Label:      "HSL Color",
			Type:       components.FormFieldHSL,
			Hue:        180,
			Saturation: 75,
			Lightness:  50,
		},
		{
			ID:    "rgb",
			Label: "RGB Color",
			Type:  components.FormFieldRGB,
			Red:   100,
			Green: 150,
			Blue:  200,
		},
	}

	// Create form component
	m.testForm = components.NewForm(&m.styles, m.zones)
	m.testForm.SetFields(fields)
	m.testForm.OnChange = func(field components.FormField) {
		// Re-render form when fields change
		var content strings.Builder
		content.WriteString(m.styles.Title.Render("Form Field Demo") + "\n\n")
		content.WriteString(m.testForm.View())
		m.detailViewport.SetContent(content.String())
	}

	// Render the form
	var content strings.Builder
	content.WriteString(m.styles.Title.Render("Form Field Demo") + "\n\n")
	content.WriteString(m.testForm.View())

	// Set the detail viewport content to show the form
	m.detailViewport.SetContent(content.String())
	m.focusedPane = PanelDetail
}
