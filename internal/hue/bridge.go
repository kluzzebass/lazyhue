package hue

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/kluzzebass/lazyhue/internal/debug"
	"github.com/kluzzebass/lazyhue/internal/hueclient"
)

// ConnectionStatus represents the state of a bridge connection.
type ConnectionStatus int

const (
	StatusDisconnected ConnectionStatus = iota
	StatusConnecting
	StatusConnected
	StatusPairing
	StatusError
)

func (s ConnectionStatus) String() string {
	switch s {
	case StatusDisconnected:
		return "disconnected"
	case StatusConnecting:
		return "connecting"
	case StatusConnected:
		return "connected"
	case StatusPairing:
		return "pairing"
	case StatusError:
		return "error"
	default:
		return "unknown"
	}
}

// Bridge represents a connection to a single Hue bridge.
type Bridge struct {
	Info     BridgeInfo
	Status   ConnectionStatus
	LastSync time.Time
	LastErr  error

	client   *hueclient.ClientWithResponses // Main API client
	extended *ExtendedClient                // Custom APIs not in spec (auth apps, entertainment, wifi, zigbee)
	state       *BridgeState
	eventStream *EventStream
	eventCancel context.CancelFunc
	onEvent func(bridgeID, resourceType, resourceID, eventType string) // Callback when events are received
	onRequest func(bridgeID, message string)                           // Callback when API requests are made
	mu          sync.RWMutex
}

// NewBridge creates a new bridge connection.
func NewBridge(info BridgeInfo) *Bridge {
	return &Bridge{
		Info:   info,
		Status: StatusDisconnected,
		state:  NewBridgeState(),
	}
}

// Connect establishes a connection to the bridge using the provided API key.
func (b *Bridge) Connect(apiKey string) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.Status = StatusConnecting

	// Create new generated client
	httpClient := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}
	client, err := hueclient.NewClientWithResponses(
		"https://"+b.Info.IPAddress,
		hueclient.WithHTTPClient(httpClient),
		hueclient.WithRequestEditorFn(func(ctx context.Context, req *http.Request) error {
			req.Header.Set("hue-application-key", apiKey)
			return nil
		}),
	)
	if err != nil {
		b.Status = StatusError
		b.LastErr = err
		return err
	}
	b.client = client

	// Create extended client for custom APIs not in the spec
	extended, err := NewExtendedClient(b.Info.IPAddress, apiKey)
	if err != nil {
		b.Status = StatusError
		b.LastErr = err
		return err
	}

	b.extended = extended
	b.Status = StatusConnected
	b.LastErr = nil

	// Start event stream
	b.startEventStream(apiKey)

	return nil
}

// OnEvent sets a callback for when the bridge receives SSE events.
// The callback receives bridgeID, resourceType, resourceID, and eventType.
func (b *Bridge) OnEvent(fn func(bridgeID, resourceType, resourceID, eventType string)) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.onEvent = fn
}

// OnRequest sets a callback for when the bridge makes API requests.
// The callback receives bridgeID and a descriptive message.
func (b *Bridge) OnRequest(fn func(bridgeID, message string)) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.onRequest = fn
}

// logRequest calls the request callback if set.
func (b *Bridge) logRequest(message string) {
	b.mu.RLock()
	callback := b.onRequest
	b.mu.RUnlock()
	if callback != nil {
		callback(b.Info.ID, message)
	}
}

// IsEventStreamConnected returns whether the SSE connection is active.
func (b *Bridge) IsEventStreamConnected() bool {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.eventStream != nil && b.eventStream.IsConnected()
}

func (b *Bridge) startEventStream(apiKey string) {
	// Stop any existing stream
	if b.eventCancel != nil {
		b.eventCancel()
	}

	es := NewEventStream(b.Info.IPAddress, apiKey)

	es.OnEvent(func(container EventContainer) {
		b.handleEvents(container)
	})

	es.OnConnect(func() {
		// Event stream connected - we can reduce polling frequency
	})

	es.OnDisconnect(func(err error) {
		// Event stream disconnected - might want to increase polling
	})

	ctx, cancel := context.WithCancel(context.Background())
	b.eventStream = es
	b.eventCancel = cancel

	// Start in background
	go es.Start(ctx)
}

// handleEvents processes events from the SSE stream.
// Applies partial updates directly to the in-memory state, avoiding API calls.
func (b *Bridge) handleEvents(container EventContainer) {
	if len(container.Events) == 0 {
		return
	}

	b.mu.RLock()
	callback := b.onEvent
	b.mu.RUnlock()

	for _, event := range container.Events {
		updates, err := ParseResourceUpdates(event)
		if err != nil {
			continue
		}

		for _, update := range updates {
			debug.Log("SSE %s: %s %s", event.Type, update.Type, update.ID)

			// Apply the update directly to our cached state (if we handle this type)
			b.applyResourceUpdate(update)

			// Notify listener for all events, even unhandled ones
			if callback != nil {
				callback(b.Info.ID, update.Type, update.ID, string(event.Type))
			}
		}
	}
}

// applyResourceUpdate applies a single resource update to the bridge state.
// Returns true if the update was applied successfully.
func (b *Bridge) applyResourceUpdate(update ResourceUpdate) bool {
	if b.state == nil {
		return false
	}

	switch update.Type {
	case "light":
		lightUpdate := LightUpdate{}
		if update.On != nil {
			lightUpdate.On = &update.On.On
		}
		if update.Dimming != nil {
			lightUpdate.Brightness = &update.Dimming.Brightness
		}
		if update.Color != nil {
			lightUpdate.ColorXY = &[2]float64{update.Color.XY.X, update.Color.XY.Y}
		}
		if update.ColorTemperature != nil && update.ColorTemperature.Mirek != nil {
			lightUpdate.Mirek = update.ColorTemperature.Mirek
		}
		return b.state.ApplyLightUpdate(update.ID, lightUpdate)

	case "grouped_light":
		var on *bool
		var brightness *float64
		if update.On != nil {
			on = &update.On.On
		}
		if update.Dimming != nil {
			brightness = &update.Dimming.Brightness
		}
		return b.state.ApplyGroupedLightUpdate(update.ID, on, brightness)

	case "motion":
		if update.Motion != nil {
			return b.state.ApplyMotionUpdate(update.ID, update.Motion.Motion)
		}
		return false

	case "scene":
		if update.Status != nil && update.Status.Active != "" {
			debug.Log("Scene %s status: %s", update.ID, update.Status.Active)
			return b.state.ApplySceneStatus(update.ID, update.Status.Active)
		}
		if update.Metadata != nil && update.Metadata.Name != nil {
			return b.state.ApplySceneMetadata(update.ID, update.Metadata.Name)
		}
		return false

	case "device":
		if update.Metadata != nil && update.Metadata.Name != nil {
			return b.state.ApplyDeviceMetadata(update.ID, update.Metadata.Name)
		}
		return false

	case "room":
		if update.Metadata != nil && update.Metadata.Name != nil {
			return b.state.ApplyRoomMetadata(update.ID, update.Metadata.Name)
		}
		return false

	case "zone":
		if update.Metadata != nil && update.Metadata.Name != nil {
			return b.state.ApplyZoneMetadata(update.ID, update.Metadata.Name)
		}
		return false

	default:
		// Unknown resource type - will be picked up by fallback polling
		return false
	}
}

// Disconnect closes the bridge connection.
func (b *Bridge) Disconnect() {
	b.mu.Lock()
	defer b.mu.Unlock()

	// Stop event stream
	if b.eventCancel != nil {
		b.eventCancel()
		b.eventCancel = nil
	}
	b.eventStream = nil

	b.client = nil
	b.extended = nil
	b.Status = StatusDisconnected
}

// IsConnected returns true if the bridge is connected.
func (b *Bridge) IsConnected() bool {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.Status == StatusConnected && b.client != nil
}

// GetState returns the current cached state.
func (b *Bridge) GetState() *BridgeState {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.state
}


// SyncAll fetches all entity types from the bridge.
func (b *Bridge) SyncAll(ctx context.Context) error {
	if err := b.SyncLights(ctx); err != nil {
		return err
	}
	if err := b.SyncRooms(ctx); err != nil {
		return err
	}
	if err := b.SyncZones(ctx); err != nil {
		return err
	}
	if err := b.SyncGroupedLights(ctx); err != nil {
		return err
	}
	if err := b.SyncScenes(ctx); err != nil {
		return err
	}
	if err := b.SyncDevices(ctx); err != nil {
		return err
	}
	// Sensor services (non-fatal if they fail - device may not have them)
	_ = b.SyncMotionSensors(ctx)
	_ = b.SyncTemperatures(ctx)
	_ = b.SyncLightLevels(ctx)
	_ = b.SyncDevicePowers(ctx)

	// Entertainment configurations (non-fatal)
	_ = b.SyncEntertainmentConfigurations(ctx)
	_ = b.SyncWifiConnectivity(ctx)
	_ = b.SyncZigbeeConnectivity(ctx)

	// Bridge resource, home, and auth apps (non-fatal)
	_ = b.SyncBridgeResource(ctx)
	_ = b.SyncBridgeHome(ctx)
	_ = b.SyncAuthApps(ctx)

	b.mu.Lock()
	b.LastSync = time.Now()
	b.mu.Unlock()
	return nil
}

// SyncAllBulk fetches all resources in a single API call.
// This is more efficient than SyncAll which makes 12+ separate calls.
func (b *Bridge) SyncAllBulk(ctx context.Context) error {
	b.mu.RLock()
	client := b.client
	extended := b.extended
	b.mu.RUnlock()

	if client == nil {
		return ErrAuthFailed
	}

	resp, err := client.GetResourcesWithResponse(ctx)
	if err != nil {
		return err
	}

	if resp.StatusCode() != 200 || len(resp.Body) == 0 {
		return ErrAuthFailed
	}

	// Parse the raw JSON to extract full objects by type
	var envelope struct {
		Data   []json.RawMessage `json:"data"`
		Errors []struct {
			Description string `json:"description"`
		} `json:"errors"`
	}
	if err := json.Unmarshal(resp.Body, &envelope); err != nil {
		return err
	}

	// Temporary maps for each resource type
	lights := make(map[string]hueclient.LightGet)
	rooms := make(map[string]hueclient.RoomGet)
	zones := make(map[string]hueclient.RoomGet)
	groupedLights := make(map[string]hueclient.GroupedLightGet)
	scenes := make(map[string]hueclient.SceneGet)
	devices := make(map[string]hueclient.DeviceGet)
	motions := make(map[string]hueclient.MotionGet)
	temperatures := make(map[string]hueclient.TemperatureGet)
	lightLevels := make(map[string]hueclient.LightLevelGet)
	devicePowers := make(map[string]hueclient.DevicePowerGet)
	bridges := make(map[string]hueclient.BridgeGet)
	bridgeHomes := make(map[string]hueclient.BridgeHomeGet)

	// Parse each resource based on its type
	for _, raw := range envelope.Data {
		// First extract just the type
		var typeOnly struct {
			Type string `json:"type"`
			ID   string `json:"id"`
		}
		if err := json.Unmarshal(raw, &typeOnly); err != nil {
			continue
		}

		switch typeOnly.Type {
		case "light":
			var light hueclient.LightGet
			if err := json.Unmarshal(raw, &light); err == nil && light.Id != nil {
				lights[*light.Id] = light
			}
		case "room":
			var room hueclient.RoomGet
			if err := json.Unmarshal(raw, &room); err == nil && room.Id != nil {
				rooms[*room.Id] = room
			}
		case "zone":
			var zone hueclient.RoomGet
			if err := json.Unmarshal(raw, &zone); err == nil && zone.Id != nil {
				zones[*zone.Id] = zone
			}
		case "grouped_light":
			var gl hueclient.GroupedLightGet
			if err := json.Unmarshal(raw, &gl); err == nil && gl.Id != nil {
				groupedLights[*gl.Id] = gl
			}
		case "scene":
			var scene hueclient.SceneGet
			if err := json.Unmarshal(raw, &scene); err == nil && scene.Id != nil {
				scenes[*scene.Id] = scene
			}
		case "device":
			var device hueclient.DeviceGet
			if err := json.Unmarshal(raw, &device); err == nil && device.Id != nil {
				devices[*device.Id] = device
			}
		case "motion":
			var motion hueclient.MotionGet
			if err := json.Unmarshal(raw, &motion); err == nil && motion.Id != nil {
				motions[*motion.Id] = motion
			}
		case "temperature":
			var temp hueclient.TemperatureGet
			if err := json.Unmarshal(raw, &temp); err == nil && temp.Id != nil {
				temperatures[*temp.Id] = temp
			}
		case "light_level":
			var ll hueclient.LightLevelGet
			if err := json.Unmarshal(raw, &ll); err == nil && ll.Id != nil {
				lightLevels[*ll.Id] = ll
			}
		case "device_power":
			var dp hueclient.DevicePowerGet
			if err := json.Unmarshal(raw, &dp); err == nil && dp.Id != nil {
				devicePowers[*dp.Id] = dp
			}
		case "bridge":
			var br hueclient.BridgeGet
			if err := json.Unmarshal(raw, &br); err == nil && br.Id != nil {
				bridges[*br.Id] = br
			}
		case "bridge_home":
			var bh hueclient.BridgeHomeGet
			if err := json.Unmarshal(raw, &bh); err == nil && bh.Id != nil {
				bridgeHomes[*bh.Id] = bh
			}
		}
	}

	// Update state with all collected resources
	b.state.UpdateLights(lights)
	b.state.UpdateRooms(rooms)
	b.state.UpdateZones(zones)
	b.state.UpdateGroupedLights(groupedLights)
	b.state.UpdateScenes(scenes)
	b.state.UpdateDevices(devices)
	b.state.UpdateMotionSensors(motions)
	b.state.UpdateTemperatures(temperatures)
	b.state.UpdateLightLevels(lightLevels)
	b.state.UpdateDevicePowers(devicePowers)

	// Bridge and BridgeHome are single resources, extract first from map
	for _, br := range bridges {
		b.state.UpdateBridgeResource(&br)
		break
	}
	for _, bh := range bridgeHomes {
		b.state.UpdateBridgeHome(&bh)
		break
	}

	// These still need separate calls (not in /resource endpoint)
	if extended != nil {
		_ = b.SyncEntertainmentConfigurations(ctx)
		_ = b.SyncWifiConnectivity(ctx)
		_ = b.SyncZigbeeConnectivity(ctx)
		_ = b.SyncAuthApps(ctx)
	}

	b.mu.Lock()
	b.LastSync = time.Now()
	b.mu.Unlock()

	return nil
}

// SyncLights fetches only lights from the bridge.
func (b *Bridge) SyncLights(ctx context.Context) error {
	b.mu.RLock()
	client := b.client
	b.mu.RUnlock()

	if client == nil {
		return ErrAuthFailed
	}

	resp, err := client.GetLightsWithResponse(ctx)
	if err != nil {
		return err
	}

	if resp.JSON200 == nil || resp.JSON200.Data == nil {
		return nil
	}

	lights := make(map[string]hueclient.LightGet)
	for _, light := range *resp.JSON200.Data {
		if light.Id != nil {
			lights[*light.Id] = light
		}
	}

	b.state.UpdateLights(lights)
	return nil
}

// SyncRooms fetches only rooms from the bridge.
func (b *Bridge) SyncRooms(ctx context.Context) error {
	b.mu.RLock()
	client := b.client
	b.mu.RUnlock()

	if client == nil {
		return ErrAuthFailed
	}

	resp, err := client.GetRoomsWithResponse(ctx)
	if err != nil {
		return err
	}

	if resp.JSON200 == nil || resp.JSON200.Data == nil {
		return nil
	}

	rooms := make(map[string]hueclient.RoomGet)
	for _, room := range *resp.JSON200.Data {
		if room.Id != nil {
			rooms[*room.Id] = room
		}
	}

	b.state.UpdateRooms(rooms)
	return nil
}

// SyncZones fetches only zones from the bridge.
func (b *Bridge) SyncZones(ctx context.Context) error {
	b.mu.RLock()
	client := b.client
	b.mu.RUnlock()

	if client == nil {
		return ErrAuthFailed
	}

	resp, err := client.GetZonesWithResponse(ctx)
	if err != nil {
		return err
	}

	if resp.JSON200 == nil || resp.JSON200.Data == nil {
		return nil
	}

	zones := make(map[string]hueclient.RoomGet)
	for _, zone := range *resp.JSON200.Data {
		if zone.Id != nil {
			zones[*zone.Id] = zone
		}
	}

	b.state.UpdateZones(zones)
	return nil
}

// SyncGroupedLights fetches only grouped lights from the bridge.
func (b *Bridge) SyncGroupedLights(ctx context.Context) error {
	b.mu.RLock()
	client := b.client
	b.mu.RUnlock()

	if client == nil {
		return ErrAuthFailed
	}

	resp, err := client.GetGroupedLightsWithResponse(ctx)
	if err != nil {
		return err
	}

	if resp.JSON200 == nil || resp.JSON200.Data == nil {
		return nil
	}

	grouped := make(map[string]hueclient.GroupedLightGet)
	for _, gl := range *resp.JSON200.Data {
		if gl.Id != nil {
			grouped[*gl.Id] = gl
		}
	}

	b.state.UpdateGroupedLights(grouped)
	return nil
}

// SyncScenes fetches only scenes from the bridge.
func (b *Bridge) SyncScenes(ctx context.Context) error {
	b.mu.RLock()
	client := b.client
	b.mu.RUnlock()

	if client == nil {
		return ErrAuthFailed
	}

	resp, err := client.GetScenesWithResponse(ctx)
	if err != nil {
		return err
	}

	if resp.JSON200 == nil || resp.JSON200.Data == nil {
		return nil
	}

	scenes := make(map[string]hueclient.SceneGet)
	for _, scene := range *resp.JSON200.Data {
		if scene.Id != nil {
			scenes[*scene.Id] = scene
		}
	}

	b.state.UpdateScenes(scenes)
	return nil
}

// SyncDevices fetches only devices from the bridge.
func (b *Bridge) SyncDevices(ctx context.Context) error {
	b.mu.RLock()
	client := b.client
	b.mu.RUnlock()

	if client == nil {
		return ErrAuthFailed
	}

	resp, err := client.GetDevicesWithResponse(ctx)
	if err != nil {
		return err
	}

	if resp.JSON200 == nil || resp.JSON200.Data == nil {
		return nil
	}

	devices := make(map[string]hueclient.DeviceGet)
	for _, device := range *resp.JSON200.Data {
		if device.Id != nil {
			devices[*device.Id] = device
		}
	}

	b.state.UpdateDevices(devices)
	return nil
}

// SyncMotionSensors fetches only motion sensors from the bridge.
func (b *Bridge) SyncMotionSensors(ctx context.Context) error {
	b.mu.RLock()
	client := b.client
	b.mu.RUnlock()

	if client == nil {
		return ErrAuthFailed
	}

	// Sync operations don't need detailed logging - state sync covers it
	resp, err := client.GetMotionSensorsWithResponse(ctx)
	if err != nil {
		return err
	}

	if resp.JSON200 == nil || resp.JSON200.Data == nil {
		return nil
	}

	sensors := make(map[string]hueclient.MotionGet)
	for _, sensor := range *resp.JSON200.Data {
		if sensor.Id != nil {
			sensors[*sensor.Id] = sensor
		}
	}

	b.state.UpdateMotionSensors(sensors)
	return nil
}

// SyncTemperatures fetches only temperature sensors from the bridge.
func (b *Bridge) SyncTemperatures(ctx context.Context) error {
	b.mu.RLock()
	client := b.client
	b.mu.RUnlock()

	if client == nil {
		return ErrAuthFailed
	}

	// Sync operations don't need detailed logging - state sync covers it
	resp, err := client.GetTemperaturesWithResponse(ctx)
	if err != nil {
		return err
	}

	if resp.JSON200 == nil || resp.JSON200.Data == nil {
		return nil
	}

	temps := make(map[string]hueclient.TemperatureGet)
	for _, temp := range *resp.JSON200.Data {
		if temp.Id != nil {
			temps[*temp.Id] = temp
		}
	}

	b.state.UpdateTemperatures(temps)
	return nil
}

// SyncLightLevels fetches only light level sensors from the bridge.
func (b *Bridge) SyncLightLevels(ctx context.Context) error {
	b.mu.RLock()
	client := b.client
	b.mu.RUnlock()

	if client == nil {
		return ErrAuthFailed
	}

	// Sync operations don't need detailed logging - state sync covers it
	resp, err := client.GetLightLevelsWithResponse(ctx)
	if err != nil {
		return err
	}

	if resp.JSON200 == nil || resp.JSON200.Data == nil {
		return nil
	}

	levels := make(map[string]hueclient.LightLevelGet)
	for _, level := range *resp.JSON200.Data {
		if level.Id != nil {
			levels[*level.Id] = level
		}
	}

	b.state.UpdateLightLevels(levels)
	return nil
}

// SyncDevicePowers fetches only device power (battery) statuses from the bridge.
func (b *Bridge) SyncDevicePowers(ctx context.Context) error {
	b.mu.RLock()
	client := b.client
	b.mu.RUnlock()

	if client == nil {
		return ErrAuthFailed
	}

	// Sync operations don't need detailed logging - state sync covers it
	resp, err := client.GetDevicePowersWithResponse(ctx)
	if err != nil {
		return err
	}

	if resp.JSON200 == nil || resp.JSON200.Data == nil {
		return nil
	}

	powers := make(map[string]hueclient.DevicePowerGet)
	for _, power := range *resp.JSON200.Data {
		if power.Id != nil {
			powers[*power.Id] = power
		}
	}

	b.state.UpdateDevicePowers(powers)
	return nil
}

// SyncBridgeResource fetches the bridge resource.
func (b *Bridge) SyncBridgeResource(ctx context.Context) error {
	b.mu.RLock()
	client := b.client
	b.mu.RUnlock()

	if client == nil {
		return ErrAuthFailed
	}

	resp, err := client.GetBridgesWithResponse(ctx)
	if err != nil {
		return err
	}

	if resp.JSON200 == nil || resp.JSON200.Data == nil || len(*resp.JSON200.Data) == 0 {
		return nil
	}

	bridge := (*resp.JSON200.Data)[0]
	b.state.UpdateBridgeResource(&bridge)
	return nil
}

// SyncBridgeHome fetches the bridge home resource.
func (b *Bridge) SyncBridgeHome(ctx context.Context) error {
	b.mu.RLock()
	client := b.client
	b.mu.RUnlock()

	if client == nil {
		return ErrAuthFailed
	}

	resp, err := client.GetBridgeHomesWithResponse(ctx)
	if err != nil {
		return err
	}

	if resp.JSON200 == nil || resp.JSON200.Data == nil || len(*resp.JSON200.Data) == 0 {
		return nil
	}

	home := (*resp.JSON200.Data)[0]
	b.state.UpdateBridgeHome(&home)
	return nil
}

// SyncAuthApps fetches the authenticated applications list.
func (b *Bridge) SyncAuthApps(ctx context.Context) error {
	b.mu.RLock()
	extended := b.extended
	b.mu.RUnlock()

	if extended == nil {
		return ErrAuthFailed
	}

	// Sync operations don't need detailed logging - state sync covers it
	apps, err := extended.GetAuthenticatedApps(ctx)
	if err != nil {
		return err
	}

	b.state.UpdateAuthApps(apps)
	return nil
}

// SyncEntertainmentConfigurations fetches entertainment configuration data.
func (b *Bridge) SyncEntertainmentConfigurations(ctx context.Context) error {
	b.mu.RLock()
	extended := b.extended
	b.mu.RUnlock()

	if extended == nil {
		return ErrAuthFailed
	}

	// Sync operations don't need detailed logging - state sync covers it
	configs, err := extended.GetEntertainmentConfigurations(ctx)
	if err != nil {
		return err
	}

	b.state.UpdateEntertainmentConfigurations(configs)
	return nil
}

// SyncWifiConnectivity fetches WiFi connectivity status (Bridge Pro only).
func (b *Bridge) SyncWifiConnectivity(ctx context.Context) error {
	b.mu.RLock()
	extended := b.extended
	b.mu.RUnlock()

	if extended == nil {
		return ErrAuthFailed
	}

	// Sync operations don't need detailed logging - state sync covers it
	wifi, err := extended.GetWifiConnectivity(ctx)
	if err != nil {
		return err
	}

	b.state.UpdateWifiConnectivity(wifi)
	return nil
}

// SyncZigbeeConnectivity fetches Zigbee connectivity status for all devices.
func (b *Bridge) SyncZigbeeConnectivity(ctx context.Context) error {
	b.mu.RLock()
	extended := b.extended
	b.mu.RUnlock()

	if extended == nil {
		return ErrAuthFailed
	}

	// Sync operations don't need detailed logging - state sync covers it
	zigbee, err := extended.GetZigbeeConnectivity(ctx)
	if err != nil {
		return err
	}

	b.state.UpdateZigbeeConnectivity(zigbee)
	return nil
}
