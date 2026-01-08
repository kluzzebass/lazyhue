// Package details provides a structured system for building detail views.
package details

import (
	"strings"

	"github.com/kluzzebass/lazyhue/internal/ui"
)

// AlignmentHints contains layout hints collected from sections.
type AlignmentHints struct {
	LabelWidth int // Maximum label width for field alignment
}

// Merge combines hints, taking the maximum values.
func (h AlignmentHints) Merge(other AlignmentHints) AlignmentHints {
	if other.LabelWidth > h.LabelWidth {
		h.LabelWidth = other.LabelWidth
	}
	return h
}

// Section is the interface all section types implement.
type Section interface {
	// CollectHints reports alignment requirements for this section.
	CollectHints() AlignmentHints

	// Render outputs the section content using provided hints.
	Render(hints AlignmentHints, styles ui.Styles) string
}

// View holds sections and renders them with consistent alignment.
type View struct {
	sections []Section
	styles   ui.Styles
}

// NewView creates a new details view.
func NewView(styles ui.Styles) *View {
	return &View{styles: styles}
}

// Add appends a section to the view.
func (v *View) Add(s Section) *View {
	v.sections = append(v.sections, s)
	return v
}

// AddAll appends multiple sections to the view.
func (v *View) AddAll(sections ...Section) *View {
	v.sections = append(v.sections, sections...)
	return v
}

// Render outputs all sections with consistent alignment.
func (v *View) Render() string {
	if len(v.sections) == 0 {
		return ""
	}

	// Phase 1: Collect hints from all sections
	hints := AlignmentHints{LabelWidth: 0} // No minimum - use actual max label width
	for _, s := range v.sections {
		hints = hints.Merge(s.CollectHints())
	}

	// Phase 2: Render all sections with consistent hints
	var out strings.Builder
	for _, s := range v.sections {
		out.WriteString(s.Render(hints, v.styles))
	}
	return out.String()
}

// Styles returns the view's styles for use in section builders.
func (v *View) Styles() ui.Styles {
	return v.styles
}
