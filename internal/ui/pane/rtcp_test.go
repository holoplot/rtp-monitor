package pane

import (
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/pion/rtcp"
)

func TestFormatRTCP(t *testing.T) {
	got := formatRTCP(&rtcp.SenderReport{SSRC: 0xabc, NTPTime: 5<<32 | 7, RTPTime: 9, PacketCount: 1, OctetCount: 2})
	want := []string{"SenderReport from abc, NTPTime 5.7, RTPTime 9, PacketCount 1, OctetCount 2"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %q, want %q", got, want)
	}
	if got := formatRTCP(&rtcp.ReceiverReport{}); got != nil {
		t.Errorf("empty receiver report: got %q, want nil", got)
	}
}

func TestRTCPLogFollowsAndTrims(t *testing.T) {
	p := NewRTCP(testStream())
	defer p.Close()

	now := time.Date(2026, 9, 28, 12, 0, 0, 0, time.UTC)
	for i := range maxLogLines + 10 {
		p.append(now, sender, &rtcp.SenderReport{SSRC: uint32(i)})
	}
	if len(p.log) != maxLogLines {
		t.Fatalf("log has %d lines, want %d", len(p.log), maxLogLines)
	}

	r := rows(p.View(120, 3))
	last := r[len(r)-1]
	if !strings.Contains(last, "10.0.0.1:5004") || !strings.Contains(last, "SenderReport from 1391") {
		t.Fatalf("last row %q is not the newest entry", last)
	}
}
