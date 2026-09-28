package pane

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/holoplot/rtp-monitor/internal/stream"
)

// SDP shows the raw session description of a stream.
type SDP struct {
	base
}

var _ Pane = (*SDP)(nil)

func NewSDP(s *stream.Stream) *SDP {
	return &SDP{base: newBase(KindSDP, s)}
}

func (p *SDP) SetStream(s *stream.Stream) { p.bind(s) }

func (p *SDP) Update(msg tea.Msg) tea.Cmd {
	p.handleScroll(msg)
	return nil
}

func (p *SDP) View(w, h int) string {
	var lines []string
	for line := range strings.SplitSeq(string(p.stream.SDP), "\n") {
		lines = append(lines, sanitize(line))
	}
	return p.renderLines(lines, w, h)
}

func (p *SDP) CopyText() string { return string(p.stream.SDP) }
func (p *SDP) Close()           {}
