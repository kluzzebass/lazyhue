package hue

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// ExtendedClient provides access to Hue APIs not covered by the generated client.
// This includes V1 API endpoints and resources not in the OpenAPI spec.
type ExtendedClient struct {
	bridgeIP string
	apiKey   string
	client   *http.Client
}

// NewExtendedClient creates a client for extended API access.
func NewExtendedClient(bridgeIP, apiKey string) (*ExtendedClient, error) {
	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}

	return &ExtendedClient{
		bridgeIP: bridgeIP,
		apiKey:   apiKey,
		client:   client,
	}, nil
}

// doRequest makes an authenticated request to the bridge.
func (c *ExtendedClient) doRequest(ctx context.Context, method, path string) ([]byte, error) {
	url := fmt.Sprintf("https://%s%s", c.bridgeIP, path)
	req, err := http.NewRequestWithContext(ctx, method, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("hue-application-key", c.apiKey)

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, nil // Return nil for non-OK responses (e.g., 404 for unsupported features)
	}

	return io.ReadAll(resp.Body)
}

// AuthV1Entry represents an authenticated application from the V1 API whitelist.
type AuthV1Entry struct {
	Username    string // The API key/username
	AppName     string // Application name (from "name" field, format: "app#device")
	CreateDate  string // When the app was registered
	LastUseDate string // When the app was last used
}

// whitelistEntry represents a single entry in the V1 config whitelist.
type whitelistEntry struct {
	Name        string `json:"name"`
	CreateDate  string `json:"create date"`
	LastUseDate string `json:"last use date"`
}

// v1ConfigResponse is the relevant part of the V1 /config response.
type v1ConfigResponse struct {
	Whitelist map[string]whitelistEntry `json:"whitelist"`
}

// GetAuthenticatedApps fetches all authenticated applications via the V1 API config endpoint.
func (c *ExtendedClient) GetAuthenticatedApps(ctx context.Context) ([]AuthV1Entry, error) {
	// Make a direct V1 API call to /api/<username>/config
	url := fmt.Sprintf("https://%s/api/%s/config", c.bridgeIP, c.apiKey)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, ErrAuthFailed
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var config v1ConfigResponse
	if err := json.Unmarshal(body, &config); err != nil {
		return nil, err
	}

	var apps []AuthV1Entry
	for username, entry := range config.Whitelist {
		apps = append(apps, AuthV1Entry{
			Username:    username,
			AppName:     entry.Name,
			CreateDate:  entry.CreateDate,
			LastUseDate: entry.LastUseDate,
		})
	}

	return apps, nil
}

// EntertainmentConfiguration represents an entertainment area configuration.
type EntertainmentConfiguration struct {
	ID                string                    `json:"id"`
	Type              string                    `json:"type"`
	Metadata          *EntertainmentMetadata    `json:"metadata,omitempty"`
	Name              string                    `json:"name,omitempty"`           // Some API versions use this
	Status            string                    `json:"status,omitempty"`         // inactive, active, streaming
	ConfigurationType string                    `json:"configuration_type,omitempty"` // screen, monitor, music, 3dspace, other
	Channels          []EntertainmentChannel    `json:"channels,omitempty"`
	Lights            []EntertainmentLightEntry `json:"light_services,omitempty"`
	Locations         *EntertainmentLocations   `json:"locations,omitempty"`
	StreamProxy       *StreamProxy              `json:"stream_proxy,omitempty"`
	ActiveStreamer    *ResourceIdentifier       `json:"active_streamer,omitempty"`
}

// EntertainmentMetadata holds metadata for an entertainment configuration.
type EntertainmentMetadata struct {
	Name string `json:"name"`
}

// EntertainmentLocations contains service locations for entertainment.
type EntertainmentLocations struct {
	ServiceLocations []ServiceLocation `json:"service_locations,omitempty"`
}

// ServiceLocation represents a light's position in the entertainment area.
type ServiceLocation struct {
	Service   *ResourceIdentifier     `json:"service,omitempty"`
	Position  *EntertainmentPosition  `json:"position,omitempty"`
	Positions []EntertainmentPosition `json:"positions,omitempty"` // Some lights have multiple positions
}

// ResourceIdentifier is a reference to a resource.
type ResourceIdentifier struct {
	RID   string `json:"rid"`
	RType string `json:"rtype"`
}

// StreamProxy contains streaming proxy information.
type StreamProxy struct {
	Mode string              `json:"mode,omitempty"` // auto, manual
	Node *ResourceIdentifier `json:"node,omitempty"`
}

// EntertainmentChannel represents a channel in an entertainment configuration.
type EntertainmentChannel struct {
	ChannelID uint8                     `json:"channel_id"`
	Position  *EntertainmentPosition    `json:"position,omitempty"`
	Members   []EntertainmentLightEntry `json:"members,omitempty"`
}

// EntertainmentPosition represents a 3D position.
type EntertainmentPosition struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
	Z float64 `json:"z"`
}

// EntertainmentLightEntry represents a light in an entertainment configuration.
type EntertainmentLightEntry struct {
	Service *struct {
		RID   string `json:"rid"`
		RType string `json:"rtype"`
	} `json:"service,omitempty"`
}

// GetEntertainmentConfigurations fetches all entertainment configurations.
func (c *ExtendedClient) GetEntertainmentConfigurations(ctx context.Context) (map[string]EntertainmentConfiguration, error) {
	body, err := c.doRequest(ctx, http.MethodGet, "/clip/v2/resource/entertainment_configuration")
	if err != nil {
		return nil, err
	}
	if body == nil {
		return nil, nil
	}

	var result struct {
		Data []EntertainmentConfiguration `json:"data"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	configs := make(map[string]EntertainmentConfiguration)
	for _, cfg := range result.Data {
		configs[cfg.ID] = cfg
	}
	return configs, nil
}

// WifiConnectivity represents a WiFi connectivity resource (Bridge Pro).
type WifiConnectivity struct {
	ID      string `json:"id"`
	IDV1    string `json:"id_v1,omitempty"`
	Status  string `json:"status"`  // "connected" or "disconnected"
	Type    string `json:"type"`    // "wifi_connectivity"
	HasSSID bool   `json:"has_ssid"`
}

// GetWifiConnectivity fetches WiFi connectivity status from the bridge.
// This is only available on Bridge Pro models with WiFi support.
func (c *ExtendedClient) GetWifiConnectivity(ctx context.Context) ([]WifiConnectivity, error) {
	body, err := c.doRequest(ctx, http.MethodGet, "/clip/v2/resource/wifi_connectivity")
	if err != nil {
		return nil, err
	}
	if body == nil {
		return nil, nil
	}

	var result struct {
		Data []WifiConnectivity `json:"data"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	return result.Data, nil
}

// ZigbeeConnectivity represents a Zigbee connectivity resource.
type ZigbeeConnectivity struct {
	ID            string `json:"id"`
	IDV1          string `json:"id_v1,omitempty"`
	Type          string `json:"type"`   // "zigbee_connectivity"
	Status        string `json:"status"` // "connected", "disconnected", "connectivity_issue", "unidirectional_incoming"
	MacAddress    string `json:"mac_address,omitempty"`
	ExtendedPanID string `json:"extended_pan_id,omitempty"`
	Channel       *struct {
		Value  string `json:"value,omitempty"`  // e.g., "channel_25"
		Status string `json:"status,omitempty"` // "set", "changing"
	} `json:"channel,omitempty"`
	Owner *struct {
		Rid   string `json:"rid,omitempty"`
		Rtype string `json:"rtype,omitempty"`
	} `json:"owner,omitempty"`
}

// GetZigbeeConnectivity fetches all Zigbee connectivity resources from the bridge.
func (c *ExtendedClient) GetZigbeeConnectivity(ctx context.Context) (map[string]ZigbeeConnectivity, error) {
	body, err := c.doRequest(ctx, http.MethodGet, "/clip/v2/resource/zigbee_connectivity")
	if err != nil {
		return nil, err
	}
	if body == nil {
		return nil, nil
	}

	var result struct {
		Data []ZigbeeConnectivity `json:"data"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	zigbees := make(map[string]ZigbeeConnectivity)
	for _, zc := range result.Data {
		zigbees[zc.ID] = zc
	}
	return zigbees, nil
}

// GetZigbeeConnectivityByID fetches a specific Zigbee connectivity resource.
func (c *ExtendedClient) GetZigbeeConnectivityByID(ctx context.Context, id string) (*ZigbeeConnectivity, error) {
	body, err := c.doRequest(ctx, http.MethodGet, "/clip/v2/resource/zigbee_connectivity/"+id)
	if err != nil {
		return nil, err
	}
	if body == nil {
		return nil, nil
	}

	var result struct {
		Data []ZigbeeConnectivity `json:"data"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	if len(result.Data) == 0 {
		return nil, nil
	}
	return &result.Data[0], nil
}
