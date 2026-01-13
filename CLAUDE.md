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

## v2 Migration Context

The migration follows the plan in `plans/charm_v2_migration.md`:
- Use native bubbles components (list, viewport, help, textinput) over homebrew
- Use bubblezone for mouse tracking instead of manual coordinate calculations
- Keep custom components minimal (color wheel, sliders, layout)

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
