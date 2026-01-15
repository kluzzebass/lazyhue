package ui

import "testing"

func TestCalculateScrollThumb_NoScrollNeeded(t *testing.T) {
	tests := []struct {
		name                                           string
		scrollPos, totalHeight, viewHeight, borderHeight int
	}{
		{"content fits in view", 0, 10, 20, 20},
		{"content exactly fits", 0, 20, 20, 20},
		{"zero border height", 0, 100, 20, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			start, end := CalculateScrollThumb(tt.scrollPos, tt.totalHeight, tt.viewHeight, tt.borderHeight)
			if start != -1 || end != -1 {
				t.Errorf("CalculateScrollThumb() = (%d, %d), want (-1, -1) for no scroll needed", start, end)
			}
		})
	}
}

func TestCalculateScrollThumb_AtTop(t *testing.T) {
	// 100 total items, 20 visible, 20 border height, scroll at top
	start, end := CalculateScrollThumb(0, 100, 20, 20)

	if start != 0 {
		t.Errorf("At top, thumb start = %d, want 0", start)
	}
	if end <= start {
		t.Errorf("Thumb end (%d) should be greater than start (%d)", end, start)
	}
}

func TestCalculateScrollThumb_AtBottom(t *testing.T) {
	totalHeight := 100
	viewHeight := 20
	borderHeight := 20
	maxScroll := totalHeight - viewHeight // 80

	start, end := CalculateScrollThumb(maxScroll, totalHeight, viewHeight, borderHeight)

	// At bottom, thumb should end at border height
	if end != borderHeight {
		t.Errorf("At bottom, thumb end = %d, want %d", end, borderHeight)
	}
	if start >= end {
		t.Errorf("Thumb start (%d) should be less than end (%d)", start, end)
	}
}

func TestCalculateScrollThumb_Middle(t *testing.T) {
	totalHeight := 100
	viewHeight := 20
	borderHeight := 20
	midScroll := (totalHeight - viewHeight) / 2 // 40

	start, end := CalculateScrollThumb(midScroll, totalHeight, viewHeight, borderHeight)

	// In middle, thumb should be somewhere in the middle of the border
	thumbSize := end - start
	if start <= 0 {
		t.Errorf("In middle, thumb start should be > 0, got %d", start)
	}
	if end >= borderHeight {
		t.Errorf("In middle, thumb end should be < borderHeight, got %d", end)
	}
	if thumbSize < 1 {
		t.Errorf("Thumb size should be at least 1, got %d", thumbSize)
	}
}

func TestCalculateScrollThumb_ClampNegativeScroll(t *testing.T) {
	// Negative scroll should be treated as 0
	start1, end1 := CalculateScrollThumb(-10, 100, 20, 20)
	start2, end2 := CalculateScrollThumb(0, 100, 20, 20)

	if start1 != start2 || end1 != end2 {
		t.Errorf("Negative scroll (%d, %d) should equal zero scroll (%d, %d)",
			start1, end1, start2, end2)
	}
}

func TestCalculateScrollThumb_ClampOverScroll(t *testing.T) {
	// Over-scroll should be clamped to max
	maxScroll := 80 // 100 - 20
	start1, end1 := CalculateScrollThumb(200, 100, 20, 20) // way over
	start2, end2 := CalculateScrollThumb(maxScroll, 100, 20, 20)

	if start1 != start2 || end1 != end2 {
		t.Errorf("Over-scroll (%d, %d) should equal max scroll (%d, %d)",
			start1, end1, start2, end2)
	}
}

func TestCalculateScrollThumb_MinThumbSize(t *testing.T) {
	// Very long content should still have at least 1-char thumb
	start, end := CalculateScrollThumb(0, 10000, 10, 10)

	thumbSize := end - start
	if thumbSize < 1 {
		t.Errorf("Thumb size should be at least 1, got %d", thumbSize)
	}
}

func TestCalculateScrollThumb_ThumbProgression(t *testing.T) {
	// As scroll increases, thumb should move down
	totalHeight := 100
	viewHeight := 20
	borderHeight := 20

	var prevStart int = -1
	for scroll := 0; scroll <= totalHeight-viewHeight; scroll += 20 {
		start, _ := CalculateScrollThumb(scroll, totalHeight, viewHeight, borderHeight)
		if prevStart >= 0 && start < prevStart {
			t.Errorf("Thumb should not move up as scroll increases: scroll=%d, start=%d, prevStart=%d",
				scroll, start, prevStart)
		}
		prevStart = start
	}
}
