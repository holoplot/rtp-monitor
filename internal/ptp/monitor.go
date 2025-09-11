package ptp

import (
	"fmt"
	"math/big"
	"net"
	"sort"
	"sync"
	"time"

	"github.com/holoplot/go-multicast/pkg/multicast"
)

type ClockIdentity struct {
	octets [8]byte
}

func (ci ClockIdentity) String() string {
	return fmt.Sprintf("%02x:%02x:%02x:%02x:%02x:%02x:%02x:%02x",
		ci.octets[0], ci.octets[1], ci.octets[2], ci.octets[3],
		ci.octets[4], ci.octets[5], ci.octets[6], ci.octets[7])
}

type Timestamp struct {
	PTP  [10]byte
	Time time.Time
}

func (ts Timestamp) String() string {
	epoch := time.Unix(0, 0).UTC()
	t := epoch.Add(time.Duration(ts.totalNanoSeconds()) * time.Nanosecond)
	return fmt.Sprintf("%s", t.Format(time.RFC3339Nano))
}

func (ts Timestamp) Seconds() uint64 {
	return uint64(ts.PTP[0])<<40 |
		uint64(ts.PTP[1])<<32 |
		uint64(ts.PTP[2])<<24 |
		uint64(ts.PTP[3])<<16 |
		uint64(ts.PTP[4])<<8 |
		uint64(ts.PTP[5])
}

func (ts Timestamp) NanoSeconds() uint64 {
	return uint64(ts.PTP[6])<<24 |
		uint64(ts.PTP[7])<<16 |
		uint64(ts.PTP[8])<<8 |
		uint64(ts.PTP[9])
}

func (ts Timestamp) totalNanoSeconds() uint64 {
	return ts.Seconds()*1_000_000_000 + ts.NanoSeconds()
}

func (ts Timestamp) InSamples(sampleRate uint32) uint32 {
	// Use math.Big to prevent overflow during multiplication
	totalNs := new(big.Int).SetUint64(ts.totalNanoSeconds())
	sampleRateBig := new(big.Int).SetUint64(uint64(sampleRate))
	billion := new(big.Int).SetUint64(1_000_000_000)

	// Calculate samples = totalNs * sampleRate / 1_000_000_000
	samples := new(big.Int).Mul(totalNs, sampleRateBig)
	samples.Div(samples, billion)

	return uint32(samples.Uint64())
}

type Transmitter struct {
	Domain        uint8
	LastTimestamp Timestamp
}

type Monitor struct {
	mutex             sync.Mutex
	multicastListener *multicast.Listener
	consumer          *multicast.Consumer
	transmitters      map[ClockIdentity]*Transmitter
}

const (
	messageTypeSync               = 0x0
	messageTypeDelayReq           = 0x1
	messageTypePDelayReq          = 0x2
	messageTypePDelayResp         = 0x3
	messageTypeFollowUp           = 0x8
	messageTypeDelayResp          = 0x9
	messageTypePDelayRespFollowUp = 0xa
	messageTypeAnnounce           = 0xb
	messageTypeSignaling          = 0xc
	messageTypeManagement         = 0xd
)

func (m *Monitor) parsePacket(_ *net.Interface, _ net.Addr, data []byte) {
	now := time.Now()

	if len(data) < 44 {
		return
	}

	messageType := data[0] & 0xf
	domainNumber := data[4]

	var clockIdentity ClockIdentity
	copy(clockIdentity.octets[:], data[20:28])

	m.mutex.Lock()
	defer m.mutex.Unlock()

	switch messageType {
	case messageTypeSync, messageTypeFollowUp:
		timeStamp := Timestamp{
			Time: now,
		}

		copy(timeStamp.PTP[:], data[34:44])

		if transmitter, ok := m.transmitters[clockIdentity]; ok {
			transmitter.LastTimestamp = timeStamp
		} else {
			m.transmitters[clockIdentity] = &Transmitter{
				Domain:        domainNumber,
				LastTimestamp: timeStamp,
			}
		}
	}
}

func (m *Monitor) ForEachTransmitter(fn func(ClockIdentity, *Transmitter)) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	var clockIDs []ClockIdentity
	for id := range m.transmitters {
		clockIDs = append(clockIDs, id)
	}

	sort.Slice(clockIDs, func(i, j int) bool {
		return clockIDs[i].String() < clockIDs[j].String()
	})

	// Iterate over sorted clock identities
	for _, id := range clockIDs {
		fn(id, m.transmitters[id])
	}
}

func NewMonitor(ifis []*net.Interface) (*Monitor, error) {
	m := &Monitor{
		multicastListener: multicast.NewListener(ifis),
		transmitters:      make(map[ClockIdentity]*Transmitter),
	}

	addr := &net.UDPAddr{
		IP:   net.IPv4(224, 0, 1, 129),
		Port: 319,
	}

	if c, err := m.multicastListener.AddConsumer(addr, m.parsePacket); err == nil {
		m.consumer = c
	} else {
		return nil, err
	}

	addr.Port = 320

	if c, err := m.multicastListener.AddConsumer(addr, m.parsePacket); err == nil {
		m.consumer = c
	} else {
		return nil, err
	}

	return m, nil
}
