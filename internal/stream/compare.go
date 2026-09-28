package stream

// DescriptionChange classifies how a stream description changed.
type DescriptionChange int

const (
	DescriptionUnchanged DescriptionChange = iota
	// DescriptionTransportChanged: same number of sources, but a destination
	// address or port differs.
	DescriptionTransportChanged
	// DescriptionFormatChanged: encoding, sample rate, channel count or the
	// number of sources differ. It takes precedence over transport changes.
	DescriptionFormatChanged
)

// CompareDescriptions reports how new differs from old. Metadata such as the
// name does not count as a change.
func CompareDescriptions(old, new StreamDescription) DescriptionChange {
	if old.ContentType != new.ContentType ||
		old.SampleRate != new.SampleRate ||
		old.ChannelCount != new.ChannelCount ||
		len(old.Sources) != len(new.Sources) {
		return DescriptionFormatChanged
	}

	for i := range old.Sources {
		a, b := old.Sources[i], new.Sources[i]
		if !a.DestinationAddress.Equal(b.DestinationAddress) || a.DestinationPort != b.DestinationPort {
			return DescriptionTransportChanged
		}
	}

	return DescriptionUnchanged
}
