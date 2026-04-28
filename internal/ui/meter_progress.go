package ui

import (
	"math"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/lucasb-eyer/go-colorful"
)

// MeterProgress represents a meter progress bar component
type MeterProgress struct {
	width           int
	backgroundStyle lipgloss.Style
}

// NewMeterProgress creates a new meter progress bar
func NewMeterProgress(width int, backgroundStyle lipgloss.Style) *MeterProgress {
	return &MeterProgress{
		width:           width,
		backgroundStyle: backgroundStyle,
	}
}

// SetWidth sets the width of the progress bar
func (p *MeterProgress) SetWidth(width int) {
	p.width = width
}

// ViewAs renders the bar filled to peakPercent, with an RMS marker at rmsPercent (0.0 to 1.0).
func (p *MeterProgress) ViewAs(peakPercent, rmsPercent float64) string {
	if p.width <= 0 {
		return ""
	}

	clamp := func(v float64) float64 {
		if v < 0 {
			return 0
		}
		if v > 1 {
			return 1
		}
		return v
	}
	peakPercent = clamp(peakPercent)
	rmsPercent = clamp(rmsPercent)

	filledWidth := int(math.Round(peakPercent * float64(p.width)))
	rmsPos := int(math.Round(rmsPercent * float64(p.width-1)))
	if rmsPos >= filledWidth {
		rmsPos = filledWidth - 1
	}

	var a []string

	for i := range filledWidth {
		pos := float64(i) / float64(p.width-1)
		if i == rmsPos {
			style := lipgloss.NewStyle().Foreground(lipgloss.Color("#FFFFFF"))
			a = append(a, style.Render("█"))
		} else {
			color := p.getGradientColor(pos)
			style := lipgloss.NewStyle().Foreground(color)
			a = append(a, style.Render("█"))
		}
	}

	s := strings.Repeat("░", p.width-filledWidth)
	a = append(a, p.backgroundStyle.Render(s))

	return strings.Join(a, "")
}

// getGradientColor returns the color at the given position in the gradient (0.0 to 1.0)
func (p *MeterProgress) getGradientColor(pos float64) lipgloss.Color {
	if pos < 0 {
		pos = 0
	}
	if pos > 1 {
		pos = 1
	}

	// Simple gradient from green to red across the entire width
	green, _ := colorful.Hex("#00FF00")
	red, _ := colorful.Hex("#FF0000")

	// Blend from green to red based on position
	c := green.BlendLuv(red, pos)

	return lipgloss.Color(c.Hex())
}
