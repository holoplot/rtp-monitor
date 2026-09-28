package ui

import (
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/holoplot/rtp-monitor/internal/ui/pane"
)

const wheelLines = 3

const dragHint = "drop on a pane to swap · release elsewhere to cancel"

// handleMouse focuses panes on click, drags borders and sends wheel events to
// the pane under the pointer.
func (a *App) handleMouse(msg tea.MouseMsg) tea.Cmd {
	m := msg.Mouse()

	switch msg.(type) {
	case tea.MouseClickMsg:
		a.resetDrag() // a release may have been lost, e.g. outside the window
		if m.Button != tea.MouseLeft {
			return nil
		}
		if b, ok := a.tree.BorderAt(m.X, m.Y); ok {
			a.drag = &b
			return nil
		}
		if id, ok := a.tree.TitleAt(m.X, m.Y); ok {
			a.paneDrag = &id
			a.tree.SetFocus(id)
			a.status, a.statusUntil = dragHint, time.Time{}
			return nil
		}
		id, ok := a.tree.PaneAt(m.X, m.Y)
		if !ok {
			return nil
		}
		a.tree.SetFocus(id)
		r := a.tree.Rects()[id]
		return a.panes[id].Update(pane.ClickMsg{X: m.X - r.X - 1, Y: m.Y - r.Y - 1})

	case tea.MouseMotionMsg:
		if a.drag != nil {
			a.tree.DragBorder(*a.drag, m.X, m.Y)
		}
		if a.paneDrag != nil {
			a.dropTarget = nil
			if id, ok := a.tree.PaneAt(m.X, m.Y); ok && id != *a.paneDrag {
				a.dropTarget = &id
			}
		}

	case tea.MouseReleaseMsg:
		a.drag = nil
		if a.paneDrag != nil {
			if id, ok := a.tree.PaneAt(m.X, m.Y); ok && id != *a.paneDrag {
				a.tree.SwapPanes(*a.paneDrag, id)
			}
			a.resetDrag()
		}

	case tea.MouseWheelMsg:
		id, ok := a.tree.PaneAt(m.X, m.Y)
		if !ok {
			return nil
		}
		switch m.Button {
		case tea.MouseWheelUp:
			return a.panes[id].Update(pane.ScrollMsg{Lines: -wheelLines})
		case tea.MouseWheelDown:
			return a.panes[id].Update(pane.ScrollMsg{Lines: wheelLines})
		}
	}

	return nil
}

// resetDrag ends any border or pane drag and removes the drag hint.
func (a *App) resetDrag() {
	a.drag, a.paneDrag, a.dropTarget = nil, nil, nil
	if a.status == dragHint {
		a.status, a.statusUntil = "", time.Time{}
	}
}
