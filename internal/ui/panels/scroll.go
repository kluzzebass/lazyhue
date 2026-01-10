package panels

// ScrollState provides common scrolling behavior for panels with a cursor and scroll offset.
// Embed this struct in panels to get consistent scrolling behavior.
type ScrollState struct {
	cursor int // Currently selected item index
	offset int // First visible item index (scroll position)
	width  int // Panel width
	height int // Panel height
}

// SetSize updates the dimensions.
func (s *ScrollState) SetSize(width, height int) {
	s.width = width
	s.height = height
}

// ViewHeight returns the number of visible lines (height minus borders).
func (s *ScrollState) ViewHeight() int {
	return max(1, s.height-2)
}

// ContentWidth returns the available content width (width minus borders).
func (s *ScrollState) ContentWidth() int {
	return max(1, s.width-2)
}

// Cursor returns the current cursor position.
func (s *ScrollState) Cursor() int {
	return s.cursor
}

// Offset returns the current scroll offset.
func (s *ScrollState) Offset() int {
	return s.offset
}

// Width returns the panel width.
func (s *ScrollState) Width() int {
	return s.width
}

// Height returns the panel height.
func (s *ScrollState) Height() int {
	return s.height
}

// SetCursor sets the cursor position and ensures it's visible.
func (s *ScrollState) SetCursor(pos int) {
	s.cursor = pos
}

// SetOffset sets the scroll offset.
func (s *ScrollState) SetOffset(offset int) {
	s.offset = offset
}

// MoveCursor moves the cursor by delta within the given item count.
// Returns the new cursor position.
func (s *ScrollState) MoveCursor(delta, itemCount int) int {
	if itemCount == 0 {
		s.cursor = 0
		return 0
	}

	s.cursor += delta
	if s.cursor < 0 {
		s.cursor = 0
	}
	if s.cursor >= itemCount {
		s.cursor = itemCount - 1
	}

	s.EnsureCursorVisible(itemCount)
	return s.cursor
}

// EnsureCursorVisible adjusts the scroll offset to keep the cursor in view.
func (s *ScrollState) EnsureCursorVisible(itemCount int) {
	viewHeight := s.ViewHeight()

	// Scroll up if cursor is above visible area
	if s.cursor < s.offset {
		s.offset = s.cursor
	}

	// Scroll down if cursor is below visible area
	if s.cursor >= s.offset+viewHeight {
		s.offset = s.cursor - viewHeight + 1
	}

	// Clamp offset to valid range
	maxOffset := max(0, itemCount-viewHeight)
	if s.offset > maxOffset {
		s.offset = maxOffset
	}
	if s.offset < 0 {
		s.offset = 0
	}
}

// ClampCursor ensures the cursor is within valid range for the given item count.
func (s *ScrollState) ClampCursor(itemCount int) {
	if itemCount == 0 {
		s.cursor = 0
		s.offset = 0
		return
	}
	if s.cursor >= itemCount {
		s.cursor = itemCount - 1
	}
	s.EnsureCursorVisible(itemCount)
}

// MoveToStart moves cursor to the first item.
func (s *ScrollState) MoveToStart() {
	s.cursor = 0
	s.offset = 0
}

// MoveToEnd moves cursor to the last item.
func (s *ScrollState) MoveToEnd(itemCount int) {
	if itemCount > 0 {
		s.cursor = itemCount - 1
	} else {
		s.cursor = 0
	}
	s.EnsureCursorVisible(itemCount)
}

// PageUp moves cursor up by one page.
func (s *ScrollState) PageUp(itemCount int) {
	s.MoveCursor(-s.ViewHeight(), itemCount)
}

// PageDown moves cursor down by one page.
func (s *ScrollState) PageDown(itemCount int) {
	s.MoveCursor(s.ViewHeight(), itemCount)
}

// VisibleRange returns the start and end indices of visible items.
func (s *ScrollState) VisibleRange(itemCount int) (start, end int) {
	start = s.offset
	end = min(s.offset+s.ViewHeight(), itemCount)
	return start, end
}

// HandleClick converts a click at relative Y coordinate to an item index.
// Returns the item index, or -1 if the click is outside the content area.
func (s *ScrollState) HandleClick(relY, itemCount int) int {
	// Content starts at y=1 (after top border)
	if relY < 1 {
		return -1
	}

	itemIndex := s.offset + (relY - 1)
	if itemIndex >= 0 && itemIndex < itemCount {
		s.cursor = itemIndex
		return itemIndex
	}
	return -1
}
