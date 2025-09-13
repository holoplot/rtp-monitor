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

	startTime time.Time

	cancelFunc    context.CancelFunc
	err           error
	wavFileFolder string

	recordings []*recording
}

type recording struct {
	ch               chan []stream.SampleFrame
	file             *os.File
	wavEncoder       *wav.Encoder
	bytesCounter     uint64
	lastRecordedTime time.Time
	err              error
}

// NewRecordModalContent creates a new VU modal content provider
func NewRecordModalContent(s *stream.Stream, wavFileFolder string) *RecordModalContent {
	v := &RecordModalContent{
		stream:        s,
		recordings:    make([]*recording, 0),
		wavFileFolder: wavFileFolder,
	}

	return v
}

func (r *RecordModalContent) rtpReceiverCallback(sourceIndex int, _ net.Addr, packet *rtp.Packet) {
	sampleFrames, err := r.receiver.ExtractSamples(packet)
	if err != nil {
		return
	}

	if sourceIndex >= len(r.recordings) {
		return
	}

	r.recordings[sourceIndex].ch <- sampleFrames
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

	ctx, cancelFunc := context.WithCancel(context.Background())
	r.cancelFunc = cancelFunc

	for i := range r.stream.Description.Sources {
		rec := &recording{
			ch:               make(chan []stream.SampleFrame, 1000),
			lastRecordedTime: r.startTime,
		}

		re := regexp.MustCompile(`[^a-zA-Z0-9]`)
		streamName := re.ReplaceAllString(r.stream.Description.Name, "_")
		fileName := fmt.Sprintf("%s_%s-%d.wav", streamName, r.startTime.Format(time.RFC3339), i)
		fileName = path.Join(r.wavFileFolder, fileName)

		outFile, err := os.Create(fileName)
		if err != nil {
			r.err = err

			return
		}

		rec.file = outFile

		rec.wavEncoder = wav.NewEncoder(outFile, int(r.stream.Description.SampleRate), 32,
			int(r.stream.Description.ChannelCount), 1)

		go func() {
			for {
				select {
				case <-ctx.Done():
					return
				case frames := <-rec.ch:
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

					if err := rec.wavEncoder.Write(buf); err != nil {
						rec.err = fmt.Errorf("failed to write to WAV file: %w", err)
						return
					}

					rec.bytesCounter += uint64(len(buf.Data) * 4)
					rec.lastRecordedTime = time.Now()
				}
			}
		}()

		r.recordings = append(r.recordings, rec)
	}

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

	for _, rec := range r.recordings {
		if rec.wavEncoder != nil {
			rec.wavEncoder.Close()
		}

		if rec.file != nil {
			rec.file.Close()
		}
	}
}

// Content returns the content lines to be displayed
func (r *RecordModalContent) Content() []string {
	l := newLineBuffer(lipgloss.NewStyle())

	l.p("RECORDING ...")
	l.p("")

	for i, rec := range r.recordings {
		l.p("Recording %d:", i+1)

		if r.err != nil {
			l.p("Error: %s", r.err)
		} else {
			dur := rec.lastRecordedTime.Sub(r.startTime)
			l.p("  ├─Channels:       %d", r.stream.Description.ChannelCount)
			l.p("  ├─Sample Rate:    %d", r.stream.Description.SampleRate)
			l.p("  ├─File:           %s", rec.file.Name())
			l.p("  ├─Duration:       %02d:%02d.%03d",
				int(dur.Minutes()),
				int(dur.Seconds())%60,
				int(dur.Milliseconds())%1000)

			l.p("  └─Recorded bytes: %s", units.HumanSize(float64(rec.bytesCounter)))
			l.p("")

			l.p("Hit ESC to stop")
		}

	}

	return l.lines()
}

// Title returns the modal title
func (r *RecordModalContent) Title() string {
	return "RECORD WAV FILES"
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
