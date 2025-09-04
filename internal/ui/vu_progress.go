package ui

import (
	"math"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/lucasb-eyer/go-colorful"
)

// VUProgress represents a VU meter progress bar component
type VUProgress struct {
	width           int
	backgroundStyle lipgloss.Style
}

// NewVUProgress creates a new VU progress bar
func NewVUProgress(width int, backgroundStyle lipgloss.Style) *VUProgress {
	return &VUProgress{
		width:           width,
		backgroundStyle: backgroundStyle,
	}
}

// SetWidth sets the width of the progress bar
func (p *VUProgress) SetWidth(width int) {
	p.width = width
}

// ViewAs renders the progress bar at the given percentage (0.0 to 1.0)
func (p *VUProgress) ViewAs(percent float64) string {
	if p.width <= 0 {
		return ""
	}

	if percent < 0 {
		percent = 0
	}
	if percent > 1 {
		percent = 1
	}

	// Calculate how much of the bar to fill
	filledWidth := int(math.Round(percent * float64(p.width)))

	var a []string

	for i := range filledWidth {
		// Calculate color for this position
		pos := float64(i) / float64(p.width-1)
		color := p.getGradientColor(pos)
		style := lipgloss.NewStyle().Foreground(color)
		a = append(a, style.Render("█"))
	}

	s := strings.Repeat("░", p.width-filledWidth)
	a = append(a, p.backgroundStyle.Render(s))

	return strings.Join(a, "")
}

// getGradientColor returns the color at the given position in the gradient (0.0 to 1.0)
func (p *VUProgress) getGradientColor(pos float64) lipgloss.Color {
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
