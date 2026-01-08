package details

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/kluzzebass/lazyhue/internal/ui"
)

// Field represents a single key-value pair.
type Field struct {
	Label      string
	Value      string
	ValueStyle lipgloss.Style // Optional custom style for value
	Indent     int            // 0 = normal (2 spaces), 1 = sub-field (6 spaces)
}

// FieldsSection renders aligned key-value pairs.
type FieldsSection struct {
	fields []Field
}

// NewFields creates a new fields section.
func NewFields() *FieldsSection {
	return &FieldsSection{}
}

// Add appends a regular field.
func (s *FieldsSection) Add(label, value string) *FieldsSection {
	s.fields = append(s.fields, Field{Label: label, Value: value})
	return s
}

// AddMuted appends a field (same as Add - muted styling removed).
func (s *FieldsSection) AddMuted(label, value string) *FieldsSection {
	return s.Add(label, value)
}

// AddStyled appends a field with a custom-styled value.
func (s *FieldsSection) AddStyled(label, value string, style lipgloss.Style) *FieldsSection {
	s.fields = append(s.fields, Field{Label: label, Value: value, ValueStyle: style})
	return s
}

// AddSub appends a sub-field with extra indentation.
func (s *FieldsSection) AddSub(label, value string) *FieldsSection {
	s.fields = append(s.fields, Field{Label: label, Value: value, Indent: 1})
	return s
}

// AddSubMuted appends a sub-field (same as AddSub - muted styling removed).
func (s *FieldsSection) AddSubMuted(label, value string) *FieldsSection {
	return s.AddSub(label, value)
}

// IsEmpty returns true if there are no fields.
func (s *FieldsSection) IsEmpty() bool {
	return len(s.fields) == 0
}

// CollectHints reports the maximum label width.
func (s *FieldsSection) CollectHints() AlignmentHints {
	hints := AlignmentHints{}
	for _, f := range s.fields {
		if len(f.Label) > hints.LabelWidth {
			hints.LabelWidth = len(f.Label)
		}
	}
	return hints
}

// Render outputs all fields with aligned labels.
func (s *FieldsSection) Render(hints AlignmentHints, styles ui.Styles) string {
	if len(s.fields) == 0 {
		return ""
	}

	var out strings.Builder
	for _, f := range s.fields {
		// Determine indentation
		indent := "  " // 2 spaces for normal fields
		if f.Indent > 0 {
			indent = "      " // 6 spaces for sub-fields
		}

		// Build label with colon, then pad to align values
		labelWithColon := f.Label + ":"
		padding := hints.LabelWidth - len(f.Label)
		if padding < 0 {
			padding = 0
		}
		// Pad after colon, then single space before value
		labelStr := labelWithColon + strings.Repeat(" ", padding) + " "

		// Apply value styling (label styling removed - all labels same color)
		var valueStr string
		if f.ValueStyle.Value() != "" {
			valueStr = f.ValueStyle.Render(f.Value)
		} else {
			valueStr = f.Value
		}

		out.WriteString(fmt.Sprintf("%s%s%s\n", indent, labelStr, valueStr))
	}
	return out.String()
}
