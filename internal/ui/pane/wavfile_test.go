package pane

import (
	"bufio"
	"encoding/binary"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/holoplot/rtp-monitor/internal/stream"
)

// wavInfo is the parsed canonical 44-byte header of a WAV file plus its
// sample data.
type wavInfo struct {
	riffSize, dataSize uint32
	format, channels   uint16
	rate, byteRate     uint32
	blockAlign, bits   uint16
	samples            []int32
}

func readWav(t *testing.T, path string) wavInfo {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(b) < 44 || string(b[0:4]) != "RIFF" || string(b[8:12]) != "WAVE" || string(b[12:16]) != "fmt " || string(b[36:40]) != "data" {
		t.Fatalf("%s: not a canonical WAV header: %q", path, b[:min(len(b), 44)])
	}
	le := binary.LittleEndian
	w := wavInfo{
		riffSize:   le.Uint32(b[4:]),
		format:     le.Uint16(b[20:]),
		channels:   le.Uint16(b[22:]),
		rate:       le.Uint32(b[24:]),
		byteRate:   le.Uint32(b[28:]),
		blockAlign: le.Uint16(b[32:]),
		bits:       le.Uint16(b[34:]),
		dataSize:   le.Uint32(b[40:]),
	}
	if le.Uint32(b[16:]) != 16 {
		t.Fatalf("%s: fmt chunk size %d", path, le.Uint32(b[16:]))
	}
	if int(w.dataSize) != len(b)-44 || int(w.riffSize) != len(b)-8 {
		t.Fatalf("%s: header sizes riff=%d data=%d, file has %d bytes", path, w.riffSize, w.dataSize, len(b))
	}
	for i := 44; i+4 <= len(b); i += 4 {
		w.samples = append(w.samples, int32(le.Uint32(b[i:])))
	}
	return w
}

func TestWavFileHeaderAndData(t *testing.T) {
	path := filepath.Join(t.TempDir(), "x.wav")
	f, err := createWavFile(path, 48000, 2)
	if err != nil {
		t.Fatal(err)
	}
	if err := f.WriteFrames([]stream.SampleFrame{{1, -1}, {1 << 30, -(1 << 30)}}); err != nil {
		t.Fatal(err)
	}
	if f.DataBytes() != 16 {
		t.Fatalf("DataBytes = %d, want 16", f.DataBytes())
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}

	w := readWav(t, path)
	if w.format != 1 || w.channels != 2 || w.rate != 48000 || w.bits != 32 || w.blockAlign != 8 || w.byteRate != 48000*8 {
		t.Fatalf("header %+v", w)
	}
	want := []int32{1, -1, 1 << 30, -(1 << 30)}
	if len(w.samples) != len(want) {
		t.Fatalf("samples %v, want %v", w.samples, want)
	}
	for i := range want {
		if w.samples[i] != want[i] {
			t.Fatalf("samples %v, want %v", w.samples, want)
		}
	}
}

// shortWriter writes at most n bytes to w and then fails, like a full disk.
type shortWriter struct {
	w io.Writer
	n int
}

func (s *shortWriter) Write(p []byte) (int, error) {
	if len(p) > s.n {
		n, _ := s.w.Write(p[:s.n])
		s.n = 0
		return n, errors.New("disk full")
	}
	s.n -= len(p)
	return s.w.Write(p)
}

func TestWavFileCloseAfterWriteError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "x.wav")
	f, err := createWavFile(path, 48000, 2)
	if err != nil {
		t.Fatal(err)
	}
	// 20 bytes reach the disk: two whole stereo frames and half of the third.
	f.w = bufio.NewWriterSize(&shortWriter{w: f.f, n: 20}, 8)
	err = f.WriteFrames([]stream.SampleFrame{{1, 2}, {3, 4}, {5, 6}, {7, 8}})
	if err == nil {
		t.Fatal("WriteFrames succeeded, want an error")
	}
	if err := f.Close(); err == nil {
		t.Fatal("Close succeeded, want the write error")
	}

	w := readWav(t, path) // checks the header sizes against the data
	if w.dataSize != 16 || len(w.samples) != 4 || w.samples[3] != 4 {
		t.Fatalf("data %d bytes, samples %v; want the two whole frames on disk", w.dataSize, w.samples)
	}
}
