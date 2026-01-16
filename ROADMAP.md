# LazyHue Roadmap

A terminal UI for Philips Hue bridge management.

> **API Coverage:** LazyHue currently uses ~8.5% of the available Hue API operations.
> See [docs/api-coverage.md](docs/api-coverage.md) for detailed analysis.
>
> **Data Utilization:** Many fetched endpoints have unused fields.
> See [docs/endpoint-utilization.md](docs/endpoint-utilization.md) for opportunities.

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

### Room/Zone Management (100% API coverage)
- [x] Create new room (`n` key)
- [x] Create new zone (`N` key)
- [x] Delete room/zone (`x` key)
- [x] Room archetype selection
- [x] Remove devices from room

### Light Control (100% API coverage)
- [x] Toggle lights on/off (space)
- [x] Turn on/off explicitly (`o`/`O`)
- [x] Brightness up/down (`+`/`-`)
- [x] Scene activation (Enter on scene)
- [x] Gradient control for gradient-capable lights
- [x] Effects control (candle, fire, prism, sparkle, etc.)

### Device Configuration
- [x] Motion sensor enable/disable
- [x] Motion sensor sensitivity adjustment
- [x] Temperature sensor display
- [x] Light level sensor display
- [x] Battery level monitoring

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

---

## 🎯 Quick Wins (API ready, needs UI)

These features have API support already implemented or partially implemented:

### Connectivity Status Display
- [ ] Show WiFi connectivity status (code in `extended_client.go`)
- [ ] Show Zigbee channel info (data already fetched)
- [ ] Display extended PAN ID for network identification

### Device Enhancements
- [ ] Software/firmware update status display (new endpoint)
- [ ] Device identify (flash) - trigger LED blink to locate device
- [ ] Show device mode for switches (pushbutton vs rocker)

### Light Enhancements (data already fetched)
- [ ] EffectsV2 speed control - adjust effect animation speed
- [ ] TimedEffects - sunrise/sunset wake-up controls
- [ ] Signaling - alert patterns (alternating, on_off, on_off_color)
- [ ] Color gamut type display (A/B/C classification)
- [ ] Light identify (flash) button

### Scene Enhancements (data already fetched)
- [ ] Display scene color palette for dynamic scenes
- [ ] Show scene image reference
- [ ] SmartScene active timeslot indicator
- [ ] SmartScene weekly schedule visualization
- [ ] SmartScene transition duration display

---

## 📋 Planned Features

### Scene Management (currently 37.5% API coverage)
- [ ] **Create new scene** (high priority)
- [ ] Edit scene light states
- [ ] Dynamic scene settings
- [ ] Scene preview
- [ ] Smart scene creation

### Button & Switch Support
- [ ] Display button device events
- [ ] Show doorbell notifications
- [ ] Rotary dial status
- [ ] Button configuration viewing

### Advanced Device Features
- [ ] Motion area configuration
- [ ] Grouped motion sensor management
- [ ] Camera motion detection status

### UI Enhancements
- [ ] Multi-select operations (bulk toggle, rename, etc.)
- [ ] Search/filter across entities
- [ ] Undo/redo for actions
- [ ] Configurable color themes
- [ ] Configurable keybindings
- [ ] Column resizing

---

## 🚀 Future Features (API available, not implemented)

### Entertainment & Sync
- [ ] Entertainment area configuration
- [ ] Hue Sync / streaming mode control
- [ ] Music-reactive lighting setup
- [ ] Video sync configuration

### Automation Engine
- [ ] View behavior scripts
- [ ] Create automation rules
- [ ] Schedule management
- [ ] Conditional triggers

### Geofencing
- [ ] Register geofence clients
- [ ] Location-based automation
- [ ] Presence detection rules

### Smart Home Integration
- [ ] HomeKit pairing status
- [ ] Matter fabric management
- [ ] Protocol bridge status

### Audio/Speakers
- [ ] Speaker device control
- [ ] Audio zone management

---

## 💡 Ideas / Maybe

- [ ] Homekit code display
- [ ] Bridge diagnostics
- [ ] Network statistics
- [ ] Zigbee mesh visualization
- [ ] Integration with other smart home systems
- [ ] Scripting/macro support
- [ ] REST API mode (headless)
- [ ] Export/import configuration
