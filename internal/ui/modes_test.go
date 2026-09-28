package ui

import (
	"strings"
	"testing"

	"github.com/holoplot/rtp-monitor/internal/layout"
)

func TestModeSwitching(t *testing.T) {
	h := newHarness(t, 120, 41)
	for _, step := range []struct {
		key  string
		want Mode
	}{
		{"ctrl+w", ModePane}, {"esc", ModeNormal},
		{"ctrl+w", ModePane}, {"ctrl+w", ModeNormal},
		{"ctrl+w", ModePane}, {"r", ModeResize}, {"m", ModeResize}, {"ctrl+w", ModeNormal},
		{"ctrl+w", ModePane}, {"m", ModeMove}, {"r", ModeMove}, {"enter", ModeNormal},
	} {
		h.press(step.key)
		if h.app.mode != step.want {
			t.Fatalf("after %s: mode %s, want %s", step.key, h.app.mode.Label(), step.want.Label())
		}
	}
	for _, k := range []string{"ctrl+p", "ctrl+n", "ctrl+h", "ctrl+q"} {
		h.press(k) // zellij's default mode keys must not do anything here
		if h.app.mode != ModeNormal {
			t.Fatalf("%s changed the mode to %s", k, h.app.mode.Label())
		}
	}
	h.press("ctrl+w")
	if !strings.Contains(h.screen(), "PANE") {
		t.Fatal("status bar does not show the mode")
	}
}

func TestPaneMode(t *testing.T) {
	h := newHarness(t, 120, 41)
	h.press("m") // 1 | 2

	h.press("ctrl+w", "l")
	if h.app.tree.Focus() != 2 {
		t.Fatalf("focus = %d, want 2", h.app.tree.Focus())
	}

	h.press("f")
	if !h.app.tree.Zoomed() || len(h.rects()) != 1 || !strings.Contains(h.screen(), "ZOOM") {
		t.Fatal("f did not zoom")
	}
	h.press("f", "h")
	if h.app.tree.Focus() != 1 {
		t.Fatalf("focus = %d, want 1", h.app.tree.Focus())
	}

	h.press("x")
	if len(h.rects()) != 2 || !strings.Contains(h.screen(), "cannot be closed") {
		t.Fatal("closing the stream list was not refused")
	}

	h.press("l", "x")
	if !h.panes[0].closed || len(h.rects()) != 1 || h.rects()[1] != (layout.Rect{W: 120, H: 40}) {
		t.Fatalf("close failed: closed=%v rects=%v", h.panes[0].closed, h.rects())
	}
	if len(h.app.panes) != 1 {
		t.Fatalf("%d panes registered, want 1", len(h.app.panes))
	}
}

func TestFlipInPaneMode(t *testing.T) {
	h := newHarness(t, 120, 41)
	h.press("m", "ctrl+w", "t")
	if got := h.rects()[1]; got != (layout.Rect{W: 120, H: 20}) {
		t.Fatalf("list rect after flip = %v", got)
	}
}

func TestResizeAndMoveModes(t *testing.T) {
	h := newHarness(t, 120, 41)
	h.press("m") // list {0,0,60,40} | meters {60,0,60,40}

	h.press("ctrl+w", "r", "l")
	if w := h.rects()[1].W; w != 62 {
		t.Fatalf("grow: width %d, want 62", w)
	}
	h.press("L")
	if w := h.rects()[1].W; w != 60 {
		t.Fatalf("shrink: width %d, want 60", w)
	}
	h.press("h")
	if !strings.Contains(h.screen(), "no split to resize") {
		t.Fatal("impossible resize not reported")
	}

	h.press("esc", "ctrl+w", "m", "l")
	if x := h.rects()[1].X; x != 60 {
		t.Fatalf("move: list at x=%d, want 60", x)
	}
	h.press("tab")
	if x := h.rects()[1].X; x != 0 {
		t.Fatalf("swap next: list at x=%d, want 0", x)
	}
}
