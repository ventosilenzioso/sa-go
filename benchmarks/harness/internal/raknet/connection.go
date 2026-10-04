package raknet

import (
	"errors"
	"fmt"
	"sort"
	"time"
)

// Errors returned by the reliability layer.
var (
	// ErrLimitExceeded indicates an anti-abuse limit was breached; the caller
	// should ban the peer (mirrors open.mp's shouldBanPeer behavior).
	ErrLimitExceeded = errors.New("raknet: peer exceeded a protocol limit")
	// ErrTooLarge indicates a payload cannot be sent even with splitting.
	ErrTooLarge = errors.New("raknet: payload too large")
	// ErrSendBacklog indicates the reliable send window is full.
	ErrSendBacklog = errors.New("raknet: reliable send backlog full")
)

// Config tunes the reliability layer. Zero values are replaced by defaults
// that match the open.mp server defaults.
type Config struct {
	MTU              int           // default 1500
	Timeout          time.Duration // no traffic for this long => lost, default 10s
	ResendMin        time.Duration // floor for resend interval, default 30ms
	ResendPingMult   float64       // resend after ping*mult, default 3.0
	MessagesLimit    int           // received messages per second, default 500
	AcksLimit        int           // received acks per second, default 3000
	MessageHoleLimit int           // default 3000
	MaxPending       int           // max reliable packets awaiting ack, default 4096
	OrderedBufCap    int           // max buffered out-of-order packets per channel, default 512
	PingInterval     time.Duration // interval for our own ID_INTERNAL_PING, default 5s
}

func (cfg *Config) applyDefaults() {
	if cfg.MTU <= 0 {
		cfg.MTU = DefaultMTUSize
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = 10 * time.Second
	}
	if cfg.ResendMin <= 0 {
		cfg.ResendMin = 30 * time.Millisecond
	}
	if cfg.ResendPingMult <= 0 {
		cfg.ResendPingMult = 3.0
	}
	if cfg.MessagesLimit <= 0 {
		cfg.MessagesLimit = 500
	}
	if cfg.AcksLimit <= 0 {
		cfg.AcksLimit = 3000
	}
	if cfg.MessageHoleLimit <= 0 {
		cfg.MessageHoleLimit = 3000
	}
	if cfg.MaxPending <= 0 {
		cfg.MaxPending = 4096
	}
	if cfg.OrderedBufCap <= 0 {
		cfg.OrderedBufCap = 512
	}
	if cfg.PingInterval <= 0 {
		cfg.PingInterval = 5 * time.Second
	}
}

type recvSlot struct {
	received bool
	deadline time.Time
}

type pendingSend struct {
	pkt        *InternalPacket
	nextResend time.Time
}

// Conn is the per-peer reliability layer (server side).
type Conn struct {
	cfg Config

	// send side
	nextMsgNum  uint16
	nextSplitID uint16
	seqWrite    [NumberOfOrderedStreams]uint16
	ordWrite    [NumberOfOrderedStreams]uint16
	outQueue    []*InternalPacket
	pending     []*pendingSend
	pendingSet  map[uint16]struct{}
	needAck     map[uint16]struct{}

	// receive side
	recvBase   uint16
	recvQueue  []recvSlot
	seqRead    [NumberOfOrderedStreams]uint16
	ordRead    [NumberOfOrderedStreams]uint16
	ordBuf     [NumberOfOrderedStreams]map[uint16]*InternalPacket
	outOfOrder int

	// stats / limits
	ping           time.Duration
	lastRecv       time.Time
	lastPingSend   time.Time
	msgWindowStart time.Time
	msgWindowCount int
	ackWindowStart time.Time
	ackWindowCount int
}

// NewConn creates a reliability-layer connection. now is the creation time.
func NewConn(cfg Config, now time.Time) *Conn {
	cfg.applyDefaults()
	return &Conn{
		cfg:            cfg,
		pendingSet:     make(map[uint16]struct{}),
		needAck:        make(map[uint16]struct{}),
		lastRecv:       now,
		lastPingSend:   now,
		msgWindowStart: now,
		ackWindowStart: now,
	}
}

func (c *Conn) maxPayloadBytes() int {
	return c.cfg.MTU - UDPHeaderSize
}

func (c *Conn) resendInterval() time.Duration {
	d := time.Duration(float64(c.ping) * c.cfg.ResendPingMult)
	if d < c.cfg.ResendMin {
		d = c.cfg.ResendMin
	}
	return d
}

// Ping returns the round-trip time observed from ID_CONNECTED_PONG.
func (c *Conn) Ping() time.Duration { return c.ping }

// Enqueue schedules a payload for delivery. The payload is split automatically
// when it exceeds what a single datagram can carry.
func (c *Conn) Enqueue(payload []byte, rel byte, channel uint8) error {
	if !IsValidReliability(rel) {
		return errors.New("raknet: invalid reliability")
	}
	if HasOrderingField(rel) && channel >= NumberOfOrderedStreams {
		return errors.New("raknet: invalid ordering channel")
	}
	if len(payload) == 0 {
		return errors.New("raknet: empty payload")
	}

	// Conservative per-fragment block size (headers are far smaller than this).
	const headerBytes = 16
	maxBlock := c.maxPayloadBytes() - headerBytes
	if maxBlock <= 0 {
		return ErrTooLarge
	}

	if len(payload) <= maxBlock {
		return c.enqueueOne(payload, rel, channel, 0, 0, 0)
	}

	// Split: fragments share splitID, reliability and orderingIndex. Fragment
	// 0 keeps the message number assigned to the whole message; later
	// fragments consume fresh message numbers (SplitPacket in ReliabilityLayer.cpp).
	splitID := c.nextSplitID
	c.nextSplitID++
	count := (len(payload) + maxBlock - 1) / maxBlock

	// One ordering index for the whole message, shared by all fragments.
	var ordIdx uint16
	if HasOrderingField(rel) {
		ordIdx = c.orderingIndex(rel, channel)
	}

	for i := 0; i < count; i++ {
		start := i * maxBlock
		end := start + maxBlock
		if end > len(payload) {
			end = len(payload)
		}
		p := &InternalPacket{
			MessageNumber:   c.nextMsgNum,
			Reliability:     rel,
			OrderingChannel: channel,
			OrderingIndex:   ordIdx,
			SplitID:         splitID,
			SplitIndex:      uint32(i),
			SplitCount:      uint32(count),
			Data:            payload[start:end],
			DataBitLength:   (end - start) * 8,
		}
		c.nextMsgNum++
		if err := c.pushOutgoing(p); err != nil {
			return err
		}
	}
	return nil
}

// orderingIndex returns the next ordering index for a stream. Callers must
// invoke it exactly once per logical message.
func (c *Conn) orderingIndex(rel byte, channel uint8) uint16 {
	if rel == ReliableOrdered {
		v := c.ordWrite[channel]
		c.ordWrite[channel]++
		return v
	}
	v := c.seqWrite[channel]
	c.seqWrite[channel]++
	return v
}

func (c *Conn) enqueueOne(payload []byte, rel byte, channel uint8, splitID uint16, splitIndex, splitCount uint32) error {
	p := &InternalPacket{
		MessageNumber:   c.nextMsgNum,
		Reliability:     rel,
		OrderingChannel: channel,
		SplitID:         splitID,
		SplitIndex:      splitIndex,
		SplitCount:      splitCount,
		Data:            payload,
		DataBitLength:   len(payload) * 8,
	}
	c.nextMsgNum++
	if HasOrderingField(rel) {
		p.OrderingIndex = c.orderingIndex(rel, channel)
	}
	return c.pushOutgoing(p)
}

func (c *Conn) pushOutgoing(p *InternalPacket) error {
	if isReliable(p.Reliability) {
		if len(c.pending)+len(c.outQueue) >= c.cfg.MaxPending {
			return ErrSendBacklog
		}
	}
	c.outQueue = append(c.outQueue, p)
	return nil
}

func isReliable(rel byte) bool {
	return rel == Reliable || rel == ReliableOrdered || rel == ReliableSequenced
}

// HandleDatagram processes one plain datagram from the peer. It returns the
// packets ready for delivery to the upper layer; system packets such as pings
// are consumed internally.
func (c *Conn) HandleDatagram(data []byte, now time.Time) ([]*InternalPacket, error) {
	d, err := ParseDatagram(data)
	if err != nil {
		return nil, err
	}
	c.lastRecv = now

	if d.HasAcks {
		c.ackWindowCount += countAcked(d.Acks)
		if now.Sub(c.ackWindowStart) >= time.Second {
			c.ackWindowStart = now
			c.ackWindowCount = 0
		}
		if c.ackWindowCount > c.cfg.AcksLimit {
			return nil, fmt.Errorf("%w: acks limit (%d > %d)", ErrLimitExceeded, c.ackWindowCount, c.cfg.AcksLimit)
		}
		for _, r := range d.Acks {
			if int(r.Max-r.Min) > c.cfg.MessageHoleLimit {
				return nil, fmt.Errorf("%w: ack range hole (%d > %d)", ErrLimitExceeded, int(r.Max-r.Min), c.cfg.MessageHoleLimit)
			}
			m := r.Min
			for {
				c.removePending(m)
				if m == r.Max {
					break
				}
				m++
			}
		}
	}

	var deliver []*InternalPacket
	for i := range d.Packets {
		p := &d.Packets[i]
		got, err := c.acceptPacket(p, now)
		if err != nil {
			return deliver, err
		}
		for _, dp := range got {
			if dp.DataBitLength >= 8 && len(dp.Data) > 0 {
				switch dp.Data[0] {
				case IDInternalPing:
					if dp.DataBitLength == 40 {
						pong := &InternalPacket{
							Reliability:   Unreliable,
							Data:          append(append([]byte{IDConnectedPong}, dp.Data[1:5]...), nowBytes32(now)...),
							DataBitLength: 72,
						}
						_ = c.pushOutgoing(pong)
						continue
					}
				case IDConnectedPong:
					if dp.DataBitLength == 72 {
						sent := le32(dp.Data[1:5])
						rttMs := uint32(now.UnixMilli()) - sent
						if rttMs < 10000 {
							c.ping = time.Duration(rttMs) * time.Millisecond
						}
						continue
					}
				}
			}
			deliver = append(deliver, dp)
		}
	}
	return deliver, nil
}

func countAcked(ranges []Range) int {
	n := 0
	for _, r := range ranges {
		n += int(r.Max-r.Min) + 1
	}
	return n
}

func nowBytes32(now time.Time) []byte {
	v := uint32(now.UnixMilli())
	return []byte{byte(v), byte(v >> 8), byte(v >> 16), byte(v >> 24)}
}

func (c *Conn) removePending(msgNum uint16) {
	if _, ok := c.pendingSet[msgNum]; !ok {
		return
	}
	delete(c.pendingSet, msgNum)
	for i, p := range c.pending {
		if p.pkt.MessageNumber == msgNum {
			c.pending = append(c.pending[:i], c.pending[i+1:]...)
			break
		}
	}
}

// acceptPacket applies dedup + ordering rules and returns the packets that
// became deliverable (possibly several when an ordering gap is filled).
func (c *Conn) acceptPacket(p *InternalPacket, now time.Time) ([]*InternalPacket, error) {
	rel := p.Reliability

	if isReliable(rel) {
		// Ack every reliable packet, duplicates included (RakNet acks before
		// dedup).
		c.needAck[p.MessageNumber] = struct{}{}

		c.msgWindowCount++
		if now.Sub(c.msgWindowStart) >= time.Second {
			c.msgWindowStart = now
			c.msgWindowCount = 0
		}
		if c.msgWindowCount > c.cfg.MessagesLimit {
			return nil, fmt.Errorf("%w: messages limit (%d > %d)", ErrLimitExceeded, c.msgWindowCount, c.cfg.MessagesLimit)
		}

		hole := uint16(p.MessageNumber - c.recvBase)

		switch {
		case hole == 0:
			if len(c.recvQueue) > 0 {
				c.recvQueue = c.recvQueue[1:]
			}
			c.recvBase++
		case hole > 32767: // already counted past => duplicate
			return nil, nil
		case int(hole) < len(c.recvQueue):
			if c.recvQueue[hole].received {
				return nil, nil
			}
			c.recvQueue[hole].received = true
		default:
			for int(hole) > len(c.recvQueue) {
				c.recvQueue = append(c.recvQueue, recvSlot{deadline: now.Add(c.cfg.Timeout)})
			}
			c.recvQueue = append(c.recvQueue, recvSlot{received: true})
		}

		// Pop leading completed/expired slots; a received slot has a zero
		// deadline which is always "expired", matching RakNet's `Peek() < time`.
		for len(c.recvQueue) > 0 {
			s := c.recvQueue[0]
			if s.received || now.After(s.deadline) {
				c.recvQueue = c.recvQueue[1:]
				c.recvBase++
				continue
			}
			break
		}
	}

	switch rel {
	case ReliableSequenced, UnreliableSequenced:
		ch := p.OrderingChannel
		if IsOlderOrderedPacket(p.OrderingIndex, c.seqRead[ch]) {
			return nil, nil
		}
		c.seqRead[ch] = p.OrderingIndex + 1
		return []*InternalPacket{p}, nil

	case ReliableOrdered:
		ch := p.OrderingChannel
		if IsOlderOrderedPacket(p.OrderingIndex, c.ordRead[ch]) {
			return nil, nil
		}
		if p.OrderingIndex != c.ordRead[ch] {
			// Newer than expected: buffer until the gap fills.
			c.outOfOrder++
			if c.outOfOrder > c.cfg.MessageHoleLimit {
				return nil, fmt.Errorf("%w: out of order (%d > %d)", ErrLimitExceeded, c.outOfOrder, c.cfg.MessageHoleLimit)
			}
			if c.ordBuf[ch] == nil {
				c.ordBuf[ch] = make(map[uint16]*InternalPacket)
			}
			if len(c.ordBuf[ch]) >= c.cfg.OrderedBufCap {
				return nil, nil
			}
			c.ordBuf[ch][p.OrderingIndex] = p
			return nil, nil
		}
		out := []*InternalPacket{p}
		c.ordRead[ch]++
		// Drain any buffered packets that are now consecutive.
		for {
			q, ok := c.ordBuf[ch][c.ordRead[ch]]
			if !ok {
				break
			}
			delete(c.ordBuf[ch], c.ordRead[ch])
			c.ordRead[ch]++
			out = append(out, q)
		}
		return out, nil
	}
	return []*InternalPacket{p}, nil
}

// IsOlderOrderedPacket is a faithful port of ReliabilityLayer::
// IsOlderOrderedPacket for a 16-bit ordering index.
func IsOlderOrderedPacket(newIdx, waiting uint16) bool {
	const maxRange = uint16(0xFFFF)
	if waiting > maxRange/2 {
		if newIdx >= waiting-(maxRange/2)+1 && newIdx < waiting {
			return true
		}
	} else {
		if newIdx >= uint16(waiting-(maxRange/2+1)) || newIdx < waiting {
			return true
		}
	}
	return false
}

// packetBits returns the exact serialized size of one internal packet.
func packetBits(p *InternalPacket) int {
	bs := New()
	AppendInternalPacket(bs, p)
	return bs.BitLen()
}

// Tick builds the datagrams to send during this cycle: due resends first,
// then new sends, with pending ACK ranges attached to the first datagram.
func (c *Conn) Tick(now time.Time) [][]byte {
	// Periodic ping keeps the client's timeout from firing on an idle server.
	if c.cfg.PingInterval > 0 && now.Sub(c.lastPingSend) >= c.cfg.PingInterval {
		c.lastPingSend = now
		t := uint32(now.UnixMilli())
		_ = c.pushOutgoing(&InternalPacket{
			Reliability:   Unreliable,
			Data:          []byte{IDInternalPing, byte(t), byte(t >> 8), byte(t >> 16), byte(t >> 24)},
			DataBitLength: 40,
		})
	}

	maxBits := c.maxPayloadBytes() * 8
	if maxBits <= 0 {
		return nil
	}

	// Work list: due resends (in send order) followed by new packets.
	type entry struct {
		pkt    *InternalPacket
		resend *pendingSend
		isNew  bool
	}
	var work []entry
	ri := c.resendInterval()
	for _, ps := range c.pending {
		if !ps.nextResend.After(now) {
			work = append(work, entry{pkt: ps.pkt, resend: ps})
		}
	}
	newPkts := c.outQueue
	for _, p := range newPkts {
		work = append(work, entry{pkt: p, isNew: true})
	}
	c.outQueue = nil

	if len(work) == 0 && len(c.needAck) == 0 {
		return nil
	}

	ackRanges, remaining := c.takeAckRanges(maxBits)

	var frames [][]byte
	bs := New()
	bitsUsed := 0

	writeFirstHeader := func() {
		if len(ackRanges) > 0 {
			bs.WriteBool(true)
			writeRangeList(bs, ackRanges)
		} else {
			bs.WriteBool(false)
		}
		bitsUsed = bs.BitLen()
	}
	writeFirstHeader()

	flush := func() {
		if bs.BitLen() > 0 {
			bs.AlignWrite()
			frames = append(frames, bs.Bytes()[:bs.Len()])
		}
		bs = New()
	}

	sent := make([]bool, len(work))
	wrotePackets := false
	for i := range work {
		need := packetBits(work[i].pkt)
		if bitsUsed+need > maxBits {
			if wrotePackets {
				flush()
				bs.WriteBool(false) // ACKs only in the first datagram
				bitsUsed = bs.BitLen()
				wrotePackets = false
			}
			if bitsUsed+need > maxBits {
				// Cannot fit even in a fresh datagram: should not happen
				// because payloads are split on enqueue. Drop it.
				if work[i].resend != nil {
					c.removePending(work[i].pkt.MessageNumber)
				}
				continue
			}
		}
		AppendInternalPacket(bs, work[i].pkt)
		wrotePackets = true
		bitsUsed = bs.BitLen()
		sent[i] = true

		if work[i].resend != nil {
			work[i].resend.nextResend = now.Add(ri)
		} else if work[i].isNew && isReliable(work[i].pkt.Reliability) {
			if _, dup := c.pendingSet[work[i].pkt.MessageNumber]; !dup {
				c.pendingSet[work[i].pkt.MessageNumber] = struct{}{}
				c.pending = append(c.pending, &pendingSend{
					pkt:        work[i].pkt,
					nextResend: now.Add(ri),
				})
			}
		}
	}

	// Keep unsent new packets queued.
	var unsent []*InternalPacket
	for i := range work {
		if work[i].isNew && !sent[i] {
			unsent = append(unsent, work[i].pkt)
		}
	}
	c.outQueue = unsent

	if bs.BitLen() > 0 {
		flush()
	}

	// Re-queue ack ranges that did not fit into the first datagram.
	for _, r := range remaining {
		for m := r.Min; ; m++ {
			c.needAck[m] = struct{}{}
			if m == r.Max {
				break
			}
		}
	}
	if len(frames) == 0 {
		return nil
	}
	return frames
}

// takeAckRanges converts the pending ACK set into sorted, merged ranges and
// removes the ranges that fit into a budget of maxBits.
func (c *Conn) takeAckRanges(maxBits int) (sent []Range, remaining []Range) {
	if len(c.needAck) == 0 {
		return nil, nil
	}
	nums := make([]uint16, 0, len(c.needAck))
	for m := range c.needAck {
		nums = append(nums, m)
	}
	sort.Slice(nums, func(i, j int) bool { return nums[i] < nums[j] })
	var all []Range
	start, prev := nums[0], nums[0]
	for _, m := range nums[1:] {
		if m == prev+1 {
			prev = m
			continue
		}
		all = append(all, Range{Min: start, Max: prev})
		start, prev = m, m
	}
	all = append(all, Range{Min: start, Max: prev})

	budget := maxBits - 16
	used := 0
	cut := len(all)
	for i, r := range all {
		cost := 1 + 16 + 16
		if r.Min == r.Max {
			cost = 1 + 16 + 1
		}
		if used+cost > budget {
			cut = i
			break
		}
		used += cost
	}
	for _, r := range all[:cut] {
		for m := r.Min; ; m++ {
			delete(c.needAck, m)
			if m == r.Max {
				break
			}
		}
	}
	return all[:cut], all[cut:]
}

// TimedOut reports whether the peer has gone silent.
func (c *Conn) TimedOut(now time.Time) bool {
	return now.Sub(c.lastRecv) > c.cfg.Timeout
}

// LastRecv returns the time of the last valid datagram.
func (c *Conn) LastRecv() time.Time { return c.lastRecv }
