package stream

import (
	"crypto/sha256"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/holoplot/sdp"
)

// DiscoveryMethod represents how an RTP stream was discovered
type DiscoveryMethod string

const (
	DiscoveryMethodSAP    DiscoveryMethod = "SAP"
	DiscoveryMethodMDNS   DiscoveryMethod = "mDNS"
	DiscoveryMethodManual DiscoveryMethod = "Manual"
)

type ContentType string

const (
	ContentTypeUndefined ContentType = "Undefined"
	ContentTypePCM16     ContentType = "PCM16"
	ContentTypePCM24     ContentType = "PCM24"
)

func (d DiscoveryMethod) String() string {
	return string(d)
}

type StreamSource struct {
	SenderAddress      net.IP
	DestinationAddress net.IP
	DestinationPort    uint16
	TTL                uint8
	FramesPerPacket    uint32

	ClockDomain    string
	ReferenceClock string
	MediaClock     string
	SyncTime       uint32
}

type StreamDescription struct {
	Sources []StreamSource
	Name    string

	SampleRate   uint32
	ChannelCount uint32
	ContentType  ContentType
}

func ParseSDP(b []byte) (*StreamDescription, string, error) {
	session, err := sdp.DecodeSession(b, sdp.Session{})
	if err != nil {
		return nil, "", err
	}

	decoder := sdp.NewDecoder(session)

	message := new(sdp.Message)
	if err := decoder.Decode(message); err != nil {
		return nil, "", fmt.Errorf("can not decode message: %w", err)
	}

	// RFC4566, 5.2:
	// the tuple of <username>, <sess-id>, <nettype>, <addrtype>, and <unicast-address>
	// forms a globally unique identifier for the session
	uniqueID := fmt.Sprintf("%s-%d-%s-%s-%s",
		message.Origin.Username,
		message.Origin.SessionID,
		message.Connection.NetworkType,
		message.Connection.AddressType,
		message.Origin.Address)

	sd := &StreamDescription{
		Name: message.Name,
	}

	for _, media := range message.Medias {
		if media.Description.Type != "audio" {
			continue
		}

		connection := media.Connection

		if connection.Blank() {
			connection = message.Connection
		}

		source := StreamSource{
			SenderAddress:      net.ParseIP(message.Origin.Address),
			DestinationAddress: connection.IP,
			DestinationPort:    uint16(media.Description.Port),
			TTL:                uint8(connection.TTL),
			ClockDomain:        media.Attribute("clock-domain"),
			ReferenceClock:     media.Attribute("ts-refclk"),
		}

		i, _ := strconv.Atoi(media.Attribute("framecount"))
		source.FramesPerPacket = uint32(i)

		s := media.Attribute("source-filter")
		a := strings.Split(s, " ")

		if len(a) == 6 {
			source.SenderAddress = net.ParseIP(a[5])
		}

		if len(source.ClockDomain) == 0 {
			source.ClockDomain = message.Attribute("clock-domain")
		}

		if len(source.ReferenceClock) == 0 {
			source.ReferenceClock = message.Attribute("ts-refclk")
		}

		mediaclk := media.Attribute("mediaclk")
		if len(s) > 0 {
			source.MediaClock = mediaclk

			if i, err := strconv.Atoi(media.Attribute("sync-time")); err == nil {
				source.SyncTime = uint32(i)
			}
		}

		s = media.Attribute("rtpmap")
		a = strings.Split(s, " ")

		if len(a) > 1 {
			b := strings.Split(a[1], "/")
			if len(b) == 3 {
				sd.ContentType = func(s string) ContentType {
					switch s {
					case "L24":
						return ContentTypePCM24
					default:
						return ContentTypeUndefined
					}
				}(b[0])

				if sampleRate, err := strconv.Atoi(b[1]); err == nil {
					sd.SampleRate = uint32(sampleRate)
				}

				if channelCount, err := strconv.Atoi(b[2]); err == nil {
					sd.ChannelCount = uint32(channelCount)
				}
			}
		}

		sd.Sources = append(sd.Sources, source)
	}

	return sd, uniqueID, nil
}

// Discovery records one way a stream has been discovered. A single stream may
// be discovered through multiple methods or on multiple interfaces; each
// (method, source) tuple gets its own Discovery entry.
type Discovery struct {
	Method DiscoveryMethod
	Source string
	// LastSeen is the timestamp of the most recent advertisement. Only the
	// SAP path refreshes this periodically, so cleanup and "last seen"
	// reporting only apply to DiscoveryMethodSAP. For mDNS and Manual this
	// is the time the discovery was first added.
	LastSeen time.Time
}

// Stream represents an RTP stream with its metadata
type Stream struct {
	ID          string
	Description StreamDescription

	SDP []byte

	// All discovery records for this stream, in the order they were first seen.
	Discoveries []Discovery

	manager *Manager
}

func (s *Stream) Name() string {
	return s.Description.Name
}

func (s *Stream) IDHash() string {
	return fmt.Sprintf("%x", sha256.Sum256([]byte(s.ID)))[:10]
}

// findDiscovery returns the index of a matching (method, source) record, or -1.
func (s *Stream) findDiscovery(method DiscoveryMethod, source string) int {
	for i, d := range s.Discoveries {
		if d.Method == method && d.Source == source {
			return i
		}
	}
	return -1
}

// AddOrRefreshDiscovery appends a new discovery record or refreshes an existing
// one's LastSeen timestamp. Returns true if a new record was added.
func (s *Stream) AddOrRefreshDiscovery(method DiscoveryMethod, source string) bool {
	now := time.Now()
	if i := s.findDiscovery(method, source); i >= 0 {
		s.Discoveries[i].LastSeen = now
		return false
	}
	s.Discoveries = append(s.Discoveries, Discovery{
		Method:   method,
		Source:   source,
		LastSeen: now,
	})
	return true
}

// RemoveDiscovery removes a (method, source) record. Returns true if removed.
func (s *Stream) RemoveDiscovery(method DiscoveryMethod, source string) bool {
	i := s.findDiscovery(method, source)
	if i < 0 {
		return false
	}
	s.Discoveries = append(s.Discoveries[:i], s.Discoveries[i+1:]...)
	return true
}

// DiscoveryLabel returns a compact one-line representation of all discoveries,
// e.g. "mDNS@eth0, SAP@eth1".
func (s *Stream) DiscoveryLabel() string {
	parts := make([]string, 0, len(s.Discoveries))
	for _, d := range s.Discoveries {
		if d.Source == "" {
			parts = append(parts, d.Method.String())
		} else {
			parts = append(parts, fmt.Sprintf("%s@%s", d.Method, d.Source))
		}
	}
	return strings.Join(parts, ", ")
}

// Address returns the formatted network address
func (s *Stream) Address() string {
	a := []string{}

	for _, source := range s.Description.Sources {
		a = append(a, fmt.Sprintf("%s:%d", source.DestinationAddress.String(), source.DestinationPort))
	}

	return strings.Join(a, ", ")
}

// CodecInfo returns formatted codec information
func (s *Stream) CodecInfo() string {
	desc := s.Description
	if desc.ContentType != "" {
		if desc.SampleRate > 0 && desc.ChannelCount > 0 {
			return fmt.Sprintf("%s %dHz %dch", desc.ContentType, desc.SampleRate, desc.ChannelCount)
		}
		return string(desc.ContentType)
	}
	return "Unknown"
}
