# Details View System Refactor

## Current Problems

1. **Monolithic file**: `details.go` is 1600+ lines mixing component logic with render functions
2. **Inconsistent buffering**: Some methods buffer, others write immediately
3. **Alignment confusion**: `maxLabelLen` resets per flush, `minLabelLen` is global
4. **Boolean flag soup**: `isItem`, `isLine` flags instead of proper types
5. **No composability**: Can't build views declaratively

## New Design

### Core Types

```go
// AlignmentHints collected from sections for consistent rendering
type AlignmentHints struct {
    LabelWidth int  // Maximum label width across all FieldsSections
}

// Section is the interface all section types implement
type Section interface {
    // CollectHints reports alignment requirements
    CollectHints() AlignmentHints
    
    // Render outputs the section content using provided hints
    Render(hints AlignmentHints, styles ui.Styles) string
}

// DetailsView holds sections and renders them with consistent alignment
type DetailsView struct {
    sections []Section
    styles   ui.Styles
}

func (v *DetailsView) Add(s Section) { v.sections = append(v.sections, s) }

func (v *DetailsView) Render() string {
    // Phase 1: Collect hints from all sections
    hints := AlignmentHints{}
    for _, s := range v.sections {
        h := s.CollectHints()
        if h.LabelWidth > hints.LabelWidth {
            hints.LabelWidth = h.LabelWidth
        }
    }
    
    // Phase 2: Render all sections with consistent hints
    var out strings.Builder
    for _, s := range v.sections {
        out.WriteString(s.Render(hints, v.styles))
    }
    return out.String()
}
```

### Section Implementations

#### 1. TextSection
Simple text output (headers, raw text, blank lines).

```go
type TextSection struct {
    content  string
    isHeader bool
}

func Header(title string) *TextSection
func Text(content string) *TextSection
func Blank() *TextSection
```

#### 2. FieldsSection
Key-value pairs with alignment.

```go
type Field struct {
    Label      string
    Value      string
    LabelMuted bool           // Gray label for technical IDs
    ValueStyle lipgloss.Style // Optional styled value
    Indent     int            // 0 = normal, 1 = sub-field (6 spaces)
}

type FieldsSection struct {
    fields []Field
}

func (s *FieldsSection) Add(label, value string) *FieldsSection
func (s *FieldsSection) AddMuted(label, value string) *FieldsSection
func (s *FieldsSection) AddStyled(label, value string, style lipgloss.Style) *FieldsSection
func (s *FieldsSection) AddSub(label, value string) *FieldsSection  // Indented sub-field
```

#### 3. ListSection
Bullet point lists with customizable bullets.

```go
type ListItem struct {
    Bullet string         // "•", "●", "○", or custom (light indicator)
    Text   string
    Style  lipgloss.Style // For the bullet
}

type ListSection struct {
    items  []ListItem
    indent int  // 0 = 2 spaces, 1 = 4 spaces (nested)
}

func (s *ListSection) Add(text string) *ListSection                    // Default bullet
func (s *ListSection) AddCustom(bullet, text string, style lipgloss.Style) *ListSection
```

### File Structure

```
internal/ui/panels/
  details/
    view.go      - DetailsView, Section interface, AlignmentHints
    text.go      - TextSection
    fields.go    - FieldsSection, Field
    list.go      - ListSection, ListItem
  
  detailspanel.go          - DetailsPanel (Bubble Tea component)
  details_bridge.go        - buildBridgeView()
  details_device.go        - buildDeviceView()
  details_light.go         - buildLightView()
  details_room.go          - buildRoomView()
  details_scene.go         - buildSceneView()
  details_entertainment.go - buildEntertainmentView()
  details_category.go      - buildLightsCategoryView(), etc.
```

### Usage Example

```go
func buildDeviceView(device openhue.DeviceGet, state *hue.BridgeState, styles ui.Styles) string {
    view := details.NewView(styles)
    
    // IDs section (no header for first section)
    ids := details.NewFields()
    ids.AddMuted("ID", *device.Id)
    ids.AddMuted("Type", string(*device.Type))
    view.Add(ids)
    
    // Product section
    view.Add(details.Header("Product"))
    product := details.NewFields()
    product.Add("Manufacturer", *device.ProductData.ManufacturerName)
    product.Add("Product", *device.ProductData.ProductName)
    product.AddMuted("Model ID", *device.ProductData.ModelId)
    view.Add(product)
    
    // Sensors section
    if hasSensors {
        view.Add(details.Header("Sensors"))
        sensors := details.NewFields()
        sensors.Add("Motion", fmt.Sprintf("%s %s", indicator, status))
        sensors.Add("Temperature", "22.6°C")
        view.Add(sensors)
    }
    
    // Services list
    view.Add(details.Header("Services"))
    services := details.NewList()
    services.Add("Device software update")
    view.Add(services)
    
    return view.Render()
}
```

### Migration Strategy

1. Create `details/` package with core types
2. Implement TextSection, FieldsSection, ListSection
3. Create `detailspanel.go` with slimmed-down DetailsPanel
4. Migrate one render function at a time to new `build*View()` pattern
5. Delete old code from `details.go` as we go
6. Final cleanup: remove old `details.go`

### Benefits

1. **Declarative**: Build views by adding sections, not writing strings
2. **Consistent alignment**: Calculated across entire view before rendering
3. **Type-safe**: No boolean flags, proper types for each section kind
4. **Composable**: Sections are independent, reusable
5. **Maintainable**: Each entity type in its own file
6. **Testable**: Sections can be unit tested independently
