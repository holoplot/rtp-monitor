package pane

import (
	"image/color"
	"math"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/holoplot/rtp-monitor/internal/theme"
	"github.com/lucasb-eyer/go-colorful"
)

var (
	meterGreen, _ = colorful.Hex("#00FF00")
	meterRed, _   = colorful.Hex("#FF0000")
	meterMarker   = lipgloss.Color("#FFFFFF")
)

// renderMeterBar renders a bar width cells wide, filled with a green-to-red
// gradient up to peak (0..1), with a white RMS marker at rms (0..1).
func renderMeterBar(peak, rms float64, width int) string {
	if width <= 0 {
		return ""
	}

	peak, rms = clamp01(peak), clamp01(rms)
	filled := int(math.Round(peak * float64(width)))
	rmsPos := min(int(math.Round(rms*float64(width-1))), filled-1)

	var b strings.Builder
	for i := range filled {
		c := gradient(float64(i) / float64(max(width-1, 1)))
		if i == rmsPos {
			c = meterMarker
		}
		b.WriteString(lipgloss.NewStyle().Foreground(c).Render("█"))
	}
	b.WriteString(lipgloss.NewStyle().Foreground(theme.UI.Muted).Render(strings.Repeat("░", width-filled)))

	return b.String()
}

func gradient(pos float64) color.Color {
	return lipgloss.Color(meterGreen.BlendLuv(meterRed, clamp01(pos)).Hex())
}

func clamp01(v float64) float64 { return max(0, min(v, 1)) }
