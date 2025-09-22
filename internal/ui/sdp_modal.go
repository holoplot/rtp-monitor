package ui

import (
	"strings"
	"time"

	"github.com/holoplot/rtp-monitor/internal/stream"
)

// SDPModalContent implements ModalContentProvider for raw SDP display
type SDPModalContent struct {
	stream *stream.Stream
}

// NewSDPModalContent creates a new SDP modal content provider
func NewSDPModalContent(stream *stream.Stream) *SDPModalContent {
	return &SDPModalContent{
		stream: stream,
	}
}

// Init initializes the content provider with dimensions
func (s *SDPModalContent) Init(width, height int) {
	// No initialization needed for SDP modal - stream is set in constructor
}

// Close closes the modal content provider
func (s *SDPModalContent) Close() {
	// No cleanup needed for SDP modal
}

// Content returns the SDP content lines to be displayed
func (s *SDPModalContent) Content() []string {
	var lines []string

	sdpLines := strings.SplitSeq(string(s.stream.SDP), "\n")
	for line := range sdpLines {
		lines = append(lines, SanitizeASCII(line))
	}

	return lines
}

// Title returns the modal title
func (s *SDPModalContent) Title() string {
	return "SDP Content"
}

// UpdateInterval returns how often the modal content should be updated (0 means no updates)
func (s *SDPModalContent) UpdateInterval() time.Duration {
	return 0 // SDP content doesn't need periodic updates
}

// AutoScroll returns whether the modal should automatically scroll to the bottom
func (s *SDPModalContent) AutoScroll() bool {
	return false
}

// Update is called periodically if UpdateInterval > 0
func (s *SDPModalContent) Update() {
	// No updates needed for static SDP content
}
