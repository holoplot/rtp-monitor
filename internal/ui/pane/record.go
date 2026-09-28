package pane

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/docker/go-units"
	"github.com/holoplot/rtp-monitor/internal/stream"
	"github.com/pion/rtp/v2"
)

// Record writes one WAV file per source while the pane is open. A format
// change finishes the current files and starts new ones.
type Record struct {
	base
	folder      string
	now         func() time.Time
	state       *recState
	finished    []recStats
	receiver    *stream.RTPReceiver
	err         error // receiver error
	encodingErr error
}

var _ Pane = (*Record)(nil)

// recState holds the recorders of one stream format.
type recState struct {
	desc      stream.StreamDescription
	recorders []*wavRecorder // nil where the file could not be created
	errs      []error
}

func (s *recState) onPacket(i int, p *rtp.Packet) {
	if i >= len(s.recorders) || s.recorders[i] == nil {
		return
	}
	frames, err := s.desc.ExtractSamples(p)
	if err != nil {
		return
	}
	s.recorders[i].Push(frames)
}

func (s *recState) close() []recStats {
	var out []recStats
	for _, r := range s.recorders {
		if r == nil {
			continue
		}
		err := r.Close()
		out = append(out, r.Parts()...)
		st := r.Stats()
		if st.Err == nil {
			st.Err = err
		}
		out = append(out, st)
	}
	return out
}

var fileNameUnsafe = regexp.MustCompile(`[^a-zA-Z0-9]`)

func NewRecord(s *stream.Stream, folder string) *Record {
	return newRecord(s, folder, time.Now)
}

func newRecord(s *stream.Stream, folder string, now func() time.Time) *Record {
	r := &Record{base: newBase(KindRecord, s), folder: folder, now: now}
	r.openFiles()
	r.startReceiver()
	return r
}

// openFiles finishes the current files and opens one WAV file per source.
func (r *Record) openFiles() {
	if r.state != nil {
		r.finished = append(r.finished, r.state.close()...)
	}

	desc := r.stream.Description
	st := &recState{
		desc:      desc,
		recorders: make([]*wavRecorder, len(desc.Sources)),
		errs:      make([]error, len(desc.Sources)),
	}
	r.state = st

	r.encodingErr = nil
	if !desc.CanDecode() {
		r.encodingErr = fmt.Errorf("unsupported encoding %q with %d channels", desc.ContentType, desc.ChannelCount)
		return
	}

	stamp := r.now().Format(time.RFC3339)
	name := fileNameUnsafe.ReplaceAllString(r.stream.Name(), "_")
	for i := range desc.Sources {
		path := uniquePath(filepath.Join(r.folder, fmt.Sprintf("%s_%s-%d.wav", name, stamp, i)))
		st.recorders[i], st.errs[i] = newWavRecorder(path, int(desc.SampleRate), int(desc.ChannelCount))
	}
}

// uniquePath returns p, or p with ".1", ".2", … before the extension if p
// exists. It only keeps searching while the candidate demonstrably exists;
// any Stat error (not just "not exist") ends the search immediately so
// os.Create can report the real error, e.g. via st.errs[i].
func uniquePath(p string) string {
	if _, err := os.Stat(p); err != nil {
		return p
	}
	ext := filepath.Ext(p)
	stem := strings.TrimSuffix(p, ext)
	for i := 1; ; i++ {
		candidate := fmt.Sprintf("%s.%d%s", stem, i, ext)
		if _, err := os.Stat(candidate); err != nil {
			return candidate
		}
	}
}

func (r *Record) startReceiver() {
	r.stopReceiver()
	st := r.state
	r.receiver, r.err = r.stream.NewRTPReceiver(func(i int, _ net.Addr, p *rtp.Packet) {
		st.onPacket(i, p)
	})
}

func (r *Record) stopReceiver() {
	if r.receiver != nil {
		r.receiver.Close()
		r.receiver = nil
	}
}

func (r *Record) SetStream(s *stream.Stream) {
	switch r.bind(s) {
	case stream.DescriptionFormatChanged:
		r.stopReceiver()
		r.openFiles()
		r.startReceiver()
	case stream.DescriptionTransportChanged:
		r.startReceiver()
	}
}

func (r *Record) Update(msg tea.Msg) tea.Cmd {
	if k, ok := msg.(tea.KeyPressMsg); ok && k.String() == "enter" && r.err != nil {
		r.startReceiver()
		return nil
	}
	r.handleScroll(msg)
	return nil
}

func (r *Record) View(w, h int) string { return r.renderLines(r.lines(), w, h) }
func (r *Record) CopyText() string     { return strings.Join(r.lines(), "\n") }

func (r *Record) Close() {
	r.stopReceiver()
	r.state.close()
}

func (r *Record) lines() []string {
	var l lineBuffer

	if r.err != nil {
		l.p("Error creating stream receiver: %v", r.err)
		l.p("Press enter to retry")
		l.p("")
	}
	if r.encodingErr != nil {
		l.p("Error: %v", r.encodingErr)
		l.p("")
	}

	desc := r.state.desc
	for i, rec := range r.state.recorders {
		l.p("Source %d:", i+1)
		if rec == nil {
			if err := r.state.errs[i]; err != nil {
				l.p("  └─ Error:    %v", err)
			}
			l.p("")
			continue
		}

		st := rec.Stats()
		errText := "-"
		if st.Err != nil {
			errText = st.Err.Error()
		}
		l.p("  ├─ File:     %s", st.Path)
		l.p("  ├─ Format:   %s, %d Hz, %d channels", desc.ContentType, desc.SampleRate, desc.ChannelCount)
		l.p("  ├─ Duration: %s", formatDuration(st.Duration))
		l.p("  ├─ Size:     %s", units.HumanSize(float64(st.Bytes)))
		l.p("  ├─ Dropped:  %d packets", st.Dropped)
		l.p("  └─ Error:    %s", errText)
		l.p("")
	}

	all := slices.Clone(r.finished)
	for _, rec := range r.state.recorders {
		if rec != nil {
			all = append(all, rec.Parts()...) // rotated out while recording
		}
	}
	var finished []recStats
	for _, st := range all {
		if st.Bytes > 0 || st.Err != nil {
			finished = append(finished, st)
		}
	}
	if len(finished) > 0 {
		l.p("Finished files:")
		for _, st := range finished {
			if st.Err != nil {
				l.p("  %s (%s) - Error: %v", st.Path, units.HumanSize(float64(st.Bytes)), st.Err)
			} else {
				l.p("  %s (%s)", st.Path, units.HumanSize(float64(st.Bytes)))
			}
		}
	}

	return l
}

func formatDuration(d time.Duration) string {
	return fmt.Sprintf("%02d:%02d.%03d", int(d.Minutes()), int(d.Seconds())%60, d.Milliseconds()%1000)
}
