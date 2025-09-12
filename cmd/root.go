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
	"github.com/holoplot/rtp-monitor/internal/version"
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
	Version: version.GetVersion(),
	RunE:    run,
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show version information",
	Long:  "Display detailed version information including build date, git commit, and Go version.",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println(version.GetVersion())
	},
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.AddCommand(versionCmd)
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
	for _, ifi := range ifis {
		if ifi.Flags&net.FlagUp == 0 {
			continue
		}

		if ifi.Flags&net.FlagLoopback != 0 {
			continue
		}

		if ifi.Flags&net.FlagMulticast == 0 {
			continue
		}

		hasIPv4Addr := func(ifi *net.Interface) bool {
			addrs, err := ifi.Addrs()
			if err != nil {
				slog.Error("failed to get network interface addresses", "interface", ifi.Name, "error", err)

				return false
			}

			for _, addr := range addrs {
				if ip, _, err := net.ParseCIDR(addr.String()); err == nil && ip.To4() != nil {
					return true
				}
			}

			return false
		}

		if !hasIPv4Addr(&ifi) {
			continue
		}

		if addrs, err := ifi.Addrs(); err != nil || len(addrs) == 0 {
			continue
		}

		multicastIfis = append(multicastIfis, &ifi)
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
