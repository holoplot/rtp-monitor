package pane

import "testing"

func TestNewCreatesEveryKind(t *testing.T) {
	ctx := Context{WavFolder: t.TempDir()}
	for _, kind := range []Kind{KindDetails, KindMeters, KindSDP, KindRTCP, KindRecord, KindFPGA} {
		p := New(kind, testStream(), ctx)
		if p.Kind() != kind || p.StreamID() != "alpha-id" {
			t.Errorf("New(%v): kind %v, stream %q", kind, p.Kind(), p.StreamID())
		}
		_ = p.View(40, 10)
		p.Close()
	}

	defer func() {
		if recover() == nil {
			t.Error("New(KindStreamList) did not panic")
		}
	}()
	New(KindStreamList, testStream(), ctx)
}
