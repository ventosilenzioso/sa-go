package raknet

import "errors"

var (
	// ErrMalformedDatagram is returned when a datagram cannot be parsed.
	ErrMalformedDatagram = errors.New("raknet: malformed datagram")
)

// Range is an inclusive message-number range used in ACK lists.
type Range struct {
	Min, Max uint16
}

// InternalPacket is one reliability-layer packet inside a datagram.
type InternalPacket struct {
	MessageNumber   uint16
	Reliability     byte
	OrderingChannel uint8
	OrderingIndex   uint16

	SplitID    uint16
	SplitIndex uint32
	SplitCount uint32

	Data          []byte
	DataBitLength int
}

// IsSplit reports whether the packet is a split fragment.
func (p *InternalPacket) IsSplit() bool { return p.SplitCount > 0 }

// Datagram is the parsed form of one UDP payload of a connected peer.
type Datagram struct {
	HasAcks bool
	Acks    []Range
	Packets []InternalPacket
}

// ParseDatagram decodes a connected-peer datagram as the SA-MP server does.
// Parsing stops at the first malformed or split packet, mirroring open.mp
// which abandons the remainder of the datagram in those cases (a split from a
// client is always dropped).
func ParseDatagram(data []byte) (*Datagram, error) {
	return parseDatagram(data, false)
}

// ParseDatagramAllowSplits decodes a datagram the way a client does: split
// fragments are returned instead of being dropped. Used by the bundled test
// client and for server-to-client traffic inspection.
func ParseDatagramAllowSplits(data []byte) (*Datagram, error) {
	return parseDatagram(data, true)
}

func parseDatagram(data []byte, allowSplits bool) (*Datagram, error) {
	bs := FromBytes(data)
	d := &Datagram{}

	hasAcks, err := bs.ReadBool()
	if err != nil {
		return nil, ErrMalformedDatagram
	}
	d.HasAcks = hasAcks
	if hasAcks {
		acks, err := readRangeList(bs)
		if err != nil {
			return nil, err
		}
		d.Acks = acks
	}

	for bs.UnreadBits() >= 16 {
		pkt, err := readInternalPacket(bs, allowSplits)
		if err != nil {
			// Leftover garbage / short read: stop here.
			break
		}
		if pkt == nil {
			// Split packet dropped (server mode): remainder abandoned.
			break
		}
		d.Packets = append(d.Packets, *pkt)
	}
	return d, nil
}

func readRangeList(bs *BitStream) ([]Range, error) {
	count, err := bs.ReadCompressedUint16()
	if err != nil {
		return nil, ErrMalformedDatagram
	}
	ranges := make([]Range, 0, count)
	for i := 0; i < int(count); i++ {
		equal, err := bs.ReadBool()
		if err != nil {
			return nil, ErrMalformedDatagram
		}
		min, err := bs.ReadUint16()
		if err != nil {
			return nil, ErrMalformedDatagram
		}
		max := min
		if !equal {
			max, err = bs.ReadUint16()
			if err != nil {
				return nil, ErrMalformedDatagram
			}
			if max < min {
				return nil, ErrMalformedDatagram
			}
		}
		ranges = append(ranges, Range{Min: min, Max: max})
	}
	return ranges, nil
}

func writeRangeList(bs *BitStream, ranges []Range) {
	bs.WriteCompressedUint16(uint16(len(ranges)))
	for _, r := range ranges {
		if r.Min == r.Max {
			bs.WriteBool(true)
			bs.WriteUint16(r.Min)
		} else {
			bs.WriteBool(false)
			bs.WriteUint16(r.Min)
			bs.WriteUint16(r.Max)
		}
	}
}

// readInternalPacket parses one internal packet. In server mode a split
// packet causes (nil, nil) to be returned so the caller abandons the datagram;
// with allowSplits the fragment is returned. Malformed data yields an error.
func readInternalPacket(bs *BitStream, allowSplits bool) (*InternalPacket, error) {
	if bs.UnreadBits() < 16 {
		return nil, ErrMalformedDatagram
	}
	msgNum, err := bs.ReadUint16()
	if err != nil {
		return nil, ErrMalformedDatagram
	}
	var relBuf [1]byte
	if err := bs.ReadBits(relBuf[:], 4, true); err != nil {
		return nil, ErrMalformedDatagram
	}
	rel := relBuf[0]
	if !IsValidReliability(rel) {
		return nil, ErrMalformedDatagram
	}
	p := &InternalPacket{MessageNumber: msgNum, Reliability: rel}

	if HasOrderingField(rel) {
		var chBuf [1]byte
		if err := bs.ReadBits(chBuf[:], 5, true); err != nil {
			return nil, ErrMalformedDatagram
		}
		if chBuf[0] >= NumberOfOrderedStreams {
			return nil, ErrMalformedDatagram
		}
		p.OrderingChannel = chBuf[0]
		idx, err := bs.ReadUint16()
		if err != nil {
			return nil, ErrMalformedDatagram
		}
		p.OrderingIndex = idx
	}

	isSplit, err := bs.ReadBool()
	if err != nil {
		return nil, ErrMalformedDatagram
	}
	if isSplit {
		sid, err := bs.ReadUint16()
		if err != nil {
			return nil, ErrMalformedDatagram
		}
		sidx, err := bs.ReadCompressedUint32()
		if err != nil {
			return nil, ErrMalformedDatagram
		}
		scnt, err := bs.ReadCompressedUint32()
		if err != nil {
			return nil, ErrMalformedDatagram
		}
		if !allowSplits {
			return nil, nil // dropped split: abandon the datagram
		}
		p.SplitID = sid
		p.SplitIndex = sidx
		p.SplitCount = scnt
	}

	bitLen, err := bs.ReadCompressedUint16()
	if err != nil {
		return nil, ErrMalformedDatagram
	}
	if bitLen == 0 || (int(bitLen)+7)/8 >= 1500 {
		return nil, ErrMalformedDatagram
	}
	p.DataBitLength = int(bitLen)
	data, err := bs.ReadAlignedBytes((int(bitLen) + 7) / 8)
	if err != nil {
		return nil, ErrMalformedDatagram
	}
	p.Data = data
	return p, nil
}

// AppendInternalPacket serializes one internal packet onto the bitstream.
func AppendInternalPacket(bs *BitStream, p *InternalPacket) {
	bs.WriteUint16(p.MessageNumber)
	bs.WriteBits([]byte{p.Reliability}, 4, true)
	if HasOrderingField(p.Reliability) {
		bs.WriteBits([]byte{p.OrderingChannel}, 5, true)
		bs.WriteUint16(p.OrderingIndex)
	}
	bs.WriteBool(p.SplitCount > 0)
	if p.SplitCount > 0 {
		bs.WriteUint16(p.SplitID)
		bs.WriteCompressedUint32(p.SplitIndex)
		bs.WriteCompressedUint32(p.SplitCount)
	}
	bs.WriteCompressedUint16(uint16(p.DataBitLength))
	bs.WriteAlignedBytes(p.Data[:((p.DataBitLength + 7) / 8)])
}

// BuildFrames assembles datagrams from ACK ranges and internal packets.
// Each returned slice is one UDP payload.
func BuildDatagramFrom(acks []Range, packets []*InternalPacket) []byte {
	bs := New()
	if len(acks) > 0 {
		bs.WriteBool(true)
		writeRangeList(bs, acks)
	} else {
		bs.WriteBool(false)
	}
	for _, p := range packets {
		AppendInternalPacket(bs, p)
	}
	bs.AlignWrite()
	return bs.Bytes()[:bs.Len()]
}
