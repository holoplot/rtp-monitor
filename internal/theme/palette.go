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

	PaneBorder, PaneBorderFocused      color.Color
	PaneBorderOffline, PaneBorderFlash color.Color
	PaneTitle                          color.Color

	StatusBarFg, StatusBarBg                   color.Color
	ModeNormal, ModePane, ModeResize, ModeMove color.Color

	Active, Error, Warning color.Color
}

// baseColors is the small set of colours a Palette is derived from.
type baseColors struct {
	Bg, Fg, Muted, Surface                    string
	Blue, Green, Orange, Magenta, Red, Yellow string
}

var tokyoNightColors = baseColors{
	Bg: "#1A1B26", Fg: "#C0CAF5", Muted: "#565F89", Surface: "#292E42",
	Blue: "#7AA2F7", Green: "#9ECE6A", Orange: "#FF9E64", Magenta: "#BB9AF7",
	Red: "#F7768E", Yellow: "#E0AF68",
}

// UI is the palette used by the pane UI.
var UI = newPalette(tokyoNightColors)

func newPalette(b baseColors) Palette {
	c := lipgloss.Color
	return Palette{
		Background: c(b.Bg), Foreground: c(b.Fg), Muted: c(b.Muted), Accent: c(b.Blue),

		TableHeaderFg: c(b.Fg), TableHeaderBg: c(b.Magenta),
		RowSelectedFg: c(b.Bg), RowSelectedBg: c(b.Green),
		ScrollBar: c(b.Muted), ScrollBarThumb: c(b.Magenta),

		PaneBorder: c(b.Muted), PaneBorderFocused: c(b.Blue),
		PaneBorderOffline: c(b.Red), PaneBorderFlash: c(b.Orange),
		PaneTitle: c(b.Fg),

		StatusBarFg: c(b.Fg), StatusBarBg: c(b.Surface),
		ModeNormal: c(b.Green), ModePane: c(b.Blue), ModeResize: c(b.Orange), ModeMove: c(b.Magenta),

		Active: c(b.Green), Error: c(b.Red), Warning: c(b.Yellow),
	}
}
