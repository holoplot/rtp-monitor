package ui

import (
	"net"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/holoplot/rtp-monitor/internal/ptp"
	"github.com/holoplot/rtp-monitor/internal/stream"
	"github.com/holoplot/rtp-monitor/internal/theme"
	"github.com/pion/rtp/v2"
)

// DetailsModalContent implements ModalContentProvider for stream details
type DetailsModalContent struct {
	mutex sync.Mutex

	stream     *stream.Stream
	receiver   *stream.RTPReceiver
	ptpMonitor *ptp.Monitor

	lastUpdate       time.Time
	sourceStatistics []*sourceStatistics

	err          error
	contentWidth int
	headerStyle  lipgloss.Style
}

type sourceStatistics struct {
	packetCount      uint64
	lastPacketCount  uint64
	lastSequence     uint16
	sequenceErrors   uint64
	packetRate       float64
	lastRTPTimestamp uint32
	lastPacketTime   time.Time
	senders          map[string]struct{}
}

// NewDetailsModalContent creates a new details modal content provider
func NewDetailsModalContent(stream *stream.Stream, ptpMonitor *ptp.Monitor) *DetailsModalContent {
	d := &DetailsModalContent{
		stream:           stream,
		ptpMonitor:       ptpMonitor,
		sourceStatistics: make([]*sourceStatistics, len(stream.Description.Sources)),
		headerStyle: lipgloss.NewStyle().
			Foreground(theme.Colors.Primary).
			Bold(true),
	}

	for i := range len(d.sourceStatistics) {
		d.sourceStatistics[i] = &sourceStatistics{
			senders: make(map[string]struct{}),
		}
	}

	return d
}

func (d *DetailsModalContent) rtpReceiverCallback(sourceIndex int, src net.Addr, packet *rtp.Packet) {
	now := time.Now()

	d.mutex.Lock()
	defer d.mutex.Unlock()

	stat := d.sourceStatistics[sourceIndex]

	stat.packetCount++
	stat.lastRTPTimestamp = packet.Timestamp
	stat.lastPacketTime = now

	stat.senders[src.String()] = struct{}{}

	if stat.packetCount > 1 {
		if packet.SequenceNumber != stat.lastSequence+1 {
			stat.sequenceErrors++
		}
	}

	stat.lastSequence = packet.SequenceNumber
}

// Init initializes the content provider with dimensions
func (d *DetailsModalContent) Init(width, height int) {
	d.lastUpdate = time.Now()

	if receiver, err := d.stream.NewRTPReceiver(d.rtpReceiverCallback); err == nil {
		d.receiver = receiver
	} else {
		d.err = err
	}

	d.contentWidth = width
}

func (d *DetailsModalContent) Close() {
	if d.receiver != nil {
		d.receiver.Close()
	}
}

// Content returns the content lines to be displayed
func (d *DetailsModalContent) Content() []string {
	s := d.stream

	l := newLineBuffer(d.headerStyle)

	l.p("Basic Information")
	l.p("  ├─ ID:               %s", s.ID)
	l.p("  ├─ ID hash:          %s", s.IDHash())
	l.p("  ├─ Name:             %s", s.Description.Name)
	l.p("  ├─ Discovery Method: %s", s.DiscoveryMethod)
	l.p("  └─ Discovery Source: %s", s.DiscoverySource)
	l.p("")

	l.p("Stream Information")
	l.p("  ├─ Content Type:   %s", s.Description.ContentType)
	l.p("  ├─ Sample Rate:    %d Hz", s.Description.SampleRate)
	l.p("  ├─ Channels:       %d", s.Description.ChannelCount)
	l.p("  └─ Codec Info:     %s", s.CodecInfo())
	l.p("")

	for i, source := range s.Description.Sources {
		l.p("Source %d information:", i+1)
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

	d.mutex.Lock()
	defer d.mutex.Unlock()

	if d.err != nil {
		l.p("Error creating stream receiver: %v", d.err)

		return l.lines()
	}

	dur := time.Since(d.lastUpdate)

	if dur > time.Second {
		for _, stats := range d.sourceStatistics {
			stats.packetRate = float64(stats.packetCount-stats.lastPacketCount) / dur.Seconds()
			stats.lastPacketCount = stats.packetCount
		}

		d.lastUpdate = time.Now()
	}

	for i, source := range s.Description.Sources {
		stats := d.sourceStatistics[i]

		l.p("Source %d statistics (%s:%d):", i+1,
			source.DestinationAddress.String(),
			source.DestinationPort)

		var senders []string

		for sender := range stats.senders {
			senders = append(senders, sender)
		}

		slices.Sort(senders)

		l.p("  ├─ Senders:         %s", strings.Join(senders, ", "))
		l.p("  ├─ Packets count:   %d", stats.packetCount)
		l.p("  ├─ Packets rate:    %.2f/s", stats.packetRate)
		l.p("  ├─ Parsing errors:  %d", d.receiver.RTPErrors(i))
		l.p("  ├─ Sequence errors: %d", stats.sequenceErrors)
		l.p("  └─ Last timestamp:  %d", stats.lastRTPTimestamp)
		l.p("")
	}

	if d.ptpMonitor != nil {
		d.ptpMonitor.ForEachTransmitter(func(ci ptp.ClockIdentity, t *ptp.Transmitter) {
			ptpSamples := t.LastTimestamp.InSamples(d.stream.Description.SampleRate)

			l.p("PTP Transmitter %s (domain %d):", ci, t.Domain)
			l.p("  ├─ PTP timestamp (UTC): %s", t.LastTimestamp.AsUTC())
			l.p("  ├─ PTP timestamp (TAI): %s", t.LastTimestamp.AsTAI())
			l.p("  └─ RTP samples:         %d", ptpSamples)
			l.p("")
		})
	} else {
		l.p("[PTP Transmitter information unavailable]")
	}

	return l.lines()
}

// Title returns the modal title
func (d *DetailsModalContent) Title() string {
	return "STREAM DETAILS"
}

// UpdateInterval returns how often the modal content should be updated
func (d *DetailsModalContent) UpdateInterval() time.Duration {
	return 50 * time.Millisecond
}

// AutoScroll returns whether the modal should automatically scroll to the bottom
func (d *DetailsModalContent) AutoScroll() bool {
	return false
}

// Update is called periodically to refresh content
func (d *DetailsModalContent) Update() {
}
