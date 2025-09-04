package ui

import (
	"fmt"
	"log/slog"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/holoplot/rtp-monitor/internal/stream"
	"github.com/pion/rtcp"
)

// DetailsModalContent implements ModalContentProvider for stream details
type RTCPModalContent struct {
	mutex sync.Mutex

	stream   *stream.Stream
	receiver *stream.RTCPReceiver

	lastUpdate time.Time
	log        []string

	headerStyle lipgloss.Style
	height      int
}

func NewRTCPModalContent(stream *stream.Stream) *RTCPModalContent {
	d := &RTCPModalContent{
		stream: stream,
		log:    make([]string, 0),
	}

	return d
}

func (d *RTCPModalContent) rtpReceiverCallback(sourceIndex int, src net.Addr, pkt rtcp.Packet) {
	now := time.Now()

	var lines []string

	switch p := pkt.(type) {
	case *rtcp.SenderReport:
		s := fmt.Sprintf("SenderReport from %x, NTPTime %d, RTPTime %d, PacketCount %d, OctetCount %d",
			p.SSRC, p.NTPTime, p.RTPTime, p.PacketCount, p.OctetCount)
		lines = append(lines, s)
	case *rtcp.ReceiverReport:
		if p.SSRC != 0 {
			s := fmt.Sprintf("ReceiverReport from %x", p.SSRC)
			lines = append(lines, s)

			for _, i := range p.Reports {
				s := fmt.Sprintf("  SSRC=%x, fractionLost=%d/%d, lastSequenceNumber=%d",
					i.SSRC, i.FractionLost, i.TotalLost, i.LastSequenceNumber)
				lines = append(lines, s)
			}
		}
	case *rtcp.SourceDescription:
		var chunks []string

		for _, i := range p.Chunks {
			s := fmt.Sprintf("Source %x: %s", i.Source, i.Items)
			chunks = append(chunks, s)
		}

		s := fmt.Sprintf("SourceDescription: %s", strings.Join(chunks, ", "))
		lines = append(lines, s)

	default:
		s := fmt.Sprintf("Unsupported packet type %T", p)
		lines = append(lines, s)
	}

	if len(lines) == 0 {
		return
	}

	d.mutex.Lock()
	defer d.mutex.Unlock()

	for _, line := range lines {
		d.log = append(d.log, fmt.Sprintf("%s | %s | %s", now.Format(time.RFC3339), src, line))
	}

	d.lastUpdate = now
}

func (d *RTCPModalContent) Init(width, height int) {
	d.lastUpdate = time.Now()

	if receiver, err := d.stream.NewRTCPReceiver(d.rtpReceiverCallback); err == nil {
		d.receiver = receiver
	} else {
		slog.Error("Failed to create receiver", "error", err)
	}

	d.height = height
}

func (d *RTCPModalContent) Close() {
	if d.receiver != nil {
		d.receiver.Close()
	}
}

// Content returns the content lines to be displayed
func (d *RTCPModalContent) Content() []string {
	var lines []string

	d.mutex.Lock()
	defer d.mutex.Unlock()

	lines = append(lines, d.log...)

	return lines
}

func (d *RTCPModalContent) Title() string {
	return "RTCP LOG"
}

// UpdateInterval returns how often the modal content should be updated
func (d *RTCPModalContent) UpdateInterval() time.Duration {
	return 500 * time.Millisecond
}

// AutoScroll returns whether the modal should automatically scroll to the bottom
func (d *RTCPModalContent) AutoScroll() bool {
	return true
}

// Update is called periodically to refresh content
func (d *RTCPModalContent) Update() {
}
