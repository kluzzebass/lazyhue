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
// This derives hue/sat from the RGB representation to keep conversions consistent.
func XyToHueSat(x, y float64) (hue int, sat int) {
	r, g, b := XyToRGB(x, y, 100)
	h, s, _ := RGBToHSV(r, g, b)
	return h, s
}

// HueSatToXY converts hue (0-360) and saturation (0-100) to XY color.
// Uses full value (brightness) to align with the XY<->RGB conversion.
func HueSatToXY(hue, sat int) (x, y float64) {
	r, g, b := HsvToRGB(hue, sat, 100)
	return RGBToXY(r, g, b)
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

// RGBToXY converts RGB values (0-255) to CIE XY color coordinates.
// This is the reverse of XyToRGB, used for converting user color inputs back to XY.
func RGBToXY(r, g, b uint8) (x, y float64) {
	// Normalize to 0-1
	rFloat := float64(r) / 255.0
	gFloat := float64(g) / 255.0
	bFloat := float64(b) / 255.0

	// Apply gamma correction
	applyGamma := func(v float64) float64 {
		if v > 0.04045 {
			return math.Pow((v+0.055)/1.055, 2.4)
		}
		return v / 12.92
	}

	rFloat = applyGamma(rFloat)
	gFloat = applyGamma(gFloat)
	bFloat = applyGamma(bFloat)

	// Convert to XYZ using Wide RGB D65 matrix (inverse of XyToRGB)
	// Coefficients from Philips Hue documentation for wide-gamut conversion.
	X := rFloat*0.664511 + gFloat*0.154324 + bFloat*0.162028
	Y := rFloat*0.283881 + gFloat*0.668433 + bFloat*0.047685
	Z := rFloat*0.000088 + gFloat*0.072310 + bFloat*0.986039

	// Convert to xy chromaticity
	sum := X + Y + Z
	if sum == 0 {
		// Default to D65 white point for black
		return 0.3127, 0.329
	}

	x = X / sum
	y = Y / sum

	// Clamp to valid range
	x = ClampFloat(x, 0.0, 1.0)
	y = ClampFloat(y, 0.0, 1.0)

	return x, y
}

// HSLToRGB converts HSL values to RGB.
// H: 0-360, S: 0-100, L: 0-100
// Returns R, G, B in 0-255 range.
func HSLToRGB(h, s, l int) (r, g, b uint8) {
	// Normalize to 0-1
	hf := float64(h%360) / 360.0
	sf := float64(s) / 100.0
	lf := float64(l) / 100.0

	if sf == 0 {
		// Achromatic (gray)
		gray := uint8(lf * 255)
		return gray, gray, gray
	}

	var q float64
	if lf < 0.5 {
		q = lf * (1 + sf)
	} else {
		q = lf + sf - lf*sf
	}
	p := 2*lf - q

	hueToRGB := func(p, q, t float64) float64 {
		if t < 0 {
			t += 1
		}
		if t > 1 {
			t -= 1
		}
		if t < 1.0/6.0 {
			return p + (q-p)*6*t
		}
		if t < 0.5 {
			return q
		}
		if t < 2.0/3.0 {
			return p + (q-p)*(2.0/3.0-t)*6
		}
		return p
	}

	rf := hueToRGB(p, q, hf+1.0/3.0)
	gf := hueToRGB(p, q, hf)
	bf := hueToRGB(p, q, hf-1.0/3.0)

	return uint8(rf * 255), uint8(gf * 255), uint8(bf * 255)
}

// RGBToHSL converts RGB values (0-255) to HSL.
// Returns H (0-360), S (0-100), L (0-100).
func RGBToHSL(r, g, b uint8) (h, s, l int) {
	rf := float64(r) / 255.0
	gf := float64(g) / 255.0
	bf := float64(b) / 255.0

	maxVal := math.Max(math.Max(rf, gf), bf)
	minVal := math.Min(math.Min(rf, gf), bf)

	lf := (maxVal + minVal) / 2.0

	if maxVal == minVal {
		// Achromatic
		return 0, 0, int(lf * 100)
	}

	d := maxVal - minVal
	var sf float64
	if lf > 0.5 {
		sf = d / (2.0 - maxVal - minVal)
	} else {
		sf = d / (maxVal + minVal)
	}

	var hf float64
	switch maxVal {
	case rf:
		hf = (gf - bf) / d
		if gf < bf {
			hf += 6
		}
	case gf:
		hf = (bf-rf)/d + 2
	case bf:
		hf = (rf-gf)/d + 4
	}
	hf /= 6.0

	return int(hf * 360), int(sf * 100), int(lf * 100)
}

// XYToHSL converts CIE XY color to HSL.
// Uses full brightness for the conversion.
func XYToHSL(x, y float64) (h, s, l int) {
	r, g, b := XyToRGB(x, y, 100)
	return RGBToHSL(r, g, b)
}

// HSLToXY converts HSL to CIE XY color.
func HSLToXY(h, s, l int) (x, y float64) {
	r, g, b := HSLToRGB(h, s, l)
	return RGBToXY(r, g, b)
}

// XYToRGBInt converts CIE XY to RGB as int values (0-255).
func XYToRGBInt(x, y float64) (r, g, b int) {
	ru, gu, bu := XyToRGB(x, y, 100)
	return int(ru), int(gu), int(bu)
}

// RGBIntToXY converts RGB int values (0-255) to CIE XY.
func RGBIntToXY(r, g, b int) (x, y float64) {
	return RGBToXY(uint8(clampInt(r, 0, 255)), uint8(clampInt(g, 0, 255)), uint8(clampInt(b, 0, 255)))
}

// clampInt clamps an int to the given range.
func clampInt(v, minV, maxV int) int {
	if v < minV {
		return minV
	}
	if v > maxV {
		return maxV
	}
	return v
}

// RGBToHSV converts RGB values (0-255) to HSV.
// Returns H (0-360), S (0-100), V (0-100).
func RGBToHSV(r, g, b uint8) (h, s, v int) {
	rf := float64(r) / 255.0
	gf := float64(g) / 255.0
	bf := float64(b) / 255.0

	maxVal := math.Max(math.Max(rf, gf), bf)
	minVal := math.Min(math.Min(rf, gf), bf)

	vf := maxVal

	if maxVal == minVal {
		// Achromatic
		return 0, 0, int(vf * 100)
	}

	d := maxVal - minVal
	sf := d / maxVal

	var hf float64
	switch maxVal {
	case rf:
		hf = (gf - bf) / d
		if gf < bf {
			hf += 6
		}
	case gf:
		hf = (bf-rf)/d + 2
	case bf:
		hf = (rf-gf)/d + 4
	}
	hf /= 6.0

	return int(hf * 360), int(sf * 100), int(vf * 100)
}

// XYToHSV converts CIE XY color to HSV.
// Uses full brightness for the conversion.
func XYToHSV(x, y float64) (h, s, v int) {
	r, g, b := XyToRGB(x, y, 100)
	return RGBToHSV(r, g, b)
}

// HSVToXY converts HSV to CIE XY color.
func HSVToXY(h, s, v int) (x, y float64) {
	r, g, b := HsvToRGB(h, s, v)
	return RGBToXY(r, g, b)
}

// GetLightColor extracts the RGB color from a light and returns it as a hex string.
// Returns empty string if light is off or has no color information.
func GetLightColor(light hueclient.LightGet) string {
	brightness := 100.0
	if light.Dimming != nil {
		brightness = float64(light.Dimming.Brightness)
	}

	// Try XY color first (color lights)
	if light.Color != nil {
		x := float64(light.Color.Xy.X)
		y := float64(light.Color.Xy.Y)
		r, g, b := XyToRGB(x, y, brightness)
		return RGBToHex(r, g, b)
	}

	// Try color temperature (white ambiance lights)
	if light.ColorTemperature != nil && light.ColorTemperature.MirekValid {
		r, g, b := MirekToRGB(light.ColorTemperature.Mirek)
		return RGBToHex(r, g, b)
	}

	// Default to warm white for non-color lights
	return "#ffcc66"
}
