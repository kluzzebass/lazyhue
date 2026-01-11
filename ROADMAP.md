# LazyHue Roadmap

A terminal UI for Philips Hue bridge management.

## ✅ Implemented Features

### Bridge Management
- [x] mDNS bridge discovery
- [x] Bridge pairing via link button
- [x] Multiple bridge support
- [x] Bridge switching
- [x] Forget bridge functionality
- [x] SSE event stream for real-time updates
- [x] Bulk resource fetching (`/clip/v2/resource`)

### Navigation & UI
- [x] Keyboard-driven navigation
- [x] Mouse support (click, scroll)
- [x] Multi-panel layout (bridges, hierarchy, details, activity log)
- [x] Tree view with expand/collapse
- [x] Tabbed views (Home, Lights, Devices, Scenes)
- [x] Help overlay with keybindings
- [x] Temporary status messages with auto-clear
- [x] Activity log (events and API requests)
- [x] Toggleable activity panel (`a` key)

### Entity Management
- [x] View rooms, zones, lights, devices, scenes
- [x] Entity details panel with comprehensive info
- [x] Rename any entity (`r` key)
  - Bridges (via device)
  - Devices
  - Lights (via parent device)
  - Rooms
  - Zones
  - Scenes
- [x] Text case transformations (`Ctrl+T` title case, `Ctrl+L` sentence case)

### Light Control
- [x] Toggle lights on/off (space)
- [x] Turn on/off explicitly (`o`/`O`)
- [x] Brightness up/down (`+`/`-`)
- [x] Scene activation (Enter on scene)

### Device Configuration
- [x] Motion sensor enable/disable
- [x] Motion sensor sensitivity adjustment

### Form System
- [x] Generic form dialog with Save/Cancel
- [x] Toggle fields (boolean)
- [x] Slider fields (integer with min/max)
- [x] Text input fields
- [x] Brightness slider (visual gradient)
- [x] Color temperature slider (warm to cool)
- [x] Color picker (XY color space)
- [x] Select/dropdown fields
- [x] Live mode (changes apply immediately)
- [x] Edit mode (changes on Save)

### Live Light Editing
- [x] Wire up form system to light control
- [x] Real-time brightness adjustment
- [x] Real-time color adjustment
- [x] Real-time color temperature adjustment
- [x] Debouncing for API calls
- [x] Color picker dialog (HSV color wheel)
- [x] Color temperature dialog (warm to cool slider)
- [x] Effects control (candle, fire, prism, sparkle, etc.)

---

## 📋 Planned Features

### Light Control (Phase 2)
- [ ] Gradient control for gradient-capable lights (needs hardware to test)

### Room/Zone Management
- [x] Create new room (`n` key)
- [x] Create new zone (`N` key)
- [x] Delete room/zone (`d` key)
- [x] Room archetype selection (in create/edit dialog)
- [x] Remove devices from room (`e` to edit)

### Scene Management
- [ ] Create new scene
- [ ] Delete scene
- [ ] Edit scene (light states)
- [ ] Dynamic scene settings
- [ ] Scene preview

### Device Management
- [ ] Temperature sensor display
- [ ] Light level sensor display
- [ ] Battery level monitoring
- [ ] Button configuration viewing
- [ ] Device identify (flash)

### Advanced Features
- [ ] Entertainment areas viewing
- [ ] Streaming status
- [ ] Automation/schedule viewing
- [ ] Firmware update status
- [ ] Multi-select operations (bulk toggle, rename, etc.)
- [ ] Search/filter across entities
- [ ] Undo/redo for actions

### UI Enhancements
- [ ] Configurable color themes
- [ ] Configurable keybindings
- [ ] Column resizing
- [ ] Export/import configuration
- [ ] Notification sounds (optional)

---

## 💡 Ideas / Maybe

- [ ] Homekit code display
- [ ] Bridge diagnostics
- [ ] Network statistics
- [ ] Zigbee mesh visualization
- [ ] Integration with other smart home systems
- [ ] Scripting/macro support
- [ ] REST API mode (headless)

---

## Technical Debt

- [ ] Add comprehensive error handling
- [ ] Add unit tests
- [ ] Add integration tests
- [ ] Improve documentation
- [ ] Performance profiling for large setups
