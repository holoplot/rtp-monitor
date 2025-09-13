package ptp

import (
	"testing"
	"time"
)

func TestTaiOffset(t *testing.T) {
	tests := []struct {
		name        string
		utcTime     time.Time
		expectedSec int
	}{
		{
			name:        "before leap seconds (1971)",
			utcTime:     time.Date(1971, time.December, 31, 12, 0, 0, 0, time.UTC),
			expectedSec: 0, // No offset before 1972
		},
		{
			name:        "first leap second (1972 June 30)",
			utcTime:     time.Date(1972, time.July, 1, 0, 0, 0, 0, time.UTC),
			expectedSec: 11, // 10 + 1 leap second
		},
		{
			name:        "second leap second (1972 Dec 31)",
			utcTime:     time.Date(1973, time.January, 1, 0, 0, 0, 0, time.UTC),
			expectedSec: 12, // 10 + 2 leap seconds
		},
		{
			name:        "after several leap seconds (1980)",
			utcTime:     time.Date(1980, time.January, 1, 0, 0, 0, 0, time.UTC),
			expectedSec: 19, // 10 + 9 leap seconds by end of 1979
		},
		{
			name:        "during 6-year gap (2002)",
			utcTime:     time.Date(2002, time.June, 15, 12, 0, 0, 0, time.UTC),
			expectedSec: 32, // No leap seconds between 1999-2004
		},
		{
			name:        "after 2005 leap second",
			utcTime:     time.Date(2006, time.January, 1, 0, 0, 0, 0, time.UTC),
			expectedSec: 33, // 10 + 23 leap seconds
		},
		{
			name:        "after 2012 leap second",
			utcTime:     time.Date(2012, time.July, 1, 0, 0, 0, 0, time.UTC),
			expectedSec: 35, // 10 + 25 leap seconds
		},
		{
			name:        "after 2015 leap second",
			utcTime:     time.Date(2015, time.July, 1, 0, 0, 0, 0, time.UTC),
			expectedSec: 36, // 10 + 26 leap seconds
		},
		{
			name:        "after most recent leap second (2016)",
			utcTime:     time.Date(2017, time.January, 1, 0, 0, 0, 0, time.UTC),
			expectedSec: 37, // 10 + 27 leap seconds (current as of 2024)
		},
		{
			name:        "current time (2024)",
			utcTime:     time.Date(2024, time.January, 1, 0, 0, 0, 0, time.UTC),
			expectedSec: 37, // Still 37 seconds
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			offset := TaiOffset(tt.utcTime)
			expectedDuration := time.Duration(tt.expectedSec) * time.Second
			if offset != expectedDuration {
				t.Errorf("TaiOffset(%s) = %v, want %v",
					tt.utcTime.Format(time.RFC3339), offset, expectedDuration)
			}
		})
	}
}

func TestLeapSecondCount(t *testing.T) {
	tests := []struct {
		name     string
		utcTime  time.Time
		expected int
	}{
		{
			name:     "before leap seconds",
			utcTime:  time.Date(1971, time.December, 31, 23, 59, 59, 0, time.UTC),
			expected: 0,
		},
		{
			name:     "after first leap second",
			utcTime:  time.Date(1972, time.July, 1, 0, 0, 0, 0, time.UTC),
			expected: 1,
		},
		{
			name:     "after both 1972 leap seconds",
			utcTime:  time.Date(1973, time.January, 1, 0, 0, 0, 0, time.UTC),
			expected: 2,
		},
		{
			name:     "end of 1979 (9 leap seconds total)",
			utcTime:  time.Date(1980, time.January, 1, 0, 0, 0, 0, time.UTC),
			expected: 9,
		},
		{
			name:     "during gap period (no new leap seconds)",
			utcTime:  time.Date(2002, time.June, 15, 12, 0, 0, 0, time.UTC),
			expected: 22, // Same count as 1998
		},
		{
			name:     "current era (2024)",
			utcTime:  time.Date(2024, time.January, 1, 0, 0, 0, 0, time.UTC),
			expected: 27, // Total leap seconds inserted
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			count := LeapSecondCount(tt.utcTime)
			if count != tt.expected {
				t.Errorf("LeapSecondCount(%s) = %d, want %d",
					tt.utcTime.Format(time.RFC3339), count, tt.expected)
			}
		})
	}
}

func TestIsLeapSecond(t *testing.T) {
	tests := []struct {
		name     string
		utcTime  time.Time
		expected bool
	}{
		{
			name:     "first leap second (1972-06-30 23:59:59)",
			utcTime:  time.Date(1972, time.June, 30, 23, 59, 59, 0, time.UTC),
			expected: true,
		},
		{
			name:     "second leap second (1972-12-31 23:59:59)",
			utcTime:  time.Date(1972, time.December, 31, 23, 59, 59, 0, time.UTC),
			expected: true,
		},
		{
			name:     "most recent leap second (2016-12-31 23:59:59)",
			utcTime:  time.Date(2016, time.December, 31, 23, 59, 59, 0, time.UTC),
			expected: true,
		},
		{
			name:     "not a leap second date",
			utcTime:  time.Date(2020, time.December, 31, 23, 59, 59, 0, time.UTC),
			expected: false,
		},
		{
			name:     "leap second date but wrong time",
			utcTime:  time.Date(2016, time.December, 31, 12, 0, 0, 0, time.UTC),
			expected: false,
		},
		{
			name:     "regular time",
			utcTime:  time.Date(2024, time.January, 15, 12, 30, 0, 0, time.UTC),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsLeapSecond(tt.utcTime)
			if result != tt.expected {
				t.Errorf("IsLeapSecond(%s) = %v, want %v",
					tt.utcTime.Format(time.RFC3339), result, tt.expected)
			}
		})
	}
}

func TestNextLeapSecond(t *testing.T) {
	tests := []struct {
		name     string
		utcTime  time.Time
		expected time.Time
	}{
		{
			name:     "before any leap seconds",
			utcTime:  time.Date(1971, time.June, 1, 0, 0, 0, 0, time.UTC),
			expected: time.Date(1972, time.June, 30, 23, 59, 59, 0, time.UTC),
		},
		{
			name:     "between first and second 1972 leap seconds",
			utcTime:  time.Date(1972, time.September, 1, 0, 0, 0, 0, time.UTC),
			expected: time.Date(1972, time.December, 31, 23, 59, 59, 0, time.UTC),
		},
		{
			name:     "during the 6-year gap (1999-2004)",
			utcTime:  time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC),
			expected: time.Date(2005, time.December, 31, 23, 59, 59, 0, time.UTC),
		},
		{
			name:     "after all leap seconds (2020)",
			utcTime:  time.Date(2020, time.January, 1, 0, 0, 0, 0, time.UTC),
			expected: time.Time{}, // No future leap seconds
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := NextLeapSecond(tt.utcTime)
			if !result.Equal(tt.expected) {
				t.Errorf("NextLeapSecond(%s) = %s, want %s",
					tt.utcTime.Format(time.RFC3339),
					result.Format(time.RFC3339),
					tt.expected.Format(time.RFC3339))
			}
		})
	}
}

func TestGetCurrentTaiOffset(t *testing.T) {
	offset := GetCurrentTaiOffset()
	// As of 2024, should be 37 seconds
	expected := 37 * time.Second
	if offset != expected {
		t.Errorf("GetCurrentTaiOffset() = %v, want %v", offset, expected)
	}
}

func TestConvertUtcToTai(t *testing.T) {
	tests := []struct {
		name        string
		utc         time.Time
		expectedSec int // Expected seconds to add
	}{
		{
			name:        "before leap seconds",
			utc:         time.Date(1971, time.June, 1, 12, 0, 0, 0, time.UTC),
			expectedSec: 0,
		},
		{
			name:        "after first leap second",
			utc:         time.Date(1972, time.July, 1, 12, 0, 0, 0, time.UTC),
			expectedSec: 11,
		},
		{
			name:        "current era",
			utc:         time.Date(2024, time.January, 1, 12, 0, 0, 0, time.UTC),
			expectedSec: 37,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tai := ConvertUtcToTai(tt.utc)
			expected := tt.utc.Add(time.Duration(tt.expectedSec) * time.Second)
			if !tai.Equal(expected) {
				t.Errorf("ConvertUtcToTai(%s) = %s, want %s",
					tt.utc.Format(time.RFC3339),
					tai.Format(time.RFC3339),
					expected.Format(time.RFC3339))
			}
		})
	}
}

func TestConvertTaiToUtc(t *testing.T) {
	tests := []struct {
		name        string
		tai         time.Time
		expectedUtc time.Time
	}{
		{
			name:        "simple conversion",
			tai:         time.Date(2024, time.January, 1, 12, 0, 37, 0, time.UTC), // TAI time
			expectedUtc: time.Date(2024, time.January, 1, 12, 0, 0, 0, time.UTC),  // UTC time (37 sec earlier)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			utc := ConvertTaiToUtc(tt.tai)
			// Allow for small differences due to approximation
			diff := utc.Sub(tt.expectedUtc)
			if diff < 0 {
				diff = -diff
			}
			if diff > time.Second {
				t.Errorf("ConvertTaiToUtc(%s) = %s, want %s (diff: %v)",
					tt.tai.Format(time.RFC3339),
					utc.Format(time.RFC3339),
					tt.expectedUtc.Format(time.RFC3339),
					diff)
			}
		})
	}
}

// TestLeapSecondsTableConsistency verifies that the leap seconds table
// has consistent offsets (each leap second adds exactly 1 second)
func TestLeapSecondsTableConsistency(t *testing.T) {
	const initialOffset = 10 // Initial TAI-UTC offset when UTC system started in 1972

	for i, entry := range leapSeconds {
		expectedOffset := initialOffset + i + 1 // +1 for this leap second
		actualOffset := int(entry.TaiOffset.Seconds())

		if actualOffset != expectedOffset {
			t.Errorf("Leap second entry %d (%s) has offset %d, want %d",
				i, entry.Date.Format("2006-01-02"), actualOffset, expectedOffset)
		}
	}
}

// TestLeapSecondsChronologicalOrder verifies that leap seconds are in chronological order
func TestLeapSecondsChronologicalOrder(t *testing.T) {
	for i := 1; i < len(leapSeconds); i++ {
		prev := leapSeconds[i-1]
		curr := leapSeconds[i]

		if !curr.Date.After(prev.Date) {
			t.Errorf("Leap second entries out of order: %s should be after %s",
				curr.Date.Format("2006-01-02"), prev.Date.Format("2006-01-02"))
		}
	}
}

// TestHistoricalAccuracy tests some known historical facts about leap seconds
func TestHistoricalAccuracy(t *testing.T) {
	// Test that 1972 had 2 leap seconds (most in any year)
	count1972 := 0
	for _, entry := range leapSeconds {
		if entry.Date.Year() == 1972 {
			count1972++
		}
	}
	if count1972 != 2 {
		t.Errorf("1972 should have had 2 leap seconds, got %d", count1972)
	}

	// Test the 6-year gap (1999-2004) with no leap seconds
	hasLeapSecondInGap := false
	for _, entry := range leapSeconds {
		year := entry.Date.Year()
		if year >= 1999 && year <= 2004 {
			hasLeapSecondInGap = true
			break
		}
	}
	if hasLeapSecondInGap {
		t.Error("There should be no leap seconds between 1999-2004")
	}

	// Test that the most recent leap second was in 2016
	lastLeapSecond := leapSeconds[len(leapSeconds)-1]
	if lastLeapSecond.Date.Year() != 2016 || lastLeapSecond.Date.Month() != time.December {
		t.Errorf("Last leap second should be 2016-12-31, got %s",
			lastLeapSecond.Date.Format("2006-01-02"))
	}

	// Test total count (27 leap seconds as of 2024)
	if len(leapSeconds) != 27 {
		t.Errorf("Should have 27 leap seconds total, got %d", len(leapSeconds))
	}
}
