package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/holoplot/rtp-monitor/internal/stream"
	"github.com/holoplot/rtp-monitor/internal/theme"
)

// TableModel represents the table component state
type TableModel struct {
	streams       []*stream.Stream
	selectedIndex int
	viewStart     int
	height        int
	width         int
	styles        TableStyles
}

// TableStyles holds the styling for the table
type TableStyles struct {
	Header      lipgloss.Style
	Border      lipgloss.Style
	Row         lipgloss.Style
	RowSelected lipgloss.Style
	ScrollBar   lipgloss.Style
	ScrollThumb lipgloss.Style
}

// NewTableModel creates a new table model
func NewTableModel() *TableModel {
	return &TableModel{
		streams:       []*stream.Stream{},
		selectedIndex: 0,
		viewStart:     0,
		height:        20,
		width:         80,
		styles:        createTableStyles(),
	}
}

// createTableStyles creates the default table styles using the current theme
func createTableStyles() TableStyles {
	return TableStyles{
		Header: lipgloss.NewStyle().
			Foreground(theme.Colors.TableHeader).
			Background(theme.Colors.Secondary).
			Bold(true).
			Padding(0, 0),
		Border: lipgloss.NewStyle().
			Foreground(theme.Colors.TableBorder),
		Row: lipgloss.NewStyle().
			Foreground(theme.Colors.TableRow).
			Background(theme.Colors.Background).
			Padding(0, 0),
		RowSelected: lipgloss.NewStyle().
			Foreground(theme.Colors.TableRowSelected).
			Background(theme.Colors.TableRowSelectedBg).
			Bold(true).
			Padding(0, 0),
		ScrollBar: lipgloss.NewStyle().
			Foreground(theme.Colors.ScrollBar),
		ScrollThumb: lipgloss.NewStyle().
			Foreground(theme.Colors.ScrollBarThumb).
			Background(theme.Colors.ScrollBarThumb),
	}
}

// SetStreams updates the streams displayed in the table
func (t *TableModel) SetStreams(streams []*stream.Stream) {
	t.streams = streams
	// Ensure selected index is valid
	if t.selectedIndex >= len(streams) {
		t.selectedIndex = len(streams) - 1
	}
	if t.selectedIndex < 0 {
		t.selectedIndex = 0
	}
	t.adjustView()
}

// SetSize sets the dimensions of the table
func (t *TableModel) SetSize(width, height int) {
	t.width = width
	t.height = height - 1 // Account for footer space
	t.adjustView()
}

// MoveUp moves the selection up
func (t *TableModel) MoveUp() {
	if t.selectedIndex > 0 {
		t.selectedIndex--
		t.adjustView()
	}
}

// MoveDown moves the selection down
func (t *TableModel) MoveDown() {
	if t.selectedIndex < len(t.streams)-1 {
		t.selectedIndex++
		t.adjustView()
	}
}

// GetSelected returns the currently selected stream
func (t *TableModel) GetSelected() *stream.Stream {
	if t.selectedIndex >= 0 && t.selectedIndex < len(t.streams) {
		return t.streams[t.selectedIndex]
	}
	return nil
}

// adjustView ensures the selected item is visible
func (t *TableModel) adjustView() {
	if len(t.streams) == 0 {
		return
	}

	visibleRows := t.height - 3 // Account for header and borders
	if visibleRows < 1 {
		visibleRows = 1
	}

	// Adjust view to keep selected item visible
	if t.selectedIndex < t.viewStart {
		t.viewStart = t.selectedIndex
	} else if t.selectedIndex >= t.viewStart+visibleRows {
		t.viewStart = t.selectedIndex - visibleRows + 1
	}

	// Ensure view doesn't go beyond bounds
	maxViewStart := len(t.streams) - visibleRows
	if maxViewStart < 0 {
		maxViewStart = 0
	}
	if t.viewStart > maxViewStart {
		t.viewStart = maxViewStart
	}
	if t.viewStart < 0 {
		t.viewStart = 0
	}
}

// Render renders the table as a string
func (t *TableModel) Render() string {
	if len(t.streams) == 0 {
		return t.renderEmpty()
	}

	var b strings.Builder

	// Render header
	b.WriteString(t.renderHeader())
	b.WriteString("\n")

	// Render rows
	visibleRows := t.height - 3
	if visibleRows < 1 {
		visibleRows = 1
	}

	endIndex := t.viewStart + visibleRows
	if endIndex > len(t.streams) {
		endIndex = len(t.streams)
	}

	for i := t.viewStart; i < endIndex; i++ {
		b.WriteString(t.renderRow(i))
		if i < endIndex-1 {
			b.WriteString("\n")
		}
	}

	// Add scrollbar if needed
	if len(t.streams) > visibleRows {
		result := t.addScrollbar(b.String(), visibleRows)
		return result
	}

	return b.String()
}

// renderEmpty renders an empty table message
func (t *TableModel) renderEmpty() string {
	message := "No RTP streams detected"

	return t.styles.Row.
		Width(t.width - 1).
		Height(1).
		Align(lipgloss.Center).
		Render(message)
}

// calculateColumnWidths calculates optimal column widths for the table
func (t *TableModel) calculateColumnWidths() []int {
	availableWidth := t.width - 1 // Reserve 1 space for scrollbar
	if availableWidth < 60 {
		availableWidth = 60 // Minimum usable width
	}

	// Distribute width proportionally to accommodate primary/secondary IPs
	// ID: 5%, Name: 25%, Address: 45%, Codec: 20%, Method: 10%
	idWidth := (availableWidth * 10) / 100
	nameWidth := (availableWidth * 30) / 100
	addressWidth := (availableWidth * 25) / 100
	codecWidth := (availableWidth * 20) / 100
	methodWidth := (availableWidth * 10) / 100
	sourceWidth := availableWidth - nameWidth - addressWidth - codecWidth

	// Ensure minimum widths
	if idWidth < 8 {
		idWidth = 8
	}

	if nameWidth < 15 {
		nameWidth = 15
	}
	if addressWidth < 25 {
		addressWidth = 25
	}
	if codecWidth < 10 {
		codecWidth = 10
	}
	if methodWidth < 6 {
		methodWidth = 6
	}
	if sourceWidth < 6 {
		sourceWidth = 6
	}

	return []int{idWidth, nameWidth, addressWidth, codecWidth, methodWidth, sourceWidth}
}

// renderHeader renders the table header
func (t *TableModel) renderHeader() string {
	headers := []string{"ID", "Name", "Address", "Codec", "Method", "Source"}
	widths := t.calculateColumnWidths()

	var headerParts []string
	for i, header := range headers {
		if i < len(widths) {
			cellContent := truncateString(header, widths[i])
			headerParts = append(headerParts, t.styles.Header.
				Width(widths[i]).
				Height(1).
				Align(lipgloss.Left).
				Render(cellContent))
		}
	}

	headerLine := lipgloss.JoinHorizontal(lipgloss.Top, headerParts...)
	// Ensure the header uses full width
	if lipgloss.Width(headerLine) < t.width-1 {
		headerLine += strings.Repeat(" ", t.width-1-lipgloss.Width(headerLine))
	}
	return headerLine
}

// renderRow renders a single table row
func (t *TableModel) renderRow(index int) string {
	stream := t.streams[index]
	widths := t.calculateColumnWidths()

	// Prepare row data
	rowData := []string{
		truncateString(stream.IDHash(), widths[0]),
		truncateString(stream.Description.Name, widths[1]),
		truncateString(stream.Address(), widths[2]),
		truncateString(stream.CodecInfo(), widths[3]),
		truncateString(stream.DiscoveryMethod.String(), widths[4]),
		truncateString(stream.DiscoverySource, widths[5]),
	}

	// Choose style based on selection and alternating rows
	var style lipgloss.Style
	if index == t.selectedIndex {
		style = t.styles.RowSelected
	} else {
		style = t.styles.Row
	}

	var rowParts []string
	for i, data := range rowData {
		cellStyle := style.Width(widths[i]).Height(1).Align(lipgloss.Left)
		rowParts = append(rowParts, cellStyle.Render(data))
	}

	rowLine := lipgloss.JoinHorizontal(lipgloss.Top, rowParts...)
	// Ensure the row uses full width
	if lipgloss.Width(rowLine) < t.width-1 {
		rowLine += strings.Repeat(" ", t.width-1-lipgloss.Width(rowLine))
	}
	return rowLine
}

// addScrollbar adds a scrollbar to the rendered content
func (t *TableModel) addScrollbar(content string, visibleRows int) string {
	lines := strings.Split(content, "\n")
	if len(lines) == 0 {
		return content
	}

	totalStreams := len(t.streams)
	scrollbarHeight := visibleRows
	thumbSize := max(1, (visibleRows*scrollbarHeight)/totalStreams)
	thumbPos := (t.viewStart * (scrollbarHeight - thumbSize)) / max(1, totalStreams-visibleRows)

	// Create scrollbar
	scrollbar := make([]string, len(lines))
	for i := range scrollbar {
		if i == 0 {
			scrollbar[i] = " " // Header line
		} else {
			lineIndex := i - 1
			if lineIndex >= thumbPos && lineIndex < thumbPos+thumbSize {
				scrollbar[i] = t.styles.ScrollThumb.Render("█")
			} else {
				scrollbar[i] = t.styles.ScrollBar.Render("│")
			}
		}
	}

	// Combine content with scrollbar
	var result []string
	for i, line := range lines {
		if i < len(scrollbar) {
			// The line should already be full width from rendering, just add scrollbar
			result = append(result, line+scrollbar[i])
		} else {
			// Pad line to full width and add scrollbar space
			lineWidth := lipgloss.Width(line)
			if lineWidth < t.width-1 {
				line += strings.Repeat(" ", t.width-1-lineWidth)
			}
			result = append(result, line+" ")
		}
	}

	return strings.Join(result, "\n")
}

// truncateString truncates a string to fit within the specified width
func truncateString(s string, width int) string {
	if width <= 0 {
		return ""
	}
	if len(s) <= width {
		// Pad to exact width for consistent table formatting
		return s + strings.Repeat(" ", width-len(s))
	}
	if width <= 3 {
		return s[:width]
	}
	return s[:width-3] + "..."
}

// max returns the maximum of two integers
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// RefreshStyles updates the table styles
func (t *TableModel) RefreshStyles() {
	t.styles = createTableStyles()
}
