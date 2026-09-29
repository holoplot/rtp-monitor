package layout

import (
	"errors"
	"reflect"
	"testing"
)

func newTree(w, h int) *Tree {
	t := New(1)
	t.SetArea(Rect{W: w, H: h})
	return t
}

func assertRects(t *testing.T, tr *Tree, want map[PaneID]Rect) {
	t.Helper()
	if got := tr.Rects(); !reflect.DeepEqual(got, want) {
		t.Fatalf("rects\n got: %v\nwant: %v", got, want)
	}
}

func TestNewTreeFillsArea(t *testing.T) {
	tr := newTree(100, 40)
	assertRects(t, tr, map[PaneID]Rect{1: {0, 0, 100, 40}})
	if tr.Focus() != 1 {
		t.Fatalf("focus = %d, want 1", tr.Focus())
	}
}

func TestAutoPlace(t *testing.T) {
	tr := newTree(100, 40)
	for id := PaneID(2); id <= 4; id++ {
		if err := tr.AutoPlace(id); err != nil {
			t.Fatalf("AutoPlace(%d): %v", id, err)
		}
	}
	assertRects(t, tr, map[PaneID]Rect{
		1: {0, 0, 50, 20}, 3: {0, 20, 50, 20},
		2: {50, 0, 50, 20}, 4: {50, 20, 50, 20},
	})
	if tr.Focus() != 1 {
		t.Fatalf("focus = %d, want 1", tr.Focus())
	}
	if got, want := tr.Panes(), []PaneID{1, 3, 2, 4}; !reflect.DeepEqual(got, want) {
		t.Fatalf("Panes() = %v, want %v", got, want)
	}
}

func TestAutoPlaceRefusesWhenTooSmall(t *testing.T) {
	tr := newTree(30, 8)
	if err := tr.AutoPlace(2); !errors.Is(err, ErrNoRoom) {
		t.Fatalf("err = %v, want ErrNoRoom", err)
	}
	assertRects(t, tr, map[PaneID]Rect{1: {0, 0, 30, 8}})
}

func TestSplitErrors(t *testing.T) {
	tr := newTree(100, 40)
	if err := tr.Split(9, 2, SplitHorizontal); !errors.Is(err, ErrUnknownPane) {
		t.Errorf("unknown target: err = %v", err)
	}
	if err := tr.Split(1, 1, SplitHorizontal); err == nil {
		t.Error("duplicate id: expected error")
	}
}

func TestClose(t *testing.T) {
	tr := newTree(100, 40)
	_ = tr.AutoPlace(2) // 1 | 2
	_ = tr.AutoPlace(3) // 1/3 | 2
	tr.SetFocus(3)

	if err := tr.Close(3); err != nil {
		t.Fatal(err)
	}
	assertRects(t, tr, map[PaneID]Rect{1: {0, 0, 50, 40}, 2: {50, 0, 50, 40}})
	if tr.Focus() != 1 {
		t.Fatalf("focus = %d, want 1", tr.Focus())
	}

	if err := tr.Close(1); err != nil {
		t.Fatal(err)
	}
	assertRects(t, tr, map[PaneID]Rect{2: {0, 0, 100, 40}})
	if tr.Focus() != 2 {
		t.Fatalf("focus = %d, want 2", tr.Focus())
	}

	if err := tr.Close(7); !errors.Is(err, ErrUnknownPane) {
		t.Errorf("unknown pane: err = %v", err)
	}
}

func TestCycleFocus(t *testing.T) {
	tr := newTree(100, 40)
	_ = tr.AutoPlace(2)
	_ = tr.AutoPlace(3) // tree order 1, 3, 2

	for _, want := range []PaneID{3, 2, 1} {
		tr.CycleFocus(true)
		if tr.Focus() != want {
			t.Fatalf("forward: focus = %d, want %d", tr.Focus(), want)
		}
	}
	tr.CycleFocus(false)
	if tr.Focus() != 2 {
		t.Fatalf("backward: focus = %d, want 2", tr.Focus())
	}
	if tr.SetFocus(9) {
		t.Fatal("SetFocus(unknown) = true")
	}
}

func TestAreaChangeScales(t *testing.T) {
	tr := newTree(100, 40)
	_ = tr.AutoPlace(2)
	tr.SetArea(Rect{W: 60, H: 20})
	assertRects(t, tr, map[PaneID]Rect{1: {0, 0, 30, 20}, 2: {30, 0, 30, 20}})
}

func TestEmptyTree(t *testing.T) {
	tr := NewEmpty()
	tr.SetArea(Rect{W: 80, H: 24})
	if tr.Focus() != 0 || len(tr.Panes()) != 0 || len(tr.Rects()) != 0 {
		t.Fatal("empty tree not empty")
	}
	tr.CycleFocus(true)
	tr.MoveFocus(Right)
	tr.ToggleZoom()
	if tr.Zoomed() || tr.Resize(Right, true) == nil || tr.Flip() == nil || tr.SwapNext() {
		t.Fatal("operation on empty tree succeeded")
	}
	if _, ok := tr.BorderAt(1, 1); ok {
		t.Fatal("border in empty tree")
	}

	if err := tr.AutoPlace(3); err != nil || tr.Focus() != 3 || tr.Rects()[3] != (Rect{W: 80, H: 24}) {
		t.Fatalf("AutoPlace into empty tree: err=%v focus=%d rects=%v", err, tr.Focus(), tr.Rects())
	}
	if err := tr.Close(3); err != nil || tr.Focus() != 0 || len(tr.Panes()) != 0 {
		t.Fatalf("closing the last pane: err=%v focus=%d", err, tr.Focus())
	}
}
