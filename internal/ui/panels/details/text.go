package details

import (
	"github.com/kluzzebass/lazyhue/internal/ui"
)

// TextType indicates what kind of text content this is.
type TextType int

const (
	TextPlain  TextType = iota // Raw text, no formatting
	TextHeader                 // Section header with styling
	TextBlank                  // Empty line
)

// TextSection renders simple text content.
type TextSection struct {
	content  string
	textType TextType
}

// Header creates a section header.
func Header(title string) *TextSection {
	return &TextSection{content: title, textType: TextHeader}
}

// Text creates a plain text section.
func Text(content string) *TextSection {
	return &TextSection{content: content, textType: TextPlain}
}

// Blank creates a blank line.
func Blank() *TextSection {
	return &TextSection{textType: TextBlank}
}

// CollectHints returns empty hints (text doesn't affect alignment).
func (s *TextSection) CollectHints() AlignmentHints {
	return AlignmentHints{}
}

// Render outputs the text content.
func (s *TextSection) Render(hints AlignmentHints, styles ui.Styles) string {
	switch s.textType {
	case TextHeader:
		return "\n" + styles.Subtitle.Render(s.content+":") + "\n"
	case TextBlank:
		return "\n"
	default:
		return s.content
	}
}
