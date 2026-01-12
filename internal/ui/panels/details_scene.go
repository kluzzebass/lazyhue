package panels

import (
	"fmt"

	"github.com/kluzzebass/lazyhue/internal/hue"
	"github.com/kluzzebass/lazyhue/internal/hueclient"
	"github.com/kluzzebass/lazyhue/internal/ui/panels/details"
)

// buildSceneView creates a details view for a scene entity.
func (p *DetailsPanel) buildSceneView(sceneAny interface{}) *details.View {
	scene, ok := sceneAny.(hueclient.SceneGet)
	if !ok {
		return details.NewView(p.styles)
	}

	view := details.NewView(p.styles)

	// IDs section
	ids := details.NewFields()
	if scene.Id != nil {
		ids.AddMuted("ID", *scene.Id)
	}
	if scene.IdV1 != nil {
		ids.AddMuted("V1 ID", *scene.IdV1)
	}
	if scene.Type != nil {
		ids.AddMuted("Type", string(*scene.Type))
	}
	if scene.Status != nil && scene.Status.Active != nil {
		ids.Add("Active", string(*scene.Status.Active))
	}
	if scene.Owner != nil && scene.Owner.Rid != nil {
		ownerName := *scene.Owner.Rid
		if scene.Owner.Rtype != nil {
			ownerName = fmt.Sprintf("%s (%s)", ownerName, string(*scene.Owner.Rtype))
		}
		ids.Add("Owner", ownerName)
	}
	if scene.Metadata != nil && scene.Metadata.Appdata != nil && *scene.Metadata.Appdata != "" {
		ids.Add("App data", *scene.Metadata.Appdata)
	}
	view.Add(ids)

	// Group (room/zone)
	if scene.Group != nil && scene.Group.Rid != nil {
		view.Add(details.Header("Group"))
		group := details.NewFields()
		rtype := "unknown"
		if scene.Group.Rtype != nil {
			rtype = string(*scene.Group.Rtype)
		}
		group.AddMuted("Type", rtype)
		group.AddMuted("ID", *scene.Group.Rid)

		if p.state != nil {
			if room, ok := p.state.GetRoom(*scene.Group.Rid); ok {
				group.Add("Name", room.RoomName("Unknown"))
			} else if zone, ok := p.state.GetZone(*scene.Group.Rid); ok {
				group.Add("Name", fmt.Sprintf("%s (zone)", zone.RoomName("Unknown")))
			}
		}
		view.Add(group)
	}

	// Speed and auto dynamic
	if scene.Speed != nil || scene.AutoDynamic != nil {
		view.Add(details.Blank())
		dynamics := details.NewFields()
		if scene.Speed != nil {
			dynamics.Add("Transition speed", fmt.Sprintf("%.2f", *scene.Speed))
		}
		if scene.AutoDynamic != nil {
			dynamics.Add("Auto dynamic", fmt.Sprintf("%v", *scene.AutoDynamic))
		}
		view.Add(dynamics)
	}

	// Palette
	if scene.Palette != nil {
		hasPalette := (scene.Palette.Color != nil && len(*scene.Palette.Color) > 0) ||
			(scene.Palette.Dimming != nil && len(*scene.Palette.Dimming) > 0) ||
			(scene.Palette.ColorTemperature != nil && len(*scene.Palette.ColorTemperature) > 0) ||
			(scene.Palette.Effects != nil && len(*scene.Palette.Effects) > 0)

		if hasPalette {
			view.Add(details.Header("Palette"))
			palette := details.NewFields()

			if scene.Palette.Color != nil && len(*scene.Palette.Color) > 0 {
				palette.Add("Colors", fmt.Sprintf("%d", len(*scene.Palette.Color)))
			}
			if scene.Palette.Dimming != nil && len(*scene.Palette.Dimming) > 0 {
				palette.Add("Dimming levels", fmt.Sprintf("%d", len(*scene.Palette.Dimming)))
			}
			if scene.Palette.ColorTemperature != nil && len(*scene.Palette.ColorTemperature) > 0 {
				palette.Add("Color temps", fmt.Sprintf("%d", len(*scene.Palette.ColorTemperature)))
			}
			if scene.Palette.Effects != nil && len(*scene.Palette.Effects) > 0 {
				palette.Add("Effects", fmt.Sprintf("%d", len(*scene.Palette.Effects)))
			}
			view.Add(palette)

			// Palette details as sub-fields
			paletteDetails := details.NewFields()
			if scene.Palette.Color != nil && len(*scene.Palette.Color) > 0 {
				for i, c := range *scene.Palette.Color {
					if c.Color != nil && c.Color.Xy != nil && c.Color.Xy.X != nil && c.Color.Xy.Y != nil {
						paletteDetails.AddSub(fmt.Sprintf("Color %d", i+1), fmt.Sprintf("XY(%.4f, %.4f)", *c.Color.Xy.X, *c.Color.Xy.Y))
					}
				}
			}
			if scene.Palette.Dimming != nil && len(*scene.Palette.Dimming) > 0 {
				for i, d := range *scene.Palette.Dimming {
					if d.Brightness != nil {
						paletteDetails.AddSub(fmt.Sprintf("Dim %d", i+1), fmt.Sprintf("%.0f%%", *d.Brightness))
					}
				}
			}
			if scene.Palette.ColorTemperature != nil && len(*scene.Palette.ColorTemperature) > 0 {
				for i, ct := range *scene.Palette.ColorTemperature {
					if ct.ColorTemperature != nil && ct.ColorTemperature.Mirek != nil {
						paletteDetails.AddSub(fmt.Sprintf("CT %d", i+1), fmt.Sprintf("%d mirek", *ct.ColorTemperature.Mirek))
					}
				}
			}
			if scene.Palette.Effects != nil && len(*scene.Palette.Effects) > 0 {
				for i, e := range *scene.Palette.Effects {
					if e.Effect != nil {
						paletteDetails.AddSub(fmt.Sprintf("Effect %d", i+1), hue.EffectDisplayName(string(*e.Effect)))
					}
				}
			}
			if !paletteDetails.IsEmpty() {
				view.Add(paletteDetails)
			}
		}
	}

	// Actions
	if scene.Actions != nil && len(*scene.Actions) > 0 {
		view.Add(details.Header(fmt.Sprintf("Actions (%d)", len(*scene.Actions))))

		for _, action := range *scene.Actions {
			targetName := "unknown"
			if action.Target != nil && action.Target.Rid != nil {
				targetName = *action.Target.Rid
				if p.state != nil {
					if light, ok := p.state.GetLight(*action.Target.Rid); ok {
						targetName = p.state.GetLightName(light)
					}
				}
			}

			actionList := details.NewList()
			actionList.Add(targetName)
			view.Add(actionList)

			if action.Action != nil {
				actionFields := details.NewFields()
				if action.Action.On != nil && action.Action.On.On != nil {
					state := "off"
					if *action.Action.On.On {
						state = "on"
					}
					actionFields.AddSub("State", state)
				}
				if action.Action.Dimming != nil && action.Action.Dimming.Brightness != nil {
					actionFields.AddSub("Brightness", fmt.Sprintf("%.0f%%", *action.Action.Dimming.Brightness))
				}
				if action.Action.ColorTemperature != nil && action.Action.ColorTemperature.Mirek != nil {
					actionFields.AddSub("Color temp", fmt.Sprintf("%d mirek", *action.Action.ColorTemperature.Mirek))
				}
				if action.Action.Color != nil && action.Action.Color.Xy != nil && action.Action.Color.Xy.X != nil && action.Action.Color.Xy.Y != nil {
					actionFields.AddSub("Color XY", fmt.Sprintf("(%.4f, %.4f)", *action.Action.Color.Xy.X, *action.Action.Color.Xy.Y))
				}
				if action.Action.Effects != nil && action.Action.Effects.Effect != nil {
					actionFields.AddSub("Effect", hue.EffectDisplayName(string(*action.Action.Effects.Effect)))
				}
				if action.Action.Gradient != nil && action.Action.Gradient.Points != nil {
					actionFields.AddSub("Gradient", fmt.Sprintf("%d points", len(*action.Action.Gradient.Points)))
				}
				if !actionFields.IsEmpty() {
					view.Add(actionFields)
				}
			}
		}
	}

	// Metadata image
	if scene.Metadata != nil && scene.Metadata.Image != nil && scene.Metadata.Image.Rid != nil {
		view.Add(details.Blank())
		image := details.NewFields()
		image.AddMuted("Image", *scene.Metadata.Image.Rid)
		view.Add(image)
	}

	return view
}
