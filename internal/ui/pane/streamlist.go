package pane

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/holoplot/rtp-monitor/internal/stream"
	"github.com/holoplot/rtp-monitor/internal/theme"
)

// StreamList is the table of discovered streams. There is exactly one, and it
// cannot be closed.
type StreamList struct {
	streams    []*stream.Stream
	selected   int
	viewStart  int
	rows       int // data rows visible in the last render
	paneCounts map[string]int
	fpga       bool
}

var _ Pane = (*StreamList)(nil)

var openKeys = map[string]Kind{
	"m": KindMeters, "R": KindRecord, "d": KindDetails,
	"s": KindSDP, "r": KindRTCP, "f": KindFPGA,
}

// NewStreamList returns an empty stream list. FPGA RX panes can only be
// opened when fpgaAvailable is set.
func NewStreamList(fpgaAvailable bool) *StreamList {
	return &StreamList{fpga: fpgaAvailable, rows: 1}
}

func (l *StreamList) Kind() Kind               { return KindStreamList }
func (l *StreamList) Title() string            { return fmt.Sprintf("Streams (%d)", len(l.streams)) }
func (l *StreamList) StreamID() string         { return "" }
func (l *StreamList) Gone() bool               { return false }
func (l *StreamList) SetStream(*stream.Stream) {}
func (l *StreamList) Close()                   {}

// SetStreams replaces the list. The selected stream stays selected if it is
// still present; otherwise the first stream is selected.
func (l *StreamList) SetStreams(streams []*stream.Stream) {
	var id string
	if s := l.Selected(); s != nil {
		id = s.ID
	}

	l.streams = streams
	l.selected = 0
	for i, s := range streams {
		if s.ID == id {
			l.selected = i
			break
		}
	}
	l.adjustView()
}

// SetPaneCounts sets how many panes are open per stream ID, shown as a marker.
func (l *StreamList) SetPaneCounts(counts map[string]int) { l.paneCounts = counts }

// Selected returns the highlighted stream, or nil if the list is empty.
func (l *StreamList) Selected() *stream.Stream {
	if l.selected >= 0 && l.selected < len(l.streams) {
		return l.streams[l.selected]
	}
	return nil
}

// CopyText returns the SDP of the highlighted stream.
func (l *StreamList) CopyText() string {
	if s := l.Selected(); s != nil {
		return string(s.SDP)
	}
	return ""
}

func (l *StreamList) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		k := msg.String()
		if kind, ok := openKeys[k]; ok {
			s := l.Selected()
			if s == nil || (kind == KindFPGA && !l.fpga) {
				return nil
			}
			return func() tea.Msg { return OpenMsg{Kind: kind, Stream: s} }
		}
		switch k {
		case "up", "k":
			l.move(-1)
		case "down", "j":
			l.move(1)
		case "pgup":
			l.move(-l.rows)
		case "pgdown":
			l.move(l.rows)
		case "home":
			l.move(-len(l.streams))
		case "end":
			l.move(len(l.streams))
		}
	case ScrollMsg:
		l.move(msg.Lines)
	case ClickMsg:
		if i := l.viewStart + msg.Y - 1; msg.Y >= 1 && i < len(l.streams) {
			l.selected = i
		}
	}
	return nil
}

func (l *StreamList) move(delta int) {
	if len(l.streams) == 0 {
		return
	}
	l.selected = max(0, min(l.selected+delta, len(l.streams)-1))
	l.adjustView()
}

func (l *StreamList) adjustView() {
	if l.selected < l.viewStart {
		l.viewStart = l.selected
	}
	if l.selected >= l.viewStart+l.rows {
		l.viewStart = l.selected - l.rows + 1
	}
	l.viewStart = max(0, min(l.viewStart, len(l.streams)-l.rows))
}

func (l *StreamList) View(w, h int) string {
	if w <= 0 || h <= 0 {
		return ""
	}
	if len(l.streams) == 0 {
		return Fit(lipgloss.Place(w, h, lipgloss.Center, lipgloss.Center, "No RTP streams detected"), w, h)
	}

	l.rows = max(h-1, 1)
	l.adjustView()

	cols := columnWidths(w - 2) // last two columns: scrollbar
	header := lipgloss.NewStyle().Bold(true).
		Foreground(theme.UI.TableHeaderFg).Background(theme.UI.TableHeaderBg)
	selected := lipgloss.NewStyle().Bold(true).
		Foreground(theme.UI.RowSelectedFg).Background(theme.UI.RowSelectedBg)

	lines := []string{header.Render(fitLine(formatRow(cols, "", "ID", "Name", "Address", "Codec", "Discovery"), w))}

	end := min(l.viewStart+l.rows, len(l.streams))
	for i := l.viewStart; i < end; i++ {
		s := l.streams[i]
		marker := ""
		if n := l.paneCounts[s.ID]; n > 0 {
			marker = fmt.Sprintf("●%d", n)
		}

		row := fitLine(formatRow(cols, marker, s.IDHash(), s.Name(), s.Address(), s.CodecInfo(), s.DiscoveryLabel()), w-2)
		if i == l.selected {
			row = selected.Render(row)
		}
		if len(l.streams) > l.rows {
			row += " " + scrollbar(i-l.viewStart, l.rows, len(l.streams), l.viewStart)
		}
		lines = append(lines, row)
	}

	return Fit(strings.Join(lines, "\n"), w, h)
}

// columnWidths splits avail cells into the marker, ID, name, address, codec
// and discovery columns. Each width includes one separating space.
func columnWidths(avail int) [6]int {
	const marker, id, codec = 4, 11, 19
	rest := max(avail-marker-id-codec, 0)
	name := rest * 30 / 100
	address := rest * 40 / 100
	return [6]int{marker, id, name, address, codec, rest - name - address}
}

func formatRow(cols [6]int, cells ...string) string {
	var b strings.Builder
	for i, c := range cells {
		if cols[i] > 0 {
			b.WriteString(fitLine(sanitize(c), cols[i]-1)) // names come from the network
			b.WriteString(" ")
		}
	}
	return b.String()
}
