package stream

import "testing"

// buildSDP returns a minimal single-source AES67 SDP with the given rtpmap value.
func buildSDP(rtpmap string, sourceFilter bool) []byte {
	s := "v=0\n" +
		"o=- 3735928559 3735928559 IN IP4 192.168.1.100\n" +
		"s=Test Audio Stream\n" +
		"c=IN IP4 239.239.239.239/10\n" +
		"t=0 0\n" +
		"a=clock-domain:0\n" +
		"m=audio 5004 RTP/AVP 96\n" +
		"a=rtpmap:" + rtpmap + "\n" +
		"a=framecount:48\n" +
		"a=mediaclk:direct=0\n" +
		"a=sync-time:1234\n"
	if sourceFilter {
		s += "a=source-filter: incl IN IP4 239.239.239.239 192.168.1.100\n"
	}
	return []byte(s)
}

func TestParseSDPContentType(t *testing.T) {
	tests := []struct {
		rtpmap   string
		want     ContentType
		rate, ch uint32
	}{
		{"96 L24/48000/8", ContentTypePCM24, 48000, 8},
		{"96 L16/44100/2", ContentTypePCM16, 44100, 2},
		{"96 AM824/48000/2", ContentTypeUndefined, 48000, 2},
	}

	for _, tt := range tests {
		sd, _, err := ParseSDP(buildSDP(tt.rtpmap, true))
		if err != nil {
			t.Fatalf("%s: %v", tt.rtpmap, err)
		}
		if sd.ContentType != tt.want || sd.SampleRate != tt.rate || sd.ChannelCount != tt.ch {
			t.Errorf("%s: got %s %d Hz %d ch, want %s %d Hz %d ch", tt.rtpmap,
				sd.ContentType, sd.SampleRate, sd.ChannelCount, tt.want, tt.rate, tt.ch)
		}
	}
}
