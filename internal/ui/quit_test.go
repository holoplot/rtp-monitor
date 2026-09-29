package ui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

func TestQuitConfirmationWithRecordings(t *testing.T) {
	h := newHarness(t, 120, 41)
	h.press("R")

	if msg := h.press("q"); msg != nil {
		t.Fatalf("quit without confirmation: %T", msg)
	}
	if !strings.Contains(h.screen(), "1 recording active, quit? y/n") {
		t.Fatalf("no confirmation prompt:\n%s", h.screen())
	}

	h.press("n")
	if h.app.confirmQuit || strings.Contains(h.screen(), "quit? y/n") {
		t.Fatal("n did not cancel")
	}

	h.press("ctrl+c")
	if _, ok := h.press("y").(tea.QuitMsg); !ok {
		t.Fatal("y did not quit")
	}
	if !h.panes[0].closed {
		t.Fatal("record pane not closed")
	}
}

func TestCloseIsIdempotent(t *testing.T) {
	h := newHarness(t, 120, 41)
	h.press("m")
	if _, ok := h.press("q").(tea.QuitMsg); !ok {
		t.Fatal("q did not quit")
	}
	h.app.Close()
	h.app.Close()
	if n := h.panes[0].closes; n != 1 {
		t.Fatalf("pane closed %d times, want 1", n)
	}

	h2 := newHarness(t, 120, 41)
	h2.press("R")
	h2.app.Close() // e.g. SIGTERM: Update never saw a quit key
	h2.app.Close()
	if n := h2.panes[0].closes; n != 1 {
		t.Fatalf("pane closed %d times without quit, want 1", n)
	}
}

func TestMessagesIgnoredAfterQuit(t *testing.T) {
	h := newHarness(t, 120, 41)
	h.press("m")
	if _, ok := h.press("q").(tea.QuitMsg); !ok {
		t.Fatal("q did not quit")
	}

	h.send(UpdateStreamsMsg{})
	if h.panes[0].offline {
		t.Fatal("stream update reached a pane after quit")
	}
	h.send(tea.MouseWheelMsg{X: 70, Y: 10, Button: tea.MouseWheelDown})
	if len(h.panes[0].msgs) != 0 {
		t.Fatalf("pane got %v after quit", h.panes[0].msgs)
	}
}
