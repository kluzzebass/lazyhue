package ui

import (
	"math"
	"testing"
)

func TestHsvToRGB(t *testing.T) {
	tests := []struct {
		name       string
		h, s, v    int
		wantR, wantG, wantB uint8
	}{
		{"red", 0, 100, 100, 255, 0, 0},
		{"green", 120, 100, 100, 0, 255, 0},
		{"blue", 240, 100, 100, 0, 0, 255},
		{"yellow", 60, 100, 100, 255, 255, 0},
		{"cyan", 180, 100, 100, 0, 255, 255},
		{"magenta", 300, 100, 100, 255, 0, 255},
		{"white", 0, 0, 100, 255, 255, 255},
		{"black", 0, 0, 0, 0, 0, 0},
		{"gray", 0, 0, 50, 127, 127, 127},
		{"half saturation red", 0, 50, 100, 255, 127, 127},
		{"half brightness red", 0, 100, 50, 127, 0, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, g, b := HsvToRGB(tt.h, tt.s, tt.v)
			// Allow ±1 tolerance for rounding
			if abs(int(r)-int(tt.wantR)) > 1 || abs(int(g)-int(tt.wantG)) > 1 || abs(int(b)-int(tt.wantB)) > 1 {
				t.Errorf("HsvToRGB(%d, %d, %d) = (%d, %d, %d), want (%d, %d, %d)",
					tt.h, tt.s, tt.v, r, g, b, tt.wantR, tt.wantG, tt.wantB)
			}
		})
	}
}

func TestMirekToRGB(t *testing.T) {
	// Test that warmer temperatures (higher mirek) produce warmer colors
	// and cooler temperatures (lower mirek) produce cooler colors
	warmR, _, warmB := MirekToRGB(500)  // 2000K - warm
	coolR, _, coolB := MirekToRGB(153)  // 6500K - cool

	// Warm should have more red relative to blue than cool
	warmRatio := float64(warmR) / float64(max(warmB, 1))
	coolRatio := float64(coolR) / float64(max(coolB, 1))

	if warmRatio <= coolRatio {
		t.Errorf("Warm (mirek=500) should have higher R/B ratio than cool (mirek=153): warm=%f, cool=%f",
			warmRatio, coolRatio)
	}

	// Sanity check: values should be non-zero for valid color temps
	for _, mirek := range []int{153, 250, 350, 500} {
		r, g, b := MirekToRGB(mirek)
		// At least one channel should be non-zero for any valid color temp
		if r == 0 && g == 0 && b == 0 {
			t.Errorf("MirekToRGB(%d) produced all-zero RGB: (%d, %d, %d)", mirek, r, g, b)
		}
	}
}

func TestXyToRGB(t *testing.T) {
	tests := []struct {
		name       string
		x, y, brightness float64
		// Check dominant color channel
		dominant string // "r", "g", "b", or "equal"
	}{
		{"red region", 0.64, 0.33, 100, "r"},
		{"green region", 0.30, 0.60, 100, "g"},
		{"blue region", 0.15, 0.06, 100, "b"},
		{"white point", 0.3127, 0.329, 100, "equal"},
		{"zero brightness", 0.64, 0.33, 0, "equal"}, // All should be near 0
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, g, b := XyToRGB(tt.x, tt.y, tt.brightness)
			switch tt.dominant {
			case "r":
				if r <= g || r <= b {
					t.Errorf("XyToRGB(%f, %f, %f) = (%d, %d, %d); expected red dominant",
						tt.x, tt.y, tt.brightness, r, g, b)
				}
			case "g":
				if g <= r || g <= b {
					t.Errorf("XyToRGB(%f, %f, %f) = (%d, %d, %d); expected green dominant",
						tt.x, tt.y, tt.brightness, r, g, b)
				}
			case "b":
				if b <= r || b <= g {
					t.Errorf("XyToRGB(%f, %f, %f) = (%d, %d, %d); expected blue dominant",
						tt.x, tt.y, tt.brightness, r, g, b)
				}
			case "equal":
				if tt.brightness == 0 {
					// All should be near 0
					if r > 10 || g > 10 || b > 10 {
						t.Errorf("XyToRGB(%f, %f, %f) = (%d, %d, %d); expected near zero",
							tt.x, tt.y, tt.brightness, r, g, b)
					}
				}
			}
		})
	}
}

func TestXyToHueSat(t *testing.T) {
	tests := []struct {
		name     string
		x, y     float64
		wantHueMin, wantHueMax int
		wantSatMin, wantSatMax int
	}{
		{"white point", 0.3127, 0.329, 0, 360, 0, 5},
		{"saturated color", 0.5, 0.4, 0, 360, 30, 100},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hue, sat := XyToHueSat(tt.x, tt.y)
			if hue < tt.wantHueMin || hue > tt.wantHueMax {
				t.Errorf("XyToHueSat(%f, %f) hue = %d; want in range [%d, %d]",
					tt.x, tt.y, hue, tt.wantHueMin, tt.wantHueMax)
			}
			if sat < tt.wantSatMin || sat > tt.wantSatMax {
				t.Errorf("XyToHueSat(%f, %f) sat = %d; want in range [%d, %d]",
					tt.x, tt.y, sat, tt.wantSatMin, tt.wantSatMax)
			}
		})
	}
}

func TestHueSatToXY(t *testing.T) {
	tests := []struct {
		name    string
		hue, sat int
	}{
		{"red", 0, 100},
		{"green", 120, 100},
		{"blue", 240, 100},
		{"white", 0, 0},
		{"half saturated", 180, 50},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			x, y := HueSatToXY(tt.hue, tt.sat)
			// Check bounds
			if x < 0 || x > 1 || y < 0 || y > 1 {
				t.Errorf("HueSatToXY(%d, %d) = (%f, %f); out of bounds [0, 1]",
					tt.hue, tt.sat, x, y)
			}
			// White should be near white point
			if tt.sat == 0 {
				if math.Abs(x-0.3127) > 0.01 || math.Abs(y-0.329) > 0.01 {
					t.Errorf("HueSatToXY(%d, %d) = (%f, %f); expected near white point (0.3127, 0.329)",
						tt.hue, tt.sat, x, y)
				}
			}
		})
	}
}

func TestRGBToXYRoundTrip(t *testing.T) {
	tests := []struct {
		name       string
		r, g, b    uint8
		tolerance  int
	}{
		{"red", 255, 0, 0, 8},
		{"green", 0, 255, 0, 8},
		{"blue", 0, 0, 255, 8},
		{"white", 255, 255, 255, 6},
		{"warm", 255, 180, 100, 8},
		{"cool", 150, 200, 255, 8},
		{"dim teal", 32, 96, 96, 10},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			x, y := RGBToXY(tt.r, tt.g, tt.b)
			r2, g2, b2 := XyToRGB(x, y, 100)
			if abs(int(r2)-int(tt.r)) > tt.tolerance ||
				abs(int(g2)-int(tt.g)) > tt.tolerance ||
				abs(int(b2)-int(tt.b)) > tt.tolerance {
				t.Errorf("RGB->XY->RGB mismatch: start (%d,%d,%d) got (%d,%d,%d)",
					tt.r, tt.g, tt.b, r2, g2, b2)
			}
		})
	}
}

func TestRGBToHSLRoundTrip(t *testing.T) {
	tests := []struct {
		name       string
		r, g, b    uint8
		tolerance  int
	}{
		{"red", 255, 0, 0, 3},
		{"green", 0, 255, 0, 3},
		{"blue", 0, 0, 255, 3},
		{"white", 255, 255, 255, 3},
		{"gray", 128, 128, 128, 3},
		{"teal", 0, 128, 128, 4},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, s, l := RGBToHSL(tt.r, tt.g, tt.b)
			r2, g2, b2 := HSLToRGB(h, s, l)
			if abs(int(r2)-int(tt.r)) > tt.tolerance ||
				abs(int(g2)-int(tt.g)) > tt.tolerance ||
				abs(int(b2)-int(tt.b)) > tt.tolerance {
				t.Errorf("RGB->HSL->RGB mismatch: start (%d,%d,%d) got (%d,%d,%d)",
					tt.r, tt.g, tt.b, r2, g2, b2)
			}
		})
	}
}

func TestRGBToHSVRoundTrip(t *testing.T) {
	tests := []struct {
		name       string
		r, g, b    uint8
		tolerance  int
	}{
		{"red", 255, 0, 0, 3},
		{"green", 0, 255, 0, 3},
		{"blue", 0, 0, 255, 3},
		{"white", 255, 255, 255, 3},
		{"gray", 128, 128, 128, 3},
		{"orange", 255, 165, 0, 4},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, s, v := RGBToHSV(tt.r, tt.g, tt.b)
			r2, g2, b2 := HsvToRGB(h, s, v)
			if abs(int(r2)-int(tt.r)) > tt.tolerance ||
				abs(int(g2)-int(tt.g)) > tt.tolerance ||
				abs(int(b2)-int(tt.b)) > tt.tolerance {
				t.Errorf("RGB->HSV->RGB mismatch: start (%d,%d,%d) got (%d,%d,%d)",
					tt.r, tt.g, tt.b, r2, g2, b2)
			}
		})
	}
}

func TestXYToHSLRoundTrip(t *testing.T) {
	tests := []struct {
		name   string
		x, y   float64
	}{
		{"red region", 0.64, 0.33},
		{"green region", 0.30, 0.60},
		{"blue region", 0.15, 0.06},
		{"white point", 0.3127, 0.329},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, s, l := XYToHSL(tt.x, tt.y)
			x2, y2 := HSLToXY(h, s, l)
			if !nearFloat(tt.x, x2, 0.05) || !nearFloat(tt.y, y2, 0.05) {
				t.Errorf("XY->HSL->XY mismatch: start (%f,%f) got (%f,%f)", tt.x, tt.y, x2, y2)
			}
		})
	}
}

func TestXYToHSVRoundTrip(t *testing.T) {
	tests := []struct {
		name   string
		x, y   float64
	}{
		{"red region", 0.64, 0.33},
		{"green region", 0.30, 0.60},
		{"blue region", 0.15, 0.06},
		{"white point", 0.3127, 0.329},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, s, v := XYToHSV(tt.x, tt.y)
			x2, y2 := HSVToXY(h, s, v)
			if !nearFloat(tt.x, x2, 0.05) || !nearFloat(tt.y, y2, 0.05) {
				t.Errorf("XY->HSV->XY mismatch: start (%f,%f) got (%f,%f)", tt.x, tt.y, x2, y2)
			}
		})
	}
}

func TestRGBIntToXYRoundTrip(t *testing.T) {
	tests := []struct {
		name    string
		r, g, b int
		tolerance int
	}{
		{"red", 255, 0, 0, 8},
		{"green", 0, 255, 0, 8},
		{"blue", 0, 0, 255, 8},
		{"white", 255, 255, 255, 6},
		{"warm", 255, 180, 100, 8},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			x, y := RGBIntToXY(tt.r, tt.g, tt.b)
			r2, g2, b2 := XYToRGBInt(x, y)
			if abs(r2-tt.r) > tt.tolerance ||
				abs(g2-tt.g) > tt.tolerance ||
				abs(b2-tt.b) > tt.tolerance {
				t.Errorf("RGBInt->XY->RGBInt mismatch: start (%d,%d,%d) got (%d,%d,%d)",
					tt.r, tt.g, tt.b, r2, g2, b2)
			}
		})
	}
}

func TestRotateColor(t *testing.T) {
	// Start with a color away from white point
	startX, startY := 0.5, 0.4

	// Rotate by 0 should return same color
	newX, newY := RotateColor(startX, startY, 0)
	if math.Abs(newX-startX) > 0.001 || math.Abs(newY-startY) > 0.001 {
		t.Errorf("RotateColor with delta=0 should return same color")
	}

	// Rotate by 2*Pi should return same color
	newX, newY = RotateColor(startX, startY, 2*math.Pi)
	if math.Abs(newX-startX) > 0.01 || math.Abs(newY-startY) > 0.01 {
		t.Errorf("RotateColor with delta=2*Pi should return approximately same color")
	}

	// Rotate should stay in bounds
	newX, newY = RotateColor(startX, startY, math.Pi/2)
	if newX < 0 || newX > 1 || newY < 0 || newY > 1 {
		t.Errorf("RotateColor result (%f, %f) out of bounds", newX, newY)
	}
}

func TestAdjustSaturation(t *testing.T) {
	startX, startY := 0.5, 0.4
	whiteX, whiteY := 0.3127, 0.329

	// Decrease saturation should move toward white
	newX, newY := AdjustSaturation(startX, startY, -0.5)
	distBefore := math.Sqrt(math.Pow(startX-whiteX, 2) + math.Pow(startY-whiteY, 2))
	distAfter := math.Sqrt(math.Pow(newX-whiteX, 2) + math.Pow(newY-whiteY, 2))
	if distAfter >= distBefore {
		t.Errorf("AdjustSaturation with negative delta should decrease distance from white")
	}

	// Increase saturation should move away from white
	newX, newY = AdjustSaturation(startX, startY, 0.5)
	distAfter = math.Sqrt(math.Pow(newX-whiteX, 2) + math.Pow(newY-whiteY, 2))
	if distAfter <= distBefore {
		t.Errorf("AdjustSaturation with positive delta should increase distance from white")
	}

	// Result should be clamped to valid range
	if newX < 0 || newX > 1 || newY < 0 || newY > 1 {
		t.Errorf("AdjustSaturation result (%f, %f) out of bounds", newX, newY)
	}
}

func TestClampFloat(t *testing.T) {
	tests := []struct {
		name     string
		v, min, max, want float64
	}{
		{"within range", 0.5, 0.0, 1.0, 0.5},
		{"below min", -0.5, 0.0, 1.0, 0.0},
		{"above max", 1.5, 0.0, 1.0, 1.0},
		{"at min", 0.0, 0.0, 1.0, 0.0},
		{"at max", 1.0, 0.0, 1.0, 1.0},
		{"negative range", -5.0, -10.0, -1.0, -5.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ClampFloat(tt.v, tt.min, tt.max)
			if got != tt.want {
				t.Errorf("ClampFloat(%f, %f, %f) = %f, want %f",
					tt.v, tt.min, tt.max, got, tt.want)
			}
		})
	}
}

func TestToTitleCase(t *testing.T) {
	tests := []struct {
		input, want string
	}{
		{"hello world", "Hello World"},
		{"HELLO WORLD", "Hello World"},
		{"hELLO wORLD", "Hello World"},
		{"", ""},
		{"a", "A"},
		{"hello", "Hello"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := ToTitleCase(tt.input)
			if got != tt.want {
				t.Errorf("ToTitleCase(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestToSentenceCase(t *testing.T) {
	tests := []struct {
		input, want string
	}{
		{"hello world", "Hello world"},
		{"HELLO WORLD", "Hello world"},
		{"", ""},
		{"a", "A"},
		{"  hello", "  Hello"}, // First letter found is 'h', gets capitalized
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := ToSentenceCase(tt.input)
			if got != tt.want {
				t.Errorf("ToSentenceCase(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestRGBToHex(t *testing.T) {
	tests := []struct {
		r, g, b uint8
		want    string
	}{
		{255, 0, 0, "#ff0000"},
		{0, 255, 0, "#00ff00"},
		{0, 0, 255, "#0000ff"},
		{255, 255, 255, "#ffffff"},
		{0, 0, 0, "#000000"},
		{128, 64, 32, "#804020"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := RGBToHex(tt.r, tt.g, tt.b)
			if got != tt.want {
				t.Errorf("RGBToHex(%d, %d, %d) = %q, want %q",
					tt.r, tt.g, tt.b, got, tt.want)
			}
		})
	}
}

func TestBrightnessIndicator(t *testing.T) {
	tests := []struct {
		brightness float64
		want       string
	}{
		{0, "○"},
		{-5, "○"},
		{10, "◔"},
		{25, "◔"},
		{37, "◔"},
		{38, "◑"},
		{50, "◑"},
		{62, "◑"},
		{63, "◕"},
		{75, "◕"},
		{87, "◕"},
		{88, "●"},
		{100, "●"},
		{150, "●"},
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			got := BrightnessIndicator(tt.brightness)
			if got != tt.want {
				t.Errorf("BrightnessIndicator(%f) = %q, want %q",
					tt.brightness, got, tt.want)
			}
		})
	}
}

func TestRGBColor(t *testing.T) {
	c := RGBColor(128, 64, 32)
	r, g, b, a := c.RGBA()
	// RGBA returns 16-bit values, convert back to 8-bit for comparison
	r8, g8, b8, a8 := uint8(r>>8), uint8(g>>8), uint8(b>>8), uint8(a>>8)
	if r8 != 128 || g8 != 64 || b8 != 32 || a8 != 255 {
		t.Errorf("RGBColor(128, 64, 32) RGBA = (%d, %d, %d, %d), want (128, 64, 32, 255)",
			r8, g8, b8, a8)
	}
}

// Helper function
func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

func nearFloat(a, b, tol float64) bool {
	return math.Abs(a-b) <= tol
}
