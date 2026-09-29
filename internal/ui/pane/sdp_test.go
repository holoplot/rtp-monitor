package pane

import (
	"strings"
	"testing"
)

func TestSDPPane(t *testing.T) {
	p := NewSDP(testStream())
	view := strings.Join(rows(p.View(40, 5)), "\n")
	if !strings.Contains(view, "s=Alpha") || strings.Contains(view, "\r") {
		t.Fatalf("view:\n%q", view)
	}
	if p.CopyText() != string(testStream().SDP) {
		t.Errorf("CopyText = %q", p.CopyText())
	}

	p.SetStream(nil)
	if !p.Offline() || p.Title() != "SDP · Alpha · offline" {
		t.Errorf("offline=%v title=%q", p.Offline(), p.Title())
	}

	s2 := testStream()
	s2.SDP = []byte("v=0\ns=Alpha v2\n")
	p.SetStream(s2)
	if !strings.Contains(strings.Join(rows(p.View(40, 5)), "\n"), "s=Alpha v2") {
		t.Error("view does not follow the new snapshot")
	}
}
