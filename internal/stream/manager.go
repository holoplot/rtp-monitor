package stream

import (
	"fmt"
	"log/slog"
	"net"
	"net/url"
	"os"
	"sort"
	"sync"
	"time"

	"github.com/bluenviron/gortsplib/v4"
	"github.com/bluenviron/gortsplib/v4/pkg/base"
	"github.com/godbus/dbus/v5"
	"github.com/holoplot/go-avahi"
	"github.com/holoplot/go-multicast/pkg/multicast"
	"github.com/holoplot/go-sap/pkg/sap"
)

const (
	cleanupPeriod = 5 * time.Second
	sapTimeout    = 10 * time.Minute

	mDnsRavennaServiceName = "_ravenna_session._sub._rtsp._tcp"
	mDnsResolveTimeout     = time.Minute

	sapAddress = "239.255.255.255:9875"
)

type UpdateCallback func([]*Stream)

// Manager manages a collection of RTP streams
type Manager struct {
	mutex   sync.Mutex
	streams map[string]*Stream

	updateCallback UpdateCallback

	multicastListener *multicast.Listener

	sapConsumer *multicast.Consumer

	mDnsStreams map[string]*Stream
}

// NewManager creates a new stream manager
func NewManager(ifis []*net.Interface) *Manager {
	m := &Manager{
		multicastListener: multicast.NewListener(ifis),
		streams:           make(map[string]*Stream),
		mDnsStreams:       make(map[string]*Stream),
	}

	go func() {
		ticker := time.NewTicker(cleanupPeriod)
		defer ticker.Stop()
		for range ticker.C {
			m.cleanupStaleStreams()
		}
	}()

	return m
}

func (m *Manager) update() {
	if m.updateCallback == nil {
		return
	}

	m.mutex.Lock()

	streams := make([]*Stream, 0, len(m.streams))
	for _, stream := range m.streams {
		streams = append(streams, stream)
	}

	m.mutex.Unlock()

	// Sort by name
	sort.Slice(streams, func(i, j int) bool {
		return streams[i].Description.Name < streams[j].Description.Name
	})

	m.updateCallback(streams)
}

func (m *Manager) OnUpdate(callback UpdateCallback) {
	m.updateCallback = callback
}

func readRTSP(uri string) ([]byte, error) {
	u, err := base.ParseURL(uri)
	if err != nil {
		return nil, fmt.Errorf("failed to parse URL: %w", err)
	}

	c := gortsplib.Client{
		Scheme: u.Scheme,
	}

	if err := c.Start(u.Scheme, u.Host); err != nil {
		return nil, fmt.Errorf("failed to start client: %w", err)
	}

	_, response, err := c.Describe(u)
	if err != nil {
		return nil, fmt.Errorf("failed to describe stream: %w", err)
	}

	return response.Body, nil
}

func (m *Manager) MonitorMDns() error {
	var err error

	dbusConn, err := dbus.SystemBus()
	if err != nil {
		return fmt.Errorf("can not connect to dbus: %w", err)
	}

	avahiServer, err := avahi.ServerNew(dbusConn)
	if err != nil {
		return fmt.Errorf("avahi.ServerNew() failed: %w", err)
	}

	keyForService := func(service avahi.Service) string {
		return fmt.Sprintf("%s.%s@%d_%d", service.Name, service.Domain, service.Interface, service.Protocol)
	}

	go func() {
		serviceBrowser, err := avahiServer.ServiceBrowserNew(avahi.InterfaceUnspec, avahi.ProtoUnspec,
			mDnsRavennaServiceName, "local", 0)
		if err != nil {
			fmt.Printf("avahi.ServiceBrowserNew() failed: %v\n", err)
			return
		}

		for {
			select {
			case avahiService, ok := <-serviceBrowser.AddChannel:
				if !ok {
					return
				}

				go func(service avahi.Service) {
					resolver, err := avahiServer.ServiceResolverNew(
						service.Interface, service.Protocol, service.Name,
						service.Type, service.Domain, service.Protocol, 0)
					if err != nil {
						fmt.Printf("avahi.ServiceResolverNew() failed: %v\n", err)
						return
					}

					for {
						select {
						case r := <-resolver.FoundChannel:
							uri := fmt.Sprintf("rtsp://%s:%d/by-name/%s",
								r.Address, r.Port, url.PathEscape(service.Name))

							sdpBytes, err := readRTSP(uri)
							if err != nil {
								return
							}

							ifiName := "unknown"

							if ifi, err := net.InterfaceByIndex(int(service.Interface)); err == nil {
								ifiName = ifi.Name
							}

							stream, err := m.AddStreamFromSDP(sdpBytes, DiscoveryMethodMDNS, ifiName)
							if err != nil {
								return
							}

							m.mutex.Lock()
							m.mDnsStreams[keyForService(service)] = stream
							m.mutex.Unlock()

							return

						case <-time.After(mDnsResolveTimeout):
							return
						}
					}
				}(avahiService)

			case avahiService, ok := <-serviceBrowser.RemoveChannel:
				if !ok {
					return
				}

				m.mutex.Lock()
				if stream, ok := m.mDnsStreams[keyForService(avahiService)]; ok {
					delete(m.mDnsStreams, keyForService(avahiService))
					delete(m.streams, stream.ID)
				}
				m.mutex.Unlock()

				m.update()
			}
		}
	}()

	return nil
}

func (m *Manager) MonitorSAP() error {
	udpAddr, err := net.ResolveUDPAddr("udp", sapAddress)
	if err != nil {
		return err
	}

	m.sapConsumer, err = m.multicastListener.AddConsumer(udpAddr, func(ifi *net.Interface, _ net.Addr, payload []byte) {
		p, err := sap.DecodePacket(payload)
		if err != nil {
			return
		}

		m.AddStreamFromSDP(p.Payload, DiscoveryMethodSAP, ifi.Name)
	})

	return nil
}

// loadSDPFiles parses all specified SDP files and adds streams to the manager
func (m *Manager) LoadSDPFiles(files []string) error {
	for _, filename := range files {
		if err := m.LoadSDPFile(filename); err != nil {
			return fmt.Errorf("failed to load SDP file %s: %w", filename, err)
		}
	}

	return nil
}

// loadSDPFile parses a single SDP file and adds its streams to the manager
func (m *Manager) LoadSDPFile(filename string) error {
	data, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	stream, err := m.AddStreamFromSDP(data, DiscoveryMethodManual, filename)
	if err != nil {
		return fmt.Errorf("failed to add stream from SDP file %s: %w", filename, err)
	}

	slog.Info("Loaded stream", "name", stream.Description.Name, "filename", filename)

	return nil
}

// AddStream adds a new stream to the manager
func (m *Manager) AddStream(stream *Stream) {
	m.mutex.Lock()
	stream.manager = m
	m.streams[stream.ID] = stream
	m.mutex.Unlock()

	m.update()
}

func (m *Manager) AddStreamFromSDP(sdp []byte, discoveryMethod DiscoveryMethod, interfaceName string) (*Stream, error) {
	description, uniqueID, err := ParseSDP(sdp)
	if err != nil {
		return nil, fmt.Errorf("failed to parse SDP: %w", err)
	}

	stream := &Stream{
		ID:              uniqueID,
		Description:     *description,
		SDP:             sdp,
		DiscoveryMethod: discoveryMethod,
		DiscoverySource: interfaceName,
		LastSeen:        time.Now(),
	}

	m.AddStream(stream)

	return stream, nil
}

// RemoveStream removes a stream from the manager
func (m *Manager) RemoveStream(id string) {
	m.mutex.Lock()
	delete(m.streams, id)
	m.mutex.Unlock()

	m.update()
}

// GetStream returns a stream by ID
func (m *Manager) GetStream(id string) (*Stream, bool) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	stream, exists := m.streams[id]
	return stream, exists
}

// GetAllStreams returns all streams as a slice, sorted by name
func (m *Manager) GetAllStreams() []*Stream {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	streams := make([]*Stream, 0, len(m.streams))
	for _, stream := range m.streams {
		streams = append(streams, stream)
	}

	return streams
}

// CleanupStaleStreams removes streams that haven't been seen for a while
func (m *Manager) cleanupStaleStreams() {
	m.mutex.Lock()

	removed := false

	for id, stream := range m.streams {
		if stream.DiscoveryMethod == DiscoveryMethodSAP && stream.IsStale(sapTimeout) {
			delete(m.streams, id)
			removed = true
		}
	}

	m.mutex.Unlock()

	if removed {
		m.update()
	}
}

// Count returns the number of managed streams
func (m *Manager) Count() int {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	return len(m.streams)
}
