package pane

import (
	"math"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
	"github.com/holoplot/rtp-monitor/internal/stream"
	"github.com/pion/rtp/v2"
)

func TestLevelsDB(t *testing.T) {
	rms, peak := levelsDB([]float64{0.25, 0.25})
	if math.Abs(rms+6.0206) > 0.001 || math.Abs(peak+6.0206) > 0.001 {
		t.Errorf("levelsDB = %.4f, %.4f; want -6.0206 twice", rms, peak)
	}
	for _, in := range [][]float64{nil, {0, 0}} {
		if rms, peak := levelsDB(in); !math.IsInf(rms, -1) || !math.IsInf(peak, -1) {
			t.Errorf("levelsDB(%v) = %v, %v; want -Inf", in, rms, peak)
		}
	}
}

func TestDBToFraction(t *testing.T) {
	for db, want := range map[float64]float64{-120: 0, -100: 0, -50: 0.5, 0: 1, 3: 1} {
		if got := dbToFraction(db); got != want {
			t.Errorf("dbToFraction(%v) = %v, want %v", db, got, want)
		}
	}
}

func fullScale(frames, channels int) *rtp.Packet {
	payload := make([]byte, 0, frames*channels*3)
	for range frames * channels {
		payload = append(payload, 0x7F, 0xFF, 0xFF)
	}
	return &rtp.Packet{Payload: payload}
}

func TestMetersShowClip(t *testing.T) {
	m := NewMeters(testStream())
	defer m.Close()

	m.state.onPacket(0, fullScale(48, 2))
	m.state.onPacket(3, fullScale(48, 2)) // unknown source: ignored

	r := rows(m.View(80, 12))
	view := strings.Join(r, "\n")
	for _, want := range []string{"239.1.1.1:5004:", "Ch1", "Ch2", "CLIP", "0dB"} {
		if !strings.Contains(view, want) {
			t.Errorf("view lacks %q:\n%s", want, view)
		}
	}
	for _, row := range r {
		if ansi.StringWidth(row) != 80 {
			t.Errorf("row width %d, want 80: %q", ansi.StringWidth(row), row)
		}
	}
}

func TestMetersRebuildOnFormatChange(t *testing.T) {
	m := NewMeters(testStream())
	defer m.Close()

	s := testStream()
	s.Description.ChannelCount = 4
	m.SetStream(s)
	if n := len(m.state.sources[0].channels); n != 4 {
		t.Fatalf("%d channel meters, want 4", n)
	}
}

func TestMetersUnsupportedEncoding(t *testing.T) {
	s := testStream()
	s.Description.ContentType = stream.ContentTypeUndefined
	m := NewMeters(s)
	defer m.Close()

	m.state.onPacket(0, fullScale(48, 2)) // must not panic
	if !strings.Contains(strings.Join(rows(m.View(80, 10)), "\n"), "Unsupported encoding") {
		t.Error("missing unsupported encoding message")
	}
}
