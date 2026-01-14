package ui

import (
	"fmt"
	"image/color"
	"math"
	"strings"
	"unicode"

	"github.com/charmbracelet/lipgloss/v2"
	"github.com/kluzzebass/lazyhue/internal/hueclient"
)

// Colorize applies a foreground color to a string.
func Colorize(s string, c color.Color) string {
	return lipgloss.NewStyle().Foreground(c).Render(s)
}

// PadRight pads a string with spaces to the given width.
func PadRight(s string, width int) string {
	sLen := lipgloss.Width(s)
	if sLen >= width {
		return s[:min(len(s), width)]
	}
	return s + strings.Repeat(" ", width-sLen)
}

// CenterText centers a string within the given width.
func CenterText(s string, width int) string {
	sLen := lipgloss.Width(s)
	if sLen >= width {
		return s[:min(len(s), width)]
	}
	leftPad := (width - sLen) / 2
	rightPad := width - sLen - leftPad
	return strings.Repeat(" ", leftPad) + s + strings.Repeat(" ", rightPad)
}

// Color conversion functions

// HsvToRGB converts HSV (hue 0-360, sat 0-100, val 0-100) to RGB.
func HsvToRGB(h, s, v int) (r, g, b uint8) {
	if s == 0 {
		gray := uint8(v * 255 / 100)
		return gray, gray, gray
	}

	hf := float64(h) / 60.0
	sf := float64(s) / 100.0
	vf := float64(v) / 100.0

	i := int(hf) % 6
	f := hf - float64(int(hf))
	p := vf * (1 - sf)
	q := vf * (1 - sf*f)
	t := vf * (1 - sf*(1-f))

	var rf, gf, bf float64
	switch i {
	case 0:
		rf, gf, bf = vf, t, p
	case 1:
		rf, gf, bf = q, vf, p
	case 2:
		rf, gf, bf = p, vf, t
	case 3:
		rf, gf, bf = p, q, vf
	case 4:
		rf, gf, bf = t, p, vf
	case 5:
		rf, gf, bf = vf, p, q
	}

	return uint8(rf * 255), uint8(gf * 255), uint8(bf * 255)
}

// XyToRGB converts CIE XY color coordinates to RGB.
// Based on the standard conversion formula for Hue lights.
func XyToRGB(x, y, brightness float64) (r, g, b uint8) {
	// Avoid division by zero
	if y == 0 {
		y = 0.00001
	}

	// Calculate XYZ
	Y := brightness / 100.0
	X := (Y / y) * x
	Z := (Y / y) * (1.0 - x - y)

	// Convert to RGB using Wide RGB D65 matrix
	rFloat := X*1.656492 - Y*0.354851 - Z*0.255038
	gFloat := -X*0.707196 + Y*1.655397 + Z*0.036152
	bFloat := X*0.051713 - Y*0.121364 + Z*1.011530

	// Apply reverse gamma correction
	applyGamma := func(v float64) float64 {
		if v <= 0.0031308 {
			return 12.92 * v
		}
		return 1.055*math.Pow(v, 1.0/2.4) - 0.055
	}

	rFloat = applyGamma(rFloat)
	gFloat = applyGamma(gFloat)
	bFloat = applyGamma(bFloat)

	// Clamp and convert to 0-255
	clamp := func(v float64) uint8 {
		if v < 0 {
			return 0
		}
		if v > 1 {
			return 255
		}
		return uint8(v * 255)
	}

	return clamp(rFloat), clamp(gFloat), clamp(bFloat)
}

// MirekToRGB converts color temperature in mirek to RGB.
// Mirek range is typically 153 (cool/6500K) to 500 (warm/2000K).
func MirekToRGB(mirek int) (r, g, b uint8) {
	// Convert mirek to Kelvin: K = 1,000,000 / mirek
	kelvin := 1000000.0 / float64(mirek)

	// Approximate RGB from color temperature
	var rFloat, gFloat, bFloat float64

	// Red
	if kelvin <= 6600 {
		rFloat = 255
	} else {
		rFloat = 329.698727446 * math.Pow(kelvin/100-60, -0.1332047592)
	}

	// Green
	if kelvin <= 6600 {
		gFloat = 99.4708025861*math.Log(kelvin/100) - 161.1195681661
	} else {
		gFloat = 288.1221695283 * math.Pow(kelvin/100-60, -0.0755148492)
	}

	// Blue
	if kelvin >= 6600 {
		bFloat = 255
	} else if kelvin <= 1900 {
		bFloat = 0
	} else {
		bFloat = 138.5177312231*math.Log(kelvin/100-10) - 305.0447927307
	}

	// Clamp to 0-255
	clamp := func(v float64) uint8 {
		if v < 0 {
			return 0
		}
		if v > 255 {
			return 255
		}
		return uint8(v)
	}

	return clamp(rFloat), clamp(gFloat), clamp(bFloat)
}

// XyToHueSat converts XY color to hue (0-360) and saturation (0-100).
func XyToHueSat(x, y float64) (hue int, sat int) {
	whiteX, whiteY := 0.3127, 0.329
	dx := x - whiteX
	dy := y - whiteY
	radius := math.Sqrt(dx*dx + dy*dy)
	angle := math.Atan2(dy, dx)

	// Convert angle to degrees (0-360)
	hue = int(angle * 180 / math.Pi)
	if hue < 0 {
		hue += 360
	}

	// Convert radius to saturation percentage (0-100)
	// Max radius is about 0.4 for fully saturated colors
	sat = int(radius / 0.4 * 100)
	if sat > 100 {
		sat = 100
	}
	if sat < 0 {
		sat = 0
	}

	return hue, sat
}

// HueSatToXY converts hue (0-360) and saturation (0-100) to XY color.
// Maps HSV hue to CIE XY color space, interpolating between RGB primaries.
func HueSatToXY(hue, sat int) (x, y float64) {
	// CIE xy coordinates for sRGB primaries and white point D65
	whiteX, whiteY := 0.3127, 0.329
	redX, redY := 0.64, 0.33
	greenX, greenY := 0.30, 0.60
	blueX, blueY := 0.15, 0.06

	// Normalize hue to 0-360
	h := ((hue % 360) + 360) % 360
	s := float64(sat) / 100.0

	// Determine which two primaries to interpolate between
	var primaryX, primaryY float64
	if h < 60 {
		// Red to Yellow (interpolate Red toward Green)
		t := float64(h) / 60.0
		primaryX = redX + t*(greenX-redX)*0.5
		primaryY = redY + t*(greenY-redY)*0.5
	} else if h < 120 {
		// Yellow to Green
		t := float64(h-60) / 60.0
		primaryX = redX + 0.5*(greenX-redX) + t*0.5*(greenX-redX)
		primaryY = redY + 0.5*(greenY-redY) + t*0.5*(greenY-redY)
	} else if h < 180 {
		// Green to Cyan (interpolate Green toward Blue)
		t := float64(h-120) / 60.0
		primaryX = greenX + t*(blueX-greenX)*0.5
		primaryY = greenY + t*(blueY-greenY)*0.5
	} else if h < 240 {
		// Cyan to Blue
		t := float64(h-180) / 60.0
		primaryX = greenX + 0.5*(blueX-greenX) + t*0.5*(blueX-greenX)
		primaryY = greenY + 0.5*(blueY-greenY) + t*0.5*(blueY-greenY)
	} else if h < 300 {
		// Blue to Magenta (interpolate Blue toward Red)
		t := float64(h-240) / 60.0
		primaryX = blueX + t*(redX-blueX)*0.5
		primaryY = blueY + t*(redY-blueY)*0.5
	} else {
		// Magenta to Red
		t := float64(h-300) / 60.0
		primaryX = blueX + 0.5*(redX-blueX) + t*0.5*(redX-blueX)
		primaryY = blueY + 0.5*(redY-blueY) + t*0.5*(redY-blueY)
	}

	// Interpolate between white and the primary based on saturation
	x = whiteX + s*(primaryX-whiteX)
	y = whiteY + s*(primaryY-whiteY)

	// Clamp to valid range
	x = ClampFloat(x, 0.0, 1.0)
	y = ClampFloat(y, 0.0, 1.0)

	return x, y
}

// RotateColor rotates a color around the color wheel by delta radians.
func RotateColor(x, y, delta float64) (newX, newY float64) {
	// Convert to polar coordinates relative to white point (0.3127, 0.329)
	whiteX, whiteY := 0.3127, 0.329
	dx := x - whiteX
	dy := y - whiteY
	radius := math.Sqrt(dx*dx + dy*dy)
	angle := math.Atan2(dy, dx)

	// Rotate
	angle += delta

	// Convert back
	newX = whiteX + radius*math.Cos(angle)
	newY = whiteY + radius*math.Sin(angle)

	// Clamp to valid range
	newX = ClampFloat(newX, 0.0, 1.0)
	newY = ClampFloat(newY, 0.0, 1.0)

	return newX, newY
}

// AdjustSaturation adjusts the saturation of a color by moving it toward/away from white point.
func AdjustSaturation(x, y, delta float64) (newX, newY float64) {
	whiteX, whiteY := 0.3127, 0.329
	dx := x - whiteX
	dy := y - whiteY

	// Scale the distance from white point
	scale := 1.0 + delta
	if scale < 0.1 {
		scale = 0.1
	}
	if scale > 2.0 {
		scale = 2.0
	}

	newX = whiteX + dx*scale
	newY = whiteY + dy*scale

	// Clamp to valid range
	newX = ClampFloat(newX, 0.0, 1.0)
	newY = ClampFloat(newY, 0.0, 1.0)

	return newX, newY
}

// ClampFloat clamps a float64 to the given range.
func ClampFloat(v, minV, maxV float64) float64 {
	if v < minV {
		return minV
	}
	if v > maxV {
		return maxV
	}
	return v
}

// Text transformation functions

// ToTitleCase capitalizes the first letter of every word.
func ToTitleCase(s string) string {
	words := strings.Fields(s)
	for i, word := range words {
		if len(word) > 0 {
			runes := []rune(word)
			runes[0] = unicode.ToUpper(runes[0])
			for j := 1; j < len(runes); j++ {
				runes[j] = unicode.ToLower(runes[j])
			}
			words[i] = string(runes)
		}
	}
	return strings.Join(words, " ")
}

// ToSentenceCase lowercases everything except the first letter of the first word.
func ToSentenceCase(s string) string {
	if len(s) == 0 {
		return s
	}
	runes := []rune(strings.ToLower(s))
	// Find first letter and capitalize it
	for i, r := range runes {
		if unicode.IsLetter(r) {
			runes[i] = unicode.ToUpper(r)
			break
		}
	}
	return string(runes)
}

// RGBToHex converts RGB values to a hex color string.
func RGBToHex(r, g, b uint8) string {
	return fmt.Sprintf("#%02x%02x%02x", r, g, b)
}

// RGBColor creates a lipgloss-compatible color from RGB values.
func RGBColor(r, g, b uint8) color.Color {
	return color.RGBA{R: r, G: g, B: b, A: 255}
}

// Brightness indicator functions

// BrightnessIndicator returns a character representing the brightness level.
// Uses circle fill characters: ○ ◔ ◑ ◕ ●
// Thresholds are centered around the visual representation:
// ○=0%, ◔=25%, ◑=50%, ◕=75%, ●=100%
func BrightnessIndicator(brightness float64) string {
	switch {
	case brightness <= 0:
		return "○" // off/empty
	case brightness < 37.5:
		return "◔" // quarter (1-37%)
	case brightness < 62.5:
		return "◑" // half (38-62%)
	case brightness < 87.5:
		return "◕" // three-quarters (63-87%)
	default:
		return "●" // full (88-100%)
	}
}

// RenderBrightnessIndicator renders a brightness indicator character with the given color.
// This is the most generic form, used when you have pre-calculated brightness and color.
func RenderBrightnessIndicator(brightness float64, c color.Color) string {
	indicatorChar := BrightnessIndicator(brightness)
	return lipgloss.NewStyle().Foreground(c).Render(indicatorChar)
}

// RenderBrightnessIndicatorFromHex renders a brightness indicator character with a hex color string.
// Convenience function for when you have a hex color string instead of a lipgloss.Color.
func RenderBrightnessIndicatorFromHex(brightness float64, hexColor string) string {
	if hexColor == "" {
		// Default to a warm white if no color provided
		hexColor = "#ffcc66"
	}
	return RenderBrightnessIndicator(brightness, lipgloss.Color(hexColor))
}

// GetLightColor extracts the RGB color from a light and returns it as a hex string.
// Returns empty string if light is off or has no color information.
func GetLightColor(light hueclient.LightGet) string {
	brightness := 100.0
	if light.Dimming != nil && light.Dimming.Brightness != nil {
		brightness = float64(*light.Dimming.Brightness)
	}

	// Try XY color first (color lights)
	if light.Color != nil && light.Color.Xy != nil {
		if light.Color.Xy.X != nil && light.Color.Xy.Y != nil {
			x := float64(*light.Color.Xy.X)
			y := float64(*light.Color.Xy.Y)
			r, g, b := XyToRGB(x, y, brightness)
			return RGBToHex(r, g, b)
		}
	}

	// Try color temperature (white ambiance lights)
	if light.ColorTemperature != nil && light.ColorTemperature.Mirek != nil {
		if light.ColorTemperature.MirekValid == nil || *light.ColorTemperature.MirekValid {
			r, g, b := MirekToRGB(*light.ColorTemperature.Mirek)
			return RGBToHex(r, g, b)
		}
	}

	// Default to warm white for non-color lights
	return "#ffcc66"
}
