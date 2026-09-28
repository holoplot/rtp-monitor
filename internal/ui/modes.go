package ui

import (
	"image/color"

	"github.com/holoplot/rtp-monitor/internal/layout"
	"github.com/holoplot/rtp-monitor/internal/theme"
)

// Mode is the input mode, as in zellij.
type Mode int

const (
	ModeNormal Mode = iota
	ModePane
	ModeResize
	ModeMove
)

func (m Mode) Label() string {
	switch m {
	case ModePane:
		return "PANE"
	case ModeResize:
		return "RESIZE"
	case ModeMove:
		return "MOVE"
	default:
		return "NORMAL"
	}
}

func (m Mode) Color() color.Color {
	switch m {
	case ModePane:
		return theme.UI.ModePane
	case ModeResize:
		return theme.UI.ModeResize
	case ModeMove:
		return theme.UI.ModeMove
	default:
		return theme.UI.ModeNormal
	}
}

func (m Mode) Help() string {
	switch m {
	case ModePane:
		return "←↓↑→/hjkl focus · x close · f zoom · t flip · c copy · r resize · m move · esc back"
	case ModeResize:
		return "←↓↑→/hjkl grow · HJKL shrink · +/- all sides · esc back"
	case ModeMove:
		return "←↓↑→/hjkl swap · tab swap next · esc back"
	default:
		return "tab focus · ctrl+w pane/resize/move · c copy · ctrl+c quit · shift+drag selects text"
	}
}

// modeSwitch returns the mode that key switches to from cur, if key is a
// mode key there. Ctrl+w is the only leader, because zellij and tmux bind
// most other Ctrl and Alt keys by default: it enters Pane mode from Normal
// mode and returns to Normal mode from any other mode. In Pane mode, r
// enters Resize mode and m enters Move mode.
func modeSwitch(cur Mode, key string) (Mode, bool) {
	switch key {
	case "ctrl+w":
		if cur == ModeNormal {
			return ModePane, true
		}
		return ModeNormal, true
	case "r":
		if cur == ModePane {
			return ModeResize, true
		}
	case "m":
		if cur == ModePane {
			return ModeMove, true
		}
	case "esc", "enter":
		if cur != ModeNormal {
			return ModeNormal, true
		}
	}
	return cur, false
}

func direction(key string) (layout.Direction, bool) {
	switch key {
	case "left", "h":
		return layout.Left, true
	case "right", "l":
		return layout.Right, true
	case "up", "k":
		return layout.Up, true
	case "down", "j":
		return layout.Down, true
	}
	return 0, false
}

// shrinkDirection maps HJKL to the side whose edge moves inward.
func shrinkDirection(key string) (layout.Direction, bool) {
	switch key {
	case "H":
		return layout.Left, true
	case "L":
		return layout.Right, true
	case "K":
		return layout.Up, true
	case "J":
		return layout.Down, true
	}
	return 0, false
}
