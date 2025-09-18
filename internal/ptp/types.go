package ptp

import (
	"errors"
	"fmt"
	"math/big"
	"time"
)

const (
	messageTypeSync               = 0x0
	messageTypeDelayReq           = 0x1
	messageTypePDelayReq          = 0x2
	messageTypePDelayResp         = 0x3
	messageTypeFollowUp           = 0x8
	messageTypeDelayResp          = 0x9
	messageTypePDelayRespFollowUp = 0xa
	messageTypeAnnounce           = 0xb
	messageTypeSignaling          = 0xc
	messageTypeManagement         = 0xd
)

type ClockIdentity struct {
	octets [8]byte
}

func (ci ClockIdentity) String() string {
	return fmt.Sprintf("%02x:%02x:%02x:%02x:%02x:%02x:%02x:%02x",
		ci.octets[0], ci.octets[1], ci.octets[2], ci.octets[3],
		ci.octets[4], ci.octets[5], ci.octets[6], ci.octets[7])
}

type Timestamp struct {
	PTP  [10]byte
	Time time.Time
}

func (ts Timestamp) Seconds() uint64 {
	return uint64(ts.PTP[0])<<40 |
		uint64(ts.PTP[1])<<32 |
		uint64(ts.PTP[2])<<24 |
		uint64(ts.PTP[3])<<16 |
		uint64(ts.PTP[4])<<8 |
		uint64(ts.PTP[5])
}

func (ts Timestamp) NanoSeconds() uint64 {
	return uint64(ts.PTP[6])<<24 |
		uint64(ts.PTP[7])<<16 |
		uint64(ts.PTP[8])<<8 |
		uint64(ts.PTP[9])
}

func (ts Timestamp) IsZero() bool {
	return ts.Seconds() == 0 && ts.NanoSeconds() == 0
}

// TotalNanoSeconds returns the total nanoseconds since PTP epoch (1900-01-01)
// using big.Int arithmetic to prevent overflow in large timestamp calculations.
func (ts Timestamp) TotalNanoSeconds() *big.Int {
	seconds := new(big.Int).SetUint64(ts.Seconds())
	nanoseconds := new(big.Int).SetUint64(ts.NanoSeconds())
	billion := new(big.Int).SetUint64(1_000_000_000)

	// Calculate seconds * 1_000_000_000 + nanoseconds
	total := new(big.Int).Mul(seconds, billion)
	total.Add(total, nanoseconds)

	return total
}

func (ts Timestamp) InSamples(sampleRate uint32) uint32 {
	// Get total nanoseconds as big.Int
	totalNs := ts.TotalNanoSeconds()
	sampleRateBig := new(big.Int).SetUint64(uint64(sampleRate))
	billion := new(big.Int).SetUint64(1_000_000_000)

	// Calculate samples = totalNs * sampleRate / 1_000_000_000
	samples := new(big.Int).Mul(totalNs, sampleRateBig)
	samples.Div(samples, billion)

	return uint32(samples.Uint64())
}

var ErrTimestampOutOfRange = errors.New("Timestamp out of range")

func (ts Timestamp) asTAI() (time.Time, error) {
	epoch := time.Unix(0, 0).UTC()
	totalNs := ts.TotalNanoSeconds()

	if !totalNs.IsInt64() {
		return time.Time{}, ErrTimestampOutOfRange
	}

	// Convert big.Int to int64 for time.Duration (may truncate for very large values)
	duration := time.Duration(totalNs.Int64()) * time.Nanosecond

	return epoch.Add(duration), nil
}

func (ts Timestamp) AsUTC() string {
	utc, err := ts.asTAI()
	if errors.Is(err, ErrTimestampOutOfRange) {
		return fmt.Sprintf("Timestamp out of range (%d s, %d ns)", ts.Seconds(), ts.NanoSeconds())
	}

	return fmt.Sprintf("%s", utc.Format(time.RFC3339Nano))
}

func (ts Timestamp) AsTAI() string {
	tai, err := ts.asTAI()
	if errors.Is(err, ErrTimestampOutOfRange) {
		return fmt.Sprintf("Timestamp out of range (%d s, %d ns)", ts.Seconds(), ts.NanoSeconds())
	}

	utc := ConvertTaiToUtc(tai)

	return fmt.Sprintf("%s", utc.Format(time.RFC3339Nano))
}
