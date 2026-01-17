package hue

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"time"

	"github.com/kluzzebass/lazyhue/internal/hueclient"
)

// ToggleLight toggles a light on or off.
func (b *Bridge) ToggleLight(lightID string) error {
	b.mu.RLock()
	client := b.client
	b.mu.RUnlock()

	if client == nil {
		return ErrAuthFailed
	}

	light, ok := b.state.GetLight(lightID)
	if !ok {
		return ErrAuthFailed
	}

	newState := !isLightOn(light)

	// Optimistic update: apply to cache immediately
	b.state.SetLightOn(lightID, newState)

	lightName := "Unknown"
	if light, ok := b.state.GetLight(lightID); ok {
		lightName = b.state.GetLightName(light)
	}
	action := "on"
	if !newState {
		action = "off"
	}
	b.logRequest(fmt.Sprintf("%s: %s", lightName, action))
	httpResp, err := client.UpdateLightWithResponse(context.Background(), toResourceId(lightID), hueclient.UpdateLightJSONRequestBody{
		On: &struct {
			On *bool `json:"on,omitempty"`
		}{On: &newState},
	})
	if err != nil {
		b.logError(fmt.Sprintf("%s: %s failed: %v", lightName, action, err))
		return err
	}
	if httpResp != nil && httpResp.HTTPResponse != nil && httpResp.HTTPResponse.StatusCode >= 400 {
		errMsg := fmt.Sprintf("%s: %s failed: HTTP %d", lightName, action, httpResp.HTTPResponse.StatusCode)
		b.logError(errMsg)
		return errors.New(errMsg)
	}
	return nil
}

// SetLightOn turns a light on or off.
func (b *Bridge) SetLightOn(lightID string, on bool) error {
	b.mu.RLock()
	client := b.client
	b.mu.RUnlock()

	if client == nil {
		return ErrAuthFailed
	}

	// Optimistic update: apply to cache immediately
	b.state.SetLightOn(lightID, on)

	// Get light name for logging - try multiple approaches
	lightName := "Unknown"
	if light, ok := b.state.GetLight(lightID); ok {
		lightName = b.state.GetLightName(light)
		if lightName == "Unknown" {
			// If GetLightName returned "Unknown", try to provide a more descriptive fallback
			lightName = fmt.Sprintf("Light %s", lightID)
		}
	} else {
		// Fallback: try to find light by searching all lights
		allLights := b.state.AllLights()
		for _, l := range allLights {
			if l.Id == lightID {
				lightName = b.state.GetLightName(l)
				if lightName == "Unknown" {
					lightName = fmt.Sprintf("Light %s", lightID)
				}
				break
			}
		}
		// If still unknown, use a descriptive format instead of raw UUID
		if lightName == "Unknown" {
			lightName = fmt.Sprintf("Light %s", lightID)
		}
	}
	action := "on"
	if !on {
		action = "off"
	}
	b.logRequest(fmt.Sprintf("%s: %s", lightName, action))
	httpResp, err := client.UpdateLightWithResponse(context.Background(), toResourceId(lightID), hueclient.UpdateLightJSONRequestBody{
		On: &struct {
			On *bool `json:"on,omitempty"`
		}{On: &on},
	})
	if err != nil {
		b.logError(fmt.Sprintf("%s: %s failed: %v", lightName, action, err))
		return err
	}
	if httpResp != nil && httpResp.HTTPResponse != nil && httpResp.HTTPResponse.StatusCode >= 400 {
		errMsg := fmt.Sprintf("%s: %s failed: HTTP %d", lightName, action, httpResp.HTTPResponse.StatusCode)
		b.logError(errMsg)
		return errors.New(errMsg)
	}
	return nil
}

// SetLightBrightness sets a light's brightness (0-100).
// Rapid calls are debounced - state is updated immediately, but API call is delayed.
func (b *Bridge) SetLightBrightness(lightID string, brightness float64) error {
	b.mu.RLock()
	client := b.client
	b.mu.RUnlock()

	if client == nil {
		return ErrAuthFailed
	}

	// Optimistic update: apply to cache immediately so rapid keypresses accumulate
	b.state.SetLightBrightness(lightID, brightness)

	// Debounce the actual API call
	b.debounceMu.Lock()
	if timer, ok := b.brightnessDebounce[lightID]; ok {
		timer.Stop()
	}

	lightName := "Unknown"
	if light, ok := b.state.GetLight(lightID); ok {
		lightName = b.state.GetLightName(light)
	}

	br := float32(brightness)
	b.brightnessDebounce[lightID] = time.AfterFunc(50*time.Millisecond, func() {
		b.mu.RLock()
		client := b.client
		b.mu.RUnlock()

		if client != nil {
			b.logRequest(fmt.Sprintf("%s: brightness %.0f%%", lightName, brightness))
			httpResp, err := client.UpdateLightWithResponse(context.Background(), toResourceId(lightID), hueclient.UpdateLightJSONRequestBody{
				Dimming: &struct {
					Brightness *float32 `json:"brightness,omitempty"`
				}{Brightness: &br},
			})
			if err != nil {
				b.logError(fmt.Sprintf("%s: brightness failed: %v", lightName, err))
			} else if httpResp != nil && httpResp.HTTPResponse != nil && httpResp.HTTPResponse.StatusCode >= 400 {
				b.logError(fmt.Sprintf("%s: brightness failed: HTTP %d", lightName, httpResp.HTTPResponse.StatusCode))
			}
		}

		// Clean up timer
		b.debounceMu.Lock()
		delete(b.brightnessDebounce, lightID)
		b.debounceMu.Unlock()
	})
	b.debounceMu.Unlock()

	return nil
}

// SetLightColor sets a light's color using CIE XY coordinates.
// Rapid calls are debounced - state is updated immediately, but API call is delayed.
func (b *Bridge) SetLightColor(lightID string, x, y float64) error {
	b.mu.RLock()
	client := b.client
	b.mu.RUnlock()

	if client == nil {
		return ErrAuthFailed
	}

	// Optimistic update
	b.state.SetLightColor(lightID, x, y)

	// Debounce the actual API call
	b.debounceMu.Lock()
	if timer, ok := b.colorDebounce[lightID]; ok {
		timer.Stop()
	}

	lightName := "Unknown"
	if light, ok := b.state.GetLight(lightID); ok {
		lightName = b.state.GetLightName(light)
	}

	xf, yf := float32(x), float32(y)
	b.colorDebounce[lightID] = time.AfterFunc(50*time.Millisecond, func() {
		b.mu.RLock()
		client := b.client
		b.mu.RUnlock()

		if client != nil {
			b.logRequest(fmt.Sprintf("%s: color", lightName))
			httpResp, err := client.UpdateLightWithResponse(context.Background(), toResourceId(lightID), hueclient.UpdateLightJSONRequestBody{
				Color: &struct {
					Xy *struct {
						X float32 `json:"x"`
						Y float32 `json:"y"`
					} `json:"xy,omitempty"`
				}{
					Xy: &struct {
						X float32 `json:"x"`
						Y float32 `json:"y"`
					}{X: xf, Y: yf},
				},
			})
			if err != nil {
				b.logError(fmt.Sprintf("%s: color failed: %v", lightName, err))
			} else if httpResp != nil && httpResp.HTTPResponse != nil && httpResp.HTTPResponse.StatusCode >= 400 {
				b.logError(fmt.Sprintf("%s: color failed: HTTP %d", lightName, httpResp.HTTPResponse.StatusCode))
			}
		}

		// Clean up timer
		b.debounceMu.Lock()
		delete(b.colorDebounce, lightID)
		b.debounceMu.Unlock()
	})
	b.debounceMu.Unlock()

	return nil
}

// SetLightColorTemperature sets a light's color temperature in mirek (153-500).
// Rapid calls are debounced - state is updated immediately, but API call is delayed.
func (b *Bridge) SetLightColorTemperature(lightID string, mirek int) error {
	b.mu.RLock()
	client := b.client
	b.mu.RUnlock()

	if client == nil {
		return ErrAuthFailed
	}

	// Optimistic update
	b.state.SetLightColorTemperature(lightID, mirek)

	// Debounce the actual API call
	b.debounceMu.Lock()
	if timer, ok := b.colorTempDebounce[lightID]; ok {
		timer.Stop()
	}

	lightName := "Unknown"
	if light, ok := b.state.GetLight(lightID); ok {
		lightName = b.state.GetLightName(light)
	}

	b.colorTempDebounce[lightID] = time.AfterFunc(50*time.Millisecond, func() {
		b.mu.RLock()
		client := b.client
		b.mu.RUnlock()

		if client != nil {
			b.logRequest(fmt.Sprintf("%s: color temp %d mirek", lightName, mirek))
			httpResp, err := client.UpdateLightWithResponse(context.Background(), toResourceId(lightID), hueclient.UpdateLightJSONRequestBody{
				ColorTemperature: &struct {
					Mirek *int `json:"mirek,omitempty"`
				}{Mirek: &mirek},
			})
			if err != nil {
				b.logError(fmt.Sprintf("%s: color temp failed: %v", lightName, err))
			} else if httpResp != nil && httpResp.HTTPResponse != nil && httpResp.HTTPResponse.StatusCode >= 400 {
				b.logError(fmt.Sprintf("%s: color temp failed: HTTP %d", lightName, httpResp.HTTPResponse.StatusCode))
			}
		}

		// Clean up timer
		b.debounceMu.Lock()
		delete(b.colorTempDebounce, lightID)
		b.debounceMu.Unlock()
	})
	b.debounceMu.Unlock()

	return nil
}

// SetLightEffect sets a light's effect (candle, fire, prism, etc.).
func (b *Bridge) SetLightEffect(lightID string, effect hueclient.Effect) error {
	b.mu.RLock()
	client := b.client
	b.mu.RUnlock()

	if client == nil {
		return ErrAuthFailed
	}

	// Optimistic update - convert Effect to LightGetEffectsStatus for state cache
	b.state.SetLightEffect(lightID, hueclient.Effect(effect))

	lightName := "Unknown"
	if light, ok := b.state.GetLight(lightID); ok {
		lightName = b.state.GetLightName(light)
	}
	effectName := EffectDisplayName(string(effect))
	b.logRequest(fmt.Sprintf("%s: effect %s", lightName, effectName))
	effectVal := hueclient.Effect(effect)
	httpResp, err := client.UpdateLightWithResponse(context.Background(), toResourceId(lightID), hueclient.UpdateLightJSONRequestBody{
		Effects: &struct {
			Effect *hueclient.Effect `json:"effect,omitempty"`
		}{Effect: &effectVal},
	})
	if err != nil {
		b.logError(fmt.Sprintf("%s: effect %s failed: %v", lightName, effectName, err))
		return err
	}
	if httpResp != nil && httpResp.HTTPResponse != nil && httpResp.HTTPResponse.StatusCode >= 400 {
		errMsg := fmt.Sprintf("%s: effect %s failed: HTTP %d", lightName, effectName, httpResp.HTTPResponse.StatusCode)
		b.logError(errMsg)
		return errors.New(errMsg)
	}
	return nil
}

// SetLightEffectSpeed sets a light's effect speed (0.0 to 1.0).
// This uses EffectsV2 API and requires the current effect to be specified.
func (b *Bridge) SetLightEffectSpeed(lightID string, speed float32) error {
	b.mu.RLock()
	client := b.client
	b.mu.RUnlock()

	if client == nil {
		return ErrAuthFailed
	}

	// Get the current effect from the light
	light, ok := b.state.GetLight(lightID)
	if !ok {
		return ErrAuthFailed
	}

	// Need to know the current effect to set speed
	var currentEffect hueclient.Effect
	if light.EffectsV2 != nil {
		currentEffect = light.EffectsV2.Status.Effect
	} else if light.Effects != nil {
		currentEffect = hueclient.Effect(light.Effects.Status)
	}

	if currentEffect == "" || currentEffect == "no_effect" {
		return errors.New("no active effect to set speed for")
	}

	lightName := "Unknown"
	if l, ok := b.state.GetLight(lightID); ok {
		lightName = b.state.GetLightName(l)
	}

	b.logRequest(fmt.Sprintf("%s: effect speed %.0f%%", lightName, speed*100))
	_, err := client.UpdateLightWithResponse(context.Background(), toResourceId(lightID), hueclient.UpdateLightJSONRequestBody{
		EffectsV2: &struct {
			Action *struct {
				Effect     hueclient.Effect `json:"effect"`
				Parameters *struct {
					Color *struct {
						Xy *struct {
							X float32 `json:"x"`
							Y float32 `json:"y"`
						} `json:"xy,omitempty"`
					} `json:"color,omitempty"`
					ColorTemperature *struct {
						Mirek *int `json:"mirek,omitempty"`
					} `json:"color_temperature,omitempty"`
					Speed *float32 `json:"speed,omitempty"`
				} `json:"parameters,omitempty"`
			} `json:"action,omitempty"`
		}{
			Action: &struct {
				Effect     hueclient.Effect `json:"effect"`
				Parameters *struct {
					Color *struct {
						Xy *struct {
							X float32 `json:"x"`
							Y float32 `json:"y"`
						} `json:"xy,omitempty"`
					} `json:"color,omitempty"`
					ColorTemperature *struct {
						Mirek *int `json:"mirek,omitempty"`
					} `json:"color_temperature,omitempty"`
					Speed *float32 `json:"speed,omitempty"`
				} `json:"parameters,omitempty"`
			}{
				Effect: currentEffect,
				Parameters: &struct {
					Color *struct {
						Xy *struct {
							X float32 `json:"x"`
							Y float32 `json:"y"`
						} `json:"xy,omitempty"`
					} `json:"color,omitempty"`
					ColorTemperature *struct {
						Mirek *int `json:"mirek,omitempty"`
					} `json:"color_temperature,omitempty"`
					Speed *float32 `json:"speed,omitempty"`
				}{
					Speed: &speed,
				},
			},
		},
	})
	return err
}

// gradientModePutBody is the request body for updating gradient mode (requires points too).
type gradientModePutBody struct {
	Gradient *gradientModePut `json:"gradient,omitempty"`
}

type gradientModePut struct {
	Mode   *hueclient.LightGetGradientMode `json:"mode,omitempty"`
	Points []gradientPointPut               `json:"points,omitempty"`
}

// SetLightGradientMode sets a light's gradient mode.
// The API requires points to be included when changing mode.
func (b *Bridge) SetLightGradientMode(lightID string, mode hueclient.LightGetGradientMode) error {
	b.mu.RLock()
	client := b.client
	b.mu.RUnlock()

	if client == nil {
		return ErrAuthFailed
	}

	// Get current points from state
	light, ok := b.state.GetLight(lightID)
	if !ok {
		return errors.New("light not found")
	}

	lightName := b.state.GetLightName(light)

	// Build wrapped points from current state
	var wrappedPoints []gradientPointPut
	if light.Gradient != nil {
		for _, pt := range light.Gradient.Points {
			wrappedPoints = append(wrappedPoints, gradientPointPut{Color: &pt.Color})
		}
	}

	// Optimistic update
	b.state.SetLightGradientMode(lightID, mode)

	b.logRequest(fmt.Sprintf("%s: gradient mode %s", lightName, mode))

	// Build request body with mode and current points
	reqBody := gradientModePutBody{
		Gradient: &gradientModePut{
			Mode:   &mode,
			Points: wrappedPoints,
		},
	}
	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		b.logError(fmt.Sprintf("%s: gradient mode failed: %v", lightName, err))
		return err
	}

	slog.Debug("gradient mode request", "lightID", lightID, "body", string(bodyBytes))

	resp, err := client.UpdateLightWithBody(context.Background(), toResourceId(lightID), "application/json", bytes.NewReader(bodyBytes))
	if err != nil {
		b.logError(fmt.Sprintf("%s: gradient mode failed: %v", lightName, err))
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(resp.Body)
		slog.Debug("gradient mode error", "lightID", lightID, "response", string(respBody))
		errMsg := fmt.Sprintf("%s: gradient mode failed: HTTP %d", lightName, resp.StatusCode)
		b.logError(errMsg)
		return errors.New(errMsg)
	}
	return nil
}

// gradientPointPut wraps a Color in a "color" field for the PUT API.
// The Hue API expects: { "color": { "xy": { "x": ..., "y": ... } } }
type gradientPointPut struct {
	Color *hueclient.ActionGetActionColor `json:"color,omitempty"`
}

// gradientPut is a custom gradient structure for PUT requests.
type gradientPut struct {
	Points []gradientPointPut `json:"points,omitempty"`
}

// lightGradientPutBody is the request body for updating gradient points.
type lightGradientPutBody struct {
	Gradient *gradientPut `json:"gradient,omitempty"`
}

// SetLightGradientPoints sets a light's gradient points (array of XY colors).
// Rapid calls are debounced - state is updated immediately, but API call is delayed.
func (b *Bridge) SetLightGradientPoints(lightID string, points []hueclient.ActionGetActionColor) error {
	b.mu.RLock()
	client := b.client
	b.mu.RUnlock()

	if client == nil {
		return ErrAuthFailed
	}

	// Optimistic update - convert ActionGetActionColor to GradientPointGet for state cache
	gradientPoints := make([]hueclient.GradientPointGet, len(points))
	for i, pt := range points {
		gradientPoints[i] = hueclient.GradientPointGet{Color: pt}
	}
	b.state.SetLightGradientPoints(lightID, gradientPoints)

	// Debounce the actual API call (use same mechanism as color)
	b.debounceMu.Lock()
	debounceKey := "gradient:" + lightID
	if timer, ok := b.colorDebounce[debounceKey]; ok {
		timer.Stop()
	}

	lightName := "Unknown"
	if light, ok := b.state.GetLight(lightID); ok {
		lightName = b.state.GetLightName(light)
	}

	// Convert points to the wrapped format expected by the PUT API
	wrappedPoints := make([]gradientPointPut, len(points))
	for i := range points {
		wrappedPoints[i] = gradientPointPut{Color: &points[i]}
	}

	b.colorDebounce[debounceKey] = time.AfterFunc(50*time.Millisecond, func() {
		b.mu.RLock()
		client := b.client
		b.mu.RUnlock()

		if client != nil {
			b.logRequest(fmt.Sprintf("%s: gradient points (%d)", lightName, len(wrappedPoints)))

			// Build custom request body with properly wrapped gradient points
			reqBody := lightGradientPutBody{
				Gradient: &gradientPut{Points: wrappedPoints},
			}
			bodyBytes, err := json.Marshal(reqBody)
			if err != nil {
				b.logError(fmt.Sprintf("%s: gradient points failed: %v", lightName, err))
				return
			}

			slog.Debug("gradient points request", "lightID", lightID, "body", string(bodyBytes))

			resp, err := client.UpdateLightWithBody(context.Background(), toResourceId(lightID), "application/json", bytes.NewReader(bodyBytes))
			if err != nil {
				b.logError(fmt.Sprintf("%s: gradient points failed: %v", lightName, err))
			} else {
				defer resp.Body.Close()
				if resp.StatusCode >= 400 {
					respBody, _ := io.ReadAll(resp.Body)
					slog.Debug("gradient points error", "lightID", lightID, "response", string(respBody))
					b.logError(fmt.Sprintf("%s: gradient points failed: HTTP %d", lightName, resp.StatusCode))
				}
			}
		}

		// Clean up timer
		b.debounceMu.Lock()
		delete(b.colorDebounce, debounceKey)
		b.debounceMu.Unlock()
	})
	b.debounceMu.Unlock()

	return nil
}

// SetLightPowerupPreset updates a light's power-on behavior preset.
func (b *Bridge) SetLightPowerupPreset(lightID string, preset hueclient.UpdateLightJSONBodyPowerupPreset) error {
	b.mu.RLock()
	client := b.client
	b.mu.RUnlock()

	if client == nil {
		return ErrAuthFailed
	}

	// Get light name BEFORE optimistic update to ensure we can retrieve it
	lightName := "Unknown"
	if light, ok := b.state.GetLight(lightID); ok {
		lightName = b.state.GetLightName(light)
	}

	// Optimistic update: apply to cache immediately - convert to Get type
	b.state.SetLightPowerupPreset(lightID, hueclient.LightGetPowerupPreset(preset))

	b.logRequest(fmt.Sprintf("%s: power-on preset changed to \"%s\"", lightName, string(preset)))
	_, err := client.UpdateLight(context.Background(), toResourceId(lightID), hueclient.UpdateLightJSONRequestBody{
		Powerup: &struct {
			Color *struct {
				Color *struct {
					Xy *struct {
						X float32 `json:"x"`
						Y float32 `json:"y"`
					} `json:"xy,omitempty"`
				} `json:"color,omitempty"`
				ColorTemperature *struct {
					Mirek *int `json:"mirek,omitempty"`
				} `json:"color_temperature,omitempty"`
				Mode hueclient.UpdateLightJSONBodyPowerupColorMode `json:"mode"`
			} `json:"color,omitempty"`
			Dimming *struct {
				Dimming *struct {
					Brightness *float32 `json:"brightness,omitempty"`
				} `json:"dimming,omitempty"`
				Mode hueclient.UpdateLightJSONBodyPowerupDimmingMode `json:"mode"`
			} `json:"dimming,omitempty"`
			On *struct {
				Mode hueclient.UpdateLightJSONBodyPowerupOnMode `json:"mode"`
				On   *struct {
					On *bool `json:"on,omitempty"`
				} `json:"on,omitempty"`
			} `json:"on,omitempty"`
			Preset hueclient.UpdateLightJSONBodyPowerupPreset `json:"preset"`
		}{
			Preset: preset,
		},
	})
	return err
}

// SetLightTimedEffect sets a light's timed effect (sunrise, sunset, or no_effect).
// Duration is in milliseconds, max 21600000 (6 hours). Duration is required for sunrise/sunset.
func (b *Bridge) SetLightTimedEffect(lightID string, effect hueclient.SupportedTimedEffects, durationMs int) error {
	b.mu.RLock()
	client := b.client
	b.mu.RUnlock()

	if client == nil {
		return ErrAuthFailed
	}

	lightName := "Unknown"
	if light, ok := b.state.GetLight(lightID); ok {
		lightName = b.state.GetLightName(light)
	}

	effectName := string(effect)
	if effectName == "no_effect" {
		b.logRequest(fmt.Sprintf("%s: timed effect stopped", lightName))
	} else {
		durationMin := durationMs / 60000
		b.logRequest(fmt.Sprintf("%s: timed effect %s (%d min)", lightName, effectName, durationMin))
	}

	body := hueclient.UpdateLightJSONRequestBody{
		TimedEffects: &struct {
			Duration *int                             `json:"duration,omitempty"`
			Effect   *hueclient.SupportedTimedEffects `json:"effect,omitempty"`
		}{
			Effect: &effect,
		},
	}

	// Duration is required for sunrise/sunset, omitted for no_effect
	if effect != hueclient.SupportedTimedEffectsNoEffect {
		body.TimedEffects.Duration = &durationMs
	}

	_, err := client.UpdateLight(context.Background(), toResourceId(lightID), body)
	return err
}

// SetLightSignaling triggers a signaling effect on a light.
// Signal can be "no_signal", "on_off", "on_off_color", or "alternating".
// Duration is in milliseconds (max 65534000 ms). Duration is ignored for no_signal.
func (b *Bridge) SetLightSignaling(lightID string, signal hueclient.SupportedSignals, durationMs int) error {
	b.mu.RLock()
	client := b.client
	b.mu.RUnlock()

	if client == nil {
		return ErrAuthFailed
	}

	lightName := "Unknown"
	if light, ok := b.state.GetLight(lightID); ok {
		lightName = b.state.GetLightName(light)
	}

	signalName := string(signal)
	if signalName == "no_signal" {
		b.logRequest(fmt.Sprintf("%s: signal stopped", lightName))
	} else {
		durationSec := durationMs / 1000
		b.logRequest(fmt.Sprintf("%s: signal %s (%d sec)", lightName, signalName, durationSec))
	}

	body := hueclient.UpdateLightJSONRequestBody{
		Signaling: &struct {
			Colors *[]struct {
				Xy *struct {
					X float32 `json:"x"`
					Y float32 `json:"y"`
				} `json:"xy,omitempty"`
			} `json:"colors,omitempty"`
			Duration int                      `json:"duration"`
			Signal   hueclient.SupportedSignals `json:"signal"`
		}{
			Signal:   signal,
			Duration: durationMs,
		},
	}

	_, err := client.UpdateLight(context.Background(), toResourceId(lightID), body)
	return err
}
