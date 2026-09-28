package ui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/holoplot/rtp-monitor/internal/layout"
	"github.com/holoplot/rtp-monitor/internal/ui/pane"
)

func TestClickFocusesAndSelects(t *testing.T) {
	h := newHarness(t, 120, 41)
	h.press("m") // list {0,0,60,40} | meters {60,0,60,40}

	h.send(tea.MouseClickMsg{X: 70, Y: 10, Button: tea.MouseLeft})
	if h.app.tree.Focus() != 2 {
		t.Fatalf("focus = %d, want 2", h.app.tree.Focus())
	}

	// Screen rows of the list: 0 border, 1 header, 2 Alpha, 3 Beta.
	h.send(tea.MouseClickMsg{X: 5, Y: 3, Button: tea.MouseLeft})
	if h.app.tree.Focus() != 1 || h.app.list.Selected().ID != "beta" {
		t.Fatalf("focus %d, selected %q", h.app.tree.Focus(), h.app.list.Selected().ID)
	}

	h.send(tea.MouseClickMsg{X: 5, Y: 40, Button: tea.MouseLeft}) // status bar
	if h.app.tree.Focus() != 1 {
		t.Fatal("click on the status bar changed focus")
	}
}

func TestDragBorder(t *testing.T) {
	h := newHarness(t, 120, 41)
	h.press("m")

	h.send(tea.MouseClickMsg{X: 59, Y: 10, Button: tea.MouseLeft})
	h.send(tea.MouseMotionMsg{X: 79, Y: 10, Button: tea.MouseLeft})
	if w := h.rects()[1].W; w != 80 {
		t.Fatalf("width after drag %d, want 80", w)
	}
	h.send(tea.MouseReleaseMsg{X: 79, Y: 10, Button: tea.MouseLeft})
	h.send(tea.MouseMotionMsg{X: 90, Y: 10})
	if w := h.rects()[1].W; w != 80 {
		t.Fatalf("width changed after release: %d", w)
	}
}

func TestDragPaneTitleSwaps(t *testing.T) {
	h := newHarness(t, 120, 41)
	h.press("m") // list {0,0,60,40} | meters {60,0,60,40}

	h.send(tea.MouseClickMsg{X: 70, Y: 0, Button: tea.MouseLeft})
	h.send(tea.MouseMotionMsg{X: 10, Y: 10, Button: tea.MouseLeft})
	if !strings.Contains(h.screen(), "drop on a pane to swap") {
		t.Fatalf("no drag hint:\n%s", h.screen())
	}
	h.send(tea.MouseReleaseMsg{X: 10, Y: 10, Button: tea.MouseLeft})

	want := map[layout.PaneID]layout.Rect{
		1: {X: 60, Y: 0, W: 60, H: 40},
		2: {X: 0, Y: 0, W: 60, H: 40},
	}
	if got := h.rects(); got[1] != want[1] || got[2] != want[2] {
		t.Fatalf("rects after swap = %v", got)
	}
	if h.app.tree.Focus() != 2 {
		t.Fatalf("focus = %d, want 2 (follows the dragged pane)", h.app.tree.Focus())
	}
	if strings.Contains(h.screen(), "drop on a pane to swap") {
		t.Fatal("drag hint still shown after release")
	}
}

func TestDragPaneTitleReleaseOnSourceCancels(t *testing.T) {
	h := newHarness(t, 120, 41)
	h.press("m")

	h.send(tea.MouseClickMsg{X: 70, Y: 0, Button: tea.MouseLeft})
	h.send(tea.MouseReleaseMsg{X: 80, Y: 10, Button: tea.MouseLeft})

	want := map[layout.PaneID]layout.Rect{
		1: {X: 0, Y: 0, W: 60, H: 40},
		2: {X: 60, Y: 0, W: 60, H: 40},
	}
	if got := h.rects(); got[1] != want[1] || got[2] != want[2] {
		t.Fatalf("rects changed = %v", got)
	}
	if h.app.tree.Focus() != 2 {
		t.Fatalf("focus = %d, want 2", h.app.tree.Focus())
	}
}

func TestDragPaneTitleReleaseOutsidePanesCancels(t *testing.T) {
	h := newHarness(t, 120, 41)
	h.press("m")

	h.send(tea.MouseClickMsg{X: 70, Y: 0, Button: tea.MouseLeft})
	h.send(tea.MouseMotionMsg{X: 10, Y: 10, Button: tea.MouseLeft})
	h.send(tea.MouseReleaseMsg{X: 10, Y: 40, Button: tea.MouseLeft}) // status bar

	want := map[layout.PaneID]layout.Rect{
		1: {X: 0, Y: 0, W: 60, H: 40},
		2: {X: 60, Y: 0, W: 60, H: 40},
	}
	if got := h.rects(); got[1] != want[1] || got[2] != want[2] {
		t.Fatalf("rects changed = %v", got)
	}
}

func TestDragPaneTitleDisabledWhileZoomed(t *testing.T) {
	h := newHarness(t, 120, 41)
	h.press("m")
	h.press("ctrl+w", "f", "esc") // zoom in on the focused (list) pane

	h.send(tea.MouseClickMsg{X: 70, Y: 0, Button: tea.MouseLeft})
	h.send(tea.MouseMotionMsg{X: 10, Y: 10, Button: tea.MouseLeft})
	h.send(tea.MouseReleaseMsg{X: 10, Y: 10, Button: tea.MouseLeft})

	h.press("ctrl+w", "f", "esc") // zoom back out to inspect the layout

	want := map[layout.PaneID]layout.Rect{
		1: {X: 0, Y: 0, W: 60, H: 40},
		2: {X: 60, Y: 0, W: 60, H: 40},
	}
	if got := h.rects(); got[1] != want[1] || got[2] != want[2] {
		t.Fatalf("rects changed while zoomed = %v", got)
	}
}

func TestWheelScrollsPaneUnderPointer(t *testing.T) {
	h := newHarness(t, 120, 41)
	h.press("m")

	h.send(tea.MouseWheelMsg{X: 70, Y: 10, Button: tea.MouseWheelDown})
	h.send(tea.MouseWheelMsg{X: 70, Y: 10, Button: tea.MouseWheelUp})
	msgs := h.panes[0].msgs
	if len(msgs) != 2 || msgs[0] != (pane.ScrollMsg{Lines: 3}) || msgs[1] != (pane.ScrollMsg{Lines: -3}) {
		t.Fatalf("pane got %v", msgs)
	}
	if h.app.tree.Focus() != 1 {
		t.Fatal("wheel changed focus")
	}
}
