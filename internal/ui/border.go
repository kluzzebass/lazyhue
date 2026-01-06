package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// BorderConfig holds configuration for custom border rendering.
type BorderConfig struct {
	PanelKey       string   // Panel key number (e.g., "1")
	Title          string   // Title to embed in top border (e.g., "Bridges")
	TitleIndicator string   // Optional indicator after title (e.g., "●" for activity)
	Tabs           []string // Tab names for tabbed panels
	ActiveTab      int      // Which tab is active
	ItemIndex      int      // Current item index (0-based)
	ItemCount      int      // Total item count
	ScrollPos      int      // Current scroll position
	TotalHeight    int      // Total scrollable height
	ViewHeight     int      // Visible viewport height
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
	contentWidth := width - 2   // left + right border
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
	for i := range contentHeight {
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

	// Build key prefix (e.g., "[1]")
	keyRendered := ""
	keyWidth := 0
	if cfg.PanelKey != "" {
		keyRendered = styles.PanelTitle.Render("[" + cfg.PanelKey + "]")
		keyWidth = lipgloss.Width(keyRendered)
	}

	// Build title
	title := cfg.Title
	maxTitleLen := width - keyWidth - 8 // leave room for corners, border segments, and indicator
	if len(title) > maxTitleLen && maxTitleLen > 2 {
		title = title[:maxTitleLen-2] + ".."
	}
	titleRendered := styles.PanelTitle.Render(title)
	titleWidth := lipgloss.Width(titleRendered)

	// Build indicator (e.g., "●" for activity)
	indicatorRendered := ""
	indicatorWidth := 0
	if cfg.TitleIndicator != "" {
		indicatorRendered = styles.PanelTitle.Render(cfg.TitleIndicator)
		indicatorWidth = lipgloss.Width(indicatorRendered)
	}

	// Calculate border segments
	// Layout: TopLeft + border + [key] + border + title + border + indicator + border... + TopRight
	leftPadding := 1
	middlePadding := 1 // between key and title
	indicatorPadding := 0
	if indicatorWidth > 0 {
		indicatorPadding = 1 // border segment before indicator
	}
	rightPadding := width - keyWidth - titleWidth - indicatorWidth - leftPadding - middlePadding - indicatorPadding

	if rightPadding < 1 {
		rightPadding = 1
	}

	if keyRendered != "" {
		result := bs.Render(border.TopLeft) +
			bs.Render(strings.Repeat(border.Top, leftPadding)) +
			keyRendered +
			bs.Render(strings.Repeat(border.Top, middlePadding)) +
			titleRendered
		if indicatorRendered != "" {
			result += bs.Render(strings.Repeat(border.Top, indicatorPadding)) + indicatorRendered
		}
		result += bs.Render(strings.Repeat(border.Top, rightPadding)) + bs.Render(border.TopRight)
		return result
	}

	result := bs.Render(border.TopLeft) +
		bs.Render(strings.Repeat(border.Top, leftPadding)) +
		titleRendered
	if indicatorRendered != "" {
		result += bs.Render(strings.Repeat(border.Top, indicatorPadding)) + indicatorRendered
	}
	result += bs.Render(strings.Repeat(border.Top, rightPadding+middlePadding)) + bs.Render(border.TopRight)
	return result
}

func buildTabbedTopBorder(border lipgloss.Border, width int, borderColor lipgloss.Color, styles Styles, cfg BorderConfig) string {
	bs := lipgloss.NewStyle().Foreground(borderColor)

	// Build key prefix (e.g., "[3]")
	keyRendered := ""
	keyWidth := 0
	if cfg.PanelKey != "" {
		keyRendered = styles.PanelTitle.Render("[" + cfg.PanelKey + "]")
		keyWidth = lipgloss.Width(keyRendered)
	}

	// Build tab string with separators
	var tabParts []string
	for i, tab := range cfg.Tabs {
		style := styles.Muted
		if i == cfg.ActiveTab {
			style = styles.SelectedItem
		}
		tabParts = append(tabParts, style.Render(tab))
		if i < len(cfg.Tabs)-1 {
			tabParts = append(tabParts, bs.Render(border.Top))
		}
	}
	tabString := strings.Join(tabParts, "")
	tabWidth := lipgloss.Width(tabString)

	// Calculate border segments
	// Layout: TopLeft + border + [key] + border + tabs + border... + TopRight
	leftPadding := 1   // after TopLeft
	middlePadding := 1 // between key and tabs
	remainingWidth := width - keyWidth - tabWidth - leftPadding - middlePadding

	if remainingWidth < 0 {
		remainingWidth = 0
	}

	if keyRendered != "" {
		return bs.Render(border.TopLeft) +
			bs.Render(strings.Repeat(border.Top, leftPadding)) +
			keyRendered +
			bs.Render(strings.Repeat(border.Top, middlePadding)) +
			tabString +
			bs.Render(strings.Repeat(border.Top, remainingWidth)) +
			bs.Render(border.TopRight)
	}

	return bs.Render(border.TopLeft) +
		bs.Render(strings.Repeat(border.Top, leftPadding)) +
		tabString +
		bs.Render(strings.Repeat(border.Top, remainingWidth+middlePadding)) +
		bs.Render(border.TopRight)
}

func buildBottomBorder(border lipgloss.Border, width int, borderColor lipgloss.Color, styles Styles, cfg BorderConfig) string {
	bs := lipgloss.NewStyle().Foreground(borderColor)
	// Plain bottom border
	return bs.Render(border.BottomLeft + strings.Repeat(border.Bottom, width) + border.BottomRight)
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
