package panels

import (
	"fmt"
	"math"
	"strings"

	"github.com/kluzzebass/lazyhue/internal/ui/panels/details"
	"github.com/openhue/openhue-go"
)

// buildDeviceView creates a details view for a device entity.
func (p *DetailsPanel) buildDeviceView(deviceAny interface{}) *details.View {
	device, ok := deviceAny.(openhue.DeviceGet)
	if !ok {
		return details.NewView(p.styles)
	}

	view := details.NewView(p.styles)

	// IDs section (headerless first section)
	ids := details.NewFields()
	if device.Id != nil {
		ids.AddMuted("ID", *device.Id)
	}
	if device.Type != nil {
		ids.AddMuted("Type", string(*device.Type))
	}
	view.Add(ids)

	// Product data section
	if device.ProductData != nil {
		pd := device.ProductData
		view.Add(details.Header("Product"))

		product := details.NewFields()
		if pd.ManufacturerName != nil {
			product.Add("Manufacturer", *pd.ManufacturerName)
		}
		if pd.ProductName != nil {
			product.Add("Product", *pd.ProductName)
		}
		if pd.ModelId != nil {
			product.Add("Model ID", *pd.ModelId)
		}
		if pd.ProductArchetype != nil {
			product.AddMuted("Archetype", string(*pd.ProductArchetype))
		}
		if pd.SoftwareVersion != nil {
			product.Add("Firmware", *pd.SoftwareVersion)
		}
		if pd.HardwarePlatformType != nil {
			product.Add("Platform", *pd.HardwarePlatformType)
		}
		if pd.Certified != nil {
			product.Add("Certified", fmt.Sprintf("%v", *pd.Certified))
		}
		view.Add(product)
	}

	// Metadata archetype (only if different from product archetype)
	if device.Metadata != nil && device.Metadata.Archetype != nil {
		metaArchetype := string(*device.Metadata.Archetype)
		productArchetype := ""
		if device.ProductData != nil && device.ProductData.ProductArchetype != nil {
			productArchetype = string(*device.ProductData.ProductArchetype)
		}
		if metaArchetype != productArchetype {
			view.Add(details.Blank())
			archetype := details.NewFields()
			archetype.AddMuted("Archetype", metaArchetype)
			view.Add(archetype)
		}
	}

	// Sensor readings
	if p.state != nil {
		sensors := details.NewFields()

		if hasMotion, isDetecting := p.state.GetDeviceMotionState(device); hasMotion {
			status := "none"
			if isDetecting {
				status = "detected"
			}
			sensors.Add("Motion", fmt.Sprintf("%s %s", p.styles.OnOffIndicator(isDetecting), status))
		}
		if hasTemp, tempC := p.state.GetDeviceTemperature(device); hasTemp {
			sensors.Add("Temperature", fmt.Sprintf("%.1f°C", tempC))
		}
		if hasLevel, level := p.state.GetDeviceLightLevel(device); hasLevel {
			lux := 0.0
			if level > 1 {
				lux = math.Pow(10, float64(level-1)/10000.0)
			}
			sensors.Add("Light level", fmt.Sprintf("%.0f lux", lux))
		}
		if hasBattery, battLevel, battState := p.state.GetDeviceBattery(device); hasBattery {
			value := fmt.Sprintf("%d%%", battLevel)
			if battState != "" {
				value += fmt.Sprintf(" (%s)", battState)
			}
			sensors.Add("Battery", value)
		}

		if !sensors.IsEmpty() {
			view.Add(details.Header("Sensors"))
			view.Add(sensors)
		}
	}

	// Zigbee connectivity
	if p.state != nil {
		if zc, ok := p.state.GetDeviceZigbeeConnectivity(device); ok {
			view.Add(details.Header("Zigbee"))

			zigbee := details.NewFields()

			// Status with styling
			statusStyle := p.styles.Muted
			if zc.Status == "connected" {
				statusStyle = p.styles.Success
			} else if zc.Status == "connectivity_issue" || zc.Status == "disconnected" {
				statusStyle = p.styles.Error
			}
			zigbee.AddStyled("Status", zc.Status, statusStyle)

			if zc.MacAddress != "" {
				zigbee.AddMuted("MAC", zc.MacAddress)
			}
			if zc.Channel != nil {
				if zc.Channel.Value != "" {
					channelVal := zc.Channel.Value
					if len(channelVal) > 8 && channelVal[:8] == "channel_" {
						channelVal = channelVal[8:]
					}
					zigbee.AddMuted("Channel", channelVal)
				}
				if zc.Channel.Status != "" {
					zigbee.AddMuted("Channel status", zc.Channel.Status)
				}
			}
			if zc.ExtendedPanID != "" {
				zigbee.AddMuted("Extended PAN ID", zc.ExtendedPanID)
			}
			if zc.ID != "" {
				zigbee.AddMuted("Resource ID", zc.ID)
			}
			if zc.IDV1 != "" {
				zigbee.AddMuted("V1 ID", zc.IDV1)
			}
			view.Add(zigbee)
		}
	}

	// Services (skip redundant ones already shown)
	if device.Services != nil && len(*device.Services) > 0 {
		redundant := map[string]bool{
			"device_power":        true,
			"motion":              true,
			"temperature":         true,
			"light_level":         true,
			"zigbee_connectivity": true,
		}

		services := details.NewList()
		for _, svc := range *device.Services {
			if svc.Rtype == nil {
				continue
			}
			rtype := string(*svc.Rtype)
			if redundant[rtype] {
				continue
			}
			name := strings.ReplaceAll(rtype, "_", " ")
			if len(name) > 0 {
				name = strings.ToUpper(name[:1]) + name[1:]
			}
			services.Add(name)
		}

		if !services.IsEmpty() {
			view.Add(details.Header("Services"))
			view.Add(services)
		}
	}

	return view
}
