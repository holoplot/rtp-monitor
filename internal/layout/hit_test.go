package layout

import "testing"

func TestPaneAt(t *testing.T) {
	tr := newTree(100, 40)
	_ = tr.AutoPlace(2)
	cases := []struct {
		x, y int
		want PaneID
		ok   bool
	}{{10, 10, 1, true}, {50, 0, 2, true}, {100, 0, 0, false}, {0, 40, 0, false}}
	for _, c := range cases {
		if got, ok := tr.PaneAt(c.x, c.y); got != c.want || ok != c.ok {
			t.Errorf("PaneAt(%d, %d) = %d, %v; want %d, %v", c.x, c.y, got, ok, c.want, c.ok)
		}
	}
	tr.ToggleZoom()
	if got, _ := tr.PaneAt(70, 10); got != 1 {
		t.Errorf("zoomed PaneAt(70, 10) = %d, want 1", got)
	}
}

func TestBorderAtAndDrag(t *testing.T) {
	tr := newTree(100, 40)
	_ = tr.AutoPlace(2)
	for _, x := range []int{49, 50} {
		if _, ok := tr.BorderAt(x, 10); !ok {
			t.Errorf("no border at x=%d", x)
		}
	}
	for _, x := range []int{0, 48, 51, 99} {
		if _, ok := tr.BorderAt(x, 10); ok {
			t.Errorf("unexpected border at x=%d", x)
		}
	}

	b, _ := tr.BorderAt(49, 10)
	tr.DragBorder(b, 69, 10)
	assertRects(t, tr, map[PaneID]Rect{1: {0, 0, 70, 40}, 2: {70, 0, 30, 40}})
	tr.DragBorder(b, 5, 10)
	assertRects(t, tr, map[PaneID]Rect{1: {0, 0, 20, 40}, 2: {20, 0, 80, 40}})
}

func TestBorderAtNested(t *testing.T) {
	tr := fourPanes(t) // 1/3 | 2/4

	b, ok := tr.BorderAt(10, 19)
	if !ok {
		t.Fatal("no border between 1 and 3")
	}
	tr.DragBorder(b, 10, 29)
	assertRects(t, tr, map[PaneID]Rect{
		1: {0, 0, 50, 30}, 3: {0, 30, 50, 10},
		2: {50, 0, 50, 20}, 4: {50, 20, 50, 20},
	})

	// At a crossing the outer (root) split wins.
	b, _ = tr.BorderAt(49, 19)
	tr.DragBorder(b, 59, 19)
	assertRects(t, tr, map[PaneID]Rect{
		1: {0, 0, 60, 30}, 3: {0, 30, 60, 10},
		2: {60, 0, 40, 20}, 4: {60, 20, 40, 20},
	})
}

func TestNoBordersWhileZoomed(t *testing.T) {
	tr := newTree(100, 40)
	_ = tr.AutoPlace(2)
	tr.ToggleZoom()
	if _, ok := tr.BorderAt(49, 10); ok {
		t.Fatal("border while zoomed")
	}
}

func TestDragBorderAfterClose(t *testing.T) {
	tr := newTree(100, 40)
	_ = tr.AutoPlace(2)
	b, _ := tr.BorderAt(49, 10)
	_ = tr.Close(2)
	tr.DragBorder(b, 30, 10) // must not panic or change anything
	assertRects(t, tr, map[PaneID]Rect{1: {0, 0, 100, 40}})
	tr.DragBorder(Border{}, 30, 10)
}
