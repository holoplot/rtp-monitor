package ui

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
)

type lineBuffer struct {
	l           []string
	headerStyle lipgloss.Style
}

func newLineBuffer(headerStyle lipgloss.Style) *lineBuffer {
	return &lineBuffer{
		headerStyle: headerStyle,
	}
}

func (l *lineBuffer) p(fmtString string, args ...any) {
	s := fmt.Sprintf(fmtString, args...)
	l.l = append(l.l, s)
}

func (l *lineBuffer) h(s string) {
	l.l = append(l.l, l.headerStyle.Render(s))
}

func (l *lineBuffer) lines() []string {
	return l.l
}
