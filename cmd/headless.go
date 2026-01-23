package cmd

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"slices"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/holoplot/rtp-monitor/internal/stream"
)

// runHeadless runs the application in headless mode
func runHeadless(manager *stream.Manager, monitorIDHashes []string, reportInterval time.Duration) error {
	slog.Info("Starting headless mode", "monitorIDs", monitorIDHashes, "reportInterval", reportInterval)

	// Track discovered streams
	discoveredStreams := make(map[string]*stream.Stream)

	var discoveredStreamsLock sync.Mutex

	// Track monitored stream receivers
	monitoredReceivers := make(map[string]*streamMonitor)

	scanStreams := func(streamsSlice []*stream.Stream) {
		discoveredStreamsLock.Lock()
		defer discoveredStreamsLock.Unlock()

		streamsMap := make(map[string]*stream.Stream)
		for _, s := range streamsSlice {
			streamsMap[s.ID] = s
		}

		// Check for newly discovered streams
		for id, s := range streamsMap {
			if _, exists := discoveredStreams[id]; !exists {
				slog.Info("Stream discovered", "id", id, "id-hash", s.IDHash(), "name", s.Name(), "address", s.Address())
				discoveredStreams[id] = s

				// Start monitoring if this stream ID is in the monitor list
				if slices.Contains(monitorIDHashes, s.IDHash()) {
					monitoredReceivers[s.ID] = startMonitoring(s, reportInterval)
				}
			}
		}

		// Check for streams that went away
		for id, s := range discoveredStreams {
			if _, exists := streamsMap[id]; !exists {
				slog.Info("Stream disappeared", "id", id, "id-hash", s.IDHash(), "name", s.Name())
				delete(discoveredStreams, id)

				// Stop monitoring if this stream was being monitored
				if monitor, exists := monitoredReceivers[id]; exists {
					monitor.Stop()
					delete(monitoredReceivers, id)
				}
			}
		}
	}

	// Trigger initial discovery for any already loaded streams
	scanStreams(manager.GetAllStreams())

	// Set up stream update callback to track newly discovered/disappeared streams
	manager.OnUpdate(scanStreams)

	// Set up signal handling for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigChan
		slog.Info("Received signal, shutting down gracefully", "signal", sig)
		cancel()
	}()

	// Wait for shutdown signal
	<-ctx.Done()

	// Stop all monitoring
	for id, monitor := range monitoredReceivers {
		slog.Info("Stopping monitor", "stream", id)
		monitor.Stop()
	}

	slog.Info("Headless mode shutdown complete")
	return nil
}

// streamMonitor monitors a specific stream for packet rate and sequence errors
type streamMonitor struct {
	stream   *stream.Stream
	receiver *stream.RTPReceiver
	ticker   *time.Ticker
	stopCh   chan struct{}

	// Statistics
	lastReportTime  time.Time
	lastPacketCount map[int]uint64
}

func startMonitoring(s *stream.Stream, reportInterval time.Duration) *streamMonitor {
	slog.Info("Starting monitoring", "stream", s.ID, "id-hash", s.IDHash(), "name", s.Name())

	monitor := &streamMonitor{
		stream:          s,
		stopCh:          make(chan struct{}),
		ticker:          time.NewTicker(reportInterval),
		lastReportTime:  time.Now(),
		lastPacketCount: make(map[int]uint64),
	}

	// Create RTP receiver
	receiver, err := s.NewRTPReceiver(nil)
	if err != nil {
		slog.Error("Failed to create RTP receiver", "stream", s.ID, "error", err)

		return nil
	}

	monitor.receiver = receiver

	// Start monitoring goroutine
	go func() {
		defer func() {
			monitor.ticker.Stop()
			monitor.receiver.Close()
			slog.Info("Monitor stopped", "stream", s.ID, "name", s.Name())
		}()

		for {
			select {
			case <-monitor.ticker.C:
				monitor.reportStats()
			case <-monitor.stopCh:
				return
			}
		}
	}()

	return monitor
}

func (m *streamMonitor) reportStats() {
	now := time.Now()
	duration := now.Sub(m.lastReportTime)

	var packetRates, sequenceErrors []string

	for i := 0; i < m.receiver.NumSources(); i++ {
		packetsInPeriod := m.receiver.PacketCount(i) - m.lastPacketCount[i]
		packetRate := float64(packetsInPeriod) / duration.Seconds()
		packetRates = append(packetRates, fmt.Sprintf("%.2f", packetRate))

		sequenceErrors = append(sequenceErrors, fmt.Sprintf("%d", m.receiver.SequenceErrors(i)))

		m.lastPacketCount[i] = m.receiver.PacketCount(i)
	}

	slog.Info("Stream statistics",
		"name", m.stream.Name(),
		"id-hash", m.stream.IDHash(),
		"packet_rate", strings.Join(packetRates, "/"),
		"sequence_errors", strings.Join(sequenceErrors, "/"),
	)

	m.lastReportTime = now
}

func (m *streamMonitor) Stop() {
	close(m.stopCh)
}
