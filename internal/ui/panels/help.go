// Package panels provides UI panel components.
package panels

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/kluzzebass/lazyhue/internal/ui"
)

// HelpContext indicates which panel is focused for contextual help.
type HelpContext int

const (
	HelpContextGlobal HelpContext = iota
	HelpContextBridges
	HelpContextScenes
	HelpContextGroups
	HelpContextLights
	HelpContextDevices
	HelpContextDetail
)

// HelpPanel displays a keybinding reference overlay.
type HelpPanel struct {
	styles       ui.Styles
	keys         ui.KeyMap
	context      HelpContext
	visible      bool
	scroll       int
	lines        []string
	screenWidth  int
	screenHeight int
}

// NewHelpPanel creates a new help panel.
func NewHelpPanel(styles ui.Styles, keys ui.KeyMap) *HelpPanel {
	return &HelpPanel{
		styles: styles,
		keys:   keys,
	}
}

// SetContext sets the help context for contextual bindings.
func (h *HelpPanel) SetContext(ctx HelpContext) {
	h.context = ctx
	h.buildLines()
}

// Toggle toggles the help panel visibility.
func (h *HelpPanel) Toggle() {
	h.visible = !h.visible
	h.scroll = 0
	if h.visible {
		h.buildLines()
	}
}

// Hide hides the help panel.
func (h *HelpPanel) Hide() {
	h.visible = false
	h.scroll = 0
}

// IsVisible returns whether the help panel is visible.
func (h *HelpPanel) IsVisible() bool {
	return h.visible
}

func (h *HelpPanel) maxScroll() int {
	return max(0, len(h.lines)-h.viewHeight())
}

// Update handles input for the help panel.
func (h *HelpPanel) Update(msg tea.Msg) {
	if !h.visible {
		return
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if h.scroll > 0 {
				h.scroll--
			}
		case "down", "j":
			if h.scroll < h.maxScroll() {
				h.scroll++
			}
		case "home", "g":
			h.scroll = 0
		case "end", "G":
			h.scroll = h.maxScroll()
		case "pgup":
			h.scroll = max(0, h.scroll-h.viewHeight())
		case "pgdown":
			h.scroll = min(h.maxScroll(), h.scroll+h.viewHeight())
		}
	case tea.MouseMsg:
		switch msg.Type {
		case tea.MouseWheelUp:
			if h.scroll > 0 {
				h.scroll--
			}
		case tea.MouseWheelDown:
			if h.scroll < h.maxScroll() {
				h.scroll++
			}
		}
	}
}

const helpPanelScreenRatio = 0.5 // 50% of screen

func (h *HelpPanel) viewHeight() int {
	if h.screenHeight <= 0 {
		return 1
	}
	vh := int(float64(h.screenHeight) * helpPanelScreenRatio)
	return max(1, vh-2) // -2 for borders
}

// Width returns the width of the help panel.
func (h *HelpPanel) Width() int {
	if h.screenWidth <= 0 {
		return 1
	}
	return max(1, int(float64(h.screenWidth)*helpPanelScreenRatio))
}

// Height returns the height of the help panel.
func (h *HelpPanel) Height() int {
	return h.viewHeight() + 2 // content + top/bottom border
}

// View renders the help panel as a complete string with no external dependencies.
func (h *HelpPanel) View() string {
	if !h.visible {
		return ""
	}

	width := h.Width()
	contentWidth := width - 4 // -2 for borders, -2 for padding
	viewHeight := h.viewHeight()

	// Clamp scroll
	if h.scroll > h.maxScroll() {
		h.scroll = h.maxScroll()
	}

	// Get visible lines
	endLine := min(h.scroll+viewHeight, len(h.lines))
	startLine := h.scroll

	// Border characters
	tl, tr, bl, br := "╭", "╮", "╰", "╯"
	horiz, vert := "─", "│"

	// Colors
	borderFg := h.styles.Theme.Primary

	// Build top border with centered title
	title := " Keybindings "
	titleLen := len(title)
	leftPad := max(0, (width-2-titleLen)/2)
	rightPad := max(0, width-2-titleLen-leftPad)
	topBorder := colorize(tl, borderFg) +
		colorize(strings.Repeat(horiz, leftPad), borderFg) +
		colorize(title, h.styles.Theme.Accent) +
		colorize(strings.Repeat(horiz, rightPad), borderFg) +
		colorize(tr, borderFg)

	// Build content lines with scroll indicator
	var contentLines []string
	for i := 0; i < viewHeight; i++ {
		lineIdx := startLine + i
		var lineContent string
		if lineIdx < endLine {
			lineContent = h.lines[lineIdx]
		}
		// Pad to content width
		lineContent = padRight(lineContent, contentWidth)

		// Right border with scroll indicator (same style as other panels)
		var rightBorderStr string
		if h.maxScroll() > 0 {
			thumbPos, thumbSize := h.scrollThumb(viewHeight)
			if i >= thumbPos && i < thumbPos+thumbSize {
				rightBorderStr = colorize("┃", borderFg) // Scroll thumb
			} else {
				rightBorderStr = colorize(vert, borderFg)
			}
		} else {
			rightBorderStr = colorize(vert, borderFg)
		}

		contentLines = append(contentLines,
			colorize(vert, borderFg)+" "+lineContent+" "+rightBorderStr)
	}

	// Build bottom border with hints
	closeHint := " Esc:close "
	scrollHint := fmt.Sprintf(" %d of %d ", h.scroll+1, max(1, len(h.lines)))
	middleLen := width - 2 - len(closeHint) - len(scrollHint)
	if middleLen < 0 {
		middleLen = 0
	}
	bottomBorder := colorize(bl, borderFg) +
		colorize(closeHint, h.styles.Theme.Muted) +
		colorize(strings.Repeat(horiz, middleLen), borderFg) +
		colorize(scrollHint, h.styles.Theme.Muted) +
		colorize(br, borderFg)

	// Combine all
	var result []string
	result = append(result, topBorder)
	result = append(result, contentLines...)
	result = append(result, bottomBorder)

	return strings.Join(result, "\n")
}

func (h *HelpPanel) scrollThumb(viewHeight int) (pos int, size int) {
	total := len(h.lines)
	if total <= viewHeight {
		return 0, viewHeight
	}

	// Thumb size proportional to visible portion
	size = max(1, viewHeight*viewHeight/total)

	// Thumb position
	scrollRange := total - viewHeight
	posRange := viewHeight - size
	if scrollRange > 0 {
		pos = h.scroll * posRange / scrollRange
	}
	return pos, size
}

func colorize(s string, color lipgloss.Color) string {
	return lipgloss.NewStyle().Foreground(color).Render(s)
}

func padRight(s string, width int) string {
	sLen := lipgloss.Width(s)
	if sLen >= width {
		return s
	}
	return s + strings.Repeat(" ", width-sLen)
}

func (h *HelpPanel) buildLines() {
	h.lines = nil

	keyColor := h.styles.Theme.Accent
	sepColor := h.styles.Theme.Muted

	// Local (contextual) section
	contextName, contextBindings := h.getContextualBindings()
	if len(contextBindings) > 0 {
		h.lines = append(h.lines, colorize(fmt.Sprintf("── %s ──", contextName), sepColor))
		for _, b := range contextBindings {
			h.lines = append(h.lines, " "+colorize(padRight(b[0], 7), keyColor)+b[1])
		}
		h.lines = append(h.lines, "")
	}

	// Global section
	h.lines = append(h.lines, colorize("── Global ──", sepColor))
	globalBindings := [][2]string{
		{"↑/k", "Move up"},
		{"↓/j", "Move down"},
		{"←/h", "Left / prev tab"},
		{"→/l", "Right / next tab"},
		{"g", "Top"},
		{"G", "Bottom"},
		{"Tab", "Next panel"},
		{"0-5", "Focus panel"},
		{"", ""},
		{"Space", "Toggle on/off"},
		{"o/O", "Turn on/off"},
		{"+/-", "Brightness"},
		{"Enter", "Select / activate"},
		{"", ""},
		{"[/]", "Prev/next bridge"},
		{"P", "Pair bridge"},
		{"R", "Refresh"},
		{"q", "Quit"},
	}

	for _, b := range globalBindings {
		if b[0] == "" {
			h.lines = append(h.lines, "")
		} else {
			h.lines = append(h.lines, " "+colorize(padRight(b[0], 7), keyColor)+b[1])
		}
	}
}

func (h *HelpPanel) getContextualBindings() (string, [][2]string) {
	switch h.context {
	case HelpContextLights:
		return "Lights", [][2]string{
			{"Space", "Toggle light"},
			{"+/-", "Brightness"},
		}
	case HelpContextGroups:
		return "Groups", [][2]string{
			{"Space", "Toggle room"},
			{"←/→", "Switch tab"},
		}
	case HelpContextScenes:
		return "Scenes", [][2]string{
			{"Enter", "Activate scene"},
			{"Space", "Expand/collapse"},
		}
	case HelpContextBridges:
		return "Bridges", [][2]string{
			{"Enter", "Select bridge"},
			{"P", "Pair bridge"},
		}
	case HelpContextDetail:
		return "Details", [][2]string{
			{"Esc", "Back to panel"},
		}
	default:
		return "", nil
	}
}

// SetSize updates the screen dimensions for responsive sizing.
func (h *HelpPanel) SetSize(width, height int) {
	h.screenWidth = width
	h.screenHeight = height
}
