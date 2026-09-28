package theme

import (
	"image/color"

	"charm.land/lipgloss/v2"
)

// Palette holds the semantic colours of the pane UI.
type Palette struct {
	Background, Foreground, Muted color.Color
	Accent                        color.Color // headings inside panes

	TableHeaderFg, TableHeaderBg color.Color
	RowSelectedFg, RowSelectedBg color.Color
	ScrollBar, ScrollBarThumb    color.Color

	PaneBorder, PaneBorderFocused   color.Color
	PaneBorderGone, PaneBorderFlash color.Color
	PaneTitle                       color.Color

	StatusBarFg, StatusBarBg                   color.Color
	ModeNormal, ModePane, ModeResize, ModeMove color.Color

	Active, Error, Warning color.Color
}

// baseColors is the small set of colours a Palette is derived from.
type baseColors struct {
	Bg, Fg, Muted, Surface                    string
	Blue, Green, Orange, Magenta, Red, Yellow string
}

var monokaiColors = baseColors{
	Bg: "#272822", Fg: "#F8F8F2", Muted: "#75715E", Surface: "#3E3D32",
	Blue: "#66D9EF", Green: "#A6E22E", Orange: "#FD971F", Magenta: "#AE81FF",
	Red: "#F92672", Yellow: "#E6DB74",
}

// UI is the palette used by the pane UI.
var UI = newPalette(monokaiColors)

func newPalette(b baseColors) Palette {
	c := lipgloss.Color
	return Palette{
		Background: c(b.Bg), Foreground: c(b.Fg), Muted: c(b.Muted), Accent: c(b.Blue),

		TableHeaderFg: c(b.Fg), TableHeaderBg: c(b.Magenta),
		RowSelectedFg: c(b.Bg), RowSelectedBg: c(b.Green),
		ScrollBar: c(b.Muted), ScrollBarThumb: c(b.Magenta),

		PaneBorder: c(b.Muted), PaneBorderFocused: c(b.Blue),
		PaneBorderGone: c(b.Red), PaneBorderFlash: c(b.Orange),
		PaneTitle: c(b.Fg),

		StatusBarFg: c(b.Fg), StatusBarBg: c(b.Surface),
		ModeNormal: c(b.Green), ModePane: c(b.Blue), ModeResize: c(b.Orange), ModeMove: c(b.Magenta),

		Active: c(b.Green), Error: c(b.Red), Warning: c(b.Yellow),
	}
}
