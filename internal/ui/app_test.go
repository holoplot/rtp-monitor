package ui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/holoplot/rtp-monitor/internal/layout"
	"github.com/holoplot/rtp-monitor/internal/stream"
	"github.com/holoplot/rtp-monitor/internal/ui/pane"
)

type fakePane struct {
	kind    pane.Kind
	s       *stream.Stream
	offline bool
	closed  bool
	closes  int
	msgs    []tea.Msg
}

func (f *fakePane) Kind() pane.Kind      { return f.kind }
func (f *fakePane) Title() string        { return f.kind.String() + " · " + f.s.Name() }
func (f *fakePane) StreamID() string     { return f.s.ID }
func (f *fakePane) Offline() bool        { return f.offline }
func (f *fakePane) View(_, _ int) string { return "fake" }
func (f *fakePane) CopyText() string     { return "copy of " + f.s.Name() }
func (f *fakePane) Close()               { f.closed = true; f.closes++ }

func (f *fakePane) SetStream(s *stream.Stream) {
	if s == nil {
		f.offline = true
		return
	}
	f.s, f.offline = s, false
}

func (f *fakePane) Update(msg tea.Msg) tea.Cmd {
	if _, tick := msg.(pane.TickMsg); !tick {
		f.msgs = append(f.msgs, msg)
	}
	return nil
}

func testStreams() []*stream.Stream {
	return []*stream.Stream{
		{ID: "alpha", Description: stream.StreamDescription{Name: "Alpha"}, SDP: []byte("v=0 alpha")},
		{ID: "beta", Description: stream.StreamDescription{Name: "Beta"}},
	}
}

func keyMsg(s string) tea.KeyPressMsg {
	special := map[string]tea.KeyPressMsg{
		"esc": {Code: tea.KeyEscape}, "enter": {Code: tea.KeyEnter}, "tab": {Code: tea.KeyTab},
		"shift+tab": {Code: tea.KeyTab, Mod: tea.ModShift}, "backspace": {Code: tea.KeyBackspace},
		"up": {Code: tea.KeyUp}, "down": {Code: tea.KeyDown}, "left": {Code: tea.KeyLeft}, "right": {Code: tea.KeyRight},
	}
	if k, ok := special[s]; ok {
		return k
	}
	if rest, ok := strings.CutPrefix(s, "ctrl+"); ok {
		return tea.KeyPressMsg{Code: []rune(rest)[0], Mod: tea.ModCtrl}
	}
	return tea.KeyPressMsg{Code: []rune(s)[0], Text: s}
}

type harness struct {
	t      *testing.T
	app    *App
	panes  []*fakePane
	copied string
}

func newHarness(t *testing.T, w, h int) *harness {
	hs := &harness{t: t}
	hs.app = NewApp(Options{
		Factory: func(k pane.Kind, s *stream.Stream, _ pane.Context) pane.Pane {
			p := &fakePane{kind: k, s: s}
			hs.panes = append(hs.panes, p)
			return p
		},
		Clipboard: func(s string) error {
			hs.copied = s
			return nil
		},
	})
	hs.send(tea.WindowSizeMsg{Width: w, Height: h})
	hs.send(UpdateStreamsMsg{Streams: testStreams()})
	return hs
}

// send delivers msg, feeds OpenMsg results back into the app and returns the
// first other message produced by a command (e.g. tea.QuitMsg), or nil.
func (h *harness) send(msg tea.Msg) tea.Msg {
	_, cmd := h.app.Update(msg)
	for cmd != nil {
		out := cmd()
		if _, ok := out.(pane.OpenMsg); !ok {
			return out
		}
		_, cmd = h.app.Update(out)
	}
	return nil
}

func (h *harness) press(keys ...string) tea.Msg {
	var last tea.Msg
	for _, k := range keys {
		last = h.send(keyMsg(k))
	}
	return last
}

func (h *harness) screen() string                       { return ansi.Strip(h.app.render()) }
func (h *harness) rects() map[layout.PaneID]layout.Rect { return h.app.tree.Rects() }

func TestOpenPaneFromStreamList(t *testing.T) {
	h := newHarness(t, 120, 41)
	h.press("m")

	if len(h.panes) != 1 || h.panes[0].kind != pane.KindMeters || h.panes[0].s.ID != "alpha" {
		t.Fatalf("panes = %+v", h.panes)
	}
	want := map[layout.PaneID]layout.Rect{
		1: {X: 0, Y: 0, W: 60, H: 40},
		2: {X: 60, Y: 0, W: 60, H: 40},
	}
	if got := h.rects(); len(got) != 2 || got[1] != want[1] || got[2] != want[2] {
		t.Fatalf("rects = %v", got)
	}
	if h.app.tree.Focus() != 1 {
		t.Fatalf("focus = %d, want the stream list", h.app.tree.Focus())
	}
	screen := h.screen()
	for _, s := range []string{"Streams (2)", "Meters · Alpha", "NORMAL"} {
		if !strings.Contains(screen, s) {
			t.Errorf("screen lacks %q", s)
		}
	}

	h.press("down", "R")
	if len(h.panes) != 2 || h.panes[1].kind != pane.KindRecord || h.panes[1].s.ID != "beta" {
		t.Fatalf("second pane = %+v", h.panes[1])
	}
}

func TestPanesFollowStreamPresence(t *testing.T) {
	h := newHarness(t, 120, 41)
	h.press("m")

	h.send(UpdateStreamsMsg{Streams: testStreams()[1:]})
	if !h.panes[0].offline {
		t.Fatal("pane not offline")
	}
	h.send(UpdateStreamsMsg{Streams: testStreams()})
	if h.panes[0].offline {
		t.Fatal("pane did not reattach")
	}
}

func TestNormalModeRouting(t *testing.T) {
	h := newHarness(t, 120, 41)
	h.press("m", "tab")
	if h.app.tree.Focus() != 2 {
		t.Fatalf("focus = %d, want 2", h.app.tree.Focus())
	}

	if msg := h.press("j", "q"); msg != nil {
		t.Fatalf("q in a stream pane produced %T", msg)
	}
	var keys []string
	for _, m := range h.panes[0].msgs {
		keys = append(keys, m.(tea.KeyPressMsg).String())
	}
	if strings.Join(keys, ",") != "j,q" {
		t.Fatalf("forwarded keys %v, want j,q", keys)
	}

	h.press("c")
	if h.copied != "copy of Alpha" {
		t.Fatalf("copied %q", h.copied)
	}
	h.press("shift+tab", "c")
	if h.app.tree.Focus() != 1 || h.copied != "v=0 alpha" {
		t.Fatalf("focus %d, copied %q", h.app.tree.Focus(), h.copied)
	}
}

func TestQuitClosesPanes(t *testing.T) {
	h := newHarness(t, 120, 41)
	h.press("m")
	if _, ok := h.press("q").(tea.QuitMsg); !ok {
		t.Fatal("q in the stream list did not quit")
	}
	if !h.panes[0].closed {
		t.Fatal("pane not closed on quit")
	}

	h2 := newHarness(t, 120, 41)
	if _, ok := h2.press("ctrl+c").(tea.QuitMsg); !ok {
		t.Fatal("ctrl+c did not quit")
	}
}

func TestNoRoomStatus(t *testing.T) {
	h := newHarness(t, 30, 9)
	h.press("m")
	if len(h.panes) != 0 {
		t.Fatal("pane created without room")
	}
	if !strings.Contains(h.screen(), "no room") {
		t.Fatalf("status missing:\n%s", h.screen())
	}
}

func TestTinyTerminal(t *testing.T) {
	h := newHarness(t, 120, 41)
	h.press("m", "m")

	h.send(tea.WindowSizeMsg{Width: 22, Height: 7})
	for _, line := range strings.Split(h.screen(), "\n") {
		if ansi.StringWidth(line) > 22 {
			t.Errorf("line wider than the terminal: %q", line)
		}
	}

	h.send(tea.WindowSizeMsg{Width: 10, Height: 4})
	if !strings.Contains(h.screen(), "terminal") {
		t.Fatalf("no too-small message:\n%s", h.screen())
	}
}

func TestAllStreamsVanish(t *testing.T) {
	h := newHarness(t, 120, 41)
	h.press("m")

	h.send(UpdateStreamsMsg{})
	if !h.panes[0].offline {
		t.Fatal("pane not offline")
	}
	h.press("m")
	if len(h.panes) != 1 {
		t.Fatal("opened a pane without streams")
	}
	if !strings.Contains(h.screen(), "No RTP streams detected") {
		t.Fatalf("empty message missing:\n%s", h.screen())
	}
}

func TestInitialSnapshotAfterLiveUpdateIgnored(t *testing.T) {
	h := newHarness(t, 120, 41) // already received a live update
	h.press("m")

	h.send(UpdateStreamsMsg{Streams: testStreams()[1:], Initial: true})
	if h.panes[0].offline || !strings.Contains(h.screen(), "Streams (2)") {
		t.Fatalf("stale initial snapshot applied:\n%s", h.screen())
	}

	fresh := NewApp(Options{})
	fresh.Update(UpdateStreamsMsg{Streams: testStreams(), Initial: true})
	if fresh.list.Selected() == nil {
		t.Fatal("initial snapshot ignored before any live update")
	}
}
