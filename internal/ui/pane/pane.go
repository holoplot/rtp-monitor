// Package pane contains the panes of the tiling UI. Every pane renders into
// the size it is given; borders and titles are drawn by package ui.
package pane

import (
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/holoplot/rtp-monitor/internal/ptp"
	"github.com/holoplot/rtp-monitor/internal/stream"
)

// Kind is the type of a pane.
type Kind int

const (
	KindStreamList Kind = iota
	KindDetails
	KindMeters
	KindSDP
	KindRTCP
	KindRecord
	KindFPGA
)

func (k Kind) String() string {
	switch k {
	case KindStreamList:
		return "Streams"
	case KindDetails:
		return "Details"
	case KindMeters:
		return "Meters"
	case KindSDP:
		return "SDP"
	case KindRTCP:
		return "RTCP"
	case KindRecord:
		return "Record"
	case KindFPGA:
		return "FPGA RX"
	default:
		return "Unknown"
	}
}

// Pane is one tile of the UI.
type Pane interface {
	Kind() Kind
	// Title is shown in the pane's top border, e.g. "Meters · Studio A".
	Title() string
	// StreamID is the ID of the bound stream, "" for the stream list.
	StreamID() string
	// Gone reports whether the bound stream is currently not announced.
	Gone() bool
	// SetStream passes the current snapshot of the bound stream, nil if gone.
	SetStream(s *stream.Stream)
	// Update receives TickMsg (all panes), keys (focused pane), ScrollMsg
	// (pane under the mouse) and ClickMsg.
	Update(msg tea.Msg) tea.Cmd
	// View renders the content area, exactly w×h cells.
	View(w, h int) string
	// CopyText is what 'c' copies to the clipboard.
	CopyText() string
	// Close releases receivers and finalizes files. It must return promptly.
	Close()
}

// TickMsg is sent to every pane on each render tick.
type TickMsg time.Time

// ScrollMsg asks a pane to scroll by Lines; negative scrolls up.
type ScrollMsg struct{ Lines int }

// ClickMsg is a left click at content-relative cell (X, Y).
type ClickMsg struct{ X, Y int }

// OpenMsg asks the UI to open a pane of Kind for Stream.
type OpenMsg struct {
	Kind   Kind
	Stream *stream.Stream
}

// Context carries the dependencies panes need besides their stream.
type Context struct {
	PTP       *ptp.Monitor // nil when PTP monitoring is unavailable
	WavFolder string
}
