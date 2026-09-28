package ui

import (
	tea "charm.land/bubbletea/v2"
	"github.com/holoplot/rtp-monitor/internal/ui/pane"
)

const wheelLines = 3

// handleMouse focuses panes on click, drags borders and sends wheel events to
// the pane under the pointer.
func (a *App) handleMouse(msg tea.MouseMsg) tea.Cmd {
	m := msg.Mouse()

	switch msg.(type) {
	case tea.MouseClickMsg:
		if m.Button != tea.MouseLeft {
			return nil
		}
		if b, ok := a.tree.BorderAt(m.X, m.Y); ok {
			a.drag = &b
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

	case tea.MouseReleaseMsg:
		a.drag = nil

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
