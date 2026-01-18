package field

import (
	"testing"

	"github.com/kluzzebass/lazyhue/internal/ui"
)

func TestColorWheelHueSatPositionEdges(t *testing.T) {
	styles := ui.DefaultStyles()
	c := NewColorWheelComponent("test-wheel", "Color", 0.3127, 0.329, &styles, nil)

	tests := []struct {
		name    string
		hue     int
		sat     int
		wantRow int
		wantCol int
	}{
		{"hue 0 top", 0, 100, 0, c.Wheel.RadiusX},
		{"hue 90 right", 90, 100, c.Wheel.RadiusY, c.Wheel.RadiusX * 2},
		{"hue 180 bottom", 180, 100, c.Wheel.RadiusY * 2, c.Wheel.RadiusX},
		{"hue 270 left", 270, 100, c.Wheel.RadiusY, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c.SetHueSatPosition(tt.hue, tt.sat)
			if c.Wheel.SelRow != tt.wantRow || c.Wheel.SelCol != tt.wantCol {
				t.Fatalf("SelRow/SelCol = (%d,%d), want (%d,%d)", c.Wheel.SelRow, c.Wheel.SelCol, tt.wantRow, tt.wantCol)
			}
		})
	}
}

func TestColorWheelHueSatPositionCenter(t *testing.T) {
	styles := ui.DefaultStyles()
	c := NewColorWheelComponent("test-wheel", "Color", 0.3127, 0.329, &styles, nil)

	c.SetHueSatPosition(120, 0)
	if c.Wheel.SelRow != c.Wheel.RadiusY || c.Wheel.SelCol != c.Wheel.RadiusX {
		t.Fatalf("SelRow/SelCol = (%d,%d), want center (%d,%d)", c.Wheel.SelRow, c.Wheel.SelCol, c.Wheel.RadiusY, c.Wheel.RadiusX)
	}
}
