package hue

import (
	"context"
	"crypto/tls"
	"net/http"

	"github.com/openhue/openhue-go"
)

// ExtendedClient wraps the openhue client to add methods not exposed by Home.
type ExtendedClient struct {
	api *openhue.ClientWithResponses
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

	return &ExtendedClient{api: client}, nil
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

// Note: Entertainment configurations are not yet supported by openhue-go v0.4.0.
// The API endpoints exist but the types are not implemented.

