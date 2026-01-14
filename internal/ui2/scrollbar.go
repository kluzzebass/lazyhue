package ui2

import (
	"image/color"

	"github.com/charmbracelet/lipgloss/v2"
)

// BuildRightBorderWithScrollbar builds right border characters with optional scrollbar.
// Returns a slice of border characters, one per line, with thick characters for the scrollbar thumb.
func BuildRightBorderWithScrollbar(border lipgloss.Border, height int, borderColor color.Color, scrollPos, totalHeight, viewHeight int) []string {
	bs := lipgloss.NewStyle().Foreground(borderColor)
	borders := make([]string, height)

	// Calculate scroll thumb position and size
	thumbStart, thumbEnd := CalculateScrollThumb(scrollPos, totalHeight, viewHeight, height)

	for i := 0; i < height; i++ {
		if totalHeight > viewHeight && i >= thumbStart && i < thumbEnd {
			// Scroll thumb - use thicker character
			borders[i] = bs.Render("┃")
		} else {
			borders[i] = bs.Render(border.Right)
		}
	}

	return borders
}

// CalculateScrollThumb calculates the scrollbar thumb position and size.
// Returns start and end indices for the thumb, or (-1, -1) if no scrollbar is needed.
func CalculateScrollThumb(scrollPos, totalHeight, viewHeight, borderHeight int) (start, end int) {
	if totalHeight <= viewHeight || borderHeight == 0 {
		return -1, -1 // No scroll needed
	}

	// Fixed thumb size based on visible ratio (minimum 1)
	thumbSize := (viewHeight * borderHeight) / totalHeight
	if thumbSize < 1 {
		thumbSize = 1
	}

	// Maximum scroll offset (when last item is visible at bottom)
	maxScrollOffset := totalHeight - viewHeight
	if maxScrollOffset <= 0 {
		return -1, -1
	}

	// Clamp scroll position
	if scrollPos < 0 {
		scrollPos = 0
	}
	if scrollPos > maxScrollOffset {
		scrollPos = maxScrollOffset
	}

	// Calculate thumb position based on scroll offset progress
	scrollProgress := float64(scrollPos) / float64(maxScrollOffset)
	maxThumbPos := borderHeight - thumbSize
	thumbPos := int(scrollProgress * float64(maxThumbPos))

	return thumbPos, thumbPos + thumbSize
}
