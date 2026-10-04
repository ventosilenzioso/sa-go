package netsim

import (
	"math/rand"
	"time"
)

// dupExtraDelay is the additional delay applied to the second copy
// when duplication occurs.
const dupExtraDelay = time.Millisecond

// Profile describes network conditions applied by Sim.
type Profile struct {
	Loss         float64
	Dup          float64
	DelayMin     time.Duration
	DelayMax     time.Duration
	ReorderProb  float64
	ReorderDepth int
}

// ProfileClean returns a profile with no loss, duplication, delay or reorder.
func ProfileClean() Profile {
	return Profile{
		Loss:         0,
		Dup:          0,
		DelayMin:     0,
		DelayMax:     0,
		ReorderProb:  0,
		ReorderDepth: 0,
	}
}

// ProfileMobile3G returns a typical 3G mobile profile:
// ~2% loss, 80-180ms delay with jitter and a little reorder.
func ProfileMobile3G() Profile {
	return Profile{
		Loss:         0.02,
		Dup:          0,
		DelayMin:     80 * time.Millisecond,
		DelayMax:     180 * time.Millisecond,
		ReorderProb:  0.02,
		ReorderDepth: 2,
	}
}

// ProfileBadMobile returns a bad-mobile profile:
// ~10% loss, 150-400ms delay with reorder.
func ProfileBadMobile() Profile {
	return Profile{
		Loss:         0.10,
		Dup:          0.01,
		DelayMin:     150 * time.Millisecond,
		DelayMax:     400 * time.Millisecond,
		ReorderProb:  0.05,
		ReorderDepth: 3,
	}
}

// ProfileFlakyWifi returns a flaky-wifi profile:
// moderate loss, low delay with jitter, occasional reorder/duplicate.
func ProfileFlakyWifi() Profile {
	return Profile{
		Loss:         0.05,
		Dup:          0.02,
		DelayMin:     5 * time.Millisecond,
		DelayMax:     60 * time.Millisecond,
		ReorderProb:  0.03,
		ReorderDepth: 2,
	}
}

// heldEntry stores one Send event held back for reorder.
// A single event may carry one packet or two copies when duplicated.
type heldEntry struct {
	pkts  [][]byte
	times []time.Time
}

// Sim is a deterministic network-condition simulator.
type Sim struct {
	p    Profile
	rng  *rand.Rand
	hold []heldEntry
}

// New creates a Sim from profile p seeded deterministically with seed.
// Profiles are normalized: probabilities clamped to [0,1], negative
// durations/depths clamped, DelayMax raised to DelayMin if inverted.
func New(p Profile, seed int64) *Sim {
	if p.Loss < 0 {
		p.Loss = 0
	}
	if p.Loss > 1 {
		p.Loss = 1
	}
	if p.Dup < 0 {
		p.Dup = 0
	}
	if p.Dup > 1 {
		p.Dup = 1
	}
	if p.ReorderProb < 0 {
		p.ReorderProb = 0
	}
	if p.ReorderProb > 1 {
		p.ReorderProb = 1
	}
	if p.DelayMin < 0 {
		p.DelayMin = 0
	}
	if p.DelayMax < 0 {
		p.DelayMax = 0
	}
	if p.DelayMax < p.DelayMin {
		p.DelayMax = p.DelayMin
	}
	if p.ReorderDepth < 0 {
		p.ReorderDepth = 0
	}
	return &Sim{
		p:   p,
		rng: rand.New(rand.NewSource(seed)),
	}
}

// copyPacket copies ingress bytes. Nil stays nil so empty/nil packets
// remain legal and distinct; non-nil packets get a fresh backing array.
func copyPacket(pkt []byte) []byte {
	if pkt == nil {
		return nil
	}
	cp := make([]byte, len(pkt))
	copy(cp, pkt)
	return cp
}

// sampleDelay draws a uniform delay in [DelayMin, DelayMax].
// It consumes RNG only when the range is non-zero.
func (s *Sim) sampleDelay() time.Duration {
	if s.p.DelayMax <= s.p.DelayMin {
		return s.p.DelayMin
	}
	span := int64(s.p.DelayMax - s.p.DelayMin)
	return s.p.DelayMin + time.Duration(s.rng.Int63n(span+1))
}

// Send ingests one packet sent at time now and returns packets to
// emit now with their delivery times.
//
// Processing order per Send (deterministic):
//  1. loss roll (drop, no further draws)
//  2. delay sample uniform in [min,max]
//  3. dup roll (second copy delivered dupExtraDelay later)
//  4. reorder roll (hold with ReorderProb if depth > 0)
//
// Hold semantics: a reordered Send is buffered as one entry. The hold
// buffer holds at most ReorderDepth entries (Send events). When a new
// reordered event arrives and the buffer is full, the oldest entry is
// released first (FIFO) to make room. Immediate (non-reordered) Sends
// overtake held entries and do not release them. Flush drains the hold
// buffer in FIFO order and never drops.
func (s *Sim) Send(now time.Time, pkt []byte) (out [][]byte, deliverAt []time.Time) {
	// Loss roll.
	if s.p.Loss >= 1 {
		return nil, nil
	}
	if s.p.Loss > 0 {
		if s.rng.Float64() < s.p.Loss {
			return nil, nil
		}
	}

	cp := copyPacket(pkt)
	delay := s.sampleDelay()
	t0 := now.Add(delay)

	// Dup roll.
	dup := false
	if s.p.Dup >= 1 {
		dup = true
	} else if s.p.Dup > 0 {
		dup = s.rng.Float64() < s.p.Dup
	}

	var pkts [][]byte
	var times []time.Time
	if dup {
		cp2 := copyPacket(pkt)
		pkts = [][]byte{cp, cp2}
		times = []time.Time{t0, t0.Add(dupExtraDelay)}
	} else {
		pkts = [][]byte{cp}
		times = []time.Time{t0}
	}

	// Reorder roll.
	reorder := false
	if s.p.ReorderDepth > 0 && s.p.ReorderProb > 0 {
		if s.p.ReorderProb >= 1 {
			reorder = true
		} else {
			reorder = s.rng.Float64() < s.p.ReorderProb
		}
	}

	if !reorder {
		return pkts, times
	}

	// Hold path: evict oldest if full, then buffer current event.
	if len(s.hold) >= s.p.ReorderDepth {
		oldest := s.hold[0]
		s.hold = s.hold[1:]
		out = append(out, oldest.pkts...)
		deliverAt = append(deliverAt, oldest.times...)
	}
	s.hold = append(s.hold, heldEntry{pkts: pkts, times: times})
	return out, deliverAt
}

// Flush drains the hold buffer in FIFO order and returns held packets
// with their original delivery times. Flush never drops.
func (s *Sim) Flush() (out [][]byte, deliverAt []time.Time) {
	for _, h := range s.hold {
		out = append(out, h.pkts...)
		deliverAt = append(deliverAt, h.times...)
	}
	s.hold = nil
	return out, deliverAt
}
