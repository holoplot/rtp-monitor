package stream

import (
	"fmt"
	"log/slog"
	"net"
	"net/url"
	"os"
	"path"
	"sort"
	"sync"
	"time"

	"github.com/bluenviron/gortsplib/v5"
	"github.com/bluenviron/gortsplib/v5/pkg/base"
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

	// mDnsServiceStreams maps an avahi service key to the stream ID it most
	// recently resolved to, so we can drop the matching mDNS Discovery record
	// when the service goes away.
	mDnsServiceStreams map[string]mDnsServiceRef
}

type mDnsServiceRef struct {
	streamID string
	source   string
}

// NewManager creates a new stream manager
func NewManager(ifis []*net.Interface) *Manager {
	m := &Manager{
		multicastListener:  multicast.NewListener(ifis),
		streams:            make(map[string]*Stream),
		mDnsServiceStreams: make(map[string]mDnsServiceRef),
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

	// Sort by name, with ID as secondary sort key
	sort.Slice(streams, func(i, j int) bool {
		nameA := streams[i].Name()
		nameB := streams[j].Name()
		if nameA == nameB {
			return streams[i].ID < streams[j].ID
		}
		return nameA < nameB
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
		Host:   u.Host,
	}

	if err := c.Start(); err != nil {
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
							m.mDnsServiceStreams[keyForService(service)] = mDnsServiceRef{
								streamID: stream.ID,
								source:   ifiName,
							}
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

				key := keyForService(avahiService)

				m.mutex.Lock()
				ref, ok := m.mDnsServiceStreams[key]
				if ok {
					delete(m.mDnsServiceStreams, key)
					if stream, exists := m.streams[ref.streamID]; exists {
						stream.RemoveDiscovery(DiscoveryMethodMDNS, ref.source)
						if len(stream.Discoveries) == 0 {
							delete(m.streams, stream.ID)
						}
					}
				}
				m.mutex.Unlock()

				if ok {
					m.update()
				}
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

	stream, err := m.AddStreamFromSDP(data, DiscoveryMethodManual, path.Base(filename))
	if err != nil {
		return fmt.Errorf("failed to add stream from SDP file %s: %w", filename, err)
	}

	slog.Info("Loaded stream", "name", stream.Name(), "filename", filename)

	return nil
}

func (m *Manager) AddStreamFromSDP(sdp []byte, discoveryMethod DiscoveryMethod, source string) (*Stream, error) {
	description, uniqueID, err := ParseSDP(sdp)
	if err != nil {
		return nil, fmt.Errorf("failed to parse SDP: %w", err)
	}

	m.mutex.Lock()

	if existing, ok := m.streams[uniqueID]; ok {
		// Refresh the existing stream and add or refresh this discovery record.
		existing.Description = *description
		existing.SDP = sdp
		existing.AddOrRefreshDiscovery(discoveryMethod, source)
		m.mutex.Unlock()

		m.update()
		return existing, nil
	}

	stream := &Stream{
		ID:          uniqueID,
		Description: *description,
		SDP:         sdp,
		Discoveries: []Discovery{{
			Method:   discoveryMethod,
			Source:   source,
			LastSeen: time.Now(),
		}},
		manager: m,
	}
	m.streams[uniqueID] = stream
	m.mutex.Unlock()

	m.update()
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

// cleanupStaleStreams expires individual SAP discovery records after sapTimeout
// of silence and drops a stream entirely once it has no remaining discoveries.
// mDNS records are removed via the avahi remove channel; manual records never
// expire.
func (m *Manager) cleanupStaleStreams() {
	m.mutex.Lock()

	now := time.Now()
	removed := false

	for id, stream := range m.streams {
		kept := stream.Discoveries[:0]
		for _, d := range stream.Discoveries {
			if d.Method == DiscoveryMethodSAP && now.Sub(d.LastSeen) > sapTimeout {
				removed = true
				continue
			}
			kept = append(kept, d)
		}
		stream.Discoveries = kept
		if len(stream.Discoveries) == 0 {
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
