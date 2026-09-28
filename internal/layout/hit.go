package layout

// Border identifies the boundary of one split, for dragging with the mouse.
type Border struct{ n *node }

// PaneAt returns the visible pane containing cell (x, y).
func (t *Tree) PaneAt(x, y int) (PaneID, bool) {
	for id, r := range t.Rects() {
		if r.Contains(x, y) {
			return id, true
		}
	}
	return 0, false
}

// TitleAt returns the visible pane whose title bar (its top border row, i.e.
// y == rect.Y and rect.X <= x < rect.X+rect.W) contains cell (x, y). Nothing
// is draggable while zoomed.
func (t *Tree) TitleAt(x, y int) (PaneID, bool) {
	if t.zoomed {
		return 0, false
	}
	for id, r := range t.Rects() {
		if y == r.Y && x >= r.X && x < r.X+r.W {
			return id, true
		}
	}
	return 0, false
}

// BorderAt returns the split whose boundary is at cell (x, y). Both border
// columns (or rows) at a boundary count: the first pane's right or bottom
// border and the second pane's left or top border. Where boundaries cross,
// the outer split wins. Nothing is draggable while zoomed.
func (t *Tree) BorderAt(x, y int) (Border, bool) {
	if t.zoomed {
		return Border{}, false
	}
	return t.borderAt(t.root, x, y)
}

func (t *Tree) borderAt(n *node, x, y int) (Border, bool) {
	if n.leaf() || !t.rects[n].Contains(x, y) {
		return Border{}, false
	}

	second := t.rects[n.second]
	if n.split == SplitHorizontal && (x == second.X-1 || x == second.X) {
		return Border{n}, true
	}
	if n.split == SplitVertical && (y == second.Y-1 || y == second.Y) {
		return Border{n}, true
	}

	if b, ok := t.borderAt(n.first, x, y); ok {
		return b, true
	}
	return t.borderAt(n.second, x, y)
}

// DragBorder moves boundary b so that the first pane's right (or bottom)
// border is at column x (or row y), clamped to the minimum pane size. Borders
// of splits that no longer exist are ignored.
func (t *Tree) DragBorder(b Border, x, y int) {
	r, ok := t.rects[b.n]
	if !ok || b.n.leaf() {
		return
	}

	target := x - r.X + 1
	if b.n.split == SplitVertical {
		target = y - r.Y + 1
	}
	_ = t.setFirstSize(b.n, target)
}
