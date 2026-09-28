package stream

import (
	"fmt"
	"strings"
	"sync"
	"testing"
)

func renamed(sdp []byte, name, sessionID string) []byte {
	s := strings.Replace(string(sdp), "s=Test Audio Stream", "s="+name, 1)
	s = strings.Replace(s, "3735928559 3735928559", sessionID+" "+sessionID, 1)
	return []byte(s)
}

func TestSnapshotsAreIsolated(t *testing.T) {
	m := NewManager(nil)
	sdp := buildSDP("96 L24/48000/8", true)

	added, err := m.AddStreamFromSDP(sdp, DiscoveryMethodManual, "a.sdp")
	if err != nil {
		t.Fatal(err)
	}
	listed := m.GetAllStreams()[0]

	if _, err := m.AddStreamFromSDP(sdp, DiscoveryMethodSAP, "eth0"); err != nil {
		t.Fatal(err)
	}

	if n := len(added.Discoveries); n != 1 {
		t.Errorf("returned stream changed: %d discoveries, want 1", n)
	}
	if n := len(listed.Discoveries); n != 1 {
		t.Errorf("listed stream changed: %d discoveries, want 1", n)
	}

	current, ok := m.GetStream(added.ID)
	if !ok {
		t.Fatal("stream missing")
	}
	if n := len(current.Discoveries); n != 2 {
		t.Errorf("current stream: %d discoveries, want 2", n)
	}

	current.Discoveries[0].Source = "mutated"
	again, _ := m.GetStream(added.ID)
	if again.Discoveries[0].Source == "mutated" {
		t.Error("mutating a snapshot changed the manager's copy")
	}
}

func TestGetAllStreamsSorted(t *testing.T) {
	m := NewManager(nil)
	sdp := buildSDP("96 L24/48000/8", true)

	for _, s := range []struct{ name, id string }{{"Bravo", "1"}, {"Alpha", "2"}} {
		if _, err := m.AddStreamFromSDP(renamed(sdp, s.name, s.id), DiscoveryMethodManual, s.name); err != nil {
			t.Fatal(err)
		}
	}

	streams := m.GetAllStreams()
	if len(streams) != 2 || streams[0].Name() != "Alpha" || streams[1].Name() != "Bravo" {
		t.Fatalf("unexpected order: %v", streams)
	}
}

// Run with -race: the callback and readers must not race with discovery updates.
func TestUpdateCallbackRace(t *testing.T) {
	m := NewManager(nil)
	sdp := buildSDP("96 L24/48000/8", true)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := range 200 {
			_, _ = m.AddStreamFromSDP(sdp, DiscoveryMethodSAP, fmt.Sprintf("eth%d", i%4))
		}
	}()

	m.OnUpdate(func(streams []*Stream) {
		for _, s := range streams {
			_ = s.DiscoveryLabel()
		}
	})
	for range 200 {
		for _, s := range m.GetAllStreams() {
			_ = s.DiscoveryLabel()
		}
	}
	wg.Wait()
}
