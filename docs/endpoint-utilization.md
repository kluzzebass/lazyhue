# Endpoint Data Utilization Analysis

This document tracks what data is available in the Hue API endpoints LazyHue already uses versus what is actually being displayed or utilized.

**Last updated:** January 2026

## Summary

LazyHue fetches data from several endpoints but only uses a subset of the available fields. This represents low-hanging fruit for feature additions since the data is already being retrieved.

---

## Light Endpoint (`LightGet`)

### Currently Used

| Field | Usage |
|-------|-------|
| `On` | Power toggle |
| `Dimming` | Brightness control |
| `Color` | XY color picker |
| `ColorTemperature` | Mirek temperature slider |
| `Gradient` | Gradient point control |
| `Effects` (v1) | Effect selection (deprecated) |
| `EffectsV2` | Effect selection with speed control |
| `EffectsV2.Status.Parameters.Speed` | Speed slider when effect is active |
| `Dynamics.Status` | Dynamic status display |
| `Dynamics.Speed` | Speed display when valid |
| `Powerup.Preset` | Power-on behavior selection |
| `Signaling.SignalValues` | Available signal types display + trigger buttons |
| `Signaling.Status` | Active signal status display |
| `Signaling` (duration) | Custom duration slider (5-60 seconds) |
| `TimedEffects.EffectValues` | Available timed effects (sunrise/sunset) + trigger buttons |
| `TimedEffects.Status` | Active timed effect status |
| `TimedEffects` (duration) | Custom duration slider (5-120 minutes) |
| `Identify` | Identify button triggers flash |
| `Color.GamutType` | Color capability tier display (A/B/C/other) |

### Available but NOT Used

| Field | Description | Potential Feature |
|-------|-------------|-------------------|
| `ContentConfiguration` | Pixel order/orientation | Lightstrip configuration |
| `Alert` | Breathe effect | Alternative identify method |
| `Color.Gamut` | Precise color boundaries | Accurate color picker constraints |

---

## Device Endpoint (`DeviceGet`)

### Currently Used

| Field | Usage |
|-------|-------|
| `Metadata.Name` | Device name display/edit |
| `Metadata.Archetype` | Device icon/type |
| `ProductData.*` | Manufacturer, model, software version |
| `Services` | Linked resource references |
| `Identify` | Identify button triggers flash |

### Available but NOT Used

| Field | Description | Potential Feature |
|-------|-------------|-------------------|
| `DeviceMode` | Switch configuration modes | Show/configure switch type |
| `DeviceMode.Mode` | `switch_dual_pushbutton`, `switch_dual_rocker`, etc. | Switch mode selector |
| `Usertest` | Identification mode (LED flash for 120s) | Extended identify mode |

---

## Scene Endpoint (`SceneGet`)

### Currently Used

| Field | Usage |
|-------|-------|
| `Metadata.Name` | Scene name display/edit |
| `Group` | Parent room/zone |
| `Actions` | Light states in scene |
| `Speed` | Dynamic speed display |
| `AutoDynamic` | Auto-dynamic flag |
| `Status.Active` | Active state indicator |

### Available but NOT Used

| Field | Description | Potential Feature |
|-------|-------------|-------------------|
| `Palette` | Dynamic scene color palette | Color palette editor |
| `Palette.Color` | Colors for dynamics | Visual palette display |
| `Palette.ColorTemperature` | Temperature values | Temperature range display |
| `Palette.Dimming` | Brightness levels | Brightness range display |
| `Palette.EffectsV2` | Effect settings | Effect configuration |
| `Metadata.Image` | Scene image reference | Scene thumbnail |
| `Metadata.Appdata` | App-specific data | Custom metadata display |

---

## SmartScene Endpoint (`SmartSceneGet`)

### Currently Used

| Field | Usage |
|-------|-------|
| `Metadata.Name` | Scene name |
| `Group` | Parent room/zone |
| `State` | Active/inactive state |

### Available but NOT Used

| Field | Description | Potential Feature |
|-------|-------------|-------------------|
| `ActiveTimeslot` | Currently active slot | Show current timeslot |
| `ActiveTimeslot.Weekday` | Day of week | Day indicator |
| `ActiveTimeslot.TimeslotId` | Slot index | Slot indicator |
| `WeekTimeslots` | Full weekly schedule | Schedule visualization |
| `TransitionDuration` | Timeslot transition time | Transition time display/edit |

---

## Zigbee Connectivity (`ZigbeeConnectivityGet`)

### Currently Used

| Field | Usage |
|-------|-------|
| `Status` | Connection status (connected/disconnected) |
| `MacAddress` | Device MAC address |
| `Channel.Value` | Zigbee channel number display |
| `Channel.Status` | Channel change indicator |

### Available but NOT Used

| Field | Description | Potential Feature |
|-------|-------------|-------------------|
| `ExtendedPanId` | Network identifier | Network info display |

---

## DeviceSoftwareUpdate (`DeviceSoftwareUpdateGet`)

### Currently Used

| Field | Usage |
|-------|-------|
| `State` | Update status display in device details |
| `Owner` | Links update to device |

### Available but NOT Used

| Field | Description | Potential Feature |
|-------|-------------|-------------------|
| `Problems` | Issues requiring attention | Problem alerts |

---

## Implementation Status

### Completed

1. ~~**Zigbee Channel**~~ - Display in device details Zigbee section
2. ~~**EffectsV2 Speed**~~ - Display + interactive slider control
3. ~~**TimedEffects**~~ - Display + trigger buttons (sunrise/sunset with 30min default)
4. ~~**Signaling**~~ - Display + trigger buttons (15sec default duration)
5. ~~**Device Identify**~~ - Flash button in device and light details
6. ~~**Firmware Status**~~ - Fetch and display update availability
7. ~~**Color Gamut Type**~~ - Show A/B/C classification in Capabilities section
8. ~~**Signaling with custom duration**~~ - Slider control (5-60 seconds)
9. ~~**TimedEffects with custom duration**~~ - Slider control (5-120 minutes)

### Higher Effort (new UI)

1. **Scene Palette Editor** - Visual palette configuration
2. **SmartScene Schedule View** - Weekly timeslot visualization
3. **ContentConfiguration** - Lightstrip pixel orientation
4. **DeviceMode Configuration** - Switch type selection
