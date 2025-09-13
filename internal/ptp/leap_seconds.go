package ptp

import "time"

// LeapSecondEntry represents a leap second insertion
type LeapSecondEntry struct {
	Date      time.Time     // Date when leap second was inserted
	TaiOffset time.Duration // Total TAI-UTC offset after this leap second
}

// leapSeconds contains all leap seconds inserted since 1972
// Data source: https://en.wikipedia.org/wiki/Leap_second
var leapSeconds = []LeapSecondEntry{
	// 1972 - 2 leap seconds inserted (June 30 and December 31)
	{time.Date(1972, time.June, 30, 23, 59, 59, 0, time.UTC), 11 * time.Second},
	{time.Date(1972, time.December, 31, 23, 59, 59, 0, time.UTC), 12 * time.Second},

	// 1973 - 1 leap second inserted (December 31)
	{time.Date(1973, time.December, 31, 23, 59, 59, 0, time.UTC), 13 * time.Second},

	// 1974 - 1 leap second inserted (December 31)
	{time.Date(1974, time.December, 31, 23, 59, 59, 0, time.UTC), 14 * time.Second},

	// 1975 - 1 leap second inserted (December 31)
	{time.Date(1975, time.December, 31, 23, 59, 59, 0, time.UTC), 15 * time.Second},

	// 1976 - 1 leap second inserted (December 31)
	{time.Date(1976, time.December, 31, 23, 59, 59, 0, time.UTC), 16 * time.Second},

	// 1977 - 1 leap second inserted (December 31)
	{time.Date(1977, time.December, 31, 23, 59, 59, 0, time.UTC), 17 * time.Second},

	// 1978 - 1 leap second inserted (December 31)
	{time.Date(1978, time.December, 31, 23, 59, 59, 0, time.UTC), 18 * time.Second},

	// 1979 - 1 leap second inserted (December 31)
	{time.Date(1979, time.December, 31, 23, 59, 59, 0, time.UTC), 19 * time.Second},

	// 1980 - No leap seconds

	// 1981 - 1 leap second inserted (June 30)
	{time.Date(1981, time.June, 30, 23, 59, 59, 0, time.UTC), 20 * time.Second},

	// 1982 - 1 leap second inserted (June 30)
	{time.Date(1982, time.June, 30, 23, 59, 59, 0, time.UTC), 21 * time.Second},

	// 1983 - 1 leap second inserted (June 30)
	{time.Date(1983, time.June, 30, 23, 59, 59, 0, time.UTC), 22 * time.Second},

	// 1984 - No leap seconds

	// 1985 - 1 leap second inserted (June 30)
	{time.Date(1985, time.June, 30, 23, 59, 59, 0, time.UTC), 23 * time.Second},

	// 1986 - No leap seconds

	// 1987 - 1 leap second inserted (December 31)
	{time.Date(1987, time.December, 31, 23, 59, 59, 0, time.UTC), 24 * time.Second},

	// 1988 - No leap seconds

	// 1989 - 1 leap second inserted (December 31)
	{time.Date(1989, time.December, 31, 23, 59, 59, 0, time.UTC), 25 * time.Second},

	// 1990 - 1 leap second inserted (December 31)
	{time.Date(1990, time.December, 31, 23, 59, 59, 0, time.UTC), 26 * time.Second},

	// 1991 - No leap seconds

	// 1992 - 1 leap second inserted (June 30)
	{time.Date(1992, time.June, 30, 23, 59, 59, 0, time.UTC), 27 * time.Second},

	// 1993 - 1 leap second inserted (June 30)
	{time.Date(1993, time.June, 30, 23, 59, 59, 0, time.UTC), 28 * time.Second},

	// 1994 - 1 leap second inserted (June 30)
	{time.Date(1994, time.June, 30, 23, 59, 59, 0, time.UTC), 29 * time.Second},

	// 1995 - 1 leap second inserted (December 31)
	{time.Date(1995, time.December, 31, 23, 59, 59, 0, time.UTC), 30 * time.Second},

	// 1996 - No leap seconds

	// 1997 - 1 leap second inserted (June 30)
	{time.Date(1997, time.June, 30, 23, 59, 59, 0, time.UTC), 31 * time.Second},

	// 1998 - 1 leap second inserted (December 31)
	{time.Date(1998, time.December, 31, 23, 59, 59, 0, time.UTC), 32 * time.Second},

	// 1999-2004 - No leap seconds (6-year gap)

	// 2005 - 1 leap second inserted (December 31)
	{time.Date(2005, time.December, 31, 23, 59, 59, 0, time.UTC), 33 * time.Second},

	// 2006-2007 - No leap seconds

	// 2008 - 1 leap second inserted (December 31)
	{time.Date(2008, time.December, 31, 23, 59, 59, 0, time.UTC), 34 * time.Second},

	// 2009-2011 - No leap seconds

	// 2012 - 1 leap second inserted (June 30)
	{time.Date(2012, time.June, 30, 23, 59, 59, 0, time.UTC), 35 * time.Second},

	// 2013-2014 - No leap seconds

	// 2015 - 1 leap second inserted (June 30)
	{time.Date(2015, time.June, 30, 23, 59, 59, 0, time.UTC), 36 * time.Second},

	// 2016 - 1 leap second inserted (December 31) - Most recent as of 2024
	{time.Date(2016, time.December, 31, 23, 59, 59, 0, time.UTC), 37 * time.Second},

	// No leap seconds since 2016 as of 2024
	// Future leap seconds will be discontinued by or before 2035 per CGPM resolution
}

// TaiOffset returns the TAI-UTC offset for a given UTC time.
//
// Before January 1, 1972: Returns 0 (no systematic TAI-UTC relationship existed)
// After January 1, 1972: TAI runs ahead of UTC by 10 seconds (initial offset)
// plus the number of leap seconds that have been inserted up to that date.
//
// The current UTC system with leap seconds was introduced on January 1, 1972,
// at which point UTC was set to be 10 seconds behind TAI.
//
// Returns the offset as a time.Duration where positive values indicate
// TAI is ahead of UTC.
func TaiOffset(utcTime time.Time) time.Duration {
	// Before the current UTC system started on January 1, 1972, there was no systematic offset
	if utcTime.Before(time.Date(1972, time.January, 1, 0, 0, 0, 0, time.UTC)) {
		return 0 * time.Second
	}

	// Initial offset: UTC was set 10 seconds behind TAI when the system started in 1972
	const initialOffset = 10 * time.Second

	// Count leap seconds that occurred before or at the given time
	offset := initialOffset
	for _, entry := range leapSeconds {
		if utcTime.After(entry.Date) || utcTime.Equal(entry.Date) {
			// Use the TAI offset from the table, which includes initial offset + leap seconds
			offset = entry.TaiOffset
		} else {
			break
		}
	}

	return offset
}

// GetCurrentTaiOffset returns the current TAI-UTC offset.
// As of 2024, this should be 37 seconds (10 initial + 27 leap seconds).
func GetCurrentTaiOffset() time.Duration {
	return TaiOffset(time.Now().UTC())
}

// LeapSecondCount returns the number of leap seconds that have been inserted
// up to the given UTC time. Returns 0 for dates before January 1, 1972.
func LeapSecondCount(utcTime time.Time) int {
	if utcTime.Before(time.Date(1972, time.January, 1, 0, 0, 0, 0, time.UTC)) {
		return 0
	}

	count := 0
	for _, entry := range leapSeconds {
		if utcTime.After(entry.Date) || utcTime.Equal(entry.Date) {
			count++
		} else {
			break
		}
	}

	return count
}

// IsLeapSecond returns true if the given UTC time represents the insertion
// of a leap second (i.e., 23:59:60 on a leap second date).
func IsLeapSecond(utcTime time.Time) bool {
	for _, entry := range leapSeconds {
		if utcTime.Equal(entry.Date) {
			return true
		}
	}
	return false
}

// NextLeapSecond returns the date of the next scheduled leap second after
// the given time, or zero time if no future leap seconds are scheduled.
//
// Note: As of the 2022 CGPM resolution, no new leap seconds will be added
// after 2035, so this function will return zero time for dates after the
// last scheduled leap second.
func NextLeapSecond(utcTime time.Time) time.Time {
	for _, entry := range leapSeconds {
		if utcTime.Before(entry.Date) {
			return entry.Date
		}
	}
	// No future leap seconds scheduled
	return time.Time{}
}

// ConvertUtcToTai converts a UTC time to TAI time by adding the appropriate offset.
func ConvertUtcToTai(utcTime time.Time) time.Time {
	return utcTime.Add(TaiOffset(utcTime))
}

// ConvertTaiToUtc converts a TAI time to UTC time by subtracting the appropriate offset.
// Note: This is approximate for times near leap seconds due to the discontinuous nature of UTC.
func ConvertTaiToUtc(taiTime time.Time) time.Time {
	// This is a simplified conversion - finding the exact UTC time from TAI
	// requires more complex logic due to UTC discontinuities during leap seconds

	// Start with an approximation using current offset to get close
	utcApprox := taiTime.Add(-GetCurrentTaiOffset())

	// Use the offset appropriate for the approximate UTC time
	actualOffset := TaiOffset(utcApprox)
	return taiTime.Add(-actualOffset)
}
