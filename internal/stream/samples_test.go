package stream

import (
	"errors"
	"math"
	"reflect"
	"testing"

	"github.com/pion/rtp/v2"
)

func TestExtractSamples(t *testing.T) {
	tests := []struct {
		name    string
		desc    StreamDescription
		payload []byte
		want    []SampleFrame
		wantErr error
	}{
		{
			name:    "L24 stereo",
			desc:    StreamDescription{ContentType: ContentTypePCM24, ChannelCount: 2},
			payload: []byte{0x12, 0x34, 0x56, 0xFF, 0xFF, 0xFF, 0x7F, 0xFF, 0xFF, 0x80, 0x00, 0x00},
			want:    []SampleFrame{{0x12345600, -256}, {0x7FFFFF00, math.MinInt32}},
		},
		{
			name:    "L16 mono",
			desc:    StreamDescription{ContentType: ContentTypePCM16, ChannelCount: 1},
			payload: []byte{0x12, 0x34, 0xFF, 0xFE},
			want:    []SampleFrame{{0x12340000}, {-131072}},
		},
		{
			name:    "undefined encoding",
			desc:    StreamDescription{ContentType: ContentTypeUndefined, ChannelCount: 2},
			payload: []byte{0, 0, 0},
			wantErr: ErrUnsupportedContentType,
		},
		{
			name:    "zero channels",
			desc:    StreamDescription{ContentType: ContentTypePCM24},
			payload: []byte{0, 0, 0},
			wantErr: ErrUnsupportedContentType,
		},
	}

	for _, tt := range tests {
		got, err := tt.desc.ExtractSamples(&rtp.Packet{Payload: tt.payload})
		if !errors.Is(err, tt.wantErr) {
			t.Errorf("%s: err = %v, want %v", tt.name, err, tt.wantErr)
			continue
		}
		if tt.wantErr == nil && !reflect.DeepEqual(got, tt.want) {
			t.Errorf("%s: got %v, want %v", tt.name, got, tt.want)
		}
		if tt.desc.CanDecode() != (tt.wantErr == nil) {
			t.Errorf("%s: CanDecode() = %v", tt.name, tt.desc.CanDecode())
		}
	}
}

func TestDetachedStreamReceivers(t *testing.T) {
	s := &Stream{}
	if _, err := s.NewRTPReceiver(nil); !errors.Is(err, ErrDetached) {
		t.Errorf("NewRTPReceiver: err = %v, want ErrDetached", err)
	}
	if _, err := s.NewRTCPReceiver(nil); !errors.Is(err, ErrDetached) {
		t.Errorf("NewRTCPReceiver: err = %v, want ErrDetached", err)
	}
}
