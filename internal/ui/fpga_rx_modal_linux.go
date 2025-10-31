//go:build linux

package ui

import (
	"context"
	"fmt"
	"net"
	"os"
	"sync"
	"time"

	"github.com/charmbracelet/lipgloss"
	rsd "github.com/holoplot/ravenna-fpga-drivers/go/stream-device"
	"github.com/holoplot/rtp-monitor/internal/stream"
	"github.com/pion/rtp/v2"
)

const (
	streamDeviceName           = "/dev/ravenna-stream-device"
	streamDeviceSampleRate     = 48000
	streamDeviceStartTrack     = 0
	streamDeviceRtpOffset      = 500
	streamDeviceRtpPayloadType = 98
)

// DetailsModalContent implements ModalContentProvider for stream details
type FpgaRxModalContent struct {
	mutex sync.Mutex

	stream   *stream.Stream
	receiver *stream.RTPReceiver

	streamDevice *rsd.Device
	rxStream     *rsd.RxStream
	rtcpData     *rsd.RxRTCPData

	lastUpdate time.Time
	err        error
	cancelFunc context.CancelFunc
}

func NewFpgaRxModalContent(stream *stream.Stream) *FpgaRxModalContent {
	d := &FpgaRxModalContent{
		stream: stream,
	}

	return d
}

func FpgaRxModalContentAvailable() bool {
	if _, err := os.Stat(streamDeviceName); err == nil {
		return true
	}

	return false
}

func (d *FpgaRxModalContent) Init(width, _ int) {
	d.lastUpdate = time.Now()

	if d.stream.Description.SampleRate != streamDeviceSampleRate {
		d.err = fmt.Errorf("error: sample rate is not %d Hz", streamDeviceSampleRate)

		return
	}

	var codecType rsd.Codec

	switch d.stream.Description.ContentType {
	case stream.ContentTypePCM24:
		codecType = rsd.StreamCodecL24
	default:
		d.err = fmt.Errorf("error: unsupported content type")

		return
	}

	var err error

	// Create a dummy RTP receiver to join the multicast group
	d.receiver, err = d.stream.NewRTPReceiver(func(_ int, _ net.Addr, _ *rtp.Packet) {})
	if err != nil {
		d.err = fmt.Errorf("error creating RTP receiver: %v", err)

		return
	}

	d.streamDevice, err = rsd.Open(streamDeviceName)
	if err != nil {
		d.err = fmt.Errorf("error opening stream device: %v", err)

		return
	}

	rxDesc := rsd.RxStreamDescription{
		Active:             true,
		Synchronous:        true,
		CodecType:          codecType,
		RtpPayloadType:     streamDeviceRtpPayloadType,
		RtpOffset:          streamDeviceRtpOffset,
		JitterBufferMargin: streamDeviceRtpOffset,
		NumChannels:        uint16(d.stream.Description.ChannelCount),
	}

	for ch := range d.stream.Description.ChannelCount {
		rxDesc.Tracks[ch] = streamDeviceStartTrack + int16(ch)
	}

	for i, source := range d.stream.Description.Sources {
		switch i {
		case 0:
			rxDesc.PrimaryDestination = net.UDPAddr{
				IP:   source.DestinationAddress,
				Port: int(source.DestinationPort),
			}
		case 1:
			rxDesc.SecondaryDestination = net.UDPAddr{
				IP:   source.DestinationAddress,
				Port: int(source.DestinationPort),
			}

			// rxDesc.HitlessProtection = true
		default:
			d.err = fmt.Errorf("too many sources")

			return
		}
	}

	d.rxStream, err = d.streamDevice.AddRxStream(rxDesc)
	if err != nil {
		d.err = fmt.Errorf("error adding RX stream: %v", err)

		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	d.cancelFunc = cancel

	go func() {
		for {
			select {
			case <-ctx.Done():
				// return
			case <-time.After(time.Second):
				rtcpData, err := d.rxStream.ReadRTCP(time.Second)
				if err == nil {
					d.mutex.Lock()
					d.rtcpData = &rtcpData
					d.lastUpdate = time.Now()
					d.mutex.Unlock()
				}
			}
		}
	}()
}

func (d *FpgaRxModalContent) Close() {
	if d.cancelFunc != nil {
		d.cancelFunc()
	}

	if d.receiver != nil {
		d.receiver.Close()
	}

	if d.streamDevice != nil {
		_ = d.streamDevice.Close()
	}
}

// Content returns the content lines to be displayed
func (d *FpgaRxModalContent) Content() []string {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	l := newLineBuffer(lipgloss.NewStyle())

	if d.err != nil {
		l.p("Error: %s", d.err)
		return l.lines()
	}

	desc := d.rxStream.Description()

	l.p("Description (stream index %d):", d.rxStream.Index())
	l.p("  ├─ Primary Destination:   %s", desc.PrimaryDestination.String())
	l.p("  ├─ Secondary Destination: %s", desc.SecondaryDestination.String())
	l.p("  ├─ Num Channels:          %d", desc.NumChannels)
	l.p("  ├─ Codec Type:            %s", desc.CodecType)
	l.p("  ├─ RTP Payload Type:      %d", desc.RtpPayloadType)
	l.p("  ├─ VLAN Tag:              %d", desc.VlanTag)
	l.p("  ├─ Jitter Buffer Margin:  %d", desc.JitterBufferMargin)
	l.p("  ├─ RTP Offset:            %d", desc.RtpOffset)
	l.p("  ├─ RTP SSRC:              %d", desc.RtpSsrc)
	l.p("  ├─ Active:                %t", desc.Active)
	l.p("  ├─ Sync Source:           %t", desc.SyncSource)
	l.p("  ├─ VLAN Tagged:           %t", desc.VlanTagged)
	l.p("  ├─ Hitless Protection:    %t", desc.HitlessProtection)
	l.p("  ├─ Synchronous:           %t", desc.Synchronous)
	l.p("  └─ RTP Filter:            %t", desc.RtpFilter)
	l.p("")

	if d.rtcpData != nil {
		l.p("RTCP statistics:")
		l.p("  ├─ Last update:       %s", d.lastUpdate.Format(time.RFC3339))
		l.p("  ├─ RTP Timestamp:     %d", d.rtcpData.RtpTimestamp)
		l.p("  ├─ Device State:      %d", d.rtcpData.DevState)
		l.p("  ├─ RTP Payload ID:    %d", d.rtcpData.RtpPayloadId)
		l.p("  ├─ Offset Estimation: %d", d.rtcpData.OffsetEstimation)
		l.p("  └─ Path Differential: %d", d.rtcpData.PathDifferential)
		l.p("")

		forInterface := func(s string, i rsd.RxRTCPInterfaceData) {
			l.p("%s:", s)
			l.p("  ├─ Playing:            %t", i.Playing)
			l.p("  ├─ Error:              %t", i.Error)
			l.p("  ├─ Misordered Packets: %d", i.MisorderedPackets)
			l.p("  ├─ Base Sequence Nr:   %d", i.BaseSequenceNr)
			l.p("  ├─ Extended Max SeqNr: %d", i.ExtendedMaxSequenceNr)
			l.p("  ├─ Received Packets:   %d", i.ReceivedPackets)
			l.p("  ├─ Peak Jitter:        %d", i.PeakJitter)
			l.p("  ├─ Estimated Jitter:   %d", i.EstimatedJitter)
			l.p("  ├─ Last Transit Time:  %d", i.LastTransitTime)
			l.p("  ├─ Offset Estimation:  %d", i.CurrentOffsetEstimation)
			l.p("  ├─ Last SSRC:          %08x", i.LastSsrc)
			l.p("  ├─ Buffer Margin Min:  %d", i.BufferMarginMin)
			l.p("  ├─ Buffer Margin Max:  %d", i.BufferMarginMax)
			l.p("  ├─ Late Packets:       %d", i.LatePackets)
			l.p("  ├─ Early Packets:      %d", i.EarlyPackets)
			l.p("  └─ Timeout Counter:    %d", i.TimeoutCounter)
			l.p("")
		}

		forInterface("Primary", d.rtcpData.Primary)
		forInterface("Secondary", d.rtcpData.Secondary)
	} else {
		l.p("No RTCP data available")
	}

	return l.lines()
}

func (d *FpgaRxModalContent) Title() string {
	return "RAVENNA FPGA RX STREAMING"
}

// UpdateInterval returns how often the modal content should be updated
func (d *FpgaRxModalContent) UpdateInterval() time.Duration {
	return 500 * time.Millisecond
}

// AutoScroll returns whether the modal should automatically scroll to the bottom
func (d *FpgaRxModalContent) AutoScroll() bool {
	return false
}

// Update is called periodically to refresh content
func (d *FpgaRxModalContent) Update() {
}
