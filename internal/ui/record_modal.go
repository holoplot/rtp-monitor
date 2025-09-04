package ui

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"os"
	"path"
	"regexp"
	"sync"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/docker/go-units"
	"github.com/go-audio/audio"
	"github.com/go-audio/wav"
	"github.com/holoplot/rtp-monitor/internal/stream"
	"github.com/holoplot/rtp-monitor/internal/theme"
	"github.com/pion/rtp/v2"
)

// VUModalContent implements ModalContentProvider for VU meter display
type RecordModalContent struct {
	mutex sync.Mutex

	width        int
	height       int
	contentWidth int

	stream   *stream.Stream
	receiver *stream.RTPReceiver

	startTime        time.Time
	lastRecordedTime time.Time

	wavFileFolder string
	ch            chan []stream.SampleFrame
	cancelFunc    context.CancelFunc
	file          *os.File
	wavEncoder    *wav.Encoder
	err           error
	bytesCounter  uint64
}

// NewRecordModalContent creates a new VU modal content provider
func NewRecordModalContent(s *stream.Stream, wavFileFolder string) *RecordModalContent {
	v := &RecordModalContent{
		stream:        s,
		ch:            make(chan []stream.SampleFrame, 1000),
		wavFileFolder: wavFileFolder,
	}

	return v
}

// createVUModalStyles creates the VU modal styles
func createRecordModalStyles() VUModalStyles {
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

func (r *RecordModalContent) rtpReceiverCallback(sourceIndex int, _ net.Addr, packet *rtp.Packet) {
	sampleFrames, err := r.receiver.ExtractSamples(packet)
	if err != nil {
		return
	}

	r.ch <- sampleFrames
}

// Init initializes the content provider with dimensions
func (r *RecordModalContent) Init(width, height int) {
	r.width = width
	r.height = height

	// Calculate content width (90% of screen, similar to original modal)
	r.contentWidth = (width * 90) / 100
	if r.contentWidth < 90 {
		r.contentWidth = 90
	}
	if r.contentWidth > width-4 {
		r.contentWidth = width - 4
	}
	r.contentWidth -= 4 // Account for modal padding

	r.startTime = time.Now()
	r.lastRecordedTime = r.startTime

	re := regexp.MustCompile(`[^a-zA-Z0-9]`)
	streamName := re.ReplaceAllString(r.stream.Description.Name, "_")
	fileName := fmt.Sprintf("%s_%s.wav", streamName, r.startTime.Format(time.RFC3339))
	fileName = path.Join(r.wavFileFolder, fileName)

	outFile, err := os.Create(fileName)
	if err != nil {
		r.err = err

		return
	}

	r.file = outFile

	r.wavEncoder = wav.NewEncoder(outFile, int(r.stream.Description.SampleRate), 32,
		int(r.stream.Description.ChannelCount), 1)

	ctx, cancelFunc := context.WithCancel(context.Background())
	r.cancelFunc = cancelFunc

	go func() {
		defer cancelFunc()
		for {
			select {
			case <-ctx.Done():
				return
			case frames := <-r.ch:
				buf := &audio.IntBuffer{
					Format: &audio.Format{
						NumChannels: int(r.stream.Description.ChannelCount),
						SampleRate:  int(r.stream.Description.SampleRate),
					},
					Data:           make([]int, 0),
					SourceBitDepth: 32,
				}

				for _, frame := range frames {
					for _, sample := range frame {
						buf.Data = append(buf.Data, int(sample))
					}
				}

				if err := r.wavEncoder.Write(buf); err != nil {
					r.err = fmt.Errorf("failed to write to WAV file: %w", err)
					return
				}

				r.bytesCounter += uint64(len(buf.Data) * 4)
				r.lastRecordedTime = time.Now()
			}
		}
	}()

	if receiver, err := r.stream.NewRTPReceiver(r.rtpReceiverCallback); err == nil {
		r.receiver = receiver
	} else {
		slog.Error("Failed to create receiver", "error", err)
	}
}

func (r *RecordModalContent) Close() {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if r.receiver != nil {
		r.receiver.Close()
	}

	if r.wavEncoder != nil {
		r.wavEncoder.Close()
	}

	if r.file != nil {
		r.file.Close()
	}
}

// Content returns the content lines to be displayed
func (r *RecordModalContent) Content() []string {
	l := newLineBuffer(lipgloss.NewStyle())

	dur := r.lastRecordedTime.Sub(r.startTime)

	l.h("RECORDING ...")
	l.p("")

	if r.err != nil {
		l.p("Error: %s", r.err)
	} else {
		l.p("Channels:     %d", r.stream.Description.ChannelCount)
		l.p("Sample Rate:  %d", r.stream.Description.SampleRate)
		l.p("File:         %s", r.file.Name())
		l.p("")
		l.p("Duration:     %02d:%02d.%03d",
			int(dur.Minutes()),
			int(dur.Seconds())%60,
			int(dur.Milliseconds())%1000)

		l.p("Record bytes: %s", units.HumanSize(float64(r.bytesCounter)))
		l.p("")

		l.p("Hit ESC to stop")
	}

	return l.lines()
}

// Title returns the modal title
func (r *RecordModalContent) Title() string {
	return "RECORD WAV FILE"
}

// UpdateInterval returns how often the modal content should be updated
func (r *RecordModalContent) UpdateInterval() time.Duration {
	// Update VU meters frequently for smooth animation
	return 50 * time.Millisecond
}

// AutoScroll returns whether the modal should automatically scroll to the bottom
func (r *RecordModalContent) AutoScroll() bool {
	return false
}

// Update is called periodically to refresh VU meter data
func (r *RecordModalContent) Update() {
}
