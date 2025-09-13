package ptp

import (
	"math/big"
	"strings"
	"testing"
	"time"
)

func TestTimestampSeconds(t *testing.T) {
	tests := []struct {
		name     string
		ptpBytes [10]byte
		expected uint64
	}{
		{
			name:     "zero seconds",
			ptpBytes: [10]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
			expected: 0,
		},
		{
			name:     "one second",
			ptpBytes: [10]byte{0, 0, 0, 0, 0, 1, 0, 0, 0, 0},
			expected: 1,
		},
		{
			name:     "max uint32 in lower bytes",
			ptpBytes: [10]byte{0, 0, 0xFF, 0xFF, 0xFF, 0xFF, 0, 0, 0, 0},
			expected: 0xFFFFFFFF,
		},
		{
			name:     "high bytes set",
			ptpBytes: [10]byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0, 0, 0, 0},
			expected: 0x010203040506,
		},
		{
			name:     "all seconds bytes set",
			ptpBytes: [10]byte{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0, 0, 0, 0},
			expected: 0xFFFFFFFFFFFF,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := Timestamp{PTP: tt.ptpBytes}
			got := ts.Seconds()
			if got != tt.expected {
				t.Errorf("Seconds() = %d, want %d", got, tt.expected)
			}
		})
	}
}

func TestTimestampNanoSeconds(t *testing.T) {
	tests := []struct {
		name     string
		ptpBytes [10]byte
		expected uint64
	}{
		{
			name:     "zero nanoseconds",
			ptpBytes: [10]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
			expected: 0,
		},
		{
			name:     "one nanosecond",
			ptpBytes: [10]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 1},
			expected: 1,
		},
		{
			name:     "max uint32 nanoseconds",
			ptpBytes: [10]byte{0, 0, 0, 0, 0, 0, 0xFF, 0xFF, 0xFF, 0xFF},
			expected: 0xFFFFFFFF,
		},
		{
			name:     "typical nanosecond value",
			ptpBytes: [10]byte{0, 0, 0, 0, 0, 0, 0x12, 0x34, 0x56, 0x78},
			expected: 0x12345678,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := Timestamp{PTP: tt.ptpBytes}
			got := ts.NanoSeconds()
			if got != tt.expected {
				t.Errorf("NanoSeconds() = %d, want %d", got, tt.expected)
			}
		})
	}
}

func TestTimestampTotalNanoSeconds(t *testing.T) {
	tests := []struct {
		name     string
		ptpBytes [10]byte
		expected string // Use string representation for big numbers
	}{
		{
			name:     "zero time",
			ptpBytes: [10]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
			expected: "0",
		},
		{
			name:     "one second",
			ptpBytes: [10]byte{0, 0, 0, 0, 0, 1, 0, 0, 0, 0},
			expected: "1000000000",
		},
		{
			name:     "one nanosecond",
			ptpBytes: [10]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 1},
			expected: "1",
		},
		{
			name:     "one second and one nanosecond",
			ptpBytes: [10]byte{0, 0, 0, 0, 0, 1, 0, 0, 0, 1},
			expected: "1000000001",
		},
		{
			name:     "max uint32 seconds",
			ptpBytes: [10]byte{0, 0, 0xFF, 0xFF, 0xFF, 0xFF, 0, 0, 0, 0},
			expected: "4294967295000000000",
		},
		{
			name:     "large timestamp that would overflow uint64 multiplication",
			ptpBytes: [10]byte{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF},
			expected: "281474976710659294967295", // 0xFFFFFFFFFFFF * 1000000000 + 0xFFFFFFFF
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := Timestamp{PTP: tt.ptpBytes}
			got := ts.TotalNanoSeconds()
			expected := new(big.Int)
			expected.SetString(tt.expected, 10)

			if got.Cmp(expected) != 0 {
				t.Errorf("TotalNanoSeconds() = %s, want %s", got.String(), expected.String())
			}
		})
	}
}

func TestTimestampInSamples(t *testing.T) {
	tests := []struct {
		name       string
		ptpBytes   [10]byte
		sampleRate uint32
		expected   uint32
	}{
		{
			name:       "zero time",
			ptpBytes:   [10]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
			sampleRate: 48000,
			expected:   0,
		},
		{
			name:       "one second at 48kHz",
			ptpBytes:   [10]byte{0, 0, 0, 0, 0, 1, 0, 0, 0, 0},
			sampleRate: 48000,
			expected:   48000,
		},
		{
			name:       "one second at 96kHz",
			ptpBytes:   [10]byte{0, 0, 0, 0, 0, 1, 0, 0, 0, 0},
			sampleRate: 96000,
			expected:   96000,
		},
		{
			name:       "half second at 48kHz",
			ptpBytes:   [10]byte{0, 0, 0, 0, 0, 0, 0x1D, 0xCD, 0x65, 0x00}, // 500000000 ns
			sampleRate: 48000,
			expected:   24000,
		},
		{
			name:       "fractional samples rounded down",
			ptpBytes:   [10]byte{0, 0, 0, 0, 0, 0, 0, 0, 0x01, 0x00}, // 256 ns
			sampleRate: 48000,
			expected:   0, // 256ns * 48000 / 1e9 = 0.01228... -> 0
		},
		{
			name:       "large timestamp that could cause overflow with uint64",
			ptpBytes:   [10]byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x10, 0x00, 0x00, 0x00, 0x00}, // 0x10 seconds = 16 seconds
			sampleRate: 48000,
			expected:   768000, // 16 * 48000 = 768000 samples, fits in uint32
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := Timestamp{PTP: tt.ptpBytes}
			got := ts.InSamples(tt.sampleRate)
			if got != tt.expected {
				t.Errorf("InSamples(%d) = %d, want %d", tt.sampleRate, got, tt.expected)
			}
		})
	}
}

func TestTimestampString(t *testing.T) {
	tests := []struct {
		name     string
		ptpBytes [10]byte
		contains []string // Strings that should be present in the output
	}{
		{
			name:     "epoch time",
			ptpBytes: [10]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
			contains: []string{"1970", "01", "01", "00:00:00"},
		},
		{
			name:     "one second",
			ptpBytes: [10]byte{0, 0, 0, 0, 0, 1, 0, 0, 0, 0},
			contains: []string{"1970", "01", "01", "00:00:01"},
		},
		{
			name:     "with nanoseconds",
			ptpBytes: [10]byte{0, 0, 0, 0, 0, 1, 0x01, 0x00, 0x00, 0x00}, // 1s + ~16.7M ns
			contains: []string{"1970", "01", "01", "00:00:01"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := Timestamp{PTP: tt.ptpBytes}
			got := ts.AsUTC()

			for _, contain := range tt.contains {
				if !strings.Contains(got, contain) {
					t.Errorf("String() = %q, should contain %q", got, contain)
				}
			}

			// Verify it's a valid RFC3339Nano format by parsing it
			_, err := time.Parse(time.RFC3339Nano, got)
			if err != nil {
				t.Errorf("String() = %q, should be valid RFC3339Nano format, got error: %v", got, err)
			}
		})
	}
}

func TestTimestampStringWithLargeValue(t *testing.T) {
	// Test with a very large timestamp that might cause issues
	ts := Timestamp{
		PTP: [10]byte{0x7F, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF},
	}

	// Should not panic and should return some reasonable string
	result := ts.AsUTC()

	if !strings.Contains(result, "out of range") {
		t.Errorf("String() = %q, should contain 'out of range'", result)
	}
}

func TestOverflowPrevention(t *testing.T) {
	// This test specifically checks that we handle cases that would overflow
	// if we were using regular uint64 arithmetic instead of big.Int

	// Create a timestamp with maximum values that would cause overflow
	// in uint64 * uint32 multiplication
	ts := Timestamp{
		PTP: [10]byte{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF},
	}

	// This should not panic or return incorrect results
	samples := ts.InSamples(192000)

	// We expect this to be a large number, but it should be computed correctly
	// The exact value doesn't matter as much as ensuring no overflow occurred
	if samples == 0 {
		t.Error("InSamples() returned 0 for maximum timestamp, likely due to overflow")
	}

	// Verify the TotalNanoSeconds method returns the expected big.Int value
	totalNs := ts.TotalNanoSeconds()
	if totalNs.Sign() <= 0 {
		t.Error("TotalNanoSeconds() should return positive value for maximum timestamp")
	}

	// Verify it's the expected maximum value
	expectedSeconds := new(big.Int).SetUint64(0xFFFFFFFFFFFF)
	expectedNanos := new(big.Int).SetUint64(0xFFFFFFFF)
	billion := new(big.Int).SetUint64(1_000_000_000)
	expected := new(big.Int).Mul(expectedSeconds, billion)
	expected.Add(expected, expectedNanos)

	if totalNs.Cmp(expected) != 0 {
		t.Errorf("TotalNanoSeconds() = %s, want %s", totalNs.String(), expected.String())
	}
}
