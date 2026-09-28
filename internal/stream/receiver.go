package stream

import (
	"net"
	"sync"

	"github.com/holoplot/go-multicast/pkg/multicast"
	"github.com/pion/rtcp"
	"github.com/pion/rtp/v2"
)

type RTPReceiverCallback func(int, net.Addr, *rtp.Packet)

type RTPReceiver struct {
	mutex          sync.Mutex
	stream         *Stream
	consumers      []*multicast.Consumer
	packetCount    map[int]uint64
	rtpErrors      map[int]uint64
	sequenceErrors map[int]uint64
	lastSequence   map[int]uint16
}

func (s *Stream) NewRTPReceiver(cb RTPReceiverCallback) (*RTPReceiver, error) {
	if s.manager == nil {
		return nil, ErrDetached
	}

	r := &RTPReceiver{
		stream:         s,
		consumers:      make([]*multicast.Consumer, 0),
		packetCount:    make(map[int]uint64),
		rtpErrors:      make(map[int]uint64),
		sequenceErrors: make(map[int]uint64),
		lastSequence:   make(map[int]uint16),
	}

	for i, source := range s.Description.Sources {
		addr := net.UDPAddr{
			IP:   source.DestinationAddress,
			Port: int(source.DestinationPort),
		}

		c, err := s.manager.multicastListener.AddConsumer(&addr, func(ifi *net.Interface, src net.Addr, payload []byte) {
			packet := &rtp.Packet{}
			if err := packet.Unmarshal(payload); err == nil {
				r.mutex.Lock()

				r.packetCount[i]++

				if r.packetCount[i] > 1 {
					if packet.SequenceNumber != r.lastSequence[i]+1 {
						r.sequenceErrors[i]++
					}
				}

				r.lastSequence[i] = packet.SequenceNumber

				r.mutex.Unlock()

				if cb != nil {
					cb(i, src, packet)
				}
			} else {
				r.mutex.Lock()
				r.rtpErrors[i]++
				r.mutex.Unlock()

			}
		})
		if err != nil {
			r.Close()
			return nil, err
		}

		r.consumers = append(r.consumers, c)
	}

	return r, nil
}

func (r *RTPReceiver) ExtractSamples(packet *rtp.Packet) ([]SampleFrame, error) {
	return r.stream.Description.ExtractSamples(packet)
}

func (r *RTPReceiver) Close() {
	for _, c := range r.consumers {
		r.stream.manager.multicastListener.RemoveConsumer(c)
	}
}

func (r *RTPReceiver) NumSources() int {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	return len(r.consumers)
}

func (r *RTPReceiver) PacketCount(i int) uint64 {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	return r.packetCount[i]
}

func (r *RTPReceiver) RTPErrors(i int) uint64 {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	return r.rtpErrors[i]
}

func (r *RTPReceiver) SequenceErrors(i int) uint64 {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	return r.sequenceErrors[i]
}

type RTCPReceiverCallback func(int, net.Addr, rtcp.Packet)

type RTCPReceiver struct {
	mutex      sync.Mutex
	stream     *Stream
	consumers  []*multicast.Consumer
	rtcpErrors map[int]uint64
}

func (s *Stream) NewRTCPReceiver(cb RTCPReceiverCallback) (*RTCPReceiver, error) {
	if s.manager == nil {
		return nil, ErrDetached
	}

	r := &RTCPReceiver{
		stream:     s,
		consumers:  make([]*multicast.Consumer, 0),
		rtcpErrors: make(map[int]uint64),
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
		if err != nil {
			r.Close()
			return nil, err
		}

		r.consumers = append(r.consumers, c)
	}

	return r, nil
}

func (r *RTCPReceiver) Close() {
	for _, c := range r.consumers {
		r.stream.manager.multicastListener.RemoveConsumer(c)
	}
}
