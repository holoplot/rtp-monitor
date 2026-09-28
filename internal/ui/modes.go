package ui

import (
	"image/color"

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
