// Package layout implements the tiling split tree behind the pane UI. It is
// pure geometry: it knows pane IDs, split directions and ratios, and computes
// rectangles for a given area. It does not render anything.
package layout

import (
	"errors"
	"fmt"
	"math"
	"slices"
)

// PaneID identifies a pane in the tree.
type PaneID int

// Split is the orientation of an inner node.
type Split int

const (
	// SplitHorizontal places the children side by side: first | second.
	SplitHorizontal Split = iota
	// SplitVertical stacks the children: first above second.
	SplitVertical
)

// Rect is a rectangle in terminal cells.
type Rect struct{ X, Y, W, H int }

// Contains reports whether the cell (x, y) lies inside r.
func (r Rect) Contains(x, y int) bool {
	return x >= r.X && x < r.X+r.W && y >= r.Y && y < r.Y+r.H
}

const (
	// MinWidth and MinHeight are the smallest pane size, border included.
	MinWidth  = 20
	MinHeight = 5
)

var (
	ErrNoRoom       = errors.New("no room: close or resize a pane")
	ErrLastPane     = errors.New("the last pane cannot be closed")
	ErrNoParent     = errors.New("pane is not part of a split")
	ErrUnknownPane  = errors.New("unknown pane")
	ErrCannotResize = errors.New("no split to resize in that direction")
)

type node struct {
	parent        *node
	pane          PaneID  // leaves only
	split         Split   // inner nodes only
	ratio         float64 // share of the first child; inner nodes only
	first, second *node
}

func (n *node) leaf() bool { return n.first == nil }

// Tree is a binary split tree of panes with a focused pane and optional zoom.
type Tree struct {
	root   *node
	focus  PaneID
	zoomed bool
	area   Rect
	rects  map[*node]Rect // every node's rectangle in the unzoomed layout
}

// New returns a tree containing the single pane root, which has focus.
func New(root PaneID) *Tree {
	t := &Tree{root: &node{pane: root}, focus: root}
	t.relayout()
	return t
}

// SetArea sets the rectangle the tree is laid out in.
func (t *Tree) SetArea(r Rect) {
	t.area = r
	t.relayout()
}

// Area returns the rectangle the tree is laid out in.
func (t *Tree) Area() Rect { return t.area }

// Focus returns the focused pane.
func (t *Tree) Focus() PaneID { return t.focus }

// SetFocus focuses id; changing focus leaves zoom. It returns false if id is
// not in the tree.
func (t *Tree) SetFocus(id PaneID) bool {
	if t.find(id) == nil {
		return false
	}
	if id != t.focus {
		t.focus = id
		t.zoomed = false
	}
	return true
}

// Contains reports whether id is in the tree.
func (t *Tree) Contains(id PaneID) bool { return t.find(id) != nil }

// Panes returns all pane IDs in tree order (depth first, first child first).
func (t *Tree) Panes() []PaneID {
	var ids []PaneID
	for _, n := range t.leafNodes() {
		ids = append(ids, n.pane)
	}
	return ids
}

// Rects returns the rectangle of every visible pane. While zoomed only the
// focused pane is visible, and it fills the whole area.
func (t *Tree) Rects() map[PaneID]Rect {
	out := make(map[PaneID]Rect)
	if t.zoomed {
		out[t.focus] = t.area
		return out
	}
	for _, n := range t.leafNodes() {
		out[n.pane] = t.rects[n]
	}
	return out
}

// CycleFocus moves focus to the next (or previous) pane in tree order.
func (t *Tree) CycleFocus(forward bool) {
	ids := t.Panes()
	i := slices.Index(ids, t.focus)
	if forward {
		i = (i + 1) % len(ids)
	} else {
		i = (i - 1 + len(ids)) % len(ids)
	}
	t.SetFocus(ids[i])
}

// Split splits pane target in two halves and puts the new pane id second
// (right of or below target). It fails with ErrNoRoom if a pane would end up
// smaller than MinWidth × MinHeight. A successful split leaves zoom.
func (t *Tree) Split(target, id PaneID, s Split) error {
	n := t.find(target)
	if n == nil {
		return ErrUnknownPane
	}
	if t.find(id) != nil {
		return fmt.Errorf("pane %d already exists", id)
	}

	saved := *n
	n.first = &node{parent: n, pane: n.pane}
	n.second = &node{parent: n, pane: id}
	n.split, n.ratio = s, 0.5
	t.relayout()

	if !t.fits() {
		*n = saved
		t.relayout()
		return ErrNoRoom
	}

	t.zoomed = false
	return nil
}

// AutoPlace inserts pane id by splitting the largest pane along its longer
// side. Cells are about twice as tall as wide, so the width is compared with
// twice the height. Ties prefer the focused pane, then the lowest ID.
func (t *Tree) AutoPlace(id PaneID) error {
	var best *node
	bestArea := -1

	for _, n := range t.leafNodes() {
		r := t.rects[n]
		area := r.W * r.H
		switch {
		case area > bestArea:
			best, bestArea = n, area
		case area == bestArea && best.pane != t.focus && (n.pane == t.focus || n.pane < best.pane):
			best = n
		}
	}

	r := t.rects[best]
	s := SplitVertical
	if r.W >= 2*r.H {
		s = SplitHorizontal
	}

	return t.Split(best.pane, id, s)
}

// Close removes pane id; its sibling takes over the parent's space. If id had
// focus, focus moves to the first pane of that sibling. Closing leaves zoom.
func (t *Tree) Close(id PaneID) error {
	n := t.find(id)
	if n == nil {
		return ErrUnknownPane
	}

	p := n.parent
	if p == nil {
		return ErrLastPane
	}

	sibling := p.first
	if sibling == n {
		sibling = p.second
	}
	t.replace(p, sibling)

	if t.focus == id {
		t.focus = firstLeaf(sibling).pane
	}
	t.zoomed = false
	t.relayout()

	return nil
}

// replace puts n where old is in the tree.
func (t *Tree) replace(old, n *node) {
	n.parent = old.parent
	switch {
	case old.parent == nil:
		t.root = n
	case old.parent.first == old:
		old.parent.first = n
	default:
		old.parent.second = n
	}
}

func firstLeaf(n *node) *node {
	for !n.leaf() {
		n = n.first
	}
	return n
}

func (t *Tree) find(id PaneID) *node {
	for _, n := range t.leafNodes() {
		if n.pane == id {
			return n
		}
	}
	return nil
}

func (t *Tree) leafNodes() []*node {
	var out []*node
	var walk func(n *node)
	walk = func(n *node) {
		if n.leaf() {
			out = append(out, n)
			return
		}
		walk(n.first)
		walk(n.second)
	}
	walk(t.root)
	return out
}

func (t *Tree) relayout() {
	t.rects = make(map[*node]Rect)
	t.layoutNode(t.root, t.area)
}

func (t *Tree) layoutNode(n *node, r Rect) {
	t.rects[n] = r
	if n.leaf() {
		return
	}
	a, b := splitRect(r, n.split, n.ratio)
	t.layoutNode(n.first, a)
	t.layoutNode(n.second, b)
}

func splitRect(r Rect, s Split, ratio float64) (Rect, Rect) {
	if s == SplitHorizontal {
		w := int(math.Round(float64(r.W) * ratio))
		return Rect{r.X, r.Y, w, r.H}, Rect{r.X + w, r.Y, r.W - w, r.H}
	}
	h := int(math.Round(float64(r.H) * ratio))
	return Rect{r.X, r.Y, r.W, h}, Rect{r.X, r.Y + h, r.W, r.H - h}
}

// fits reports whether every pane is at least MinWidth × MinHeight.
func (t *Tree) fits() bool {
	for _, n := range t.leafNodes() {
		if r := t.rects[n]; r.W < MinWidth || r.H < MinHeight {
			return false
		}
	}
	return true
}
