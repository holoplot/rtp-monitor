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
