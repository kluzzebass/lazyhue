## Unified Component System Plan

### Goals

- Single system for layout, routing, focus, and capture.
- Deterministic event flow with no leakage or special-case routing.
- Composable from top-level panels down to leaf controls (toggle/button).
- Consistent drag, modal, and editing behavior across all components.

### Core Concepts

#### Component Contract

- `Layout(bounds Rect)`
- `View() string`
- `RouteEvent(msg tea.Msg) (handled bool, cmd tea.Cmd)`
- Focus: `Focus()`, `Blur()`, `IsFocused()`, `CanFocus()`
- State: `IsActive()` (overlays/modals), `IsCapturing()` (drag/capture)
- Optional: `HitTest(x, y int) bool` (default uses bounds)

#### Event Flow (Single Pipeline)

1. Capture stack (drag capture, modal capture)
2. Focused component subtree
3. Parent navigation (container-level keys)
4. Global hotkeys (only if unhandled)

No routing through side channels in `app.Update` except global hotkeys.

### Routing Architecture

#### Root Component Tree

- One tree for all UI:
  - Tree panel component
  - Detail panel component (viewport + overlay stack)
  - Log panel component

#### Capture Manager

- Central capture manager for drag and modal capture.
- Replace `StartCaptureMsg`/`EndCaptureMsg` with direct capture calls:
  - `CaptureManager.Push(node)`
  - `CaptureManager.Pop(node)`
  - `CaptureManager.Active()`

#### Windows and Z-Stack

- Define a `WindowStack` (or reuse `StackedContainer`) for z-axis ordering.
- Topmost active window receives input first and can capture all events.
- Hit-testing respects z-order: topmost window with `HitTest` wins.
- Windows can be modal or non-modal; modals block lower layers by default.
- Windows participate in the same routing pipeline via capture and focus.

#### Replace Bubblezone

- Remove `bubblezone` from all components; no zone tags in rendering.
- Introduce a built-in hit-test registry:
  - `RegisterRegion(id, rect, z, owner)`
  - `HitTest(x, y) -> topmost region`
- Regions are derived from component layout bounds (and optional sub-regions).
- Z-order comes from window stack + registration order.
- Mouse routing uses hit-test results instead of zone IDs.
- API sketch:
  - `type Region struct { ID string; Rect Rect; Z int; Owner Component }`
  - `type HitTestRegistry interface { Register(Region); Clear(owner Component) }`
  - `HitTest(x, y int) (Region, bool)`
- Migration steps:
  - Phase A: register regions in `Layout()` for interactive controls.
  - Phase B: replace `Zones.Mark` with no-op wrappers or remove markers.
  - Phase C: route all mouse events via `HitTestRegistry`.
  - Phase D: delete bubblezone dependency and cleanup imports.
- Sub-region handling:
  - Components can register multiple regions (e.g., slider bar + knob + label).
  - Regions include a `Role` string to disambiguate behavior:
    - Example roles: `bar`, `knob`, `label`, `track`, `swatch`, `option`.
  - Hit tests return the topmost region; components route based on role.
  - For overlapping regions in the same component, higher `Z` wins.

### Layout System

#### Base Layout Component

- Responsible for:
  - Bounds calculation
  - Hit-testing mouse events
  - Routing keys to focused child

#### Responsive Layout

- Layout components can switch between horizontal/vertical modes based on
  bounds.
- Support breakpoints (e.g., min width) for side-by-side vs stacked rendering.
- Re-layout is triggered on terminal resize (`tea.WindowSizeMsg`).
- Components should be able to declare preferred layout modes and constraints.

#### Grid as Layout

- Grid becomes a standard layout component.
- No special routing for mouse wheel or drag.
- Uses the same routing rules as other containers.

### Focus and Editing

#### Focus Rules

- Leaf components only respond to keys when focused.
- Containers handle navigation keys when children do not.

#### Editing Rules

- Editing affects local handling/visuals only.
- Editing does not bypass routing or global pipeline.

### Mouse and Drag

- Mouse events flow: capture -> hit-test -> focused.
- Dragging requires capture ownership (no implicit drag).

### Migration Plan (Incremental)

#### Phase 1: Infrastructure

- Add `CaptureManager` to component system.
- Rewire root event handling to use the unified pipeline.
- Keep `StartCaptureMsg` temporarily but bridge to manager.

#### Phase 2: Containers and Grid

- Refactor `Container.RouteEvent` to avoid "try all children".
- Refactor `Grid` to follow layout routing rules.

#### Phase 3: Field Components

- Enforce `IsFocused()` for key handling in fields.
- Replace capture messages with capture manager APIs.

#### Phase 4: Panels and Modals

- Modals become components using `IsActive()` and capture.
- Remove panel-specific routing from `app.Update`.

### Testing Strategy

- Routing tests: one input -> one handler.
- Capture tests: drag start/drag/drag end in grid and panels.
- Focus tests: keys only affect focused component.
- No-leak tests: test page vs. detail panel.
- Tracing: enable event/message logging per component and routing stage.

### Tracing and Diagnostics

- Provide a structured, toggleable trace channel for routing decisions.
- Log at each pipeline stage: capture, focused subtree, parent navigation,
  global.
- Include component IDs, event types, and hit-test results in trace output.
- Support scoped tracing by component ID to reduce noise.
- Allow runtime toggles via a debug key or config flag.
- Provide a ring buffer of recent events for post-mortem inspection.

### Proposed Files

- `internal/ui/component/capture_manager.go` (new)
- `internal/ui/component/layout/base_layout.go` (new)
- `internal/ui/component/layout/grid.go` (refactor)
- `internal/ui/component/container.go` (refactor)
- Field components: enforce focused handling and capture
- `internal/app/app.go` (remove special routing paths)
