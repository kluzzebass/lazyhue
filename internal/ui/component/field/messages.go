package field

// FieldChangedMsg is sent when a field value changes.
// The parent component handles this to make API calls.
type FieldChangedMsg struct {
	FieldID string
	Value   any // Type depends on field (bool, int, float64, string, etc.)
}

// ToggleValue is the value type for toggle fields.
type ToggleValue struct {
	On bool
}

// SliderValue is the value type for slider fields.
type SliderValue struct {
	Value int
}

// ColorValue is the value type for color fields.
type ColorValue struct {
	X, Y float64 // CIE xy color space
}

// HSLValue is the value type for HSL fields.
type HSLValue struct {
	Hue, Saturation, Lightness int
}

// RGBValue is the value type for RGB fields.
type RGBValue struct {
	Red, Green, Blue int
}

// SelectValue is the value type for select/dropdown fields.
type SelectValue struct {
	Index int
}

// TextValue is the value type for text fields.
type TextValue struct {
	Text string
}

// ButtonValue is the value type for button fields.
type ButtonValue struct {
	Pressed bool
}

// GradientPoint represents a single point in a gradient.
type GradientPoint struct {
	X, Y float64 // CIE xy color space
}

// GradientValue is the value type for gradient editor fields.
type GradientValue struct {
	Points []GradientPoint
}

// StartCaptureMsg is sent when a field wants to capture mouse events.
// This is used for drag operations on sliders and color wheels.
type StartCaptureMsg struct {
	FieldID string
}

// EndCaptureMsg is sent when a field releases mouse capture.
type EndCaptureMsg struct {
	FieldID string
}

// BlinkTickMsg is sent by the blink timer for color wheel indicators.
type BlinkTickMsg struct {
	FieldID string
}
