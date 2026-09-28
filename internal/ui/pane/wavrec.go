package pane

import (
	"fmt"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/go-audio/audio"
	"github.com/go-audio/wav"
	"github.com/holoplot/rtp-monitor/internal/stream"
)

const recorderQueue = 1000 // packets buffered between receiver and disk

// wavRecorder writes sample frames to a 32-bit WAV file on its own goroutine.
// Push never blocks, so receiver callbacks can call it directly.
type wavRecorder struct {
	path     string
	file     *os.File
	enc      *wav.Encoder
	rate     int
	channels int
	started  time.Time

	mu     sync.RWMutex // guards closed and closing ch
	closed bool
	ch     chan []stream.SampleFrame
	done   chan struct{}

	bytes     atomic.Uint64
	dropped   atomic.Uint64
	lastWrite atomic.Int64 // unix nanoseconds

	errMu sync.Mutex
	err   error
}

// recStats is a snapshot of a recorder's progress.
type recStats struct {
	Path           string
	Bytes, Dropped uint64
	Duration       time.Duration
	Err            error
}

func newWavRecorder(path string, sampleRate, channels int) (*wavRecorder, error) {
	f, err := os.Create(path)
	if err != nil {
		return nil, err
	}

	r := &wavRecorder{
		path:     path,
		file:     f,
		enc:      wav.NewEncoder(f, sampleRate, 32, channels, 1),
		rate:     sampleRate,
		channels: channels,
		started:  time.Now(),
		ch:       make(chan []stream.SampleFrame, recorderQueue),
		done:     make(chan struct{}),
	}
	go r.run()

	return r, nil
}

// Push queues frames for writing. It returns false, and counts a dropped
// packet, if the queue is full or the recorder is closed.
func (r *wavRecorder) Push(frames []stream.SampleFrame) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if r.closed {
		return false
	}
	select {
	case r.ch <- frames:
		return true
	default:
		r.dropped.Add(1)
		return false
	}
}

func (r *wavRecorder) run() {
	defer close(r.done)

	for frames := range r.ch {
		if r.writeErr() != nil {
			continue // drain the queue after a write error
		}

		buf := &audio.IntBuffer{
			Format:         &audio.Format{NumChannels: r.channels, SampleRate: r.rate},
			SourceBitDepth: 32,
			Data:           make([]int, 0, len(frames)*r.channels),
		}
		for _, frame := range frames {
			for _, s := range frame {
				buf.Data = append(buf.Data, int(s))
			}
		}

		if err := r.enc.Write(buf); err != nil {
			r.setWriteErr(fmt.Errorf("failed to write to WAV file: %w", err))
			continue
		}
		r.bytes.Add(uint64(len(buf.Data) * 4))
		r.lastWrite.Store(time.Now().UnixNano())
	}
}

// Close writes the queued frames, finalizes the WAV header and closes the
// file. Files without audio data are removed. Calling Close twice is safe.
func (r *wavRecorder) Close() error {
	r.mu.Lock()
	if r.closed {
		r.mu.Unlock()
		return nil
	}
	r.closed = true
	close(r.ch)
	r.mu.Unlock()

	<-r.done

	err := r.enc.Close()
	if cerr := r.file.Close(); err == nil {
		err = cerr
	}
	if r.bytes.Load() == 0 {
		_ = os.Remove(r.path)
	}
	return err
}

func (r *wavRecorder) Stats() recStats {
	st := recStats{Path: r.path, Bytes: r.bytes.Load(), Dropped: r.dropped.Load(), Err: r.writeErr()}
	if last := r.lastWrite.Load(); last > 0 {
		st.Duration = time.Unix(0, last).Sub(r.started)
	}
	return st
}

func (r *wavRecorder) writeErr() error {
	r.errMu.Lock()
	defer r.errMu.Unlock()
	return r.err
}

func (r *wavRecorder) setWriteErr(err error) {
	r.errMu.Lock()
	r.err = err
	r.errMu.Unlock()
}
