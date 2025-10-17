package ui

import (
	"fmt"
	"runtime"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/holoplot/rtp-monitor/internal/clipboard"
	"github.com/holoplot/rtp-monitor/internal/ptp"
	"github.com/holoplot/rtp-monitor/internal/stream"
	"github.com/holoplot/rtp-monitor/internal/theme"
	"github.com/holoplot/rtp-monitor/internal/version"
	overlay "github.com/rmhubbert/bubbletea-overlay"
)

// BackgroundModel represents just the background view for overlay
type BackgroundModel struct {
	parent *Model
}

// Init implements tea.Model interface for BackgroundModel
func (b *BackgroundModel) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model interface for BackgroundModel
func (b *BackgroundModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	return b, nil
}

// View implements tea.Model interface for BackgroundModel
func (b *BackgroundModel) View() string {
	// Return the main view without modal overlay
	return b.parent.renderMainView()
}

// Model represents the main UI model
type Model struct {
	table         *TableModel
	modal         *ModalModel
	overlay       *overlay.Model
	background    *BackgroundModel
	streamManager *stream.Manager
	ptpMonitor    *ptp.Monitor
	width         int
	height        int
	lastUpdate    time.Time
	quitting      bool
	wavFileFolder string
}

// NewModel creates a new UI model
func NewModel(manager *stream.Manager, ptpMonitor *ptp.Monitor, wavFileFolder string) *Model {
	m := &Model{
		table:         NewTableModel(),
		modal:         NewModalModel(),
		streamManager: manager,
		ptpMonitor:    ptpMonitor,
		width:         80,
		height:        24,
		lastUpdate:    time.Now(),
		wavFileFolder: wavFileFolder,
	}
	m.background = &BackgroundModel{parent: m}
	return m
}

// Init initializes the UI model
func (m *Model) Init() tea.Cmd {
	return tea.Batch(
		func() tea.Cmd {
			return func() tea.Msg {
				return UpdateStreamsMsg{
					Streams: m.streamManager.GetAllStreams(),
				}
			}
		}(),
		m.modalTickCmd(),
	)
}

// Update handles UI updates
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.table.SetSize(msg.Width, msg.Height-2) // Leave space for header and footer

		// Pass window size to overlay if it exists
		if m.overlay != nil {
			m.overlay.Update(msg)
		}
		return m, nil

	case tea.KeyMsg:
		return m.handleKeypress(msg)

	case modalTickMsg:
		if !m.quitting && m.modal.IsVisible() {
			m.modal.UpdateContent()
			return m, tea.Batch(m.modalTickCmd())
		}
		return m, nil

	case UpdateStreamsMsg:
		m.table.SetStreams(msg.Streams)
		m.lastUpdate = time.Now()

		modalStreamMissing := func() bool {
			if !m.modal.IsVisible() {
				return false
			}

			for _, stream := range msg.Streams {
				if stream.ID == m.modal.stream.ID {
					return false
				}
			}

			return true
		}

		if modalStreamMissing() {
			m.modal.Hide()
		}

		return m, nil
	}

	return m, nil
}

func isLinux() bool {
	return runtime.GOOS == "linux"
}

// handleKeypress handles keyboard input
func (m *Model) handleKeypress(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Handle modal input first if any modal is visible
	if m.modal.IsVisible() {
		switch msg.String() {
		case "x", "q":
			m.modal.Hide()
			return m, nil
		case "up", "k":
			m.modal.ScrollUp()
			return m, nil
		case "down", "j":
			m.modal.ScrollDown()
			return m, nil
		case "pgup", "page_up":
			m.modal.ScrollPageUp()
			return m, nil
		case "pgdown", "page_down":
			m.modal.ScrollPageDown()
			return m, nil
		case "home":
			m.modal.ScrollToTop()
			return m, nil
		case "end":
			m.modal.ScrollToBottom()
			return m, nil
		case "c", "d", "f", "v", "r", "R", "s":
			// Allow modal switching - fall through to main keypress handling
		default:
			// For any other keys when modal is open, consume the input
			return m, nil
		}
	}

	// Handle main UI input
	switch msg.String() {
	case "q", "ctrl+c":
		m.quitting = true
		return m, tea.Quit

	case "up", "k":
		m.table.MoveUp()
		return m, nil

	case "down", "j":
		m.table.MoveDown()
		return m, nil

	case "c":
		// Show controls modal for selected stream
		selected := m.table.GetSelected()

		if m.modal.IsVisible() {
			s := strings.Join(m.modal.provider.Content(), "\n")
			clipboard.WriteString(s)
		} else if selected != nil {
			clipboard.Write(selected.SDP)
		}

		return m, nil

	case "d":
		// Show details modal for selected stream
		selected := m.table.GetSelected()
		if selected != nil {
			if m.modal.IsVisible() {
				m.modal.Hide()
			}
			detailsProvider := NewDetailsModalContent(selected, m.ptpMonitor)
			m.modal.Show(selected, detailsProvider, m.width, m.height)
			return m, m.modalTickCmd() // Start updates immediately
		}
		return m, nil

	case "f":
		if isLinux() {
			// Show FPGA RX modal for selected stream
			selected := m.table.GetSelected()
			if selected != nil {
				if m.modal.IsVisible() {
					m.modal.Hide()
				}
				fpgaRxProvider := NewFpgaRxModalContent(selected)
				m.modal.Show(selected, fpgaRxProvider, m.width, m.height)
				return m, m.modalTickCmd() // Start updates immediately
			}
		}
		return m, nil

	case "v":
		// Show VU meters modal for selected stream
		selected := m.table.GetSelected()
		if selected != nil {
			if m.modal.IsVisible() {
				m.modal.Hide()
			}
			vuProvider := NewVUModalContent(selected)
			m.modal.Show(selected, vuProvider, m.width, m.height)
			return m, m.modalTickCmd() // Start updates immediately
		}
		return m, nil

	case "s":
		// Show SDP modal for selected stream
		selected := m.table.GetSelected()
		if selected != nil {
			if m.modal.IsVisible() {
				m.modal.Hide()
			}
			sdpProvider := NewSDPModalContent(selected)
			m.modal.Show(selected, sdpProvider, m.width, m.height)
			return m, m.modalTickCmd() // Start updates immediately
		}
		return m, nil

	case "r":
		// Show SDP modal for selected stream
		selected := m.table.GetSelected()
		if selected != nil {
			if m.modal.IsVisible() {
				m.modal.Hide()
			}
			rtcpProvider := NewRTCPModalContent(selected)
			m.modal.Show(selected, rtcpProvider, m.width, m.height)
			return m, m.modalTickCmd() // Start updates immediately
		}
		return m, nil

	case "R":
		// Show recording modal for selected stream
		selected := m.table.GetSelected()
		if selected != nil {
			if m.modal.IsVisible() {
				m.modal.Hide()
			}
			recordProvider := NewRecordModalContent(selected, m.wavFileFolder)
			m.modal.Show(selected, recordProvider, m.width, m.height)
			return m, m.modalTickCmd() // Start updates immediately
		}
		return m, nil

	case "home":
		m.table.selectedIndex = 0
		m.table.adjustView()
		return m, nil

	case "end":
		if len(m.table.streams) > 0 {
			m.table.selectedIndex = len(m.table.streams) - 1
			m.table.adjustView()
		}
		return m, nil

	case "page_up":
		visibleRows := m.table.height - 3
		for i := 0; i < visibleRows && m.table.selectedIndex > 0; i++ {
			m.table.selectedIndex--
		}
		m.table.adjustView()
		return m, nil

	case "page_down":
		visibleRows := m.table.height - 3
		maxIndex := len(m.table.streams) - 1
		for i := 0; i < visibleRows && m.table.selectedIndex < maxIndex; i++ {
			m.table.selectedIndex++
		}
		m.table.adjustView()
		return m, nil
	}

	return m, nil
}

// View renders the UI
func (m *Model) View() string {
	if m.quitting {
		return "Goodbye!\n"
	}

	// If modal is visible, create overlay
	if m.modal.IsVisible() {
		if m.overlay == nil {
			// Create overlay with modal centered over main view
			m.overlay = overlay.New(
				m.modal,                        // foreground (modal)
				m.background,                   // background (main view)
				overlay.Center, overlay.Center, // center position
				0, 0, // no offset
			)
		}
		return m.overlay.View()
	} else {
		// Reset overlay when modal is hidden
		m.overlay = nil
	}

	return m.renderMainView()
}

// renderMainView renders the main view without modal overlay
func (m *Model) renderMainView() string {
	// Header
	header := m.renderHeader()

	// Table
	table := m.table.Render()

	// Footer
	footer := m.renderFooter()

	// Calculate available height for table and add padding to push footer to bottom
	headerHeight := lipgloss.Height(header)
	footerHeight := lipgloss.Height(footer)
	tableHeight := m.height - headerHeight - footerHeight - 1

	// Add padding to push footer to bottom
	padding := ""
	if tableHeight > lipgloss.Height(table) {
		paddingLines := tableHeight - lipgloss.Height(table)
		if paddingLines > 0 {
			padding = strings.Repeat("\n", paddingLines)
		}
	}

	// Combine all parts
	return lipgloss.JoinVertical(lipgloss.Left,
		header,
		table,
		padding,
		footer,
	)
}

func (m *Model) renderHeader() string {
	title := lipgloss.NewStyle().
		Foreground(theme.Colors.Primary).
		Bold(true).
		Render(fmt.Sprintf("RTP Stream Monitor %s", version.GetShortVersion()))

	streamCount := fmt.Sprintf("Streams: %d", len(m.table.streams))
	lastUpdate := fmt.Sprintf("Last Update: %s", m.lastUpdate.Format("15:04:05"))

	info := lipgloss.JoinHorizontal(lipgloss.Bottom,
		lipgloss.NewStyle().Foreground(theme.Colors.Secondary).Render(streamCount),
		lipgloss.NewStyle().Margin(0, 2).Render("│"),
		lipgloss.NewStyle().Foreground(theme.Colors.Secondary).Render(lastUpdate),
	)

	// Create a full-width header with title on left, info on right
	titleWidth := lipgloss.Width(title)
	infoWidth := lipgloss.Width(info)
	padding := m.width - titleWidth - infoWidth
	if padding < 0 {
		padding = 0
	}

	return lipgloss.JoinHorizontal(lipgloss.Bottom,
		title,
		strings.Repeat(" ", padding),
		info,
	)
}

// renderFooter renders the application footer with help text
func (m *Model) renderFooter() string {
	selected := m.table.GetSelected()
	var selectedInfo string
	if selected != nil {
		selectedInfo = fmt.Sprintf("Selected: %s (%s)", selected.Name(), selected.Address())
	} else {
		selectedInfo = "No stream selected"
	}

	help := []string{
		"↑/↓: Navigate",
		"c: Copy to clipboard",
		"d: Details",
	}

	if isLinux() {
		help = append(help, "f: FPGA RX")
	}

	help = append(help, []string{
		"r: RTCP",
		"R: Record wav",
		"s: SDP",
		"v: VU Meters",
		"q: Quit",
	}...)

	selectedStyle := lipgloss.NewStyle().
		Foreground(theme.Colors.Highlight).
		Render(selectedInfo)

	helpStyle := lipgloss.NewStyle().
		Foreground(theme.Colors.Secondary).
		Render(strings.Join(help, " │ "))

	return lipgloss.JoinVertical(lipgloss.Left,
		selectedStyle,
		helpStyle,
	)
}

// modalTickMsg represents a modal update tick message
type modalTickMsg time.Time

// modalTickCmd returns a command that sends modal tick messages
func (m *Model) modalTickCmd() tea.Cmd {
	return tea.Tick(50*time.Millisecond, func(t time.Time) tea.Msg {
		return modalTickMsg(t)
	})
}

// UpdateStreamsMsg contains updated stream data
type UpdateStreamsMsg struct {
	Streams []*stream.Stream
}
