package stream

import (
	"errors"
	"net"
	"sync"

	"github.com/holoplot/go-multicast/pkg/multicast"
	"github.com/pion/rtcp"
	"github.com/pion/rtp/v2"
)

type RTPReceiverCallback func(int, net.Addr, *rtp.Packet)

type RTPReceiver struct {
	mutex     sync.Mutex
	stream    *Stream
	consumers []*multicast.Consumer
	rtpErrors map[int]int
}

func (s *Stream) NewRTPReceiver(cb RTPReceiverCallback) (*RTPReceiver, error) {
	r := &RTPReceiver{
		stream:    s,
		consumers: make([]*multicast.Consumer, 0),
		rtpErrors: make(map[int]int),
	}

	for i, source := range s.Description.Sources {
		addr := net.UDPAddr{
			IP:   source.DestinationAddress,
			Port: int(source.DestinationPort),
		}

		c, err := s.manager.multicastListener.AddConsumer(&addr, func(ifi *net.Interface, src net.Addr, payload []byte) {
			packet := &rtp.Packet{}
			if err := packet.Unmarshal(payload); err == nil {
				cb(i, src, packet)
			} else {
				r.mutex.Lock()
				defer r.mutex.Unlock()

				r.rtpErrors[i]++
			}
		})
		if err == nil {
			r.consumers = append(r.consumers, c)
		} else {
			return nil, err
		}
	}

	return r, nil
}

type (
	Sample      int32
	SampleFrame []Sample
)

var (
	ErrUnsupportedContentType = errors.New("unsupported content type")
)

func (r *RTPReceiver) ExtractSamples(packet *rtp.Packet) ([]SampleFrame, error) {
	var bytesPerSample uint32

	switch r.stream.Description.ContentType {
	case ContentTypePCM24:
		bytesPerSample = 3
	default:
		return nil, ErrUnsupportedContentType
	}

	channels := r.stream.Description.ChannelCount
	bytesPerFrame := bytesPerSample * channels
	numFrames := uint32(len(packet.Payload)) / bytesPerFrame

	var (
		i      uint32
		frames []SampleFrame
	)

	for range numFrames {
		frame := make(SampleFrame, channels)

		for ch := range channels {
			switch bytesPerSample {
			case 3:
				value := uint32(packet.Payload[i])<<24 |
					uint32(packet.Payload[i+1])<<16 |
					uint32(packet.Payload[i+2])<<8

				frame[ch] = Sample(value)

			default:
				return nil, ErrUnsupportedContentType
			}

			i += bytesPerSample
		}

		frames = append(frames, frame)
	}

	return frames, nil
}

func (r *RTPReceiver) Close() {
	for _, c := range r.consumers {
		r.stream.manager.multicastListener.RemoveConsumer(c)
	}
}

func (r *RTPReceiver) RTPErrors(i int) int {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	return r.rtpErrors[i]
}

type RTCPReceiverCallback func(int, net.Addr, rtcp.Packet)

type RTCPReceiver struct {
	mutex      sync.Mutex
	stream     *Stream
	consumers  []*multicast.Consumer
	rtcpErrors map[int]int
}

func (s *Stream) NewRTCPReceiver(cb RTCPReceiverCallback) (*RTCPReceiver, error) {
	r := &RTCPReceiver{
		stream:     s,
		consumers:  make([]*multicast.Consumer, 0),
		rtcpErrors: make(map[int]int),
	}

	for i, source := range s.Description.Sources {
		addr := net.UDPAddr{
			IP:   source.DestinationAddress,
			Port: int(source.DestinationPort) + 1,
		}

		c, err := s.manager.multicastListener.AddConsumer(&addr, func(ifi *net.Interface, src net.Addr, payload []byte) {
			if pkts, err := rtcp.Unmarshal(payload); err != nil {
				r.mutex.Lock()
				defer r.mutex.Unlock()

				r.rtcpErrors[i]++
			} else {
				for _, pkt := range pkts {
					cb(i, src, pkt)
				}
			}
		})
		if err == nil {
			r.consumers = append(r.consumers, c)
		} else {
			return nil, err
		}
	}

	return r, nil
}

func (r *RTCPReceiver) Close() {
	for _, c := range r.consumers {
		r.stream.manager.multicastListener.RemoveConsumer(c)
	}
}
