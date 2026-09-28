package pane

import (
	"fmt"
	"os"
	"slices"
	"sync"
	"sync/atomic"
	"time"

	"github.com/holoplot/rtp-monitor/internal/stream"
)

const recorderQueue = 1000 // packets buffered between receiver and disk

// maxWavDataBytes caps the sample data of one WAV file, whose RIFF sizes are
// 32 bit. A variable so tests can lower it.
var maxWavDataBytes uint64 = 4<<30 - 1<<20

// wavRecorder writes sample frames to 32-bit WAV files on its own goroutine.
// Push never blocks, so receiver callbacks can call it directly. Before a file
// would exceed maxWavDataBytes it is finalized and recording continues in a
// new file named after the first one with ".1", ".2", … (see uniquePath).
type wavRecorder struct {
	basePath          string
	sampleRate, chans int
	file              *wavFile // nil after a failed rotation; owned by run until done

	fileMu  sync.Mutex // guards path, started and parts
	path    string
	started time.Time
	parts   []recStats // files finished by rotation

	mu     sync.RWMutex // guards closed and closing ch
	closed bool
	ch     chan []stream.SampleFrame
	done   chan struct{}

	bytes     atomic.Uint64 // of the current file
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
	f, err := createWavFile(path, sampleRate, channels)
	if err != nil {
		return nil, err
	}

	r := &wavRecorder{
		basePath:   path,
		sampleRate: sampleRate,
		chans:      channels,
		file:       f,
		path:       path,
		started:    time.Now(),
		ch:         make(chan []stream.SampleFrame, recorderQueue),
		done:       make(chan struct{}),
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

		// Rotate between packets so frames are never split across files.
		need := uint64(len(frames)) * r.file.frameBytes()
		if r.file.DataBytes() > 0 && r.file.DataBytes()+need > maxWavDataBytes {
			if err := r.rotate(); err != nil {
				r.setWriteErr(fmt.Errorf("failed to start the next WAV file: %w", err))
				continue
			}
		}

		before := r.file.DataBytes()
		if err := r.file.WriteFrames(frames); err != nil {
			r.setWriteErr(fmt.Errorf("failed to write to WAV file: %w", err))
			continue
		}
		r.bytes.Add(r.file.DataBytes() - before)
		r.lastWrite.Store(time.Now().UnixNano())
	}
}

// rotate finalizes the current file and continues in a new one.
func (r *wavRecorder) rotate() error {
	st := r.Stats()
	st.Dropped, st.Err = 0, r.file.Close()
	r.file = nil

	r.fileMu.Lock()
	r.parts = append(r.parts, st)
	r.fileMu.Unlock()

	// Stats describe the new file from here on, even if creating it fails, so
	// the finished file is not reported twice.
	path := uniquePath(r.basePath)
	r.fileMu.Lock()
	r.path, r.started = path, time.Now()
	r.bytes.Store(0)
	r.lastWrite.Store(0)
	r.fileMu.Unlock()

	f, err := createWavFile(path, r.sampleRate, r.chans)
	if err != nil {
		return err
	}
	r.file = f
	return nil
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

	if r.file == nil {
		return nil
	}
	err := r.file.Close()
	if r.file.DataBytes() == 0 {
		_ = os.Remove(r.path)
	}
	return err
}

// Stats describes the current file.
func (r *wavRecorder) Stats() recStats {
	r.fileMu.Lock()
	st := recStats{Path: r.path, Bytes: r.bytes.Load(), Dropped: r.dropped.Load(), Err: r.writeErr()}
	if last := r.lastWrite.Load(); last > 0 {
		st.Duration = time.Unix(0, last).Sub(r.started)
	}
	r.fileMu.Unlock()
	return st
}

// Parts describes the files finished by rotation, oldest first.
func (r *wavRecorder) Parts() []recStats {
	r.fileMu.Lock()
	defer r.fileMu.Unlock()
	return slices.Clone(r.parts)
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
