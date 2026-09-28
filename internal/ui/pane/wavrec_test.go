package pane

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
	"time"

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

	w := readWav(t, path)
	if w.channels != 2 || w.rate != 48000 || w.bits != 32 || len(w.samples) != 40 {
		t.Fatalf("decoded %d ch, %d Hz, %d bit, %d samples", w.channels, w.rate, w.bits, len(w.samples))
	}
	if w.samples[0] != 1<<16 || w.samples[1] != -(1<<16) {
		t.Fatalf("first frame %v", w.samples[:2])
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

func lowerWavLimit(t *testing.T, n uint64) {
	t.Helper()
	old := maxWavDataBytes
	maxWavDataBytes = n
	t.Cleanup(func() { maxWavDataBytes = old })
}

func TestWavRecorderRotatesBeforeLimit(t *testing.T) {
	lowerWavLimit(t, 40) // two packets of two stereo frames (16 bytes each) per file
	dir := t.TempDir()
	path := filepath.Join(dir, "rec.wav")
	r, err := newWavRecorder(path, 48000, 2)
	if err != nil {
		t.Fatal(err)
	}

	var next stream.Sample
	for range 5 {
		frames := []stream.SampleFrame{{next, next + 1}, {next + 2, next + 3}}
		next += 4
		if !r.Push(frames) {
			t.Fatal("push dropped")
		}
	}
	if err := r.Close(); err != nil {
		t.Fatal(err)
	}

	paths := []string{path, filepath.Join(dir, "rec.1.wav"), filepath.Join(dir, "rec.2.wav")}
	sizes := []uint64{32, 32, 16}

	parts := r.Parts()
	if len(parts) != 2 {
		t.Fatalf("parts = %+v, want 2", parts)
	}
	for i, p := range parts {
		if p.Path != paths[i] || p.Bytes != sizes[i] || p.Err != nil {
			t.Fatalf("part %d = %+v, want %s with %d bytes", i, p, paths[i], sizes[i])
		}
	}
	if st := r.Stats(); st.Path != paths[2] || st.Bytes != sizes[2] {
		t.Fatalf("stats %+v, want %s with %d bytes", st, paths[2], sizes[2])
	}

	var all []int32
	for i, p := range paths {
		w := readWav(t, p) // checks the header sizes against the data
		if uint64(w.dataSize) != sizes[i] || w.dataSize > 40 {
			t.Fatalf("%s: %d data bytes, want %d", p, w.dataSize, sizes[i])
		}
		all = append(all, w.samples...)
	}
	if len(all) != int(next) {
		t.Fatalf("%d samples across files, want %d", len(all), next)
	}
	for i, s := range all {
		if s != int32(i) {
			t.Fatalf("sample %d = %d: data lost or reordered across files", i, s)
		}
	}
	if entries, _ := os.ReadDir(dir); len(entries) != 3 {
		t.Fatalf("%d files, want 3", len(entries))
	}
}

func TestWavRecorderFailedRotation(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root ignores directory permissions")
	}
	lowerWavLimit(t, 16)
	dir := t.TempDir()
	path := filepath.Join(dir, "rec.wav")
	r, err := newWavRecorder(path, 48000, 2)
	if err != nil {
		t.Fatal(err)
	}
	r.Push([]stream.SampleFrame{{1, 2}, {3, 4}})
	waitFor(t, func() bool { return r.Stats().Bytes == 16 })

	if err := os.Chmod(dir, 0o555); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(dir, 0o755) })
	r.Push([]stream.SampleFrame{{5, 6}})
	waitFor(t, func() bool { return r.Stats().Err != nil })

	st := r.Stats()
	if want := filepath.Join(dir, "rec.1.wav"); st.Path != want || st.Bytes != 0 {
		t.Fatalf("stats %+v, want the failed file %s with no bytes", st, want)
	}
	if parts := r.Parts(); len(parts) != 1 || parts[0].Path != path || parts[0].Err != nil {
		t.Fatalf("parts %+v, want only the finished %s", parts, path)
	}
	_ = r.Close()
}

func waitFor(t *testing.T, cond func() bool) {
	t.Helper()
	for deadline := time.Now().Add(5 * time.Second); !cond(); {
		if time.Now().After(deadline) {
			t.Fatal("timed out")
		}
		time.Sleep(time.Millisecond)
	}
}
