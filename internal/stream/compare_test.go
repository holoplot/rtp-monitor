package stream

import (
	"net"
	"slices"
	"testing"
)

func TestCompareDescriptions(t *testing.T) {
	base := StreamDescription{
		Name: "A", SampleRate: 48000, ChannelCount: 2, ContentType: ContentTypePCM24,
		Sources: []StreamSource{{DestinationAddress: net.ParseIP("239.1.1.1"), DestinationPort: 5004}},
	}
	with := func(f func(d *StreamDescription)) StreamDescription {
		d := base
		d.Sources = slices.Clone(base.Sources)
		f(&d)
		return d
	}

	tests := []struct {
		name string
		new  StreamDescription
		want DescriptionChange
	}{
		{"identical", with(func(d *StreamDescription) {}), DescriptionUnchanged},
		{"renamed", with(func(d *StreamDescription) { d.Name = "B" }), DescriptionUnchanged},
		{"port", with(func(d *StreamDescription) { d.Sources[0].DestinationPort = 5006 }), DescriptionTransportChanged},
		{"address", with(func(d *StreamDescription) { d.Sources[0].DestinationAddress = net.ParseIP("239.1.1.2") }), DescriptionTransportChanged},
		{"sample rate", with(func(d *StreamDescription) { d.SampleRate = 96000 }), DescriptionFormatChanged},
		{"channels", with(func(d *StreamDescription) { d.ChannelCount = 8 }), DescriptionFormatChanged},
		{"encoding", with(func(d *StreamDescription) { d.ContentType = ContentTypePCM16 }), DescriptionFormatChanged},
		{"source added", with(func(d *StreamDescription) { d.Sources = append(d.Sources, d.Sources[0]) }), DescriptionFormatChanged},
		{"format and transport", with(func(d *StreamDescription) {
			d.ChannelCount = 8
			d.Sources[0].DestinationPort = 5006
		}), DescriptionFormatChanged},
	}

	for _, tt := range tests {
		if got := CompareDescriptions(base, tt.new); got != tt.want {
			t.Errorf("%s: got %d, want %d", tt.name, got, tt.want)
		}
	}
}
