package layout

import "testing"

func TestRect_Contains(t *testing.T) {
	r := Rect{X: 10, Y: 20, Width: 30, Height: 40}

	tests := []struct {
		name     string
		x, y     int
		expected bool
	}{
		{"inside", 25, 40, true},
		{"top-left corner", 10, 20, true},
		{"bottom-right edge", 39, 59, true}, // Width 30 means X goes 10-39, Height 40 means Y goes 20-59
		{"outside right", 40, 40, false},
		{"outside bottom", 25, 60, false},
		{"outside left", 9, 40, false},
		{"outside top", 25, 19, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := r.Contains(tt.x, tt.y)
			if got != tt.expected {
				t.Errorf("Rect%+v.Contains(%d, %d) = %v, want %v", r, tt.x, tt.y, got, tt.expected)
			}
		})
	}
}

func TestFixed(t *testing.T) {
	spec := Fixed(42)
	if spec.Fixed != 42 {
		t.Errorf("Fixed(42).Fixed = %d, want 42", spec.Fixed)
	}
	if spec.Weight != 0 {
		t.Errorf("Fixed(42).Weight = %f, want 0", spec.Weight)
	}
}

func TestFlex(t *testing.T) {
	spec := Flex(0.5)
	if spec.Weight != 0.5 {
		t.Errorf("Flex(0.5).Weight = %f, want 0.5", spec.Weight)
	}
	if spec.Fixed != 0 {
		t.Errorf("Flex(0.5).Fixed = %d, want 0", spec.Fixed)
	}
}

func TestHSplit(t *testing.T) {
	left := NewLeaf("left")
	right := NewLeaf("right")
	container := HSplit(
		Child{Size: Flex(1), Node: left},
		Child{Size: Flex(1), Node: right},
	)

	if !container.Horizontal {
		t.Error("HSplit should create horizontal container")
	}
	if len(container.Children) != 2 {
		t.Errorf("HSplit created container with %d children, want 2", len(container.Children))
	}
}

func TestVSplit(t *testing.T) {
	top := NewLeaf("top")
	bottom := NewLeaf("bottom")
	container := VSplit(
		Child{Size: Flex(1), Node: top},
		Child{Size: Flex(1), Node: bottom},
	)

	if container.Horizontal {
		t.Error("VSplit should create vertical container")
	}
	if len(container.Children) != 2 {
		t.Errorf("VSplit created container with %d children, want 2", len(container.Children))
	}
}

func TestContainer_Layout_HSplit_EqualFlex(t *testing.T) {
	left := NewLeaf("left")
	right := NewLeaf("right")
	container := HSplit(
		Child{Size: Flex(1), Node: left},
		Child{Size: Flex(1), Node: right},
	)

	container.Layout(Rect{X: 0, Y: 0, Width: 100, Height: 50})

	leftBounds := left.Bounds()
	rightBounds := right.Bounds()

	// Left panel should be at X=0 with width ~50
	if leftBounds.X != 0 {
		t.Errorf("left.X = %d, want 0", leftBounds.X)
	}
	if leftBounds.Width != 50 {
		t.Errorf("left.Width = %d, want 50", leftBounds.Width)
	}
	if leftBounds.Height != 50 {
		t.Errorf("left.Height = %d, want 50", leftBounds.Height)
	}

	// Right panel should be at X=50 with width 50
	if rightBounds.X != 50 {
		t.Errorf("right.X = %d, want 50", rightBounds.X)
	}
	if rightBounds.Width != 50 {
		t.Errorf("right.Width = %d, want 50", rightBounds.Width)
	}
}

func TestContainer_Layout_VSplit_EqualFlex(t *testing.T) {
	top := NewLeaf("top")
	bottom := NewLeaf("bottom")
	container := VSplit(
		Child{Size: Flex(1), Node: top},
		Child{Size: Flex(1), Node: bottom},
	)

	container.Layout(Rect{X: 0, Y: 0, Width: 80, Height: 100})

	topBounds := top.Bounds()
	bottomBounds := bottom.Bounds()

	// Top panel should be at Y=0 with height ~50
	if topBounds.Y != 0 {
		t.Errorf("top.Y = %d, want 0", topBounds.Y)
	}
	if topBounds.Height != 50 {
		t.Errorf("top.Height = %d, want 50", topBounds.Height)
	}
	if topBounds.Width != 80 {
		t.Errorf("top.Width = %d, want 80", topBounds.Width)
	}

	// Bottom panel should be at Y=50 with height 50
	if bottomBounds.Y != 50 {
		t.Errorf("bottom.Y = %d, want 50", bottomBounds.Y)
	}
	if bottomBounds.Height != 50 {
		t.Errorf("bottom.Height = %d, want 50", bottomBounds.Height)
	}
}

func TestContainer_Layout_Fixed(t *testing.T) {
	sidebar := NewLeaf("sidebar")
	content := NewLeaf("content")
	container := HSplit(
		Child{Size: Fixed(20), Node: sidebar},
		Child{Size: Flex(1), Node: content},
	)

	container.Layout(Rect{X: 0, Y: 0, Width: 100, Height: 50})

	sidebarBounds := sidebar.Bounds()
	contentBounds := content.Bounds()

	// Sidebar should be fixed at 20 wide
	if sidebarBounds.Width != 20 {
		t.Errorf("sidebar.Width = %d, want 20", sidebarBounds.Width)
	}

	// Content should take remaining space (80)
	if contentBounds.X != 20 {
		t.Errorf("content.X = %d, want 20", contentBounds.X)
	}
	if contentBounds.Width != 80 {
		t.Errorf("content.Width = %d, want 80", contentBounds.Width)
	}
}

func TestContainer_Layout_MixedWeights(t *testing.T) {
	a := NewLeaf("a")
	b := NewLeaf("b")
	c := NewLeaf("c")
	container := HSplit(
		Child{Size: Flex(1), Node: a},   // 20%
		Child{Size: Flex(2), Node: b},   // 40%
		Child{Size: Flex(2), Node: c},   // 40%
	)

	container.Layout(Rect{X: 0, Y: 0, Width: 100, Height: 50})

	// a gets 1/5 = 20
	if a.Bounds().Width != 20 {
		t.Errorf("a.Width = %d, want 20", a.Bounds().Width)
	}
	// b gets 2/5 = 40
	if b.Bounds().Width != 40 {
		t.Errorf("b.Width = %d, want 40", b.Bounds().Width)
	}
	// c gets remaining 40
	if c.Bounds().Width != 40 {
		t.Errorf("c.Width = %d, want 40", c.Bounds().Width)
	}
}

func TestContainer_Layout_Offset(t *testing.T) {
	leaf := NewLeaf("leaf")
	container := HSplit(Child{Size: Flex(1), Node: leaf})

	// Layout with non-zero offset
	container.Layout(Rect{X: 10, Y: 20, Width: 30, Height: 40})

	bounds := leaf.Bounds()
	if bounds.X != 10 || bounds.Y != 20 {
		t.Errorf("leaf positioned at (%d, %d), want (10, 20)", bounds.X, bounds.Y)
	}
}

func TestContainer_Layout_Empty(t *testing.T) {
	container := HSplit()
	// Should not panic
	container.Layout(Rect{X: 0, Y: 0, Width: 100, Height: 100})
}

func TestContainer_LeafAt(t *testing.T) {
	left := NewLeaf("left")
	right := NewLeaf("right")
	container := HSplit(
		Child{Size: Flex(1), Node: left},
		Child{Size: Flex(1), Node: right},
	)
	container.Layout(Rect{X: 0, Y: 0, Width: 100, Height: 50})

	// Point in left panel
	found := container.LeafAt(25, 25)
	if found == nil || found.ID != "left" {
		t.Errorf("LeafAt(25, 25) = %v, want leaf 'left'", found)
	}

	// Point in right panel
	found = container.LeafAt(75, 25)
	if found == nil || found.ID != "right" {
		t.Errorf("LeafAt(75, 25) = %v, want leaf 'right'", found)
	}

	// Point outside container
	found = container.LeafAt(200, 200)
	if found != nil {
		t.Errorf("LeafAt(200, 200) = %v, want nil", found)
	}
}

func TestLeaf_Size(t *testing.T) {
	leaf := NewLeaf("test")
	leaf.Layout(Rect{X: 10, Y: 20, Width: 30, Height: 40})

	w, h := leaf.Size()
	if w != 30 || h != 40 {
		t.Errorf("leaf.Size() = (%d, %d), want (30, 40)", w, h)
	}
}

func TestLeaf_LeafAt(t *testing.T) {
	leaf := NewLeaf("test")
	leaf.Layout(Rect{X: 10, Y: 20, Width: 30, Height: 40})

	// Point inside
	found := leaf.LeafAt(25, 35)
	if found != leaf {
		t.Errorf("LeafAt inside bounds returned wrong leaf")
	}

	// Point outside
	found = leaf.LeafAt(5, 35)
	if found != nil {
		t.Errorf("LeafAt outside bounds should return nil")
	}
}

func TestTree_Get(t *testing.T) {
	left := NewLeaf("left")
	right := NewLeaf("right")
	root := HSplit(
		Child{Size: Flex(1), Node: left},
		Child{Size: Flex(1), Node: right},
	)

	tree := NewTree(root)

	// Get existing leaves
	if tree.Get("left") != left {
		t.Error("Get('left') should return left leaf")
	}
	if tree.Get("right") != right {
		t.Error("Get('right') should return right leaf")
	}

	// Get non-existent leaf
	if tree.Get("nonexistent") != nil {
		t.Error("Get('nonexistent') should return nil")
	}
}

func TestTree_Layout(t *testing.T) {
	leaf := NewLeaf("test")
	root := HSplit(Child{Size: Flex(1), Node: leaf})
	tree := NewTree(root)

	tree.Layout(200, 100)

	bounds := leaf.Bounds()
	if bounds.Width != 200 || bounds.Height != 100 {
		t.Errorf("After Layout(200, 100), leaf bounds = %+v", bounds)
	}
}

func TestTree_At(t *testing.T) {
	left := NewLeaf("left")
	right := NewLeaf("right")
	root := HSplit(
		Child{Size: Flex(1), Node: left},
		Child{Size: Flex(1), Node: right},
	)
	tree := NewTree(root)
	tree.Layout(100, 50)

	// Point in left panel
	found := tree.At(25, 25)
	if found == nil || found.ID != "left" {
		t.Errorf("At(25, 25) = %v, want 'left'", found)
	}

	// Point in right panel
	found = tree.At(75, 25)
	if found == nil || found.ID != "right" {
		t.Errorf("At(75, 25) = %v, want 'right'", found)
	}
}

func TestTree_Bounds(t *testing.T) {
	leaf := NewLeaf("test")
	root := HSplit(Child{Size: Flex(1), Node: leaf})
	tree := NewTree(root)
	tree.Layout(100, 50)

	bounds := tree.Bounds("test")
	if bounds.Width != 100 || bounds.Height != 50 {
		t.Errorf("Bounds('test') = %+v, want Width=100, Height=50", bounds)
	}

	// Non-existent leaf returns zero rect
	bounds = tree.Bounds("nonexistent")
	if bounds.Width != 0 || bounds.Height != 0 {
		t.Errorf("Bounds('nonexistent') = %+v, want zero rect", bounds)
	}
}

func TestNestedLayout(t *testing.T) {
	// Create a nested layout:
	// +-------+-------+
	// |       |  top  |
	// | left  +-------+
	// |       | bottom|
	// +-------+-------+
	left := NewLeaf("left")
	top := NewLeaf("top")
	bottom := NewLeaf("bottom")

	root := HSplit(
		Child{Size: Flex(1), Node: left},
		Child{Size: Flex(1), Node: VSplit(
			Child{Size: Flex(1), Node: top},
			Child{Size: Flex(1), Node: bottom},
		)},
	)

	tree := NewTree(root)
	tree.Layout(100, 100)

	// Left should be 50x100
	leftBounds := left.Bounds()
	if leftBounds.Width != 50 || leftBounds.Height != 100 {
		t.Errorf("left bounds = %+v, want 50x100", leftBounds)
	}

	// Top should be 50x50 at (50, 0)
	topBounds := top.Bounds()
	if topBounds.X != 50 || topBounds.Y != 0 {
		t.Errorf("top position = (%d, %d), want (50, 0)", topBounds.X, topBounds.Y)
	}
	if topBounds.Width != 50 || topBounds.Height != 50 {
		t.Errorf("top size = %dx%d, want 50x50", topBounds.Width, topBounds.Height)
	}

	// Bottom should be 50x50 at (50, 50)
	bottomBounds := bottom.Bounds()
	if bottomBounds.X != 50 || bottomBounds.Y != 50 {
		t.Errorf("bottom position = (%d, %d), want (50, 50)", bottomBounds.X, bottomBounds.Y)
	}

	// Test At() for nested layout
	if tree.At(25, 50).ID != "left" {
		t.Error("At(25, 50) should find 'left'")
	}
	if tree.At(75, 25).ID != "top" {
		t.Error("At(75, 25) should find 'top'")
	}
	if tree.At(75, 75).ID != "bottom" {
		t.Error("At(75, 75) should find 'bottom'")
	}
}

func TestContainer_Bounds(t *testing.T) {
	container := HSplit(Child{Size: Flex(1), Node: NewLeaf("test")})
	bounds := Rect{X: 10, Y: 20, Width: 100, Height: 50}
	container.Layout(bounds)

	got := container.Bounds()
	if got != bounds {
		t.Errorf("Container.Bounds() = %+v, want %+v", got, bounds)
	}
}
