//go:build !linux

package pane

import (
	tea "charm.land/bubbletea/v2"
	"github.com/holoplot/rtp-monitor/internal/stream"
)

// FPGAAvailable reports whether the Ravenna FPGA stream device exists.
func FPGAAvailable() bool { return false }

// FPGA is a placeholder; FPGA streaming is only available on Linux.
type FPGA struct {
	base
}

var _ Pane = (*FPGA)(nil)

func NewFPGA(s *stream.Stream) *FPGA { return &FPGA{base: newBase(KindFPGA, s)} }

func (f *FPGA) SetStream(s *stream.Stream) { f.bind(s) }

func (f *FPGA) Update(msg tea.Msg) tea.Cmd {
	f.handleScroll(msg)
	return nil
}

func (f *FPGA) View(w, h int) string {
	return f.renderLines([]string{"FPGA streaming is only available on Linux"}, w, h)
}

func (f *FPGA) CopyText() string { return "" }
func (f *FPGA) Close()           {}
