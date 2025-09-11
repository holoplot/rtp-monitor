package version

import (
	"fmt"
	"runtime"
)

// Version information set at build time
var (
	// Version is the main version number that is being run at the moment.
	// This should be set via -ldflags at build time using git describe
	Version = ""

	// GitCommit is the git sha1 that was compiled. This will be filled in by the compiler.
	GitCommit = ""

	// BuildDate is the date the binary was built
	BuildDate = ""

	// GoVersion is the version of Go that was used to compile the binary
	GoVersion = runtime.Version()
)

// GetVersion returns the full version information
func GetVersion() string {
	version := Version
	if version == "" || version == "unknown" {
		version = "unknown"
	}

	buildDate := BuildDate
	if buildDate != "" && buildDate != "unknown" {
		// Convert 2025-09-11_13:47:21_UTC to "2025-09-11 13:47:21 UTC"
		if len(buildDate) >= 21 && buildDate[10] == '_' && buildDate[19] == '_' {
			buildDate = fmt.Sprintf("%s %s %s",
				buildDate[:10],
				buildDate[11:19],
				buildDate[20:])
		}
	}

	if GitCommit != "" && GitCommit != "unknown" && buildDate != "" && buildDate != "unknown" {
		return fmt.Sprintf("%s (commit: %.7s, built: %s, go: %s)",
			version, GitCommit, buildDate, GoVersion)
	} else if GitCommit != "" && GitCommit != "unknown" {
		return fmt.Sprintf("%s (commit: %.7s, go: %s)",
			version, GitCommit, GoVersion)
	} else if buildDate != "" && buildDate != "unknown" {
		return fmt.Sprintf("%s (built: %s, go: %s)",
			version, buildDate, GoVersion)
	}
	return fmt.Sprintf("%s (go: %s)", version, GoVersion)
}

// GetShortVersion returns just the version number
func GetShortVersion() string {
	if Version == "" || Version == "unknown" {
		return "unknown"
	}
	return Version
}

// GetBuildInfo returns detailed build information
func GetBuildInfo() map[string]string {
	return map[string]string{
		"version":   Version,
		"gitCommit": GitCommit,
		"buildDate": BuildDate,
		"goVersion": GoVersion,
	}
}
