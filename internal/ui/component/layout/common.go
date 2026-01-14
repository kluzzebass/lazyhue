package layout

import "strings"

// HAlign represents horizontal alignment.
type HAlign int

const (
	HAlignLeft HAlign = iota
	HAlignCenter
	HAlignRight
)

// VAlign represents vertical alignment.
type VAlign int

const (
	VAlignTop VAlign = iota
	VAlignMiddle
	VAlignBottom
)

// Spacing represents spacing on all four sides (used for both padding and margin).
type Spacing struct {
	Top    int
	Right  int
	Bottom int
	Left   int
}

// NewSpacing creates spacing with the same value on all sides.
func NewSpacing(all int) Spacing {
	return Spacing{Top: all, Right: all, Bottom: all, Left: all}
}

// NewSpacingXY creates spacing with horizontal (x) and vertical (y) values.
func NewSpacingXY(x, y int) Spacing {
	return Spacing{Top: y, Right: x, Bottom: y, Left: x}
}

// NewSpacingTRBL creates spacing with specific values for each side.
func NewSpacingTRBL(top, right, bottom, left int) Spacing {
	return Spacing{Top: top, Right: right, Bottom: bottom, Left: left}
}

// ApplySpacing applies spacing (margin or padding) to content.
func ApplySpacing(content string, s Spacing) string {
	if s.Top == 0 && s.Right == 0 && s.Bottom == 0 && s.Left == 0 {
		return content
	}

	lines := strings.Split(content, "\n")

	// Find max width for right spacing
	maxWidth := 0
	for _, line := range lines {
		w := len([]rune(line))
		if w > maxWidth {
			maxWidth = w
		}
	}

	// Apply left and right spacing to each line
	leftPad := strings.Repeat(" ", s.Left)
	for i, line := range lines {
		lineWidth := len([]rune(line))
		rightPad := strings.Repeat(" ", maxWidth-lineWidth+s.Right)
		lines[i] = leftPad + line + rightPad
	}

	// Apply top spacing
	topPadLine := strings.Repeat(" ", s.Left+maxWidth+s.Right)
	topLines := make([]string, s.Top)
	for i := range topLines {
		topLines[i] = topPadLine
	}

	// Apply bottom spacing
	bottomLines := make([]string, s.Bottom)
	for i := range bottomLines {
		bottomLines[i] = topPadLine
	}

	// Combine
	result := append(topLines, lines...)
	result = append(result, bottomLines...)

	return strings.Join(result, "\n")
}

// AlignHorizontal aligns content horizontally within the given width.
func AlignHorizontal(content string, width int, align HAlign) string {
	if width <= 0 {
		return content
	}

	lines := strings.Split(content, "\n")
	for i, line := range lines {
		lineWidth := len([]rune(line))
		if lineWidth >= width {
			continue
		}

		padding := width - lineWidth
		switch align {
		case HAlignLeft:
			lines[i] = line + strings.Repeat(" ", padding)
		case HAlignCenter:
			left := padding / 2
			right := padding - left
			lines[i] = strings.Repeat(" ", left) + line + strings.Repeat(" ", right)
		case HAlignRight:
			lines[i] = strings.Repeat(" ", padding) + line
		}
	}

	return strings.Join(lines, "\n")
}

// AlignVertical aligns content vertically within the given height.
func AlignVertical(content string, height int, align VAlign, width int) string {
	if height <= 0 {
		return content
	}

	lines := strings.Split(content, "\n")
	if len(lines) >= height {
		return content
	}

	// Determine line width for empty padding lines
	lineWidth := width
	if lineWidth <= 0 {
		for _, line := range lines {
			w := len([]rune(line))
			if w > lineWidth {
				lineWidth = w
			}
		}
	}

	emptyLine := strings.Repeat(" ", lineWidth)
	padding := height - len(lines)

	switch align {
	case VAlignTop:
		for i := 0; i < padding; i++ {
			lines = append(lines, emptyLine)
		}
	case VAlignMiddle:
		top := padding / 2
		bottom := padding - top
		topLines := make([]string, top)
		for i := range topLines {
			topLines[i] = emptyLine
		}
		bottomLines := make([]string, bottom)
		for i := range bottomLines {
			bottomLines[i] = emptyLine
		}
		lines = append(topLines, lines...)
		lines = append(lines, bottomLines...)
	case VAlignBottom:
		topLines := make([]string, padding)
		for i := range topLines {
			topLines[i] = emptyLine
		}
		lines = append(topLines, lines...)
	}

	return strings.Join(lines, "\n")
}
