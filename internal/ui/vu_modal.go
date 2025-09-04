package ui

import (
	"fmt"
	"log/slog"
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

// VUModalContent implements ModalContentProvider for VU meter display
type VUModalContent struct {
	mutex sync.Mutex

	width        int
	height       int
	styles       VUModalStyles
	contentWidth int

	stream   *stream.Stream
	receiver *stream.RTPReceiver

	sourceMeters []*sourceMeters
}

// VUModalStyles holds the styling for the VU modal content
type VUModalStyles struct {
	StreamName lipgloss.Style
	MeterClip  lipgloss.Style
	ScaleLabel lipgloss.Style
	Reset      lipgloss.Style
	Background lipgloss.Style
}

type sourceMeters struct {
	channelMeters []*channelMeter
}

// channelMeter holds the current state of a VU meter
type channelMeter struct {
	maxSample     floatSample
	levels        *ring.RingBuffer[floatSample]
	clipIndicator bool
	clipTime      time.Time
	progressBar   *VUProgress // VU progress bars for each channel
}

// NewVUModalContent creates a new VU modal content provider
func NewVUModalContent(s *stream.Stream) *VUModalContent {
	v := &VUModalContent{
		stream:       s,
		styles:       createVUModalStyles(),
		sourceMeters: make([]*sourceMeters, len(s.Description.Sources)),
	}

	for i := range len(s.Description.Sources) {
		sourceMeter := &sourceMeters{
			channelMeters: make([]*channelMeter, s.Description.ChannelCount),
		}

		sourceMeter.channelMeters = make([]*channelMeter, s.Description.ChannelCount)

		for i := range s.Description.ChannelCount {
			sourceMeter.channelMeters[i] = &channelMeter{
				levels:      ring.NewRingBuffer[floatSample](10000),
				progressBar: NewVUProgress(50, v.styles.Background), // Default width
			}
		}

		v.sourceMeters[i] = sourceMeter
	}

	return v
}

// createVUModalStyles creates the VU modal styles
func createVUModalStyles() VUModalStyles {
	return VUModalStyles{
		StreamName: lipgloss.NewStyle().
			Foreground(theme.Colors.Secondary).
			Bold(true).
			Width(20),
		MeterClip: lipgloss.NewStyle().
			Foreground(theme.Colors.StatusError).
			Background(theme.Colors.Background).
			Bold(true),
		ScaleLabel: lipgloss.NewStyle().
			Foreground(theme.Colors.Secondary),
		Reset: lipgloss.NewStyle().
			Foreground(theme.Colors.Primary).
			Background(theme.Colors.Background),
		Background: lipgloss.NewStyle().
			Background(theme.Colors.Background),
	}
}

func (v *VUModalContent) rtpReceiverCallback(sourceIndex int, _ net.Addr, packet *rtp.Packet) {
	if sourceIndex >= len(v.sourceMeters) {
		panic(fmt.Sprintf("source %d out of range", sourceIndex))
	}

	channelMeters := v.sourceMeters[sourceIndex].channelMeters

	sampleFrames, err := v.receiver.ExtractSamples(packet)
	if err != nil {
		return
	}

	for _, frame := range sampleFrames {
		for ch, value := range frame {
			s := floatSample(int32(value)) / floatSample(math.MaxInt32)
			s = floatSample(math.Abs(float64(s)))

			channelMeters[ch].levels.Push(s)
		}
	}
}

// Init initializes the content provider with dimensions
func (v *VUModalContent) Init(width, height int) {
	v.width = width
	v.height = height

	// Calculate content width (90% of screen, similar to original modal)
	v.contentWidth = (width * 90) / 100
	if v.contentWidth < 90 {
		v.contentWidth = 90
	}
	if v.contentWidth > width-4 {
		v.contentWidth = width - 4
	}
	v.contentWidth -= 4 // Account for modal padding

	if receiver, err := v.stream.NewRTPReceiver(v.rtpReceiverCallback); err == nil {
		v.receiver = receiver
	} else {
		slog.Error("Failed to create receiver", "error", err)
	}
}

func (v *VUModalContent) Close() {
	v.mutex.Lock()
	defer v.mutex.Unlock()

	if v.receiver != nil {
		v.receiver.Close()
	}
}

func (v *VUModalContent) renderSourceMeters(sm *sourceMeters, meterWidth int) []string {
	if len(sm.channelMeters) == 0 {
		return []string{"No meter data available"}
	}

	var lines []string

	// dB Scale (shown once at the top)
	scale := fmt.Sprintf("%15s%s", "", v.renderDBScale(meterWidth))
	lines = append(lines, scale)
	lines = append(lines, "")

	for ch, meter := range sm.channelMeters {
		samples := meter.levels.ToSlice()
		db := math.Inf(-1)

		if len(samples) > 0 {
			avg := floatSample(0)

			for _, sample := range samples {
				avg += sample
			}

			avg /= floatSample(len(samples))
			db = math.Log10(float64(avg)) * 20

			if math.IsNaN(db) {
				panic(fmt.Sprintf("NaN encountered in channel %d, len(samples)=%d, avg=%f samples=%v", ch+1, len(samples), avg, samples))
			}
		}

		channelLabel := fmt.Sprintf("Ch%d", ch+1)
		dbText := fmt.Sprintf("%6.1f dB", db)
		meterLine := v.renderVUMeter(meter, db, meterWidth)
		clipIndicator := v.renderClipIndicator(meter.clipIndicator)

		line := fmt.Sprintf("  %-3s %s %s %s", channelLabel, dbText, meterLine, clipIndicator)
		lines = append(lines, line)
	}

	lines = append(lines, "")
	lines = append(lines, "")

	return lines
}

// Content returns the content lines to be displayed
func (v *VUModalContent) Content() []string {
	var lines []string

	// Calculate meter width: total width minus labels, dB text, and clip indicator
	meterWidth := v.contentWidth - 55
	if meterWidth < 20 {
		meterWidth = 20
	}

	v.mutex.Lock()
	defer v.mutex.Unlock()

	for i, source := range v.stream.Description.Sources {
		ip := fmt.Sprintf("%s:%d", source.DestinationAddress, source.DestinationPort)
		lines = append(lines, fmt.Sprintf("%s:", ip))
		lines = append(lines, "")
		lines = append(lines, v.renderSourceMeters(v.sourceMeters[i], meterWidth)...)
	}

	return lines
}

// Title returns the modal title
func (v *VUModalContent) Title() string {
	return "VU METERS"
}

// UpdateInterval returns how often the modal content should be updated
func (v *VUModalContent) UpdateInterval() time.Duration {
	// Update VU meters frequently for smooth animation
	return 50 * time.Millisecond
}

// AutoScroll returns whether the modal should automatically scroll to the bottom
func (v *VUModalContent) AutoScroll() bool {
	return false
}

// Update is called periodically to refresh VU meter data
func (v *VUModalContent) Update() {
}

// renderDBScale renders the dB scale at the top
func (v *VUModalContent) renderDBScale(width int) string {
	markers := []float64{-100, -80, -60, -40, -20, -10, -6, -3, 0}
	scale := strings.Repeat(" ", width)
	scaleRunes := []rune(scale)

	for _, db := range markers {
		percentage := v.dbToPercentage(db)
		pos := int(percentage * float64(width))
		if pos < 0 {
			pos = 0
		}
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

// renderVUMeter renders a single VU meter using progress component
func (v *VUModalContent) renderVUMeter(meter *channelMeter, level float64, width int) string {
	if width < 10 {
		width = 10
	}

	meter.progressBar.SetWidth(width)
	percentage := v.dbToPercentage(level)

	return meter.progressBar.ViewAs(percentage)
}

// renderClipIndicator renders the clip indicator
func (v *VUModalContent) renderClipIndicator(clipping bool) string {
	s := ""

	if clipping {
		s = "CLIP"
	}

	return v.styles.MeterClip.Width(4).Render(s)
}

// dbToPercentage converts dB value to percentage for progress bar
func (v *VUModalContent) dbToPercentage(db float64) float64 {
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
