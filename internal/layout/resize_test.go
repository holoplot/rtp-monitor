package layout

import (
	"errors"
	"testing"
)

func TestResizeGrowAndShrink(t *testing.T) {
	tr := newTree(100, 40)
	_ = tr.AutoPlace(2) // 1 | 2, focus 1

	if err := tr.Resize(Right, true); err != nil {
		t.Fatal(err)
	}
	assertRects(t, tr, map[PaneID]Rect{1: {0, 0, 52, 40}, 2: {52, 0, 48, 40}})

	_ = tr.Resize(Right, false)
	if err := tr.Resize(Right, false); err != nil {
		t.Fatal(err)
	}
	assertRects(t, tr, map[PaneID]Rect{1: {0, 0, 48, 40}, 2: {48, 0, 52, 40}})

	if err := tr.Resize(Left, true); !errors.Is(err, ErrCannotResize) {
		t.Fatalf("Left: err = %v, want ErrCannotResize", err)
	}
	if err := tr.Resize(Up, true); !errors.Is(err, ErrCannotResize) {
		t.Fatalf("Up: err = %v, want ErrCannotResize", err)
	}

	tr.SetFocus(2)
	if err := tr.Resize(Left, true); err != nil {
		t.Fatal(err)
	}
	assertRects(t, tr, map[PaneID]Rect{1: {0, 0, 46, 40}, 2: {46, 0, 54, 40}})
}

func TestResizeClampsToMinimum(t *testing.T) {
	tr := newTree(100, 40)
	_ = tr.AutoPlace(2)
	for range 20 {
		_ = tr.Resize(Right, true)
	}
	assertRects(t, tr, map[PaneID]Rect{1: {0, 0, 80, 40}, 2: {80, 0, 20, 40}})
	if err := tr.Resize(Right, true); !errors.Is(err, ErrNoRoom) {
		t.Fatalf("err = %v, want ErrNoRoom", err)
	}
}

func TestResizeUsesNearestMatchingAncestor(t *testing.T) {
	tr := fourPanes(t) // 1/3 | 2/4, focus 1
	if err := tr.Resize(Down, true); err != nil {
		t.Fatal(err)
	}
	assertRects(t, tr, map[PaneID]Rect{
		1: {0, 0, 50, 21}, 3: {0, 21, 50, 19},
		2: {50, 0, 50, 20}, 4: {50, 20, 50, 20},
	})
	if err := tr.Resize(Right, true); err != nil {
		t.Fatal(err)
	}
	assertRects(t, tr, map[PaneID]Rect{
		1: {0, 0, 52, 21}, 3: {0, 21, 52, 19},
		2: {52, 0, 48, 20}, 4: {52, 20, 48, 20},
	})
}

func TestResizeAll(t *testing.T) {
	tr := newTree(100, 40)
	_ = tr.AutoPlace(2)
	if err := tr.ResizeAll(true); err != nil {
		t.Fatal(err)
	}
	assertRects(t, tr, map[PaneID]Rect{1: {0, 0, 52, 40}, 2: {52, 0, 48, 40}})
	if err := newTree(100, 40).ResizeAll(true); !errors.Is(err, ErrNoRoom) {
		t.Fatalf("single pane: err = %v, want ErrNoRoom", err)
	}
}

func TestFlip(t *testing.T) {
	tr := newTree(100, 40)
	if err := tr.Flip(); !errors.Is(err, ErrNoParent) {
		t.Fatalf("single pane: err = %v, want ErrNoParent", err)
	}
	_ = tr.AutoPlace(2)
	if err := tr.Flip(); err != nil {
		t.Fatal(err)
	}
	assertRects(t, tr, map[PaneID]Rect{1: {0, 0, 100, 20}, 2: {0, 20, 100, 20}})

	low := newTree(100, 8)
	_ = low.AutoPlace(2)
	if err := low.Flip(); !errors.Is(err, ErrNoRoom) {
		t.Fatalf("too low: err = %v, want ErrNoRoom", err)
	}
	assertRects(t, low, map[PaneID]Rect{1: {0, 0, 50, 8}, 2: {50, 0, 50, 8}})
}

func TestZoom(t *testing.T) {
	tr := fourPanes(t)
	tr.ToggleZoom()
	if !tr.Zoomed() {
		t.Fatal("not zoomed")
	}
	assertRects(t, tr, map[PaneID]Rect{1: {0, 0, 100, 40}})

	tr.MoveFocus(Right)
	if tr.Zoomed() || tr.Focus() != 2 {
		t.Fatalf("zoomed = %v, focus = %d after moving focus", tr.Zoomed(), tr.Focus())
	}

	tr.ToggleZoom()
	_ = tr.AutoPlace(5)
	if tr.Zoomed() {
		t.Fatal("still zoomed after adding a pane")
	}

	tr.ToggleZoom()
	_ = tr.Close(4)
	if tr.Zoomed() {
		t.Fatal("still zoomed after closing a pane")
	}
}
