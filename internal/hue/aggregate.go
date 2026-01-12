package hue

import (
	"fmt"

	"github.com/kluzzebass/lazyhue/internal/hueclient"
	"github.com/kluzzebass/lazyhue/internal/ui2"
)

// CalculateRoomAggregate calculates the average brightness and color from a list of lights.
// Returns the average brightness (0-100) and a hex color string representing the average color.
func CalculateRoomAggregate(lights []hueclient.LightGet) (float64, string) {
	if len(lights) == 0 {
		return 0, ""
	}

	var totalBrightness float64
	var totalR, totalG, totalB float64
	var onCount int

	for _, light := range lights {
		if light.On != nil && light.On.On != nil && *light.On.On {
			onCount++
			if light.Dimming != nil && light.Dimming.Brightness != nil {
				totalBrightness += float64(*light.Dimming.Brightness)
			} else {
				totalBrightness += 100.0
			}

			// Get color for this light
			colorHex := ui2.GetLightColor(light)
			if colorHex != "" && len(colorHex) == 7 && colorHex[0] == '#' {
				var r, g, b int
				fmt.Sscanf(colorHex, "#%02x%02x%02x", &r, &g, &b)
				totalR += float64(r)
				totalG += float64(g)
				totalB += float64(b)
			}
		}
	}

	if onCount == 0 {
		return 0, ""
	}

	brightness := totalBrightness / float64(onCount)
	avgR := int(totalR / float64(onCount))
	avgG := int(totalG / float64(onCount))
	avgB := int(totalB / float64(onCount))
	indicatorColor := fmt.Sprintf("#%02x%02x%02x", avgR, avgG, avgB)

	return brightness, indicatorColor
}
