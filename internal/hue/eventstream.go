package hue

import (
	"bufio"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/kluzzebass/lazyhue/internal/debug"
)

// EventType represents the type of SSE event from the bridge.
type EventType string

const (
	EventTypeUpdate EventType = "update"
	EventTypeAdd    EventType = "add"
	EventTypeDelete EventType = "delete"
	EventTypeError  EventType = "error"
)

// BridgeEvent represents a single event from the bridge event stream.
type BridgeEvent struct {
	ID           string          `json:"id"`
	CreationTime string          `json:"creationtime"`
	Type         EventType       `json:"type"`
	Data         json.RawMessage `json:"data"` // Keep raw for flexible parsing
}

// EventContainer wraps one or more events from the SSE stream.
type EventContainer struct {
	ID        string // SSE event ID (e.g., "1634576695:0")
	Timestamp time.Time
	Events    []BridgeEvent
}

// ResourceUpdate represents a partial update to a resource.
type ResourceUpdate struct {
	ID    string `json:"id"`
	Type  string `json:"type"`
	Owner *struct {
		Rid   string `json:"rid"`
		Rtype string `json:"rtype"`
	} `json:"owner,omitempty"`
	// Common fields that might be updated
	On *struct {
		On bool `json:"on"`
	} `json:"on,omitempty"`
	Dimming *struct {
		Brightness float64 `json:"brightness"`
	} `json:"dimming,omitempty"`
	Color *struct {
		XY struct {
			X float64 `json:"x"`
			Y float64 `json:"y"`
		} `json:"xy"`
	} `json:"color,omitempty"`
	ColorTemperature *struct {
		Mirek *int `json:"mirek,omitempty"`
	} `json:"color_temperature,omitempty"`
	Motion *struct {
		Motion bool `json:"motion"`
	} `json:"motion,omitempty"`
	Temperature *struct {
		Temperature float64 `json:"temperature"`
	} `json:"temperature,omitempty"`
	LightLevel *struct {
		LightLevel int `json:"light_level"`
	} `json:"light_level,omitempty"`
	// Scene status - can be a string or an object depending on resource type
	// Using json.RawMessage to handle both cases
	Status json.RawMessage `json:"status,omitempty"`
	// Metadata for device/room/zone/scene renames
	Metadata *struct {
		Name      *string `json:"name,omitempty"`
		Archetype *string `json:"archetype,omitempty"`
	} `json:"metadata,omitempty"`
	// Children for room/zone updates (devices/services)
	Children *[]struct {
		Rid   string `json:"rid"`
		Rtype string `json:"rtype"`
	} `json:"children,omitempty"`
	// Services for zone updates (lights)
	Services *[]struct {
		Rid   string `json:"rid"`
		Rtype string `json:"rtype"`
	} `json:"services,omitempty"`
	// Button event data
	Button *struct {
		LastEvent string `json:"last_event"` // "initial_press", "repeat", "short_release", "long_release", "double_short_release"
	} `json:"button,omitempty"`
	// Device power (battery) event data
	PowerState *struct {
		BatteryLevel *int    `json:"battery_level,omitempty"`
		BatteryState *string `json:"battery_state,omitempty"` // "normal", "low", "critical"
	} `json:"power_state,omitempty"`
	// Rotary dial event data
	RelativeRotary *struct {
		LastEvent *struct {
			Action   string `json:"action"`   // "start", "repeat"
			Rotation *struct {
				Direction string `json:"direction"` // "clock_wise", "counter_clock_wise"
				Steps     int    `json:"steps"`
				Duration  int    `json:"duration"`
			} `json:"rotation,omitempty"`
		} `json:"last_event,omitempty"`
	} `json:"relative_rotary,omitempty"`
	// Contact sensor event data
	ContactReport *struct {
		State string `json:"state"` // "contact", "no_contact"
	} `json:"contact_report,omitempty"`
	// Tamper sensor event data
	TamperReports *[]struct {
		State string `json:"state"` // "tampered", "not_tampered"
	} `json:"tamper_reports,omitempty"`
	// Zigbee connectivity data
	ConnectivityStatus *struct {
		Status string `json:"status"` // "connected", "disconnected", "connectivity_issue", "unidirectional_incoming"
	} `json:"zigbee_connectivity,omitempty"`
	// Software update data
	SoftwareUpdate *struct {
		State string `json:"state"` // "no_update", "update_available", "installing", etc.
	} `json:"software_update,omitempty"`
	// Geolocation data
	Geolocation *struct {
		IsConfigured bool `json:"is_configured"`
	} `json:"is_configured,omitempty"`
	// Smart scene state - can be a string or an object depending on resource type
	// Using json.RawMessage to handle both cases
	State json.RawMessage `json:"state,omitempty"`
}

// EventStream manages the SSE connection to a bridge.
type EventStream struct {
	bridgeIP string
	apiKey   string
	client   *http.Client

	// Callbacks
	onEvent      func(container EventContainer)
	onConnect    func()
	onDisconnect func(err error)

	// State
	connected    bool
	lastEventID  string
	reconnectMin time.Duration
	reconnectMax time.Duration
}

// NewEventStream creates a new event stream manager.
func NewEventStream(bridgeIP, apiKey string) *EventStream {
	return &EventStream{
		bridgeIP: bridgeIP,
		apiKey:   apiKey,
		client: &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			},
			Timeout: 0, // No timeout for SSE
		},
		reconnectMin: 1 * time.Second,
		reconnectMax: 30 * time.Second,
	}
}

// OnEvent sets the callback for when events are received.
func (es *EventStream) OnEvent(fn func(container EventContainer)) {
	es.onEvent = fn
}

// OnConnect sets the callback for when the stream connects.
func (es *EventStream) OnConnect(fn func()) {
	es.onConnect = fn
}

// OnDisconnect sets the callback for when the stream disconnects.
func (es *EventStream) OnDisconnect(fn func(err error)) {
	es.onDisconnect = fn
}

// IsConnected returns whether the event stream is connected.
func (es *EventStream) IsConnected() bool {
	return es.connected
}

// Start begins listening to the event stream. Blocks until context is cancelled.
// Automatically reconnects on disconnection with exponential backoff.
func (es *EventStream) Start(ctx context.Context) {
	reconnectDelay := es.reconnectMin

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		err := es.connect(ctx)
		if err != nil {
			debug.Log("EventStream disconnected: %v", err)
			if es.onDisconnect != nil {
				es.onDisconnect(err)
			}
		}

		es.connected = false

		// Check if we should stop
		select {
		case <-ctx.Done():
			return
		default:
		}

		// Exponential backoff for reconnection
		debug.Log("EventStream reconnecting in %v", reconnectDelay)
		select {
		case <-ctx.Done():
			return
		case <-time.After(reconnectDelay):
		}

		reconnectDelay *= 2
		if reconnectDelay > es.reconnectMax {
			reconnectDelay = es.reconnectMax
		}
	}
}

func (es *EventStream) connect(ctx context.Context) error {
	url := fmt.Sprintf("https://%s/eventstream/clip/v2", es.bridgeIP)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("hue-application-key", es.apiKey)
	req.Header.Set("Accept", "text/event-stream")

	// Resume from last event if we have one
	if es.lastEventID != "" {
		req.Header.Set("Last-Event-ID", es.lastEventID)
	}

	resp, err := es.client.Do(req)
	if err != nil {
		return fmt.Errorf("connect: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	es.connected = true
	debug.Log("EventStream connected to %s", es.bridgeIP)

	if es.onConnect != nil {
		es.onConnect()
	}

	// Read SSE stream
	scanner := bufio.NewScanner(resp.Body)
	var currentID string
	var currentData strings.Builder

	for scanner.Scan() {
		line := scanner.Text()

		if strings.HasPrefix(line, "id: ") {
			currentID = strings.TrimPrefix(line, "id: ")
		} else if strings.HasPrefix(line, "data: ") {
			currentData.WriteString(strings.TrimPrefix(line, "data: "))
		} else if line == "" && currentData.Len() > 0 {
			// Empty line = end of event
			es.processEvent(currentID, currentData.String())
			es.lastEventID = currentID
			currentData.Reset()
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("read stream: %w", err)
	}

	return nil
}

func (es *EventStream) processEvent(id, data string) {

	// Parse the event container (it's an array of events)
	var events []BridgeEvent
	if err := json.Unmarshal([]byte(data), &events); err != nil {
		debug.Log("EventStream parse error: %v", err)
		return
	}

	container := EventContainer{
		ID:        id,
		Timestamp: time.Now(),
		Events:    events,
	}

	if es.onEvent != nil {
		es.onEvent(container)
	}
}

// ParseResourceUpdates extracts resource updates from an event.
func ParseResourceUpdates(event BridgeEvent) ([]ResourceUpdate, error) {
	var updates []ResourceUpdate
	if err := json.Unmarshal(event.Data, &updates); err != nil {
		return nil, err
	}
	return updates, nil
}
