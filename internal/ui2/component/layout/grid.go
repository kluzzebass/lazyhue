package layout

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/kluzzebass/lazyhue/internal/ui2/component"
	"github.com/kluzzebass/lazyhue/internal/ui2/component/field"
)

// RowType indicates whether a row is a normal cell row or a section header.
type RowType int

const (
	// RowTypeNormal is a row with cells arranged in columns.
	RowTypeNormal RowType = iota
	// RowTypeSection is a full-width section header row.
	RowTypeSection
)

// GridCell represents a single cell in the grid.
type GridCell struct {
	Component component.Component
	ColSpan   int     // Number of columns this cell spans (default 1)
	Padding   Spacing // Inner spacing (inside cell, around component)
	Margin    Spacing // Outer spacing (outside cell boundary)
}

// GridRow represents a row in the grid - either cells or a section header.
type GridRow struct {
	Type    RowType
	Cells   []GridCell          // Used for RowTypeNormal
	Section component.Component // Used for RowTypeSection
}

// Grid is a 2D layout container that arranges components in rows and columns.
// Column widths are calculated based on the maximum content width in each column.
// Supports both normal cell rows and full-width section header rows.
type Grid struct {
	*component.BaseComponent

	rows               []GridRow
	colWidths          []int  // Calculated column widths
	colGap             int    // Gap between columns
	rowGap             int    // Gap between rows (0 = no extra gap beyond newlines)
	focusRow           int    // Currently focused row
	focusCol           int    // Currently focused column (cell index, not grid column)
	ShowFocusIndicator bool   // Whether to show "> " prefix on focused row
	FocusIndicator     string // The focus indicator string (default "> ")
	BlurIndicator      string // The blur indicator string (default "  ")
}

// NewGrid creates a new grid layout.
func NewGrid() *Grid {
	return &Grid{
		BaseComponent:      component.NewBaseComponent(),
		colGap:             2,
		rowGap:             0,
		ShowFocusIndicator: false,
		FocusIndicator:     "> ",
		BlurIndicator:      "  ",
	}
}

// SetFocusIndicator enables focus indicators and optionally sets custom strings.
func (g *Grid) SetFocusIndicator(show bool, focus, blur string) *Grid {
	g.ShowFocusIndicator = show
	if focus != "" {
		g.FocusIndicator = focus
	}
	if blur != "" {
		g.BlurIndicator = blur
	}
	return g
}

// SetGaps sets the column and row gaps.
func (g *Grid) SetGaps(colGap, rowGap int) *Grid {
	g.colGap = colGap
	g.rowGap = rowGap
	return g
}

// SetRows sets all rows at once and recalculates column widths.
func (g *Grid) SetRows(rows []GridRow) *Grid {
	// Normalize ColSpan for all cells (0 means 1)
	for i := range rows {
		if rows[i].Type == RowTypeNormal {
			for j := range rows[i].Cells {
				if rows[i].Cells[j].ColSpan == 0 {
					rows[i].Cells[j].ColSpan = 1
				}
			}
		}
	}
	g.rows = rows
	g.calculateColumnWidths()
	return g
}

// AddRow adds a normal cell row to the grid.
func (g *Grid) AddRow(cells ...GridCell) *Grid {
	// Default ColSpan to 1 if not set
	for i := range cells {
		if cells[i].ColSpan == 0 {
			cells[i].ColSpan = 1
		}
	}
	g.rows = append(g.rows, GridRow{Type: RowTypeNormal, Cells: cells})
	g.calculateColumnWidths()
	return g
}

// AddSection adds a full-width section header row to the grid.
func (g *Grid) AddSection(section component.Component) *Grid {
	g.rows = append(g.rows, GridRow{Type: RowTypeSection, Section: section})
	return g
}

// calculateColumnWidths determines the width of each column based on content.
func (g *Grid) calculateColumnWidths() {
	if len(g.rows) == 0 {
		g.colWidths = nil
		return
	}

	// Find maximum number of grid columns (only from normal rows)
	maxCols := 0
	for _, row := range g.rows {
		if row.Type != RowTypeNormal {
			continue
		}
		cols := 0
		for _, cell := range row.Cells {
			cols += cell.ColSpan
		}
		if cols > maxCols {
			maxCols = cols
		}
	}

	// Initialize column widths
	g.colWidths = make([]int, maxCols)

	// Calculate width for each column based on content (only from normal rows)
	for _, row := range g.rows {
		if row.Type != RowTypeNormal {
			continue
		}
		gridCol := 0
		for _, cell := range row.Cells {
			if cell.Component != nil && cell.ColSpan == 1 {
				// Only single-span cells contribute to column width calculation
				width := g.getCellWidth(cell)
				if width > g.colWidths[gridCol] {
					g.colWidths[gridCol] = width
				}
			}
			gridCol += cell.ColSpan
		}
	}
}

// getCellWidth returns the display width of a cell's content including padding and margin.
func (g *Grid) getCellWidth(cell GridCell) int {
	var content string
	if cell.Component != nil {
		// Use ViewControl() for field components to match what's rendered
		if cr, ok := cell.Component.(field.ControlRenderer); ok {
			content = cr.ViewControl()
		} else {
			content = cell.Component.View()
		}
	}

	// Apply cell padding and margin to match rendering
	content = ApplySpacing(content, cell.Padding)
	content = ApplySpacing(content, cell.Margin)

	if content == "" {
		return 0
	}

	lines := strings.Split(content, "\n")

	maxWidth := 0
	for _, line := range lines {
		// Count visible characters (this is simplified - doesn't handle ANSI)
		width := len([]rune(stripAnsi(line)))
		if width > maxWidth {
			maxWidth = width
		}
	}
	return maxWidth
}

// stripAnsi removes ANSI escape sequences from a string.
// This is a simplified version - for full support use a proper library.
func stripAnsi(s string) string {
	var result strings.Builder
	inEscape := false

	for _, r := range s {
		if r == '\x1b' {
			inEscape = true
			continue
		}
		if inEscape {
			if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
				inEscape = false
			}
			continue
		}
		result.WriteRune(r)
	}

	return result.String()
}

// Init initializes the grid.
func (g *Grid) Init() tea.Cmd {
	return nil
}

// Update handles events for the grid.
func (g *Grid) Update(msg tea.Msg) (component.Component, tea.Cmd) {
	// Route to focused cell
	if g.focusRow >= 0 && g.focusRow < len(g.rows) {
		row := g.rows[g.focusRow]
		if row.Type == RowTypeNormal && g.focusCol >= 0 && g.focusCol < len(row.Cells) {
			cell := row.Cells[g.focusCol]
			if cell.Component != nil {
				_, cmd := cell.Component.Update(msg)
				return g, cmd
			}
		}
	}
	return g, nil
}

// RouteEvent routes events to the appropriate cell.
func (g *Grid) RouteEvent(msg tea.Msg) (bool, tea.Cmd) {
	// For mouse clicks, try ALL cells - the clicked one should handle it and get focus
	if _, isMouseClick := msg.(tea.MouseClickMsg); isMouseClick {
		for rowIdx, row := range g.rows {
			if row.Type != RowTypeNormal {
				continue
			}
			for colIdx, cell := range row.Cells {
				if cell.Component == nil {
					continue
				}
				if handled, cmd := cell.Component.RouteEvent(msg); handled {
					// Move focus to the clicked cell
					g.SetFocus(rowIdx, colIdx)
					return true, cmd
				}
			}
		}
		return false, nil
	}

	// For mouse wheel, route to focused cell (for dropdowns, sliders, etc.)
	if _, isMouseWheel := msg.(tea.MouseWheelMsg); isMouseWheel {
		if g.focusRow >= 0 && g.focusRow < len(g.rows) {
			row := g.rows[g.focusRow]
			if row.Type == RowTypeNormal && g.focusCol >= 0 && g.focusCol < len(row.Cells) {
				cell := row.Cells[g.focusCol]
				if cell.Component != nil {
					if handled, cmd := cell.Component.RouteEvent(msg); handled {
						return true, cmd
					}
				}
			}
		}
		return false, nil
	}

	// For other events (keyboard), route to focused cell first
	if g.focusRow >= 0 && g.focusRow < len(g.rows) {
		row := g.rows[g.focusRow]
		if row.Type == RowTypeNormal && g.focusCol >= 0 && g.focusCol < len(row.Cells) {
			cell := row.Cells[g.focusCol]
			if cell.Component != nil {
				if handled, cmd := cell.Component.RouteEvent(msg); handled {
					return true, cmd
				}
			}
		}
	}

	// Handle grid-level navigation
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if g.moveFocus(-1, 0) {
				return true, nil
			}
		case "down", "j":
			if g.moveFocus(1, 0) {
				return true, nil
			}
		}
	}

	return false, nil
}

// moveFocus moves focus by the given delta.
func (g *Grid) moveFocus(rowDelta, colDelta int) bool {
	if len(g.rows) == 0 {
		return false
	}

	// Blur current
	if g.focusRow >= 0 && g.focusRow < len(g.rows) {
		row := g.rows[g.focusRow]
		if row.Type == RowTypeNormal && g.focusCol >= 0 && g.focusCol < len(row.Cells) {
			if row.Cells[g.focusCol].Component != nil {
				row.Cells[g.focusCol].Component.Blur()
			}
		}
	}

	// Find next focusable cell
	newRow := g.focusRow + rowDelta
	newCol := g.focusCol + colDelta

	for newRow >= 0 && newRow < len(g.rows) {
		row := g.rows[newRow]

		// Skip section rows - they're not focusable
		if row.Type == RowTypeSection {
			if rowDelta >= 0 {
				newRow++
			} else {
				newRow--
			}
			continue
		}

		// Clamp column to valid range
		if newCol < 0 {
			newCol = 0
		}
		if newCol >= len(row.Cells) {
			newCol = len(row.Cells) - 1
		}

		// Find focusable cell in this row
		for col := newCol; col >= 0 && col < len(row.Cells); {
			cell := row.Cells[col]
			if cell.Component != nil && cell.Component.CanFocus() {
				g.focusRow = newRow
				g.focusCol = col
				cell.Component.Focus()
				return true
			}

			if colDelta >= 0 {
				col++
			} else {
				col--
			}
		}

		if rowDelta >= 0 {
			newRow++
		} else {
			newRow--
		}
		newCol = g.focusCol // Reset column search for next row
	}

	// Restore focus to current if no valid target
	if g.focusRow >= 0 && g.focusRow < len(g.rows) {
		row := g.rows[g.focusRow]
		if row.Type == RowTypeNormal && g.focusCol >= 0 && g.focusCol < len(row.Cells) {
			if row.Cells[g.focusCol].Component != nil {
				row.Cells[g.focusCol].Component.Focus()
			}
		}
	}

	return false
}

// Focus focuses the first focusable cell.
func (g *Grid) Focus() {
	g.BaseComponent.Focus()
	g.focusRow = -1
	g.focusCol = 0
	g.moveFocus(1, 0) // Find first focusable
}

// Blur blurs the currently focused cell.
func (g *Grid) Blur() {
	g.BaseComponent.Blur()
	if g.focusRow >= 0 && g.focusRow < len(g.rows) {
		row := g.rows[g.focusRow]
		if row.Type == RowTypeNormal && g.focusCol >= 0 && g.focusCol < len(row.Cells) {
			if row.Cells[g.focusCol].Component != nil {
				row.Cells[g.focusCol].Component.Blur()
			}
		}
	}
}

// View renders the grid.
func (g *Grid) View() string {
	if len(g.rows) == 0 {
		return ""
	}

	var out strings.Builder

	for rowIdx, row := range g.rows {
		if rowIdx > 0 {
			out.WriteString("\n")
			// Add extra row gap lines
			for i := 0; i < g.rowGap; i++ {
				out.WriteString("\n")
			}
		}

		switch row.Type {
		case RowTypeSection:
			// Section rows render their component directly (full width)
			if row.Section != nil {
				sectionContent := row.Section.View()
				// Add blank focus indicator space for alignment if enabled
				if g.ShowFocusIndicator {
					sectionContent = g.BlurIndicator + sectionContent
				}
				out.WriteString(sectionContent)
			}
		case RowTypeNormal:
			// Normal rows render cells in columns
			isFocused := rowIdx == g.focusRow
			rowLines := g.renderRow(row, isFocused)
			out.WriteString(strings.Join(rowLines, "\n"))
		}
	}

	return out.String()
}

// renderRow renders a single row, returning lines (for multi-line cell support).
func (g *Grid) renderRow(row GridRow, isFocused bool) []string {
	// Get rendered content for each cell
	cellContents := make([][]string, len(row.Cells))
	maxLines := 1

	for i, cell := range row.Cells {
		var content string
		if cell.Component != nil {
			// Use ViewControl() for field components to avoid rendering the label twice
			if cr, ok := cell.Component.(field.ControlRenderer); ok {
				content = cr.ViewControl()
			} else {
				content = cell.Component.View()
			}
		}

		// Apply cell padding (inner spacing)
		content = ApplySpacing(content, cell.Padding)

		// Apply cell margin (outer spacing)
		content = ApplySpacing(content, cell.Margin)

		if content == "" {
			cellContents[i] = []string{""}
		} else {
			cellContents[i] = strings.Split(content, "\n")
		}

		if len(cellContents[i]) > maxLines {
			maxLines = len(cellContents[i])
		}
	}

	// Determine focus indicator for this row
	var indicator string
	if g.ShowFocusIndicator {
		if isFocused {
			indicator = g.FocusIndicator
		} else {
			indicator = g.BlurIndicator
		}
	}

	// Build output lines
	lines := make([]string, maxLines)

	for lineIdx := 0; lineIdx < maxLines; lineIdx++ {
		var lineBuilder strings.Builder

		// Add focus indicator (only on first line, spaces on subsequent lines)
		if indicator != "" {
			if lineIdx == 0 {
				lineBuilder.WriteString(indicator)
			} else {
				lineBuilder.WriteString(strings.Repeat(" ", len(indicator)))
			}
		}

		gridCol := 0

		for cellIdx, cell := range row.Cells {
			if cellIdx > 0 {
				lineBuilder.WriteString(strings.Repeat(" ", g.colGap))
			}

			// Calculate width for this cell (sum of spanned columns + gaps)
			cellWidth := 0
			for c := 0; c < cell.ColSpan && gridCol+c < len(g.colWidths); c++ {
				if c > 0 {
					cellWidth += g.colGap
				}
				cellWidth += g.colWidths[gridCol+c]
			}

			// Get content for this line
			content := ""
			if lineIdx < len(cellContents[cellIdx]) {
				content = cellContents[cellIdx][lineIdx]
			}

			// Pad content to cell width
			contentWidth := len([]rune(stripAnsi(content)))
			lineBuilder.WriteString(content)
			if contentWidth < cellWidth {
				lineBuilder.WriteString(strings.Repeat(" ", cellWidth-contentWidth))
			}

			gridCol += cell.ColSpan
		}

		lines[lineIdx] = lineBuilder.String()
	}

	return lines
}

// Children returns all components in the grid.
func (g *Grid) Children() []component.Component {
	var children []component.Component
	for _, row := range g.rows {
		switch row.Type {
		case RowTypeSection:
			if row.Section != nil {
				children = append(children, row.Section)
			}
		case RowTypeNormal:
			for _, cell := range row.Cells {
				if cell.Component != nil {
					children = append(children, cell.Component)
				}
			}
		}
	}
	return children
}

// FocusedCell returns the currently focused row and column indices.
func (g *Grid) FocusedCell() (int, int) {
	return g.focusRow, g.focusCol
}

// SetFocus sets focus to a specific cell (only works for normal rows).
func (g *Grid) SetFocus(row, col int) {
	// Blur current
	if g.focusRow >= 0 && g.focusRow < len(g.rows) {
		r := g.rows[g.focusRow]
		if r.Type == RowTypeNormal && g.focusCol >= 0 && g.focusCol < len(r.Cells) {
			if r.Cells[g.focusCol].Component != nil {
				r.Cells[g.focusCol].Component.Blur()
			}
		}
	}

	// Focus new (only normal rows can be focused)
	if row >= 0 && row < len(g.rows) {
		r := g.rows[row]
		if r.Type == RowTypeNormal && col >= 0 && col < len(r.Cells) {
			g.focusRow = row
			g.focusCol = col
			if r.Cells[col].Component != nil {
				r.Cells[col].Component.Focus()
			}
		}
	}
}
