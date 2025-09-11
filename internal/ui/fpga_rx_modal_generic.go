//go:build !linux

package ui

import (
	"time"

	"github.com/holoplot/rtp-monitor/internal/stream"
)

// DetailsModalContent implements ModalContentProvider for stream details
type FpgaRxModalContent struct {
}

func NewFpgaRxModalContent(stream *stream.Stream) *FpgaRxModalContent {
	return &FpgaRxModalContent{}
}

func (d *FpgaRxModalContent) Init(_, _ int) {}

func (d *FpgaRxModalContent) Close() {
}

// Content returns the content lines to be displayed
func (d *FpgaRxModalContent) Content() []string {
	return []string{"FPGA streaming is only available on Linux"}
}

func (d *FpgaRxModalContent) Title() string {
	return "RAVENNA FPGA RX STREAMING [UNAVAILABLE]"
}

// UpdateInterval returns how often the modal content should be updated
func (d *FpgaRxModalContent) UpdateInterval() time.Duration {
	return 0
}

// AutoScroll returns whether the modal should automatically scroll to the bottom
func (d *FpgaRxModalContent) AutoScroll() bool {
	return false
}

// Update is called periodically to refresh content
func (d *FpgaRxModalContent) Update() {
}
