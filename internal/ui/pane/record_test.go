package pane

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/holoplot/rtp-monitor/internal/stream"
	"github.com/pion/rtp/v2"
)

var fixedNow = func() time.Time { return time.Date(2026, 9, 28, 12, 0, 0, 0, time.UTC) }

func TestRecordWritesAndRotates(t *testing.T) {
	dir := t.TempDir()
	r := newRecord(testStream(), dir, fixedNow)

	r.state.onPacket(0, &rtp.Packet{Payload: make([]byte, 48*2*3)})
	r.state.onPacket(4, &rtp.Packet{Payload: make([]byte, 48*2*3)}) // unknown source: ignored

	first := filepath.Join(dir, "Alpha_2026-09-28T12:00:00Z-0.wav")
	if view := strings.Join(rows(r.View(160, 30)), "\n"); !strings.Contains(view, first) {
		t.Fatalf("view lacks %s:\n%s", first, view)
	}

	s2 := testStream()
	s2.Description.ChannelCount = 4
	r.SetStream(s2)

	if len(r.finished) != 1 || r.finished[0].Bytes != 48*2*4 || r.finished[0].Path != first {
		t.Fatalf("finished = %+v", r.finished)
	}
	if got := r.state.recorders[0].Stats().Path; !strings.HasSuffix(got, "-0.1.wav") {
		t.Fatalf("second file %s, want a .1 suffix", got)
	}
	if !strings.Contains(strings.Join(rows(r.View(160, 30)), "\n"), "Finished files:") {
		t.Fatal("finished files not listed")
	}

	r.Close()
	entries, _ := os.ReadDir(dir)
	if len(entries) != 1 {
		t.Fatalf("%d files left, want 1 (the empty second file is removed)", len(entries))
	}
}

func TestRecordTransportChangeKeepsFiles(t *testing.T) {
	r := newRecord(testStream(), t.TempDir(), fixedNow)
	defer r.Close()
	rec := r.state.recorders[0]

	s := testStream()
	s.Description.Sources[0].DestinationPort = 5006
	r.SetStream(s)
	if r.state.recorders[0] != rec || len(r.finished) != 0 {
		t.Fatal("transport change rotated the files")
	}

	r.SetStream(nil)
	if !r.Gone() || r.state.recorders[0] != rec {
		t.Fatal("gone stream stopped the recording")
	}
}

func TestRecordUnwritableFolder(t *testing.T) {
	r := newRecord(testStream(), filepath.Join(t.TempDir(), "missing"), fixedNow)
	defer r.Close()

	if !errors.Is(r.state.errs[0], fs.ErrNotExist) {
		t.Fatalf("errs[0] = %v, want ErrNotExist", r.state.errs[0])
	}
	if !strings.Contains(strings.Join(rows(r.View(120, 20)), "\n"), "Error:") {
		t.Error("error not shown")
	}
	r.state.onPacket(0, &rtp.Packet{Payload: make([]byte, 48*2*3)}) // must not panic
}

func TestRecordFolderIsRegularFile(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "not-a-dir")
	if err := os.WriteFile(file, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	done := make(chan *Record, 1)
	go func() { done <- newRecord(testStream(), file, fixedNow) }()

	select {
	case r := <-done:
		defer r.Close()
		if r.state.errs[0] == nil {
			t.Fatal("errs[0] = nil, want an error (ENOTDIR)")
		}
		if !strings.Contains(strings.Join(rows(r.View(120, 20)), "\n"), "Error:") {
			t.Error("error not shown")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("newRecord did not return within 2s: uniquePath likely looping forever")
	}
}

func TestRecordFinishedFileErrorShown(t *testing.T) {
	r := newRecord(testStream(), t.TempDir(), fixedNow)
	defer r.Close()

	r.finished = append(r.finished, recStats{Path: "x.wav", Err: errors.New("boom")})

	if !strings.Contains(strings.Join(rows(r.View(160, 30)), "\n"), "boom") {
		t.Error("error on a finished file not shown")
	}
}

func TestRecordUnsupportedEncoding(t *testing.T) {
	dir := t.TempDir()
	s := testStream()
	s.Description.ContentType = stream.ContentTypeUndefined
	r := newRecord(s, dir, fixedNow)
	defer r.Close()

	if !strings.Contains(strings.Join(rows(r.View(120, 20)), "\n"), "unsupported encoding") {
		t.Error("missing unsupported encoding message")
	}
	if entries, _ := os.ReadDir(dir); len(entries) != 0 {
		t.Errorf("%d files created, want 0", len(entries))
	}
}

func TestRecordListsRotatedFiles(t *testing.T) {
	lowerWavLimit(t, 500) // one 384-byte packet per file
	dir := t.TempDir()
	r := newRecord(testStream(), dir, fixedNow)
	defer r.Close()

	for range 3 {
		r.state.onPacket(0, &rtp.Packet{Payload: make([]byte, 48*2*3)})
	}
	s2 := testStream()
	s2.Description.ChannelCount = 4
	r.SetStream(s2)

	base := filepath.Join(dir, "Alpha_2026-09-28T12:00:00Z-0")
	want := []string{base + ".wav", base + ".1.wav", base + ".2.wav"}
	if len(r.finished) != len(want) {
		t.Fatalf("finished = %+v, want %d files", r.finished, len(want))
	}
	for i, st := range r.finished {
		if st.Path != want[i] || st.Bytes != 384 {
			t.Fatalf("finished[%d] = %+v, want %s with 384 bytes", i, st, want[i])
		}
	}
	view := strings.Join(rows(r.View(160, 40)), "\n")
	if !strings.Contains(view, base+".1.wav (384B)") {
		t.Fatalf("rotated file not listed:\n%s", view)
	}
}
