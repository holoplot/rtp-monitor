package pane

import (
	"fmt"

	"github.com/holoplot/rtp-monitor/internal/stream"
)

// New creates a pane of the given kind for stream s.
func New(kind Kind, s *stream.Stream, ctx Context) Pane {
	switch kind {
	case KindDetails:
		return NewDetails(s, ctx.PTP)
	case KindMeters:
		return NewMeters(s)
	case KindSDP:
		return NewSDP(s)
	case KindRTCP:
		return NewRTCP(s)
	case KindRecord:
		return NewRecord(s, ctx.WavFolder)
	case KindFPGA:
		return NewFPGA(s)
	default:
		panic(fmt.Sprintf("pane.New: unsupported kind %v", kind))
	}
}
