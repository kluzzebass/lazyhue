// Package panels provides UI panel components.
package panels

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/kluzzebass/lazyhue/internal/ui"
)

// PanelInfo holds panel key and title for help display.
type PanelInfo struct {
	Key   string
	Title string
}

// HelpItem represents a selectable item in the help panel.
type HelpItem struct {
	RawKey   string    // Unformatted key (e.g., "<enter>")
	Desc     string    // Description
	Action   ui.Action // Action to execute (ActionNone for non-executable)
	IsSep    bool      // Is this a separator/header line
	SepText  string    // Full separator text (pre-formatted)
}

// HelpExecuteMsg is sent when user selects a keybinding to execute.
type HelpExecuteMsg struct {
	Action ui.Action
}

// HelpPanel displays keybindings with selectable items.
type HelpPanel struct {
	styles         ui.Styles
	panelBindings  []ui.Binding
	globalBindings []ui.Binding
	panelTitle     string
	panelKeys      []PanelInfo

	// State
	visible     bool
	items       []HelpItem // Flat list of all items (including separators)
	cursor      int        // Current selection
	scroll      int        // Scroll offset
	maxKeyWidth int        // For formatting keys at render time

	// Sizing
	screenWidth  int
	screenHeight int
}

// NewHelpPanel creates a new help panel.
func NewHelpPanel(styles ui.Styles) *HelpPanel {
	return &HelpPanel{
		styles: styles,
	}
}

// SetPanelBindings sets the focused panel's bindings and title for contextual help.
func (h *HelpPanel) SetPanelBindings(title string, bindings []ui.Binding) {
	h.panelTitle = title
	h.panelBindings = bindings
}

// SetGlobalBindings sets the global bindings to display.
func (h *HelpPanel) SetGlobalBindings(bindings []ui.Binding) {
	h.globalBindings = bindings
}

// SetPanelKeys sets the available panel focus keys.
func (h *HelpPanel) SetPanelKeys(panels []PanelInfo) {
	h.panelKeys = panels
}

// Toggle toggles the help panel visibility.
func (h *HelpPanel) Toggle() {
	if h.visible {
		h.Hide()
	} else {
		h.buildItems()
		h.cursor = 0
		h.scroll = 0
		// Move cursor to first selectable item
		for h.cursor < len(h.items) && h.items[h.cursor].Action == ui.ActionNone {
			h.cursor++
		}
		h.visible = true
	}
}

// Hide hides the help panel.
func (h *HelpPanel) Hide() {
	h.visible = false
}

// IsVisible returns whether the help panel is visible.
func (h *HelpPanel) IsVisible() bool {
	return h.visible
}

// SetSize updates the screen dimensions.
func (h *HelpPanel) SetSize(width, height int) {
	h.screenWidth = width
	h.screenHeight = height
}

// Update handles input for the help panel.
func (h *HelpPanel) Update(msg tea.Msg) tea.Cmd {
	if !h.visible {
		return nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		return h.handleKey(msg)
	case tea.MouseMsg:
		return h.handleMouse(msg)
	}
	return nil
}

func (h *HelpPanel) handleKey(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "esc", "q", "?":
		h.Hide()
		return nil
	case "enter":
		if h.cursor >= 0 && h.cursor < len(h.items) {
			item := h.items[h.cursor]
			if item.Action != ui.ActionNone {
				h.Hide()
				return func() tea.Msg { return HelpExecuteMsg{Action: item.Action} }
			}
		}
	case "up", "k":
		h.moveCursor(-1)
	case "down", "j":
		h.moveCursor(1)
	case "home", "g":
		h.cursor = 0
		for h.cursor < len(h.items) && h.items[h.cursor].Action == ui.ActionNone {
			h.cursor++
		}
		h.ensureVisible()
	case "end", "G":
		h.cursor = len(h.items) - 1
		for h.cursor >= 0 && h.items[h.cursor].Action == ui.ActionNone {
			h.cursor--
		}
		h.ensureVisible()
	case "pgup":
		for i := 0; i < h.viewHeight()-2; i++ {
			h.moveCursor(-1)
		}
	case "pgdown":
		for i := 0; i < h.viewHeight()-2; i++ {
			h.moveCursor(1)
		}
	}
	return nil
}

func (h *HelpPanel) moveCursor(delta int) {
	if len(h.items) == 0 {
		return
	}

	newCursor := h.cursor + delta

	// Skip non-selectable items
	for newCursor >= 0 && newCursor < len(h.items) {
		if h.items[newCursor].Action != ui.ActionNone {
			break
		}
		newCursor += delta
	}

	if newCursor >= 0 && newCursor < len(h.items) && h.items[newCursor].Action != ui.ActionNone {
		h.cursor = newCursor
		h.ensureVisible()
	}
}

func (h *HelpPanel) ensureVisible() {
	viewHeight := h.viewHeight()
	if h.cursor < h.scroll {
		h.scroll = h.cursor
	} else if h.cursor >= h.scroll+viewHeight {
		h.scroll = h.cursor - viewHeight + 1
	}
}

func (h *HelpPanel) handleMouse(msg tea.MouseMsg) tea.Cmd {
	switch msg.Type {
	case tea.MouseWheelUp:
		h.moveCursor(-1)
	case tea.MouseWheelDown:
		h.moveCursor(1)
	}
	return nil
}

func (h *HelpPanel) width() int {
	return max(20, h.screenWidth/2)
}

func (h *HelpPanel) height() int {
	return max(10, h.screenHeight/2)
}

func (h *HelpPanel) viewHeight() int {
	return h.height() - 2 // borders
}

// Width returns the popup width.
func (h *HelpPanel) Width() int {
	return h.width()
}

// Height returns the popup height.
func (h *HelpPanel) Height() int {
	return h.height()
}

func (h *HelpPanel) buildItems() {
	h.items = nil
	sepStyle := lipgloss.NewStyle().Foreground(h.styles.Theme.Muted)

	// Calculate max key width first
	h.maxKeyWidth = 0
	for _, b := range h.panelBindings {
		if w := lipgloss.Width(b.Display); w > h.maxKeyWidth {
			h.maxKeyWidth = w
		}
	}
	for _, b := range h.globalBindings {
		if w := lipgloss.Width(b.Display); w > h.maxKeyWidth {
			h.maxKeyWidth = w
		}
	}
	for _, p := range h.panelKeys {
		if w := lipgloss.Width(p.Key); w > h.maxKeyWidth {
			h.maxKeyWidth = w
		}
	}

	headerPad := strings.Repeat(" ", h.maxKeyWidth+1)

	// Panel-specific bindings
	if len(h.panelBindings) > 0 && h.panelTitle != "" {
		h.items = append(h.items, HelpItem{
			IsSep:   true,
			SepText: headerPad + sepStyle.Render("─── "+h.panelTitle+" ───"),
		})
		for _, b := range h.panelBindings {
			h.items = append(h.items, HelpItem{
				RawKey: b.Display,
				Desc:   b.Desc,
				Action: b.Action,
			})
		}
		h.items = append(h.items, HelpItem{IsSep: true, SepText: ""})
	}

	// Separate navigation from other global bindings
	var navBindings []ui.Binding
	var otherGlobalBindings []ui.Binding
	navActions := map[ui.Action]bool{
		ui.ActionUp:       true,
		ui.ActionDown:     true,
		ui.ActionTop:      true,
		ui.ActionBottom:   true,
		ui.ActionPageUp:   true,
		ui.ActionPageDown: true,
	}
	for _, b := range h.globalBindings {
		if navActions[b.Action] {
			navBindings = append(navBindings, b)
		} else {
			otherGlobalBindings = append(otherGlobalBindings, b)
		}
	}

	// Global bindings (non-navigation)
	h.items = append(h.items, HelpItem{
		IsSep:   true,
		SepText: headerPad + sepStyle.Render("─── Global ───"),
	})
	for _, b := range otherGlobalBindings {
		h.items = append(h.items, HelpItem{
			RawKey: b.Display,
			Desc:   b.Desc,
			Action: b.Action,
		})
	}

	// Panel focus keys
	if len(h.panelKeys) > 0 {
		h.items = append(h.items, HelpItem{IsSep: true, SepText: ""})
		for _, p := range h.panelKeys {
			h.items = append(h.items, HelpItem{
				RawKey: p.Key,
				Desc:   "Focus " + p.Title,
				Action: ui.ActionNone, // Panel focus handled differently
			})
		}
	}

	// Navigation section
	if len(navBindings) > 0 {
		h.items = append(h.items, HelpItem{IsSep: true, SepText: ""})
		h.items = append(h.items, HelpItem{
			IsSep:   true,
			SepText: headerPad + sepStyle.Render("─── Navigation ───"),
		})
		for _, b := range navBindings {
			h.items = append(h.items, HelpItem{
				RawKey: b.Display,
				Desc:   b.Desc,
				Action: b.Action,
			})
		}
	}
}

// View renders the help panel.
func (h *HelpPanel) View() string {
	if !h.visible {
		return ""
	}

	width := h.width()
	contentWidth := width - 4 // borders + padding
	viewHeight := h.viewHeight()

	// Border styling
	tl, tr, bl, br := "╭", "╮", "╰", "╯"
	horiz, vert := "─", "│"
	borderFg := h.styles.Theme.Primary

	// Title
	title := "Keybindings"
	titleLen := len(title) + 2
	leftPad := max(0, (width-2-titleLen)/2)
	rightPad := max(0, width-2-titleLen-leftPad)
	topBorder := ui.Colorize(tl, borderFg) +
		ui.Colorize(strings.Repeat(horiz, leftPad), borderFg) +
		ui.Colorize(" "+title+" ", h.styles.Theme.Accent) +
		ui.Colorize(strings.Repeat(horiz, rightPad), borderFg) +
		ui.Colorize(tr, borderFg)

	// Content lines
	var contentLines []string
	keyStyle := lipgloss.NewStyle().Foreground(h.styles.Theme.Accent)
	selectStyle := lipgloss.NewStyle().Background(h.styles.Theme.Primary).Foreground(h.styles.Theme.Background)

	for i := h.scroll; i < min(h.scroll+viewHeight, len(h.items)); i++ {
		item := h.items[i]
		isSelected := i == h.cursor && item.Action != ui.ActionNone

		var line string
		if item.IsSep {
			// Separator or header - use pre-formatted text
			line = item.SepText
		} else if item.RawKey == "" {
			line = item.Desc
		} else {
			// Format key with proper padding
			keyLen := lipgloss.Width(item.RawKey)
			padding := h.maxKeyWidth - keyLen
			paddedKey := strings.Repeat(" ", padding) + item.RawKey

			if isSelected {
				// Selected: apply selection style to entire line
				line = paddedKey + " " + item.Desc
			} else {
				// Not selected: color the key
				line = keyStyle.Render(paddedKey) + " " + item.Desc
			}
		}

		// Pad to content width
		lineWidth := lipgloss.Width(line)
		if lineWidth < contentWidth {
			line = line + strings.Repeat(" ", contentWidth-lineWidth)
		}

		// Apply selection background to entire line
		if isSelected {
			line = selectStyle.Render(line)
		}

		contentLines = append(contentLines, line)
	}

	// Pad to fill view height
	for len(contentLines) < viewHeight {
		contentLines = append(contentLines, strings.Repeat(" ", contentWidth))
	}

	// Wrap with borders
	var wrappedLines []string
	maxScroll := max(0, len(h.items)-viewHeight)
	for i, line := range contentLines {
		rightBorderStr := ui.Colorize(vert, borderFg)
		// Scroll indicator
		if maxScroll > 0 {
			thumbPos, thumbSize := h.scrollThumb(viewHeight, maxScroll)
			if i >= thumbPos && i < thumbPos+thumbSize {
				rightBorderStr = ui.Colorize("┃", borderFg)
			}
		}
		wrappedLines = append(wrappedLines, ui.Colorize(vert, borderFg)+" "+line+" "+rightBorderStr)
	}

	// Bottom border with hint
	hint := ui.Colorize(" Enter:execute  Esc:close ", h.styles.Theme.Muted)
	hintLen := lipgloss.Width(hint)
	bottomPad := max(0, width-2-hintLen)
	bottomBorder := ui.Colorize(bl, borderFg) +
		ui.Colorize(strings.Repeat(horiz, bottomPad), borderFg) +
		hint +
		ui.Colorize(br, borderFg)

	var result []string
	result = append(result, topBorder)
	result = append(result, wrappedLines...)
	result = append(result, bottomBorder)

	return strings.Join(result, "\n")
}

func (h *HelpPanel) scrollThumb(viewHeight, maxScroll int) (pos, size int) {
	total := len(h.items)
	if total <= viewHeight {
		return 0, viewHeight
	}
	size = max(1, viewHeight*viewHeight/total)
	if maxScroll > 0 {
		pos = h.scroll * (viewHeight - size) / maxScroll
	}
	return pos, size
}
