//go:build linux

package pane

import (
	"context"
	"fmt"
	"net"
	"os"
	"strings"
	"sync"
	"time"

	tea "charm.land/bubbletea/v2"
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

// FPGAAvailable reports whether the Ravenna FPGA stream device exists.
func FPGAAvailable() bool {
	_, err := os.Stat(streamDeviceName)
	return err == nil
}

// FPGA streams a stream through the Ravenna FPGA receiver and shows its state.
type FPGA struct {
	base
	receiver *stream.RTPReceiver
	device   *rsd.Device
	rxStream *rsd.RxStream
	cancel   context.CancelFunc
	err      error

	mu         sync.Mutex // guards rtcpData and lastUpdate
	rtcpData   *rsd.RxRTCPData
	lastUpdate time.Time
	pollWg     sync.WaitGroup // guards polling goroutine lifecycle
}

var _ Pane = (*FPGA)(nil)

func NewFPGA(s *stream.Stream) *FPGA {
	f := &FPGA{base: newBase(KindFPGA, s)}
	f.start()
	return f
}

func (f *FPGA) start() {
	f.stop()
	if f.err = f.open(); f.err != nil {
		f.stop()
	}
}

func (f *FPGA) open() error {
	desc := f.stream.Description
	if desc.SampleRate != streamDeviceSampleRate {
		return fmt.Errorf("sample rate is not %d Hz", streamDeviceSampleRate)
	}
	if desc.ContentType != stream.ContentTypePCM24 {
		return fmt.Errorf("unsupported content type %q", desc.ContentType)
	}
	if len(desc.Sources) > 2 {
		return fmt.Errorf("too many sources (%d)", len(desc.Sources))
	}

	rxDesc := rsd.RxStreamDescription{
		Active:             true,
		Synchronous:        true,
		CodecType:          rsd.StreamCodecL24,
		RtpPayloadType:     streamDeviceRtpPayloadType,
		RtpOffset:          streamDeviceRtpOffset,
		JitterBufferMargin: streamDeviceRtpOffset,
		NumChannels:        uint16(desc.ChannelCount),
	}
	if int(desc.ChannelCount) > len(rxDesc.Tracks) {
		return fmt.Errorf("too many channels (%d)", desc.ChannelCount)
	}
	for ch := range desc.ChannelCount {
		rxDesc.Tracks[ch] = streamDeviceStartTrack + int16(ch)
	}
	for i, src := range desc.Sources {
		addr := net.UDPAddr{IP: src.DestinationAddress, Port: int(src.DestinationPort)}
		if i == 0 {
			rxDesc.PrimaryDestination = addr
		} else {
			rxDesc.SecondaryDestination = addr
		}
	}

	var err error

	// A receiver that ignores packets joins the multicast groups for the FPGA.
	if f.receiver, err = f.stream.NewRTPReceiver(func(int, net.Addr, *rtp.Packet) {}); err != nil {
		return fmt.Errorf("creating RTP receiver: %w", err)
	}
	if f.device, err = rsd.Open(streamDeviceName); err != nil {
		return fmt.Errorf("opening stream device: %w", err)
	}
	if f.rxStream, err = f.device.AddRxStream(rxDesc); err != nil {
		return fmt.Errorf("adding RX stream: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	f.cancel = cancel
	f.pollWg.Add(1)
	go f.pollRTCP(ctx, f.rxStream)

	return nil
}

func (f *FPGA) pollRTCP(ctx context.Context, rx *rsd.RxStream) {
	defer f.pollWg.Done()

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			data, err := rx.ReadRTCP(time.Second)
			if err != nil {
				continue
			}
			f.mu.Lock()
			f.rtcpData, f.lastUpdate = &data, time.Now()
			f.mu.Unlock()
		}
	}
}

func (f *FPGA) stop() {
	if f.cancel != nil {
		f.cancel()
		f.cancel = nil
	}
	f.pollWg.Wait() // wait for polling goroutine to exit before closing resources
	if f.receiver != nil {
		f.receiver.Close()
		f.receiver = nil
	}
	if f.device != nil {
		_ = f.device.Close()
		f.device = nil
	}
	f.rxStream = nil

	f.mu.Lock()
	f.rtcpData = nil
	f.mu.Unlock()
}

func (f *FPGA) SetStream(s *stream.Stream) {
	if f.bind(s) != stream.DescriptionUnchanged {
		f.start()
	}
}

func (f *FPGA) Update(msg tea.Msg) tea.Cmd {
	if k, ok := msg.(tea.KeyPressMsg); ok && k.String() == "enter" && f.err != nil {
		f.start()
		return nil
	}
	f.handleScroll(msg)
	return nil
}

func (f *FPGA) View(w, h int) string { return f.renderLines(f.lines(), w, h) }
func (f *FPGA) CopyText() string     { return strings.Join(f.lines(), "\n") }
func (f *FPGA) Close()               { f.stop() }

func (f *FPGA) lines() []string {
	var l lineBuffer

	if f.err != nil {
		l.p("Error: %s", f.err)
		l.p("Press enter to retry")
		return l
	}

	desc := f.rxStream.Description()
	l.p("Description (stream index %d):", f.rxStream.Index())
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

	f.mu.Lock()
	defer f.mu.Unlock()

	if f.rtcpData == nil {
		l.p("No RTCP data available")
		return l
	}

	d := f.rtcpData
	l.p("RTCP statistics:")
	l.p("  ├─ Last update:       %s", f.lastUpdate.Format(time.RFC3339))
	l.p("  ├─ RTP Timestamp:     %d", d.RtpTimestamp)
	l.p("  ├─ Device State:      %d", d.DevState)
	l.p("  ├─ RTP Payload ID:    %d", d.RtpPayloadId)
	l.p("  ├─ Offset Estimation: %d", d.OffsetEstimation)
	l.p("  └─ Path Differential: %d", d.PathDifferential)
	l.p("")

	for _, iface := range []struct {
		name string
		data rsd.RxRTCPInterfaceData
	}{{"Primary", d.Primary}, {"Secondary", d.Secondary}} {
		i := iface.data
		l.p("%s:", iface.name)
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

	return l
}
