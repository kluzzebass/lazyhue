package details

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/kluzzebass/lazyhue/internal/ui"
)

// ListItem represents a single bullet point.
type ListItem struct {
	Bullet      string         // "•", "●", "○", or custom (e.g., colored light indicator)
	BulletStyle lipgloss.Style // Style for the bullet
	Text        string         // Item text
	TextStyle   lipgloss.Style // Optional style for the text
	Suffix      string         // Optional suffix (e.g., " (50%, xy: 0.5, 0.3)")
	Indent      int            // 0 = normal (2 spaces), 1 = nested (4 spaces)
}

// ListSection renders bullet point lists.
type ListSection struct {
	items []ListItem
}

// NewList creates a new list section.
func NewList() *ListSection {
	return &ListSection{}
}

// Add appends a simple bullet item with default bullet.
func (s *ListSection) Add(text string) *ListSection {
	s.items = append(s.items, ListItem{Bullet: "•", Text: text})
	return s
}

// AddStyled appends an item with styled text.
func (s *ListSection) AddStyled(text string, style lipgloss.Style) *ListSection {
	s.items = append(s.items, ListItem{Bullet: "•", Text: text, TextStyle: style})
	return s
}

// AddCustom appends an item with a custom bullet and optional bullet style.
func (s *ListSection) AddCustom(bullet, text string, bulletStyle lipgloss.Style) *ListSection {
	s.items = append(s.items, ListItem{Bullet: bullet, BulletStyle: bulletStyle, Text: text})
	return s
}

// AddCustomFull appends a fully customized item.
func (s *ListSection) AddCustomFull(item ListItem) *ListSection {
	if item.Bullet == "" {
		item.Bullet = "•"
	}
	s.items = append(s.items, item)
	return s
}

// AddNested appends a nested (further indented) item.
func (s *ListSection) AddNested(text string) *ListSection {
	s.items = append(s.items, ListItem{Bullet: "•", Text: text, Indent: 1})
	return s
}

// AddNestedCustom appends a nested item with custom bullet.
func (s *ListSection) AddNestedCustom(bullet, text string, bulletStyle lipgloss.Style) *ListSection {
	s.items = append(s.items, ListItem{Bullet: bullet, BulletStyle: bulletStyle, Text: text, Indent: 1})
	return s
}

// IsEmpty returns true if there are no items.
func (s *ListSection) IsEmpty() bool {
	return len(s.items) == 0
}

// CollectHints returns empty hints (lists don't affect label alignment).
func (s *ListSection) CollectHints() AlignmentHints {
	return AlignmentHints{}
}

// Render outputs all list items.
func (s *ListSection) Render(hints AlignmentHints, styles ui.Styles) string {
	if len(s.items) == 0 {
		return ""
	}

	var out strings.Builder
	for _, item := range s.items {
		// Determine indentation
		indent := "  " // 2 spaces for normal items
		if item.Indent > 0 {
			indent = "    " // 4 spaces for nested items
		}

		// Build bullet
		var bulletStr string
		if item.BulletStyle.Value() != "" {
			bulletStr = item.BulletStyle.Render(item.Bullet)
		} else {
			bulletStr = item.Bullet
		}

		// Build text
		var textStr string
		if item.TextStyle.Value() != "" {
			textStr = item.TextStyle.Render(item.Text)
		} else {
			textStr = item.Text
		}

		// Add suffix if present
		if item.Suffix != "" {
			textStr += item.Suffix
		}

		out.WriteString(fmt.Sprintf("%s%s %s\n", indent, bulletStr, textStr))
	}
	return out.String()
}
