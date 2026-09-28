package pane

import (
	"fmt"
	"math"
	"net"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/holoplot/rtp-monitor/internal/ring"
	"github.com/holoplot/rtp-monitor/internal/stream"
	"github.com/holoplot/rtp-monitor/internal/theme"
	"github.com/pion/rtp/v2"
)

const (
	clipThreshold = -0.1 // dBFS
	clipHold      = 5 * time.Second
	meterWindow   = 2400 // squared samples kept per channel
	staleAfter    = time.Second
	// meterPrefix is the width of "  Ch1  -12.3 dB ", the text left of a bar.
	meterPrefix = 17
)

// Meters shows peak and RMS meters for every channel of every source.
type Meters struct {
	base
	state    *meterState
	receiver *stream.RTPReceiver
	err      error
}

var _ Pane = (*Meters)(nil)

// meterState holds the levels for one receiver and stream format.
type meterState struct {
	desc    stream.StreamDescription
	sources []*sourceMeter
}

type sourceMeter struct {
	lastPacket atomic.Int64 // unix nanoseconds, written by the callback
	channels   []*channelMeter
}

type channelMeter struct {
	levels   *ring.RingBuffer[float64]
	clipTime time.Time // UI goroutine only
}

func newMeterState(desc stream.StreamDescription) *meterState {
	m := &meterState{desc: desc, sources: make([]*sourceMeter, len(desc.Sources))}
	for i := range m.sources {
		sm := &sourceMeter{channels: make([]*channelMeter, desc.ChannelCount)}
		for ch := range sm.channels {
			sm.channels[ch] = &channelMeter{levels: ring.NewRingBuffer[float64](meterWindow)}
		}
		m.sources[i] = sm
	}
	return m
}

// onPacket runs on a receiver goroutine; the ring buffers are thread-safe.
func (m *meterState) onPacket(i int, p *rtp.Packet) {
	if i >= len(m.sources) {
		return
	}
	frames, err := m.desc.ExtractSamples(p)
	if err != nil {
		return
	}

	sm := m.sources[i]
	sm.lastPacket.Store(time.Now().UnixNano())
	for _, frame := range frames {
		for ch, v := range frame {
			if ch >= len(sm.channels) {
				break
			}
			s := float64(v) / math.MaxInt32
			sm.channels[ch].levels.Push(s * s)
		}
	}
}

func NewMeters(s *stream.Stream) *Meters {
	m := &Meters{base: newBase(KindMeters, s)}
	m.start()
	return m
}

func (m *Meters) start() {
	m.stop()
	state := newMeterState(m.stream.Description)
	m.state = state
	m.receiver, m.err = m.stream.NewRTPReceiver(func(i int, _ net.Addr, p *rtp.Packet) {
		state.onPacket(i, p)
	})
}

func (m *Meters) stop() {
	if m.receiver != nil {
		m.receiver.Close()
		m.receiver = nil
	}
}

func (m *Meters) SetStream(s *stream.Stream) {
	if m.bind(s) != stream.DescriptionUnchanged {
		m.start()
	}
}

func (m *Meters) Update(msg tea.Msg) tea.Cmd {
	if k, ok := msg.(tea.KeyPressMsg); ok && k.String() == "enter" && m.err != nil {
		m.start()
		return nil
	}
	m.handleScroll(msg)
	return nil
}

func (m *Meters) View(w, h int) string { return m.renderLines(m.lines(w), w, h) }
func (m *Meters) CopyText() string     { return ansiFree(m.lines(80)) }
func (m *Meters) Close()               { m.stop() }

func (m *Meters) lines(w int) []string {
	var out []string
	if m.err != nil {
		out = append(out, fmt.Sprintf("Error creating stream receiver: %v", m.err), "Press enter to retry", "")
	}

	desc := m.state.desc
	if !desc.CanDecode() {
		return append(out, fmt.Sprintf("Unsupported encoding %q with %d channels", desc.ContentType, desc.ChannelCount))
	}

	barW := max(w-2-meterPrefix-5, 10) // scrollbar, prefix, " CLIP"
	clip := lipgloss.NewStyle().Foreground(theme.UI.Error).Bold(true).Render("CLIP")
	now := time.Now()

	for i, src := range desc.Sources {
		sm := m.state.sources[i]
		out = append(out, fmt.Sprintf("%s:%d:", src.DestinationAddress, src.DestinationPort), "",
			strings.Repeat(" ", meterPrefix)+dbScale(barW), "")

		stale := now.Sub(time.Unix(0, sm.lastPacket.Load())) > staleAfter
		for ch, cm := range sm.channels {
			if stale {
				cm.levels.Clear()
			}
			rms, peak := levelsDB(cm.levels.ToSlice())
			if peak > clipThreshold {
				cm.clipTime = now
			}
			indicator := ""
			if now.Sub(cm.clipTime) < clipHold {
				indicator = clip
			}
			out = append(out, fmt.Sprintf("  %-4s %6.1f dB %s %s",
				"Ch"+strconv.Itoa(ch+1), rms, renderMeterBar(dbToFraction(peak), dbToFraction(rms), barW), indicator))
		}
		out = append(out, "")
	}
	return out
}

// levelsDB returns the RMS and peak level in dBFS of squared samples.
func levelsDB(squares []float64) (rms, peak float64) {
	if len(squares) == 0 {
		return math.Inf(-1), math.Inf(-1)
	}
	var sum, peakSq float64
	for _, s := range squares {
		sum += s
		peakSq = max(peakSq, s)
	}
	return 10 * math.Log10(sum/float64(len(squares))), 10 * math.Log10(peakSq)
}

// dbToFraction maps -100..0 dBFS linearly to 0..1.
func dbToFraction(db float64) float64 { return clamp01((db + 100) / 100) }

// dbScale renders dB labels positioned like the meter bars below them.
func dbScale(width int) string {
	scale := []rune(strings.Repeat(" ", width))
	for _, db := range []float64{-100, -80, -60, -40, -20, -10, -6, -3, 0} {
		pos := min(int(dbToFraction(db)*float64(width)), width-1)
		label := strconv.Itoa(int(db))
		if db == 0 {
			label = "0dB"
			pos = max(width-len(label), 0)
		}
		for j, r := range label {
			if pos+j < len(scale) {
				scale[pos+j] = r
			}
		}
	}
	return string(scale)
}

func ansiFree(lines []string) string {
	for i, l := range lines {
		lines[i] = ansi.Strip(l)
	}
	return strings.Join(lines, "\n")
}
