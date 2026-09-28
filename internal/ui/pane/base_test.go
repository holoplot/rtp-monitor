package pane

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
	"github.com/holoplot/rtp-monitor/internal/stream"
)

func rows(s string) []string { return strings.Split(ansi.Strip(s), "\n") }

func TestFit(t *testing.T) {
	if got, want := Fit("abc\nlonger line\nx\ny", 5, 3), "abc  \nlong…\nx    "; got != want {
		t.Errorf("Fit = %q, want %q", got, want)
	}
	if got := Fit("日本語", 4, 1); ansi.StringWidth(got) != 4 {
		t.Errorf("wide runes: width %d, want 4 (%q)", ansi.StringWidth(got), got)
	}
	if Fit("x", 0, 3) != "" || Fit("x", 3, 0) != "" {
		t.Error("Fit with zero size must be empty")
	}
}

func TestSanitize(t *testing.T) {
	if got, want := sanitize("a\tb\r\x01c"), "a    b c"; got != want {
		t.Errorf("sanitize = %q, want %q", got, want)
	}
}

func TestRenderLinesScrolling(t *testing.T) {
	b := newBase(KindSDP, testStream())
	lines := numbered(10)

	r := rows(b.renderLines(lines, 10, 3))
	if len(r) != 3 || !strings.HasPrefix(r[0], "l1 ") || !strings.HasSuffix(r[0], "█") || !strings.HasSuffix(r[2], "│") {
		t.Fatalf("top: %q", r)
	}

	b.handleScroll(key("end"))
	r = rows(b.renderLines(lines, 10, 3))
	if !strings.HasPrefix(r[2], "l10") || !strings.HasSuffix(r[2], "█") {
		t.Fatalf("end: %q", r)
	}

	b.handleScroll(ScrollMsg{Lines: -100})
	r = rows(b.renderLines(lines, 10, 3))
	if !strings.HasPrefix(r[0], "l1 ") {
		t.Fatalf("after wheel up: %q", r)
	}
	for _, row := range r {
		if ansi.StringWidth(row) != 10 {
			t.Errorf("row %q has width %d, want 10", row, ansi.StringWidth(row))
		}
	}

	if b.handleScroll(key("x")) {
		t.Error("handleScroll consumed an unrelated key")
	}
}

func TestRenderLinesFollow(t *testing.T) {
	b := newBase(KindRTCP, testStream())
	b.follow = true

	_ = b.renderLines(numbered(5), 10, 3)
	if r := rows(b.renderLines(numbered(8), 10, 3)); !strings.HasPrefix(r[2], "l8") {
		t.Fatalf("not following: %q", r)
	}

	b.handleScroll(key("up"))
	if r := rows(b.renderLines(numbered(12), 10, 3)); !strings.HasPrefix(r[2], "l7") {
		t.Fatalf("scrolled up but still following: %q", r)
	}

	b.handleScroll(key("end"))
	if r := rows(b.renderLines(numbered(14), 10, 3)); !strings.HasPrefix(r[2], "l14") {
		t.Fatalf("end did not resume following: %q", r)
	}
}

func TestBindAndTitle(t *testing.T) {
	s := testStream()
	b := newBase(KindMeters, s)
	if b.Title() != "Meters · Alpha" || b.StreamID() != "alpha-id" || b.Kind() != KindMeters {
		t.Fatalf("title %q, id %q, kind %v", b.Title(), b.StreamID(), b.Kind())
	}

	if b.bind(nil) != stream.DescriptionUnchanged || !b.Gone() || b.Title() != "Meters · Alpha · gone" {
		t.Fatalf("after bind(nil): gone=%v title=%q", b.Gone(), b.Title())
	}

	s2 := testStream()
	s2.Description.ChannelCount = 8
	if got := b.bind(s2); got != stream.DescriptionFormatChanged || b.Gone() {
		t.Fatalf("bind(changed) = %v, gone = %v", got, b.Gone())
	}
}
