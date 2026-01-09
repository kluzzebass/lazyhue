package panels

import (
	"fmt"
	"strings"
	"sync"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/kluzzebass/lazyhue/internal/ui"
)

// LogEntry represents a single log entry.
type LogEntry struct {
	Time           time.Time
	Type           string  // "event", "request", "response", "error"
	Message        string
	ResourceType   string  // For events: "light", "scene", etc.
	ResourceName   string  // For events: the human-readable name (styled differently)
	Details        string  // For events: "on 50%", "activated", etc.
	IndicatorColor string  // Hex color for brightness indicator (lights)
	Brightness     float64 // Brightness 0-100 for indicator
	IsOn           bool    // Whether the light/entity is on
}

// LogPanel displays a rolling log of events and requests.
type LogPanel struct {
	styles     ui.Styles
	width      int
	height     int
	entries    []LogEntry
	offset     int // scroll offset
	maxSize    int // max entries to keep
	panelKey   string
	panelTitle string
	mu         sync.Mutex
}

// NewLogPanel creates a new log panel.
func NewLogPanel(styles ui.Styles) *LogPanel {
	return &LogPanel{
		styles:     styles,
		entries:    make([]LogEntry, 0, 100),
		maxSize:    100, // Keep last 100 entries
		panelKey:   "3",
		panelTitle: "Activity",
	}
}

// SetSize sets the panel dimensions.
func (p *LogPanel) SetSize(width, height int) {
	p.width = width
	p.height = height
}

// AddEntry adds a new log entry.
func (p *LogPanel) AddEntry(entryType, message string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	entry := LogEntry{
		Time:    time.Now(),
		Type:    entryType,
		Message: message,
	}

	// If scrolled away from top, increment offset to keep current view stable
	if p.offset > 0 {
		p.offset++
	}

	p.entries = append(p.entries, entry)

	// Trim if too many entries
	if len(p.entries) > p.maxSize {
		// When trimming, we lose old entries - adjust offset if needed
		p.entries = p.entries[len(p.entries)-p.maxSize:]
		// Clamp offset to valid range
		maxOff := p.maxOffset()
		if p.offset > maxOff {
			p.offset = maxOff
		}
	}
}

// EventDetails contains rich information about an SSE event.
type EventDetails struct {
	ResourceType   string  // "light", "scene", "motion", etc.
	ResourceName   string  // Human-readable name
	EventType      string  // "update", "add", "delete"
	Details        string  // What changed: "on", "off", "50%", "activated", "motion detected"
	IndicatorColor string  // Hex color for brightness indicator (lights)
	Brightness     float64 // Brightness 0-100 for indicator
	IsOn           bool    // Whether the light/entity is on
}

// AddEvent adds an SSE event entry with rich details.
func (p *LogPanel) AddEvent(details EventDetails) {
	p.mu.Lock()
	defer p.mu.Unlock()

	entry := LogEntry{
		Time:           time.Now(),
		Type:           "event",
		ResourceType:   details.ResourceType,
		ResourceName:   details.ResourceName,
		Details:        details.Details,
		IndicatorColor: details.IndicatorColor,
		Brightness:     details.Brightness,
		IsOn:           details.IsOn,
	}

	// If scrolled away from top, increment offset to keep current view stable
	if p.offset > 0 {
		p.offset++
	}

	p.entries = append(p.entries, entry)

	// Trim if too many entries
	if len(p.entries) > p.maxSize {
		p.entries = p.entries[len(p.entries)-p.maxSize:]
		maxOff := p.maxOffset()
		if p.offset > maxOff {
			p.offset = maxOff
		}
	}
}

// AddRequest adds an API request entry.
func (p *LogPanel) AddRequest(method, endpoint string) {
	msg := fmt.Sprintf("%s %s", method, endpoint)
	p.AddEntry("request", msg)
}

// AddResponse adds an API response entry.
func (p *LogPanel) AddResponse(status int, duration time.Duration) {
	msg := fmt.Sprintf("%d (%s)", status, duration.Round(time.Millisecond))
	p.AddEntry("response", msg)
}

// AddError adds an error entry.
func (p *LogPanel) AddError(err error) {
	p.AddEntry("error", err.Error())
}

func (p *LogPanel) maxOffset() int {
	viewHeight := p.height - 2
	if viewHeight < 1 {
		viewHeight = 1
	}
	maxOff := len(p.entries) - viewHeight
	if maxOff < 0 {
		maxOff = 0
	}
	return maxOff
}

// ScrollUp scrolls the log up by n lines (to see older entries).
func (p *LogPanel) ScrollUp(n int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	maxOffset := p.maxOffset()
	p.offset += n
	if p.offset > maxOffset {
		p.offset = maxOffset
	}
}

// ScrollDown scrolls the log down by n lines (to see newer entries).
func (p *LogPanel) ScrollDown(n int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.offset -= n
	if p.offset < 0 {
		p.offset = 0
	}
}

// ScrollToTop scrolls to oldest entries (top of log).
func (p *LogPanel) ScrollToTop() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.offset = p.maxOffset()
}

// ScrollToEnd scrolls to newest entries (bottom of log).
func (p *LogPanel) ScrollToEnd() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.offset = 0
}

// View renders the log panel.
func (p *LogPanel) View(active bool) string {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Calculate content dimensions (border takes 2 chars width, 2 chars height)
	contentWidth := p.width - 2
	contentHeight := p.height - 2

	if contentWidth < 1 {
		contentWidth = 1
	}
	if contentHeight < 1 {
		contentHeight = 1
	}

	// Build visible entries (chronological order - oldest at top, newest at bottom)
	var lines []string
	viewHeight := contentHeight
	totalEntries := len(p.entries)

	// Calculate which entries to show
	// offset=0 means we're viewing the newest entries (at the end of the slice)
	// higher offset means scrolling up to see older entries
	endIdx := totalEntries - p.offset
	startIdx := endIdx - viewHeight
	if startIdx < 0 {
		startIdx = 0
	}
	if endIdx > totalEntries {
		endIdx = totalEntries
	}

	for i := startIdx; i < endIdx; i++ {
		entry := p.entries[i]
		line := p.renderEntry(entry, contentWidth)
		lines = append(lines, line)
	}

	// Pad to fill height
	for len(lines) < viewHeight {
		lines = append(lines, strings.Repeat(" ", contentWidth))
	}

	content := strings.Join(lines, "\n")

	// Use standard bordered panel rendering
	// For log panel, ItemIndex shows the last visible line number
	lastVisibleIdx := len(p.entries) - p.offset
	if lastVisibleIdx < 0 {
		lastVisibleIdx = 0
	}

	cfg := ui.BorderConfig{
		PanelKey:    p.panelKey,
		Title:       p.panelTitle,
		ItemIndex:   lastVisibleIdx - 1, // 0-based index of last visible
		ItemCount:   len(p.entries),
		ScrollPos:   p.offset,
		TotalHeight: len(p.entries),
		ViewHeight:  contentHeight,
	}

	return ui.RenderBorderedPanel(content, p.width, p.height, active, p.styles, cfg)
}


func (p *LogPanel) renderEntry(entry LogEntry, width int) string {
	// Format: HH:MM:SS [TYPE] message
	timeStr := entry.Time.Format("15:04:05")

	var typeStyle lipgloss.Style
	switch entry.Type {
	case "event":
		typeStyle = lipgloss.NewStyle().Foreground(p.styles.Theme.Success)
	case "request":
		typeStyle = lipgloss.NewStyle().Foreground(p.styles.Theme.Secondary)
	case "response":
		typeStyle = lipgloss.NewStyle().Foreground(p.styles.Theme.Muted)
	case "error":
		typeStyle = lipgloss.NewStyle().Foreground(p.styles.Theme.Error)
	default:
		typeStyle = p.styles.Muted
	}

	typeIndicator := typeStyle.Render(string(entry.Type[0]))

	var line string
	if entry.Type == "event" && entry.ResourceType != "" {
		// For events, style the resource name in a faded color
		nameStyle := lipgloss.NewStyle().Foreground(p.styles.Theme.Muted)

		// Build brightness/color indicator for lights
		var indicator string
		if (entry.ResourceType == "light" || entry.ResourceType == "grouped_light") && entry.IsOn {
			indicatorChar := brightnessIndicatorLog(entry.Brightness)
			if entry.IndicatorColor != "" {
				indicatorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(entry.IndicatorColor))
				indicator = indicatorStyle.Render(indicatorChar) + " "
			} else {
				indicator = lipgloss.NewStyle().Foreground(p.styles.Theme.OnColor).Render(indicatorChar) + " "
			}
		} else if (entry.ResourceType == "light" || entry.ResourceType == "grouped_light") && !entry.IsOn {
			indicator = p.styles.Muted.Render("○") + " "
		}

		if entry.ResourceName != "" {
			if entry.Details != "" {
				line = fmt.Sprintf("%s %s %s%s %s → %s",
					p.styles.Muted.Render(timeStr),
					typeIndicator,
					indicator,
					entry.ResourceType,
					nameStyle.Render(entry.ResourceName),
					entry.Details)
			} else {
				line = fmt.Sprintf("%s %s %s%s %s",
					p.styles.Muted.Render(timeStr),
					typeIndicator,
					indicator,
					entry.ResourceType,
					nameStyle.Render(entry.ResourceName))
			}
		} else {
			// No resource name - show details if available
			if entry.Details != "" {
				line = fmt.Sprintf("%s %s %s%s → %s",
					p.styles.Muted.Render(timeStr),
					typeIndicator,
					indicator,
					entry.ResourceType,
					entry.Details)
			} else {
				line = fmt.Sprintf("%s %s %s%s update",
					p.styles.Muted.Render(timeStr),
					typeIndicator,
					indicator,
					entry.ResourceType)
			}
		}
	} else {
		line = fmt.Sprintf("%s %s %s", p.styles.Muted.Render(timeStr), typeIndicator, entry.Message)
	}

	// Truncate if needed
	if lipgloss.Width(line) > width {
		line = ansiTruncateLog(line, width-1) + "…"
	}

	return line
}

// ansiTruncateLog truncates a string with ANSI codes to n visual columns.
func ansiTruncateLog(s string, n int) string {
	if n <= 0 {
		return ""
	}

	var result strings.Builder
	col := 0
	runes := []rune(s)
	i := 0

	for i < len(runes) && col < n {
		if runes[i] == '\x1b' {
			// Consume entire escape sequence
			result.WriteRune(runes[i])
			i++
			for i < len(runes) && !((runes[i] >= 'A' && runes[i] <= 'Z') || (runes[i] >= 'a' && runes[i] <= 'z')) {
				result.WriteRune(runes[i])
				i++
			}
			if i < len(runes) {
				result.WriteRune(runes[i])
				i++
			}
		} else {
			result.WriteRune(runes[i])
			col++
			i++
		}
	}

	return result.String()
}

// Update handles input for the log panel.
func (p *LogPanel) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			p.ScrollUp(1)
		case "down", "j":
			p.ScrollDown(1)
		case "pgup":
			p.ScrollUp(10)
		case "pgdown":
			p.ScrollDown(10)
		case "home", "g":
			p.ScrollToTop()
		case "end", "G":
			p.ScrollToEnd()
		}
	case tea.MouseMsg:
		if msg.Action == tea.MouseActionPress {
			switch msg.Button {
			case tea.MouseButtonWheelUp:
				p.ScrollUp(3)
			case tea.MouseButtonWheelDown:
				p.ScrollDown(3)
			}
		}
	}
	return nil
}

// Title returns the panel title.
func (p *LogPanel) Title() string {
	return "Activity"
}

// brightnessIndicatorLog returns a character representing the brightness level.
// Uses circle fill characters: ○ ◔ ◑ ◕ ●
func brightnessIndicatorLog(brightness float64) string {
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
