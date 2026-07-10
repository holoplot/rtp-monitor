package ui

import (
	"fmt"
	"math"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/holoplot/rtp-monitor/internal/ring"
	"github.com/holoplot/rtp-monitor/internal/stream"
	"github.com/holoplot/rtp-monitor/internal/theme"
	"github.com/pion/rtp/v2"
)

type floatSample float64

const (
	clipThreshold = -0.1 // dBFS
	clipTimeout   = time.Second * 5
)

// MeterModalContent implements ModalContentProvider for Meter meter display
type MeterModalContent struct {
	mutex sync.Mutex

	width        int
	height       int
	styles       MeterModalStyles
	contentWidth int

	stream   *stream.Stream
	receiver *stream.RTPReceiver

	err error

	sourceMeters []*sourceMeters
}

// MeterModalStyles holds the styling for the Meter modal content
type MeterModalStyles struct {
	StreamName lipgloss.Style
	MeterClip  lipgloss.Style
	ScaleLabel lipgloss.Style
	Background lipgloss.Style
}

type sourceMeters struct {
	channelMeters []*channelMeter
	lastUpdate    time.Time
}

// channelMeter holds the current state of a meter channel
type channelMeter struct {
	levels      *ring.RingBuffer[floatSample]
	clipTime    time.Time
	progressBar *MeterProgress
}

// NewMeterModalContent creates a new Meter modal content provider
func NewMeterModalContent(s *stream.Stream) *MeterModalContent {
	v := &MeterModalContent{
		stream:       s,
		styles:       createMeterModalStyles(),
		sourceMeters: make([]*sourceMeters, len(s.Description.Sources)),
	}

	for i := range len(s.Description.Sources) {
		sourceMeter := &sourceMeters{
			channelMeters: make([]*channelMeter, s.Description.ChannelCount),
			lastUpdate:    time.Now(),
		}

		sourceMeter.channelMeters = make([]*channelMeter, s.Description.ChannelCount)

		for i := range s.Description.ChannelCount {
			sourceMeter.channelMeters[i] = &channelMeter{
				levels:      ring.NewRingBuffer[floatSample](2400),
				progressBar: NewMeterProgress(50, v.styles.Background), // Default width
			}
		}

		v.sourceMeters[i] = sourceMeter
	}

	return v
}

// createMeterModalStyles creates the Meter modal styles
func createMeterModalStyles() MeterModalStyles {
	return MeterModalStyles{
		StreamName: lipgloss.NewStyle().
			Foreground(theme.Colors.Secondary).
			Bold(true).
			Width(20),
		MeterClip: lipgloss.NewStyle().
			Foreground(theme.Colors.StatusError).
			Bold(true),
		ScaleLabel: lipgloss.NewStyle().
			Foreground(theme.Colors.Secondary),
		Background: lipgloss.NewStyle().
			Background(theme.Colors.Background),
	}
}

func (v *MeterModalContent) rtpReceiverCallback(sourceIndex int, _ net.Addr, packet *rtp.Packet) {
	// The callback might fire before NewRTPReceiver() returns. Just ignore that packet.
	if v.receiver == nil {
		return
	}

	if sourceIndex >= len(v.sourceMeters) {
		panic(fmt.Sprintf("source %d out of range", sourceIndex))
	}

	channelMeters := v.sourceMeters[sourceIndex].channelMeters
	v.sourceMeters[sourceIndex].lastUpdate = time.Now()

	sampleFrames, err := v.receiver.ExtractSamples(packet)
	if err != nil {
		return
	}

	for _, frame := range sampleFrames {
		for ch, value := range frame {
			s := floatSample(int32(value)) / floatSample(math.MaxInt32)
			channelMeters[ch].levels.Push(s * s)
		}
	}
}

// Init initializes the content provider with dimensions
func (v *MeterModalContent) Init(width, height int) {
	v.width = width
	v.height = height

	// Calculate content width (90% of screen, similar to original modal)
	v.contentWidth = max((width*90)/100, 90)
	if v.contentWidth > width-4 {
		v.contentWidth = width - 4
	}
	v.contentWidth -= 4 // Account for modal padding

	if receiver, err := v.stream.NewRTPReceiver(v.rtpReceiverCallback); err == nil {
		v.receiver = receiver
	} else {
		v.err = err
	}
}

func (v *MeterModalContent) Close() {
	v.mutex.Lock()
	defer v.mutex.Unlock()

	if v.receiver != nil {
		v.receiver.Close()
	}
}

func (v *MeterModalContent) renderSourceMeters(sm *sourceMeters, meterWidth int) []string {
	if len(sm.channelMeters) == 0 {
		return []string{"No meter data available"}
	}

	var lines []string

	// dB Scale (shown once at the top)
	scale := fmt.Sprintf("%15s%s", "", v.renderDBScale(meterWidth))
	lines = append(lines, scale)
	lines = append(lines, "")

	for ch, meter := range sm.channelMeters {
		if time.Since(sm.lastUpdate) > time.Second {
			meter.levels.Clear()
		}

		samples := meter.levels.ToSlice()
		rmsDB := math.Inf(-1)
		peakDB := math.Inf(-1)

		if len(samples) > 0 {
			sumSquares := floatSample(0)
			peakSquared := floatSample(0)

			for _, sample := range samples {
				sumSquares += sample
				if sample > peakSquared {
					peakSquared = sample
				}
			}

			meanSquares := sumSquares / floatSample(len(samples))
			rmsDB = 10 * math.Log10(float64(meanSquares))
			peakDB = 10 * math.Log10(float64(peakSquared))

			if math.IsNaN(rmsDB) {
				panic(fmt.Sprintf("NaN encountered in channel %d, len(samples)=%d, meanSquares=%f samples=%v", ch+1, len(samples), meanSquares, samples))
			}

			if peakDB > clipThreshold {
				meter.clipTime = time.Now()
			}
		}

		clipping := time.Since(meter.clipTime) < clipTimeout

		channelLabel := fmt.Sprintf("Ch%d", ch+1)
		dbText := fmt.Sprintf("%6.1f dB", rmsDB)
		meterLine := v.renderMeterMeter(meter, peakDB, rmsDB, meterWidth)
		clipIndicator := v.renderClipIndicator(clipping)

		line := fmt.Sprintf("  %-3s %s %s %s", channelLabel, dbText, meterLine, clipIndicator)
		lines = append(lines, line)
	}

	lines = append(lines, "")
	lines = append(lines, "")

	return lines
}

// Content returns the content lines to be displayed
func (v *MeterModalContent) Content() []string {
	var lines []string

	// Calculate meter width: total width minus labels, dB text, and clip indicator
	meterWidth := max(v.contentWidth-55, 20)

	v.mutex.Lock()
	defer v.mutex.Unlock()

	if v.err != nil {
		lines = append(lines, fmt.Sprintf("Error creating stream receiver: %v", v.err))

		return lines
	}

	for i, source := range v.stream.Description.Sources {
		ip := fmt.Sprintf("%s:%d", source.DestinationAddress, source.DestinationPort)
		lines = append(lines, fmt.Sprintf("%s:", ip))
		lines = append(lines, "")
		lines = append(lines, v.renderSourceMeters(v.sourceMeters[i], meterWidth)...)
	}

	return lines
}

// Title returns the modal title
func (v *MeterModalContent) Title() string {
	return "METERS"
}

// UpdateInterval returns how often the modal content should be updated
func (v *MeterModalContent) UpdateInterval() time.Duration {
	// Update Meter meters frequently for smooth animation
	return 50 * time.Millisecond
}

// AutoScroll returns whether the modal should automatically scroll to the bottom
func (v *MeterModalContent) AutoScroll() bool {
	return false
}

// Update is called periodically to refresh Meter meter data
func (v *MeterModalContent) Update() {
}

// renderDBScale renders the dB scale at the top
func (v *MeterModalContent) renderDBScale(width int) string {
	markers := []float64{-100, -80, -60, -40, -20, -10, -6, -3, 0}
	scale := strings.Repeat(" ", width)
	scaleRunes := []rune(scale)

	for _, db := range markers {
		percentage := v.dbToPercentage(db)
		pos := max(int(percentage*float64(width)), 0)
		if pos >= width {
			pos = width - 1
		}

		label := strconv.Itoa(int(db))
		if db == 0 {
			label = "0dB"
		}

		// Place label at position, but don't overflow
		labelRunes := []rune(label)
		for i, r := range labelRunes {
			if pos+i < len(scaleRunes) && pos+i >= 0 {
				scaleRunes[pos+i] = r
			}
		}
	}

	return string(scaleRunes)
}

// renderMeterMeter renders a meter bar showing peak extent with an RMS marker
func (v *MeterModalContent) renderMeterMeter(meter *channelMeter, peakDB, rmsDB float64, width int) string {
	if width < 10 {
		width = 10
	}

	meter.progressBar.SetWidth(width)

	return meter.progressBar.ViewAs(v.dbToPercentage(peakDB), v.dbToPercentage(rmsDB))
}

// renderClipIndicator renders the clip indicator
func (v *MeterModalContent) renderClipIndicator(clipping bool) string {
	s := ""

	if clipping {
		s = "CLIP"
	}

	return v.styles.MeterClip.Width(4).Render(s)
}

// dbToPercentage converts dB value to percentage for progress bar
func (v *MeterModalContent) dbToPercentage(db float64) float64 {
	// Map -60dB to 0dB to 0.0 to 1.0
	if db <= -100 {
		return 0.0
	}
	if db >= 0 {
		return 1.0
	}

	percentage := (db + 100) / 100
	if percentage < 0 {
		percentage = 0
	}
	if percentage > 1 {
		percentage = 1
	}

	return percentage
}
