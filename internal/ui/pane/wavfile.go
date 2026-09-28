package pane

import (
	"bufio"
	"encoding/binary"
	"os"

	"github.com/holoplot/rtp-monitor/internal/stream"
)

const (
	wavHeaderSize = 44
	wavBufferSize = 64 << 10
)

// wavFile writes 32-bit PCM samples to a canonical WAV file. The header's
// size fields are patched in Close.
type wavFile struct {
	f        *os.File
	w        *bufio.Writer
	channels int
	data     uint64 // bytes of sample data written
	sample   [4]byte
}

func createWavFile(path string, sampleRate, channels int) (*wavFile, error) {
	f, err := os.Create(path)
	if err != nil {
		return nil, err
	}

	var h [wavHeaderSize]byte
	le := binary.LittleEndian
	copy(h[0:], "RIFF")
	le.PutUint32(h[4:], wavHeaderSize-8) // patched in Close
	copy(h[8:], "WAVE")
	copy(h[12:], "fmt ")
	le.PutUint32(h[16:], 16)
	le.PutUint16(h[20:], 1) // PCM
	le.PutUint16(h[22:], uint16(channels))
	le.PutUint32(h[24:], uint32(sampleRate))
	le.PutUint32(h[28:], uint32(sampleRate*channels*4))
	le.PutUint16(h[32:], uint16(channels*4))
	le.PutUint16(h[34:], 32)
	copy(h[36:], "data")
	le.PutUint32(h[40:], 0) // patched in Close

	if _, err := f.Write(h[:]); err != nil {
		_ = f.Close()
		return nil, err
	}

	return &wavFile{f: f, w: bufio.NewWriterSize(f, wavBufferSize), channels: channels}, nil
}

// frameBytes is the size of one frame in the file.
func (w *wavFile) frameBytes() uint64 { return uint64(w.channels) * 4 }

// WriteFrames appends frames. Each frame is written with exactly the file's
// channel count, padded with silence or truncated if necessary.
func (w *wavFile) WriteFrames(frames []stream.SampleFrame) error {
	for _, frame := range frames {
		for c := range w.channels {
			var s stream.Sample
			if c < len(frame) {
				s = frame[c]
			}
			binary.LittleEndian.PutUint32(w.sample[:], uint32(s))
			if _, err := w.w.Write(w.sample[:]); err != nil {
				return err
			}
		}
		w.data += w.frameBytes()
	}
	return nil
}

// DataBytes returns the number of sample data bytes written so far.
func (w *wavFile) DataBytes() uint64 { return w.data }

// Close flushes buffered samples, patches the header sizes, syncs and closes
// the file.
func (w *wavFile) Close() error {
	err := w.w.Flush()
	if err == nil {
		err = w.patchHeader()
	}
	if err == nil {
		err = w.f.Sync()
	}
	if cerr := w.f.Close(); err == nil {
		err = cerr
	}
	return err
}

func (w *wavFile) patchHeader() error {
	var b [4]byte
	binary.LittleEndian.PutUint32(b[:], uint32(wavHeaderSize-8+w.data))
	if _, err := w.f.WriteAt(b[:], 4); err != nil {
		return err
	}
	binary.LittleEndian.PutUint32(b[:], uint32(w.data))
	_, err := w.f.WriteAt(b[:], 40)
	return err
}
