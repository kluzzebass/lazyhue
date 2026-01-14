# Charm v2 Migration Plan

## Philosophy

**DO NOT simply reimplement the existing UI with v2 libraries.**

Instead, we will:
1. **Leverage native components** from bubbles wherever possible
2. **Adopt bubblezone** for mouse tracking instead of manual coordinate math
3. **Reduce homebrew code** - mixing bubbletea with too much custom code causes problems
4. **Use proven patterns** from the Charm ecosystem

The goal is a cleaner, more maintainable codebase that leans heavily on battle-tested libraries.

## Overview

| Library | Current Version | Target Version | Import Path |
|---------|-----------------|----------------|-------------|
| bubbletea | v1.3.10 | v2.0.0-rc.2 | `charm.land/bubbletea/v2` |
| bubbles | v0.21.0 | v2.0.0-rc.1 | `charm.land/bubbles/v2` |
| lipgloss | v1.1.0 | v2.0.0-beta.3 | `charm.land/lipgloss/v2` |
| bubblezone | - (not used) | v2.0.0-alpha.3 | `github.com/lrstanley/bubblezone` |

## Native Components to Adopt

### Bubblezone for Mouse Tracking

[bubblezone v2.0.0-alpha.3](https://github.com/lrstanley/bubblezone/releases/tag/v2.0.0-alpha.3) provides:
- Automatic mouse zone tracking via string markers
- No manual coordinate calculation needed
- Click detection by zone ID
- Works with any lipgloss-styled content

**Replaces our homebrew:**
- All manual mouse coordinate calculations in `popup.go`, `form.go`, `detailspanel.go`
- The complex `handleMouse` functions with zone-based detection
- Field position calculations (`getFieldRowY`, `rowForField`, etc.)

**How it works:**
```go
// Mark a zone
zone.Mark("brightness-slider", sliderView)

// In Update, check for clicks
if zone.Get("brightness-slider").InBounds(msg) {
    // Handle brightness click
}
```

### Bubbles Components to Adopt

| Bubbles Component | Replaces Our | Benefits |
|-------------------|--------------|----------|
| `list.Model` | `treepanel.go`, `listpanel.go` | Built-in filtering, pagination, keyboard nav |
| `table.Model` | Custom entity lists | Columns, sorting, selection |
| `viewport.Model` | Already used | ✓ Keep using, now has horizontal scroll |
| `textinput.Model` | Already used | ✓ Keep using, improved in v2 |
| `textarea.Model` | N/A | Could use for multi-line descriptions |
| `help.Model` | `help.go` | Native help with key binding display |
| `progress.Model` | Custom sliders? | Multiple color stops, animations |
| `spinner.Model` | Pairing animation | Built-in spinner styles |
| `paginator.Model` | Large lists | Page navigation |
| `filepicker.Model` | N/A | Future file operations |
| `cursor.Model` | Blinking indicators | Proper cursor blink management |
| `key.Binding` | `binding.go` | Structured key bindings with help text |

### Components We Should Build With Native Bubbles

1. **Tree Panel** → Use `list.Model` with custom item delegate for tree indentation
2. **Bridge List** → Use `list.Model` 
3. **Details Panel** → Use `viewport.Model` + bubblezone for clickable fields
4. **Form/Popup** → Use bubblezone for field clicks + native textinput
5. **Help Overlay** → Use native `help.Model`
6. **Activity Log** → Use `viewport.Model` with virtual scrolling

### Components That Stay Custom (Minimal)

1. **Color Wheel** - Unique rendering, but use bubblezone for click detection
2. **Sliders** - Custom rendering, but use bubblezone for click detection  
3. **Layout System** - Keep `layout/layout.go` (clean flex-box implementation)

## Architecture Analysis

### Current Structure

```
internal/
├── app/          # Application layer (tea.Model, Update, View)
├── config/       # Configuration (independent of UI)
├── debug/        # Debug logging (independent of UI)
├── hue/          # Service layer (independent of UI) ✓
├── hueclient/    # API client (independent of UI) ✓
└── ui/           # UI components (WILL BE MIGRATED)
    ├── binding.go
    ├── border.go
    ├── color.go
    ├── theme.go
    ├── layout/
    └── panels/
```

### Service Layer Independence

The following packages have NO dependency on Charm libraries and require NO changes:
- `internal/config/` - Configuration management
- `internal/debug/` - Debug logging
- `internal/hue/` - Bridge management, state, actions, SSE events
- `internal/hueclient/` - Generated OpenAPI client

### UI Layer (Requires Migration)

The following packages depend on Charm libraries:
- `internal/ui/` - Styles, colors, bindings, layout
- `internal/ui/panels/` - All panel components
- `internal/app/` - Main application model, handlers, render

### Problems With Current Homebrew Implementation

Issues we've debugged extensively that native components would solve:

| Problem | Root Cause | Native Solution |
|---------|------------|-----------------|
| Mouse clicks off by 1-2 rows | Manual coordinate math | bubblezone zone detection |
| Color wheel indicator jumps | Position recalculation errors | bubblezone + stable state |
| Scroll wheel not working | Wrong event type checks | Native viewport handles this |
| Focus state reset on SSE | State not preserved properly | Proper model composition |
| Form field position bugs | `rowForField` complexity | bubblezone per-field zones |
| Blink timer management | Manual timer coordination | Native `cursor.Model` |
| Dropdown position bugs | Manual Y offset calculation | bubblezone for dropdown |

**Key insight:** Most of our bugs come from manual coordinate calculations. Bubblezone eliminates this entire class of bugs.

## Migration Strategy

### Phase 1: Parallel UI Development

Create a new `internal/ui/` package alongside the existing `internal/ui/`:

```
internal/
├── ui/           # OLD - v1 implementation (keep for reference)
└── ui/          # NEW - v2 implementation
    ├── binding.go
    ├── border.go
    ├── color.go
    ├── theme.go
    ├── layout/
    └── panels/
```

This allows:
1. Side-by-side comparison during development
2. Ability to run both versions for testing
3. Gradual migration without breaking the working app
4. Reference for complex logic (color wheel, forms, etc.)

### Phase 2: Application Layer Migration

Create `internal/app/` alongside existing `internal/app/`:

```
internal/
├── app/          # OLD - v1 app layer
└── app/         # NEW - v2 app layer
    ├── app.go
    ├── handlers.go
    ├── messages.go
    └── ...
```

### Phase 3: Integration & Testing

1. Create a new `cmd/lazyhue/main.go` entry point
2. Run both versions in parallel for comparison
3. Verify all features work correctly

### Phase 4: Cleanup

1. Remove old `internal/ui/` and `internal/app/`
2. Rename `ui` → `ui` and `app` → `app`
3. Update `cmd/lazyhue/main.go`
4. Remove v1 dependencies from `go.mod`

## Key API Changes

### Bubbletea v2

Based on [v2.0.0-rc.2 release notes](https://github.com/charmbracelet/bubbletea/releases/tag/v2.0.0-rc.2):

#### Breaking Changes

1. **Import Path**: `charm.land/bubbletea/v2`

2. **Model Interface**: The `Update` method signature may change
   ```go
   // v1
   func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd)
   
   // v2 - verify actual signature
   func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd)
   ```

3. **Mouse Events**: New `tea.MouseMsg` structure
   - `msg.Action` replaces `msg.Type` in some cases
   - New `tea.MouseActionPress`, `tea.MouseActionRelease`, `tea.MouseActionMotion`

4. **Keyboard Enhancements**: New keyboard enhancement flags API
   - Better key event handling
   - Improved modifier key support

5. **Renderer Changes**: 
   - New `View` system with `NewView()` and `SetContent()`
   - Mode 2026 support
   - Better cursor visibility management

6. **PasteMsg**: Now uses `msg.Content` instead of string conversion

#### New Features

- View-based rendering with layers
- Better terminal capability detection
- Improved performance with skip-flush optimization

### Lipgloss v2

Based on [v2.0.0-beta.3](https://github.com/charmbracelet/lipgloss/releases/tag/v2.0.0-beta.3):

1. **Import Path**: `charm.land/lipgloss/v2`

2. **New Features**:
   - `style.PaddingChar(rune)` - Customize padding character
   - `style.MarginChar(rune)` - Customize margin character
   - Reverted to regular space (instead of NBSP `\u00a0`) for padding

3. **Style Changes**: Verify any deprecated methods
   - `NewStyle()` may have different signature
   - Color handling may differ
   - Padding/margin now uses regular spaces (copy-paste friendly)

### Bubbles v2

Based on [v2.0.0-rc.1 release notes](https://github.com/charmbracelet/bubbles/releases/tag/v2.0.0-rc.1):

1. **Import Path**: `charm.land/bubbles/v2`

2. **Progress**: Improved blend algorithm, multiple color stops

3. **Textarea**: 
   - New `PageUp`, `PageDown` support
   - `ScrollYOffset`, `ScrollPosition` methods
   - `MoveToBeginning`, `MoveToEnd` exposed
   - Uses pointer receiver for Model methods

4. **Viewport**: 
   - Horizontal mouse wheel scrolling
   - Soft-wrap improvements
   - Performance improvements

5. **Textinput**: Improved placeholder handling

6. **Help**: Uses setter/getter for help width

## Component Migration Checklist

### Core Infrastructure (`internal/ui/`)

- [ ] `theme.go` - Styles using lipgloss v2
- [ ] `color.go` - Color utilities (mostly pure Go, keep as-is)
- [ ] `keys.go` - Key bindings using `key.Binding` from bubbles
- [ ] `layout/layout.go` - Keep existing flex layout system
- [ ] `zones.go` - Bubblezone integration and zone ID constants

### Panel Components - Using Native Bubbles

| Old File | New Approach | Native Components Used |
|----------|--------------|----------------------|
| `treepanel.go` | Rewrite | `list.Model` with tree delegate |
| `listpanel.go` | Remove | Merged into tree using `list.Model` |
| `bridges.go` | Rewrite | `list.Model` for bridge selection |
| `help.go` | Rewrite | Native `help.Model` |
| `logpanel.go` | Rewrite | `viewport.Model` with `list.Model` |
| `popup.go` | Rewrite | bubblezone + native inputs |
| `form.go` | Rewrite | bubblezone + `textinput.Model` |
| `detailspanel.go` | Rewrite | `viewport.Model` + bubblezone |
| `pairing.go` | Rewrite | `spinner.Model` + `progress.Model` |
| `colorwheel.go` | Keep/adapt | Custom render + bubblezone clicks |
| `header.go` | Simplify | Pure lipgloss render |
| `status.go` | Simplify | Pure lipgloss render |
| `scroll.go` | Remove | Native viewport handles scrolling |

### Details View Builders (`internal/ui/panels/details/`)

These are mostly pure rendering, keep with minimal changes:
- [ ] `view.go` - View builder interface
- [ ] `fields.go` - Field rendering
- [ ] `list.go` - List rendering  
- [ ] `text.go` - Text rendering
- [ ] `details_*.go` - Entity-specific views

### Application Layer (`internal/app/`)

- [ ] `messages.go` - Message types
- [ ] `app.go` - Main model, Init, Update, View
- [ ] `handlers.go` - Event handlers
- [ ] `bindings.go` - Key binding setup
- [ ] `dispatch.go` - Action dispatch
- [ ] `commands.go` - tea.Cmd factories
- [ ] `panels_state.go` - Panel state management
- [ ] `render.go` - View rendering
- [ ] `actions.go` - Action definitions

## Testing Strategy

### Unit Tests

1. Run existing tests against v1 (baseline)
2. Create equivalent tests for v2 components
3. Compare behavior

### Integration Tests

1. Build both `lazyhue` (v1) and `lazyhue` (v2) binaries
2. Run side-by-side with same bridge
3. Verify:
   - All panels render correctly
   - Mouse input works
   - Keyboard navigation works
   - Color wheel works
   - Forms work
   - SSE events update correctly
   - All CRUD operations work

### Manual Testing Checklist

- [ ] Bridge discovery and pairing
- [ ] Light on/off toggle
- [ ] Brightness adjustment (slider)
- [ ] Color selection (color wheel)
- [ ] Color temperature adjustment
- [ ] Effects selection
- [ ] Room/zone creation
- [ ] Room/zone editing
- [ ] Room/zone deletion
- [ ] Scene activation
- [ ] Activity log scrolling
- [ ] Help overlay
- [ ] Popup forms
- [ ] Tree navigation
- [ ] Panel focus switching

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| RC/beta instability | High | Keep v1 as fallback, report bugs upstream |
| API changes before final release | Medium | Pin to specific versions, monitor changelogs |
| Mouse handling differences | High | Extensive testing, reference v1 implementation |
| Color rendering differences | Medium | Visual comparison testing |
| Performance regressions | Low | Benchmark critical paths |

## Timeline Estimate

| Phase | Estimated Duration | Notes |
|-------|-------------------|-------|
| Phase 1: UI2 scaffolding | 1-2 days | Basic structure, theme, colors |
| Phase 1: Panel migration | 3-5 days | Port all panels |
| Phase 2: App layer migration | 2-3 days | Port app, handlers, render |
| Phase 3: Integration testing | 1-2 days | Fix bugs, verify features |
| Phase 4: Cleanup | 0.5 day | Remove old code, rename |
| **Total** | **7-12 days** | |

## Dependencies to Add

```bash
# Add v2 dependencies (these can coexist with v1 during migration)
go get charm.land/bubbletea/v2@v2.0.0-rc.2
go get charm.land/bubbles/v2@v2.0.0-rc.1
go get charm.land/lipgloss/v2@v2.0.0-beta.3

# Bubblezone for mouse tracking
go get github.com/lrstanley/bubblezone@v2.0.0-alpha.3
```

## Open Questions

1. **Bubblezone + List Integration**: How well does bubblezone work with nested `list.Model` components? Any special handling needed?

2. **View System**: The new View-based rendering - does it change how we compose panel views?

3. **List.Model for Trees**: Best pattern for using `list.Model` with hierarchical data (expand/collapse)?

4. **Help.Model Customization**: Can we style the native help to match our theme?

5. **Zone IDs in Dynamic Content**: Best practices for zone IDs when content changes (e.g., SSE updates)?

6. **Keyboard Enhancements**: Should we enable the new keyboard enhancement flags for better key detection?

## Pre-Implementation Research

Before writing code, we should:

### 1. Study Bubblezone Examples
- [ ] Read bubblezone README and examples
- [ ] Understand zone marking syntax
- [ ] Understand zone event detection API
- [ ] Test basic bubblezone + lipgloss integration

### 2. Study Bubbles v2 List Component
- [ ] Read list.Model documentation
- [ ] Understand ItemDelegate interface for custom rendering
- [ ] Study filtering and pagination APIs
- [ ] Look at examples of tree-like structures with list

### 3. Study Bubbles v2 Help Component  
- [ ] Read help.Model documentation
- [ ] Understand key.Binding integration
- [ ] Test styling options

### 4. Build Minimal Prototypes
- [ ] Prototype: bubblezone with clickable regions
- [ ] Prototype: list.Model with custom tree delegate
- [ ] Prototype: help.Model with our keybindings
- [ ] Prototype: form with bubblezone field detection

### 5. Review Charm Example Apps
Look at how official Charm apps structure their code:
- [ ] `charm.land/gum` - CLI tool patterns
- [ ] `charm.land/soft-serve` - TUI app patterns
- [ ] `charm.land/mods` - Model composition patterns

## Notes

- The v2 releases use the new `charm.land/*` domain instead of `github.com/charmbracelet/*`
- Both import paths can coexist in `go.mod` during migration
- The service layer (`internal/hue/`, `internal/hueclient/`) requires NO changes
- Configuration layer (`internal/config/`) requires NO changes

## References

### Release Notes
- [Bubbletea v2.0.0-rc.2 Release](https://github.com/charmbracelet/bubbletea/releases/tag/v2.0.0-rc.2)
- [Bubbles v2.0.0-rc.1 Release](https://github.com/charmbracelet/bubbles/releases/tag/v2.0.0-rc.1)
- [Lipgloss v2.0.0-beta.3 Release](https://github.com/charmbracelet/lipgloss/releases/tag/v2.0.0-beta.3)
- [Bubblezone v2.0.0-alpha.3 Release](https://github.com/lrstanley/bubblezone/releases/tag/v2.0.0-alpha.3)

### Documentation & Examples
- [Bubblezone README](https://github.com/lrstanley/bubblezone) - Mouse zone tracking
- [Bubbles Examples](https://github.com/charmbracelet/bubbles/tree/master/examples)
- [Bubbletea Examples](https://github.com/charmbracelet/bubbletea/tree/master/examples)

### Reference Applications
- [Soft Serve](https://github.com/charmbracelet/soft-serve) - Git server TUI
- [Glow](https://github.com/charmbracelet/glow) - Markdown reader
- [Mods](https://github.com/charmbracelet/mods) - AI CLI tool
