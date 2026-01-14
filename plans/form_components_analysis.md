# Form Components Analysis

## Analysis: Bubbles v2 Components vs Custom Components

### Can Use Bubbles Components:
- **FormFieldText** → `bubbles/v2/textinput` ✓
  - Direct replacement available
  - Already used in v1 form implementation

### Need Custom Components:
- **FormFieldToggle** - Boolean on/off toggle
- **FormFieldSlider** - Generic integer slider with min/max
- **FormFieldBrightness** - Brightness slider (0-100) with visual bar
- **FormFieldColorTemp** - Color temperature slider with colored gradient
- **FormFieldColor** - Color wheel (XY color space) - ColorWheel component exists in v1
- **FormFieldSelect** - Dropdown selection (could use list.Model but custom is simpler)
- **FormFieldRadio** - Radio button group (horizontal/vertical)
- **FormFieldHSL** - HSL color picker (3 sliders: Hue, Saturation, Lightness)
- **FormFieldRGB** - RGB color picker (3 sliders: Red, Green, Blue)

## Implementation Plan

1. ✅ Create form field types and structures (`formfield.go`)
2. Create reusable rendering functions for form components
3. Implement test form handler in `internal/app/`
4. Wire up key binding for test form

## Component Structure

Components to create:
- ✅ `formfield.go` - Form field types and structures
- Render functions for each component type (can be in a single file or separate files)
- Color wheel component (lift from v1 `colorwheel.go`)
- Form test handler in `internal/app/`

## Scale

The v1 form system is 4000+ lines. This is a large task that should be built incrementally.
We'll start with types and rendering functions, then add interaction logic.
