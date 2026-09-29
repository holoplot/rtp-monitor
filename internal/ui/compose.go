package ui

import (
	"image/color"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/holoplot/rtp-monitor/internal/layout"
	"github.com/holoplot/rtp-monitor/internal/theme"
	"github.com/holoplot/rtp-monitor/internal/ui/pane"
)

func (a *App) render() string {
	if a.quitting {
		return ""
	}
	return a.renderPanes() + "\n" + a.renderStatusBar()
}

func (a *App) renderPanes() string {
	area := a.tree.Area()
	if area.W < layout.MinWidth || area.H < layout.MinHeight {
		msg := lipgloss.Place(max(area.W, 1), max(area.H, 1), lipgloss.Center, lipgloss.Center, "terminal too small")
		return pane.Fit(msg, area.W, area.H)
	}

	var layers []*lipgloss.Layer
	for id, r := range a.tree.Rects() {
		layers = append(layers, lipgloss.NewLayer(a.renderPane(id, r)).X(r.X).Y(r.Y))
	}
	if a.picker {
		r := a.pickerRect()
		layers = append(layers, lipgloss.NewLayer(a.renderPicker(r)).X(r.X).Y(r.Y).Z(1))
	}

	canvas := lipgloss.NewCanvas(area.W, area.H)
	canvas.Compose(lipgloss.NewCompositor(layers...))
	return pane.Fit(canvas.Render(), area.W, area.H)
}

// renderPane draws pane id with its border and title into exactly r.W×r.H cells.
func (a *App) renderPane(id layout.PaneID, r layout.Rect) string {
	if r.W < 3 || r.H < 3 {
		return pane.Fit("", r.W, r.H)
	}

	p := a.panes[id]
	title := lipgloss.NewStyle().Foreground(theme.UI.PaneTitle).Bold(id == a.tree.Focus())

	content := pane.Fit(p.View(r.W-2, r.H-2), r.W-2, r.H-2)
	if p.Offline() {
		// Grey out panes whose stream vanished: drop their colours.
		title = title.Foreground(theme.UI.Muted)
		content = lipgloss.NewStyle().Foreground(theme.UI.Muted).Render(ansi.Strip(content))
	}
	return box(p.Title(), content, r, a.borderColor(id), title)
}

// pickerRect is where the stream picker floats: centred, the full width
// less a margin, as tall as the list needs.
func (a *App) pickerRect() layout.Rect {
	const margin = 2
	area := a.tree.Area()
	w := max(area.W-2*margin, min(area.W, layout.MinWidth))
	h := min(a.list.Height()+2, area.H)
	return layout.Rect{X: (area.W - w) / 2, Y: (area.H - h) / 2, W: w, H: h}
}

func (a *App) renderPicker(r layout.Rect) string {
	if r.W < 3 || r.H < 3 {
		return pane.Fit("", r.W, r.H)
	}
	content := pane.Fit(a.list.View(r.W-2, r.H-2), r.W-2, r.H-2)
	title := lipgloss.NewStyle().Foreground(theme.UI.PaneTitle).Bold(true)
	return box(a.list.Title(), content, r, theme.UI.PaneBorderFocused, title)
}

// box draws content (r.W-2 × r.H-2 cells) with a rounded border in colour c
// and title inset in the top border.
func box(title, content string, r layout.Rect, c color.Color, titleStyle lipgloss.Style) string {
	body := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder(), false, true, true, true).
		BorderForeground(c).
		Width(r.W).
		Height(r.H - 1).
		Render(content)

	return titleBar(title, r.W, lipgloss.NewStyle().Foreground(c), titleStyle) + "\n" + body
}

func (a *App) borderColor(id layout.PaneID) color.Color {
	switch {
	case a.dropTarget != nil && *a.dropTarget == id:
		return theme.UI.PaneBorderFlash
	case a.now().Before(a.flash[id]):
		return theme.UI.PaneBorderFlash
	case id == a.tree.Focus():
		return theme.UI.PaneBorderFocused
	default:
		return theme.UI.PaneBorder
	}
}

// titleBar renders a pane's top border with the title inset: ╭─ Title ───╮
func titleBar(title string, w int, border, text lipgloss.Style) string {
	if w < 2 {
		return strings.Repeat(" ", max(w, 0))
	}
	inner := w - 2
	if inner < 5 {
		return border.Render("╭" + strings.Repeat("─", inner) + "╮")
	}
	label := " " + ansi.Truncate(title, inner-3, "…") + " "
	fill := inner - 1 - ansi.StringWidth(label)
	return border.Render("╭─") + text.Render(label) + border.Render(strings.Repeat("─", fill)+"╮")
}

func (a *App) renderStatusBar() string {
	if a.width <= 0 {
		return ""
	}

	label, bg, help := a.mode.Label(), a.mode.Color(), a.mode.Help()
	switch {
	case a.picker:
		label, bg, help = "STREAMS", theme.UI.ModePicker, a.pickerHelp()
	case a.tree.Zoomed():
		help = "ZOOM · " + help
	}
	badge := lipgloss.NewStyle().Bold(true).Padding(0, 1).
		Foreground(theme.UI.Background).Background(bg).
		Render(label)

	msg := ""
	if a.status != "" {
		msg = lipgloss.NewStyle().Foreground(theme.UI.Warning).Background(theme.UI.StatusBarBg).
			Render(" " + a.status + " ")
	}

	helpW := max(a.width-lipgloss.Width(badge)-lipgloss.Width(msg), 0)
	bar := lipgloss.NewStyle().Foreground(theme.UI.StatusBarFg).Background(theme.UI.StatusBarBg)

	return pane.Fit(badge+bar.Render(pane.Fit(" "+help, helpW, 1))+msg, a.width, 1)
}

func (a *App) pickerHelp() string {
	help := "m meters · R record · d details · s SDP · r RTCP"
	if a.opts.FPGA {
		help += " · f FPGA"
	}
	help += " · c copy SDP"
	if len(a.panes) > 0 {
		help += " · esc close"
	}
	return help + " · q quit"
}
