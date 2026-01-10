package panels

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/progress"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/kluzzebass/lazyhue/internal/ui"
)

// PairingCancelledMsg is sent when user cancels pairing.
type PairingCancelledMsg struct {
	BridgeID string
}

// PairingPanel shows a pairing dialog with countdown.
type PairingPanel struct {
	styles       ui.Styles
	visible      bool
	bridgeID     string
	bridgeName   string
	startTime    time.Time
	timeout      time.Duration
	screenWidth  int
	screenHeight int
	progress     progress.Model
}

// NewPairingPanel creates a new pairing panel.
func NewPairingPanel(styles ui.Styles) *PairingPanel {
	prog := progress.New(
		progress.WithSolidFill("#00ff00"), // Start green, will be updated dynamically
		progress.WithoutPercentage(),
		progress.WithFillCharacters('█', '░'),
	)
	prog.EmptyColor = "#333333" // Dark empty portion
	return &PairingPanel{
		styles:   styles,
		timeout:  60 * time.Second,
		progress: prog,
	}
}

// percentToColor returns a color transitioning from green (1.0) through yellow (0.5) to red (0.0).
func percentToColor(percent float64) string {
	// Clamp percent
	if percent < 0 {
		percent = 0
	}
	if percent > 1 {
		percent = 1
	}

	var r, g int
	if percent > 0.5 {
		// Green to yellow (0.5-1.0): increase red from 0 to 255
		t := (1.0 - percent) * 2 // 0 at 100%, 1 at 50%
		r = int(255 * t)
		g = 255
	} else {
		// Yellow to red (0-0.5): decrease green from 255 to 0
		t := percent * 2 // 1 at 50%, 0 at 0%
		r = 255
		g = int(255 * t)
	}

	return fmt.Sprintf("#%02x%02x00", r, g)
}

// Show displays the pairing panel.
func (p *PairingPanel) Show(bridgeID, bridgeName string) {
	p.bridgeID = bridgeID
	p.bridgeName = bridgeName
	p.startTime = time.Now()
	p.visible = true

	// Create a fresh progress bar to reset all animation state
	p.progress = progress.New(
		progress.WithSolidFill("#00ff00"),
		progress.WithoutPercentage(),
		progress.WithFillCharacters('█', '░'),
	)
	p.progress.EmptyColor = "#333333"
	p.updateProgressWidth()
}

// Hide hides the pairing panel.
func (p *PairingPanel) Hide() {
	p.visible = false
}

// IsVisible returns whether the panel is visible.
func (p *PairingPanel) IsVisible() bool {
	return p.visible
}

// SetSize updates screen dimensions.
func (p *PairingPanel) SetSize(width, height int) {
	p.screenWidth = width
	p.screenHeight = height
	p.updateProgressWidth()
}

func (p *PairingPanel) updateProgressWidth() {
	// Progress bar width = content width - time display width (" XXs" = 4 chars)
	contentWidth := p.Width() - 4 // borders + padding
	barWidth := contentWidth - 4  // time display
	if barWidth > 0 {
		p.progress.Width = barWidth
	}
}

// RemainingTime returns seconds remaining.
func (p *PairingPanel) RemainingTime() int {
	elapsed := time.Since(p.startTime)
	remaining := p.timeout - elapsed
	if remaining < 0 {
		return 0
	}
	return int(remaining.Seconds())
}

// Update handles input and progress animation.
func (p *PairingPanel) Update(msg tea.Msg) tea.Cmd {
	if !p.visible {
		return nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "esc" {
			p.Hide()
			return func() tea.Msg {
				return PairingCancelledMsg{BridgeID: p.bridgeID}
			}
		}
		return nil

	case progress.FrameMsg:
		// Handle progress animation frames
		progressModel, cmd := p.progress.Update(msg)
		p.progress = progressModel.(progress.Model)
		return cmd

	default:
		// Any other message (like tick) - update progress based on elapsed time
		remaining := p.RemainingTime()
		percent := float64(remaining) / p.timeout.Seconds()
		// Update color based on percentage (green -> yellow -> red)
		p.progress.FullColor = percentToColor(percent)
		cmd := p.progress.SetPercent(percent)
		return cmd
	}
}

// Width returns the popup width.
func (p *PairingPanel) Width() int {
	w := p.screenWidth - 4
	if w < 20 {
		w = 20 // Minimum width
	}
	return min(50, w)
}

// Height returns the popup height.
func (p *PairingPanel) Height() int {
	return 7
}

// View renders the pairing panel.
func (p *PairingPanel) View() string {
	if !p.visible {
		return ""
	}

	width := p.Width()
	contentWidth := width - 4 // borders + padding

	remaining := p.RemainingTime()

	// Border styling
	tl, tr, bl, br := "╭", "╮", "╰", "╯"
	horiz, vert := "─", "│"
	borderFg := p.styles.Theme.Primary

	// Title
	title := "Pairing"
	titleLen := len(title) + 2
	leftPad := max(0, (width-2-titleLen)/2)
	rightPad := max(0, width-2-titleLen-leftPad)
	topBorder := ui.Colorize(tl, borderFg) +
		ui.Colorize(strings.Repeat(horiz, leftPad), borderFg) +
		ui.Colorize(" "+title+" ", p.styles.Theme.Accent) +
		ui.Colorize(strings.Repeat(horiz, rightPad), borderFg) +
		ui.Colorize(tr, borderFg)

	// Content
	var lines []string

	// Message
	msg := fmt.Sprintf("Press the link button on %s", p.bridgeName)
	if len(msg) > contentWidth {
		msg = msg[:contentWidth-3] + "..."
	}
	lines = append(lines, p.padLine(msg, contentWidth))

	// Empty line
	lines = append(lines, strings.Repeat(" ", contentWidth))

	// Progress bar with time display
	timeStr := fmt.Sprintf(" %2ds", remaining)
	progressLine := p.progress.View() + p.styles.Muted.Render(timeStr)
	lines = append(lines, progressLine)

	// Empty line
	lines = append(lines, strings.Repeat(" ", contentWidth))

	// Wrap with borders
	var wrappedLines []string
	for _, line := range lines {
		wrappedLines = append(wrappedLines,
			ui.Colorize(vert, borderFg)+" "+line+" "+ui.Colorize(vert, borderFg))
	}

	// Bottom border with hint
	hint := ui.Colorize(" Esc:cancel ", p.styles.Theme.Muted)
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

func (p *PairingPanel) padLine(s string, width int) string {
	sLen := lipgloss.Width(s)
	if sLen >= width {
		return s
	}
	return s + strings.Repeat(" ", width-sLen)
}
