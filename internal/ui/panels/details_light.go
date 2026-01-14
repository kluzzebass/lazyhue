package panels

import (
	"fmt"

	"github.com/kluzzebass/lazyhue/internal/hue"
	"github.com/kluzzebass/lazyhue/internal/hueclient"
	"github.com/kluzzebass/lazyhue/internal/ui/panels/details"
)

// buildLightView creates a details view for a light entity.
func (p *DetailsPanel) buildLightView(lightAny interface{}) *details.View {
	if lightAny == nil {
		return details.NewView(p.styles)
	}
	light, ok := lightAny.(hueclient.LightGet)
	if !ok {
		return details.NewView(p.styles)
	}

	view := details.NewView(p.styles)

	// Get owning device info for product details
	var device *hueclient.DeviceGet
	if p.state != nil && light.Owner != nil && light.Owner.Rid != nil {
		if d, ok := p.state.GetDevice(*light.Owner.Rid); ok {
			device = &d
		}
	}

	// Product info from device
	if device != nil && device.ProductData != nil {
		pd := device.ProductData
		product := details.NewFields()
		if pd.ProductName != nil {
			product.Add("Product", *pd.ProductName)
		}
		if pd.ManufacturerName != nil {
			product.Add("Manufacturer", *pd.ManufacturerName)
		}
		if pd.ModelId != nil {
			product.Add("Model", *pd.ModelId)
		}
		if pd.SoftwareVersion != nil {
			product.Add("Firmware", *pd.SoftwareVersion)
		}
		if pd.HardwarePlatformType != nil {
			product.Add("Hardware", *pd.HardwarePlatformType)
		}
		view.Add(product)
		view.Add(details.Blank())
	}

	// Classification
	class := details.NewFields()
	if device != nil && device.ProductData != nil && device.ProductData.ProductArchetype != nil {
		class.Add("Archetype", string(*device.ProductData.ProductArchetype))
	}
	if light.Type != nil {
		class.Add("Type", string(*light.Type))
	}
	if light.Mode != nil {
		class.Add("Mode", string(*light.Mode))
	}
	if !class.IsEmpty() {
		view.Add(class)
		view.Add(details.Blank())
	}

	// Name section
	nameFields := details.NewFields()
	if device != nil {
		if device.Metadata != nil && device.Metadata.Name != nil {
			nameFields.Add("Name", *device.Metadata.Name)
		}
	}
	// Show alternate name if available and different from current name
	if light.Metadata != nil && light.Metadata.Name != nil {
		altName := *light.Metadata.Name
		currentName := ""
		if device != nil && device.Metadata != nil && device.Metadata.Name != nil {
			currentName = *device.Metadata.Name
		}
		if altName != currentName {
			nameFields.AddMuted("Alternate name", altName)
		}
	}
	if !nameFields.IsEmpty() {
		view.Add(nameFields)
		view.Add(details.Blank())
	}

	// IDs
	ids := details.NewFields()
	if light.Id != nil {
		ids.Add("Light ID", *light.Id)
	}
	if light.Owner != nil && light.Owner.Rid != nil {
		ids.Add("Device ID", *light.Owner.Rid)
	}
	if light.IdV1 != nil {
		ids.Add("V1 ID", *light.IdV1)
	}
	view.Add(ids)

	// Current State
	view.Add(details.Header("State"))
	indicator := RenderLightIndicatorFromLight(light, p.styles)
	status := "off"
	if IsLightOn(light) {
		status = "on"
	}
	stateList := details.NewList()
	stateList.AddCustomFull(details.ListItem{Bullet: indicator, Text: status})
	view.Add(stateList)

	stateFields := details.NewFields()
	if light.Dimming != nil {
		if light.Dimming.Brightness != nil {
			stateFields.Add("Brightness", fmt.Sprintf("%.0f%%", float64(*light.Dimming.Brightness)))
		}
		if light.Dimming.MinDimLevel != nil {
			stateFields.Add("Min dim level", fmt.Sprintf("%.0f%%", float64(*light.Dimming.MinDimLevel)))
		}
	}

	if light.Color != nil && light.Color.Xy != nil {
		xy := light.Color.Xy
		if xy.X != nil && xy.Y != nil {
			x, y := *xy.X, *xy.Y
			stateFields.Add("Color XY", fmt.Sprintf("(%.4f, %.4f)", x, y))
		}
		if light.Color.GamutType != nil {
			stateFields.Add("Gamut", string(*light.Color.GamutType))
		}
	}

	if light.ColorTemperature != nil {
		if light.ColorTemperature.Mirek != nil {
			mirek := *light.ColorTemperature.Mirek
			kelvin := 1000000 / int(mirek)
			stateFields.Add("Color temp", fmt.Sprintf("%d mirek (~%dK)", mirek, kelvin))
		}
		if light.ColorTemperature.MirekSchema != nil {
			schema := light.ColorTemperature.MirekSchema
			if schema.MirekMinimum != nil && schema.MirekMaximum != nil {
				minK := 1000000 / int(*schema.MirekMaximum)
				maxK := 1000000 / int(*schema.MirekMinimum)
				stateFields.Add("CT range", fmt.Sprintf("%dK - %dK", minK, maxK))
			}
		}
	}
	if !stateFields.IsEmpty() {
		view.Add(stateFields)
	}

	// Dynamics
	if light.Dynamics != nil {
		view.Add(details.Header("Dynamics"))
		dynamics := details.NewFields()
		if light.Dynamics.Status != nil {
			dynamics.Add("Status", string(*light.Dynamics.Status))
		}
		if light.Dynamics.Speed != nil {
			dynamics.Add("Speed", fmt.Sprintf("%.2f", *light.Dynamics.Speed))
		}
		view.Add(dynamics)
	}

	// Capabilities
	var caps []string
	if light.Dimming != nil {
		caps = append(caps, "Dimming")
	}
	if light.Color != nil {
		caps = append(caps, "Color")
	}
	if light.ColorTemperature != nil {
		caps = append(caps, "Color Temperature")
	}
	if light.Gradient != nil {
		caps = append(caps, "Gradient")
	}
	if light.Effects != nil {
		caps = append(caps, "Effects")
	}
	if light.TimedEffects != nil {
		caps = append(caps, "Timed Effects")
	}

	view.Add(details.Header("Capabilities"))
	capsList := details.NewList()
	if len(caps) > 0 {
		for _, cap := range caps {
			capsList.Add(cap)
		}
	} else {
		capsList.AddStyled("On/Off only", p.styles.Muted)
	}
	view.Add(capsList)

	// Effects
	if light.Effects != nil && light.Effects.EffectValues != nil && len(*light.Effects.EffectValues) > 0 {
		view.Add(details.Header("Available Effects"))
		effectsList := details.NewList()
		for _, effect := range *light.Effects.EffectValues {
			effectsList.Add(hue.EffectDisplayName(string(effect)))
		}
		view.Add(effectsList)
	}

	// Gradient
	if light.Gradient != nil {
		view.Add(details.Header("Gradient"))
		gradient := details.NewFields()
		if light.Gradient.Mode != nil {
			gradient.AddMuted("Mode", string(*light.Gradient.Mode))
		}
		if light.Gradient.PixelCount != nil {
			gradient.Add("Pixels", fmt.Sprintf("%d", *light.Gradient.PixelCount))
		}
		if light.Gradient.Points != nil && len(*light.Gradient.Points) > 0 {
			gradient.Add("Points", fmt.Sprintf("%d", len(*light.Gradient.Points)))
		}
		view.Add(gradient)
	}

	// Signaling
	if light.Signaling != nil && light.Signaling.SignalValues != nil && len(*light.Signaling.SignalValues) > 0 {
		view.Add(details.Header("Signaling Modes"))
		sigList := details.NewList()
		for _, sig := range *light.Signaling.SignalValues {
			sigList.Add(string(sig))
		}
		view.Add(sigList)
	}

	// Powerup behavior
	if light.Powerup != nil && light.Powerup.Preset != nil {
		view.Add(details.Header("Power-on Behavior"))
		powerup := details.NewFields()
		powerup.Add("Preset", string(*light.Powerup.Preset))
		view.Add(powerup)
	}

	// Device services
	if device != nil && device.Services != nil && len(*device.Services) > 0 {
		view.Add(details.Header("Device Services"))
		servicesList := details.NewList()
		for _, svc := range *device.Services {
			rtype := "unknown"
			if svc.Rtype != nil {
				rtype = string(*svc.Rtype)
			}
			if svc.Rid != nil && light.Id != nil && *svc.Rid == *light.Id {
				servicesList.Add(fmt.Sprintf("%s %s", rtype, p.styles.Muted.Render("(this)")))
			} else {
				servicesList.Add(rtype)
			}
		}
		view.Add(servicesList)
	}

	return view
}
