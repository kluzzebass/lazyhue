# Hue API Coverage Analysis

This document tracks the differential between what the `hueclient` (generated from the oapi-hue OpenAPI spec) supports and what LazyHue currently implements.

**Last updated:** January 2025
**OpenAPI spec version:** oapi-hue v0.3.0

## Summary

| Metric | Value |
|--------|-------|
| Total API operations available | 350+ |
| Operations currently in use | ~30 |
| Coverage | ~8.5% |

## Currently Implemented

### Full Coverage (100%)

#### Lights
| Operation | Used | Location |
|-----------|------|----------|
| GetLights | ✓ | state.go |
| GetLightById | ✓ | state.go |
| UpdateLight | ✓ | actions.go |
| UpdateLightWithBody | ✓ | actions.go (gradients) |
| GetLightLevels | ✓ | state.go |
| UpdateLightLevel | ✓ | actions.go |

#### Rooms
| Operation | Used | Location |
|-----------|------|----------|
| GetRooms | ✓ | state.go |
| GetRoomById | ✓ | state.go |
| CreateRoom | ✓ | actions.go |
| UpdateRoom | ✓ | actions.go |
| DeleteRoom | ✓ | actions.go |

#### Zones
| Operation | Used | Location |
|-----------|------|----------|
| GetZones | ✓ | state.go |
| GetZoneById | ✓ | state.go |
| CreateZone | ✓ | actions.go |
| UpdateZone | ✓ | actions.go |
| DeleteZone | ✓ | actions.go |

#### Temperature Sensors
| Operation | Used | Location |
|-----------|------|----------|
| GetTemperatures | ✓ | state.go |
| UpdateTemperature | ✓ | actions.go |

### Partial Coverage

#### Grouped Lights (37.5%)
| Operation | Used | Notes |
|-----------|------|-------|
| GetGroupedLights | ✓ | |
| GetGroupedLightById | ✓ | |
| UpdateGroupedLight | ✓ | |
| UpdateGroupedLightWithBody | ✗ | Could use for bulk operations |
| DeleteGroupedLight | ✗ | |

#### Scenes (37.5%)
| Operation | Used | Notes |
|-----------|------|-------|
| GetScenes | ✓ | |
| GetSceneById | ✓ | |
| CreateScene | ✗ | **High priority** |
| UpdateScene | ✓ | For recall |
| DeleteScene | ✓ | |
| GetSmartScenes | ✓ | |
| UpdateSmartScene | ✓ | |
| DeleteSmartScene | ✗ | |

#### Devices (33%)
| Operation | Used | Notes |
|-----------|------|-------|
| GetDevices | ✓ | |
| GetDeviceById | ✓ | |
| UpdateDevice | ✓ | Name/archetype |
| DeleteDevice | ✗ | |
| GetDevicePowers | ✓ | Battery levels |
| GetDeviceSoftwareUpdate | ✗ | Firmware status |
| UpdateDeviceSoftwareUpdate | ✗ | Trigger updates |

#### Motion Sensors (11%)
| Operation | Used | Notes |
|-----------|------|-------|
| GetMotions | ✓ | |
| UpdateMotion | ✓ | Enable/disable |
| GetCameraMotion | ✗ | Camera-based detection |
| GetGroupedMotion | ✗ | Motion groups |
| GetConvenienceAreaMotion | ✗ | |
| GetSecurityAreaMotion | ✗ | |

#### Bridges (33%)
| Operation | Used | Notes |
|-----------|------|-------|
| GetBridges | ✓ | |
| GetBridgeById | ✓ | |
| UpdateBridge | ✗ | |
| GetBridgeHomes | ✗ | |

## Not Implemented (0%)

### Entertainment Configuration
Control Hue Sync and entertainment areas for music/video sync.

| Operation | Description |
|-----------|-------------|
| GetEntertainmentConfigurations | List entertainment areas |
| GetEntertainmentConfigurationById | Get specific area |
| CreateEntertainmentConfiguration | Create entertainment area |
| UpdateEntertainmentConfiguration | Modify settings |
| DeleteEntertainmentConfiguration | Remove area |
| StartEntertainmentStream | Begin streaming mode |
| StopEntertainmentStream | End streaming mode |

### Behavior/Automation
Rules engine for conditional lighting automation.

| Operation | Description |
|-----------|-------------|
| GetBehaviorScripts | List available behaviors |
| GetBehaviorInstances | List active automations |
| CreateBehaviorInstance | Create automation rule |
| UpdateBehaviorInstance | Modify rule |
| DeleteBehaviorInstance | Remove rule |

### Geofencing
Location-based lighting triggers.

| Operation | Description |
|-----------|-------------|
| GetGeofenceClients | List registered devices |
| CreateGeofenceClient | Register device |
| UpdateGeofenceClient | Update location |
| DeleteGeofenceClient | Remove device |
| GetGeolocation | Bridge location settings |
| UpdateGeolocation | Set bridge location |

### Motion Areas
Advanced motion zone configuration.

| Operation | Description |
|-----------|-------------|
| GetMotionAreaCandidates | Potential motion zones |
| GetMotionAreaConfigurations | Current zone configs |
| CreateMotionAreaConfiguration | Create motion zone |
| UpdateMotionAreaConfiguration | Modify zone |
| DeleteMotionAreaConfiguration | Remove zone |

### Connectivity
Network and protocol status.

| Operation | Description | Notes |
|-----------|-------------|-------|
| GetZigbeeConnectivity | Zigbee mesh status | **Already in extended_client.go** |
| GetWifiConnectivity | WiFi status | **Already in extended_client.go** |
| UpdateWifiConnectivity | Configure WiFi | |

### HomeKit/Matter
Smart home protocol integrations.

| Operation | Description |
|-----------|-------------|
| GetHomekitStatus | HomeKit pairing status |
| UpdateHomekit | HomeKit settings |
| ResetHomekit | Unpair HomeKit |
| GetMatter | Matter status |
| UpdateMatter | Matter settings |
| GetMatterFabrics | Matter fabric list |
| DeleteMatterFabric | Remove fabric |

### Buttons & Rotary Controls
Physical device input handling.

| Operation | Description |
|-----------|-------------|
| GetButtons | List button devices |
| UpdateButton | Configure button |
| GetRelativeRotary | Rotary dial devices |
| UpdateRelativeRotary | Configure dial |

### Audio/Speakers
Media and audio control.

| Operation | Description |
|-----------|-------------|
| GetSpeakers | List speaker devices |
| UpdateSpeaker | Control speaker |

### Service Groups
Logical device groupings.

| Operation | Description |
|-----------|-------------|
| GetServiceGroups | List service groups |
| CreateServiceGroup | Create group |
| UpdateServiceGroup | Modify group |
| DeleteServiceGroup | Remove group |

## Implementation Priority

### Quick Wins (code partially exists)
1. **Connectivity display** - WiFi/Zigbee status already in `extended_client.go`, just needs UI
2. **Device power monitoring** - API calls exist, enhance display
3. **Software update notifications** - Add firmware status to device details

### Medium Effort
4. **Scene creation** - High user value, moderate complexity
5. **Button/doorbell events** - Show physical device interactions
6. **Motion area configuration** - Advanced motion zone setup

### High Effort / High Value
7. **Entertainment configuration** - Hue Sync / music-reactive lighting
8. **Automation engine** - Rules and schedules
9. **Geofencing** - Location-based automation

## File Locations

| File | Purpose |
|------|---------|
| `internal/hueclient/client.go` | Generated API client (do not edit) |
| `internal/hue/state.go` | State synchronization (Get* calls) |
| `internal/hue/actions.go` | User actions (Update/Create/Delete) |
| `internal/hue/extended_client.go` | Custom API extensions |
| `internal/hue/bridge.go` | Bridge management and SSE |
