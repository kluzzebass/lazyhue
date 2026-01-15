# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

LazyHue is a terminal user interface (TUI) for managing Philips Hue bridges and lights. Built with Go and the Charm ecosystem (Bubble Tea, Lipgloss, Bubbles), it follows the lazygit/lazydocker UX philosophy with keyboard-driven navigation and mouse support.

## Build & Run Commands

```bash
# Build
just build

# Run
just run

# Build and run
just dev

# Development: watch for changes and auto-rebuild/restart (requires fswatch)
just watch
```

## Architecture

```
┌─────────────────────────────────────┐
│  UI Layer (ui/)                    │
│  - Panels (tree, bridges, etc.)     │
│  - Components (form, colorwheel)    │
│  - Layout system, themes, zones     │
└──────────────┬──────────────────────┘
               │ tea.Cmd / tea.Msg
┌──────────────▼──────────────────────┐
│  Application Layer (app/)          │
│  - Main tea.Model                   │
│  - Event handling & rendering       │
└──────────────┬──────────────────────┘
               │ Service calls
┌──────────────▼──────────────────────┐
│  Service Layer (hue/)               │
│  - Bridge Manager (multi-bridge)    │
│  - Bridge connection, SSE events    │
│  - Light actions, state management  │
└──────────────┬──────────────────────┘
               │
┌──────────────▼──────────────────────┐
│  API Client (hueclient/)            │
│  - Generated from Hue OpenAPI spec  │
└─────────────────────────────────────┘
```

**Key Directories:**
- `cmd/lazyhue/` - Entry point
- `internal/app/` - Bubble Tea application model
- `internal/ui/` - UI components and panels
- `internal/hue/` - Service layer (bridge management, state, actions)
- `internal/hueclient/` - Generated OpenAPI client (do not edit directly)
- `internal/config/` - Configuration and credential persistence
- `plans/` - Technical documentation

**Service Layer Independence:** The `hue/`, `hueclient/`, `config/`, and `debug/` packages have no Charm dependencies.

## Key Patterns

### Message-Driven Updates
All state changes flow through Bubble Tea messages:
```go
type bridgeEventMsg struct { bridgeID, resourceType, resourceID, eventType string }
type stateSyncedMsg struct { bridgeID string }
```

### Bubblezone for Mouse Handling
Uses bubblezone for click detection:
```go
zone.Mark("field-id", fieldView)  // Mark zones
zone.Get("field-id").InBounds(msg) // Detect clicks
```

### Layout System
Flex-box style layout in `ui/layout/layout.go`:
```go
layout.HSplit(
    layout.Child{Size: layout.Flex(0.4), Node: treePanel},
    layout.Child{Size: layout.Flex(0.6), Node: detailPanel},
)
```

## Hue API Workarounds

### Gradient PUT Requests

The generated `hueclient` types are incorrect for gradient PUT operations. The Hue API expects a different structure for PUT than what it returns in GET.

**Problem 1: Gradient points need a `color` wrapper**

GET returns:
```json
{ "gradient": { "points": [{ "xy": { "x": 0.5, "y": 0.5 } }] } }
```

PUT expects:
```json
{ "gradient": { "points": [{ "color": { "xy": { "x": 0.5, "y": 0.5 } } }] } }
```

**Problem 2: Gradient mode changes require points**

You cannot send just the mode - the API returns HTTP 400 `missing: ['points']`. You must include the current points when changing mode.

**Solution** (in `internal/hue/actions.go`):

Custom structs wrap the data correctly:
```go
type gradientPointPut struct {
    Color *hueclient.Color `json:"color,omitempty"`
}

type gradientPut struct {
    Points []gradientPointPut `json:"points,omitempty"`
}

type lightGradientPutBody struct {
    Gradient *gradientPut `json:"gradient,omitempty"`
}
```

Use `UpdateLightWithBody()` with raw JSON instead of the generated types:
```go
body := lightGradientPutBody{...}
jsonBody, _ := json.Marshal(body)
_, err := b.client.UpdateLightWithBody(ctx, lightID, "application/json", bytes.NewReader(jsonBody))
```

## Component Architecture (MANDATORY)

**CRITICAL: All UI elements MUST use the component architecture.** Do NOT fall back into old patterns of complex if/else/switch/case statements for event handling in app.go.

### The Component Interface

Every UI element implements `component.Component`:
```go
type Component interface {
    Layout(bounds Rect)
    Bounds() Rect
    Update(msg tea.Msg) (Component, tea.Cmd)
    RouteEvent(msg tea.Msg) (handled bool, cmd tea.Cmd)  // KEY METHOD
    View() string
    IsFocused() bool
    CanFocus() bool
    Focus()
    Blur()
    Children() []Component
    Parent() Component
    SetParent(parent Component)
}
```

### RouteEvent is the Key

Components handle their own events via `RouteEvent()`. The parent routes events down, children consume them:
```go
func (f *FormComponent) RouteEvent(msg tea.Msg) (bool, tea.Cmd) {
    // Route to focused child
    if handled, cmd := f.fields[f.cursor].RouteEvent(msg); handled {
        return true, cmd
    }
    // Handle form-level navigation only if child didn't consume
    // ...
}
```

### Field Components (internal/ui/component/field/)

**USE THESE for all form fields:**
- `ToggleComponent` - boolean on/off
- `SliderComponent`, `BrightnessSliderComponent`, `ColorTempSliderComponent` - value sliders
- `SelectComponent` - dropdown selection
- `RadioComponent` - radio button groups
- `TextComponent` - text input (wraps bubbles textinput)
- `ColorWheelComponent` - XY color picker
- `HSLComponent`, `RGBComponent` - 3-slider color pickers
- `HeaderComponent` - non-focusable section headers
- `FormComponent` - container that manages field navigation

### Message-Based Communication

Fields communicate via messages, NOT callbacks:
```go
// Field sends this when value changes
type FieldChangedMsg struct {
    FieldID string
    Value   any  // ToggleValue, SliderValue, ColorValue, etc.
}

// App.go handles it in Update():
case field.FieldChangedMsg:
    m.handleNewFieldChange(msg)
```

### PROHIBITED Patterns

**DO NOT:**
- Add switch/case blocks in app.go to handle specific field types
- Check `lightForm.Editing`, `lightForm.DropdownOpen`, `lightForm.MouseCaptureIdx`
- Route mouse events manually based on field type
- Use callbacks like `OnChange func(field FormField)`
- Add conditional logic like `if field.Type == FormFieldColor { ... }`

**INSTEAD:**
- Create a new field component type if needed
- Let components handle their own events via `RouteEvent()`
- Use `FieldChangedMsg` for value change notifications
- Use `StartCaptureMsg`/`EndCaptureMsg` for mouse capture

### Key Handling Priority

**Inner components have priority over global keys.** If a field component is focused or editing, it handles keys first. Route events from the inside out:

```go
// In app.go Update(), route to focused component FIRST
if handled, cmd := m.componentRoot.RouteEvent(msg); handled {
    return m, cmd
}
// Only then handle global keys (if any)
```

If you run into conflicts between component keys and global shortcuts, **the component wins**. We can figure out globals later if needed - don't add complexity to work around it.

### Building Forms

```go
// Create field components
fields := []component.Component{
    field.NewHeaderComponent("controls", "Controls", styles, zones),
    field.NewToggleComponent("on", "Power", true, styles, zones),
    field.NewBrightnessSliderComponent("brightness", "Brightness", 75, styles, zones),
    field.NewColorWheelComponent("color", "Color", 0.5, 0.5, styles, zones),
}

// Set on FormComponent
m.lightFormComponent.SetFields(fields)
```

## Dependencies

Framework:
- `github.com/charmbracelet/bubbletea/v2`
- `github.com/charmbracelet/bubbles/v2`
- `github.com/charmbracelet/lipgloss/v2`
- `github.com/lrstanley/bubblezone/v2`

Service layer:
- `github.com/grandcat/zeroconf` - mDNS bridge discovery
- `github.com/oapi-codegen/runtime` - Generated client runtime
