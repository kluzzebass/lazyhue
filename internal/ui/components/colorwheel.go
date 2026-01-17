// Package components provides reusable UI components.
package components

import (
	"fmt"
	"math"

	"github.com/charmbracelet/lipgloss/v2"
	"github.com/kluzzebass/lazyhue/internal/ui"
)

// ColorWheel handles rendering and interaction for an elliptical color picker.
type ColorWheel struct {
	RadiusX int
	RadiusY int

	// Selection position
	SelRow int
	SelCol int

	// Current color in XY space
	ColorX float64
	ColorY float64

	// Original color for cancel
	OriginalX float64
	OriginalY float64

	// Whether the position has been set by user interaction
	PosValid bool

	// Blink state for selected cell
	BlinkOn bool

	// Grayscale renders the wheel in grayscale (for inactive state)
	Grayscale bool
}

// NewColorWheel creates a new color wheel with standard dimensions.
func NewColorWheel() *ColorWheel {
	return &ColorWheel{
		RadiusX: 9,
		RadiusY: 4,
	}
}

// Width returns the width of the wheel in characters.
func (w *ColorWheel) Width() int {
	return w.RadiusX*2 + 1
}

// Height returns the height of the wheel in rows.
func (w *ColorWheel) Height() int {
	return w.RadiusY*2 + 1
}

// SetColor sets the current color and updates the cursor position.
func (w *ColorWheel) SetColor(x, y float64) {
	w.ColorX = x
	w.ColorY = y
	if !w.PosValid {
		w.updatePositionFromColor()
	}
}

// SetOriginal stores the original color for cancel operations.
func (w *ColorWheel) SetOriginal(x, y float64) {
	w.OriginalX = x
	w.OriginalY = y
}

// RestoreOriginal reverts to the original color.
func (w *ColorWheel) RestoreOriginal() {
	w.ColorX = w.OriginalX
	w.ColorY = w.OriginalY
	w.updatePositionFromColor()
}

// cellIsInside checks if a cell is inside the ellipse.
func (w *ColorWheel) cellIsInside(row, col int) bool {
	effectiveRadiusX := float64(w.RadiusX) + 0.5
	hiResRadiusY := float64(w.RadiusY * 2)
	hiResCenterY := float64(w.RadiusY*2) + 0.5
	effectiveRadiusY := hiResRadiusY + 1.0

	dx := col - w.RadiusX
	hiResRowTop := float64(row * 2)
	hiResRowBot := float64(row*2 + 1)
	dyTop := hiResRowTop - hiResCenterY
	dyBot := hiResRowBot - hiResCenterY
	topIn := float64(dx*dx)/(effectiveRadiusX*effectiveRadiusX)+(dyTop*dyTop)/(effectiveRadiusY*effectiveRadiusY) <= 1.0
	botIn := float64(dx*dx)/(effectiveRadiusX*effectiveRadiusX)+(dyBot*dyBot)/(effectiveRadiusY*effectiveRadiusY) <= 1.0
	return topIn || botIn
}

// MoveLeft moves the cursor left if valid.
func (w *ColorWheel) MoveLeft() bool {
	newCol := w.SelCol - 1
	if newCol >= 0 && w.cellIsInside(w.SelRow, newCol) {
		w.SelCol = newCol
		w.updateColorFromPosition()
		w.PosValid = true
		return true
	}
	return false
}

// MoveRight moves the cursor right if valid.
func (w *ColorWheel) MoveRight() bool {
	newCol := w.SelCol + 1
	if newCol < w.Width() && w.cellIsInside(w.SelRow, newCol) {
		w.SelCol = newCol
		w.updateColorFromPosition()
		w.PosValid = true
		return true
	}
	return false
}

// MoveUp moves the cursor up if valid.
func (w *ColorWheel) MoveUp() bool {
	if w.SelRow <= 0 {
		return false
	}
	newRow := w.SelRow - 1
	if w.cellIsInside(newRow, w.SelCol) {
		w.SelRow = newRow
		w.updateColorFromPosition()
		w.PosValid = true
		return true
	}
	// Try to find a valid column in the new row
	for offset := 1; offset <= w.RadiusX; offset++ {
		if w.cellIsInside(newRow, w.SelCol-offset) {
			w.SelRow = newRow
			w.SelCol -= offset
			w.updateColorFromPosition()
			w.PosValid = true
			return true
		}
		if w.cellIsInside(newRow, w.SelCol+offset) {
			w.SelRow = newRow
			w.SelCol += offset
			w.updateColorFromPosition()
			w.PosValid = true
			return true
		}
	}
	return false
}

// MoveDown moves the cursor down if valid.
func (w *ColorWheel) MoveDown() bool {
	if w.SelRow >= w.Height()-1 {
		return false
	}
	newRow := w.SelRow + 1
	if w.cellIsInside(newRow, w.SelCol) {
		w.SelRow = newRow
		w.updateColorFromPosition()
		w.PosValid = true
		return true
	}
	// Try to find a valid column in the new row
	for offset := 1; offset <= w.RadiusX; offset++ {
		if w.cellIsInside(newRow, w.SelCol-offset) {
			w.SelRow = newRow
			w.SelCol -= offset
			w.updateColorFromPosition()
			w.PosValid = true
			return true
		}
		if w.cellIsInside(newRow, w.SelCol+offset) {
			w.SelRow = newRow
			w.SelCol += offset
			w.updateColorFromPosition()
			w.PosValid = true
			return true
		}
	}
	return false
}

// HandleClick handles a mouse click at the given wheel-relative coordinates.
// Returns true if the click was inside the wheel.
func (w *ColorWheel) HandleClick(row, col int) bool {
	if row < 0 || row >= w.Height() || col < 0 || col >= w.Width() {
		return false
	}
	if !w.cellIsInside(row, col) {
		return false
	}
	w.SelRow = row
	w.SelCol = col
	w.updateColorFromPosition()
	w.PosValid = true
	return true
}

// updatePositionFromColor updates the cursor position from the current XY color.
func (w *ColorWheel) updatePositionFromColor() {
	hue, sat := ui.XyToHueSat(w.ColorX, w.ColorY)

	// The wheel uses: hue = -angle*180/π + 90
	// So: angle = (90 - hue) * π/180
	angleRad := float64(90-hue) * math.Pi / 180
	dist := float64(sat) / 100.0
	xNorm := dist * math.Cos(angleRad)
	yNorm := -dist * math.Sin(angleRad)

	w.SelCol = w.RadiusX + int(xNorm*float64(w.RadiusX)+0.5)
	w.SelRow = w.RadiusY + int(yNorm*float64(w.RadiusY)+0.5)

	// Clamp to valid range
	if w.SelRow < 0 {
		w.SelRow = 0
	}
	if w.SelRow > w.RadiusY*2 {
		w.SelRow = w.RadiusY * 2
	}
	if w.SelCol < 0 {
		w.SelCol = 0
	}
	if w.SelCol > w.RadiusX*2 {
		w.SelCol = w.RadiusX * 2
	}

	// Mark position as valid after updating from color
	w.PosValid = true
}

// updateColorFromPosition updates the XY color from the current cursor position.
func (w *ColorWheel) updateColorFromPosition() {
	dx := w.SelCol - w.RadiusX
	dy := w.SelRow - w.RadiusY

	// Normalize to -1..1 range
	xNorm := float64(dx) / float64(w.RadiusX)
	yNorm := float64(dy) / float64(w.RadiusY)

	// Calculate distance (saturation)
	dist := math.Sqrt(xNorm*xNorm + yNorm*yNorm)
	if dist > 1.0 {
		dist = 1.0
	}
	sat := int(dist * 100)

	// Calculate hue - same formula as rendering
	angle := math.Atan2(-yNorm, xNorm)
	hue := -int(angle*180/math.Pi) + 90
	hue = ((hue % 360) + 360) % 360

	// Convert to XY
	w.ColorX, w.ColorY = ui.HueSatToXY(hue, sat)
}

// Render returns the color wheel as a string.
func (w *ColorWheel) Render() string {
	centerRow := w.RadiusY
	diameterY := w.Height()

	effectiveRadiusX := float64(w.RadiusX) + 0.5
	hiResRadiusY := float64(w.RadiusY * 2)
	hiResCenterY := float64(w.RadiusY*2) + 0.5
	effectiveRadiusY := hiResRadiusY + 1.0

	var result string

	for subRow := 0; subRow < diameterY; subRow++ {
		maxCells := w.Width()

		for col := 0; col < maxCells; col++ {
			dx := col - w.RadiusX

			// Check top sub-pixel
			hiResRowTop := float64(subRow * 2)
			dyTop := hiResRowTop - hiResCenterY
			topInside := float64(dx*dx)/(effectiveRadiusX*effectiveRadiusX)+(dyTop*dyTop)/(effectiveRadiusY*effectiveRadiusY) <= 1.0

			// Check bottom sub-pixel
			hiResRowBot := float64(subRow*2 + 1)
			dyBot := hiResRowBot - hiResCenterY
			botInside := float64(dx*dx)/(effectiveRadiusX*effectiveRadiusX)+(dyBot*dyBot)/(effectiveRadiusY*effectiveRadiusY) <= 1.0

			if !topInside && !botInside {
				result += " "
				continue
			}

			// Calculate color for this cell
			var useY float64
			if topInside && botInside {
				useY = float64(subRow - centerRow)
			} else if topInside {
				useY = float64(dyTop) / 2.0
			} else {
				useY = float64(dyBot) / 2.0
			}

			xNorm := float64(dx) / float64(w.RadiusX)
			yNorm := useY / float64(w.RadiusY)
			dist := math.Sqrt(xNorm*xNorm + yNorm*yNorm)

			// Calculate angle - Red at right, counter-clockwise
			angle := math.Atan2(-yNorm, xNorm)
			blockHue := -int(angle*180/math.Pi) + 90
			blockHue = ((blockHue % 360) + 360) % 360

			blockSat := int(dist * 100)
			if blockSat > 100 {
				blockSat = 100
			}

			// Reduce saturation if inactive (grayscale mode)
			sat := blockSat
			if w.Grayscale {
				sat = blockSat / 4 // 25% saturation for muted look
			}
			cr, cg, cb := ui.HsvToRGB(blockHue, sat, 100)

			blockColor := fmt.Sprintf("#%02X%02X%02X", cr, cg, cb)

			// Check if this is the selected cell
			isSelected := subRow == w.SelRow && col == w.SelCol

			if isSelected {
				// Blinking cursor
				bgColor := "#1F2937"
				if w.BlinkOn {
					if topInside && botInside {
						result += lipgloss.NewStyle().Background(lipgloss.Color(blockColor)).Render(" ")
					} else if topInside {
						result += lipgloss.NewStyle().Foreground(lipgloss.Color(blockColor)).Render("▀")
					} else {
						result += lipgloss.NewStyle().Foreground(lipgloss.Color(blockColor)).Render("▄")
					}
				} else {
					if topInside && botInside {
						result += lipgloss.NewStyle().Background(lipgloss.Color(bgColor)).Render(" ")
					} else if topInside {
						result += lipgloss.NewStyle().Foreground(lipgloss.Color(bgColor)).Render("▀")
					} else {
						result += lipgloss.NewStyle().Foreground(lipgloss.Color(bgColor)).Render("▄")
					}
				}
			} else if topInside && botInside {
				result += lipgloss.NewStyle().Background(lipgloss.Color(blockColor)).Render(" ")
			} else if topInside {
				result += lipgloss.NewStyle().Foreground(lipgloss.Color(blockColor)).Render("▀")
			} else {
				result += lipgloss.NewStyle().Foreground(lipgloss.Color(blockColor)).Render("▄")
			}
		}
		result += "\n"
	}

	return result
}
