package panels

import (
	"fmt"

	"github.com/kluzzebass/lazyhue/internal/hue"
	"github.com/kluzzebass/lazyhue/internal/ui/panels/details"
)

// buildEntertainmentView creates a details view for an entertainment configuration.
func (p *DetailsPanel) buildEntertainmentView(cfgAny interface{}) *details.View {
	cfg, ok := cfgAny.(hue.EntertainmentConfiguration)
	if !ok {
		return details.NewView(p.styles)
	}

	view := details.NewView(p.styles)

	// ID and basic info
	ids := details.NewFields()
	ids.AddMuted("ID", cfg.ID)

	name := cfg.ID
	if cfg.Metadata != nil && cfg.Metadata.Name != "" {
		name = cfg.Metadata.Name
	} else if cfg.Name != "" {
		name = cfg.Name
	}
	ids.Add("Name", name)

	// Status with styling
	if cfg.Status != "" {
		statusStyle := p.styles.Muted
		if cfg.Status == "active" {
			statusStyle = p.styles.Success
		} else if cfg.Status == "inactive" {
			statusStyle = p.styles.Muted
		}
		ids.AddStyled("Status", cfg.Status, statusStyle)
	}
	if cfg.ConfigurationType != "" {
		ids.AddMuted("Type", cfg.ConfigurationType)
	}
	view.Add(ids)

	// Channels
	if len(cfg.Channels) > 0 {
		view.Add(details.Header(fmt.Sprintf("Channels (%d)", len(cfg.Channels))))

		for _, ch := range cfg.Channels {
			posStr := ""
			if ch.Position != nil {
				posStr = fmt.Sprintf(" at (%.2f, %.2f, %.2f)", ch.Position.X, ch.Position.Y, ch.Position.Z)
			}

			channelList := details.NewList()
			channelList.Add(fmt.Sprintf("Channel %d%s", ch.ChannelID, posStr))
			view.Add(channelList)

			// Channel members
			if len(ch.Members) > 0 {
				memberList := details.NewList()
				for _, member := range ch.Members {
					if member.Service != nil && member.Service.RID != "" {
						lightName := member.Service.RID
						if p.state != nil {
							if light, ok := p.state.GetLight(member.Service.RID); ok {
								lightName = p.state.GetLightName(light)
							}
						}
						memberList.AddNested(lightName)
					}
				}
				view.Add(memberList)
			}
		}
	}

	// Lights
	if len(cfg.Lights) > 0 {
		view.Add(details.Header(fmt.Sprintf("Lights (%d)", len(cfg.Lights))))
		lightsList := details.NewList()
		for _, light := range cfg.Lights {
			if light.Service != nil && light.Service.RID != "" {
				lightName := light.Service.RID
				if p.state != nil {
					if l, ok := p.state.GetLight(light.Service.RID); ok {
						lightName = p.state.GetLightName(l)
					}
				}
				lightsList.Add(lightName)
			}
		}
		view.Add(lightsList)
	}

	// Light positions
	if cfg.Locations != nil && len(cfg.Locations.ServiceLocations) > 0 {
		view.Add(details.Header("Light Positions"))

		for _, loc := range cfg.Locations.ServiceLocations {
			if loc.Service != nil {
				lightName := loc.Service.RID
				if p.state != nil {
					if l, ok := p.state.GetLight(loc.Service.RID); ok {
						lightName = p.state.GetLightName(l)
					}
				}

				posList := details.NewList()
				posList.Add(lightName)
				view.Add(posList)

				posFields := details.NewFields()
				if loc.Position != nil {
					posFields.AddSub("Position", fmt.Sprintf("(%.2f, %.2f, %.2f)", loc.Position.X, loc.Position.Y, loc.Position.Z))
				}
				for i, pos := range loc.Positions {
					posFields.AddSub(fmt.Sprintf("Position %d", i+1), fmt.Sprintf("(%.2f, %.2f, %.2f)", pos.X, pos.Y, pos.Z))
				}
				if !posFields.IsEmpty() {
					view.Add(posFields)
				}
			}
		}
	}

	// Stream proxy
	if cfg.StreamProxy != nil {
		view.Add(details.Header("Stream Proxy"))
		proxy := details.NewFields()
		if cfg.StreamProxy.Mode != "" {
			proxy.Add("Mode", cfg.StreamProxy.Mode)
		}
		if cfg.StreamProxy.Node != nil {
			proxy.Add("Node", cfg.StreamProxy.Node.RID)
		}
		view.Add(proxy)
	}

	// Active streamer
	if cfg.ActiveStreamer != nil && cfg.ActiveStreamer.RID != "" {
		view.Add(details.Blank())
		streamer := details.NewFields()
		streamer.Add("Active Streamer", cfg.ActiveStreamer.RID)
		view.Add(streamer)
	}

	return view
}
