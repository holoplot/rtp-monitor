package layout

import "testing"

// fourPanes returns the layout 1/3 | 2/4 on a 100×40 area with focus on 1.
func fourPanes(t *testing.T) *Tree {
	t.Helper()
	tr := newTree(100, 40)
	for id := PaneID(2); id <= 4; id++ {
		if err := tr.AutoPlace(id); err != nil {
			t.Fatal(err)
		}
	}
	return tr
}

func TestNeighbor(t *testing.T) {
	tr := fourPanes(t)
	tests := []struct {
		from PaneID
		d    Direction
		want PaneID
		ok   bool
	}{
		{1, Right, 2, true}, {1, Down, 3, true}, {1, Left, 0, false}, {1, Up, 0, false},
		{4, Left, 3, true}, {4, Up, 2, true}, {3, Right, 4, true}, {2, Down, 4, true},
	}
	for _, tt := range tests {
		got, ok := tr.Neighbor(tt.from, tt.d)
		if got != tt.want || ok != tt.ok {
			t.Errorf("Neighbor(%d, %d) = %d, %v; want %d, %v", tt.from, tt.d, got, ok, tt.want, tt.ok)
		}
	}
}

func TestNeighborTieUsesTreeOrder(t *testing.T) {
	tr := newTree(100, 40)
	_ = tr.AutoPlace(2)               // 1 | 2
	_ = tr.Split(2, 3, SplitVertical) // 1 | 2/3
	if got, _ := tr.Neighbor(1, Right); got != 2 {
		t.Fatalf("Neighbor(1, Right) = %d, want 2", got)
	}
	if got, _ := tr.Neighbor(3, Left); got != 1 {
		t.Fatalf("Neighbor(3, Left) = %d, want 1", got)
	}
}

func TestMoveFocus(t *testing.T) {
	tr := fourPanes(t)
	if !tr.MoveFocus(Right) || tr.Focus() != 2 {
		t.Fatalf("focus = %d, want 2", tr.Focus())
	}
	if tr.MoveFocus(Right) || tr.Focus() != 2 {
		t.Fatalf("moving past the edge changed focus to %d", tr.Focus())
	}
}

func TestSwap(t *testing.T) {
	tr := newTree(100, 40)
	_ = tr.AutoPlace(2)
	if !tr.Swap(Right) {
		t.Fatal("Swap(Right) = false")
	}
	assertRects(t, tr, map[PaneID]Rect{1: {50, 0, 50, 40}, 2: {0, 0, 50, 40}})
	if tr.Focus() != 1 {
		t.Fatalf("focus = %d, want 1 (focus follows the pane)", tr.Focus())
	}
	if tr.Swap(Right) {
		t.Fatal("Swap(Right) at the edge = true")
	}
}

func TestSwapNext(t *testing.T) {
	tr := fourPanes(t) // tree order 1, 3, 2, 4
	if !tr.SwapNext() {
		t.Fatal("SwapNext() = false")
	}
	assertRects(t, tr, map[PaneID]Rect{
		3: {0, 0, 50, 20}, 1: {0, 20, 50, 20},
		2: {50, 0, 50, 20}, 4: {50, 20, 50, 20},
	})
	if New(1).SwapNext() {
		t.Fatal("SwapNext() with one pane = true")
	}
}
