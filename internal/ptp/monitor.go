package ptp

import (
	"net"
	"sort"
	"sync"
	"time"

	"github.com/holoplot/go-multicast/pkg/multicast"
)

type Transmitter struct {
	Domain        uint8
	LastTimestamp Timestamp
	IfiName       string
}

type Monitor struct {
	mutex             sync.Mutex
	multicastListener *multicast.Listener
	consumers         []*multicast.Consumer
	transmitters      map[ClockIdentity]*Transmitter
}

func (m *Monitor) parsePacket(ifi *net.Interface, _ net.Addr, data []byte) {
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

		if timeStamp.IsZero() {
			return
		}

		if transmitter, ok := m.transmitters[clockIdentity]; ok {
			transmitter.LastTimestamp = timeStamp
			transmitter.IfiName = ifi.Name
		} else {
			m.transmitters[clockIdentity] = &Transmitter{
				Domain:        domainNumber,
				LastTimestamp: timeStamp,
				IfiName:       ifi.Name,
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

	for _, port := range []int{319, 320} {
		addr := &net.UDPAddr{
			IP:   net.IPv4(224, 0, 1, 129),
			Port: port,
		}

		c, err := m.multicastListener.AddConsumer(addr, m.parsePacket)
		if err != nil {
			return nil, err
		}

		m.consumers = append(m.consumers, c)
	}

	return m, nil
}
