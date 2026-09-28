package pane

import (
	"fmt"
	"net"

	tea "charm.land/bubbletea/v2"
	"github.com/holoplot/rtp-monitor/internal/stream"
)

// key builds a key press for a Bubble Tea key string such as "up" or "R".
func key(s string) tea.KeyPressMsg {
	special := map[string]rune{
		"up": tea.KeyUp, "down": tea.KeyDown, "left": tea.KeyLeft, "right": tea.KeyRight,
		"pgup": tea.KeyPgUp, "pgdown": tea.KeyPgDown, "home": tea.KeyHome, "end": tea.KeyEnd,
		"enter": tea.KeyEnter, "esc": tea.KeyEscape, "tab": tea.KeyTab, "backspace": tea.KeyBackspace,
	}
	if code, ok := special[s]; ok {
		return tea.KeyPressMsg{Code: code}
	}
	return tea.KeyPressMsg{Code: []rune(s)[0], Text: s}
}

// testStream returns a detached L24 stereo stream; receivers on it fail with stream.ErrDetached.
func testStream() *stream.Stream {
	return &stream.Stream{
		ID:  "alpha-id",
		SDP: []byte("v=0\r\ns=Alpha\r\n"),
		Description: stream.StreamDescription{
			Name: "Alpha", SampleRate: 48000, ChannelCount: 2, ContentType: stream.ContentTypePCM24,
			Sources: []stream.StreamSource{{DestinationAddress: net.ParseIP("239.1.1.1"), DestinationPort: 5004}},
		},
	}
}

func numbered(n int) []string {
	lines := make([]string, n)
	for i := range lines {
		lines[i] = fmt.Sprintf("l%d", i+1)
	}
	return lines
}
