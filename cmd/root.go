package cmd

import (
	"fmt"
	"log/slog"
	"net"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/holoplot/rtp-monitor/internal/ptp"
	"github.com/holoplot/rtp-monitor/internal/stream"
	"github.com/holoplot/rtp-monitor/internal/ui"
	"github.com/spf13/cobra"
)

var (
	interfaceNames []string
	sdpFiles       []string
	wavFileFolder  string
)

var rootCmd = &cobra.Command{
	Use:   "rtp-monitor",
	Short: "Monitor RTP streams in your network",
	Long: `RTP Stream Monitor monitors and displays Real-time Transport Protocol (RTP)
streams in your network. It can discover streams via mDNS, SAP, or manual configuration.

The application provides a terminal-based user interface for interactive monitoring of RTP streams.`,
	Version: "0.1.0",
	RunE:    run,
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.Flags().StringArrayVar(&interfaceNames, "interface", []string{}, "Network interface to use (can be used multiple times)")
	rootCmd.Flags().StringArrayVar(&sdpFiles, "sdp", []string{}, "SDP file to parse (can be used multiple times)")
	rootCmd.Flags().StringVar(&wavFileFolder, "wav", "", "Folder to save WAV files")
}

// run is the main execution function
func run(cmd *cobra.Command, args []string) error {
	var ifis []net.Interface

	if len(interfaceNames) > 0 {
		for _, ifiName := range interfaceNames {
			ifi, err := net.InterfaceByName(ifiName)
			if err != nil {
				slog.Error("failed to get network interface", "interface", ifiName, "error", err)
				os.Exit(1)
			}

			ifis = append(ifis, *ifi)
		}
	} else {
		var err error

		ifis, err = net.Interfaces()
		if err != nil {
			slog.Error("failed to get network interfaces", "error", err)
			os.Exit(1)
		}
	}

	// Filter to multicast-capable interfaces
	var multicastIfis []*net.Interface
	for i := range ifis {
		if ifis[i].Flags&net.FlagMulticast != 0 && ifis[i].Flags&net.FlagUp != 0 {
			multicastIfis = append(multicastIfis, &ifis[i])
		}
	}

	if len(multicastIfis) == 0 {
		slog.Error("no multicast-capable interfaces found")
		os.Exit(1)
	}

	ifiNames := func() []string {
		names := make([]string, len(multicastIfis))
		for i := range multicastIfis {
			names[i] = multicastIfis[i].Name
		}

		return names
	}

	slog.Info("Multicast-capable interfaces found", "interfaces", ifiNames())

	manager := stream.NewManager(multicastIfis)

	// Parse SDP files if provided
	if err := manager.LoadSDPFiles(sdpFiles); err != nil {
		return fmt.Errorf("error loading SDP files: %w", err)
	}

	if err := manager.MonitorSAP(); err != nil {
		slog.Error("error monitoring SAP", "error", err)
	}

	if err := manager.MonitorMDns(); err != nil {
		slog.Error("error monitoring mDNS", "error", err)
	}

	// Track PTP Transitters
	ptpMonitor, err := ptp.NewMonitor(multicastIfis)
	if err != nil {
		slog.Error("error monitoring PTP - are you root?", "error", err)
	}

	model := ui.NewModel(manager, ptpMonitor, wavFileFolder)

	// Create a new Bubble Tea program
	p := tea.NewProgram(model, tea.WithAltScreen())

	manager.OnUpdate(func(s []*stream.Stream) {
		p.Send(ui.UpdateStreamsMsg{
			Streams: s,
		})
	})

	// Run the program
	if _, err := p.Run(); err != nil {
		slog.Error("error running UI", "error", err)
		os.Exit(1)
	}

	return nil
}
