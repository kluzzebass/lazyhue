// Package components provides reusable UI components for the v2 UI.
package components

// FormFieldType defines the type of form field.
type FormFieldType int

const (
	FormFieldToggle FormFieldType = iota // Boolean toggle (on/off)
	FormFieldSlider                      // Integer slider with min/max
	FormFieldText                        // Text input (uses bubbles textinput)
	FormFieldBrightness                  // Brightness slider (0-100) with visual bar
	FormFieldColorTemp                   // Color temperature slider (warm to cool)
	FormFieldColor                       // Color picker (XY color space)
	FormFieldSelect                      // Dropdown selection from options
	FormFieldRadio                       // Radio button group (inline options)
	FormFieldHSL                         // HSL color picker (3 sliders: Hue, Saturation, Lightness)
	FormFieldRGB                         // RGB color picker (3 sliders: Red, Green, Blue)
	FormFieldHeader                      // Section header (non-interactive, Label only)
)

// FormSelectOption represents an option for FormFieldSelect or FormFieldRadio.
type FormSelectOption struct {
	Label string // Display label
	Value int    // Value when selected
}

// FormField represents a single field in a form.
type FormField struct {
	ID    string        // Unique identifier
	Label string        // Display label
	Type  FormFieldType // Field type

	// Value fields (usage depends on Type)
	Value     int    // Current value (0/1 for toggle, actual value for slider/brightness/temp/select/radio)
	Min       int    // Minimum value (for sliders)
	Max       int    // Maximum value (for sliders)
	TextValue string // Current text value (for text fields)

	// Color fields (for FormFieldColor)
	ColorX float64 // CIE x coordinate (0.0-1.0)
	ColorY float64 // CIE y coordinate (0.0-1.0)

	// Select/Radio field options
	Options  []FormSelectOption // Available options for select/radio fields
	Vertical bool               // For radio buttons: true = vertical layout, false = horizontal

	// HSL fields (for FormFieldHSL)
	Hue        int // 0-360 degrees
	Saturation int // 0-100 percent
	Lightness  int // 0-100 percent

	// RGB fields (for FormFieldRGB)
	Red   int // 0-255
	Green int // 0-255
	Blue  int // 0-255

	// Toggle labels (for FormFieldToggle) - if empty, defaults to "On"/"Off"
	ToggleOnLabel  string
	ToggleOffLabel string

	// Read-only flag - if true, field cannot be edited (shows value only)
	ReadOnly bool

	// Link support - allows fields to navigate to related entities
	IsLink         bool   // If true, field is clickable and navigates to LinkEntityID
	LinkEntityType string // Type of entity to navigate to ("light", "device", "scene", etc.)
	LinkEntityID   string // ID of entity to navigate to
}
