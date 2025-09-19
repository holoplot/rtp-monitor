package ui

import (
	"strings"
	"time"
	"unicode"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/holoplot/rtp-monitor/internal/stream"
	"github.com/holoplot/rtp-monitor/internal/theme"
)

// ModalContentProvider defines the interface for modal content providers
type ModalContentProvider interface {
	// Init initializes the content provider with dimensions
	Init(width, height int)

	// Content returns the content lines to be displayed
	Content() []string

	// Title returns the modal title
	Title() string

	// UpdateInterval returns how often the modal content should be updated (0 means no updates)
	UpdateInterval() time.Duration

	// AutoScroll returns whether the modal should automatically scroll to the bottom
	AutoScroll() bool

	// Update is called periodically if UpdateInterval > 0
	Update()

	// Close closes the modal
	Close()
}

// sanitizeASCII removes or replaces non-printable characters from a string
func SanitizeASCII(s string) string {
	var result strings.Builder
	for _, r := range s {
		if unicode.IsPrint(r) || r == '\t' {
			result.WriteRune(r)
		} else if r == '\r' {
			// Skip carriage returns
			continue
		} else {
			// Replace other non-printable characters with a space
			result.WriteRune(' ')
		}
	}
	return result.String()
}

// ModalModel represents the generic modal component
type ModalModel struct {
	provider     ModalContentProvider
	stream       *stream.Stream
	width        int
	height       int
	scrollOffset int
	visible      bool
	styles       ModalStyles
	lastUpdate   time.Time
}

// ModalStyles holds the styling for the modal
type ModalStyles struct {
	Overlay     lipgloss.Style
	Container   lipgloss.Style
	Header      lipgloss.Style
	Content     lipgloss.Style
	ScrollBar   lipgloss.Style
	ScrollThumb lipgloss.Style
}

// NewModalModel creates a new modal model
func NewModalModel() *ModalModel {
	return &ModalModel{
		visible: false,
		styles:  createModalStyles(),
	}
}

// createModalStyles creates the modal styles using the current theme
func createModalStyles() ModalStyles {
	return ModalStyles{
		Overlay: lipgloss.NewStyle().
			Background(theme.Colors.Background).
			Foreground(theme.Colors.Foreground).
			Width(0).
			Height(0),
		Container: lipgloss.NewStyle().
			Background(theme.Colors.Background).
			Foreground(theme.Colors.Foreground).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(theme.Colors.Primary).
			Padding(1),
		Header: lipgloss.NewStyle().
			Foreground(theme.Colors.Primary).
			Bold(true).
			Align(lipgloss.Center).
			MarginBottom(1),
		Content: lipgloss.NewStyle().
			Background(theme.Colors.Background).
			Foreground(theme.Colors.Foreground),
		ScrollBar: lipgloss.NewStyle().
			Foreground(theme.Colors.ScrollBar),
		ScrollThumb: lipgloss.NewStyle().
			Foreground(theme.Colors.ScrollBarThumb).
			Background(theme.Colors.ScrollBarThumb),
	}
}

// Show displays the modal with the given content provider and data
func (m *ModalModel) Show(stream *stream.Stream, provider ModalContentProvider, width, height int) {
	m.stream = stream
	m.provider = provider
	m.width = width
	m.height = height
	m.scrollOffset = 0
	m.visible = true
	m.lastUpdate = time.Now()

	if m.provider != nil {
		m.provider.Init(width, height)
	}
}

// Hide closes the modal
func (m *ModalModel) Hide() {
	if m.provider != nil {
		m.provider.Close()
	}
	m.visible = false
	m.provider = nil
	m.scrollOffset = 0
}

// IsVisible returns whether the modal is currently visible
func (m *ModalModel) IsVisible() bool {
	return m.visible
}

// ScrollUp scrolls the content up
func (m *ModalModel) ScrollUp() {
	if m.scrollOffset > 0 {
		m.scrollOffset--
	}
}

// ScrollDown scrolls the content down
func (m *ModalModel) ScrollDown() {
	maxScroll := m.getMaxScroll()

	if m.scrollOffset < maxScroll {
		m.scrollOffset++
	}
}

// ScrollPageUp scrolls up by one page
func (m *ModalModel) ScrollPageUp() {
	contentHeight := m.height - 8 // Account for modal padding, header, and borders
	if contentHeight < 1 {
		contentHeight = 1
	}

	m.scrollOffset -= contentHeight
	if m.scrollOffset < 0 {
		m.scrollOffset = 0
	}
}

// ScrollPageDown scrolls down by one page
func (m *ModalModel) ScrollPageDown() {
	contentHeight := m.height - 8 // Account for modal padding, header, and borders
	if contentHeight < 1 {
		contentHeight = 1
	}

	maxScroll := m.getMaxScroll()
	m.scrollOffset += contentHeight
	if m.scrollOffset > maxScroll {
		m.scrollOffset = maxScroll
	}
}

// ScrollToTop scrolls to the beginning of content
func (m *ModalModel) ScrollToTop() {
	m.scrollOffset = 0
}

// ScrollToBottom scrolls to the end of content
func (m *ModalModel) ScrollToBottom() {
	m.scrollOffset = m.getMaxScroll()
}

// Update updates the modal content if needed
// UpdateContent is called periodically to refresh content
func (m *ModalModel) UpdateContent() {
	if !m.visible || m.provider == nil {
		return
	}

	updateInterval := m.provider.UpdateInterval()
	if updateInterval > 0 && time.Since(m.lastUpdate) >= updateInterval {
		m.provider.Update()
		if m.provider.AutoScroll() {
			m.ScrollToBottom()
		}
		m.lastUpdate = time.Now()
	}
}

// getModalDimensions returns consistent modal and content dimensions
func (m *ModalModel) getModalDimensions() (modalWidth, modalHeight, contentWidth, contentHeight int) {
	// Calculate modal dimensions (80% of screen, but at least 60x20)
	modalWidth = (m.width * 80) / 100
	if modalWidth < 60 {
		modalWidth = 60
	}
	if modalWidth > m.width-4 {
		modalWidth = m.width - 4
	}

	modalHeight = (m.height * 80) / 100
	if modalHeight < 20 {
		modalHeight = 20
	}
	if modalHeight > m.height-4 {
		modalHeight = m.height - 4
	}

	// Content area dimensions (account for borders and padding)
	contentWidth = modalWidth - 4
	contentHeight = modalHeight - 4

	return modalWidth, modalHeight, contentWidth, contentHeight
}

// getScrollableContentDimensions returns dimensions for scrollable content calculations
func (m *ModalModel) getScrollableContentDimensions() (availableWidth, availableHeight int) {
	_, _, contentWidth, contentHeight := m.getModalDimensions()

	// Account for title line and scrollbar
	availableWidth = contentWidth - 2   // Account for scrollbar
	availableHeight = contentHeight - 1 // Account for title line

	if availableHeight < 1 {
		availableHeight = 1
	}

	return availableWidth, availableHeight
}

// getMaxScroll returns the maximum scroll offset
func (m *ModalModel) getMaxScroll() int {
	if m.provider == nil {
		return 0
	}

	_, availableHeight := m.getScrollableContentDimensions()
	contentLines := m.provider.Content()

	// Calculate actual rendered lines accounting for wrapping
	totalRenderedLines := len(contentLines)

	maxScroll := totalRenderedLines - availableHeight
	if maxScroll < 0 {
		maxScroll = 0
	}
	return maxScroll
}

// Render renders the modal
func (m *ModalModel) Render() string {
	if !m.visible || m.provider == nil {
		return ""
	}

	// Use shared dimension calculation
	modalWidth, modalHeight, contentWidth, _ := m.getModalDimensions()

	// Get content and calculate scrolling
	availableWidth, availableHeight := m.getScrollableContentDimensions()
	contentLines := m.provider.Content()
	totalLines := len(contentLines)

	// Truncate long lines to fit available width, accounting for ANSI sequences
	for i, line := range contentLines {
		visualWidth := ansi.StringWidth(line)
		if visualWidth > availableWidth {
			contentLines[i] = ansi.Truncate(line, availableWidth, "…")
		}
	}

	// Get visible lines based on scroll position
	visibleLines := m.getVisibleLines(contentLines, availableHeight)

	// Add scrollbar if needed
	needsScrollbar := totalLines > availableHeight
	if needsScrollbar {
		visibleLines = m.addScrollbarToVisibleLines(visibleLines, availableWidth, availableHeight, totalLines)
	}

	// Create title line (centered)
	title := m.provider.Title() + " | " + m.stream.Name()
	titleLine := m.createCenteredTitle(title, contentWidth)

	// Join content and apply content styling to ensure proper foreground color
	contentText := strings.Join(visibleLines, "\n")
	// content := m.styles.Content.Render(contentText)

	// Combine title and content
	modalContent := lipgloss.JoinVertical(lipgloss.Left, titleLine, contentText)

	// Container - return just the styled container like ChatGPT's example
	return m.styles.Container.
		Width(modalWidth).
		Height(modalHeight).
		Render(modalContent)
}

// Init implements tea.Model interface
func (m *ModalModel) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model interface
func (m *ModalModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Update content periodically
	m.UpdateContent()

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			m.ScrollUp()
		case "down", "j":
			m.ScrollDown()
		case "page_up":
			m.ScrollPageUp()
		case "page_down":
			m.ScrollPageDown()
		case "home":
			m.ScrollToTop()
		case "end":
			m.ScrollToBottom()
		}
	}
	return m, nil
}

// View implements tea.Model interface
func (m *ModalModel) View() string {
	return m.Render()
}

// createCenteredTitle creates a centered title line for the modal
func (m *ModalModel) createCenteredTitle(title string, width int) string {
	// Center the title
	totalPadding := width - len(title)
	leftPadding := totalPadding / 2
	rightPadding := totalPadding - leftPadding

	centeredTitle := strings.Repeat(" ", leftPadding) + title + strings.Repeat(" ", rightPadding)

	// Apply title styling (centered, bold, primary color)
	return m.styles.Header.
		Width(width).
		Render(centeredTitle)
}

// getVisibleLines returns the lines that should be visible in the content area
func (m *ModalModel) getVisibleLines(contentLines []string, maxLines int) []string {
	if m.provider == nil {
		return []string{}
	}

	// Handle edge case where maxLines is negative or zero
	if maxLines <= 0 {
		return []string{}
	}

	start := m.scrollOffset
	end := start + maxLines

	if start >= len(contentLines) {
		start = len(contentLines) - 1
		if start < 0 {
			start = 0
		}
	}
	if start < 0 {
		start = 0
	}

	if end > len(contentLines) {
		end = len(contentLines)
	}

	return contentLines[start:end]
}

// addScrollbarToVisibleLines adds a scrollbar to the visible lines only
func (m *ModalModel) addScrollbarToVisibleLines(visibleLines []string, availableWidth, visibleHeight, totalLines int) []string {
	if totalLines <= visibleHeight || visibleHeight <= 0 {
		return visibleLines
	}

	// Calculate scrollbar properties
	thumbSize := (visibleHeight * visibleHeight) / totalLines
	if thumbSize < 1 {
		thumbSize = 1
	}
	if thumbSize > visibleHeight {
		thumbSize = visibleHeight
	}

	maxThumbPos := visibleHeight - thumbSize
	if maxThumbPos < 0 {
		maxThumbPos = 0
	}

	// Calculate thumb position based on scroll offset
	var thumbPos int
	if totalLines > visibleHeight && maxThumbPos >= 0 {
		maxScroll := totalLines - visibleHeight
		if maxScroll > 0 {
			thumbPos = (m.scrollOffset * maxThumbPos) / maxScroll
		}

		// Ensure thumb is visible at all positions
		if thumbPos < 0 {
			thumbPos = 0
		}
		if thumbPos > maxThumbPos {
			thumbPos = maxThumbPos
		}

		// Ensure thumb fits within visible area
		if thumbPos+thumbSize > visibleHeight {
			thumbPos = visibleHeight - thumbSize
			if thumbPos < 0 {
				thumbPos = 0
			}
		}
	}

	// Create scrollbar for visible lines only
	var result []string
	for i, line := range visibleLines {
		if i >= visibleHeight {
			break
		}

		scrollChar := m.styles.ScrollBar.Render("│")
		if i >= thumbPos && i < thumbPos+thumbSize && thumbSize > 0 {
			scrollChar = m.styles.ScrollThumb.Render("█")
		}

		// Pad line to fixed width and add scrollbar at right edge
		// Use ansi.StringWidth to get visual width, not byte/rune count
		visualWidth := ansi.StringWidth(line)
		if visualWidth > availableWidth {
			line = ansi.Truncate(line, availableWidth, "…")
			visualWidth = availableWidth
		}

		padding := availableWidth - visualWidth
		if padding < 0 {
			padding = 0
		}

		result = append(result, line+strings.Repeat(" ", padding)+scrollChar)
	}

	return result
}
