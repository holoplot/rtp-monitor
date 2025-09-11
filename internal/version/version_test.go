package version

import (
	"strings"
	"testing"
)

func TestGetShortVersion(t *testing.T) {
	// Save original values
	originalVersion := Version
	defer func() { Version = originalVersion }()

	// Test default version
	Version = "1.2.3"
	got := GetShortVersion()
	if got != "1.2.3" {
		t.Errorf("GetShortVersion() = %v, want %v", got, "1.2.3")
	}
}

func TestGetVersion(t *testing.T) {
	// Save original values
	originalVersion := Version
	originalGitCommit := GitCommit
	originalBuildDate := BuildDate
	defer func() {
		Version = originalVersion
		GitCommit = originalGitCommit
		BuildDate = originalBuildDate
	}()

	tests := []struct {
		name      string
		version   string
		commit    string
		buildDate string
		wantParts []string
	}{
		{
			name:      "git describe with tag",
			version:   "v1.2.3-5-gabcdef0",
			commit:    "abc123def456ghi789",
			buildDate: "2025-09-11_13:45:30_UTC",
			wantParts: []string{"v1.2.3-5-gabcdef0", "commit: abc123d", "built: 2025-09-11 13:45:30 UTC", "go:"},
		},
		{
			name:      "git describe dirty",
			version:   "v2.0.0-dirty",
			commit:    "def456abc123",
			buildDate: "",
			wantParts: []string{"v2.0.0-dirty", "commit: def456a", "go:"},
		},
		{
			name:      "git describe hash only",
			version:   "abcdef0",
			commit:    "",
			buildDate: "2025-01-01_12:00:00_UTC",
			wantParts: []string{"abcdef0", "built: 2025-01-01 12:00:00 UTC", "go:"},
		},
		{
			name:      "unknown version",
			version:   "unknown",
			commit:    "",
			buildDate: "",
			wantParts: []string{"unknown", "go:"},
		},
		{
			name:      "empty version falls back to unknown",
			version:   "",
			commit:    "abc123",
			buildDate: "unknown",
			wantParts: []string{"unknown", "commit: abc123", "go:"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			Version = tt.version
			GitCommit = tt.commit
			BuildDate = tt.buildDate

			got := GetVersion()

			// Check that all expected parts are present
			for _, part := range tt.wantParts {
				if !strings.Contains(got, part) {
					t.Errorf("GetVersion() = %v, missing part %v", got, part)
				}
			}

			// Check that version is always present
			if !strings.Contains(got, tt.version) {
				t.Errorf("GetVersion() = %v, missing version %v", got, tt.version)
			}
		})
	}
}

func TestGetBuildInfo(t *testing.T) {
	// Save original values
	originalVersion := Version
	originalGitCommit := GitCommit
	originalBuildDate := BuildDate
	defer func() {
		Version = originalVersion
		GitCommit = originalGitCommit
		BuildDate = originalBuildDate
	}()

	Version = "v1.0.0-2-gabcdef0"
	GitCommit = "abc123"
	BuildDate = "2025-09-11_13:45:30_UTC"

	info := GetBuildInfo()

	expectedKeys := []string{"version", "gitCommit", "buildDate", "goVersion"}
	for _, key := range expectedKeys {
		if _, exists := info[key]; !exists {
			t.Errorf("GetBuildInfo() missing key %v", key)
		}
	}

	if info["version"] != "v1.0.0-2-gabcdef0" {
		t.Errorf("GetBuildInfo()['version'] = %v, want %v", info["version"], "v1.0.0-2-gabcdef0")
	}

	if info["gitCommit"] != "abc123" {
		t.Errorf("GetBuildInfo()['gitCommit'] = %v, want %v", info["gitCommit"], "abc123")
	}

	if info["buildDate"] != "2025-09-11_13:45:30_UTC" {
		t.Errorf("GetBuildInfo()['buildDate'] = %v, want %v", info["buildDate"], "2025-09-11_13:45:30_UTC")
	}

	// goVersion should contain "go"
	if !strings.Contains(info["goVersion"], "go") {
		t.Errorf("GetBuildInfo()['goVersion'] = %v, should contain 'go'", info["goVersion"])
	}
}

func TestBuildDateFormatting(t *testing.T) {
	// Save original values
	originalVersion := Version
	originalBuildDate := BuildDate
	defer func() {
		Version = originalVersion
		BuildDate = originalBuildDate
	}()

	Version = "v1.0.0-1-gabcdef0-dirty"
	BuildDate = "2025-09-11_13:45:30_UTC"

	got := GetVersion()

	// Should contain formatted date with spaces instead of underscores
	if !strings.Contains(got, "2025-09-11 13:45:30 UTC") {
		t.Errorf("GetVersion() = %v, should contain formatted date '2025-09-11 13:45:30 UTC'", got)
	}

	// Should not contain the underscored version
	if strings.Contains(got, "2025-09-11_13:45:30_UTC") {
		t.Errorf("GetVersion() = %v, should not contain raw underscore format", got)
	}
}

func TestEdgeCases(t *testing.T) {
	// Save original values
	originalVersion := Version
	originalBuildDate := BuildDate
	defer func() {
		Version = originalVersion
		BuildDate = originalBuildDate
	}()

	// Test with malformed build date
	Version = "v1.0.0"
	BuildDate = "invalid"

	got := GetVersion()

	// Should handle malformed date gracefully
	if !strings.Contains(got, "v1.0.0") {
		t.Errorf("GetVersion() should still contain version even with invalid build date")
	}

	// Test with empty version (should fall back to unknown)
	Version = ""
	BuildDate = ""

	got = GetVersion()

	// Should fall back to unknown
	if !strings.Contains(got, "unknown") {
		t.Errorf("GetVersion() should fall back to 'unknown' for empty version")
	}
	if !strings.Contains(got, "go:") {
		t.Errorf("GetVersion() should always contain Go version info")
	}
}
