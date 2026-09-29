package layout

const (
	resizeStepX = 2
	resizeStepY = 1
)

// Resize moves the edge on side d of the focused pane outward (grow) or
// inward (shrink) by one step: 2 columns or 1 row. The edge belongs to the
// nearest ancestor split on that axis where the pane borders side d. The move
// is clamped to the minimum pane size.
func (t *Tree) Resize(d Direction, grow bool) error {
	child := t.find(t.focus)
	if child == nil {
		return ErrUnknownPane
	}
	for p := child.parent; p != nil; child, p = p, p.parent {
		if (d == Left || d == Right) != (p.split == SplitHorizontal) {
			continue
		}
		if (d == Right || d == Down) != (p.first == child) {
			continue
		}

		step := resizeStepX
		if p.split == SplitVertical {
			step = resizeStepY
		}
		if d == Left || d == Up {
			step = -step
		}
		if !grow {
			step = -step
		}

		return t.setFirstSize(p, t.firstSize(p)+step)
	}

	return ErrCannotResize
}

// ResizeAll grows or shrinks the focused pane on both axes, using whichever
// side of each axis can move.
func (t *Tree) ResizeAll(grow bool) error {
	moved := false
	for _, sides := range [][2]Direction{{Right, Left}, {Down, Up}} {
		for _, d := range sides {
			if t.Resize(d, grow) == nil {
				moved = true
				break
			}
		}
	}
	if !moved {
		return ErrNoRoom
	}
	return nil
}

// Flip toggles the orientation of the focused pane's parent split.
func (t *Tree) Flip() error {
	n := t.find(t.focus)
	if n == nil {
		return ErrUnknownPane
	}
	p := n.parent
	if p == nil {
		return ErrNoParent
	}

	old := p.split
	if old == SplitHorizontal {
		p.split = SplitVertical
	} else {
		p.split = SplitHorizontal
	}
	t.relayout()

	if !t.fits() {
		p.split = old
		t.relayout()
		return ErrNoRoom
	}
	return nil
}

// ToggleZoom shows the focused pane alone, or returns to the full layout.
func (t *Tree) ToggleZoom() { t.zoomed = !t.zoomed && t.root != nil }

// Zoomed reports whether the focused pane is shown alone.
func (t *Tree) Zoomed() bool { return t.zoomed }

func (t *Tree) firstSize(p *node) int {
	a, _ := splitRect(t.rects[p], p.split, p.ratio)
	return extent(a, p.split)
}

func extent(r Rect, s Split) int {
	if s == SplitHorizontal {
		return r.W
	}
	return r.H
}

// setFirstSize sets the size of p's first child to target, clamped so both
// children keep their minimum sizes. If that still makes a nested pane too
// small (nested ratios round independently), it backs off one cell at a time
// toward the current size. It returns ErrNoRoom if the boundary cannot move.
func (t *Tree) setFirstSize(p *node, target int) error {
	total := extent(t.rects[p], p.split)
	cur := t.firstSize(p)
	lo, hi := minExtent(p.first, p.split), total-minExtent(p.second, p.split)
	if hi < lo {
		return ErrNoRoom
	}
	target = max(lo, min(target, hi))

	oldRatio := p.ratio
	for v := target; v != cur; {
		p.ratio = float64(v) / float64(total)
		t.relayout()
		if t.fits() {
			return nil
		}
		if v > cur {
			v--
		} else {
			v++
		}
	}

	p.ratio = oldRatio
	t.relayout()
	return ErrNoRoom
}

// minExtent is the smallest size of subtree n along the axis of split s.
func minExtent(n *node, s Split) int {
	if n.leaf() {
		if s == SplitHorizontal {
			return MinWidth
		}
		return MinHeight
	}
	a, b := minExtent(n.first, s), minExtent(n.second, s)
	if n.split == s {
		return a + b
	}
	return max(a, b)
}
