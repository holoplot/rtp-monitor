package layout

import "slices"

// Direction is a side of a pane.
type Direction int

const (
	Left Direction = iota
	Right
	Up
	Down
)

// Neighbor returns the pane adjacent to id on side d: of all panes whose
// rectangle touches that side, the one overlapping it most along the other
// axis. Ties go to the pane that comes first in tree order.
func (t *Tree) Neighbor(id PaneID, d Direction) (PaneID, bool) {
	n := t.find(id)
	if n == nil {
		return 0, false
	}
	r := t.rects[n]

	var best PaneID
	bestOverlap := 0

	for _, c := range t.leafNodes() {
		if c == n {
			continue
		}
		cr := t.rects[c]

		var adjacent bool
		var o int
		switch d {
		case Left:
			adjacent, o = cr.X+cr.W == r.X, overlap(r.Y, r.H, cr.Y, cr.H)
		case Right:
			adjacent, o = cr.X == r.X+r.W, overlap(r.Y, r.H, cr.Y, cr.H)
		case Up:
			adjacent, o = cr.Y+cr.H == r.Y, overlap(r.X, r.W, cr.X, cr.W)
		case Down:
			adjacent, o = cr.Y == r.Y+r.H, overlap(r.X, r.W, cr.X, cr.W)
		}

		if adjacent && o > bestOverlap {
			best, bestOverlap = c.pane, o
		}
	}

	return best, bestOverlap > 0
}

// MoveFocus focuses the neighbour of the focused pane on side d.
func (t *Tree) MoveFocus(d Direction) bool {
	id, ok := t.Neighbor(t.focus, d)
	if !ok {
		return false
	}
	return t.SetFocus(id)
}

// Swap exchanges the focused pane with its neighbour on side d. Focus follows
// the moved pane.
func (t *Tree) Swap(d Direction) bool {
	id, ok := t.Neighbor(t.focus, d)
	if !ok {
		return false
	}
	t.swapPanes(t.focus, id)
	return true
}

// SwapNext exchanges the focused pane with the next pane in tree order.
func (t *Tree) SwapNext() bool {
	ids := t.Panes()
	if len(ids) < 2 {
		return false
	}
	i := slices.Index(ids, t.focus)
	t.swapPanes(t.focus, ids[(i+1)%len(ids)])
	return true
}

// SwapPanes exchanges panes a and b. It returns false if either pane is
// unknown or a == b. Focus is unchanged, so it stays on the same pane ID.
func (t *Tree) SwapPanes(a, b PaneID) bool {
	if a == b || t.find(a) == nil || t.find(b) == nil {
		return false
	}
	t.swapPanes(a, b)
	return true
}

func (t *Tree) swapPanes(a, b PaneID) {
	na, nb := t.find(a), t.find(b)
	na.pane, nb.pane = nb.pane, na.pane
}

// overlap returns the length of the intersection of [a, a+al) and [b, b+bl).
func overlap(a, al, b, bl int) int {
	return max(0, min(a+al, b+bl)-max(a, b))
}
