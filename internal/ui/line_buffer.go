package ui

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
)

type lineBuffer struct {
	l []string
}

func newLineBuffer(headerStyle lipgloss.Style) *lineBuffer {
	return &lineBuffer{}
}

func (l *lineBuffer) p(fmtString string, args ...any) {
	s := fmt.Sprintf(fmtString, args...)
	l.l = append(l.l, s)
}

func (l *lineBuffer) lines() []string {
	return l.l
}
