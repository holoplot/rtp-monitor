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
	c := a.borderColor(id, p)
	border := lipgloss.NewStyle().Foreground(c)
	title := lipgloss.NewStyle().Foreground(theme.UI.PaneTitle).Bold(id == a.tree.Focus())

	content := pane.Fit(p.View(r.W-2, r.H-2), r.W-2, r.H-2)
	body := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder(), false, true, true, true).
		BorderForeground(c).
		Width(r.W).
		Height(r.H - 1).
		Render(content)

	return titleBar(p.Title(), r.W, border, title) + "\n" + body
}

func (a *App) borderColor(id layout.PaneID, p pane.Pane) color.Color {
	switch {
	case a.now().Before(a.flash[id]):
		return theme.UI.PaneBorderFlash
	case id == a.tree.Focus():
		return theme.UI.PaneBorderFocused
	case p.Gone():
		return theme.UI.PaneBorderGone
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

	badge := lipgloss.NewStyle().Bold(true).Padding(0, 1).
		Foreground(theme.UI.Background).Background(a.mode.Color()).
		Render(a.mode.Label())

	help := a.mode.Help()
	if a.mode == ModeNormal && a.tree.Focus() == a.listID {
		help = a.listHelp() + " · " + help
	}
	if a.tree.Zoomed() {
		help = "ZOOM · " + help
	}

	msg := ""
	if a.status != "" {
		msg = lipgloss.NewStyle().Foreground(theme.UI.Warning).Background(theme.UI.StatusBarBg).
			Render(" " + a.status + " ")
	}

	helpW := max(a.width-lipgloss.Width(badge)-lipgloss.Width(msg), 0)
	bar := lipgloss.NewStyle().Foreground(theme.UI.StatusBarFg).Background(theme.UI.StatusBarBg)

	return pane.Fit(badge+bar.Render(pane.Fit(" "+help, helpW, 1))+msg, a.width, 1)
}

func (a *App) listHelp() string {
	help := "m meters · R record · d details · s SDP · r RTCP"
	if a.opts.FPGA {
		help += " · f FPGA"
	}
	return help + " · q quit"
}
