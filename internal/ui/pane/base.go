package pane

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/holoplot/rtp-monitor/internal/stream"
)

// base implements stream binding, the offline state and scrolling for panes
// bound to one stream. Panes embed it.
type base struct {
	kind    Kind
	stream  *stream.Stream // latest snapshot; kept while the stream is offline
	offline bool

	scroll     int
	follow     bool // log panes: stay at the bottom while pinned there
	pinned     bool
	pageHeight int
	maxScroll  int
}

func newBase(kind Kind, s *stream.Stream) base {
	return base{kind: kind, stream: s, pinned: true}
}

func (b *base) Kind() Kind       { return b.kind }
func (b *base) StreamID() string { return b.stream.ID }
func (b *base) Offline() bool    { return b.offline }

func (b *base) Title() string {
	title := b.kind.String() + " · " + sanitize(b.stream.Name())
	if b.offline {
		title += " · offline"
	}
	return title
}

// bind stores a new snapshot and reports how its description differs from
// the previous one. nil marks the stream as offline and keeps the old snapshot.
func (b *base) bind(s *stream.Stream) stream.DescriptionChange {
	if s == nil {
		b.offline = true
		return stream.DescriptionUnchanged
	}
	change := stream.CompareDescriptions(b.stream.Description, s.Description)
	b.stream, b.offline = s, false
	return change
}

// handleScroll applies scroll keys and ScrollMsg. It reports whether msg was consumed.
func (b *base) handleScroll(msg tea.Msg) bool {
	switch msg := msg.(type) {
	case ScrollMsg:
		b.scrollTo(b.scroll + msg.Lines)
	case tea.KeyPressMsg:
		switch msg.String() {
		case "up", "k":
			b.scrollTo(b.scroll - 1)
		case "down", "j":
			b.scrollTo(b.scroll + 1)
		case "pgup":
			b.scrollTo(b.scroll - max(b.pageHeight, 1))
		case "pgdown":
			b.scrollTo(b.scroll + max(b.pageHeight, 1))
		case "home":
			b.scrollTo(0)
		case "end":
			b.scrollTo(b.maxScroll)
		default:
			return false
		}
	default:
		return false
	}
	return true
}

func (b *base) scrollTo(v int) {
	b.scroll = max(0, min(v, b.maxScroll))
	b.pinned = b.scroll >= b.maxScroll
}

// renderLines renders the visible part of lines into exactly w×h cells, with
// a scrollbar in the last column when they don't fit.
func (b *base) renderLines(lines []string, w, h int) string {
	if w <= 0 || h <= 0 {
		return ""
	}

	b.pageHeight = h
	b.maxScroll = max(0, len(lines)-h)
	if b.follow && b.pinned {
		b.scroll = b.maxScroll
	}
	b.scroll = min(b.scroll, b.maxScroll)

	bar := len(lines) > h && w > 2
	textW := w
	if bar {
		textW = w - 2
	}

	out := make([]string, h)
	for i := range out {
		line := ""
		if b.scroll+i < len(lines) {
			line = lines[b.scroll+i]
		}
		out[i] = fitLine(line, textW)
		if bar {
			out[i] += " " + scrollbar(i, h, len(lines), b.scroll)
		}
	}
	return strings.Join(out, "\n")
}
