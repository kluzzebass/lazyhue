package panels

import (
	"fmt"

	"github.com/kluzzebass/lazyhue/internal/hue"
	"github.com/kluzzebass/lazyhue/internal/ui"
)

// Header renders the top header bar.
type Header struct {
	styles ui.Styles
	width  int
}

// NewHeader creates a new header.
func NewHeader(styles ui.Styles) *Header {
	return &Header{styles: styles}
}

// SetWidth updates the header width.
func (h *Header) SetWidth(width int) {
	h.width = width
}

// View renders the header.
func (h *Header) View(bridge *hue.Bridge, bridgeCount int) string {
	left := "lazyhue"

	var middle string
	var right string

	if bridge != nil {
		middle = fmt.Sprintf("Bridge: %s (%s)", bridge.Info.Name, bridge.Info.IPAddress)

		switch bridge.Status {
		case hue.StatusConnected:
			right = h.styles.Connected.Render("● Connected")
		case hue.StatusConnecting:
			right = h.styles.Muted.Render("○ Connecting...")
		case hue.StatusPairing:
			right = h.styles.Muted.Render("○ Pairing...")
		case hue.StatusDisconnected:
			right = h.styles.Disconnected.Render("○ Disconnected")
		case hue.StatusError:
			right = h.styles.Error.Render("● Error")
		}

		if bridgeCount > 1 {
			right = fmt.Sprintf("[%d bridges] %s", bridgeCount, right)
		}
	} else {
		middle = "No bridge connected"
		right = h.styles.Muted.Render("○ Discovering...")
	}

	// Simple layout: left | middle | right
	content := fmt.Sprintf("%s │ %s │ %s", left, middle, right)
	return h.styles.Header.Width(h.width).Render(content)
}

