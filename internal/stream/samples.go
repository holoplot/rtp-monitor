package stream

import (
	"errors"

	"github.com/pion/rtp/v2"
)

type (
	// Sample is a PCM sample left-aligned in a signed 32-bit integer.
	Sample      int32
	SampleFrame []Sample
)

var (
	ErrUnsupportedContentType = errors.New("unsupported content type")
	ErrDetached               = errors.New("stream is not attached to a manager")
)

// bytesPerSample returns the sample size of the description's encoding.
func (d *StreamDescription) bytesPerSample() (uint32, error) {
	if d.ChannelCount == 0 {
		return 0, ErrUnsupportedContentType
	}

	switch d.ContentType {
	case ContentTypePCM16:
		return 2, nil
	case ContentTypePCM24:
		return 3, nil
	default:
		return 0, ErrUnsupportedContentType
	}
}

// CanDecode reports whether ExtractSamples supports the description's encoding.
func (d *StreamDescription) CanDecode() bool {
	_, err := d.bytesPerSample()
	return err == nil
}

// ExtractSamples decodes the PCM payload of packet into sample frames.
func (d *StreamDescription) ExtractSamples(packet *rtp.Packet) ([]SampleFrame, error) {
	bytesPerSample, err := d.bytesPerSample()
	if err != nil {
		return nil, err
	}

	channels := d.ChannelCount
	numFrames := uint32(len(packet.Payload)) / (bytesPerSample * channels)
	frames := make([]SampleFrame, 0, numFrames)

	var i uint32

	for range numFrames {
		frame := make(SampleFrame, channels)

		for ch := range channels {
			value := uint32(packet.Payload[i])<<24 | uint32(packet.Payload[i+1])<<16
			if bytesPerSample == 3 {
				value |= uint32(packet.Payload[i+2]) << 8
			}

			frame[ch] = Sample(value)
			i += bytesPerSample
		}

		frames = append(frames, frame)
	}

	return frames, nil
}
