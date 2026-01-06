---
name: lazyhue Technical Plan
overview: A comprehensive technical plan for building lazyhue, a terminal-based Philips Hue management tool using Go, Bubble Tea, and openhue-go, following the lazygit/lazydocker UX philosophy.
todos:
  - id: bridge-discovery
    content: Implement bridge discovery and authentication flow
    status: completed
  - id: credential-storage
    content: Build secure credential storage layer
    status: completed
  - id: entity-models
    content: Define domain models and relationship mapping from openhue-go types
    status: completed
  - id: bridge-manager
    content: Create multi-bridge manager with connection state tracking
    status: completed
  - id: state-poller
    content: Implement background polling with optimistic updates
    status: completed
  - id: tui-layout
    content: Build panel layout with entity list and detail view
    status: completed
  - id: keybindings
    content: Implement keybinding system with context-aware actions
    status: completed
  - id: light-controls
    content: Add light/group toggle, brightness, and color controls
    status: in-progress
  - id: scene-activation
    content: Implement scene browsing and activation
    status: in-progress
  - id: help-overlay
    content: Build help overlay with full keybinding reference
    status: pending
  - id: color-picker
    content: Implement color picker for color-capable lights
    status: pending
  - id: color-temp-picker
    content: Implement color temperature picker for CT-capable lights
    status: pending
  - id: visual-mode
    content: Implement visual/multi-select mode for bulk operations
    status: pending
  - id: zones-support
    content: Implement zones listing and control (currently placeholder)
    status: pending
  - id: entertainment-support
    content: Implement entertainment areas listing and control (currently placeholder)
    status: pending
---

# lazyhue Technical Design Plan

## High-Level Architecture

```mermaid
graph TB
    subgraph ui [UI Layer]
        TUI[Bubble Tea TUI]
        Panels[Panel Components]
        Keys[Keybinding Handler]
    end
    
    subgraph domain [Domain Layer]
        State[Application State]
        Entities[Entity Models]
        Actions[Action Dispatcher]
    end
    
    subgraph hue [Hue Layer]
        BridgeMgr[Bridge Manager]
        Discovery[Discovery Service]
        Auth[Authenticator]
        Poller[State Poller]
    end
    
    subgraph persist [Persistence Layer]
        Config[Config Manager]
        Credentials[Credential Store]
    end
    
    TUI --> State
    Panels --> State
    Keys --> Actions
    Actions --> BridgeMgr
    BridgeMgr --> Discovery
    BridgeMgr --> Auth
    BridgeMgr --> Poller
    BridgeMgr --> Entities
    Config --> BridgeMgr
    Credentials --> Auth
```

---

## 1. Package Structure

```
lazyhue/
├── cmd/
│   └── lazyhue/
│       └── main.go              # Entry point, program setup
├── internal/
│   ├── app/
│   │   ├── app.go               # Root Bubble Tea model
│   │   └── messages.go          # Custom tea.Msg types
│   ├── ui/
│   │   ├── theme.go             # Colors, styles (lipgloss)
│   │   ├── keys.go              # Keybinding definitions
│   │   └── panels/
│   │       ├── bridges.go       # Bridge list panel (top-left)
│   │       ├── entities.go      # Entity list panel (lights/rooms/zones)
│   │       ├── details.go       # Entity details panel
│   │       ├── header.go        # Header bar component
│   │       └── status.go        # Status bar component
│   ├── hue/
│   │   ├── bridge.go            # Bridge connection wrapper
│   │   ├── state.go             # BridgeState with openhue-go types
│   │   ├── actions.go           # Light/scene control actions
│   │   ├── manager.go           # Multi-bridge coordinator
│   │   ├── discovery.go         # Discovery orchestration
│   │   └── auth.go              # Authentication flow handler
│   └── config/
│       ├── config.go            # Configuration loading/saving
│       └── credentials.go       # Secure credential storage
├── go.mod
└── go.sum
```

**Design Rationale:**

- `internal/` prevents external imports, keeping the API surface clean
- **No domain package** - we use openhue-go types (`LightGet`, `RoomGet`, etc.) directly
- This enables partial queries (sync only lights) without rebuilding a relationship graph
- Relationships are resolved on-demand via `BridgeState` helper methods
- `panels/` as a subdirectory allows each panel to be self-contained

---

## 2. Bridge Discovery and Connection

### Discovery Flow

```mermaid
sequenceDiagram
    participant App
    participant Discovery
    participant mDNS
    participant MeetHue
    participant Auth
    participant Credentials

    App->>Discovery: DiscoverBridges()
    Discovery->>mDNS: Query _hue._tcp.local
    alt mDNS Success
        mDNS-->>Discovery: BridgeInfo[]
    else mDNS Timeout
        Discovery->>MeetHue: GET discovery.meethue.com
        MeetHue-->>Discovery: BridgeInfo[]
    end
    Discovery-->>App: []BridgeInfo
    
    loop For each bridge
        App->>Credentials: LoadApiKey(bridgeId)
        alt Key exists
            Credentials-->>App: apiKey
            App->>App: Connect(bridge, apiKey)
        else No key
            App->>Auth: StartPairing(bridge)
            Auth-->>App: PairingRequired msg
        end
    end
```

### Multi-Bridge Architecture

```go
// Pseudocode: Bridge Manager
type BridgeManager struct {
    bridges     map[string]*BridgeConnection  // keyed by bridge ID
    activeBridge string
    poller      *Poller
}

type BridgeConnection struct {
    info        BridgeInfo
    home        *openhue.Home
    state       *BridgeState      // cached entities
    lastSync    time.Time
    status      ConnectionStatus  // connected, disconnected, pairing
}
```

**Key Decisions:**

- Use openhue-go's `BridgeDiscovery` with 2-second mDNS timeout, fallback to meethue.com
- Store API keys in `~/.config/lazyhue/credentials.json` with 0600 permissions
- Connection status tracked per-bridge to handle partial connectivity
- Pairing UI shows inline in a modal, polling for link button press

### Credential Storage

```go
// ~/.config/lazyhue/credentials.json
type CredentialStore struct {
    Bridges map[string]BridgeCredential `json:"bridges"`
}

type BridgeCredential struct {
    BridgeID   string `json:"bridge_id"`
    BridgeName string `json:"bridge_name"`
    ApiKey     string `json:"api_key"`
    LastUsed   string `json:"last_used"`
}
```

**Security Note:** API keys are stored in plaintext. Future enhancement could integrate with system keychain via `go-keyring`. For Stage 1, file permissions are sufficient.

### Error Handling Strategy

| Scenario | Behavior |
|----------|----------|
| No bridges found | Show discovery panel with retry option |
| Bridge unreachable | Mark disconnected, show in status bar, continue with others |
| Auth key rejected | Clear stored key, prompt re-pairing |
| Partial API failure | Log error, show stale data indicator, retry on next poll |

---

## 3. Entity Modeling

### Using openhue-go Types Directly

We use openhue-go's auto-generated types (`LightGet`, `RoomGet`, `SceneGet`, etc.) directly rather than maintaining our own domain models. This provides:

- **Partial sync support** - Can fetch only lights without rebuilding a relationship graph
- **No translation overhead** - Types from API go straight to cache
- **Auto-updated with library** - New API fields available immediately
- **Lazy relationship resolution** - Resolve room→lights on-demand, not upfront

### BridgeState Structure

```go
type BridgeState struct {
    Lights        map[string]openhue.LightGet
    Rooms         map[string]openhue.RoomGet
    Scenes        map[string]openhue.SceneGet
    GroupedLights map[string]openhue.GroupedLightGet
    Devices       map[string]openhue.DeviceGet
}
```

### On-Demand Relationship Resolution

```go
// Resolve room → lights when needed, not at sync time
func (s *BridgeState) RoomLights(room openhue.RoomGet) []openhue.LightGet

// Resolve room → grouped light for control
func (s *BridgeState) RoomGroupedLight(room openhue.RoomGet) (openhue.GroupedLightGet, bool)

// Resolve room → scenes for scene picker
func (s *BridgeState) RoomScenes(roomID string) []openhue.SceneGet
```

### Entity Relationships

The Hue v2 API uses resource references (IDs). Key mappings:

| Entity | Contains | Controllable Via |
|--------|----------|------------------|
| Room | Devices (children) → Lights (via owner) | GroupedLight service |
| Light | - | Direct LightPut |
| Scene | Actions for a room/zone | Recall action |
| GroupedLight | Virtual aggregate of room/zone | GroupedLightPut |
| Device | Physical hardware, hosts services | Identify, rename |

### Partial Sync Strategy

```go
// Full sync on connect
bridge.SyncAll(ctx)  // Fetches lights, rooms, scenes, grouped lights, devices

// Efficient polling - only sync what changes frequently
bridge.SyncLights(ctx)        // Just lights (on tick)
bridge.SyncGroupedLights(ctx) // Just grouped lights (if needed)
```

---

## 4. TUI Layout and Navigation

### Primary Layout

```
┌────────────────────┬────────────────────────────────────────────────┐
│ [1] Bridges        │ [0] Details                                    │
│ ────────────────── │                                                │
│ > Living Room Hub● │  Living Room                                   │
│   Upstairs Hub   ○ │  ──────────────────────────────                │
├────────────────────┤  Status: 3/4 lights on                         │
│ [2] Groups         │  Brightness: 80%                               │
│ Rooms│Zones│Ent    │                                                │
│ ────────────────── │  Lights:                                       │
│ > Living Room   ● │    Ceiling Lamp      ● 100%                   │
│   Bedroom       ○ │    Floor Lamp        ● 60%                    │
├────────────────────┤    Table Light       ○ Off                    │
│ [3] Lights         │    Strip Light       ● 80%                    │
│ ────────────────── │                                                │
│   Ceiling Lamp  ● │  Scenes:                                       │
│   Floor Lamp    ● │    Energize  Relax  Concentrate  Dimmed       │
├────────────────────┤                                                │
│ [4] Devices        │                                                │
│ ────────────────── │                                                │
│   Hue Bridge       │                                                │
│   Motion Sensor    │                                                │
├────────────────────┤                                                │
│ [5] Scenes         │                                                │
│ ────────────────── │                                                │
│   Energize         │                                                │
│   Relax            │                                                │
├────────────────────┴────────────────────────────────────────────────┤
│ 0-5 panels  ↑↓ nav  ←→ tabs  ⏎ select  space toggle  P pair  q quit│
└─────────────────────────────────────────────────────────────────────┘
```

### Panel Architecture

| Key | Panel | Purpose | Component |
|-----|-------|---------|-----------|
| `0` | Details | Show selected entity details, child lights, scenes | `bubbles/viewport` |
| `1` | Bridges | List all bridges with connection status | `BridgePanel` with custom delegate |
| `2` | Groups | Tabbed view: Rooms, Zones, Entertainment | `TabbedPanel` with tabs |
| `3` | Lights | List all lights | `ListPanel` |
| `4` | Devices | List all devices (bridges, sensors, etc.) | `ListPanel` |
| `5` | Scenes | List all scenes | `ListPanel` |
| - | Status Bar | Context-sensitive keybindings, messages | Static render |

**Features:**
- **Number key shortcuts (0-5)**: Jump directly to any panel
- **Tab/Shift+Tab**: Cycle through panels sequentially
- **Arrow keys in Groups panel**: Switch between Rooms/Zones/Entertainment tabs
- **No auto-pairing**: User must explicitly press `P` to pair a bridge

### Focus Model

```go
type FocusRegion int

const (
    FocusDetail  FocusRegion = iota  // 0
    FocusBridges                     // 1
    FocusGroups                      // 2
    FocusLights                      // 3
    FocusDevices                     // 4
    FocusScenes                      // 5
)
```

- `Tab` / `Shift+Tab` cycles focus between major regions
- `h` / `l` (vim-style) also switch left/right panels
- `1`, `2`, `3` jump to Rooms, Zones, Scenes sections in entity list
- Focus state determines which keybindings are active

### Bridge Switching

- Header shows current bridge name/IP
- `B` opens bridge picker (popup list)
- `[` / `]` cycle through connected bridges quickly
- Disconnected bridges shown with indicator, selectable to attempt reconnect

---

## 5. Keybinding Design

### Core Navigation

| Key | Action | Context |
|-----|--------|---------|
| `j` / `↓` | Move cursor down | List panels |
| `k` / `↑` | Move cursor up | List panels |
| `g` | Jump to top | List panels |
| `G` | Jump to bottom | List panels |
| `Tab` | Next panel | Global |
| `Shift+Tab` | Previous panel | Global |
| `Enter` | Expand/select item | Entity list |
| `Esc` | Close overlay / deselect | Global |
| `q` | Quit | Global |
| `?` | Toggle help overlay | Global |

### Light/Group Actions

| Key | Action | Context |
|-----|--------|---------|
| `Space` | Toggle on/off | Light or group selected |
| `o` | Turn on | Light or group selected |
| `O` | Turn off | Light or group selected |
| `b` | Brightness picker (slider) | Light or group selected |
| `+` / `=` | Increase brightness 10% | Light or group selected |
| `-` | Decrease brightness 10% | Light or group selected |
| `c` | Color picker | Color-capable light |
| `t` | Color temperature picker | CT-capable light |

### Scene Actions

| Key | Action | Context |
|-----|--------|---------|
| `s` | Open scene picker for current room/zone | Room or zone selected |
| `Enter` | Activate scene | Scene list |
| `1`-`9` | Quick-activate scene by position | Scene list visible |

### Bulk Operations

| Key | Action | Context |
|-----|--------|---------|
| `v` | Enter visual/multi-select mode | Entity list |
| `a` | Select all in current section | Visual mode |
| `Space` | Toggle selection | Visual mode |
| `Enter` | Apply action to selection | Visual mode |

### Bridge Management

| Key | Action |
|-----|--------|
| `B` | Open bridge picker |
| `[` | Previous bridge |
| `]` | Next bridge |
| `R` | Refresh/resync current bridge |
| `P` | Pair new bridge |

### Mouse Support

Like lazygit, mouse interaction is fully supported as a complement to keyboard navigation:

- Click to select items in lists
- Click to switch focus between panels
- Scroll wheel for list/viewport navigation
- Double-click to activate (toggle light, recall scene)
- Mouse support enabled via `tea.WithMouseCellMotion()` program option

This provides accessibility for users who prefer mouse interaction while maintaining keyboard-first design.

### Discoverability

- Status bar shows 4-5 most relevant keys for current context
- `?` opens full help overlay organized by category
- Keys are grouped logically (navigation, actions, bulk, system)
- Conflicting keys are avoided by context separation

---

## 6. State Management and Concurrency

### Bubble Tea Model Structure

```go
type Model struct {
    // UI State
    focus        FocusRegion
    entityList   list.Model
    detailView   viewport.Model
    scenePicker  list.Model
    bridgePicker list.Model
    helpOverlay  viewport.Model
    showHelp     bool
    
    // Domain State
    bridges      *hue.BridgeManager
    activeBridge string
    
    // Derived/Cached
    currentEntities []domain.Entity
    selectedEntity  domain.Entity
    
    // System
    width, height int
    err           error
    statusMsg     string
}
```

### Message Types

```go
// Bridge lifecycle
type BridgesDiscoveredMsg struct { Bridges []hue.BridgeInfo }
type BridgeConnectedMsg struct { BridgeID string }
type BridgeDisconnectedMsg struct { BridgeID string; Err error }
type PairingRequiredMsg struct { Bridge hue.BridgeInfo }
type PairingSuccessMsg struct { BridgeID string; ApiKey string }

// State sync
type StateSyncedMsg struct { BridgeID string; State *domain.BridgeState }
type StateSyncErrorMsg struct { BridgeID string; Err error }

// Actions
type LightToggledMsg struct { LightID string; NewState bool }
type BrightnessChangedMsg struct { EntityID string; Brightness float64 }
type SceneActivatedMsg struct { SceneID string }
type ActionErrorMsg struct { Err error }
```

### Polling vs Event-Driven

**Approach: Polling with smart intervals**

```go
func (p *Poller) Start(ctx context.Context, send func(tea.Msg)) {
    ticker := time.NewTicker(2 * time.Second)
    defer ticker.Stop()
    
    for {
        select {
        case <-ctx.Done():
            return
        case <-ticker.C:
            for _, bridge := range p.manager.ConnectedBridges() {
                state, err := bridge.SyncState(ctx)
                if err != nil {
                    send(StateSyncErrorMsg{BridgeID: bridge.ID, Err: err})
                } else {
                    send(StateSyncedMsg{BridgeID: bridge.ID, State: state})
                }
            }
        }
    }
}
```

**Rationale:**

- Hue bridges support SSE (Server-Sent Events) but openhue-go doesn't expose them
- Polling at 2-second intervals provides responsive feel without overloading bridge
- Poll interval could be configurable (1-10 seconds)
- Optimistic updates: UI updates immediately on action, poll corrects if needed

### Latency Handling

| Strategy | Implementation |
|----------|----------------|
| Optimistic updates | Apply state change to local model before API call returns |
| Stale indicators | Show "syncing..." badge when last sync > 5s ago |
| Debounced actions | Brightness slider sends updates every 100ms max |
| Graceful degradation | If bridge unreachable, show cached state with warning |

---

## 7. Extensibility Considerations

### Plugin Points (Stage 2+)

```go
// Action hook interface
type ActionHook interface {
    Name() string
    Key() string
    Applies(entity domain.Entity) bool
    Execute(ctx context.Context, entity domain.Entity) error
}

// Entity provider for future sensor/automation support
type EntityProvider interface {
    EntityType() string
    Fetch(ctx context.Context, bridge *BridgeConnection) ([]domain.Entity, error)
}
```

### Planned Extensions

| Feature | Package Location | Notes |
|---------|------------------|-------|
| Schedules | `internal/domain/schedule.go` | Time-based automations |
| Sensors | `internal/domain/sensor.go` | Motion, temperature, buttons |
| Automations | `internal/domain/automation.go` | Rules and behaviors |
| Entertainment | `internal/hue/entertainment.go` | Streaming API integration |
| Custom Actions | `internal/plugins/` | User-defined keybindings |

### Configuration Evolution

```json
// ~/.config/lazyhue/config.json (future)
{
  "poll_interval": "2s",
  "default_bridge": "living-room-hub",
  "keybindings": {
    "toggle": "space",
    "brightness_up": "="
  },
  "themes": {
    "active": "dracula"
  }
}
```

---

## 8. Design Tradeoffs

| Decision | Alternatives Considered | Rationale |
|----------|------------------------|-----------|
| openhue-go types directly | Custom domain models | Enables partial queries; no translation layer; auto-updated with library |
| Polling over SSE | Implement SSE client | openhue-go doesn't support SSE; polling is simpler for Stage 1 |
| Partial sync (lights only) | Full sync on every tick | Lights change frequently; rooms/scenes rarely change; reduces network load |
| On-demand relationship resolution | Upfront graph building | Allows fetching rooms without lights; lazy loading for performance |
| File-based credentials | Keychain integration | Simpler initial implementation; keychain can be added later |
| Single main model | Nested sub-models | Easier to reason about for initial implementation; can refactor later |
| bubbles/list over custom | Custom virtual list | bubbles/list has filtering, good defaults; optimize only if needed |
| 2-panel layout | 3-panel (list/tree/detail) | Matches lazygit simplicity; tree navigation adds complexity |
| Vim-style keybindings | Emacs/hybrid | Target audience (devs) familiar with vim; matches lazygit precedent |

---

## 9. Startup Sequence

```mermaid
sequenceDiagram
    participant Main
    participant Config
    participant Discovery
    participant BridgeMgr
    participant TUI

    Main->>Config: LoadConfig()
    Main->>Config: LoadCredentials()
    Main->>Discovery: DiscoverBridges(timeout=2s)
    Discovery-->>Main: []BridgeInfo
    
    Main->>BridgeMgr: Initialize(bridges, credentials)
    
    loop Each known bridge with credentials
        BridgeMgr->>BridgeMgr: Connect(bridge)
        BridgeMgr->>BridgeMgr: SyncAll()
    end
    
    Main->>TUI: tea.NewProgram(model)
    TUI->>TUI: Run()
    
    Note over TUI: If no bridges connected,<br/>show discovery/pairing panel
```

**Fast Startup Goal:**

- Target: <500ms to first render
- Discovery runs with short timeout (2s)
- Initial render shows "connecting..." state
- Full state sync happens in background after UI appears

---

## 10. Testing Strategy

| Layer | Approach |
|-------|----------|
| Domain | Unit tests with mock data |
| Hue layer | Interface-based mocking (openhue-go provides mock) |
| UI components | Snapshot tests for View() output |
| Integration | Test against Hue bridge emulator or real hardware |

---

## Summary

This design prioritizes:

1. **Fast startup** through async discovery and background sync
2. **Keyboard-first UX** with single-key actions and vim-style navigation
3. **Multi-bridge support** as a first-class concern
4. **Clean architecture** separating UI, domain, and bridge communication
5. **Extensibility** through well-defined interfaces for future features

The implementation should proceed in phases:

1. Bridge discovery, auth, and credential storage
2. Entity fetching and domain model mapping
3. Basic TUI with entity list and detail view
4. Light control actions (toggle, brightness)
5. Scene activation
6. Multi-bridge switching
7. Polish (help overlay, status messages, error handling)

