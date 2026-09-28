package pane

import (
	"maps"
	"net"
	"slices"
	"strings"
	"sync"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/holoplot/rtp-monitor/internal/ptp"
	"github.com/holoplot/rtp-monitor/internal/stream"
	"github.com/pion/rtp/v2"
)

// Details shows stream metadata, per-source packet statistics and PTP transmitters.
type Details struct {
	base
	ptp      *ptp.Monitor
	stats    *rxStats
	receiver *stream.RTPReceiver
	err      error
	lastRate time.Time
}

var _ Pane = (*Details)(nil)

// rxStats holds per-source packet statistics for one receiver. Each receiver
// gets its own rxStats, so callbacks of a replaced receiver can't corrupt the
// statistics of the new one.
type rxStats struct {
	mu      sync.Mutex
	sources []*sourceStats
}

type sourceStats struct {
	packets, ratePackets uint64
	rate                 float64
	lastTimestamp        uint32
	senders              map[string]struct{}
}

func newRxStats(n int) *rxStats {
	r := &rxStats{sources: make([]*sourceStats, n)}
	for i := range r.sources {
		r.sources[i] = &sourceStats{senders: map[string]struct{}{}}
	}
	return r
}

func (r *rxStats) onPacket(i int, src net.Addr, p *rtp.Packet) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if i >= len(r.sources) {
		return
	}
	s := r.sources[i]
	s.packets++
	s.lastTimestamp = p.Timestamp
	s.senders[src.String()] = struct{}{}
}

// updateRates recomputes packet rates over elapsed, the time since the last call.
func (r *rxStats) updateRates(elapsed time.Duration) {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, s := range r.sources {
		s.rate = float64(s.packets-s.ratePackets) / elapsed.Seconds()
		s.ratePackets = s.packets
	}
}

func NewDetails(s *stream.Stream, ptpMonitor *ptp.Monitor) *Details {
	d := &Details{base: newBase(KindDetails, s), ptp: ptpMonitor}
	d.start()
	return d
}

func (d *Details) start() {
	d.stop()
	stats := newRxStats(len(d.stream.Description.Sources))
	d.stats, d.lastRate = stats, time.Now()
	d.receiver, d.err = d.stream.NewRTPReceiver(func(i int, src net.Addr, p *rtp.Packet) {
		stats.onPacket(i, src, p)
	})
}

func (d *Details) stop() {
	if d.receiver != nil {
		d.receiver.Close()
		d.receiver = nil
	}
}

func (d *Details) SetStream(s *stream.Stream) {
	if d.bind(s) != stream.DescriptionUnchanged {
		d.start()
	}
}

func (d *Details) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case TickMsg:
		now := time.Time(msg)
		if elapsed := now.Sub(d.lastRate); elapsed >= time.Second {
			d.stats.updateRates(elapsed)
			d.lastRate = now
		}
		return nil
	case tea.KeyPressMsg:
		if msg.String() == "enter" && d.err != nil {
			d.start()
			return nil
		}
	}
	d.handleScroll(msg)
	return nil
}

func (d *Details) View(w, h int) string { return d.renderLines(d.lines(), w, h) }
func (d *Details) CopyText() string     { return strings.Join(d.lines(), "\n") }
func (d *Details) Close()               { d.stop() }

func (d *Details) lines() []string {
	s := d.stream
	var l lineBuffer

	l.p("Basic Information")
	l.p("  ├─ ID:               %s", s.ID)
	l.p("  ├─ ID hash:          %s", s.IDHash())
	l.p("  └─ Name:             %s", s.Name())
	l.p("")

	l.p("Discovered via (%d)", len(s.Discoveries))
	for i, disc := range s.Discoveries {
		branch := "├─"
		if i == len(s.Discoveries)-1 {
			branch = "└─"
		}
		source := disc.Source
		if source == "" {
			source = "-"
		}
		// Only SAP announcements repeat, so "last seen" is only meaningful there.
		if disc.Method == stream.DiscoveryMethodSAP {
			l.p("  %s %s @ %s   (last seen %s ago)", branch, disc.Method, source, time.Since(disc.LastSeen).Truncate(time.Second))
		} else {
			l.p("  %s %s @ %s", branch, disc.Method, source)
		}
	}
	l.p("")

	l.p("Stream Information")
	l.p("  ├─ Content Type:   %s", s.Description.ContentType)
	l.p("  ├─ Sample Rate:    %d Hz", s.Description.SampleRate)
	l.p("  ├─ Channels:       %d", s.Description.ChannelCount)
	l.p("  └─ Codec Info:     %s", s.CodecInfo())
	l.p("")

	if d.err != nil {
		l.p("Error creating stream receiver: %v", d.err)
		l.p("Press enter to retry")
		l.p("")
	}

	d.stats.mu.Lock()
	for i, source := range s.Description.Sources {
		if i >= len(d.stats.sources) {
			break
		}
		st := d.stats.sources[i]
		senders := slices.Sorted(maps.Keys(st.senders))

		l.p("Source %d statistics (%s:%d):", i+1, source.DestinationAddress, source.DestinationPort)
		l.p("  ├─ Senders:         %s", strings.Join(senders, ", "))
		l.p("  ├─ Packets count:   %d", st.packets)
		l.p("  ├─ Packets rate:    %.2f/s", st.rate)
		if d.receiver != nil {
			l.p("  ├─ Parsing errors:  %d", d.receiver.RTPErrors(i))
			l.p("  ├─ Sequence errors: %d", d.receiver.SequenceErrors(i))
		}
		l.p("  └─ Last timestamp:  %d", st.lastTimestamp)
		l.p("")
	}
	d.stats.mu.Unlock()

	if d.ptp != nil {
		d.ptp.ForEachTransmitter(func(ci ptp.ClockIdentity, t *ptp.Transmitter) {
			l.p("PTP Transmitter %s, domain %d, interface %s:", ci, t.Domain, t.IfiName)
			l.p("  ├─ PTP timestamp (UTC): %s", t.LastTimestamp.AsUTC())
			l.p("  ├─ PTP timestamp (TAI): %s", t.LastTimestamp.AsTAI())
			l.p("  └─ RTP samples:         %d", t.LastTimestamp.InSamples(s.Description.SampleRate))
			l.p("")
		})
	} else {
		l.p("PTP unavailable (run as root to monitor PTP transmitters)")
		l.p("")
	}

	for i, source := range s.Description.Sources {
		l.p("Source %d information (from SDP):", i+1)
		l.p("  ├─ Sender address:         %s", source.SenderAddress)
		l.p("  ├─ Destination address:    %s:%d", source.DestinationAddress, source.DestinationPort)
		l.p("  ├─ TTL:                    %d", source.TTL)
		l.p("  ├─ Frames per packet:      %d", source.FramesPerPacket)
		l.p("  ├─ Clock domain:           %s", source.ClockDomain)
		l.p("  ├─ Reference clock:        %s", source.ReferenceClock)
		l.p("  ├─ Media clock:            %s", source.MediaClock)
		l.p("  └─ Sync time:              %d", source.SyncTime)
		l.p("")
	}

	// Names and SDP attributes come from the network.
	for i := range l {
		l[i] = sanitize(l[i])
	}
	return l
}
