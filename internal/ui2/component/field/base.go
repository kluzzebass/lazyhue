package field

import (
	"github.com/kluzzebass/lazyhue/internal/ui2"
	"github.com/kluzzebass/lazyhue/internal/ui2/component"
	zone "github.com/lrstanley/bubblezone/v2"
)

// BaseField provides common functionality for all field components.
// Embed this in concrete field types.
type BaseField struct {
	*component.BaseComponent

	// Field identification
	ID    string
	Label string

	// Field state
	ReadOnly bool
	Editing  bool // Whether the field is in edit mode

	// Shared resources
	Styles *ui2.Styles
	Zones  *zone.Manager

	// Label width for alignment (set by parent form)
	MaxLabelWidth int
}

// NewBaseField creates a new base field with the given ID and label.
func NewBaseField(id, label string, styles *ui2.Styles, zones *zone.Manager) *BaseField {
	base := component.NewBaseComponent()
	base.SetFocusState(component.FocusPassive)

	return &BaseField{
		BaseComponent: base,
		ID:            id,
		Label:         label,
		Styles:        styles,
		Zones:         zones,
	}
}

// ZoneID returns the bubblezone ID for this field.
func (b *BaseField) ZoneID() string {
	return "field-" + b.ID
}

// IsEditing returns whether the field is in edit mode.
func (b *BaseField) IsEditing() bool {
	return b.Editing
}

// SetEditing sets the edit mode state.
func (b *BaseField) SetEditing(editing bool) {
	b.Editing = editing
}

// SetMaxLabelWidth sets the label width for alignment.
func (b *BaseField) SetMaxLabelWidth(width int) {
	b.MaxLabelWidth = width
}

// FieldLabel returns the field's label text.
func (b *BaseField) FieldLabel() string {
	return b.Label
}

// FieldHeight returns how many rows this field takes up.
// Override in multi-row fields like HSL, RGB, ColorWheel.
func (b *BaseField) FieldHeight() int {
	return 1
}

// ControlRenderer is an interface for fields that support separate control rendering.
type ControlRenderer interface {
	// ViewControl renders only the control portion (no label).
	ViewControl() string
	// FieldLabel returns the field's label text.
	FieldLabel() string
	// FieldHeight returns how many rows this field takes up.
	FieldHeight() int
}

// Option represents a selectable option for dropdowns and radio buttons.
type Option struct {
	Label string
	Value int
}
