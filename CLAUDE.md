# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

LazyHue is a terminal user interface (TUI) for managing Philips Hue bridges and lights. Built with Go and the Charm ecosystem (Bubble Tea, Lipgloss, Bubbles), it follows the lazygit/lazydocker UX philosophy with keyboard-driven navigation and mouse support.

**Current State:** Active migration from Bubble Tea v1 to v2 on the `bubbletea-v2-migration` branch. The v2 code lives in `app2/` and `ui2/` directories.

## Build & Run Commands

```bash
# Build v2 (current development target)
go build -o lazyhue2 ./cmd/lazyhue2

# Build v1 (legacy)
go build -o lazyhue ./cmd/lazyhue

# Run v2
./lazyhue2

# Development: auto-restart on binary change (requires fswatch)
./watch.sh ./lazyhue2
```

## Architecture

```
┌─────────────────────────────────────┐
│  UI Layer (ui2/)                    │
│  - Panels (tree, bridges, etc.)     │
│  - Components (form, colorwheel)    │
│  - Layout system, themes, zones     │
└──────────────┬──────────────────────┘
               │ tea.Cmd / tea.Msg
┌──────────────▼──────────────────────┐
│  Application Layer (app2/)          │
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
- `cmd/lazyhue2/` - v2 entry point
- `internal/app2/` - v2 Bubble Tea application model
- `internal/ui2/` - v2 UI components and panels
- `internal/hue/` - Service layer (bridge management, state, actions)
- `internal/hueclient/` - Generated OpenAPI client (do not edit directly)
- `internal/config/` - Configuration and credential persistence
- `plans/` - Technical documentation and migration plans

**Service Layer Independence:** The `hue/`, `hueclient/`, `config/`, and `debug/` packages have no Charm dependencies and require no changes during UI migrations.

## Key Patterns

### Message-Driven Updates
All state changes flow through Bubble Tea messages:
```go
type bridgeEventMsg struct { bridgeID, resourceType, resourceID, eventType string }
type stateSyncedMsg struct { bridgeID string }
```

### Bubblezone for Mouse Handling
v2 uses bubblezone for click detection instead of manual coordinate math:
```go
zone.Mark("field-id", fieldView)  // Mark zones
zone.Get("field-id").InBounds(msg) // Detect clicks
```

### Layout System
Flex-box style layout in `ui2/layout/layout.go`:
```go
layout.HSplit(
    layout.Child{Size: layout.Flex(0.4), Node: treePanel},
    layout.Child{Size: layout.Flex(0.6), Node: detailPanel},
)
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

### Field Components (internal/ui2/component/field/)

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

### Legacy Code to DELETE

The following are OLD patterns that should be replaced and eventually removed:
- `internal/ui2/components/form.go` - Old monolithic Form
- `internal/ui2/components/formfield.go` - Old FormField struct
- `internal/ui2/components/form_handlers.go` - Old event handlers
- `m.lightForm *components.Form` in app.go - Replace with `m.lightFormComponent`
- All the conditional Form handling in app.go Update() switch/case

## v2 Migration Context

The migration follows the plan in `plans/charm_v2_migration.md`:
- Use native bubbles components (list, viewport, help, textinput) over homebrew
- Use bubblezone for mouse tracking instead of manual coordinate calculations
- Keep custom components minimal (color wheel, sliders, layout)
- **Use component architecture for ALL interactive UI elements**

Both v1 (`ui/`, `app/`) and v2 (`ui2/`, `app2/`) coexist during migration for comparison.

## Dependencies

Framework (v2 - active development):
- `github.com/charmbracelet/bubbletea/v2`
- `github.com/charmbracelet/bubbles/v2`
- `github.com/charmbracelet/lipgloss/v2`
- `github.com/lrstanley/bubblezone/v2`

Service layer:
- `github.com/grandcat/zeroconf` - mDNS bridge discovery
- `github.com/oapi-codegen/runtime` - Generated client runtime
