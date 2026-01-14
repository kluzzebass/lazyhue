package component

import (
	"log/slog"

	"github.com/charmbracelet/bubbles/v2/viewport"
	tea "github.com/charmbracelet/bubbletea/v2"

	"github.com/kluzzebass/lazyhue/internal/ui2/components"
	"github.com/kluzzebass/lazyhue/internal/ui2/panels"
)

// TreePanelComponent wraps a TreePanel to implement the Component interface.
type TreePanelComponent struct {
	*BaseComponent
	panel *panels.TreePanel
	id    string
}

// NewTreePanelComponent creates a new tree panel component.
func NewTreePanelComponent(id string, panel *panels.TreePanel) *TreePanelComponent {
	t := &TreePanelComponent{
		BaseComponent: NewBaseComponent(),
		panel:         panel,
		id:            id,
	}
	t.BaseComponent.SetFocusState(FocusPassive)
	return t
}

// Layout sets the bounds and updates the panel size.
func (t *TreePanelComponent) Layout(bounds Rect) {
	t.BaseComponent.Layout(bounds)
	t.panel.SetSize(bounds.Width, bounds.Height)
	slog.Debug("TreePanelComponent.Layout", "id", t.id, "bounds", bounds)
}

// Update handles events by delegating to the panel.
func (t *TreePanelComponent) Update(msg tea.Msg) (Component, tea.Cmd) {
	slog.Debug("TreePanelComponent.Update", "id", t.id, "msg_type", slog.Any("%T", msg))

	// Delegate to panel
	var cmd tea.Cmd
	t.panel, cmd = t.panel.Update(msg)
	return t, cmd
}

// RouteEvent checks if this component can handle the event.
// Panels always consume key events when focused, but check bounds for mouse events.
func (t *TreePanelComponent) RouteEvent(msg tea.Msg) (bool, tea.Cmd) {
	if !t.IsFocused() {
		slog.Debug("TreePanelComponent.RouteEvent: not focused, ignoring", "id", t.id)
		return false, nil
	}

	slog.Debug("TreePanelComponent.RouteEvent", "id", t.id, "msg_type", slog.Any("%T", msg))

	// For mouse events, check if it's within bounds
	if mouse, ok := msg.(tea.MouseClickMsg); ok {
		if !t.bounds.Contains(mouse.X, mouse.Y) {
			slog.Debug("TreePanelComponent.RouteEvent: mouse outside bounds", "id", t.id)
			return false, nil
		}
	}

	// Update and consume the event
	updated, cmd := t.Update(msg)
	*t = *updated.(*TreePanelComponent)
	return true, cmd
}

// View renders the panel.
func (t *TreePanelComponent) View() string {
	return t.panel.View(t.IsFocused())
}

// Panel returns the underlying TreePanel.
func (t *TreePanelComponent) Panel() *panels.TreePanel {
	return t.panel
}

// FormComponent wraps a Form to implement the Component interface.
type FormComponent struct {
	*BaseComponent
	form *components.Form
	id   string
}

// NewFormComponent creates a new form component.
func NewFormComponent(id string, form *components.Form) *FormComponent {
	f := &FormComponent{
		BaseComponent: NewBaseComponent(),
		form:          form,
		id:            id,
	}
	f.BaseComponent.SetFocusState(FocusPassive)
	return f
}

// Layout sets the bounds (form handles its own sizing).
func (f *FormComponent) Layout(bounds Rect) {
	f.BaseComponent.Layout(bounds)
	slog.Debug("FormComponent.Layout", "id", f.id, "bounds", bounds)
}

// Update handles events by delegating to the form.
func (f *FormComponent) Update(msg tea.Msg) (Component, tea.Cmd) {
	slog.Debug("FormComponent.Update", "id", f.id, "msg_type", slog.Any("%T", msg))

	// Delegate to form
	var cmd tea.Cmd
	f.form, cmd = f.form.Update(msg)
	return f, cmd
}

// RouteEvent checks if this component can handle the event.
func (f *FormComponent) RouteEvent(msg tea.Msg) (bool, tea.Cmd) {
	if !f.IsFocused() {
		slog.Debug("FormComponent.RouteEvent: not focused, ignoring", "id", f.id)
		return false, nil
	}

	slog.Debug("FormComponent.RouteEvent", "id", f.id, "msg_type", slog.Any("%T", msg))

	// For mouse events, check if it's within bounds
	if mouse, ok := msg.(tea.MouseClickMsg); ok {
		if !f.bounds.Contains(mouse.X, mouse.Y) {
			slog.Debug("FormComponent.RouteEvent: mouse outside bounds", "id", f.id)
			return false, nil
		}
	}

	// Update and consume the event
	updated, cmd := f.Update(msg)
	*f = *updated.(*FormComponent)
	return true, cmd
}

// View renders the form.
func (f *FormComponent) View() string {
	return f.form.View()
}

// Form returns the underlying Form.
func (f *FormComponent) Form() *components.Form {
	return f.form
}

// ViewportComponent wraps a bubbles viewport to implement the Component interface.
type ViewportComponent struct {
	*BaseComponent
	viewport *viewport.Model
	id       string
}

// NewViewportComponent creates a new viewport component.
func NewViewportComponent(id string, vp *viewport.Model) *ViewportComponent {
	v := &ViewportComponent{
		BaseComponent: NewBaseComponent(),
		viewport:      vp,
		id:            id,
	}
	v.BaseComponent.SetFocusState(FocusPassive)
	return v
}

// Layout sets the bounds and updates the viewport size.
func (v *ViewportComponent) Layout(bounds Rect) {
	v.BaseComponent.Layout(bounds)
	v.viewport.SetWidth(bounds.Width)
	v.viewport.SetHeight(bounds.Height)
	slog.Debug("ViewportComponent.Layout", "id", v.id, "bounds", bounds)
}

// Update handles events by delegating to the viewport.
func (v *ViewportComponent) Update(msg tea.Msg) (Component, tea.Cmd) {
	slog.Debug("ViewportComponent.Update", "id", v.id, "msg_type", slog.Any("%T", msg))

	// Delegate to viewport
	var cmd tea.Cmd
	*v.viewport, cmd = v.viewport.Update(msg)
	return v, cmd
}

// RouteEvent checks if this component can handle the event.
func (v *ViewportComponent) RouteEvent(msg tea.Msg) (bool, tea.Cmd) {
	if !v.IsFocused() {
		slog.Debug("ViewportComponent.RouteEvent: not focused, ignoring", "id", v.id)
		return false, nil
	}

	slog.Debug("ViewportComponent.RouteEvent", "id", v.id, "msg_type", slog.Any("%T", msg))

	// For mouse events, check if it's within bounds
	if mouse, ok := msg.(tea.MouseClickMsg); ok {
		if !v.bounds.Contains(mouse.X, mouse.Y) {
			slog.Debug("ViewportComponent.RouteEvent: mouse outside bounds", "id", v.id)
			return false, nil
		}
	}

	// Update and consume the event
	updated, cmd := v.Update(msg)
	*v = *updated.(*ViewportComponent)
	return true, cmd
}

// View renders the viewport.
func (v *ViewportComponent) View() string {
	return v.viewport.View()
}

// Viewport returns the underlying viewport.
func (v *ViewportComponent) Viewport() *viewport.Model {
	return v.viewport
}
