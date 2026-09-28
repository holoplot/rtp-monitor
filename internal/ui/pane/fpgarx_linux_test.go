//go:build linux

package pane

import (
	"strings"
	"testing"
)

func TestFPGARejectsWrongSampleRate(t *testing.T) {
	s := testStream()
	s.Description.SampleRate = 44100
	f := NewFPGA(s)
	defer f.Close()

	if !strings.Contains(strings.Join(rows(f.View(80, 10)), "\n"), "sample rate is not 48000 Hz") {
		t.Error("missing sample rate error")
	}
}
