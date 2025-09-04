package theme

import "github.com/charmbracelet/lipgloss"

// Colors defines all colors used in the application (Monokai dark theme)
var Colors = struct {
	// Table colors
	TableHeader        lipgloss.Color
	TableBorder        lipgloss.Color
	TableRow           lipgloss.Color
	TableRowSelected   lipgloss.Color
	TableRowSelectedBg lipgloss.Color

	// UI element colors
	Background     lipgloss.Color
	Foreground     lipgloss.Color
	ScrollBar      lipgloss.Color
	ScrollBarThumb lipgloss.Color

	// Status colors
	StatusActive   lipgloss.Color
	StatusInactive lipgloss.Color
	StatusError    lipgloss.Color
	StatusWarning  lipgloss.Color

	// Accent colors
	Primary   lipgloss.Color
	Secondary lipgloss.Color
	Highlight lipgloss.Color
}{
	// Table colors - Monokai dark
	TableHeader:        lipgloss.Color("#F8F8F2"),
	TableBorder:        lipgloss.Color("#75715E"),
	TableRow:           lipgloss.Color("#F8F8F2"),
	TableRowSelected:   lipgloss.Color("#272822"),
	TableRowSelectedBg: lipgloss.Color("#A6E22E"),

	// UI element colors
	Background:     lipgloss.Color("#272822"),
	Foreground:     lipgloss.Color("#F8F8F2"),
	ScrollBar:      lipgloss.Color("#75715E"),
	ScrollBarThumb: lipgloss.Color("#AE81FF"),

	// Status colors
	StatusActive:   lipgloss.Color("#A6E22E"),
	StatusInactive: lipgloss.Color("#75715E"),
	StatusError:    lipgloss.Color("#F92672"),
	StatusWarning:  lipgloss.Color("#E6DB74"),

	// Accent colors
	Primary:   lipgloss.Color("#66D9EF"),
	Secondary: lipgloss.Color("#AE81FF"),
	Highlight: lipgloss.Color("#FD971F"),
}
