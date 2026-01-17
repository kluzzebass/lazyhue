// Package layout provides a simple recursive layout system for TUI panels.
package layout

// Rect represents a rectangular region.
type Rect struct {
	X, Y, Width, Height int
}

// Contains returns true if the point (x, y) is inside the rectangle.
func (r Rect) Contains(x, y int) bool {
	return x >= r.X && x < r.X+r.Width && y >= r.Y && y < r.Y+r.Height
}

// Node is implemented by all layout nodes.
type Node interface {
	// Layout calculates positions for this node and its children.
	Layout(bounds Rect)

	// Bounds returns this node's calculated bounds.
	Bounds() Rect

	// LeafAt returns the leaf panel at the given coordinates, or nil.
	LeafAt(x, y int) *Leaf
}

// SizeSpec defines how a child should be sized.
type SizeSpec struct {
	Fixed  int     // Fixed size in cells (0 = use flex)
	Weight float64 // Flex weight (ignored if Fixed > 0)
	Min    int     // Minimum size (0 = no minimum)
	Max    int     // Maximum size (0 = no maximum)
}

// Fixed creates a fixed-size spec.
func Fixed(size int) SizeSpec {
	return SizeSpec{Fixed: size}
}

// Flex creates a flexible-size spec with the given weight.
func Flex(weight float64) SizeSpec {
	return SizeSpec{Weight: weight}
}

// FlexWithConstraints creates a flexible-size spec with min/max constraints.
func FlexWithConstraints(weight float64, min, max int) SizeSpec {
	return SizeSpec{Weight: weight, Min: min, Max: max}
}

// Child pairs a size spec with a node.
type Child struct {
	Size SizeSpec
	Node Node
}

// Container holds multiple children in a row or column.
type Container struct {
	Horizontal bool // true = side-by-side, false = stacked
	Children   []Child
	bounds     Rect
}

// HSplit creates a horizontal split (children side-by-side).
func HSplit(children ...Child) *Container {
	return &Container{Horizontal: true, Children: children}
}

// VSplit creates a vertical split (children stacked).
func VSplit(children ...Child) *Container {
	return &Container{Horizontal: false, Children: children}
}

// Layout calculates bounds for this container and all children.
func (c *Container) Layout(bounds Rect) {
	c.bounds = bounds

	if len(c.Children) == 0 {
		return
	}

	// Get total available space
	var totalSpace int
	if c.Horizontal {
		totalSpace = bounds.Width
	} else {
		totalSpace = bounds.Height
	}

	// Calculate sizes with min/max constraints
	sizes := c.calculateSizes(totalSpace)

	// Layout children with calculated sizes
	offset := 0
	for i, child := range c.Children {
		size := sizes[i]

		var childBounds Rect
		if c.Horizontal {
			childBounds = Rect{
				X:      bounds.X + offset,
				Y:      bounds.Y,
				Width:  size,
				Height: bounds.Height,
			}
		} else {
			childBounds = Rect{
				X:      bounds.X,
				Y:      bounds.Y + offset,
				Width:  bounds.Width,
				Height: size,
			}
		}

		child.Node.Layout(childBounds)
		offset += size
	}
}

// calculateSizes distributes space among children respecting min/max constraints.
func (c *Container) calculateSizes(totalSpace int) []int {
	n := len(c.Children)
	sizes := make([]int, n)
	constrained := make([]bool, n) // Whether child hit a min/max constraint

	// First pass: calculate initial sizes and total fixed/flex
	var totalFixed int
	var totalWeight float64
	for i, child := range c.Children {
		if child.Size.Fixed > 0 {
			sizes[i] = child.Size.Fixed
			constrained[i] = true
			totalFixed += child.Size.Fixed
		} else {
			totalWeight += child.Size.Weight
		}
	}

	// Available space for flex children
	available := totalSpace - totalFixed
	if available < 0 {
		available = 0
	}

	// Second pass: calculate flex sizes and apply min/max constraints
	// May need multiple iterations as constraining one child affects others
	for iteration := 0; iteration < n; iteration++ {
		changed := false

		// Calculate unconstrained flex weight
		var unconstrainedWeight float64
		for i, child := range c.Children {
			if !constrained[i] && child.Size.Fixed == 0 {
				unconstrainedWeight += child.Size.Weight
			}
		}

		if unconstrainedWeight == 0 {
			break
		}

		// Distribute available space among unconstrained flex children
		for i, child := range c.Children {
			if constrained[i] || child.Size.Fixed > 0 {
				continue
			}

			// Calculate proportional size
			size := int(float64(available) * child.Size.Weight / unconstrainedWeight)

			// Apply min constraint
			if child.Size.Min > 0 && size < child.Size.Min {
				sizes[i] = child.Size.Min
				constrained[i] = true
				available -= child.Size.Min
				changed = true
				continue
			}

			// Apply max constraint
			if child.Size.Max > 0 && size > child.Size.Max {
				sizes[i] = child.Size.Max
				constrained[i] = true
				available -= child.Size.Max
				changed = true
				continue
			}

			sizes[i] = size
		}

		if !changed {
			break
		}
	}

	// Final pass: ensure total equals totalSpace (give remainder to last unconstrained)
	var total int
	lastUnconstrained := -1
	for i := range sizes {
		total += sizes[i]
		if !constrained[i] {
			lastUnconstrained = i
		}
	}

	if lastUnconstrained >= 0 && total != totalSpace {
		sizes[lastUnconstrained] += totalSpace - total
	}

	return sizes
}

// Bounds returns this container's bounds.
func (c *Container) Bounds() Rect {
	return c.bounds
}

// LeafAt finds the leaf at the given coordinates.
func (c *Container) LeafAt(x, y int) *Leaf {
	if !c.bounds.Contains(x, y) {
		return nil
	}
	for _, child := range c.Children {
		if leaf := child.Node.LeafAt(x, y); leaf != nil {
			return leaf
		}
	}
	return nil
}

// Leaf represents a terminal panel in the layout tree.
type Leaf struct {
	ID     string // Identifier for this panel
	bounds Rect
}

// NewLeaf creates a new leaf with the given ID.
func NewLeaf(id string) *Leaf {
	return &Leaf{ID: id}
}

// Layout stores the bounds for this leaf.
func (l *Leaf) Layout(bounds Rect) {
	l.bounds = bounds
}

// Bounds returns this leaf's bounds.
func (l *Leaf) Bounds() Rect {
	return l.bounds
}

// LeafAt returns this leaf if the point is inside, nil otherwise.
func (l *Leaf) LeafAt(x, y int) *Leaf {
	if l.bounds.Contains(x, y) {
		return l
	}
	return nil
}

// Size returns width and height.
func (l *Leaf) Size() (int, int) {
	return l.bounds.Width, l.bounds.Height
}

// Tree wraps a root node and provides convenience methods.
type Tree struct {
	Root   Node
	leaves map[string]*Leaf
}

// NewTree creates a layout tree, collecting all leaves by ID.
func NewTree(root Node) *Tree {
	t := &Tree{
		Root:   root,
		leaves: make(map[string]*Leaf),
	}
	t.collectLeaves(root)
	return t
}

func (t *Tree) collectLeaves(node Node) {
	switch n := node.(type) {
	case *Leaf:
		t.leaves[n.ID] = n
	case *Container:
		for _, child := range n.Children {
			t.collectLeaves(child.Node)
		}
	}
}

// Layout recalculates the entire tree.
func (t *Tree) Layout(width, height int) {
	t.Root.Layout(Rect{X: 0, Y: 0, Width: width, Height: height})
}

// Get returns the leaf with the given ID.
func (t *Tree) Get(id string) *Leaf {
	return t.leaves[id]
}

// At returns the leaf at the given coordinates.
func (t *Tree) At(x, y int) *Leaf {
	return t.Root.LeafAt(x, y)
}

// Bounds returns the bounds for a panel by ID.
func (t *Tree) Bounds(id string) Rect {
	if leaf := t.leaves[id]; leaf != nil {
		return leaf.Bounds()
	}
	return Rect{}
}

