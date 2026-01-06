package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// BorderConfig holds configuration for custom border rendering.
type BorderConfig struct {
	Title       string   // Title to embed in top border
	Tabs        []string // Tab names for tabbed panels
	TabPrefix   string   // Prefix before tabs (e.g., "[2]")
	ActiveTab   int      // Which tab is active
	ItemIndex   int      // Current item index (0-based)
	ItemCount   int      // Total item count
	ScrollPos   int      // Current scroll position
	TotalHeight int      // Total scrollable height
	ViewHeight  int      // Visible viewport height
}

// RenderBorderedPanel renders content with custom borders including title, tabs, and scroll indicators.
func RenderBorderedPanel(content string, width, height int, active bool, styles Styles, cfg BorderConfig) string {
	border := lipgloss.RoundedBorder()

	// Choose border color based on active state
	borderColor := styles.Theme.Muted
	if active {
		borderColor = styles.Theme.Primary
	}

	// Calculate content area dimensions
	contentWidth := width - 2  // left + right border
	contentHeight := height - 2 // top + bottom border

	// Build top border with title/tabs
	topBorder := buildTopBorder(border, contentWidth, borderColor, styles, cfg)

	// Build bottom border with item count
	bottomBorder := buildBottomBorder(border, contentWidth, borderColor, styles, cfg)

	// Build left and right borders (right has scroll indicator)
	leftBorder := lipgloss.NewStyle().Foreground(borderColor).Render(border.Left)
	rightBorders := buildRightBorder(border, contentHeight, borderColor, cfg)

	// Pad/truncate content to fit
	lines := strings.Split(content, "\n")
	paddedLines := make([]string, contentHeight)
	for i := 0; i < contentHeight; i++ {
		if i < len(lines) {
			// Truncate or pad line to content width
			line := lines[i]
			lineWidth := lipgloss.Width(line)
			if lineWidth > contentWidth {
				line = truncateString(line, contentWidth)
			} else if lineWidth < contentWidth {
				line = line + strings.Repeat(" ", contentWidth-lineWidth)
			}
			paddedLines[i] = line
		} else {
			paddedLines[i] = strings.Repeat(" ", contentWidth)
		}
	}

	// Assemble the panel
	var result strings.Builder
	result.WriteString(topBorder)
	result.WriteString("\n")

	for i, line := range paddedLines {
		result.WriteString(leftBorder)
		result.WriteString(line)
		result.WriteString(rightBorders[i])
		if i < len(paddedLines)-1 {
			result.WriteString("\n")
		}
	}

	result.WriteString("\n")
	result.WriteString(bottomBorder)

	return result.String()
}

func buildTopBorder(border lipgloss.Border, width int, borderColor lipgloss.Color, styles Styles, cfg BorderConfig) string {
	bs := lipgloss.NewStyle().Foreground(borderColor)

	if len(cfg.Tabs) > 0 {
		// Build tabs on border
		return buildTabbedTopBorder(border, width, borderColor, styles, cfg)
	}

	if cfg.Title != "" {
		// Build title on border
		return buildTitledTopBorder(border, width, borderColor, styles, cfg)
	}

	// Plain top border
	return bs.Render(border.TopLeft + strings.Repeat(border.Top, width) + border.TopRight)
}

func buildTitledTopBorder(border lipgloss.Border, width int, borderColor lipgloss.Color, styles Styles, cfg BorderConfig) string {
	bs := lipgloss.NewStyle().Foreground(borderColor)
	title := cfg.Title

	// Truncate title if too long
	maxTitleLen := width - 4 // leave room for corners and some border
	if len(title) > maxTitleLen {
		title = title[:maxTitleLen-2] + ".."
	}

	titleRendered := styles.PanelTitle.Render(title)
	titleWidth := lipgloss.Width(titleRendered)

	// Calculate remaining border width
	remainingWidth := width - titleWidth
	leftPadding := 1
	rightPadding := remainingWidth - leftPadding

	if rightPadding < 0 {
		rightPadding = 0
	}

	return bs.Render(border.TopLeft) +
		bs.Render(strings.Repeat(border.Top, leftPadding)) +
		titleRendered +
		bs.Render(strings.Repeat(border.Top, rightPadding)) +
		bs.Render(border.TopRight)
}

func buildTabbedTopBorder(border lipgloss.Border, width int, borderColor lipgloss.Color, styles Styles, cfg BorderConfig) string {
	bs := lipgloss.NewStyle().Foreground(borderColor)

	// Build prefix if present (e.g., "[2] ")
	prefix := ""
	prefixWidth := 0
	if cfg.TabPrefix != "" {
		prefix = styles.PanelTitle.Render(cfg.TabPrefix) + " "
		prefixWidth = lipgloss.Width(prefix)
	}

	// Build tab string
	var tabParts []string
	for i, tab := range cfg.Tabs {
		style := styles.Muted
		if i == cfg.ActiveTab {
			style = styles.SelectedItem
		}
		tabParts = append(tabParts, style.Render(tab))
		if i < len(cfg.Tabs)-1 {
			tabParts = append(tabParts, bs.Render("─"))
		}
	}
	tabString := strings.Join(tabParts, "")
	tabWidth := lipgloss.Width(tabString)

	// Calculate remaining border width
	remainingWidth := width - prefixWidth - tabWidth
	leftPadding := 1
	rightPadding := remainingWidth - leftPadding

	if rightPadding < 0 {
		rightPadding = 0
	}

	return bs.Render(border.TopLeft) +
		bs.Render(strings.Repeat(border.Top, leftPadding)) +
		prefix +
		tabString +
		bs.Render(strings.Repeat(border.Top, rightPadding)) +
		bs.Render(border.TopRight)
}

func buildBottomBorder(border lipgloss.Border, width int, borderColor lipgloss.Color, styles Styles, cfg BorderConfig) string {
	bs := lipgloss.NewStyle().Foreground(borderColor)

	if cfg.ItemCount == 0 {
		// Plain bottom border
		return bs.Render(border.BottomLeft + strings.Repeat(border.Bottom, width) + border.BottomRight)
	}

	// Build "X of Y" indicator
	countStr := styles.Muted.Render(formatCount(cfg.ItemIndex+1, cfg.ItemCount))
	countWidth := lipgloss.Width(countStr)

	// Calculate remaining border width
	remainingWidth := width - countWidth
	rightPadding := 1
	leftPadding := remainingWidth - rightPadding

	if leftPadding < 0 {
		leftPadding = 0
	}

	return bs.Render(border.BottomLeft) +
		bs.Render(strings.Repeat(border.Bottom, leftPadding)) +
		countStr +
		bs.Render(strings.Repeat(border.Bottom, rightPadding)) +
		bs.Render(border.BottomRight)
}

func buildRightBorder(border lipgloss.Border, height int, borderColor lipgloss.Color, cfg BorderConfig) []string {
	bs := lipgloss.NewStyle().Foreground(borderColor)
	borders := make([]string, height)

	// Calculate scroll thumb position and size
	thumbStart, thumbEnd := calculateScrollThumb(cfg.ScrollPos, cfg.TotalHeight, cfg.ViewHeight, height)

	for i := 0; i < height; i++ {
		if cfg.TotalHeight > cfg.ViewHeight && i >= thumbStart && i < thumbEnd {
			// Scroll thumb - use thicker character
			borders[i] = bs.Render("┃")
		} else {
			borders[i] = bs.Render(border.Right)
		}
	}

	return borders
}

func calculateScrollThumb(scrollPos, totalHeight, viewHeight, borderHeight int) (start, end int) {
	if totalHeight <= viewHeight || borderHeight == 0 {
		return -1, -1 // No scroll needed
	}

	// Fixed thumb size based on visible ratio (minimum 1)
	thumbSize := max(1, (viewHeight*borderHeight)/totalHeight)

	// Calculate thumb position based on selection progress through list
	maxScrollPos := totalHeight - 1
	if maxScrollPos <= 0 {
		return -1, -1
	}

	// Clamp scroll position
	if scrollPos < 0 {
		scrollPos = 0
	}
	if scrollPos > maxScrollPos {
		scrollPos = maxScrollPos
	}

	scrollProgress := float64(scrollPos) / float64(maxScrollPos)
	maxThumbPos := borderHeight - thumbSize
	thumbPos := int(scrollProgress * float64(maxThumbPos))

	return thumbPos, thumbPos + thumbSize
}

func formatCount(current, total int) string {
	return strings.Repeat(" ", 1) + itoa(current) + " of " + itoa(total) + " "
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var result []byte
	for n > 0 {
		result = append([]byte{byte('0' + n%10)}, result...)
		n /= 10
	}
	return string(result)
}

func truncateString(s string, maxWidth int) string {
	if lipgloss.Width(s) <= maxWidth {
		return s
	}
	// Simple truncation - for ANSI strings this might cut mid-sequence
	// A proper implementation would use lipgloss or similar
	runes := []rune(s)
	for len(runes) > 0 && lipgloss.Width(string(runes)) > maxWidth {
		runes = runes[:len(runes)-1]
	}
	return string(runes)
}

