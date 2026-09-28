package pane

import (
	"fmt"
	"strings"
	"unicode"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/holoplot/rtp-monitor/internal/theme"
)

// fitLine truncates or pads s, which may contain ANSI sequences, to exactly w cells.
func fitLine(s string, w int) string {
	if w <= 0 {
		return ""
	}
	if ansi.StringWidth(s) > w {
		s = ansi.Truncate(s, w, "…")
	}
	return s + strings.Repeat(" ", w-ansi.StringWidth(s))
}

// Fit clips or pads s to exactly w×h cells.
func Fit(s string, w, h int) string {
	if w <= 0 || h <= 0 {
		return ""
	}
	lines := strings.Split(s, "\n")
	out := make([]string, h)
	for i := range out {
		line := ""
		if i < len(lines) {
			line = lines[i]
		}
		out[i] = fitLine(line, w)
	}
	return strings.Join(out, "\n")
}

// sanitize replaces tabs by four spaces, drops carriage returns and replaces
// other non-printable runes by spaces.
func sanitize(s string) string {
	var b strings.Builder
	for _, r := range s {
		switch {
		case r == '\t':
			b.WriteString("    ")
		case r == '\r':
		case unicode.IsPrint(r):
			b.WriteRune(r)
		default:
			b.WriteRune(' ')
		}
	}
	return b.String()
}

// lineBuffer collects formatted lines.
type lineBuffer []string

func (l *lineBuffer) p(format string, args ...any) {
	*l = append(*l, fmt.Sprintf(format, args...))
}

// scrollbar returns the scrollbar cell for row i of a view h rows high that
// shows total lines starting at offset.
func scrollbar(i, h, total, offset int) string {
	thumb := max(1, h*h/total)
	pos := 0
	if maxOffset := total - h; maxOffset > 0 {
		pos = offset * (h - thumb) / maxOffset
	}
	if i >= pos && i < pos+thumb {
		return lipgloss.NewStyle().Foreground(theme.UI.ScrollBarThumb).Render("█")
	}
	return lipgloss.NewStyle().Foreground(theme.UI.ScrollBar).Render("│")
}
