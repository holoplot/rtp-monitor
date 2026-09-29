package ui

import (
	"fmt"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/holoplot/rtp-monitor/internal/clipboard"
	"github.com/holoplot/rtp-monitor/internal/layout"
	"github.com/holoplot/rtp-monitor/internal/ptp"
	"github.com/holoplot/rtp-monitor/internal/stream"
	"github.com/holoplot/rtp-monitor/internal/ui/pane"
)

const (
	tickInterval   = 50 * time.Millisecond
	flashDuration  = time.Second
	statusDuration = 3 * time.Second
)

// StreamSource provides the streams known at startup.
type StreamSource interface {
	GetAllStreams() []*stream.Stream
}

// Factory creates stream panes; tests replace it.
type Factory func(kind pane.Kind, s *stream.Stream, ctx pane.Context) pane.Pane

// Options configures an App.
type Options struct {
	Streams   StreamSource // optional
	PTP       *ptp.Monitor // nil when PTP monitoring is unavailable
	WavFolder string
	FPGA      bool               // FPGA RX panes can be opened
	Factory   Factory            // defaults to pane.New
	Clipboard func(string) error // defaults to clipboard.WriteString
}

// UpdateStreamsMsg carries the current stream snapshots from the manager.
type UpdateStreamsMsg struct {
	Streams []*stream.Stream
	// Initial marks the startup snapshot from Options.Streams. It is dropped
	// if a live update from the manager arrived first, as it may be older.
	Initial bool
}

// App is the root model of the tiling pane UI.
type App struct {
	opts   Options
	tree   *layout.Tree
	panes  map[layout.PaneID]pane.Pane
	list   *pane.StreamList // shown in the picker
	picker bool             // the stream picker is open
	nextID layout.PaneID

	mode          Mode
	width, height int
	status        string
	statusUntil   time.Time // zero: status stays until replaced
	flash         map[layout.PaneID]time.Time
	confirmQuit   bool
	drag          *layout.Border
	paneDrag      *layout.PaneID // source pane while dragging its title bar
	dropTarget    *layout.PaneID // pane under the pointer during a pane drag
	gotUpdate     bool           // a live UpdateStreamsMsg arrived
	quitting      bool
	closed        bool
	now           func() time.Time
}

type tickMsg time.Time

func tick() tea.Cmd {
	return tea.Tick(tickInterval, func(t time.Time) tea.Msg { return tickMsg(t) })
}

func NewApp(opts Options) *App {
	if opts.Factory == nil {
		opts.Factory = pane.New
	}
	if opts.Clipboard == nil {
		opts.Clipboard = clipboard.WriteString
	}

	return &App{
		opts:   opts,
		tree:   layout.NewEmpty(),
		list:   pane.NewStreamList(opts.FPGA),
		picker: true,
		nextID: 1,
		panes:  map[layout.PaneID]pane.Pane{},
		flash:  map[layout.PaneID]time.Time{},
		now:    time.Now,
	}
}

func (a *App) Init() tea.Cmd {
	cmds := []tea.Cmd{tick()}
	if src := a.opts.Streams; src != nil {
		cmds = append(cmds, func() tea.Msg { return UpdateStreamsMsg{Streams: src.GetAllStreams(), Initial: true} })
	}
	return tea.Batch(cmds...)
}

func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if a.quitting {
		return a, nil // panes are closed
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.width, a.height = msg.Width, msg.Height
		a.tree.SetArea(layout.Rect{W: msg.Width, H: max(msg.Height-1, 0)})
		return a, nil

	case tickMsg:
		cmds := []tea.Cmd{tick()}
		for _, p := range a.panes {
			cmds = append(cmds, p.Update(pane.TickMsg(msg)))
		}
		if !a.statusUntil.IsZero() && time.Time(msg).After(a.statusUntil) {
			a.status, a.statusUntil = "", time.Time{}
		}
		return a, tea.Batch(cmds...)

	case UpdateStreamsMsg:
		if msg.Initial && a.gotUpdate {
			return a, nil
		}
		a.gotUpdate = a.gotUpdate || !msg.Initial
		a.setStreams(msg.Streams)
		return a, nil

	case pane.OpenMsg:
		a.open(msg.Kind, msg.Stream)
		return a, nil

	case tea.MouseMsg:
		return a, a.handleMouse(msg)

	case tea.KeyPressMsg:
		return a.handleKey(msg)
	}

	return a, nil
}

func (a *App) View() tea.View {
	v := tea.NewView(a.render())
	v.AltScreen = true
	v.MouseMode = tea.MouseModeCellMotion
	return v
}

func (a *App) handleKey(k tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	key := k.String()
	if a.confirmQuit {
		a.confirmQuit = false
		a.status, a.statusUntil = "", time.Time{}
		if key == "y" {
			return a.quit()
		}
		return a, nil
	}
	if key == "ctrl+c" {
		return a.requestQuit()
	}
	if a.picker {
		return a.pickerKey(k)
	}
	if next, ok := modeSwitch(a.mode, key); ok {
		a.mode = next
		return a, nil
	}

	switch a.mode {
	case ModePane:
		a.paneModeKey(key)
	case ModeResize:
		a.resizeModeKey(key)
	case ModeMove:
		a.moveModeKey(key)
	default:
		return a.normalKey(k)
	}
	return a, nil
}

func (a *App) paneModeKey(key string) {
	if d, ok := direction(key); ok {
		a.tree.MoveFocus(d)
		return
	}
	switch key {
	case "x":
		a.closePane(a.tree.Focus())
	case "f":
		a.tree.ToggleZoom()
	case "t":
		if err := a.tree.Flip(); err != nil {
			a.setStatus(err.Error())
		}
	case "c":
		a.copyFocused()
	}
}

// pickerKey handles keys while the stream picker is open. It is modal: keys
// do not reach the panes, and mode keys are ignored.
func (a *App) pickerKey(k tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch k.String() {
	case "esc", "n":
		a.closePicker()
		return a, nil
	case "q":
		return a.requestQuit()
	case "c":
		a.copy(a.list.CopyText())
		return a, nil
	case "ctrl+w":
		return a, nil
	}
	return a, a.list.Update(k)
}

// openPicker shows the stream picker in Normal mode.
func (a *App) openPicker() {
	a.picker, a.mode = true, ModeNormal
	a.resetDrag()
}

// closePicker hides the stream picker unless no pane is open: the screen is
// never blank.
func (a *App) closePicker() {
	if len(a.panes) > 0 {
		a.picker = false
	}
}

func (a *App) resizeModeKey(key string) {
	var err error
	if d, ok := direction(key); ok {
		err = a.tree.Resize(d, true)
	} else if d, ok := shrinkDirection(key); ok {
		err = a.tree.Resize(d, false)
	} else {
		switch key {
		case "+", "=":
			err = a.tree.ResizeAll(true)
		case "-":
			err = a.tree.ResizeAll(false)
		}
	}
	if err != nil {
		a.setStatus(err.Error())
	}
}

func (a *App) moveModeKey(key string) {
	if d, ok := direction(key); ok {
		a.tree.Swap(d)
		return
	}
	if key == "tab" {
		a.tree.SwapNext()
	}
}

func (a *App) closePane(id layout.PaneID) {
	p, ok := a.panes[id]
	if !ok {
		return
	}
	if err := a.tree.Close(id); err != nil {
		a.setStatus(err.Error())
		return
	}
	p.Close()
	delete(a.panes, id)
	delete(a.flash, id)
	a.updatePaneCounts()
	if len(a.panes) == 0 {
		a.openPicker()
	}
}

func (a *App) normalKey(k tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	focused := a.tree.Focus()
	switch k.String() {
	case "tab":
		a.tree.CycleFocus(true)
		return a, nil
	case "shift+tab":
		a.tree.CycleFocus(false)
		return a, nil
	case "c":
		a.copyFocused()
		return a, nil
	case "n":
		a.openPicker()
		return a, nil
	case "q":
		return a.requestQuit()
	}
	return a, a.panes[focused].Update(k)
}

func (a *App) setStreams(streams []*stream.Stream) {
	a.list.SetStreams(streams)

	byID := make(map[string]*stream.Stream, len(streams))
	for _, s := range streams {
		byID[s.ID] = s
	}
	for _, p := range a.panes {
		if id := p.StreamID(); id != "" {
			p.SetStream(byID[id])
		}
	}
}

// open places a new pane of kind for s. Focus stays where it is; the new
// pane's border flashes briefly.
func (a *App) open(kind pane.Kind, s *stream.Stream) {
	if s == nil {
		return
	}
	id := a.nextID
	if err := a.tree.AutoPlace(id); err != nil {
		a.setStatus(err.Error())
		return
	}
	a.nextID++
	a.panes[id] = a.opts.Factory(kind, s, pane.Context{PTP: a.opts.PTP, WavFolder: a.opts.WavFolder})
	a.flash[id] = a.now().Add(flashDuration)
	a.updatePaneCounts()
}

func (a *App) updatePaneCounts() {
	counts := map[string]int{}
	for _, p := range a.panes {
		if id := p.StreamID(); id != "" {
			counts[id]++
		}
	}
	a.list.SetPaneCounts(counts)
}

func (a *App) copyFocused() {
	a.copy(a.panes[a.tree.Focus()].CopyText())
}

func (a *App) copy(text string) {
	if text == "" {
		a.setStatus("nothing to copy")
		return
	}
	if err := a.opts.Clipboard(text); err != nil {
		a.setStatus("copy failed: " + err.Error())
		return
	}
	a.setStatus("copied to clipboard")
}

func (a *App) setStatus(s string) {
	a.status = s
	a.statusUntil = a.now().Add(statusDuration)
}

func (a *App) requestQuit() (tea.Model, tea.Cmd) {
	n := 0
	for _, p := range a.panes {
		if p.Kind() == pane.KindRecord {
			n++
		}
	}
	if n == 0 {
		return a.quit()
	}

	noun := "recordings"
	if n == 1 {
		noun = "recording"
	}
	a.confirmQuit = true
	a.status, a.statusUntil = fmt.Sprintf("%d %s active, quit? y/n", n, noun), time.Time{}
	return a, nil
}

func (a *App) quit() (tea.Model, tea.Cmd) {
	a.quitting = true
	a.Close()
	return a, tea.Quit
}

// Close closes every pane, finalizing recordings. It is idempotent and must be
// called once the program has stopped, whatever made it stop (quit key,
// signal, panic), so no WAV file is left unfinalized.
func (a *App) Close() {
	if a.closed {
		return
	}
	a.closed = true
	for _, p := range a.panes {
		p.Close()
	}
}
