package ui

import (
	"testing"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
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
