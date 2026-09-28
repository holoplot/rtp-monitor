package pane

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/go-audio/wav"
	"github.com/holoplot/rtp-monitor/internal/stream"
)

func TestWavRecorderWritesFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "rec.wav")
	r, err := newWavRecorder(path, 48000, 2)
	if err != nil {
		t.Fatal(err)
	}

	frames := []stream.SampleFrame{{1 << 16, -(1 << 16)}, {2 << 16, -(2 << 16)}}
	for range 10 {
		if !r.Push(frames) {
			t.Fatal("push dropped")
		}
	}
	if err := r.Close(); err != nil {
		t.Fatal(err)
	}
	if st := r.Stats(); st.Bytes != 160 || st.Dropped != 0 || st.Path != path {
		t.Fatalf("stats %+v", st)
	}

	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	d := wav.NewDecoder(f)
	buf, err := d.FullPCMBuffer()
	if err != nil {
		t.Fatal(err)
	}
	if d.NumChans != 2 || d.SampleRate != 48000 || d.BitDepth != 32 || len(buf.Data) != 40 {
		t.Fatalf("decoded %d ch, %d Hz, %d bit, %d samples", d.NumChans, d.SampleRate, d.BitDepth, len(buf.Data))
	}
	if buf.Data[0] != 1<<16 || buf.Data[1] != -(1<<16) {
		t.Fatalf("first frame %v", buf.Data[:2])
	}
}

func TestWavRecorderRemovesEmptyFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "empty.wav")
	r, err := newWavRecorder(path, 48000, 2)
	if err != nil {
		t.Fatal(err)
	}
	_ = r.Close()
	if _, err := os.Stat(path); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("empty file kept: %v", err)
	}
}

func TestWavRecorderPushNeverBlocks(t *testing.T) {
	r := &wavRecorder{ch: make(chan []stream.SampleFrame, 1)} // no writer goroutine
	if !r.Push(nil) {
		t.Fatal("first push dropped")
	}
	if r.Push(nil) {
		t.Fatal("push into a full queue succeeded")
	}
	if r.dropped.Load() != 1 {
		t.Fatalf("dropped = %d, want 1", r.dropped.Load())
	}
}

func TestWavRecorderPushAfterClose(t *testing.T) {
	r, err := newWavRecorder(filepath.Join(t.TempDir(), "x.wav"), 48000, 1)
	if err != nil {
		t.Fatal(err)
	}
	_ = r.Close()
	if r.Push([]stream.SampleFrame{{1}}) {
		t.Fatal("push after close succeeded")
	}
	_ = r.Close() // idempotent
}

func TestWavRecorderCreateError(t *testing.T) {
	if _, err := newWavRecorder(filepath.Join(t.TempDir(), "missing", "x.wav"), 48000, 1); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("err = %v, want ErrNotExist", err)
	}
}
