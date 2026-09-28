package pane

import (
	"net"
	"strings"
	"testing"
	"time"

	"github.com/pion/rtp/v2"
)

var sender = &net.UDPAddr{IP: net.ParseIP("10.0.0.1"), Port: 5004}

func TestDetailsShowsStatistics(t *testing.T) {
	d := NewDetails(testStream(), nil)
	defer d.Close()

	d.stats.onPacket(0, sender, &rtp.Packet{Header: rtp.Header{Timestamp: 42}})
	d.stats.onPacket(5, sender, &rtp.Packet{}) // unknown source index: ignored

	view := strings.Join(rows(d.View(100, 80)), "\n")
	for _, want := range []string{
		"Alpha", "Packets count:   1", "Last timestamp:  42", "10.0.0.1:5004",
		"Error creating stream receiver", "PTP unavailable", "239.1.1.1:5004",
	} {
		if !strings.Contains(view, want) {
			t.Errorf("view lacks %q", want)
		}
	}
}

func TestDetailsRates(t *testing.T) {
	d := NewDetails(testStream(), nil)
	defer d.Close()

	t0 := time.Now()
	d.lastRate = t0
	for range 10 {
		d.stats.onPacket(0, sender, &rtp.Packet{})
	}
	d.Update(TickMsg(t0.Add(500 * time.Millisecond))) // too early: no update
	d.Update(TickMsg(t0.Add(2 * time.Second)))

	if view := strings.Join(rows(d.View(100, 80)), "\n"); !strings.Contains(view, "Packets rate:    5.00/s") {
		t.Errorf("rate missing:\n%s", view)
	}
}

func TestDetailsRestartsOnTransportChange(t *testing.T) {
	d := NewDetails(testStream(), nil)
	defer d.Close()
	old := d.stats

	renamed := testStream()
	renamed.Description.Name = "Alpha 2"
	d.SetStream(renamed)
	if d.stats != old {
		t.Fatal("rename restarted the receiver")
	}

	moved := testStream()
	moved.Description.Sources[0].DestinationPort = 5006
	d.SetStream(moved)
	if d.stats == old {
		t.Fatal("transport change did not restart the receiver")
	}

	d.SetStream(nil)
	if !d.Gone() {
		t.Fatal("not gone")
	}
}
