package ui

import (
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/holoplot/rtp-monitor/internal/layout"
	"github.com/holoplot/rtp-monitor/internal/theme"
)

func TestTitleBarWidth(t *testing.T) {
	s := lipgloss.NewStyle()
	for _, c := range []struct {
		title string
		w     int
	}{
		{"Meters · Alpha", 40},
		{"Meters · 日本語 a very long stream name indeed", 20},
		{"x", 6},
		{"x", 2},
	} {
		if got := ansi.StringWidth(titleBar(c.title, c.w, s, s)); got != c.w {
			t.Errorf("titleBar(%q, %d) has width %d", c.title, c.w, got)
		}
	}
}

func TestOfflinePaneGreyedOut(t *testing.T) {
	h := newHarness(t, 120, 41)
	h.press("m")
	var id layout.PaneID
	for pid := range h.rects() {
		if pid != h.app.listID {
			id = pid
		}
	}
	h.app.tree.SetFocus(h.app.listID)
	delete(h.app.flash, id)
	h.send(UpdateStreamsMsg{})

	muted := lipgloss.NewStyle().Foreground(theme.UI.Muted).Render("x")
	mutedSeq := muted[:strings.Index(muted, "x")]
	red := lipgloss.NewStyle().Foreground(theme.UI.Error).Render("x")
	redSeq := red[:strings.Index(red, "x")]

	got := h.app.renderPane(id, h.rects()[id])
	if strings.Contains(got, redSeq) {
		t.Error("offline pane drawn in red")
	}
	for i, line := range strings.Split(got, "\n") {
		if ansi.Strip(line) != "" && !strings.Contains(line, mutedSeq) {
			t.Errorf("line %d not greyed out: %q", i, line)
		}
	}
}
