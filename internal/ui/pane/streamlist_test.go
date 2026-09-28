package pane

import (
	"fmt"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
	"github.com/holoplot/rtp-monitor/internal/stream"
)

func listStreams() []*stream.Stream {
	return []*stream.Stream{
		{ID: "a", Description: stream.StreamDescription{Name: "Alpha"}, SDP: []byte("v=0 alpha")},
		{ID: "b", Description: stream.StreamDescription{Name: "Bravo"}},
		{ID: "c", Description: stream.StreamDescription{Name: "Charlie"}},
	}
}

func TestStreamListKeepsSelection(t *testing.T) {
	l := NewStreamList(false)
	s := listStreams()
	l.SetStreams(s)
	l.Update(key("down"))
	if l.Selected().ID != "b" {
		t.Fatalf("selected %q, want b", l.Selected().ID)
	}
	l.SetStreams([]*stream.Stream{s[2], s[1]})
	if l.Selected().ID != "b" {
		t.Fatalf("after reorder selected %q, want b", l.Selected().ID)
	}
	l.SetStreams([]*stream.Stream{s[0], s[2]})
	if l.Selected().ID != "a" {
		t.Fatalf("after removal selected %q, want a", l.Selected().ID)
	}
}

func TestStreamListOpenKeys(t *testing.T) {
	l := NewStreamList(false)
	l.SetStreams(listStreams())
	l.Update(key("down"))

	for k, want := range map[string]Kind{"m": KindMeters, "R": KindRecord, "d": KindDetails, "s": KindSDP, "r": KindRTCP} {
		cmd := l.Update(key(k))
		if cmd == nil {
			t.Fatalf("%s: no command", k)
		}
		msg, ok := cmd().(OpenMsg)
		if !ok || msg.Kind != want || msg.Stream.ID != "b" {
			t.Errorf("%s: got %#v, want kind %v for stream b", k, msg, want)
		}
	}

	if l.Update(key("f")) != nil {
		t.Error("f opened an FPGA pane without the device")
	}
	withFPGA := NewStreamList(true)
	withFPGA.SetStreams(listStreams())
	if msg := withFPGA.Update(key("f"))().(OpenMsg); msg.Kind != KindFPGA {
		t.Errorf("f: kind %v, want FPGA", msg.Kind)
	}
}

func TestStreamListClickSelects(t *testing.T) {
	l := NewStreamList(false)
	l.SetStreams(listStreams())
	_ = l.View(80, 10)

	l.Update(ClickMsg{X: 5, Y: 3})
	if l.Selected().ID != "c" {
		t.Fatalf("selected %q, want c", l.Selected().ID)
	}
	l.Update(ClickMsg{X: 5, Y: 0}) // header
	l.Update(ClickMsg{X: 5, Y: 9}) // below the last row
	if l.Selected().ID != "c" {
		t.Fatalf("selection changed to %q", l.Selected().ID)
	}
}

func TestStreamListView(t *testing.T) {
	l := NewStreamList(false)
	l.SetStreams(listStreams())
	l.SetPaneCounts(map[string]int{"b": 2})

	r := rows(l.View(100, 5))
	if len(r) != 5 {
		t.Fatalf("%d rows, want 5", len(r))
	}
	view := strings.Join(r, "\n")
	for _, want := range []string{"Name", "Alpha", "Charlie", "●2"} {
		if !strings.Contains(view, want) {
			t.Errorf("view lacks %q:\n%s", want, view)
		}
	}
	for _, row := range r {
		if ansi.StringWidth(row) != 100 {
			t.Errorf("row width %d, want 100: %q", ansi.StringWidth(row), row)
		}
	}
	if l.Title() != "Streams (3)" || l.CopyText() != "v=0 alpha" {
		t.Errorf("title %q, copy %q", l.Title(), l.CopyText())
	}
}

func TestStreamListScrollsToSelection(t *testing.T) {
	var streams []*stream.Stream
	for i := range 10 {
		streams = append(streams, &stream.Stream{ID: fmt.Sprint(i), Description: stream.StreamDescription{Name: fmt.Sprintf("s%d", i)}})
	}
	l := NewStreamList(false)
	l.SetStreams(streams)
	_ = l.View(80, 4) // header + 3 rows
	l.Update(key("end"))
	view := strings.Join(rows(l.View(80, 4)), "\n")
	if !strings.Contains(view, "s9") || strings.Contains(view, "s6") {
		t.Fatalf("view does not end at the selection:\n%s", view)
	}
}

func TestStreamListWideNames(t *testing.T) {
	l := NewStreamList(false)
	l.SetStreams([]*stream.Stream{{ID: "w", Description: stream.StreamDescription{
		Name: "Überstudio 日本語 " + strings.Repeat("x", 200)}}})
	for _, row := range rows(l.View(60, 4)) {
		if ansi.StringWidth(row) != 60 {
			t.Errorf("row width %d, want 60: %q", ansi.StringWidth(row), row)
		}
	}
}

func TestStreamListEmpty(t *testing.T) {
	l := NewStreamList(true)
	if !strings.Contains(ansi.Strip(l.View(60, 5)), "No RTP streams detected") {
		t.Error("missing empty message")
	}
	l.Update(key("down"))
	if l.Update(key("m")) != nil || l.Selected() != nil || l.CopyText() != "" {
		t.Error("keys on an empty list must do nothing")
	}
}
