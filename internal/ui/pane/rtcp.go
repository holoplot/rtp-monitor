package pane

import (
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/holoplot/rtp-monitor/internal/stream"
	"github.com/pion/rtcp"
)

const maxLogLines = 5000

// RTCP shows a log of the RTCP packets of a stream. It follows new entries
// until the user scrolls up.
type RTCP struct {
	base
	mu       sync.Mutex
	log      []string
	receiver *stream.RTCPReceiver
	err      error
}

var _ Pane = (*RTCP)(nil)

func NewRTCP(s *stream.Stream) *RTCP {
	p := &RTCP{base: newBase(KindRTCP, s)}
	p.follow = true
	p.start()
	return p
}

func (p *RTCP) start() {
	p.stop()
	p.receiver, p.err = p.stream.NewRTCPReceiver(func(_ int, src net.Addr, pkt rtcp.Packet) {
		p.append(time.Now(), src, pkt)
	})
}

func (p *RTCP) stop() {
	if p.receiver != nil {
		p.receiver.Close()
		p.receiver = nil
	}
}

func (p *RTCP) append(now time.Time, src net.Addr, pkt rtcp.Packet) {
	lines := formatRTCP(pkt)
	if len(lines) == 0 {
		return
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	for _, line := range lines {
		p.log = append(p.log, fmt.Sprintf("%s | %s | %s", now.Format(time.RFC3339), src, line))
	}
	if n := len(p.log) - maxLogLines; n > 0 {
		p.log = append(p.log[:0:0], p.log[n:]...)
	}
}

func formatRTCP(pkt rtcp.Packet) []string {
	switch p := pkt.(type) {
	case *rtcp.SenderReport:
		return []string{fmt.Sprintf("SenderReport from %x, NTPTime %d.%d, RTPTime %d, PacketCount %d, OctetCount %d",
			p.SSRC, p.NTPTime>>32, p.NTPTime&0xFFFFFFFF, p.RTPTime, p.PacketCount, p.OctetCount)}
	case *rtcp.ReceiverReport:
		if p.SSRC == 0 {
			return nil
		}
		lines := []string{fmt.Sprintf("ReceiverReport from %x", p.SSRC)}
		for _, r := range p.Reports {
			lines = append(lines, fmt.Sprintf("  SSRC=%x, fractionLost=%d/%d, lastSequenceNumber=%d",
				r.SSRC, r.FractionLost, r.TotalLost, r.LastSequenceNumber))
		}
		return lines
	case *rtcp.SourceDescription:
		var chunks []string
		for _, c := range p.Chunks {
			chunks = append(chunks, fmt.Sprintf("Source %x: %s", c.Source, c.Items))
		}
		return []string{"SourceDescription: " + strings.Join(chunks, ", ")}
	default:
		return []string{fmt.Sprintf("Unsupported packet type %T", p)}
	}
}

func (p *RTCP) SetStream(s *stream.Stream) {
	if p.bind(s) != stream.DescriptionUnchanged {
		p.start()
	}
}

func (p *RTCP) Update(msg tea.Msg) tea.Cmd {
	if k, ok := msg.(tea.KeyPressMsg); ok && k.String() == "enter" && p.err != nil {
		p.start()
		return nil
	}
	p.handleScroll(msg)
	return nil
}

func (p *RTCP) lines() []string {
	p.mu.Lock()
	defer p.mu.Unlock()

	var lines []string
	if p.err != nil {
		lines = append(lines, fmt.Sprintf("Error creating stream receiver: %v", p.err), "Press enter to retry", "")
	}
	return append(lines, p.log...)
}

func (p *RTCP) View(w, h int) string { return p.renderLines(p.lines(), w, h) }
func (p *RTCP) CopyText() string     { return strings.Join(p.lines(), "\n") }
func (p *RTCP) Close()               { p.stop() }
