package hue

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/openhue/openhue-go"
)

// ExtendedClient wraps the openhue client to add methods not exposed by Home.
type ExtendedClient struct {
	api      *openhue.ClientWithResponses
	bridgeIP string
	apiKey   string
}

// NewExtendedClient creates a client for extended API access.
func NewExtendedClient(bridgeIP, apiKey string) (*ExtendedClient, error) {
	authFn := func(ctx context.Context, req *http.Request) error {
		req.Header.Set("hue-application-key", apiKey)
		return nil
	}

	// Skip SSL verification (Hue bridge uses self-signed certs)
	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}

	client, err := openhue.NewClientWithResponses("https://"+bridgeIP, openhue.WithRequestEditorFn(authFn))
	if err != nil {
		return nil, err
	}

	return &ExtendedClient{
		api:      client,
		bridgeIP: bridgeIP,
		apiKey:   apiKey,
	}, nil
}

// GetZones fetches all zones from the bridge.
// Zones use the same RoomGet type as rooms.
func (c *ExtendedClient) GetZones(ctx context.Context) (map[string]openhue.RoomGet, error) {
	resp, err := c.api.GetZonesWithResponse(ctx)
	if err != nil {
		return nil, err
	}

	if resp.HTTPResponse.StatusCode != http.StatusOK {
		return nil, ErrAuthFailed
	}

	zones := make(map[string]openhue.RoomGet)
	if resp.JSON200 != nil && resp.JSON200.Data != nil {
		for _, zone := range *resp.JSON200.Data {
			if zone.Id != nil {
				zones[*zone.Id] = zone
			}
		}
	}

	return zones, nil
}

// GetMotionSensors fetches all motion sensors from the bridge.
func (c *ExtendedClient) GetMotionSensors(ctx context.Context) (map[string]openhue.MotionGet, error) {
	resp, err := c.api.GetMotionSensorsWithResponse(ctx)
	if err != nil {
		return nil, err
	}

	if resp.HTTPResponse.StatusCode != http.StatusOK {
		return nil, ErrAuthFailed
	}

	sensors := make(map[string]openhue.MotionGet)
	if resp.JSON200 != nil && resp.JSON200.Data != nil {
		for _, sensor := range *resp.JSON200.Data {
			if sensor.Id != nil {
				sensors[*sensor.Id] = sensor
			}
		}
	}

	return sensors, nil
}

// GetTemperatures fetches all temperature sensors from the bridge.
func (c *ExtendedClient) GetTemperatures(ctx context.Context) (map[string]openhue.TemperatureGet, error) {
	resp, err := c.api.GetTemperaturesWithResponse(ctx)
	if err != nil {
		return nil, err
	}

	if resp.HTTPResponse.StatusCode != http.StatusOK {
		return nil, ErrAuthFailed
	}

	temps := make(map[string]openhue.TemperatureGet)
	if resp.JSON200 != nil && resp.JSON200.Data != nil {
		for _, temp := range *resp.JSON200.Data {
			if temp.Id != nil {
				temps[*temp.Id] = temp
			}
		}
	}

	return temps, nil
}

// GetLightLevels fetches all light level sensors from the bridge.
func (c *ExtendedClient) GetLightLevels(ctx context.Context) (map[string]openhue.LightLevelGet, error) {
	resp, err := c.api.GetLightLevelsWithResponse(ctx)
	if err != nil {
		return nil, err
	}

	if resp.HTTPResponse.StatusCode != http.StatusOK {
		return nil, ErrAuthFailed
	}

	levels := make(map[string]openhue.LightLevelGet)
	if resp.JSON200 != nil && resp.JSON200.Data != nil {
		for _, level := range *resp.JSON200.Data {
			if level.Id != nil {
				levels[*level.Id] = level
			}
		}
	}

	return levels, nil
}

// GetDevicePowers fetches all device power (battery) statuses from the bridge.
func (c *ExtendedClient) GetDevicePowers(ctx context.Context) (map[string]openhue.DevicePowerGet, error) {
	resp, err := c.api.GetDevicePowersWithResponse(ctx)
	if err != nil {
		return nil, err
	}

	if resp.HTTPResponse.StatusCode != http.StatusOK {
		return nil, ErrAuthFailed
	}

	powers := make(map[string]openhue.DevicePowerGet)
	if resp.JSON200 != nil && resp.JSON200.Data != nil {
		for _, power := range *resp.JSON200.Data {
			if power.Id != nil {
				powers[*power.Id] = power
			}
		}
	}

	return powers, nil
}

// GetBridges fetches all bridge resources from the bridge.
func (c *ExtendedClient) GetBridges(ctx context.Context) ([]openhue.BridgeGet, error) {
	resp, err := c.api.GetBridgesWithResponse(ctx)
	if err != nil {
		return nil, err
	}

	if resp.HTTPResponse.StatusCode != http.StatusOK {
		return nil, ErrAuthFailed
	}

	var bridges []openhue.BridgeGet
	if resp.JSON200 != nil && resp.JSON200.Data != nil {
		bridges = *resp.JSON200.Data
	}

	return bridges, nil
}

// GetBridgeHome fetches the bridge home resource.
func (c *ExtendedClient) GetBridgeHome(ctx context.Context) (*openhue.BridgeHomeGet, error) {
	resp, err := c.api.GetBridgeHomesWithResponse(ctx)
	if err != nil {
		return nil, err
	}

	if resp.HTTPResponse.StatusCode != http.StatusOK {
		return nil, ErrAuthFailed
	}

	if resp.JSON200 != nil && resp.JSON200.Data != nil && len(*resp.JSON200.Data) > 0 {
		return &(*resp.JSON200.Data)[0], nil
	}

	return nil, nil
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
	Name           string `json:"name"`
	CreateDate     string `json:"create date"`
	LastUseDate    string `json:"last use date"`
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

	resp, err := http.DefaultClient.Do(req)
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

// Note: Entertainment configurations are not yet supported by openhue-go v0.4.0.
// The API endpoints exist but the types are not implemented.

